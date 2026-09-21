package az

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

// Even cloned positions must reject a transition: these baselines act only on
// root information, unlike one-simulation or uniform-prior MCTS.
type rootOnlyPosition struct{ Position }

func (p rootOnlyPosition) Clone() Position          { return rootOnlyPosition{p.Position.Clone()} }
func (p rootOnlyPosition) Apply(SearchAction) error { return fmt.Errorf("unexpected search traversal") }

type rootPolicyEvaluator struct{ calls int }

func (e *rootPolicyEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	e.calls += len(requests)
	results := make([]EvaluationResult, len(requests))
	for i, request := range requests {
		results[i].PolicyLogits = make([]float32, len(request.Actions))
		for j, action := range request.Actions {
			results[i].PolicyLogits[j] = float32(action.Amount)
		}
	}
	return results, nil
}

func TestArenaPolicyAndRandomAreNotSearch(t *testing.T) {
	position := rootOnlyPosition{&toyPosition{spec: immediateChoiceSpec(91)}}
	policy := &rootPolicyEvaluator{}
	actions, err := selectArenaActions(context.Background(), policy, "policy", SearchConfig{
		Simulations: 100, RootSeeds: []int64{7}, DirichletEpsilon: 1, DirichletTotalConcentration: 0.01,
	}, []Position{position})
	if err != nil || actions[0].Key() != toyAction(2).Key() || policy.calls != 1 {
		t.Fatalf("policy-only did not use exactly the root prior: %v %v calls=%d", actions, err, policy.calls)
	}
	counts := map[string]int{}
	for seed := int64(0); seed < 1000; seed++ {
		random, err := selectArenaActions(context.Background(), nil, "random", SearchConfig{
			Simulations: 100, RootSeeds: []int64{seed},
		}, []Position{position})
		if err != nil {
			t.Fatal(err)
		}
		counts[random[0].Key()]++
		reversed := immediateChoiceSpec(91)
		node := reversed.nodes[0]
		node.edges[0], node.edges[1] = node.edges[1], node.edges[0]
		reversed.nodes[0] = node
		again, err := selectArenaActions(context.Background(), nil, "random", SearchConfig{RootSeeds: []int64{seed}}, []Position{rootOnlyPosition{&toyPosition{spec: reversed}}})
		if err != nil || again[0].Key() != random[0].Key() {
			t.Fatal("random action changed under candidate permutation")
		}
	}
	if counts[toyAction(1).Key()] < 400 || counts[toyAction(1).Key()] > 600 {
		t.Fatalf("random-legal sampler is not approximately uniform: %v", counts)
	}
}

func TestArenaModesPersistAndBatchEquivalently(t *testing.T) {
	for _, mode := range []string{"policy", "random"} {
		evaluator := &batchingArenaEvaluator{id: "root-policy"}
		config := ArenaConfig{HoldoutSuiteID: "test-suite", CandidateMode: mode, BaselineMode: mode,
			CandidateSimulations: 99, BaselineSimulations: 99, GamesPerBatch: 4, MaxPlies: 2}
		cases := HoldoutCases()[:2]
		batched, err := RunPairedArena(context.Background(), evaluator, evaluator, cases, config, "test")
		if err != nil {
			t.Fatal(err)
		}
		config.GamesPerBatch = 2
		serial, err := RunPairedArena(context.Background(), evaluator, evaluator, cases, config, "test")
		if err != nil || !reflect.DeepEqual(batched.Games, serial.Games) {
			t.Fatalf("%s batch changed games: %v", mode, err)
		}
		if batched.Config.CandidateSimulations != 0 || batched.Config.BaselineSimulations != 0 || batched.Config.CandidateMode != mode {
			t.Fatal("report mislabeled non-search budget")
		}
		if mode == "random" && (batched.CandidateID != "random-legal" || evaluator.maxBatch != 0) {
			t.Fatal("random baseline used or claimed a neural evaluator")
		}
		if _, err := WriteArenaReport(t.TempDir(), batched); err != nil {
			t.Fatal(err)
		}
		originalMode := batched.Config.CandidateMode
		batched.Config.CandidateMode = ""
		if _, err := WriteArenaReport(t.TempDir(), batched); err == nil {
			t.Fatal("persisted missing decision mode")
		}
		batched.Config.CandidateMode = originalMode
		batched.Config.CandidateSimulations = 1
		if _, err := WriteArenaReport(t.TempDir(), batched); err == nil {
			t.Fatal("persisted non-search agent with search work")
		}
	}
}

func TestArenaModeValidation(t *testing.T) {
	for _, mode := range []string{"mcts", "invalid"} {
		if _, _, err := normalizeArenaMode(mode, 0); err == nil {
			t.Fatalf("accepted %s with zero traversals", mode)
		}
	}
}
