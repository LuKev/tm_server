package arena

import (
	"testing"

	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
)

func TestEvaluateReturnsGames(t *testing.T) {
	result, err := Evaluate(model.NewHeuristicEvaluator(), model.NewHeuristicEvaluator(), Config{
		Games:      2,
		MaxPlies:   1,
		Scenario:   "random_base",
		RandomSeed: 1,
		Search: mcts.Config{
			Simulations: 1,
			CPUCT:       1.5,
			Temperature: 0,
			MaxDepth:    1,
			RandomSeed:  2,
		},
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result.Games != 2 {
		t.Fatalf("games = %d, want 2", result.Games)
	}
}

func TestEvaluateWithWorkersReturnsGames(t *testing.T) {
	result, err := Evaluate(model.NewHeuristicEvaluator(), model.NewHeuristicEvaluator(), Config{
		Games:      4,
		MaxPlies:   1,
		Scenario:   "training_mix",
		Workers:    2,
		RandomSeed: 3,
		Search: mcts.Config{
			Simulations: 0,
			CPUCT:       1.5,
			Temperature: 0,
			MaxDepth:    1,
		},
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result.Games != 4 {
		t.Fatalf("games = %d, want 4", result.Games)
	}
	if result.Workers != 2 {
		t.Fatalf("workers = %d, want 2", result.Workers)
	}
	if result.SearchNanos <= 0 {
		t.Fatalf("expected search timing metrics: %#v", result)
	}
}

func TestDecidePromotionUsesConfidenceGate(t *testing.T) {
	decision := DecidePromotion(Result{
		Games:       100,
		WinRate:     0.57,
		WinRateCI95: [2]float64{0.48, 0.66},
	}, PromotionPolicy{
		MinWinRate:        0.55,
		MinGames:          50,
		MinCI95LowerBound: 0.50,
	})
	if decision.Promoted {
		t.Fatal("expected confidence gate to block promotion")
	}
	if len(decision.BlockingReasons) != 1 {
		t.Fatalf("blocking reasons = %v, want one confidence reason", decision.BlockingReasons)
	}
}

func TestDecidePromotionAutoPromotesFirstCandidate(t *testing.T) {
	decision := DecidePromotion(Result{
		Games:       2,
		WinRate:     0,
		WinRateCI95: [2]float64{0, 0},
	}, PromotionPolicy{
		MinWinRate:        0.55,
		MinGames:          50,
		MinCI95LowerBound: 0.50,
		AutoPromote:       true,
	})
	if !decision.Promoted {
		t.Fatalf("expected auto promotion, got blockers %v", decision.BlockingReasons)
	}
}
