package main

import "net/http"

func newPort(service services) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", service.exas.HandleGet)
	mux.HandleFunc("POST /", service.exas.HandlePost)

	return mux
}
