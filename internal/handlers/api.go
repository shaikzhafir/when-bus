package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/shaikzhafir/when-bus/internal/generated"
	log "github.com/shaikzhafir/when-bus/internal/logging"
	"github.com/shaikzhafir/when-bus/internal/services"
)

func writeJSONError(w http.ResponseWriter, status int, reason, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   message,
		"reason":  reason,
	})
}

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
	busArrival, err := a.service.GetBusArrival(params.BusStopCode)
	if err != nil {
		log.Error(
			"GetBusArrival failed: reason=upstream_or_parse_error busStopCode=%s method=%s query=%q remote_addr=%s err=%v",
			params.BusStopCode,
			r.Method,
			r.URL.RawQuery,
			r.RemoteAddr,
			err,
		)
		writeJSONError(w, http.StatusInternalServerError, "upstream_or_parse_error", err.Error())
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
		log.Error(
			"GetBusArrival encode failed: reason=json_encode_error busStopCode=%s err=%v",
			params.BusStopCode,
			err,
		)
		writeJSONError(w, http.StatusInternalServerError, "json_encode_error", err.Error())
		return
	}
}

// GetNearestBusStops implements the generated ServerInterface
func (a *APIHandler) GetNearestBusStops(w http.ResponseWriter, r *http.Request, params generated.GetNearestBusStopsParams) {
	nearestStops, err := a.service.GetNearestBusStops(params.Lat, params.Lng)
	if err != nil {
		log.Error(
			"GetNearestBusStops failed: reason=upstream_or_parse_error lat=%f lng=%f method=%s query=%q remote_addr=%s err=%v",
			params.Lat,
			params.Lng,
			r.Method,
			r.URL.RawQuery,
			r.RemoteAddr,
			err,
		)
		writeJSONError(w, http.StatusInternalServerError, "upstream_or_parse_error", err.Error())
		return
	}

	// Convert service NearestBusStopWithArrivals to generated format
	response := make(generated.NearestBusStopsResponse, len(nearestStops))
	for i, stop := range nearestStops {
		// Convert service BusDisplayInfo to generated BusDisplayInfo
		generatedArrivals := make([]generated.BusDisplayInfo, len(stop.Arrivals))
		for j, arrival := range stop.Arrivals {
			generatedArrivals[j] = generated.BusDisplayInfo{
				ServiceNo:    arrival.ServiceNo,
				Operator:     arrival.Operator,
				NextBuses:    arrival.NextBuses,
				LoadStatus:   arrival.LoadStatus,
				IsWheelchair: arrival.IsWheelchair,
			}
		}

		response[i] = generated.NearestBusStopWithArrivals{
			BusStopCode: stop.BusStopCode,
			RoadName:    stop.RoadName,
			Description: stop.Description,
			Distance:    float32(stop.Distance),
			Arrivals:    generatedArrivals,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error(
			"GetNearestBusStops encode failed: reason=json_encode_error lat=%f lng=%f err=%v",
			params.Lat,
			params.Lng,
			err,
		)
		writeJSONError(w, http.StatusInternalServerError, "json_encode_error", err.Error())
		return
	}
}
