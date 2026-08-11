package az

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
)

type inferenceRequest struct {
	Type     string                    `json:"type"`
	Requests []inferencePositionRecord `json:"requests"`
}

type inferencePositionRecord struct {
	State   json.RawMessage `json:"state"`
	Actions []SearchAction  `json:"actions"`
}

type inferenceResponse struct {
	Results  []EvaluationResult `json:"results,omitempty"`
	Metadata *InferenceMetadata `json:"metadata,omitempty"`
	Error    string             `json:"error,omitempty"`
}

type InferenceMetadata struct {
	ModelID          string `json:"model_id"`
	CheckpointSHA256 string `json:"checkpoint_sha256"`
	ModelConfig      string `json:"model_config"`
	Device           string `json:"device"`
	DType            string `json:"dtype"`
	Parameters       int64  `json:"parameters"`
	TorchThreads     int    `json:"torch_threads"`
}

type PythonEvaluatorOptions struct {
	Checkpoint   string
	ModelConfig  string
	Device       string
	Seed         int64
	TorchThreads int
}

// PythonEvaluator is the local correctness transport to a long-lived PyTorch
// process. It is intentionally JSON-lines; the scaled path can replace the
// transport without changing the batch-shaped Evaluator interface.
type PythonEvaluator struct {
	mu         sync.Mutex
	command    *exec.Cmd
	stdin      io.WriteCloser
	decoder    *json.Decoder
	metadata   InferenceMetadata
	options    PythonEvaluatorOptions
	stats      InferenceStats
	latencies  durationWindow
	queueWaits durationWindow
}

const inferenceDurationWindowSize = 4096

// durationWindow retains a fixed-size recent sample for operational latency
// percentiles. Aggregate durations and counts remain whole-run totals.
type durationWindow struct {
	values [inferenceDurationWindowSize]time.Duration
	count  int
	next   int
}

func (w *durationWindow) observe(value time.Duration) {
	w.values[w.next] = value
	w.next = (w.next + 1) % len(w.values)
	if w.count < len(w.values) {
		w.count++
	}
}

func (w *durationWindow) samples() []time.Duration {
	return append([]time.Duration(nil), w.values[:w.count]...)
}

// InferenceStats measures the complete Go-to-Python evaluation path. TotalDuration
// starts before the evaluator mutex, so it includes queueing as well as request
// serialization, tensor materialization, model execution, and the response.
type InferenceStats struct {
	Batches           int64         `json:"batches"`
	Positions         int64         `json:"positions"`
	LegalActions      int64         `json:"legal_actions"`
	MaxBatchSize      int           `json:"max_batch_size"`
	MaxLegalActions   int           `json:"max_legal_actions"`
	TotalDuration     time.Duration `json:"-"`
	QueueWaitDuration time.Duration `json:"-"`
	ServiceDuration   time.Duration `json:"-"`
	LatencyP50        time.Duration `json:"-"`
	LatencyP95        time.Duration `json:"-"`
	QueueWaitP95      time.Duration `json:"-"`
	LatencySamples    int           `json:"latency_samples"`
	LatencySampleCap  int           `json:"latency_sample_capacity"`
}

func StartPythonEvaluator(ctx context.Context, executable, checkpoint, modelConfig string, seed int64) (*PythonEvaluator, error) {
	return StartPythonEvaluatorWithOptions(ctx, executable, PythonEvaluatorOptions{
		Checkpoint: checkpoint, ModelConfig: modelConfig, Device: "cpu", Seed: seed, TorchThreads: 1,
	})
}

func StartPythonEvaluatorWithOptions(ctx context.Context, executable string, options PythonEvaluatorOptions) (*PythonEvaluator, error) {
	if executable == "" {
		return nil, fmt.Errorf("inference executable is required")
	}
	if options.ModelConfig == "" {
		options.ModelConfig = "auto"
	}
	if options.Device == "" {
		options.Device = "cpu"
	}
	if options.TorchThreads < 0 {
		return nil, fmt.Errorf("torch threads must be non-negative")
	}
	args := []string{
		"--model-config", options.ModelConfig,
		"--seed", fmt.Sprint(options.Seed),
		"--device", options.Device,
		"--torch-threads", fmt.Sprint(options.TorchThreads),
	}
	if options.Checkpoint != "" {
		args = append(args, "--checkpoint", options.Checkpoint)
	}
	command := exec.CommandContext(ctx, executable, args...)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	evaluator := &PythonEvaluator{
		command: command, stdin: stdin, decoder: json.NewDecoder(bufio.NewReader(stdout)), options: options,
	}
	if err := evaluator.readMetadata(); err != nil {
		_ = evaluator.Close()
		return nil, err
	}
	return evaluator, nil
}

func (e *PythonEvaluator) readMetadata() error {
	if err := json.NewEncoder(e.stdin).Encode(inferenceRequest{Type: "metadata"}); err != nil {
		return fmt.Errorf("write inference metadata request: %w", err)
	}
	var response inferenceResponse
	if err := e.decoder.Decode(&response); err != nil {
		return fmt.Errorf("read inference metadata: %w", err)
	}
	if response.Error != "" {
		return fmt.Errorf("inference service: %s", response.Error)
	}
	if response.Metadata == nil {
		return fmt.Errorf("inference service did not provide metadata")
	}
	if err := validateInferenceMetadata(*response.Metadata, e.options); err != nil {
		return err
	}
	e.metadata = *response.Metadata
	return nil
}

func validateInferenceMetadata(metadata InferenceMetadata, expected PythonEvaluatorOptions) error {
	if metadata.ModelID == "" {
		return fmt.Errorf("inference service did not provide model identity")
	}
	if metadata.ModelConfig == "" || metadata.Device == "" || metadata.DType != "float32" || metadata.Parameters <= 0 || metadata.TorchThreads <= 0 {
		return fmt.Errorf("inference service provided incomplete model metadata")
	}
	if expected.ModelConfig != "" && expected.ModelConfig != "auto" && metadata.ModelConfig != expected.ModelConfig {
		return fmt.Errorf("inference model config %q does not match requested %q", metadata.ModelConfig, expected.ModelConfig)
	}
	if expected.Device != "" && expected.Device != "auto" && metadata.Device != expected.Device {
		return fmt.Errorf("inference device %q does not match requested %q", metadata.Device, expected.Device)
	}
	if expected.TorchThreads > 0 && metadata.TorchThreads != expected.TorchThreads {
		return fmt.Errorf("inference torch threads %d do not match requested %d", metadata.TorchThreads, expected.TorchThreads)
	}
	return nil
}

func (e *PythonEvaluator) ModelID() string { return e.metadata.ModelID }

func (e *PythonEvaluator) CheckpointSHA256() string { return e.metadata.CheckpointSHA256 }

func (e *PythonEvaluator) Metadata() InferenceMetadata { return e.metadata }

func (e *PythonEvaluator) Stats() InferenceStats {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := e.stats
	out.LatencyP50 = durationPercentile(e.latencies.samples(), 0.50)
	out.LatencyP95 = durationPercentile(e.latencies.samples(), 0.95)
	out.QueueWaitP95 = durationPercentile(e.queueWaits.samples(), 0.95)
	out.LatencySamples = e.latencies.count
	out.LatencySampleCap = inferenceDurationWindowSize
	return out
}

func (e *PythonEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	requested := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	queueWait := time.Since(requested)
	serviceStarted := time.Now()
	records := make([]inferencePositionRecord, len(requests))
	legalActions := 0
	maxLegalActions := 0
	for i, request := range requests {
		position, ok := request.Position.(*GamePosition)
		if !ok {
			return nil, fmt.Errorf("Python evaluator requires *GamePosition, got %T", request.Position)
		}
		// MCTS hashes a leaf before evaluation. Reuse that immutable canonical
		// record instead of serializing the same position a second time.
		state, err := position.canonicalBytes()
		if err != nil {
			return nil, err
		}
		records[i] = inferencePositionRecord{State: state, Actions: request.Actions}
		legalActions += len(request.Actions)
		if len(request.Actions) > maxLegalActions {
			maxLegalActions = len(request.Actions)
		}
	}
	if err := json.NewEncoder(e.stdin).Encode(inferenceRequest{Type: "evaluate", Requests: records}); err != nil {
		return nil, fmt.Errorf("write inference request: %w", err)
	}
	var response inferenceResponse
	if err := e.decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("read inference response: %w", err)
	}
	if response.Error != "" {
		return nil, fmt.Errorf("inference service: %s", response.Error)
	}
	e.stats.Batches++
	e.stats.Positions += int64(len(requests))
	e.stats.LegalActions += int64(legalActions)
	if len(requests) > e.stats.MaxBatchSize {
		e.stats.MaxBatchSize = len(requests)
	}
	if maxLegalActions > e.stats.MaxLegalActions {
		e.stats.MaxLegalActions = maxLegalActions
	}
	serviceDuration := time.Since(serviceStarted)
	totalDuration := time.Since(requested)
	e.stats.QueueWaitDuration += queueWait
	e.stats.ServiceDuration += serviceDuration
	e.stats.TotalDuration += totalDuration
	e.latencies.observe(totalDuration)
	e.queueWaits.observe(queueWait)
	return response.Results, nil
}

func durationPercentile(values []time.Duration, quantile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(math.Ceil(quantile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func (e *PythonEvaluator) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.command == nil {
		return nil
	}
	closeErr := e.stdin.Close()
	waitErr := e.command.Wait()
	e.command = nil
	if closeErr != nil {
		return closeErr
	}
	return waitErr
}
