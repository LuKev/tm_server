package model

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

type HTTPEvaluator struct {
	URL            string
	BatchURL       string
	BinaryURL      string
	BinaryBatchURL string
	Client         *http.Client
	Fallback       Evaluator
	binaryDisabled atomic.Bool
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

type httpBinaryBatchEvaluateRequest struct {
	PerspectivePlayerID string     `json:"perspectivePlayerId"`
	InputSize           int        `json:"inputSize"`
	Count               int        `json:"count"`
	Features            string     `json:"features"`
	LegalActions        [][]string `json:"legalActions"`
}

func NewHTTPEvaluator(url string, fallback Evaluator) *HTTPEvaluator {
	if fallback == nil {
		fallback = NewHeuristicEvaluator()
	}
	return &HTTPEvaluator{
		URL:            url,
		BatchURL:       batchURLFor(url),
		BinaryURL:      binaryURLFor(url),
		BinaryBatchURL: binaryBatchURLFor(url),
		Fallback:       fallback,
		Client:         &http.Client{Timeout: 3 * time.Second},
	}
}

func (e *HTTPEvaluator) Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
	if e == nil || e.URL == "" {
		return fallbackEval(e, position, legal, perspectivePlayerID)
	}
	if !e.binaryDisabled.Load() && e.BinaryBatchURL != "" {
		if evals, ok := e.postBinaryBatch(e.BinaryBatchURL, []*env.Position{position}, [][]actions.Option{legal}, perspectivePlayerID); ok && len(evals) == 1 {
			return evals[0]
		}
	}
	return e.evaluateJSON(position, legal, perspectivePlayerID)
}

func (e *HTTPEvaluator) evaluateJSON(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
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
	if !e.binaryDisabled.Load() && e.BinaryBatchURL != "" {
		if evals, ok := e.postBinaryBatch(e.BinaryBatchURL, positions, legal, perspectivePlayerID); ok {
			return evals
		}
	}
	return e.evaluateBatchJSON(positions, legal, perspectivePlayerID)
}

func (e *HTTPEvaluator) evaluateBatchJSON(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []Evaluation {
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

func (e *HTTPEvaluator) postBinaryBatch(url string, positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) ([]Evaluation, bool) {
	req := binaryBatchRequest(positions, legal, perspectivePlayerID)
	if req.Count == 0 || req.InputSize == 0 {
		return fallbackBatchEval(e, positions, legal, perspectivePlayerID), true
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, false
	}
	client := e.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		e.binaryDisabled.Store(true)
		return nil, false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	var out httpBatchEvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, false
	}
	if len(out.Responses) != len(positions) {
		return nil, false
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
	return evals, true
}

func binaryBatchRequest(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) httpBinaryBatchEvaluateRequest {
	encodings := make([][]float64, 0, len(positions))
	inputSize := 0
	for _, position := range positions {
		encoding := []float64(nil)
		if position != nil {
			encoding = position.Encode()
		}
		if len(encoding) > inputSize {
			inputSize = len(encoding)
		}
		encodings = append(encodings, encoding)
	}
	features := make([]byte, 4*inputSize*len(encodings))
	offset := 0
	for _, encoding := range encodings {
		for i := 0; i < inputSize; i++ {
			value := float32(0)
			if i < len(encoding) {
				value = float32(encoding[i])
			}
			binary.LittleEndian.PutUint32(features[offset:offset+4], math.Float32bits(value))
			offset += 4
		}
	}
	legalIDs := make([][]string, 0, len(positions))
	for i := range positions {
		if i < len(legal) {
			legalIDs = append(legalIDs, actionIDs(legal[i]))
		} else {
			legalIDs = append(legalIDs, nil)
		}
	}
	return httpBinaryBatchEvaluateRequest{
		PerspectivePlayerID: perspectivePlayerID,
		InputSize:           inputSize,
		Count:               len(positions),
		Features:            base64.StdEncoding.EncodeToString(features),
		LegalActions:        legalIDs,
	}
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

func binaryURLFor(url string) string {
	if url == "" {
		return ""
	}
	if strings.HasSuffix(url, "/evaluate") {
		return strings.TrimSuffix(url, "/evaluate") + "/evaluate_binary"
	}
	return strings.TrimRight(url, "/") + "/evaluate_binary"
}

func binaryBatchURLFor(url string) string {
	if url == "" {
		return ""
	}
	if strings.HasSuffix(url, "/evaluate") {
		return strings.TrimSuffix(url, "/evaluate") + "/evaluate_batch_binary"
	}
	return strings.TrimRight(url, "/") + "/evaluate_batch_binary"
}
