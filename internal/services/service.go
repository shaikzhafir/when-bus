package services

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"time"

	log "github.com/shaikzhafir/when-bus/internal/logging"
)

type Service interface {
	GetBusArrival(code string) ([]BusDisplayInfo, error)
	GetNearestBusStops(lat, lng float32) ([]string, error)
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
	Distance    float64
}

type service struct {
}

func NewService() Service {
	return &service{}
}

func (s *service) GetBusArrival(code string) ([]BusDisplayInfo, error) {
	// write an api call to get bus arrival
	// api url is https://datamall2.mytransport.sg/ltaodataservice/v3/BusArrival?BusStopCode=71119
	// it needs an api key
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://datamall2.mytransport.sg/ltaodataservice/v3/BusArrival", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add query parameter
	q := req.URL.Query()
	q.Add("BusStopCode", code)
	req.URL.RawQuery = q.Encode()

	// Add API key header
	req.Header.Add("AccountKey", os.Getenv("LTA_API_KEY"))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}
	busArrivalResp := BusArrivalResponse{}

	if err := json.Unmarshal(body, &busArrivalResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
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

		// Process arrival times and load status
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

// haversineDistance calculates the distance between two points on Earth using the Haversine formula
// Returns distance in kilometers
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlng/2)*math.Sin(dlng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadiusKm * c

	return distance
}

func (s *service) GetNearestBusStops(lat, lng float32) ([]string, error) {
	// Fetch all bus stops from LTA API (paginated, 500 per page)
	client := &http.Client{}
	allBusStops := make([]BusStop, 0)
	skip := 0
	const pageSize = 500

	for {
		// Build URL with skip parameter
		url := fmt.Sprintf("https://datamall2.mytransport.sg/ltaodataservice/v3/BusStops?$skip=%d", skip)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		// Add API key header
		req.Header.Add("AccountKey", os.Getenv("LTA_API_KEY"))

		resp, err := client.Do(req)
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

		// Append bus stops from this page
		allBusStops = append(allBusStops, busStopsResp.Value...)

		// If we got less than pageSize results, we've reached the end
		if len(busStopsResp.Value) < pageSize {
			break
		}

		// Move to next page
		skip += pageSize
	}

	log.Info("Fetched %d bus stops total", len(allBusStops))

	// Calculate distance for each bus stop
	distances := make([]busStopDistance, 0, len(allBusStops))
	for _, stop := range allBusStops {
		distance := haversineDistance(float64(lat), float64(lng), stop.Latitude, stop.Longitude)
		distances = append(distances, busStopDistance{
			BusStopCode: stop.BusStopCode,
			Distance:    distance,
		})
	}

	// Sort by distance
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance < distances[j].Distance
	})

	// Return the nearest 2 bus stop codes
	result := make([]string, 0, 2)
	for i := 0; i < 2 && i < len(distances); i++ {
		result = append(result, distances[i].BusStopCode)
	}

	return result, nil
}
