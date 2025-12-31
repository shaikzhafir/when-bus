package main

import (
	"net/http"
	"os"

	"github.com/shaikzhafir/when-bus/internal/generated"
	h "github.com/shaikzhafir/when-bus/internal/handlers"
	log "github.com/shaikzhafir/when-bus/internal/logging"
	"github.com/shaikzhafir/when-bus/internal/services"
)

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
	// Register API handlers using generated code
	apiHandler := h.NewAPIHandler(services.NewService())
	generated.HandlerFromMux(apiHandler, mux)
	return mux
}
