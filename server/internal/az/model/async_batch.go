package model

import (
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

type asyncBatchCall struct {
	positions   []*env.Position
	legal       [][]actions.Option
	perspective string
	response    chan []Evaluation
}

// AsyncBatchEvaluator merges concurrent batch-evaluation calls before passing
// them to the wrapped evaluator. It is most useful when several self-play
// workers share one HTTP model server.
type AsyncBatchEvaluator struct {
	base       BatchEvaluator
	maxBatch   int
	flushDelay time.Duration
	queue      chan asyncBatchCall
}

func NewAsyncBatchEvaluator(base Evaluator, maxBatch int, flushDelay time.Duration) Evaluator {
	batch, ok := base.(BatchEvaluator)
	if !ok || maxBatch <= 1 {
		return base
	}
	if flushDelay <= 0 {
		flushDelay = time.Millisecond
	}
	evaluator := &AsyncBatchEvaluator{
		base:       batch,
		maxBatch:   maxBatch,
		flushDelay: flushDelay,
		queue:      make(chan asyncBatchCall, 1024),
	}
	go evaluator.run()
	return evaluator
}

func (e *AsyncBatchEvaluator) Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
	if e == nil {
		return NewHeuristicEvaluator().Evaluate(position, legal, perspectivePlayerID)
	}
	evals := e.EvaluateBatch([]*env.Position{position}, [][]actions.Option{legal}, perspectivePlayerID)
	if len(evals) == 0 {
		return e.base.Evaluate(position, legal, perspectivePlayerID)
	}
	return evals[0]
}

func (e *AsyncBatchEvaluator) EvaluateBatch(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []Evaluation {
	if e == nil || e.base == nil || len(positions) == 0 {
		return nil
	}
	call := asyncBatchCall{
		positions:   positions,
		legal:       legal,
		perspective: perspectivePlayerID,
		response:    make(chan []Evaluation, 1),
	}
	e.queue <- call
	return <-call.response
}

func (e *AsyncBatchEvaluator) run() {
	var pending []asyncBatchCall
	var timer *time.Timer
	var timerC <-chan time.Time
	for {
		select {
		case call := <-e.queue:
			pending = append(pending, call)
			if totalBatchSize(pending) >= e.maxBatch {
				if timer != nil {
					timer.Stop()
					timer = nil
					timerC = nil
				}
				e.flush(pending)
				pending = nil
				continue
			}
			if timer == nil {
				timer = time.NewTimer(e.flushDelay)
				timerC = timer.C
			}
		case <-timerC:
			e.flush(pending)
			pending = nil
			timer = nil
			timerC = nil
		}
	}
}

func totalBatchSize(calls []asyncBatchCall) int {
	total := 0
	for _, call := range calls {
		total += len(call.positions)
	}
	return total
}

func (e *AsyncBatchEvaluator) flush(calls []asyncBatchCall) {
	if len(calls) == 0 {
		return
	}
	groups := make(map[string][]asyncBatchCall)
	for _, call := range calls {
		groups[call.perspective] = append(groups[call.perspective], call)
	}
	for perspective, grouped := range groups {
		e.flushPerspective(perspective, grouped)
	}
}

func (e *AsyncBatchEvaluator) flushPerspective(perspective string, calls []asyncBatchCall) {
	var positions []*env.Position
	var legal [][]actions.Option
	for _, call := range calls {
		positions = append(positions, call.positions...)
		for i := range call.positions {
			if i < len(call.legal) {
				legal = append(legal, call.legal[i])
			} else {
				legal = append(legal, nil)
			}
		}
	}
	evals := e.base.EvaluateBatch(positions, legal, perspective)
	offset := 0
	for _, call := range calls {
		n := len(call.positions)
		end := offset + n
		if end > len(evals) {
			call.response <- e.base.EvaluateBatch(call.positions, call.legal, call.perspective)
		} else {
			call.response <- append([]Evaluation(nil), evals[offset:end]...)
		}
		offset = end
	}
}
