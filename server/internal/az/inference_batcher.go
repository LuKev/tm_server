package az

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchingEvaluatorOptions controls cross-search inference batching. A batch is
// dispatched when it reaches MaxPositions or when MaxWait has elapsed since
// the first queued request. Individual Evaluate calls are never split.
type BatchingEvaluatorOptions struct {
	MaxPositions int
	MaxWait      time.Duration
}

// BatchingEvaluatorStats separates time waiting for a cross-search batch from
// the underlying evaluator's serialization/model-service measurements.
type BatchingEvaluatorStats struct {
	Calls              int64         `json:"calls"`
	Positions          int64         `json:"positions"`
	Dispatches         int64         `json:"dispatches"`
	SizeFlushes        int64         `json:"size_flushes"`
	DeadlineFlushes    int64         `json:"deadline_flushes"`
	MaxBatchSize       int           `json:"max_batch_size"`
	QueueWaitDuration  time.Duration `json:"-"`
	QueueWaitP95       time.Duration `json:"-"`
	QueueWaitSamples   int           `json:"queue_wait_samples"`
	QueueWaitSampleCap int           `json:"queue_wait_sample_capacity"`
}

type batchingEvaluationResponse struct {
	results []EvaluationResult
	err     error
}

type batchingEvaluationCall struct {
	ctx      context.Context
	requests []EvaluationRequest
	queuedAt time.Time
	response chan batchingEvaluationResponse
}

// BatchingEvaluator coalesces concurrent Evaluator calls onto one underlying
// batch-shaped evaluator. It owns only the broker goroutine; Close does not
// close the underlying evaluator. Close waits for an in-flight underlying
// Evaluate call to return; the underlying evaluator owns interruption of that
// service call.
type BatchingEvaluator struct {
	underlying Evaluator
	options    BatchingEvaluatorOptions
	ctx        context.Context
	cancel     context.CancelFunc
	calls      chan *batchingEvaluationCall
	done       chan struct{}
	closeOnce  sync.Once

	statsMu    sync.Mutex
	stats      BatchingEvaluatorStats
	queueWaits durationWindow
}

func NewBatchingEvaluator(ctx context.Context, underlying Evaluator, options BatchingEvaluatorOptions) (*BatchingEvaluator, error) {
	if underlying == nil {
		return nil, fmt.Errorf("batching evaluator underlying evaluator is nil")
	}
	if options.MaxPositions <= 0 {
		return nil, fmt.Errorf("batching evaluator max positions must be positive")
	}
	if options.MaxWait < 0 {
		return nil, fmt.Errorf("batching evaluator max wait must be non-negative")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	brokerCtx, cancel := context.WithCancel(ctx)
	broker := &BatchingEvaluator{
		underlying: underlying,
		options:    options,
		ctx:        brokerCtx,
		cancel:     cancel,
		calls:      make(chan *batchingEvaluationCall, options.MaxPositions*2),
		done:       make(chan struct{}),
	}
	go broker.run()
	return broker, nil
}

func (b *BatchingEvaluator) Evaluate(ctx context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	if len(requests) > b.options.MaxPositions {
		return nil, fmt.Errorf("evaluation call has %d positions, exceeding configured maximum %d", len(requests), b.options.MaxPositions)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	call := &batchingEvaluationCall{
		ctx:      ctx,
		requests: append([]EvaluationRequest(nil), requests...),
		queuedAt: time.Now(),
		response: make(chan batchingEvaluationResponse, 1),
	}
	select {
	case b.calls <- call:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.done:
		return nil, fmt.Errorf("batching evaluator is closed")
	}
	select {
	case response := <-call.response:
		return response.results, response.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.done:
		select {
		case response := <-call.response:
			return response.results, response.err
		default:
			return nil, fmt.Errorf("batching evaluator is closed")
		}
	}
}

func (b *BatchingEvaluator) ModelID() string {
	if evaluator, ok := b.underlying.(identifiedEvaluator); ok {
		return evaluator.ModelID()
	}
	return ""
}

func (b *BatchingEvaluator) CheckpointSHA256() string {
	if evaluator, ok := b.underlying.(checkpointEvaluator); ok {
		return evaluator.CheckpointSHA256()
	}
	return ""
}

func (b *BatchingEvaluator) Stats() BatchingEvaluatorStats {
	b.statsMu.Lock()
	defer b.statsMu.Unlock()
	out := b.stats
	out.QueueWaitP95 = durationPercentile(b.queueWaits.samples(), 0.95)
	out.QueueWaitSamples = b.queueWaits.count
	out.QueueWaitSampleCap = inferenceDurationWindowSize
	return out
}

func (b *BatchingEvaluator) Close() {
	b.closeOnce.Do(b.cancel)
	<-b.done
}

func (b *BatchingEvaluator) run() {
	defer close(b.done)
	var pending []*batchingEvaluationCall
	positionCount := 0
	var timer *time.Timer
	var timerC <-chan time.Time
	stopTimer := func() {
		if timer == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer = nil
		timerC = nil
	}
	startTimer := func(queuedAt time.Time) bool {
		remaining := b.options.MaxWait - time.Since(queuedAt)
		if remaining <= 0 {
			return false
		}
		timer = time.NewTimer(remaining)
		timerC = timer.C
		return true
	}
	dispatch := func(reason string) {
		stopTimer()
		if len(pending) == 0 {
			return
		}
		if err := b.ctx.Err(); err != nil {
			for _, call := range pending {
				call.response <- batchingEvaluationResponse{err: err}
			}
			pending = nil
			positionCount = 0
			return
		}
		b.dispatch(pending, reason)
		pending = nil
		positionCount = 0
	}
	failPending := func(err error) {
		stopTimer()
		for _, call := range pending {
			call.response <- batchingEvaluationResponse{err: err}
		}
	}

	for {
		select {
		case <-b.ctx.Done():
			failPending(b.ctx.Err())
			for {
				select {
				case call := <-b.calls:
					call.response <- batchingEvaluationResponse{err: b.ctx.Err()}
				default:
					return
				}
			}
		case <-timerC:
			dispatch("deadline")
		case first := <-b.calls:
			// An inference call may have occupied the broker while more roots
			// queued work. Drain that ready backlog before applying the oldest
			// deadline; otherwise every expired request would flush alone.
			stopTimer()
			call := first
			for call != nil {
				if err := call.ctx.Err(); err != nil {
					call.response <- batchingEvaluationResponse{err: err}
				} else {
					callPositions := len(call.requests)
					if len(pending) > 0 && positionCount+callPositions > b.options.MaxPositions {
						dispatch("size")
					}
					pending = append(pending, call)
					positionCount += callPositions
					if positionCount >= b.options.MaxPositions {
						dispatch("size")
					}
				}
				select {
				case call = <-b.calls:
				default:
					call = nil
				}
			}
			if len(pending) > 0 && !startTimer(pending[0].queuedAt) {
				dispatch("deadline")
			}
		}
	}
}

func (b *BatchingEvaluator) dispatch(calls []*batchingEvaluationCall, reason string) {
	now := time.Now()
	active := calls[:0]
	positionCount := 0
	for _, call := range calls {
		if err := call.ctx.Err(); err != nil {
			call.response <- batchingEvaluationResponse{err: err}
			continue
		}
		active = append(active, call)
		positionCount += len(call.requests)
	}
	if len(active) == 0 {
		return
	}
	requests := make([]EvaluationRequest, 0, positionCount)
	for _, call := range active {
		requests = append(requests, call.requests...)
	}
	results, err := b.underlying.Evaluate(b.ctx, requests)
	if err == nil && len(results) != len(requests) {
		err = fmt.Errorf("batched evaluator returned %d results for %d requests", len(results), len(requests))
	}

	b.statsMu.Lock()
	b.stats.Calls += int64(len(active))
	b.stats.Positions += int64(positionCount)
	b.stats.Dispatches++
	if reason == "size" {
		b.stats.SizeFlushes++
	} else {
		b.stats.DeadlineFlushes++
	}
	if positionCount > b.stats.MaxBatchSize {
		b.stats.MaxBatchSize = positionCount
	}
	for _, call := range active {
		wait := now.Sub(call.queuedAt)
		b.stats.QueueWaitDuration += wait
		b.queueWaits.observe(wait)
	}
	b.statsMu.Unlock()

	offset := 0
	for _, call := range active {
		count := len(call.requests)
		response := batchingEvaluationResponse{err: err}
		if err == nil {
			response.results = append([]EvaluationResult(nil), results[offset:offset+count]...)
		}
		call.response <- response
		offset += count
	}
}
