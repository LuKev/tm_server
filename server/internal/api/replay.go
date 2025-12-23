package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lukev/tm_server/internal/replay"
)

type ReplayHandler struct {
	manager *replay.ReplayManager
}

func NewReplayHandler(manager *replay.ReplayManager) *ReplayHandler {
	return &ReplayHandler{manager: manager}
}

func (h *ReplayHandler) RegisterRoutes(router *mux.Router) {
	s := router.PathPrefix("/api/replay").Subrouter()
	s.HandleFunc("/start", h.handleStart).Methods("POST")
	s.HandleFunc("/next", h.handleNext).Methods("POST")
	s.HandleFunc("/state", h.handleState).Methods("GET")
	s.HandleFunc("/provide_info", h.handleProvideInfo).Methods("POST")
}

func (h *ReplayHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleStart called")
	var req struct {
		GameID string `json:"gameId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("handleStart decode error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("handleStart for game %s\n", req.GameID)

	session, err := h.manager.StartReplay(req.GameID)
	if err != nil {
		fmt.Printf("StartReplay error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	players := make([]string, 0)
	if session.Simulator.GetState() != nil {
		for p := range session.Simulator.GetState().Players {
			players = append(players, p)
		}
	}

	resp := map[string]interface{}{
		"gameId":       session.GameID,
		"missingInfo":  session.MissingInfo,
		"players":      players,
		"currentIndex": session.Simulator.CurrentIndex,
		"totalActions": len(session.Simulator.Actions),
		"logStrings":   session.LogStrings,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf("handleStart encode error: %v\n", err)
	} else {
		fmt.Println("handleStart success")
	}
}

func (h *ReplayHandler) handleNext(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string `json:"gameId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("handleNext called for game %s\n", req.GameID)

	session := h.manager.GetSession(req.GameID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := session.Simulator.StepForward(); err != nil {
		fmt.Printf("StepForward failed for game %s: %v\n", req.GameID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("StepForward success for game %s, returning state\n", req.GameID)
	// Return new state
	state := session.Simulator.GetState()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (h *ReplayHandler) handleState(w http.ResponseWriter, r *http.Request) {
	gameID := r.URL.Query().Get("gameId")
	fmt.Printf("handleState called for game %s\n", gameID)
	if gameID == "" {
		http.Error(w, "missing gameId", http.StatusBadRequest)
		return
	}

	session := h.manager.GetSession(gameID)
	if session == nil {
		fmt.Printf("handleState: session not found for %s\n", gameID)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	state := session.Simulator.GetState()
	if state == nil {
		fmt.Printf("handleState: state is nil for %s\n", gameID)
		http.Error(w, "state is nil", http.StatusInternalServerError)
		return
	}
	fmt.Printf("handleState: returning state for %s. Players: %d\n", gameID, len(state.Players))
	for pid := range state.Players {
		fmt.Printf(" - Player: %s\n", pid)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(state); err != nil {
		fmt.Printf("handleState encode error: %v\n", err)
	} else {
		fmt.Println("handleState success")
	}
}

func (h *ReplayHandler) handleProvideInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string                   `json:"gameId"`
		Info   *replay.ProvidedGameInfo `json:"info"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("handleProvideInfo called for game %s\n", req.GameID)

	session := h.manager.GetSession(req.GameID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := h.manager.ProvideInfo(req.GameID, req.Info); err != nil {
		fmt.Printf("ProvideInfo failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
