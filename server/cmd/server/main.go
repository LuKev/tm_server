package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/lukev/tm_server/internal/api"
	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/lobby"
	"github.com/lukev/tm_server/internal/replay"
	"github.com/lukev/tm_server/internal/websocket"
)

func main() {
	if err := verifyRequiredNeuralEvaluator(); err != nil {
		log.Fatal(err)
	}
	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create managers
	gameMgr := game.NewManager()
	lobbyMgr := lobby.NewManager()
	botMgr := websocket.NewBotManager(gameMgr)
	// Get scripts directory from environment or default to relative path
	scriptDir := os.Getenv("SCRIPTS_DIR")
	if scriptDir == "" {
		scriptDir = "./scripts"
	}
	replayMgr := replay.NewReplayManager(scriptDir)
	replayMgr.SetSourceAnchoredLeechOrdering(true)
	replayHandler := api.NewReplayHandler(replayMgr)
	aiHandler := api.NewAIHandler(gameMgr)

	deps := websocket.ServerDeps{
		Lobby: lobbyMgr,
		Games: gameMgr,
		Bots:  botMgr,
	}

	// Set up router
	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/api/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, deps, w, r)
	})

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// CORS middleware for development
	router.Use(corsMiddleware)

	// Register replay routes
	replayHandler.RegisterRoutes(router)
	aiHandler.RegisterRoutes(router)

	// Start server
	addr := strings.TrimSpace(os.Getenv("PORT"))
	if addr == "" {
		addr = "8080"
	}
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	log.Printf("Terra Mystica server starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func verifyRequiredNeuralEvaluator() error {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("TM_AZ_REQUIRE_NEURAL")), "true") {
		return nil
	}
	modelURL := strings.TrimSpace(os.Getenv("TM_AZ_MODEL_URL"))
	if modelURL == "" {
		return fmt.Errorf("TM_AZ_REQUIRE_NEURAL=true requires TM_AZ_MODEL_URL")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	health, err := model.ProbeHTTP(modelURL, client)
	if err != nil {
		return fmt.Errorf("required neural evaluator is unavailable: %w", err)
	}
	log.Printf("neural evaluator ready: architecture=%s actions=%d input=%d schema=%s", health.Architecture, health.ActionCount, health.InputSize, health.ObservationSchema)
	return nil
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
