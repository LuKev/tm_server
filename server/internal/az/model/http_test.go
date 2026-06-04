package model

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

func TestHTTPEvaluatorUsesLegalPriors(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatal(err)
	}
	legal := position.LegalActions()
	if len(legal) < 2 {
		t.Fatal("expected multiple legal actions")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(httpEvaluateResponse{
			Priors: map[string]float64{legal[1].ID: 100},
			Value:  0.25,
		})
	}))
	defer server.Close()
	eval := NewHTTPEvaluator(server.URL, NewHeuristicEvaluator()).Evaluate(position, legal, "p1")
	if eval.Value != 0.25 {
		t.Fatalf("value = %v, want 0.25", eval.Value)
	}
	if eval.Priors[legal[1].ID] <= eval.Priors[legal[0].ID] {
		t.Fatalf("expected HTTP prior to dominate")
	}
}

func TestHTTPEvaluatorFallsBack(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatal(err)
	}
	legal := position.LegalActions()
	eval := NewHTTPEvaluator("http://127.0.0.1:1/missing", NewHeuristicEvaluator()).Evaluate(position, legal, "p1")
	if len(eval.Priors) != len(legal) {
		t.Fatalf("priors = %d, want %d", len(eval.Priors), len(legal))
	}
}

func TestHTTPEvaluatorUsesBatchEndpoint(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatal(err)
	}
	legal := position.LegalActions()
	if len(legal) < 2 {
		t.Fatal("expected multiple legal actions")
	}
	var sawBatch bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/evaluate_batch" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		sawBatch = true
		var req httpBatchEvaluateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if len(req.Requests) != 2 {
			t.Fatalf("requests = %d, want 2", len(req.Requests))
		}
		_ = json.NewEncoder(w).Encode(httpBatchEvaluateResponse{
			Responses: []httpEvaluateResponse{
				{Priors: map[string]float64{legal[0].ID: 10}, Value: 0.1},
				{Priors: map[string]float64{legal[1].ID: 10}, Value: -0.2},
			},
		})
	}))
	defer server.Close()
	evaluator := NewHTTPEvaluator(server.URL+"/evaluate", NewHeuristicEvaluator())
	evals := evaluator.EvaluateBatch([]*env.Position{position, position}, [][]actions.Option{legal, legal}, "p1")
	if !sawBatch {
		t.Fatal("expected batch endpoint")
	}
	if len(evals) != 2 || evals[0].Value != 0.1 || evals[1].Value != -0.2 {
		t.Fatalf("unexpected evals: %#v", evals)
	}
	if evals[0].Priors[legal[0].ID] <= evals[0].Priors[legal[1].ID] {
		t.Fatalf("expected first batch prior to dominate")
	}
}
