package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := ":8000"
	if p := os.Getenv("APP_PORT"); p != "" {
		addr = ":" + p
	}
	greeting := os.Getenv("APP_GREETING")
	if greeting == "" {
		greeting = "hello from Elyro Workspace"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"message": greeting})
	})

	log.Printf("listening on http://0.0.0.0%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, payload map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":"encode"}`, http.StatusInternalServerError)
	}
}
