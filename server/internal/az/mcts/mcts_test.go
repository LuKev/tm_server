package mcts

import (
	"testing"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/model"
)

func TestSearchReturnsRankedActions(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	result := Search(position, model.NewHeuristicEvaluator(), Config{
		Simulations: 1,
		CPUCT:       1.5,
		Temperature: 1,
		RandomSeed:  1,
		MaxDepth:    1,
	})
	if len(result.Actions) == 0 {
		t.Fatal("expected ranked actions")
	}
	if result.Selected.ID == "" {
		t.Fatal("expected selected action")
	}
}

func TestSearchUsesBatchEvaluator(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	evaluator := &countingBatchEvaluator{fallback: model.NewHeuristicEvaluator()}
	result := Search(position, evaluator, Config{
		Simulations: 4,
		BatchSize:   2,
		CPUCT:       1.5,
		Temperature: 1,
		RandomSeed:  1,
		MaxDepth:    2,
	})
	if len(result.Actions) == 0 {
		t.Fatal("expected ranked actions")
	}
	if evaluator.batchCalls == 0 {
		t.Fatal("expected batch evaluator to be used")
	}
}

func TestSearchZeroSimulationsUsesPolicyPriors(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	result := Search(position, model.NewHeuristicEvaluator(), Config{
		Simulations: 0,
		CPUCT:       1.5,
		Temperature: 1,
		RandomSeed:  1,
		MaxDepth:    1,
	})
	if result.Simulations != 0 {
		t.Fatalf("simulations = %d, want 0", result.Simulations)
	}
	if len(result.Actions) == 0 {
		t.Fatal("expected policy-ranked actions")
	}
	if result.Actions[0].Visits != 0 {
		t.Fatalf("visits = %d, want 0 in policy-prior mode", result.Actions[0].Visits)
	}
	if result.Actions[0].Prob <= 0 {
		t.Fatalf("probability = %v, want positive prior probability", result.Actions[0].Prob)
	}
}

func TestTreeCanAdvanceToSelectedChild(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	tree := NewTree(position)
	result := tree.Search(model.NewHeuristicEvaluator(), Config{
		Simulations: 2,
		CPUCT:       1.5,
		Temperature: 1,
		RandomSeed:  1,
		MaxDepth:    2,
	})
	if result.Selected.ID == "" {
		t.Fatal("expected selected action")
	}
	var selected actions.Option
	for _, option := range position.LegalActions() {
		if option.ID == result.Selected.ID {
			selected = option
			break
		}
	}
	next, err := position.Apply(selected)
	if err != nil {
		t.Fatalf("apply selected action: %v", err)
	}
	tree.Advance(selected.ID, next)
	nextResult := tree.Search(model.NewHeuristicEvaluator(), Config{
		Simulations: 1,
		CPUCT:       1.5,
		Temperature: 1,
		RandomSeed:  2,
		MaxDepth:    2,
	})
	if len(nextResult.Actions) == 0 {
		t.Fatal("expected ranked actions after advancing tree")
	}
}

type countingBatchEvaluator struct {
	fallback   *model.HeuristicEvaluator
	batchCalls int
}

func (e *countingBatchEvaluator) Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) model.Evaluation {
	return e.fallback.Evaluate(position, legal, perspectivePlayerID)
}

func (e *countingBatchEvaluator) EvaluateBatch(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []model.Evaluation {
	e.batchCalls++
	return e.fallback.EvaluateBatch(positions, legal, perspectivePlayerID)
}
