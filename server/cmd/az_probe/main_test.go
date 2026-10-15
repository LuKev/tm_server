package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lukev/tm_server/internal/az"
	"github.com/lukev/tm_server/internal/models"
)

type graph struct {
	node         int
	terminalOnly bool
}

func (g *graph) Clone() az.Position { copy := *g; return &copy }
func (g *graph) DecisionPlayer() az.PlayerID {
	if g.node == 2 {
		return "b"
	}
	return "a"
}
func (g *graph) IsTerminal() bool { return g.node == 3 }
func (g *graph) Outcome(owner az.PlayerID) float32 {
	if owner == "a" {
		return 1
	}
	return -1
}
func (g *graph) CanonicalHash() az.Hash128 {
	var h az.Hash128
	h[0] = byte(g.node)
	if g.terminalOnly {
		h[1] = 1
	}
	return h
}
func (g *graph) LegalActions() []az.SearchAction {
	if g.IsTerminal() {
		return nil
	}
	if g.node != 0 || g.terminalOnly {
		return []az.SearchAction{{Amount: 3}}
	}
	return []az.SearchAction{{Amount: 1}, {Amount: 2}, {Amount: 3}}
}
func (g *graph) Apply(action az.SearchAction) error {
	for _, a := range g.LegalActions() {
		if a.Key() == action.Key() {
			g.node = action.Amount
			return nil
		}
	}
	return fmt.Errorf("illegal action")
}

type constantEvaluator struct{}

func (constantEvaluator) Evaluate(_ context.Context, requests []az.EvaluationRequest) ([]az.EvaluationResult, error) {
	results := make([]az.EvaluationResult, len(requests))
	for i, request := range requests {
		if request.Position.IsTerminal() {
			return nil, fmt.Errorf("terminal sent to network")
		}
		logits := make([]float32, len(request.Actions))
		for j := range logits {
			logits[j] = float32(-j)
		}
		results[i] = az.EvaluationResult{PolicyLogits: logits, Value: .6}
	}
	return results, nil
}

func TestOneStepOwnerValuesAndTerminalExact(t *testing.T) {
	p := &graph{}
	values, err := oneStepValues(context.Background(), constantEvaluator{}, p)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []float32{.6, -.6, 1} {
		if values[i].RootOwnerValue != want {
			t.Fatalf("child%d=%v want%v", i, values[i], want)
		}
	}
	if values[0].LearnedChildValue == nil || *values[0].LearnedChildValue != .6 || values[1].LearnedChildValue == nil || *values[1].LearnedChildValue != .6 || values[2].LearnedChildValue != nil {
		t.Fatal("raw child values/terminal labeling wrong")
	}
	if p.node != 0 {
		t.Fatal("probe mutated root")
	}
	p.terminalOnly = true
	view, err := probe(context.Background(), constantEvaluator{}, p, az.TrajectoryStep{Actions: p.LegalActions(), ChosenIndex: 0}, 0, 11, []int{8})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Searches) != 3 {
		t.Fatal(view.Searches)
	}
	for _, index := range []int{1, 2} {
		r := view.Searches[index].Result
		if r.Visits[0] != 8 || r.QValues[0] != 1 || r.Priors[0] != 1 {
			t.Fatalf("terminal value changed: %+v", r)
		}
	}
	if view.Searches[1].Result.RootValue != .6 || view.Searches[2].Result.RootValue != 0 {
		t.Fatal("learned value ablation not applied")
	}
}

func TestValueAblationPreservesNonuniformPriorsAndSeparateCaches(t *testing.T) {
	p := &graph{}
	view, err := probe(context.Background(), constantEvaluator{}, p, az.TrajectoryStep{Actions: p.LegalActions(), ChosenIndex: 0}, 0, 11, []int{8})
	if err != nil {
		t.Fatal(err)
	}
	priors := view.Searches[0].Result.Priors
	if !(priors[0] > priors[1] && priors[1] > priors[2]) {
		t.Fatal("expected nonuniform policy")
	}
	for _, search := range view.Searches {
		if !reflect.DeepEqual(search.Result.Priors, priors) {
			t.Fatal("ablation changed root priors")
		}
		wantValue := float32(.6)
		if search.Mode == "zero_learned_value" {
			wantValue = 0
		}
		if search.Result.RootValue != wantValue {
			t.Fatal("root evaluation cache contaminated")
		}
	}
	shared := &reusedEvaluator{results: []az.EvaluationResult{{PolicyLogits: []float32{3, 2, 1}, Value: .6}}}
	results, err := (zeroValues{shared}).Evaluate(context.Background(), []az.EvaluationRequest{{Position: p, Actions: p.LegalActions()}})
	if err != nil || results[0].Value != 0 || shared.results[0].Value != .6 || !reflect.DeepEqual(results[0].PolicyLogits, shared.results[0].PolicyLogits) {
		t.Fatal("wrapper mutated evaluator result or policy")
	}
}

type reusedEvaluator struct{ results []az.EvaluationResult }

func (e *reusedEvaluator) Evaluate(context.Context, []az.EvaluationRequest) ([]az.EvaluationResult, error) {
	return e.results, nil
}

func TestReplayAndRejectModifiedActionHash(t *testing.T) {
	p, err := az.NewBaseGame(734, models.FactionNomads, models.FactionGiants)
	if err != nil {
		t.Fatal(err)
	}
	trajectories, err := az.GenerateSelfPlay(context.Background(), []*az.GamePosition{p}, []int64{734}, az.UniformEvaluator{}, az.SelfPlayConfig{Search: az.SearchConfig{Simulations: 1}, SearchSeeds: []int64{912}, EngineCommit: "test", MaxPlies: 2000})
	if err != nil {
		t.Fatal(err)
	}
	trajectory := trajectories[0]
	ref, err := az.WriteTrajectoryShard(t.TempDir(), trajectory)
	if err != nil {
		t.Fatal(err)
	}
	loaded, _, err := az.ReadTrajectoryShard(ref.Path)
	if err != nil {
		t.Fatal(err)
	}
	positions, err := replayPositions(loaded, []int{0, len(loaded.Steps) - 1})
	if err != nil || len(positions) != 2 {
		t.Fatalf("replay: %v", err)
	}
	first := loaded.Steps[0]
	loaded.Steps[0].StateHash = "wrong"
	if _, err := replayPositions(loaded, []int{0}); err == nil {
		t.Fatal("accepted bad hash")
	}
	loaded.Steps[0] = first
	loaded.Steps[0].Actions = append([]az.SearchAction(nil), first.Actions...)
	loaded.Steps[0].Actions[0].Amount = 123
	if _, err := replayPositions(loaded, []int{0}); err == nil {
		t.Fatal("accepted wrong legal identity")
	}
	if err := os.Chmod(ref.Path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ref.Path, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := az.ReadTrajectoryShard(ref.Path); err == nil {
		t.Fatal("accepted modified shard")
	}
}

func TestImmutablePublicationAndBoundedFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	if err := publish(path, "first"); err != nil {
		t.Fatal(err)
	}
	if err := publish(path, "second"); err == nil {
		t.Fatal("overwrote report")
	}
	bytes, _ := os.ReadFile(path)
	if string(bytes) != `"first"` {
		t.Fatal(string(bytes))
	}
	for _, text := range []string{"", "8,8", "999", "8,32,128,8"} {
		if _, err := parseList(text, 3, func(v int) bool { return v == 8 || v == 32 || v == 128 }); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
}
