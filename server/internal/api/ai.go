package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/replay"
)

type AIHandler struct {
	games         *game.Manager
	evaluator     model.Evaluator
	modelURL      string
	requireNeural bool
}

func NewAIHandler(games *game.Manager) *AIHandler {
	modelURL := os.Getenv("TM_AZ_MODEL_URL")
	evaluator := model.LoadEvaluator(model.EvaluatorConfig{HTTPURL: modelURL})
	return &AIHandler{
		games: games, evaluator: evaluator, modelURL: modelURL,
		requireNeural: strings.EqualFold(strings.TrimSpace(os.Getenv("TM_AZ_REQUIRE_NEURAL")), "true"),
	}
}

func (h *AIHandler) RegisterRoutes(router *mux.Router) {
	s := router.PathPrefix("/api/ai").Subrouter()
	s.HandleFunc("/suggest", h.handleSuggest).Methods("POST")
	s.HandleFunc("/execute", h.handleExecute).Methods("POST")
	s.HandleFunc("/status", h.handleStatus).Methods("GET")
}

func (h *AIHandler) handleStatus(w http.ResponseWriter, _ *http.Request) {
	response := map[string]any{"mode": "heuristic", "neural": false}
	status := http.StatusServiceUnavailable
	if h.modelURL != "" {
		health, err := model.ProbeHTTP(h.modelURL, nil)
		if err != nil {
			response["mode"] = "neural_unavailable"
			response["error"] = err.Error()
		} else {
			response["mode"] = "neural"
			response["neural"] = true
			response["model"] = health
			status = http.StatusOK
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

type aiSuggestRequest struct {
	GameID       string      `json:"gameId"`
	Snapshot     string      `json:"snapshot"`
	RootPlayerID string      `json:"rootPlayerId"`
	TopN         int         `json:"topN"`
	Search       mcts.Config `json:"search"`
}

type aiSuggestResponse struct {
	GameID       string         `json:"gameId,omitempty"`
	RootPlayerID string         `json:"rootPlayerId"`
	TurnPlayerID string         `json:"turnPlayerId"`
	Round        int            `json:"round"`
	Phase        game.GamePhase `json:"phase"`
	Revision     int            `json:"revision,omitempty"`
	Result       mcts.Result    `json:"result"`
}

type aiExecuteRequest struct {
	GameID           string      `json:"gameId"`
	RootPlayerID     string      `json:"rootPlayerId"`
	ActionID         string      `json:"actionId"`
	Confirm          bool        `json:"confirm"`
	ExpectedRevision *int        `json:"expectedRevision"`
	ActionRequestID  string      `json:"actionRequestId"`
	TopN             int         `json:"topN"`
	Search           mcts.Config `json:"search"`
}

type aiExecuteResponse struct {
	GameID       string             `json:"gameId"`
	RootPlayerID string             `json:"rootPlayerId"`
	TurnPlayerID string             `json:"turnPlayerId"`
	Round        int                `json:"round"`
	Phase        game.GamePhase     `json:"phase"`
	Revision     int                `json:"revision"`
	Executed     bool               `json:"executed"`
	Selected     mcts.RankedAction  `json:"selected"`
	Result       mcts.Result        `json:"result"`
	ActionResult *game.ActionResult `json:"actionResult,omitempty"`
}

func (h *AIHandler) handleSuggest(w http.ResponseWriter, r *http.Request) {
	var req aiSuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	gs, err := h.resolveState(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	position := env.NewPosition(gs, req.RootPlayerID)
	failures := model.FailureCount(h.evaluator)
	result := mcts.Search(position, h.evaluator, req.Search)
	if h.requireNeural && model.FailureCount(h.evaluator) != failures {
		http.Error(w, "neural evaluator failed during search", http.StatusServiceUnavailable)
		return
	}
	if req.TopN > 0 && len(result.Actions) > req.TopN {
		result.Actions = result.Actions[:req.TopN]
	}
	resp := aiSuggestResponse{
		GameID:       req.GameID,
		RootPlayerID: result.RootPlayerID,
		TurnPlayerID: position.CurrentPlayerID(),
		Round:        gs.Round,
		Phase:        gs.Phase,
		Result:       result,
	}
	if req.GameID != "" {
		if revision, ok := h.games.GetRevision(req.GameID); ok {
			resp.Revision = revision
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AIHandler) handleExecute(w http.ResponseWriter, r *http.Request) {
	var req aiExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.GameID == "" {
		http.Error(w, "gameId is required", http.StatusBadRequest)
		return
	}
	gs, ok := h.games.GetGame(req.GameID)
	if !ok || gs == nil {
		http.Error(w, fmt.Sprintf("game not found: %s", req.GameID), http.StatusBadRequest)
		return
	}
	revision, _ := h.games.GetRevision(req.GameID)
	position := env.NewPosition(gs, req.RootPlayerID)
	legal := position.LegalActions()
	if len(legal) == 0 {
		http.Error(w, "no legal actions", http.StatusBadRequest)
		return
	}
	failures := model.FailureCount(h.evaluator)
	result := mcts.Search(position, h.evaluator, req.Search)
	if h.requireNeural && model.FailureCount(h.evaluator) != failures {
		http.Error(w, "neural evaluator failed during search", http.StatusServiceUnavailable)
		return
	}
	selected := result.Selected
	if req.ActionID != "" {
		ranked, ok := rankedActionByID(result.Actions, req.ActionID)
		if !ok {
			http.Error(w, fmt.Sprintf("action is not in search result: %s", req.ActionID), http.StatusBadRequest)
			return
		}
		selected = ranked
	}
	if selected.ID == "" {
		http.Error(w, "search did not select an action", http.StatusBadRequest)
		return
	}
	option, ok := optionByID(legal, selected.ID)
	if !ok {
		http.Error(w, fmt.Sprintf("selected action is no longer legal: %s", selected.ID), http.StatusBadRequest)
		return
	}
	resp := aiExecuteResponse{
		GameID:       req.GameID,
		RootPlayerID: result.RootPlayerID,
		TurnPlayerID: position.CurrentPlayerID(),
		Round:        gs.Round,
		Phase:        gs.Phase,
		Revision:     revision,
		Executed:     false,
		Selected:     selected,
		Result:       result,
	}
	if req.TopN > 0 && len(resp.Result.Actions) > req.TopN {
		resp.Result.Actions = resp.Result.Actions[:req.TopN]
	}
	if req.Confirm {
		expectedRevision := -1
		if req.ExpectedRevision != nil {
			expectedRevision = *req.ExpectedRevision
		}
		actionResult, err := h.games.ExecuteActionWithMeta(req.GameID, option.Action, game.ActionMeta{
			ActionID:         req.ActionRequestID,
			ExpectedRevision: expectedRevision,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		resp.Executed = true
		resp.ActionResult = actionResult
		if actionResult != nil {
			resp.Revision = actionResult.Revision
		}
		if updated, ok := h.games.GetGame(req.GameID); ok && updated != nil {
			resp.Round = updated.Round
			resp.Phase = updated.Phase
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AIHandler) resolveState(req aiSuggestRequest) (*game.GameState, error) {
	if req.Snapshot != "" {
		return replay.ParseSnapshot(req.Snapshot)
	}
	if req.GameID == "" {
		position, err := env.BuiltInScenario("base_nomads_witches")
		if err != nil {
			return nil, err
		}
		return position.State, nil
	}
	gs, ok := h.games.GetGame(req.GameID)
	if !ok || gs == nil {
		return nil, fmt.Errorf("game not found: %s", req.GameID)
	}
	return gs, nil
}

func rankedActionByID(actions []mcts.RankedAction, id string) (mcts.RankedAction, bool) {
	for _, action := range actions {
		if action.ID == id {
			return action, true
		}
	}
	return mcts.RankedAction{}, false
}

func optionByID(options []actions.Option, id string) (actions.Option, bool) {
	for _, option := range options {
		if option.ID == id {
			return option, true
		}
	}
	return actions.Option{}, false
}
