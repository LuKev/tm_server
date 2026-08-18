package az

import (
	"context"
	"sync"
	"testing"
	"time"
)

type recordingEvaluator struct {
	mu         sync.Mutex
	batchSizes []int
}

type blockingFirstEvaluator struct {
	mu            sync.Mutex
	batchSizes    []int
	firstStarted  chan struct{}
	secondStarted chan struct{}
	releaseFirst  chan struct{}
}

func newBlockingFirstEvaluator() *blockingFirstEvaluator {
	return &blockingFirstEvaluator{
		firstStarted: make(chan struct{}), secondStarted: make(chan struct{}), releaseFirst: make(chan struct{}),
	}
}

func (e *blockingFirstEvaluator) Evaluate(ctx context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	e.mu.Lock()
	call := len(e.batchSizes)
	e.batchSizes = append(e.batchSizes, len(requests))
	e.mu.Unlock()
	if call == 0 {
		close(e.firstStarted)
		<-e.releaseFirst
	} else if call == 1 {
		close(e.secondStarted)
	}
	return UniformEvaluator{}.Evaluate(ctx, requests)
}

func (e *blockingFirstEvaluator) batches() []int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]int(nil), e.batchSizes...)
}

func (e *recordingEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	e.mu.Lock()
	e.batchSizes = append(e.batchSizes, len(requests))
	e.mu.Unlock()
	results := make([]EvaluationResult, len(requests))
	for index, request := range requests {
		results[index].PolicyLogits = make([]float32, len(request.Actions))
		if len(request.Actions) > 0 {
			results[index].Value = float32(request.Actions[0].Amount) / 100
		}
	}
	return results, nil
}

func (e *recordingEvaluator) batches() []int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]int(nil), e.batchSizes...)
}

func TestInferenceMetadataMatchesExplicitRequest(t *testing.T) {
	base := InferenceMetadata{
		ModelID: "random-test", ModelConfig: "large", Device: "mps",
		DType: "float32", Parameters: 29_114_114, TorchThreads: 4,
	}
	tests := []struct {
		name    string
		options PythonEvaluatorOptions
		wantErr bool
	}{
		{name: "exact", options: PythonEvaluatorOptions{ModelConfig: "large", Device: "mps", TorchThreads: 4}},
		{name: "auto_resolves", options: PythonEvaluatorOptions{ModelConfig: "auto", Device: "auto", TorchThreads: 0}},
		{name: "wrong_config", options: PythonEvaluatorOptions{ModelConfig: "debug", Device: "mps", TorchThreads: 4}, wantErr: true},
		{name: "wrong_device", options: PythonEvaluatorOptions{ModelConfig: "large", Device: "cpu", TorchThreads: 4}, wantErr: true},
		{name: "wrong_threads", options: PythonEvaluatorOptions{ModelConfig: "large", Device: "mps", TorchThreads: 1}, wantErr: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateInferenceMetadata(base, testCase.options)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("validation error = %v, wantErr=%v", err, testCase.wantErr)
			}
		})
	}
}

func TestInferenceDurationWindowIsBoundedAndRecent(t *testing.T) {
	var window durationWindow
	for index := 0; index < inferenceDurationWindowSize+904; index++ {
		window.observe(time.Duration(index) * time.Millisecond)
	}
	samples := window.samples()
	if len(samples) != inferenceDurationWindowSize {
		t.Fatalf("sample count = %d, want %d", len(samples), inferenceDurationWindowSize)
	}
	if got := durationPercentile(samples, 0); got != 904*time.Millisecond {
		t.Fatalf("oldest retained latency = %v, want 904ms", got)
	}
	if got := durationPercentile(samples, 1); got != 4999*time.Millisecond {
		t.Fatalf("newest retained latency = %v, want 4999ms", got)
	}
}

func TestBatchingEvaluatorFlushesConcurrentCallsAtSizeAndPreservesResults(t *testing.T) {
	underlying := &recordingEvaluator{}
	batcher, err := NewBatchingEvaluator(context.Background(), underlying, BatchingEvaluatorOptions{
		MaxPositions: 4,
		MaxWait:      100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer batcher.Close()

	start := make(chan struct{})
	results := make([]float32, 4)
	errors := make([]error, 4)
	var wait sync.WaitGroup
	wait.Add(4)
	for index := range results {
		go func(index int) {
			defer wait.Done()
			<-start
			response, evaluateErr := batcher.Evaluate(context.Background(), []EvaluationRequest{{
				Actions: []SearchAction{{Amount: index + 1}},
			}})
			errors[index] = evaluateErr
			if evaluateErr == nil {
				results[index] = response[0].Value
			}
		}(index)
	}
	close(start)
	wait.Wait()
	for index, evaluateErr := range errors {
		if evaluateErr != nil {
			t.Fatalf("call %d: %v", index, evaluateErr)
		}
		if want := float32(index+1) / 100; results[index] != want {
			t.Fatalf("call %d value = %v, want %v", index, results[index], want)
		}
	}
	if got := underlying.batches(); len(got) != 1 || got[0] != 4 {
		t.Fatalf("underlying batches = %v, want [4]", got)
	}
	stats := batcher.Stats()
	if stats.Calls != 4 || stats.Positions != 4 || stats.Dispatches != 1 || stats.SizeFlushes != 1 || stats.DeadlineFlushes != 0 || stats.MaxBatchSize != 4 {
		t.Fatalf("unexpected batcher stats: %+v", stats)
	}
}

func TestBatchingEvaluatorDeadlineFlushesPartialBatch(t *testing.T) {
	underlying := &recordingEvaluator{}
	batcher, err := NewBatchingEvaluator(context.Background(), underlying, BatchingEvaluatorOptions{
		MaxPositions: 8,
		MaxWait:      2 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer batcher.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := batcher.Evaluate(ctx, []EvaluationRequest{{Actions: []SearchAction{{Amount: 1}}}}); err != nil {
		t.Fatal(err)
	}
	if got := underlying.batches(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("underlying batches = %v, want [1]", got)
	}
	stats := batcher.Stats()
	if stats.DeadlineFlushes != 1 || stats.SizeFlushes != 0 || stats.QueueWaitP95 <= 0 {
		t.Fatalf("unexpected deadline stats: %+v", stats)
	}
}

func TestBatchingEvaluatorRejectsInvalidConfiguration(t *testing.T) {
	for _, options := range []BatchingEvaluatorOptions{
		{MaxPositions: 0},
		{MaxPositions: 1, MaxWait: -time.Nanosecond},
	} {
		if _, err := NewBatchingEvaluator(context.Background(), UniformEvaluator{}, options); err == nil {
			t.Fatalf("accepted invalid options %+v", options)
		}
	}
}

func TestBatchingEvaluatorRejectsCallLargerThanMaximum(t *testing.T) {
	batcher, err := NewBatchingEvaluator(context.Background(), UniformEvaluator{}, BatchingEvaluatorOptions{MaxPositions: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer batcher.Close()
	if _, err := batcher.Evaluate(context.Background(), make([]EvaluationRequest, 3)); err == nil {
		t.Fatal("accepted an Evaluate call larger than MaxPositions")
	}
}

func TestBatchingEvaluatorDeadlineStartsWhenRequestIsQueued(t *testing.T) {
	underlying := newBlockingFirstEvaluator()
	batcher, err := NewBatchingEvaluator(context.Background(), underlying, BatchingEvaluatorOptions{
		MaxPositions: 8,
		MaxWait:      100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer batcher.Close()
	firstDone := make(chan error, 1)
	go func() {
		_, evaluateErr := batcher.Evaluate(context.Background(), make([]EvaluationRequest, 8))
		firstDone <- evaluateErr
	}()
	<-underlying.firstStarted
	secondDone := make(chan error, 3)
	for range 3 {
		go func() {
			_, evaluateErr := batcher.Evaluate(context.Background(), []EvaluationRequest{{}})
			secondDone <- evaluateErr
		}()
	}
	queueDeadline := time.Now().Add(time.Second)
	for len(batcher.calls) < 3 && time.Now().Before(queueDeadline) {
		time.Sleep(time.Millisecond)
	}
	if len(batcher.calls) != 3 {
		t.Fatalf("queued calls = %d, want 3", len(batcher.calls))
	}
	time.Sleep(150 * time.Millisecond)
	releasedAt := time.Now()
	close(underlying.releaseFirst)
	select {
	case <-underlying.secondStarted:
		if elapsed := time.Since(releasedAt); elapsed >= 75*time.Millisecond {
			t.Fatalf("expired queued request received a fresh deadline: %v", elapsed)
		}
	case <-time.After(time.Second):
		t.Fatal("second request was not dispatched")
	}
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := <-secondDone; err != nil {
			t.Fatal(err)
		}
	}
	if got := underlying.batches(); len(got) != 2 || got[0] != 8 || got[1] != 3 {
		t.Fatalf("ready backlog was not batched after service: %v", got)
	}
}

func TestBatchingEvaluatorCancellationAndClose(t *testing.T) {
	underlying := newBlockingFirstEvaluator()
	batcher, err := NewBatchingEvaluator(context.Background(), underlying, BatchingEvaluatorOptions{
		MaxPositions: 8,
		MaxWait:      time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	firstDone := make(chan error, 1)
	go func() {
		_, evaluateErr := batcher.Evaluate(context.Background(), make([]EvaluationRequest, 8))
		firstDone <- evaluateErr
	}()
	<-underlying.firstStarted

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancelledDone := make(chan error, 1)
	go func() {
		_, evaluateErr := batcher.Evaluate(cancelledCtx, []EvaluationRequest{{}})
		cancelledDone <- evaluateErr
	}()
	cancel()
	if err := <-cancelledDone; err == nil {
		t.Fatal("cancelled queued call returned no error")
	}

	pendingDone := make(chan error, 1)
	go func() {
		_, evaluateErr := batcher.Evaluate(context.Background(), []EvaluationRequest{{}})
		pendingDone <- evaluateErr
	}()
	closed := make(chan struct{})
	go func() {
		batcher.Close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned while the underlying evaluation was in flight")
	case <-time.After(10 * time.Millisecond):
	}
	close(underlying.releaseFirst)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after the underlying evaluation completed")
	}
	if err := <-firstDone; err != nil {
		t.Fatalf("in-flight call failed after service completion: %v", err)
	}
	if err := <-pendingDone; err == nil {
		t.Fatal("queued call survived Close")
	}
	if _, err := batcher.Evaluate(context.Background(), []EvaluationRequest{{}}); err == nil {
		t.Fatal("Evaluate succeeded after Close")
	}
	if got := underlying.batches(); len(got) != 1 || got[0] != 8 {
		t.Fatalf("cancelled or pending calls reached underlying evaluator: %v", got)
	}
}
