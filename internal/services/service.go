package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Service interface {
	GetBusArrival(code string) ([]BusDisplayInfo, error)
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
