package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/shaikzhafir/go-htmx-starter/internal/services"
)

func NewAPIHandler(svc services.Service) *APIHandler {
	return &APIHandler{service: svc}
}

type APIHandler struct {
	service services.Service
}

func (a *APIHandler) GetFakeData() http.HandlerFunc {
	// handle the request
	return func(w http.ResponseWriter, r *http.Request) {
		// get params from request
		// do some processing
		// return response
		name := r.FormValue("name")
		w.Write([]byte("wow this is fake data!, u wrote " + name + "in the request!"))
	}
}

func (a *APIHandler) GetBusArrival() http.HandlerFunc {
	// render profile page
	return func(w http.ResponseWriter, r *http.Request) {
		// render profile page
		busArrival, err := a.service.GetBusArrival()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(busArrival); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
