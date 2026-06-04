package model

import (
	"sync"
	"testing"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

func TestAsyncBatchEvaluatorMergesConcurrentCalls(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := position.LegalActions()
	base := &recordingBatchEvaluator{fallback: NewHeuristicEvaluator()}
	evaluator := NewAsyncBatchEvaluator(base, 4, 20*time.Millisecond)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			evals := evaluator.(BatchEvaluator).EvaluateBatch([]*env.Position{position}, [][]actions.Option{legal}, "p1")
			if len(evals) != 1 {
				t.Errorf("got %d evals, want 1", len(evals))
			}
		}()
	}
	wg.Wait()
	if base.maxSeen < 4 {
		t.Fatalf("max merged batch = %d, want at least 4", base.maxSeen)
	}
}

type recordingBatchEvaluator struct {
	fallback *HeuristicEvaluator
	mu       sync.Mutex
	maxSeen  int
}

func (e *recordingBatchEvaluator) Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
	return e.fallback.Evaluate(position, legal, perspectivePlayerID)
}

func (e *recordingBatchEvaluator) EvaluateBatch(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []Evaluation {
	e.mu.Lock()
	if len(positions) > e.maxSeen {
		e.maxSeen = len(positions)
	}
	e.mu.Unlock()
	return e.fallback.EvaluateBatch(positions, legal, perspectivePlayerID)
}
