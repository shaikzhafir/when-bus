package main

import (
	"encoding/json"
	"net/http"
	"os"

	whenbus "github.com/shaikzhafir/when-bus"
	"github.com/shaikzhafir/when-bus/internal/generated"
	h "github.com/shaikzhafir/when-bus/internal/handlers"
	log "github.com/shaikzhafir/when-bus/internal/logging"
	"github.com/shaikzhafir/when-bus/internal/services"
)

const healthPath = "/health"

func main() {
	mux := initServeMux()
	if os.Getenv("ENV") == "prod" {
		prodUrl := os.Getenv("PROD_URL")
		log.Info("starting server on %s", prodUrl)
		err := http.ListenAndServe(prodUrl, mux)
		if err != nil {
			log.Error("error starting server: %v", err)
		}
	} else {
		log.Info("starting server on port 8080")
		err := http.ListenAndServe(":8080", mux)
		if err != nil {
			log.Error("error starting server: %v", err)
		}
	}
}

func initServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+healthPath, healthHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(whenbus.StaticFS())))
	mux.Handle("GET /{$}", http.FileServer(whenbus.PublicFS()))

	apiHandler := h.NewAPIHandler(services.NewService())
	generated.HandlerWithOptions(apiHandler, generated.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: h.OpenAPIValidationErrorHandler,
	})
	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
