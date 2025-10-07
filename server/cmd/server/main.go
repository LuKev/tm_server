package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lukev/tm_server/internal/websocket"
)

func main() {
	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Set up router
	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, w, r)
	})

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// CORS middleware for development
	router.Use(corsMiddleware)

	// Start server
	addr := ":8080"
	log.Printf("Terra Mystica server starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
