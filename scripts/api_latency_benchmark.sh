#!/usr/bin/env bash
#
# HTTP benchmark: GET /getNearestBusStops with varying lat/lng and optional concurrency
# ("spam"). No client geolocation — measures server + network only.
#
# Usage:
#   ./scripts/api_latency_benchmark.sh
#   CONCURRENCY=25 ITERATIONS=200 VARY=random ./scripts/api_latency_benchmark.sh
#   BASE_URL=https://host NEAREST_ONLY=1 JITTER_DEG=0.05 ./scripts/api_latency_benchmark.sh
#
# Env:
#   BASE_URL          default http://localhost:8080
#   ITERATIONS        total nearest-stops requests (default 20)
#   CONCURRENCY       parallel requests per burst (default 1 = sequential)
#   WARMUP            discarded warmup reqs for nearest (default 3)
#   VARY              random | grid  (default random)
#   CENTER_LAT, CENTER_LNG   base point; fallback LAT/LNG for backward compat
#   JITTER_DEG        random: uniform ±jitter in deg on each axis (default 0.03 ~3 km)
#   STEP_DEG          grid: step between points (default 0.002)
#   SEED              added to srand() in random mode for reproducibility
#   NEAREST_ONLY      1 = skip /getBusArrival section (default 0)
#   INCLUDE_BUS_ARRIVAL  explicit 0 also skips bus arrival
#   BUS_STOP_CODE     for optional baseline /getBusArrival
#
# Requires: curl, awk
#
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ITERATIONS="${ITERATIONS:-20}"
CONCURRENCY="${CONCURRENCY:-1}"
WARMUP="${WARMUP:-3}"
VARY="${VARY:-random}"
JITTER_DEG="${JITTER_DEG:-0.03}"
STEP_DEG="${STEP_DEG:-0.002}"
SEED="${SEED:-0}"

CENTER_LAT="${CENTER_LAT:-${LAT:-1.326277}}"
CENTER_LNG="${CENTER_LNG:-${LNG:-103.890342}}"
BUS_STOP_CODE="${BUS_STOP_CODE:-71119}"

NEAREST_ONLY="${NEAREST_ONLY:-0}"
INCLUDE_BUS_ARRIVAL="${INCLUDE_BUS_ARRIVAL:-1}"
if [[ "${NEAREST_ONLY}" == "1" ]]; then INCLUDE_BUS_ARRIVAL=0; fi

arrival_url="${BASE_URL}/getBusArrival?busStopCode=${BUS_STOP_CODE}"

curl_format='%{http_code}\t%{time_total}\t%{time_namelookup}\t%{time_connect}\t%{time_appconnect}\t%{time_starttransfer}\n'

run_curl() {
  local url=$1
  curl -sS -o /dev/null -w "${curl_format}" --compressed "${url}"
}

# Print "lat lng" for request index i (0-based).
gen_lat_lng() {
  local i=$1
  case "${VARY}" in
    grid)
      awk -v i="$i" -v cLat="$CENTER_LAT" -v cLng="$CENTER_LNG" -v step="${STEP_DEG}" '
        BEGIN {
          w = 9
          dx = (i % w) - int(w / 2)
          dy = int(i / w) % w - int(w / 2)
          lat = cLat + dy * step
          lng = cLng + dx * step
          printf "%.6f %.6f\n", lat, lng
        }'
      ;;
    random|*)
      awk -v i="$i" -v cLat="$CENTER_LAT" -v cLng="$CENTER_LNG" \
        -v spread="${JITTER_DEG}" -v seed="${SEED}" '
        BEGIN {
          srand(i + seed + 1)
          lat = cLat + (rand() * 2 - 1) * spread
          lng = cLng + (rand() * 2 - 1) * spread
          printf "%.6f %.6f\n", lat, lng
        }'
      ;;
  esac
}

nearest_url_at() {
  local lat=$1
  local lng=$2
  echo "${BASE_URL}/getNearestBusStops?lat=${lat}&lng=${lng}"
}

warmup_nearest() {
  local i
  for ((i = 0; i < WARMUP; i++)); do
    read -r la ln < <(gen_lat_lng "$i")
    run_curl "$(nearest_url_at "${la}" "${ln}")" >/dev/null || true
  done
}

collect_times_fixed_url() {
  local url=$1
  local out_file=$2
  : >"${out_file}"
  local i line code t_total
  for ((i = 0; i < ITERATIONS; i++)); do
    line="$(run_curl "${url}")" || {
      echo "request failed for ${url}" >&2
      exit 1
    }
    code="$(echo "${line}" | awk -F'\t' '{print $1}')"
    t_total="$(echo "${line}" | awk -F'\t' '{print $2}')"
    if [[ "${code}" != "200" ]]; then
      echo "non-200 HTTP ${code} for ${url} (iteration $((i + 1)))" >&2
      echo "${line}" >&2
      exit 1
    fi
    echo "${t_total}" >>"${out_file}"
  done
}

# Varying lat/lng; writes one time_total per line to out_file. Uses burst parallelism (macOS bash 3.2 safe).
collect_nearest_varying() {
  local out_file=$1
  rm -f "${out_file}"
  local i_start bsz idx la ln url
  local wall_start wall_end
  wall_start=$(date +%s)
  for ((i_start = 0; i_start < ITERATIONS; i_start += CONCURRENCY)); do
    bsz=$CONCURRENCY
    ((i_start + bsz > ITERATIONS)) && bsz=$((ITERATIONS - i_start))
    local -a pids=()
    for ((b = 0; b < bsz; b++)); do
      idx=$((i_start + b))
      read -r la ln < <(gen_lat_lng "${idx}")
      url="$(nearest_url_at "${la}" "${ln}")"
      (
        line="$(run_curl "${url}")" || exit 2
        code="$(echo "${line}" | awk -F'\t' '{print $1}')"
        t_total="$(echo "${line}" | awk -F'\t' '{print $2}')"
        if [[ "${code}" != "200" ]]; then
          echo "non-200 HTTP ${code} url=${url}" >&2
          echo "${line}" >&2
          exit 3
        fi
        echo "${t_total}"
      ) >"${tmpdir}/slot_${idx}.txt" &
      pids+=($!)
    done
    local pid
    for pid in "${pids[@]}"; do
      wait "${pid}"
    done
    for ((b = 0; b < bsz; b++)); do
      idx=$((i_start + b))
      cat "${tmpdir}/slot_${idx}.txt" >>"${out_file}"
    done
  done
  wall_end=$(date +%s)
  echo $((wall_end - wall_start)) >"${tmpdir}/wall_seconds.txt"
}

summarize_file() {
  local path=$1
  if [[ ! -s "${path}" ]]; then
    echo "  (no samples)" >&2
    return 1
  fi
  sort -n "${path}" | awk '
    function ceil_pos(x) { return int(x) + (x > int(x) ? 1 : 0) }
    { n++; sum += $1; a[n] = $1 }
    END {
      if (n < 1) { print "no samples"; exit 1 }
      mean = sum / n
      minv = a[1]
      maxv = a[n]
      p50i = int((n + 1) / 2)
      if (p50i < 1) p50i = 1
      p95i = ceil_pos(n * 0.95)
      if (p95i < 1) p95i = 1
      if (p95i > n) p95i = n
      printf "  n=%d  mean=%.3fs  min=%.3fs  p50=%.3fs  p95=%.3fs  max=%.3fs\n", n, mean, minv, a[p50i], a[p95i], maxv
    }
  '
}

sample_breakdown_url() {
  local url=$1
  local line
  line="$(run_curl "${url}")"
  local code t_total t_lookup t_conn t_tls t_ttfb
  code="$(echo "${line}" | awk -F'\t' '{print $1}')"
  t_total="$(echo "${line}" | awk -F'\t' '{print $2}')"
  t_lookup="$(echo "${line}" | awk -F'\t' '{print $3}')"
  t_conn="$(echo "${line}" | awk -F'\t' '{print $4}')"
  t_tls="$(echo "${line}" | awk -F'\t' '{print $5}')"
  t_ttfb="$(echo "${line}" | awk -F'\t' '{print $6}')"
  echo "  (single sample) HTTP ${code}"
  echo "    time_total=${t_total}s  dns=${t_lookup}s  connect=${t_conn}s  tls=${t_tls}s  ttfb=${t_ttfb}s"
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

echo "when-bus API benchmark — /getNearestBusStops (varying lat/lng)"
echo "BASE_URL=${BASE_URL}  ITERATIONS=${ITERATIONS}  CONCURRENCY=${CONCURRENCY}  VARY=${VARY}"
echo "CENTER=${CENTER_LAT},${CENTER_LNG}  JITTER_DEG=${JITTER_DEG}  STEP_DEG=${STEP_DEG}"
echo ""

echo "Warmup (${WARMUP} requests, discarded)..."
warmup_nearest

echo ""
echo "GET /getNearestBusStops — varying coordinates, ${ITERATIONS} requests, concurrency ${CONCURRENCY} per burst"
collect_nearest_varying "${tmpdir}/nearest.txt"
summarize_file "${tmpdir}/nearest.txt"
read -r la ln < <(gen_lat_lng 0)
sample_breakdown_url "$(nearest_url_at "${la}" "${ln}")"
wall_s="$(cat "${tmpdir}/wall_seconds.txt" 2>/dev/null || echo "?")"
echo "  wall-clock for all nearest requests: ${wall_s}s (full run: bursts, not additive per-request RTT)"

if [[ "${INCLUDE_BUS_ARRIVAL}" == "1" ]]; then
  echo ""
  echo "Baseline: GET /getBusArrival?busStopCode=${BUS_STOP_CODE} (fixed URL, sequential)"
  echo "Warmup bus arrival..."
  for ((i = 0; i < WARMUP; i++)); do
    run_curl "${arrival_url}" >/dev/null || true
  done
  collect_times_fixed_url "${arrival_url}" "${tmpdir}/arrival.txt"
  summarize_file "${tmpdir}/arrival.txt"
  sample_breakdown_url "${arrival_url}"
fi

echo ""
echo "Interpretation:"
echo "  - Varying lat/lng exercises nearest-cache keys and stop selection; high CONCURRENCY stresses the server."
echo "  - Per-request stats are still single-request RTTs; wall-clock shows how long the full spam batch took."
echo "  - NEAREST_ONLY=1 skips /getBusArrival if you only want nearest spam."
