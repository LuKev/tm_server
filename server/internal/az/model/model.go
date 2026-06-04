package model

import (
	"math"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

// Evaluation is the policy/value model output for one position.
type Evaluation struct {
	Priors map[string]float64 `json:"priors"`
	Value  float64            `json:"value"`
}

// Evaluator provides policy priors and value estimates.
type Evaluator interface {
	Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation
}

// BatchEvaluator can evaluate multiple positions in one call. MCTS uses this
// when available to amortize neural inference overhead.
type BatchEvaluator interface {
	Evaluator
	EvaluateBatch(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []Evaluation
}

// HeuristicEvaluator is the bootstrap evaluator used until a trained network
// is available. It gives every legal move non-zero probability and mildly
// prefers normal point-producing actions over conversions.
type HeuristicEvaluator struct{}

func NewHeuristicEvaluator() *HeuristicEvaluator {
	return &HeuristicEvaluator{}
}

func (e *HeuristicEvaluator) Evaluate(position *env.Position, legal []actions.Option, perspectivePlayerID string) Evaluation {
	priors := make(map[string]float64, len(legal))
	total := 0.0
	for _, option := range legal {
		weight := actionWeight(option)
		if weight <= 0 {
			weight = 1
		}
		priors[option.ID] = weight
		total += weight
	}
	if total > 0 {
		for id, weight := range priors {
			priors[id] = weight / total
		}
	}
	value := 0.0
	if position != nil {
		value = position.ValueFor(perspectivePlayerID)
	}
	return Evaluation{Priors: priors, Value: math.Max(-1, math.Min(1, value))}
}

func (e *HeuristicEvaluator) EvaluateBatch(positions []*env.Position, legal [][]actions.Option, perspectivePlayerID string) []Evaluation {
	out := make([]Evaluation, 0, len(positions))
	for i, position := range positions {
		options := []actions.Option(nil)
		if i < len(legal) {
			options = legal[i]
		}
		out = append(out, e.Evaluate(position, options, perspectivePlayerID))
	}
	return out
}

func actionWeight(option actions.Option) float64 {
	switch option.Type {
	case "pass", "pass_final":
		return 0.7
	case "conversion", "burn":
		return 0.2
	case "transform_build", "upgrade", "power", "power_spade_build", "special_giants_build", "special_nomads_build":
		return 2.0
	case "cult_priest", "advance_shipping", "advance_digging", "special_auren", "special_water2":
		return 1.4
	default:
		return 1.0
	}
}
