package services

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTestService creates a service wired to the given mock servers.
func newTestService(busStopsURL, busArrivalURL string) *service {
	return &service{
		client:            &http.Client{Timeout: 5 * time.Second},
		busStopsBaseURL:   busStopsURL,
		busArrivalBaseURL: busArrivalURL,
		nearestCache:      make(map[string]nearestCacheEntry),
	}
}

// ---------- helpers to build mock LTA JSON responses ----------

func busStopsJSON(stops []BusStop) []byte {
	resp := BusStopsResponse{Value: stops}
	b, _ := json.Marshal(resp)
	return b
}

func busArrivalJSON(services []BusArrival) []byte {
	resp := BusArrivalResponse{Services: services}
	b, _ := json.Marshal(resp)
	return b
}

func arrivalTimeFromNow(minutes int) string {
	return time.Now().Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339)
}

// testBusStops returns 6 bus stops at known Singapore-ish coordinates.
func testBusStops() []BusStop {
	return []BusStop{
		{BusStopCode: "11111", RoadName: "Orchard Rd", Description: "Orchard Stn", Latitude: 1.3048, Longitude: 103.8318},
		{BusStopCode: "22222", RoadName: "Orchard Rd", Description: "ION Orchard", Latitude: 1.3040, Longitude: 103.8321},
		{BusStopCode: "33333", RoadName: "Scotts Rd", Description: "Far East Plaza", Latitude: 1.3070, Longitude: 103.8330},
		{BusStopCode: "44444", RoadName: "Paterson Rd", Description: "Paterson Stn", Latitude: 1.3010, Longitude: 103.8300},
		{BusStopCode: "55555", RoadName: "Tanglin Rd", Description: "Tanglin Mall", Latitude: 1.2990, Longitude: 103.8250},
		{BusStopCode: "66666", RoadName: "Bukit Timah", Description: "Botanic Gardens", Latitude: 1.3220, Longitude: 103.8150},
	}
}

// mockBusStopsServer returns a server that serves the bus stops list in a single page.
func mockBusStopsServer(stops []BusStop) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(busStopsJSON(stops))
	}))
}

// mockBusArrivalServer returns a server that responds with a fixed arrival payload for any bus stop code.
func mockBusArrivalServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		arrivals := []BusArrival{
			{
				ServiceNo: "36",
				Operator:  "SBST",
				NextBus: struct {
					EstimatedArrival string `json:"EstimatedArrival"`
					Load             string `json:"Load"`
					Feature          string `json:"Feature"`
				}{EstimatedArrival: arrivalTimeFromNow(3), Load: "SEA", Feature: "WAB"},
				NextBus2: struct {
					EstimatedArrival string `json:"EstimatedArrival"`
					Load             string `json:"Load"`
					Feature          string `json:"Feature"`
				}{EstimatedArrival: arrivalTimeFromNow(8), Load: "SDA", Feature: ""},
				NextBus3: struct {
					EstimatedArrival string `json:"EstimatedArrival"`
					Load             string `json:"Load"`
					Feature          string `json:"Feature"`
				}{EstimatedArrival: arrivalTimeFromNow(15), Load: "LSD", Feature: ""},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(busArrivalJSON(arrivals))
	}))
}

// ---------- Unit tests ----------

func TestHaversineDistance(t *testing.T) {
	// Orchard MRT to Botanic Gardens MRT ≈ 2.2 km
	d := haversineDistance(1.3048, 103.8318, 1.3220, 103.8150)
	if d < 1.5 || d > 3.0 {
		t.Fatalf("expected ~2.2 km, got %.2f km", d)
	}

	// Same point should be 0
	d = haversineDistance(1.3, 103.8, 1.3, 103.8)
	if d != 0 {
		t.Fatalf("expected 0, got %f", d)
	}
}

func TestNearestCacheKey_RoundsCoordinates(t *testing.T) {
	// Same bucket at nearestCacheKeyDecimals (6)
	k1 := nearestCacheKey(1.304811111, 103.831811111)
	k2 := nearestCacheKey(1.304811114, 103.831811114)
	if k1 != k2 {
		t.Fatalf("expected same key for nearby coords, got %q vs %q", k1, k2)
	}

	k3 := nearestCacheKey(1.3050, 103.8320)
	if k1 == k3 {
		t.Fatalf("expected different key for distant coords, both got %q", k1)
	}
}

func TestGetBusArrival_ParsesResponse(t *testing.T) {
	srv := mockBusArrivalServer()
	defer srv.Close()

	svc := newTestService("", srv.URL)

	result, err := svc.GetBusArrival("11111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result))
	}
	if result[0].ServiceNo != "36" {
		t.Errorf("expected service 36, got %s", result[0].ServiceNo)
	}
	if len(result[0].NextBuses) != 3 {
		t.Errorf("expected 3 arrival times, got %d", len(result[0].NextBuses))
	}
	if !result[0].IsWheelchair {
		t.Error("expected wheelchair accessible (WAB feature on NextBus)")
	}
	if len(result[0].LoadStatus) != 3 {
		t.Errorf("expected 3 load statuses, got %d", len(result[0].LoadStatus))
	}
}

func TestGetBusArrival_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	svc := newTestService("", srv.URL)
	_, err := svc.GetBusArrival("99999")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestGetBusArrival_EmptyServices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(busArrivalJSON(nil))
	}))
	defer srv.Close()

	svc := newTestService("", srv.URL)
	result, err := svc.GetBusArrival("11111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 services, got %d", len(result))
	}
}

// ---------- Bus stops cache tests ----------

func TestLoadBusStops_CachesResults(t *testing.T) {
	var callCount atomic.Int32
	stops := testBusStops()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(busStopsJSON(stops))
	}))
	defer srv.Close()

	svc := newTestService(srv.URL, "")

	// First call should hit the server
	result, err := svc.loadBusStops()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(stops) {
		t.Fatalf("expected %d stops, got %d", len(stops), len(result))
	}
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", callCount.Load())
	}

	// Second call should use cache — no additional HTTP call
	result2, err := svc.loadBusStops()
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if len(result2) != len(stops) {
		t.Fatalf("cached result length mismatch: %d", len(result2))
	}
	if callCount.Load() != 1 {
		t.Fatalf("expected still 1 HTTP call after cache hit, got %d", callCount.Load())
	}
}

func TestLoadBusStops_StaleDataFallback(t *testing.T) {
	stops := testBusStops()
	svc := newTestService("http://127.0.0.1:1", "")

	// Pre-populate with stale data
	svc.busStops = stops
	svc.busStopsTime = time.Now().Add(-48 * time.Hour) // expired

	result, err := svc.loadBusStops()
	if err != nil {
		t.Fatalf("expected stale fallback, got error: %v", err)
	}
	if len(result) != len(stops) {
		t.Fatalf("expected stale data (%d stops), got %d", len(stops), len(result))
	}
}

func TestLoadBusStops_ErrorWhenNoCacheAndServerDown(t *testing.T) {
	svc := newTestService("http://127.0.0.1:1", "")

	_, err := svc.loadBusStops()
	if err == nil {
		t.Fatal("expected error when server is down and no cached data")
	}
}

func TestLoadBusStops_PaginatedFetch(t *testing.T) {
	page1 := make([]BusStop, 500)
	for i := range page1 {
		page1[i] = BusStop{BusStopCode: "A" + string(rune(i)), Latitude: 1.3, Longitude: 103.8}
	}
	page2 := testBusStops() // 6 stops — short page signals end

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		skip := r.URL.Query().Get("$skip")
		if skip == "0" || skip == "" {
			w.Write(busStopsJSON(page1))
		} else {
			w.Write(busStopsJSON(page2))
		}
	}))
	defer srv.Close()

	svc := newTestService(srv.URL, "")
	result, err := svc.loadBusStops()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 506 {
		t.Fatalf("expected 506 stops (500 + 6), got %d", len(result))
	}
	if callCount.Load() != 2 {
		t.Fatalf("expected 2 paginated HTTP calls, got %d", callCount.Load())
	}
}

// ---------- Nearest bus stops cache tests ----------

func TestGetNearestBusStops_ReturnsNearest4(t *testing.T) {
	stopsSrv := mockBusStopsServer(testBusStops())
	defer stopsSrv.Close()
	arrivalSrv := mockBusArrivalServer()
	defer arrivalSrv.Close()

	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)

	result, err := svc.GetNearestBusStops(1.3048, 103.8318)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != nearestStopsCount {
		t.Fatalf("expected %d nearest stops, got %d", nearestStopsCount, len(result))
	}

	// Results should be sorted by distance ascending
	for i := 1; i < len(result); i++ {
		if result[i].Distance < result[i-1].Distance {
			t.Errorf("results not sorted by distance: [%d]=%.4f < [%d]=%.4f",
				i, result[i].Distance, i-1, result[i-1].Distance)
		}
	}

	// Each stop should have arrivals populated
	for i, stop := range result {
		if len(stop.Arrivals) == 0 {
			t.Errorf("stop %d (%s) has no arrivals", i, stop.BusStopCode)
		}
	}
}

func TestGetNearestBusStops_CacheHit(t *testing.T) {
	var arrivalCalls atomic.Int32
	var busStopCalls atomic.Int32
	stopsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		busStopCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(busStopsJSON(testBusStops()))
	}))
	defer stopsSrv.Close()

	arrivalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		arrivalCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(busArrivalJSON(nil))
	}))
	defer arrivalSrv.Close()

	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)

	// First call populates cache
	_, err := svc.GetNearestBusStops(1.3048, 103.8318)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	firstArrivals := arrivalCalls.Load()
	if busStopCalls.Load() != 1 {
		t.Fatalf("expected 1 bus-stops fetch, got %d", busStopCalls.Load())
	}

	// Second call: metadata cache hit skips bus-stop list work, arrivals still refetched.
	_, err = svc.GetNearestBusStops(1.3048, 103.8318)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if busStopCalls.Load() != 1 {
		t.Fatalf("expected still 1 bus-stops HTTP call on cache hit, got %d", busStopCalls.Load())
	}
	want := firstArrivals + nearestStopsCount
	if arrivalCalls.Load() != want {
		t.Fatalf("expected %d arrival calls after cache hit refresh, got %d", want, arrivalCalls.Load())
	}
}

func TestGetNearestBusStops_NearbyCoordsCacheHit(t *testing.T) {
	var arrivalCalls atomic.Int32
	stopsSrv := mockBusStopsServer(testBusStops())
	defer stopsSrv.Close()

	arrivalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		arrivalCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(busArrivalJSON(nil))
	}))
	defer arrivalSrv.Close()

	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)

	// First call
	_, err := svc.GetNearestBusStops(1.304811111, 103.831811111)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	firstCount := arrivalCalls.Load()

	// Coordinates that share the same 6-decimal cache key
	_, err = svc.GetNearestBusStops(1.304811114, 103.831811114)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := firstCount + nearestStopsCount
	if arrivalCalls.Load() != want {
		t.Fatalf("expected %d total arrival calls after metadata cache hit, got %d",
			want, arrivalCalls.Load())
	}
}

func TestGetNearestBusStops_ArrivalErrorsDoNotFail(t *testing.T) {
	stopsSrv := mockBusStopsServer(testBusStops())
	defer stopsSrv.Close()
	arrivalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer arrivalSrv.Close()

	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)

	result, err := svc.GetNearestBusStops(1.3048, 103.8318)
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if len(result) != nearestStopsCount {
		t.Fatalf("expected %d stops even with failed arrivals, got %d", nearestStopsCount, len(result))
	}
	for _, stop := range result {
		if len(stop.Arrivals) != 0 {
			t.Errorf("expected empty arrivals on error, got %d for %s", len(stop.Arrivals), stop.BusStopCode)
		}
	}
}

// ---------- Concurrency and goroutine leak tests ----------

func TestGetNearestBusStops_ConcurrentCallersNoRace(t *testing.T) {
	stopsSrv := mockBusStopsServer(testBusStops())
	defer stopsSrv.Close()
	arrivalSrv := mockBusArrivalServer()
	defer arrivalSrv.Close()

	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := svc.GetNearestBusStops(1.3048, 103.8318)
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d returned error: %v", i, err)
		}
	}
}

func TestGetNearestBusStops_NoGoroutineLeak(t *testing.T) {
	stopsSrv := mockBusStopsServer(testBusStops())
	defer stopsSrv.Close()
	arrivalSrv := mockBusArrivalServer()
	defer arrivalSrv.Close()

	// Use a transport with keep-alives disabled so idle HTTP connections
	// don't inflate the goroutine count.
	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)
	svc.client = &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}

	// Stabilize goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := runtime.NumGoroutine()

	for i := 0; i < 10; i++ {
		_, err := svc.GetNearestBusStops(1.3+float64(i)*0.01, 103.83)
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
	}

	// Give goroutines time to wind down
	runtime.GC()
	time.Sleep(300 * time.Millisecond)
	after := runtime.NumGoroutine()

	leaked := after - before
	if leaked > 2 {
		t.Fatalf("possible goroutine leak: %d before, %d after (delta=%d)", before, after, leaked)
	}
}

func TestGetNearestBusStops_SlowArrivalDoesNotHang(t *testing.T) {
	stopsSrv := mockBusStopsServer(testBusStops())
	defer stopsSrv.Close()

	arrivalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write(busArrivalJSON(nil))
	}))
	defer arrivalSrv.Close()

	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)

	start := time.Now()
	result, err := svc.GetNearestBusStops(1.3048, 103.8318)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != nearestStopsCount {
		t.Fatalf("expected %d stops, got %d", nearestStopsCount, len(result))
	}

	// With 4 concurrent 100ms calls, total should be ~100ms, not ~400ms.
	// Use a generous bound to avoid flaky tests.
	if elapsed > 350*time.Millisecond {
		t.Fatalf("arrivals appear to be fetched sequentially: took %v (expected <350ms for concurrent 100ms calls)", elapsed)
	}
}

func TestLoadBusStops_ConcurrentFirstLoad(t *testing.T) {
	var callCount atomic.Int32
	stops := testBusStops()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		time.Sleep(50 * time.Millisecond) // simulate latency
		w.Header().Set("Content-Type", "application/json")
		w.Write(busStopsJSON(stops))
	}))
	defer srv.Close()

	svc := newTestService(srv.URL, "")

	const goroutines = 10
	var wg sync.WaitGroup
	results := make([][]BusStop, goroutines)
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			r, err := svc.loadBusStops()
			results[idx] = r
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d returned error: %v", i, err)
		}
	}
	for i, r := range results {
		if len(r) != len(stops) {
			t.Errorf("goroutine %d got %d stops, expected %d", i, len(r), len(stops))
		}
	}

	// Double-checked locking: only 1 goroutine should have actually fetched.
	// Under high contention some may slip through before the write lock is acquired,
	// but typically we expect very few calls.
	if callCount.Load() > 2 {
		t.Errorf("expected <=2 HTTP calls due to double-checked locking, got %d", callCount.Load())
	}
}

func TestNearestCacheEviction(t *testing.T) {
	stopsSrv := mockBusStopsServer(testBusStops())
	defer stopsSrv.Close()
	arrivalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(busArrivalJSON(nil))
	}))
	defer arrivalSrv.Close()

	svc := newTestService(stopsSrv.URL, arrivalSrv.URL)

	// Manually insert an old cache entry
	svc.nearestMu.Lock()
	svc.nearestCache["0.0000,0.0000"] = nearestCacheEntry{
		nearest:   nil,
		fetchedAt: time.Now().Add(-5 * time.Minute),
	}
	svc.nearestMu.Unlock()

	// Trigger a new query which writes to cache and runs lazy eviction
	_, err := svc.GetNearestBusStops(1.3048, 103.8318)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc.nearestMu.RLock()
	_, staleExists := svc.nearestCache["0.0000,0.0000"]
	svc.nearestMu.RUnlock()

	if staleExists {
		t.Error("expected stale cache entry to be evicted")
	}
}

func TestHaversineDistance_Symmetry(t *testing.T) {
	d1 := haversineDistance(1.3048, 103.8318, 1.3220, 103.8150)
	d2 := haversineDistance(1.3220, 103.8150, 1.3048, 103.8318)
	if math.Abs(d1-d2) > 1e-10 {
		t.Fatalf("haversine should be symmetric: %.10f vs %.10f", d1, d2)
	}
}

func TestGetBusArrival_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	svc := newTestService("", srv.URL)
	_, err := svc.GetBusArrival("11111")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}
