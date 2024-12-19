package main

import (
	"fmt"
	"net/http"

	"github.com/shaikzhafir/go-htmx-starter/internal/commons"
	h "github.com/shaikzhafir/go-htmx-starter/internal/handlers"
	log "github.com/shaikzhafir/go-htmx-starter/internal/logging"
	"github.com/shaikzhafir/go-htmx-starter/internal/services"
)

// Config configures the main ServeMux.
type Config struct {
	ClientID     string
	ClientSecret string
}

func main() {
	mux := initServeMux()
	log.Info("Server started on port %d", commons.DefaultPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", commons.DefaultPort), mux)
	if err != nil {
		log.Error("Error starting server %s", err.Error())
	}
}

func initServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	handlers := h.NewAPIHandler(services.NewService())
	// css and js files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// other paths
	mux.HandleFunc("/api", handlers.GetFakeData())
	mux.HandleFunc("/getBusArrival", handlers.GetBusArrival())
	// html files (also catch all paths)
	fs := http.FileServer(http.Dir("frontend/dist"))
	mux.Handle("/", fs)
	return mux
}
