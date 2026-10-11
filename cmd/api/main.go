package main

import (
	"log"
	"net/http"
	"time"

	"github.com/TMseabra/auth-api/internal/config"
)

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	return mux
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}` + "\n"))
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("auth-api listening on", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
