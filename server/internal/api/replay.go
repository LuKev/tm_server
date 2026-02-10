package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lukev/tm_server/internal/game"
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
	s.HandleFunc("/import", h.handleImport).Methods("POST")
	s.HandleFunc("/import_text", h.handleImportText).Methods("POST")
	s.HandleFunc("/import_form", h.handleImportForm).Methods("POST")
	s.HandleFunc("/next", h.handleNext).Methods("POST")
	s.HandleFunc("/jump", h.handleJump).Methods("POST")
	s.HandleFunc("/state", h.handleState).Methods("GET")
	s.HandleFunc("/snapshot", h.handleSnapshot).Methods("GET")
	s.HandleFunc("/provide_info", h.handleProvideInfo).Methods("POST")
}

func (h *ReplayHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleStart called")
	var req struct {
		GameID  string `json:"gameId"`
		Restart bool   `json:"restart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("handleStart decode error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("handleStart for game %s (restart=%v)\n", req.GameID, req.Restart)

	session, err := h.manager.StartReplay(req.GameID, req.Restart)
	if err != nil {
		fmt.Printf("StartReplay error: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
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
		"logLocations": session.LogLocations,
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

	session := h.manager.GetSession(req.GameID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := session.Simulator.StepForward(); err != nil {
		fmt.Printf("StepForward failed for game %s: %v\n", req.GameID, err)

		// Check if it's a MissingInfoError - return structured JSON
		if missingErr, ok := err.(*game.MissingInfoError); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) // 409 Conflict for missing info
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":            missingErr.Error(),
				"type":             missingErr.Type,
				"players":          missingErr.Players,
				"round":            missingErr.Round,
				"allMissingPasses": missingErr.AllMissingPasses,
			})
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return new state
	state := session.Simulator.GetState()
	serialized := game.SerializeState(state, req.GameID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serialized)
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
	serialized := game.SerializeState(state, gameID)
	if err := json.NewEncoder(w).Encode(serialized); err != nil {
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

func (h *ReplayHandler) handleJump(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string `json:"gameId"`
		Index  int    `json:"index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.manager.JumpTo(req.GameID, req.Index); err != nil {
		fmt.Printf("JumpTo failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return new state
	session := h.manager.GetSession(req.GameID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	state := session.Simulator.GetState()
	serialized := game.SerializeState(state, req.GameID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serialized)
}

func (h *ReplayHandler) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	gameID := r.URL.Query().Get("gameId")
	if gameID == "" {
		http.Error(w, "missing gameId", http.StatusBadRequest)
		return
	}

	session := h.manager.GetSession(gameID)
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	state := session.Simulator.GetState()
	if state == nil {
		http.Error(w, "state is nil", http.StatusInternalServerError)
		return
	}

	snapshot := replay.GenerateSnapshot(state)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"snapshot_%s_%d.yaml\"", gameID, session.Simulator.CurrentIndex))
	w.Write([]byte(snapshot))
}

func (h *ReplayHandler) handleImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID string `json:"gameId"`
		HTML   string `json:"html"`
	}
	// Increase max body size for large HTML logs
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.GameID == "" || req.HTML == "" {
		http.Error(w, "missing gameId or html", http.StatusBadRequest)
		return
	}

	fmt.Printf("handleImport called for game %s\n", req.GameID)

	if err := h.manager.ImportLog(req.GameID, req.HTML); err != nil {
		fmt.Printf("ImportLog failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *ReplayHandler) handleImportForm(w http.ResponseWriter, r *http.Request) {
	// Increase max body size for large HTML logs
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	gameID := r.FormValue("gameId")
	htmlContent := r.FormValue("html")

	if gameID == "" || htmlContent == "" {
		http.Error(w, "missing gameId or html", http.StatusBadRequest)
		return
	}

	fmt.Printf("handleImportForm called for game %s\n", gameID)

	if err := h.manager.ImportLog(gameID, htmlContent); err != nil {
		fmt.Printf("ImportLog failed: %v\n", err)
		http.Error(w, "Import failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to the replay page on the frontend
	http.Redirect(w, r, "https://kezilu.com/tm/replay/"+gameID, http.StatusSeeOther)
}

func (h *ReplayHandler) handleImportText(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID  string `json:"gameId"`
		LogText string `json:"logText"`
		Format  string `json:"format"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024) // 10MB limit

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.GameID == "" || req.LogText == "" {
		http.Error(w, "missing gameId or logText", http.StatusBadRequest)
		return
	}

	if err := h.manager.ImportText(req.GameID, req.LogText, req.Format); err != nil {
		fmt.Printf("ImportText failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
