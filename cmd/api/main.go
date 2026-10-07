package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	return mux
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           newMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("auth-api listening on", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
