// az_probe inspects search on verified archived self-play positions. It never
// produces training data or adopts a checkpoint.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/lukev/tm_server/internal/az"
)

type zeroValues struct{ az.Evaluator }

func (e zeroValues) Evaluate(ctx context.Context, requests []az.EvaluationRequest) ([]az.EvaluationResult, error) {
	results, err := e.Evaluator.Evaluate(ctx, requests)
	if err != nil {
		return nil, err
	}
	// Search handles terminal outcomes itself; only learned values are ablated.
	copyResults := append([]az.EvaluationResult(nil), results...)
	for i := range copyResults {
		copyResults[i].Value = 0
	}
	return copyResults, nil
}

type childValue struct {
	Action            az.SearchAction `json:"action"`
	DecisionOwner     az.PlayerID     `json:"child_decision_owner"`
	Terminal          bool            `json:"terminal"`
	RootOwnerValue    float32         `json:"root_owner_value"`
	LearnedChildValue *float32        `json:"learned_child_owner_value,omitempty"`
}

type searchView struct {
	Mode        string          `json:"mode"`
	Simulations int             `json:"simulations"`
	Result      az.SearchResult `json:"result"`
	Selected    az.SearchAction `json:"selected"`
}

type positionView struct {
	Ply            int             `json:"ply"`
	Hash           string          `json:"state_hash"`
	DecisionOwner  az.PlayerID     `json:"decision_owner"`
	Round          int             `json:"round"`
	Phase          int             `json:"phase"`
	SearchSeed     int64           `json:"search_seed"`
	ArchivedChosen az.SearchAction `json:"archived_chosen"`
	ArchivedVisits []int           `json:"archived_visits"`
	ChildValues    []childValue    `json:"one_step_children"`
	Searches       []searchView    `json:"searches"`
}

func replayPositions(t az.Trajectory, plies []int) (map[int]az.Position, error) {
	if az.IsHoldoutSeed(t.Manifest.Seed) {
		return nil, fmt.Errorf("probe accepts self-play, not reserved evaluation setups")
	}
	p, err := az.NewBaseGame(t.Manifest.Seed, t.Manifest.Factions[0], t.Manifest.Factions[1])
	if err != nil {
		return nil, err
	}
	wanted := make(map[int]bool)
	for _, ply := range plies {
		if ply < 0 || ply >= len(t.Steps) {
			return nil, fmt.Errorf("ply %d outside trajectory", ply)
		}
		wanted[ply] = true
	}
	positions := make(map[int]az.Position)
	for ply, step := range t.Steps {
		if p.CanonicalHash().String() != step.StateHash || string(p.DecisionPlayer()) != fmt.Sprintf("p%d", step.DecisionSeat) {
			return nil, fmt.Errorf("replay state/owner mismatch at ply %d", ply)
		}
		actions := p.LegalActions()
		if len(actions) != len(step.Actions) {
			return nil, fmt.Errorf("legal action count mismatch at ply %d", ply)
		}
		for i := range actions {
			if actions[i].Key() != step.Actions[i].Key() {
				return nil, fmt.Errorf("legal action identity mismatch at ply %d index %d", ply, i)
			}
		}
		if wanted[ply] {
			positions[ply] = p.Clone()
		}
		if step.ChosenIndex < 0 || step.ChosenIndex >= len(actions) {
			return nil, fmt.Errorf("invalid archived action at ply %d", ply)
		}
		if err := p.Apply(actions[step.ChosenIndex]); err != nil {
			return nil, err
		}
		if p.CanonicalHash().String() != step.NextStateHash {
			return nil, fmt.Errorf("next state mismatch at ply %d", ply)
		}
	}
	if !p.IsTerminal() || p.CanonicalHash().String() != t.TerminalStateHash {
		return nil, fmt.Errorf("terminal replay mismatch")
	}
	return positions, nil
}

func oneStepValues(ctx context.Context, evaluator az.Evaluator, p az.Position) ([]childValue, error) {
	actions := p.LegalActions()
	values := make([]childValue, len(actions))
	owner := p.DecisionPlayer()
	// Small batches avoid padding thousands of legal actions from unrelated children.
	for start := 0; start < len(actions); start += 16 {
		requests := []az.EvaluationRequest{}
		indices := []int{}
		for i := start; i < len(actions) && i < start+16; i++ {
			child := p.Clone()
			if err := child.Apply(actions[i]); err != nil {
				return nil, err
			}
			values[i] = childValue{Action: actions[i], DecisionOwner: child.DecisionPlayer(), Terminal: child.IsTerminal()}
			if child.IsTerminal() {
				values[i].RootOwnerValue = child.Outcome(owner)
			} else {
				requests = append(requests, az.EvaluationRequest{Position: child, Actions: child.LegalActions()})
				indices = append(indices, i)
			}
		}
		if len(requests) == 0 {
			continue
		}
		results, err := evaluator.Evaluate(ctx, requests)
		if err != nil {
			return nil, err
		}
		if len(results) != len(requests) {
			return nil, fmt.Errorf("child evaluation count mismatch")
		}
		for j, result := range results {
			v := result.Value
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || v < -1 || v > 1 {
				return nil, fmt.Errorf("invalid child value")
			}
			rawValue := v
			values[indices[j]].LearnedChildValue = &rawValue
			if values[indices[j]].DecisionOwner != owner {
				v = -v
			}
			values[indices[j]].RootOwnerValue = v
		}
	}
	return values, nil
}

func probe(ctx context.Context, evaluator az.Evaluator, p az.Position, step az.TrajectoryStep, ply int, seed int64, budgets []int) (positionView, error) {
	view := positionView{Ply: ply, Hash: p.CanonicalHash().String(), DecisionOwner: p.DecisionPlayer(), ArchivedChosen: step.Actions[step.ChosenIndex], ArchivedVisits: step.Visits}
	view.SearchSeed = seed
	if gamePosition, ok := p.(*az.GamePosition); ok {
		state := gamePosition.StateClone()
		view.Round, view.Phase = state.Round, int(state.Phase)
	}
	children, err := oneStepValues(ctx, evaluator, p)
	if err != nil {
		return view, err
	}
	view.ChildValues = children
	for _, budget := range append([]int{0}, budgets...) {
		for _, mode := range []string{"normal", "zero_learned_value"} {
			if budget == 0 && mode != "normal" {
				continue
			}
			var selectedEvaluator az.Evaluator = evaluator
			if mode == "zero_learned_value" {
				selectedEvaluator = zeroValues{evaluator}
			}
			search, err := az.NewMCTS(selectedEvaluator, az.SearchConfig{Simulations: budget, CPUCT: 1.5, Seed: seed})
			if err != nil {
				return view, err
			}
			result, err := search.Search(ctx, p)
			if err != nil {
				return view, err
			}
			selected, err := result.SelectAction(0, nil)
			if err != nil {
				return view, err
			}
			if budget == 0 {
				mode = "policy_only"
			}
			view.Searches = append(view.Searches, searchView{mode, budget, result, selected})
		}
	}
	return view, nil
}

func parseList(text string, limit int, allowed func(int) bool) ([]int, error) {
	parts := strings.Split(text, ",")
	if len(parts) > limit {
		return nil, fmt.Errorf("too many values (maximum %d)", limit)
	}
	values, seen := []int{}, map[int]bool{}
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || !allowed(value) || seen[value] {
			return nil, fmt.Errorf("invalid or duplicate value %q", part)
		}
		values, seen[value] = append(values, value), true
	}
	return values, nil
}

func publish(path string, report any) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".probe-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Chmod(0o444); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Hard-link publication cannot overwrite an existing report.
	return os.Link(f.Name(), path)
}

func run() error {
	trajectory := flag.String("trajectory", "", "content-addressed self-play shard")
	pliesFlag := flag.String("plies", "", "zero-based archived decision indices, max16")
	budgetsFlag := flag.String("budgets", "8", "comma-separated subset of 8,32,128; policy-only always included")
	checkpoint := flag.String("checkpoint", "", "frozen checkpoint")
	latest := flag.String("checkpoint-latest", "", "resolve and pin latest reference once")
	inference := flag.String("inference", "", "az_inference executable")
	device := flag.String("device", "cpu", "inference device")
	output := flag.String("output", "", "new immutable report file")
	commit := flag.String("engine-commit", "", "diagnostic source revision")
	seed := flag.Int64("search-seed", 1, "controlled diagnostic seed base; not replay of archived self-play RNG")
	flag.Parse()
	if *trajectory == "" || *inference == "" || *output == "" || *commit == "" {
		return fmt.Errorf("trajectory, inference, output and engine-commit required")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) {
		return fmt.Errorf("output already exists or inaccessible")
	}
	plies, err := parseList(*pliesFlag, 16, func(v int) bool { return v >= 0 })
	if err != nil {
		return err
	}
	budgets, err := parseList(*budgetsFlag, 3, func(v int) bool { return v == 8 || v == 32 || v == 128 })
	if err != nil {
		return err
	}
	path, err := az.ResolveCheckpointReference(*checkpoint, *latest)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("frozen checkpoint required")
	}
	t, ref, err := az.ReadTrajectoryShard(*trajectory)
	if err != nil {
		return err
	}
	positions, err := replayPositions(t, plies)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	evaluator, err := az.StartPythonEvaluatorWithOptions(ctx, *inference, az.PythonEvaluatorOptions{Checkpoint: path, ModelConfig: "auto", Device: *device, TorchThreads: 1})
	if err != nil {
		return err
	}
	defer evaluator.Close()
	views := []positionView{}
	for _, ply := range plies {
		view, err := probe(ctx, evaluator, positions[ply], t.Steps[ply], ply, *seed+int64(ply), budgets)
		if err != nil {
			return err
		}
		views = append(views, view)
	}
	return publish(*output, map[string]any{"format_version": 1, "purpose": "archived self-play search diagnostic; not strength evidence", "engine_commit": *commit, "trajectory_sha256": ref.SHA256, "trajectory_manifest": t.Manifest, "checkpoint": evaluator.Metadata(), "c_puct": 1.5, "root_noise_epsilon": 0, "selection_temperature": 0, "search_seed_base": *seed, "search_seed_rule": "diagnostic seed base + ply; not archived self-play RNG replay", "positions": views})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
