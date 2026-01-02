package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/shaikzhafir/when-bus/internal/generated"
	"github.com/shaikzhafir/when-bus/internal/services"
)

func NewAPIHandler(svc services.Service) *APIHandler {
	return &APIHandler{service: svc}
}

type APIHandler struct {
	service services.Service
}

// Implement the generated ServerInterface
var _ generated.ServerInterface = (*APIHandler)(nil)

// GetFakeData implements the generated ServerInterface
func (a *APIHandler) GetFakeData(w http.ResponseWriter, r *http.Request, params generated.GetFakeDataParams) {
	name := ""
	if params.Name != nil {
		name = *params.Name
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("wow this is fake data!, u wrote " + name + " in the request!"))
}

// GetBusArrival implements the generated ServerInterface
func (a *APIHandler) GetBusArrival(w http.ResponseWriter, r *http.Request, params generated.GetBusArrivalParams) {
	log.Printf("Fetching bus arrival for bus stop code: %s", params.BusStopCode)
	busArrival, err := a.service.GetBusArrival(params.BusStopCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert service BusDisplayInfo to generated BusDisplayInfo
	generatedBusArrival := make([]generated.BusDisplayInfo, len(busArrival))
	for i, bus := range busArrival {
		generatedBusArrival[i] = generated.BusDisplayInfo{
			ServiceNo:    bus.ServiceNo,
			Operator:     bus.Operator,
			NextBuses:    bus.NextBuses,
			LoadStatus:   bus.LoadStatus,
			IsWheelchair: bus.IsWheelchair,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(generatedBusArrival); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetNearestBusStops implements the generated ServerInterface
func (a *APIHandler) GetNearestBusStops(w http.ResponseWriter, r *http.Request, params generated.GetNearestBusStopsParams) {
	log.Printf("Finding nearest bus stops for coordinates: lat=%f, lng=%f", params.Lat, params.Lng)
	busStopCodes, err := a.service.GetNearestBusStops(params.Lat, params.Lng)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := generated.NearestBusStopsResponse{
		BusStopCodes: busStopCodes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
