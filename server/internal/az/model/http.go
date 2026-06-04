package model

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

type HTTPEvaluator struct {
	URL      string
	BatchURL string
	Client   *http.Client
	Fallback Evaluator
}

type httpEvaluateRequest struct {
	Encoding            []float64 `json:"encoding"`
	LegalActions        []string  `json:"legalActions"`
	PerspectivePlayerID string    `json:"perspectivePlayerId"`
}

type httpEvaluateResponse struct {
	Priors map[string]float64 `json:"priors"`
	Value  float64            `json:"value"`
}

type httpBatchEvaluateRequest struct {
	Requests []httpEvaluateRequest `json:"requests"`
}

type httpBatchEvaluateResponse struct {
	Responses []httpEvaluateResponse `json:"responses"`
}

func NewHTTPEvaluator(url string, fallback Evaluator) *HTTPEvaluator {
	if fallback == nil {
		fallback = NewHeuristicEvaluator()
	}
	return &HTTPEvaluator{
		URL:      url,
		BatchURL: batchURLFor(url),
		Fallback: fallback,
		Client:   &http.Client{Timeout: 3 * time.Second},
	}
}

func (e *HTTPEvaluator) Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
	if e == nil || e.URL == "" {
		return fallbackEval(e, position, legal, perspectivePlayerID)
	}
	req := httpEvaluateRequest{
		PerspectivePlayerID: perspectivePlayerID,
	}
	if position != nil {
		req.Encoding = position.Encode()
	}
	req.LegalActions = make([]string, 0, len(legal))
	for _, option := range legal {
		req.LegalActions = append(req.LegalActions, option.ID)
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return fallbackEval(e, position, legal, perspectivePlayerID)
	}
	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	resp, err := client.Post(e.URL, "application/json", bytes.NewReader(raw))
	if err != nil {
		return fallbackEval(e, position, legal, perspectivePlayerID)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallbackEval(e, position, legal, perspectivePlayerID)
	}
	var out httpEvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fallbackEval(e, position, legal, perspectivePlayerID)
	}
	priors := normalizeLegalPriors(legal, out.Priors)
	value := math.Max(-1, math.Min(1, out.Value))
	return Evaluation{Priors: priors, Value: value}
}

func (e *HTTPEvaluator) EvaluateBatch(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []Evaluation {
	if e == nil || e.BatchURL == "" || len(positions) == 0 {
		return fallbackBatchEval(e, positions, legal, perspectivePlayerID)
	}
	requests := make([]httpEvaluateRequest, 0, len(positions))
	for i, position := range positions {
		req := httpEvaluateRequest{PerspectivePlayerID: perspectivePlayerID}
		if position != nil {
			req.Encoding = position.Encode()
		}
		if i < len(legal) {
			req.LegalActions = actionIDs(legal[i])
		}
		requests = append(requests, req)
	}
	raw, err := json.Marshal(httpBatchEvaluateRequest{Requests: requests})
	if err != nil {
		return fallbackBatchEval(e, positions, legal, perspectivePlayerID)
	}
	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	resp, err := client.Post(e.BatchURL, "application/json", bytes.NewReader(raw))
	if err != nil {
		return fallbackBatchEval(e, positions, legal, perspectivePlayerID)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallbackBatchEval(e, positions, legal, perspectivePlayerID)
	}
	var out httpBatchEvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fallbackBatchEval(e, positions, legal, perspectivePlayerID)
	}
	if len(out.Responses) != len(positions) {
		return fallbackBatchEval(e, positions, legal, perspectivePlayerID)
	}
	evals := make([]Evaluation, 0, len(out.Responses))
	for i, response := range out.Responses {
		options := []actions.Option(nil)
		if i < len(legal) {
			options = legal[i]
		}
		evals = append(evals, Evaluation{
			Priors: normalizeLegalPriors(options, response.Priors),
			Value:  math.Max(-1, math.Min(1, response.Value)),
		})
	}
	return evals
}

func fallbackEval(e *HTTPEvaluator, position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
	if e != nil && e.Fallback != nil {
		return e.Fallback.Evaluate(position, legal, perspectivePlayerID)
	}
	return NewHeuristicEvaluator().Evaluate(position, legal, perspectivePlayerID)
}

func fallbackBatchEval(e *HTTPEvaluator, positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []Evaluation {
	if e != nil {
		if batch, ok := e.Fallback.(BatchEvaluator); ok {
			return batch.EvaluateBatch(positions, legal, perspectivePlayerID)
		}
	}
	out := make([]Evaluation, 0, len(positions))
	for i, position := range positions {
		options := []actions.Option(nil)
		if i < len(legal) {
			options = legal[i]
		}
		out = append(out, fallbackEval(e, position, options, perspectivePlayerID))
	}
	return out
}

func normalizeLegalPriors(legal []actions.Option, raw map[string]float64) map[string]float64 {
	priors := make(map[string]float64, len(legal))
	total := 0.0
	for _, option := range legal {
		weight := raw[option.ID]
		if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			weight = 0
		}
		weight += 1e-6
		priors[option.ID] = weight
		total += weight
	}
	if total <= 0 {
		uniform := 0.0
		if len(legal) > 0 {
			uniform = 1.0 / float64(len(legal))
		}
		for _, option := range legal {
			priors[option.ID] = uniform
		}
		return priors
	}
	for id, weight := range priors {
		priors[id] = weight / total
	}
	return priors
}

func actionIDs(options []actions.Option) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.ID)
	}
	return out
}

func batchURLFor(url string) string {
	if url == "" {
		return ""
	}
	if strings.HasSuffix(url, "/evaluate") {
		return strings.TrimSuffix(url, "/evaluate") + "/evaluate_batch"
	}
	return strings.TrimRight(url, "/") + "/evaluate_batch"
}
