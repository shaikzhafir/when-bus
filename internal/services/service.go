package services

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	log "github.com/shaikzhafir/when-bus/internal/logging"
)

const (
	busStopsCacheTTL  = 24 * time.Hour
	nearestCacheTTL   = 30 * time.Second
	httpClientTimeout = 10 * time.Second
	nearestStopsCount = 4
	// nearestCacheKeyDecimals: only used for nearest-metadata cache keys, not for Haversine.
	// 6 decimals ≈ 0.1 m latitude; distance always uses full float64 from the request.
	nearestCacheKeyDecimals = 6
)

type Service interface {
	GetBusArrival(code string) ([]BusDisplayInfo, error)
	GetNearestBusStops(lat, lng float64) ([]NearestBusStopWithArrivals, error)
}

// UI will be displaying this
type BusDisplayInfo struct {
	ServiceNo    string
	Operator     string
	NextBuses    []string // Array of arrival times in minutes
	LoadStatus   []string // Array of load statuses
	IsWheelchair bool     // WAB feature check
}

type BusArrivalResponse struct {
	Services []BusArrival `json:"Services"`
}

type BusArrival struct {
	ServiceNo string `json:"ServiceNo"`
	Operator  string `json:"Operator"`
	NextBus   struct {
		EstimatedArrival string `json:"EstimatedArrival"`
		Load             string `json:"Load"`
		Feature          string `json:"Feature"`
	} `json:"NextBus"`
	NextBus2 struct {
		EstimatedArrival string `json:"EstimatedArrival"`
		Load             string `json:"Load"`
		Feature          string `json:"Feature"`
	} `json:"NextBus2"`
	NextBus3 struct {
		EstimatedArrival string `json:"EstimatedArrival"`
		Load             string `json:"Load"`
		Feature          string `json:"Feature"`
	} `json:"NextBus3"`
}

type BusStop struct {
	BusStopCode string  `json:"BusStopCode"`
	RoadName    string  `json:"RoadName"`
	Description string  `json:"Description"`
	Latitude    float64 `json:"Latitude"`
	Longitude   float64 `json:"Longitude"`
}

type BusStopsResponse struct {
	Value []BusStop `json:"value"`
}

type busStopDistance struct {
	BusStopCode string
	RoadName    string
	Description string
	Distance    float64
}

// NearestBusStopWithArrivals represents a bus stop with its arrival information
type NearestBusStopWithArrivals struct {
	BusStopCode string
	RoadName    string
	Description string
	Distance    float64
	Arrivals    []BusDisplayInfo
}

type nearestCacheEntry struct {
	// Only nearest-stop metadata is cached; arrivals are fetched on every request
	// so NextBuses stay aligned with LTA (real-time).
	nearest   []busStopDistance
	fetchedAt time.Time
}

type service struct {
	client *http.Client

	busStopsBaseURL   string
	busArrivalBaseURL string

	// Bus stops are static infrastructure; cache them for 24h.
	busStopsMu   sync.RWMutex
	busStops     []BusStop
	busStopsTime time.Time

	// Short-lived cache for which stops are nearest (metadata only). Arrivals are
	// always refreshed so NextBuses are not stale on cache hits.
	nearestMu    sync.RWMutex
	nearestCache map[string]nearestCacheEntry
}

func NewService() Service {
	return &service{
		client:            &http.Client{Timeout: httpClientTimeout},
		busStopsBaseURL:   "https://datamall2.mytransport.sg/ltaodataservice/BusStops",
		busArrivalBaseURL: "https://datamall2.mytransport.sg/ltaodataservice/v3/BusArrival",
		nearestCache:      make(map[string]nearestCacheEntry),
	}
}

func (s *service) GetBusArrival(code string) ([]BusDisplayInfo, error) {
	req, err := http.NewRequest("GET", s.busArrivalBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	q := req.URL.Query()
	q.Add("BusStopCode", code)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("AccountKey", os.Getenv("LTA_API_KEY"))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LTA BusArrival API HTTP %d for BusStopCode=%q (body prefix: %.200q)", resp.StatusCode, code, string(body))
	}

	var busArrivalResp BusArrivalResponse
	if err := json.Unmarshal(body, &busArrivalResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling LTA BusArrival response for BusStopCode=%q: %v (body prefix: %.200q)", code, err, string(body))
	}

	var result []BusDisplayInfo
	for _, svc := range busArrivalResp.Services {
		info := BusDisplayInfo{
			ServiceNo:    svc.ServiceNo,
			Operator:     svc.Operator,
			NextBuses:    make([]string, 0, 3),
			LoadStatus:   make([]string, 0, 3),
			IsWheelchair: false,
		}

		buses := []struct {
			arrival string
			load    string
			feature string
		}{
			{svc.NextBus.EstimatedArrival, svc.NextBus.Load, svc.NextBus.Feature},
			{svc.NextBus2.EstimatedArrival, svc.NextBus2.Load, svc.NextBus2.Feature},
			{svc.NextBus3.EstimatedArrival, svc.NextBus3.Load, svc.NextBus3.Feature},
		}

		for _, bus := range buses {
			if bus.arrival != "" {
				arrivalTime, err := time.Parse(time.RFC3339, bus.arrival)
				if err == nil {
					minutesToArrival := time.Until(arrivalTime).Minutes()
					info.NextBuses = append(info.NextBuses, fmt.Sprintf("%.0f", minutesToArrival))
					info.LoadStatus = append(info.LoadStatus, bus.load)
				}
			}
			if bus.feature == "WAB" {
				info.IsWheelchair = true
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// haversineDistance calculates the distance between two points on Earth using the Haversine formula.
// Returns distance in kilometers.
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlng/2)*math.Sin(dlng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// loadBusStops returns the cached bus stop list, refreshing from LTA if the cache is expired or empty.
// Uses double-checked locking so only one goroutine fetches while others wait.
func (s *service) loadBusStops() ([]BusStop, error) {
	s.busStopsMu.RLock()
	if len(s.busStops) > 0 && time.Since(s.busStopsTime) < busStopsCacheTTL {
		stops := s.busStops
		s.busStopsMu.RUnlock()
		return stops, nil
	}
	s.busStopsMu.RUnlock()

	s.busStopsMu.Lock()
	defer s.busStopsMu.Unlock()

	if len(s.busStops) > 0 && time.Since(s.busStopsTime) < busStopsCacheTTL {
		return s.busStops, nil
	}

	stops, err := s.fetchAllBusStops()
	if err != nil {
		// If we have stale data, prefer returning it over failing
		if len(s.busStops) > 0 {
			log.Warn("failed to refresh bus stops cache, using stale data: %v", err)
			return s.busStops, nil
		}
		return nil, err
	}

	s.busStops = stops
	s.busStopsTime = time.Now()
	log.Info("refreshed bus stops cache: %d stops loaded", len(stops))
	return stops, nil
}

func (s *service) fetchAllBusStops() ([]BusStop, error) {
	allBusStops := make([]BusStop, 0, 5500)
	skip := 0
	const pageSize = 500

	for {
		url := fmt.Sprintf("%s?$skip=%d", s.busStopsBaseURL, skip)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Add("AccountKey", os.Getenv("LTA_API_KEY"))

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %v", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %v", err)
		}

		var busStopsResp BusStopsResponse
		if err := json.Unmarshal(body, &busStopsResp); err != nil {
			return nil, fmt.Errorf("error unmarshaling response: %v", err)
		}

		allBusStops = append(allBusStops, busStopsResp.Value...)

		if len(busStopsResp.Value) < pageSize {
			break
		}

		skip += pageSize
	}

	return allBusStops, nil
}

// nearestCacheKey rounds lat/lng for the metadata cache only (see nearestCacheKeyDecimals).
// Haversine distance always uses the full-precision lat/lng from the request.
func nearestCacheKey(lat, lng float64) string {
	return fmt.Sprintf("%.*f,%.*f", nearestCacheKeyDecimals, lat, nearestCacheKeyDecimals, lng)
}

// fetchArrivalsForNearest runs one BusArrival request per stop, concurrently.
func (s *service) fetchArrivalsForNearest(nearest []busStopDistance) []NearestBusStopWithArrivals {
	count := len(nearest)
	result := make([]NearestBusStopWithArrivals, count)
	var wg sync.WaitGroup
	for i, stop := range nearest {
		wg.Add(1)
		go func(idx int, d busStopDistance) {
			defer wg.Done()
			arrivals, err := s.GetBusArrival(d.BusStopCode)
			if err != nil {
				arrivals = []BusDisplayInfo{}
			}
			result[idx] = NearestBusStopWithArrivals{
				BusStopCode: d.BusStopCode,
				RoadName:    d.RoadName,
				Description: d.Description,
				Distance:    d.Distance,
				Arrivals:    arrivals,
			}
		}(i, stop)
	}
	wg.Wait()
	return result
}

func (s *service) GetNearestBusStops(lat, lng float64) ([]NearestBusStopWithArrivals, error) {
	key := nearestCacheKey(lat, lng)

	s.nearestMu.RLock()
	entry, cacheOk := s.nearestCache[key]
	if cacheOk && time.Since(entry.fetchedAt) < nearestCacheTTL {
		meta := entry.nearest
		s.nearestMu.RUnlock()
		log.Info("nearest bus stops cache hit for key=%s (refreshing arrivals)", key)
		return s.fetchArrivalsForNearest(meta), nil
	}
	s.nearestMu.RUnlock()

	allBusStops, err := s.loadBusStops()
	if err != nil {
		return nil, err
	}

	distances := make([]busStopDistance, 0, len(allBusStops))
	for _, stop := range allBusStops {
		distance := haversineDistance(lat, lng, stop.Latitude, stop.Longitude)
		distances = append(distances, busStopDistance{
			BusStopCode: stop.BusStopCode,
			RoadName:    stop.RoadName,
			Description: stop.Description,
			Distance:    distance,
		})
	}

	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance < distances[j].Distance
	})

	n := nearestStopsCount
	if len(distances) < n {
		n = len(distances)
	}
	// Copy only the nearest n rows so the cache does NOT pin the full ~5k distances slice.
	nearestMeta := make([]busStopDistance, n)
	copy(nearestMeta, distances[:n])

	out := s.fetchArrivalsForNearest(nearestMeta)

	s.nearestMu.Lock()
	s.nearestCache[key] = nearestCacheEntry{
		nearest:   nearestMeta,
		fetchedAt: time.Now(),
	}
	for k, v := range s.nearestCache {
		if time.Since(v.fetchedAt) > nearestCacheTTL*2 {
			delete(s.nearestCache, k)
		}
	}
	s.nearestMu.Unlock()

	return out, nil
}
