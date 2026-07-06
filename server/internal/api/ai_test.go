package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/game"
)

func TestAIExecutePreviewAndConfirm(t *testing.T) {
	t.Setenv("TM_AZ_MODEL_URL", "")

	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	games := game.NewManager()
	games.CreateGameWithState("g1", position.State)

	router := mux.NewRouter()
	NewAIHandler(games).RegisterRoutes(router)

	preview := postAIExecute(t, router, map[string]interface{}{
		"gameId":       "g1",
		"rootPlayerId": "p1",
		"confirm":      false,
		"search": map[string]interface{}{
			"simulations": 1,
			"maxDepth":    10,
			"randomSeed":  1,
		},
	})
	if preview.Executed {
		t.Fatal("preview should not execute")
	}
	if preview.Selected.ID == "" {
		t.Fatal("expected selected action")
	}
	if revision, ok := games.GetRevision("g1"); !ok || revision != 0 {
		t.Fatalf("preview mutated game revision: %d %v", revision, ok)
	}

	expectedRevision := 0
	confirmed := postAIExecute(t, router, map[string]interface{}{
		"gameId":           "g1",
		"rootPlayerId":     "p1",
		"actionId":         preview.Selected.ID,
		"confirm":          true,
		"expectedRevision": expectedRevision,
		"actionRequestId":  "test-action-1",
		"search": map[string]interface{}{
			"simulations": 1,
			"maxDepth":    10,
			"randomSeed":  1,
		},
	})
	if !confirmed.Executed {
		t.Fatal("confirmed request should execute")
	}
	if confirmed.ActionResult == nil || confirmed.ActionResult.Revision != 1 {
		t.Fatalf("unexpected action result: %#v", confirmed.ActionResult)
	}
	if revision, ok := games.GetRevision("g1"); !ok || revision != 1 {
		t.Fatalf("confirmed request did not advance revision: %d %v", revision, ok)
	}
}

func postAIExecute(t *testing.T, handler http.Handler, payload map[string]interface{}) aiExecuteResponse {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/ai/execute", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", resp.Code, resp.Body.String())
	}
	var out aiExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}
