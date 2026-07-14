package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := getenv("RELAY_HTTP_ADDR", ":8090")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service": "coderoam-relay",
			"status":  "ok",
		})
	})
	mux.HandleFunc("GET /v1/connect", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "WebSocket relay handshake is not implemented in the starter", http.StatusNotImplemented)
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("CodeRoam relay listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
