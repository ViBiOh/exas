package main

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httputils"
)

func newPort(clients clients, services services) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", services.exas.HandleGet)
	mux.HandleFunc("POST /", services.exas.HandlePost)

	return httputils.Handler(mux, clients.health,
		clients.telemetry.Middleware("http"),
	)
}
