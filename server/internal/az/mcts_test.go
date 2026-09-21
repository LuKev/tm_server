package az

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

type toyEdge struct {
	action SearchAction
	next   int
}

type toyNode struct {
	player   PlayerID
	edges    []toyEdge
	winner   PlayerID
	terminal bool
	value    float32
}

type toySpec struct {
	id    byte
	nodes map[int]toyNode
}

type toyPosition struct {
	spec *toySpec
	node int
}

func (p *toyPosition) Clone() Position { return &toyPosition{spec: p.spec, node: p.node} }

func (p *toyPosition) DecisionPlayer() PlayerID {
	node := p.spec.nodes[p.node]
	if node.terminal {
		return ""
	}
	return node.player
}

func (p *toyPosition) LegalActions() []SearchAction {
	node := p.spec.nodes[p.node]
	actions := make([]SearchAction, len(node.edges))
	for i, edge := range node.edges {
		actions[i] = edge.action
	}
	return actions
}

func (p *toyPosition) Apply(action SearchAction) error {
	for _, edge := range p.spec.nodes[p.node].edges {
		if edge.action.Key() == action.Key() {
			p.node = edge.next
			return nil
		}
	}
	return fmt.Errorf("invalid toy action %s", action.Key())
}

func (p *toyPosition) IsTerminal() bool { return p.spec.nodes[p.node].terminal }

func (p *toyPosition) Outcome(player PlayerID) float32 {
	node := p.spec.nodes[p.node]
	if !node.terminal || node.winner == "" {
		return 0
	}
	if node.winner == player {
		return 1
	}
	return -1
}

func (p *toyPosition) CanonicalHash() Hash128 {
	return Hash128{p.spec.id, byte(p.node), byte(p.node >> 8)}
}

func toyAction(id int) SearchAction {
	return SearchAction{Kind: game.ActionConversion, Conversion: game.ConversionWorkerToCoin, Amount: id}
}

func immediateChoiceSpec(id byte) *toySpec {
	return &toySpec{id: id, nodes: map[int]toyNode{
		0: {player: "A", edges: []toyEdge{{toyAction(1), 1}, {toyAction(2), 2}}},
		1: {terminal: true, winner: "B"},
		2: {terminal: true, winner: "A"},
	}}
}

func variedImmediateChoiceSpec(id byte, branches, winningAction int) *toySpec {
	nodes := map[int]toyNode{}
	edges := make([]toyEdge, branches)
	for action := 1; action <= branches; action++ {
		nodeID := action
		winner := PlayerID("B")
		if action == winningAction {
			winner = "A"
		}
		nodes[nodeID] = toyNode{terminal: true, winner: winner}
		edges[action-1] = toyEdge{action: toyAction(action), next: nodeID}
	}
	nodes[0] = toyNode{player: "A", edges: edges}
	return &toySpec{id: id, nodes: nodes}
}

func TestPUCTFindsKnownOptimalMove(t *testing.T) {
	search, err := NewMCTS(UniformEvaluator{}, SearchConfig{Simulations: 32, CPUCT: 1.5})
	if err != nil {
		t.Fatal(err)
	}
	position := &toyPosition{spec: immediateChoiceSpec(1)}
	result, err := search.Search(context.Background(), position)
	if err != nil {
		t.Fatal(err)
	}
	action, err := result.SelectAction(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if action.Key() != toyAction(2).Key() {
		t.Fatalf("search selected losing action %s", action.Key())
	}
	if result.Visits[1] <= result.Visits[0] || result.QValues[1] <= 0 {
		t.Fatalf("unexpected root statistics: visits=%v q=%v", result.Visits, result.QValues)
	}
}

func TestFreshRootTraversalUsesPrior(t *testing.T) {
	search, err := NewMCTS(constantValueEvaluator{preferredKey: toyAction(2).Key()}, SearchConfig{Simulations: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := search.Search(context.Background(), &toyPosition{spec: immediateChoiceSpec(121)})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Visits, []int{0, 1}) {
		t.Fatalf("first traversal ignored the strongest prior: %v", result.Visits)
	}
}

func TestSearchCandidateOrderingAndSeededTies(t *testing.T) {
	selected := map[string]bool{}
	for seed := int64(0); seed < 24; seed++ {
		for _, simulations := range []int{0, 1, 12} {
			for _, epsilon := range []float64{0, 0.25} {
				config := SearchConfig{Simulations: simulations, Seed: seed,
					DirichletEpsilon: epsilon, DirichletTotalConcentration: 10}
				var want SearchResult
				for order := 0; order < 2; order++ {
					spec := variedImmediateChoiceSpec(122, 6, 4)
					if order == 1 {
						node := spec.nodes[0]
						for i, j := 0, len(node.edges)-1; i < j; i, j = i+1, j-1 {
							node.edges[i], node.edges[j] = node.edges[j], node.edges[i]
						}
						spec.nodes[0] = node
					}
					search, err := NewMCTS(UniformEvaluator{}, config)
					if err != nil {
						t.Fatal(err)
					}
					got, err := search.Search(context.Background(), &toyPosition{spec: spec})
					if err != nil {
						t.Fatal(err)
					}
					if order == 0 {
						want = got
					} else if !reflect.DeepEqual(got, want) {
						t.Fatalf("candidate order changed seed %d budget %d noise %v", seed, simulations, epsilon)
					}
					if simulations == 1 && epsilon == 0 {
						action, err := got.SelectAction(0, nil)
						if err != nil {
							t.Fatal(err)
						}
						selected[action.Key()] = true
					}
				}
			}
		}
	}
	if len(selected) < 3 {
		t.Fatalf("uniform-prior ties still favor a fixed action: %v", selected)
	}
}

func TestVisitPolicyTieIsCandidateOrderIndependent(t *testing.T) {
	first := SearchResult{RootHash: Hash128{123}, tieSeed: 21,
		Actions: []SearchAction{toyAction(1), toyAction(2), toyAction(3)},
		Priors:  []float32{0.25, 0.5, 0.5}, Visits: []int{4, 4, 4}}
	second := first
	second.Actions = []SearchAction{first.Actions[2], first.Actions[1], first.Actions[0]}
	second.Priors = []float32{0.5, 0.5, 0.25}
	for _, budget := range []int{0, 4} {
		first.Visits = []int{budget, budget, budget}
		second.Visits = []int{budget, budget, budget}
		a, err := first.SelectAction(0, nil)
		if err != nil {
			t.Fatal(err)
		}
		b, err := second.SelectAction(0, nil)
		if err != nil {
			t.Fatal(err)
		}
		if a.Key() != b.Key() || a.Key() == toyAction(1).Key() {
			t.Fatalf("tie selection not invariant/prior-aware: %v versus %v", a, b)
		}
	}
}

func TestExactForcedTreeBackupsAndBudget(t *testing.T) {
	for _, winner := range []PlayerID{"A", "B", ""} {
		for _, owner := range []PlayerID{"A", "B"} {
			t.Run(fmt.Sprintf("winner_%s_middle_%s", winner, owner), func(t *testing.T) {
				spec := &toySpec{id: 124, nodes: map[int]toyNode{
					0: {player: "A", edges: []toyEdge{{toyAction(1), 1}}},
					1: {player: owner, value: 0.5, edges: []toyEdge{{toyAction(2), 2}}},
					2: {terminal: true, winner: winner},
				}}
				for _, budget := range []int{0, 1, 2, 9} {
					evaluator := &countingEvaluator{}
					search, err := NewMCTS(evaluator, SearchConfig{Simulations: budget})
					if err != nil {
						t.Fatal(err)
					}
					result, err := search.Search(context.Background(), &toyPosition{spec: spec})
					if err != nil {
						t.Fatal(err)
					}
					if result.Visits[0] != budget {
						t.Fatalf("visits %v, want %d", result.Visits, budget)
					}
					terminal := (&toyPosition{spec: spec, node: 2}).Outcome("A")
					wantQ := float32(0)
					if budget > 1 {
						wantQ = terminal * float32(budget-1) / float32(budget)
					}
					if math.Abs(float64(result.QValues[0]-wantQ)) > 1e-7 {
						t.Fatalf("budget %d Q=%v want=%v", budget, result.QValues, wantQ)
					}
					wantRequests := 1
					if budget > 0 {
						wantRequests++
					}
					if evaluator.requests != wantRequests {
						t.Fatalf("evaluated %d positions, want %d", evaluator.requests, wantRequests)
					}
				}
			})
		}
	}
	search, err := NewMCTS(&countingEvaluator{}, SearchConfig{Simulations: 9})
	if err != nil {
		t.Fatal(err)
	}
	result, err := search.Search(context.Background(), &toyPosition{spec: immediateChoiceSpec(125), node: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Actions) != 0 || search.stats.EvaluationPositions != 0 || search.stats.SimulationTraversals != 0 {
		t.Fatalf("terminal root performed work or returned actions: %+v %+v", result, search.stats)
	}
}

func TestBackupMixedOwnerPathHasExactValues(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		for _, winner := range []PlayerID{"A", "B", ""} {
			path := []*searchEdge{{parentPlayer: "A"}, {parentPlayer: "A"}, {parentPlayer: "B"}, {parentPlayer: "A"}}
			spec := &toySpec{id: 126, nodes: map[int]toyNode{0: {terminal: terminal, winner: winner}}}
			leaf := &searchNode{position: &toyPosition{spec: spec}, player: "B", value: 0.75}
			backupPath(path, leaf)
			for index, edge := range path {
				want := float64(-0.75)
				if edge.parentPlayer == "B" {
					want = 0.75
				}
				if terminal {
					want = 0
					if winner != "" {
						want = -1
						if edge.parentPlayer == winner {
							want = 1
						}
					}
				}
				if edge.visits != 1 || edge.q() != want {
					t.Fatalf("terminal=%v winner=%s edge=%d: visits=%d Q=%v want=%v", terminal, winner, index, edge.visits, edge.q(), want)
				}
			}
		}
	}
}

func TestSearchTelemetryCountsBatchingAndCacheReuse(t *testing.T) {
	telemetry := &SearchTelemetry{}
	search, err := NewMCTS(UniformEvaluator{}, SearchConfig{
		Simulations: 4,
		CPUCT:       1.5,
		Telemetry:   telemetry,
	})
	if err != nil {
		t.Fatal(err)
	}
	positions := []Position{
		&toyPosition{spec: immediateChoiceSpec(91)},
		&toyPosition{spec: immediateChoiceSpec(91)},
	}
	if _, err := search.SearchBatch(context.Background(), positions); err != nil {
		t.Fatal(err)
	}
	stats := telemetry.Snapshot()
	if stats.SearchCalls != 1 || stats.RootPositions != 2 || stats.SimulationTraversals != 8 {
		t.Fatalf("unexpected basic telemetry: %+v", stats)
	}
	if stats.EvaluationBatches == 0 || stats.EvaluationPositions == 0 || stats.ExpandedNodes == 0 {
		t.Fatalf("missing search work telemetry: %+v", stats)
	}
	if stats.EvaluationBatchDedupHits == 0 {
		t.Fatalf("identical roots were not counted as batch deduplication: %+v", stats)
	}
	if stats.WallDuration <= 0 {
		t.Fatalf("search duration was not recorded: %+v", stats)
	}
	if got := stats.ByPhaseFaction["synthetic"].RootPositions; got != 2 {
		t.Fatalf("synthetic root dimension count = %d, want 2", got)
	}
	if got := stats.ByPhaseFaction["synthetic"].EvaluationBatchDedupHits; got == 0 {
		t.Fatal("batch deduplication was not attributed to its metric dimension")
	}
}

func TestDurationPercentileUsesNearestRank(t *testing.T) {
	values := []time.Duration{9 * time.Millisecond, time.Millisecond, 5 * time.Millisecond, 3 * time.Millisecond}
	if got := durationPercentile(values, 0.50); got != 3*time.Millisecond {
		t.Fatalf("p50 = %v, want 3ms", got)
	}
	if got := durationPercentile(values, 0.95); got != 9*time.Millisecond {
		t.Fatalf("p95 = %v, want 9ms", got)
	}
	if got := durationPercentile(nil, 0.95); got != 0 {
		t.Fatalf("empty p95 = %v, want 0", got)
	}
}

type toyValueEvaluator struct{}

func (toyValueEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	results := make([]EvaluationResult, len(requests))
	for i, request := range requests {
		position := request.Position.(*toyPosition)
		results[i] = EvaluationResult{
			PolicyLogits: make([]float32, len(request.Actions)),
			Value:        position.spec.nodes[position.node].value,
		}
	}
	return results, nil
}

type requestSensitiveEvaluator struct{}

func (requestSensitiveEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	results := make([]EvaluationResult, len(requests))
	for requestIndex, request := range requests {
		position, ok := request.Position.(*toyPosition)
		if !ok {
			return nil, fmt.Errorf("unexpected position type %T", request.Position)
		}
		results[requestIndex].PolicyLogits = make([]float32, len(request.Actions))
		for actionIndex, action := range request.Actions {
			results[requestIndex].PolicyLogits[actionIndex] = float32((int(position.spec.id)%7 + 1) * action.Amount)
		}
		results[requestIndex].Value = float32(int(position.spec.id)%9-4) / 4
	}
	return results, nil
}

func TestBackupUsesDecisionOwnerIdentity(t *testing.T) {
	tests := []struct {
		name        string
		childPlayer PlayerID
		wantQ       float32
	}{
		{name: "same_player_edge", childPlayer: "A", wantQ: 0.75},
		{name: "opponent_reaction_edge", childPlayer: "B", wantQ: -0.75},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := &toySpec{id: byte(10 + i), nodes: map[int]toyNode{
				0: {player: "A", edges: []toyEdge{{toyAction(1), 1}}},
				1: {player: test.childPlayer, value: 0.75, edges: []toyEdge{{toyAction(2), 2}}},
				2: {terminal: true, winner: "A"},
			}}
			search, err := NewMCTS(toyValueEvaluator{}, SearchConfig{Simulations: 1})
			if err != nil {
				t.Fatal(err)
			}
			result, err := search.Search(context.Background(), &toyPosition{spec: spec})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.QValues) != 1 || result.QValues[0] != test.wantQ {
				t.Fatalf("Q=%v, want %v", result.QValues, test.wantQ)
			}
		})
	}
}

type countingEvaluator struct {
	calls      int
	requests   int
	batchSizes []int
}

func (e *countingEvaluator) Evaluate(ctx context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	e.calls++
	e.requests += len(requests)
	e.batchSizes = append(e.batchSizes, len(requests))
	return (UniformEvaluator{}).Evaluate(ctx, requests)
}

func TestSerialAndBatchedSearchAgree(t *testing.T) {
	positions := []Position{
		&toyPosition{spec: immediateChoiceSpec(20)},
		&toyPosition{spec: immediateChoiceSpec(21)},
	}
	config := SearchConfig{Simulations: 24, CPUCT: 1.25, Seed: 9}
	want := make([]SearchResult, len(positions))
	for i, position := range positions {
		serial, err := NewMCTS(UniformEvaluator{}, config)
		if err != nil {
			t.Fatal(err)
		}
		want[i], err = serial.Search(context.Background(), position)
		if err != nil {
			t.Fatal(err)
		}
	}
	evaluator := &countingEvaluator{}
	batched, err := NewMCTS(evaluator, config)
	if err != nil {
		t.Fatal(err)
	}
	got, err := batched.SearchBatch(context.Background(), positions)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("batched search differs from serial\ngot=%#v\nwant=%#v", got, want)
	}
	foundBatch := false
	for _, size := range evaluator.batchSizes {
		foundBatch = foundBatch || size == len(positions)
	}
	if !foundBatch {
		t.Fatalf("evaluator never received a full leaf batch: %v", evaluator.batchSizes)
	}
}

func TestSerialLockstepAndConcurrentRootSearchAgree(t *testing.T) {
	positions := []Position{
		&toyPosition{spec: variedImmediateChoiceSpec(71, 8, 2)},
		&toyPosition{spec: variedImmediateChoiceSpec(72, 8, 5)},
		&toyPosition{spec: variedImmediateChoiceSpec(73, 8, 7)},
		&toyPosition{spec: variedImmediateChoiceSpec(74, 8, 3)},
	}
	config := SearchConfig{
		Simulations: 32, CPUCT: 1.5,
		DirichletTotalConcentration: 10, DirichletEpsilon: 0.25,
		RootSeeds: []int64{101, 202, 303, 404},
	}
	evaluator := requestSensitiveEvaluator{}
	lockstep, err := NewMCTS(evaluator, config)
	if err != nil {
		t.Fatal(err)
	}
	want, err := lockstep.SearchBatch(context.Background(), positions)
	if err != nil {
		t.Fatal(err)
	}
	options := []BatchingEvaluatorOptions{
		{MaxPositions: 1, MaxWait: 0},
		{MaxPositions: 2, MaxWait: time.Millisecond},
		{MaxPositions: 8, MaxWait: 2 * time.Millisecond},
	}
	for _, option := range options {
		t.Run(fmt.Sprintf("max_%d_wait_%s", option.MaxPositions, option.MaxWait), func(t *testing.T) {
			batcher, err := NewBatchingEvaluator(context.Background(), evaluator, option)
			if err != nil {
				t.Fatal(err)
			}
			got, err := SearchRootsConcurrent(context.Background(), batcher, config, positions)
			batcher.Close()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("concurrent roots differ from lockstep roots\ngot=%#v\nwant=%#v", got, want)
			}

			reorderedPositions := []Position{positions[3], positions[2], positions[1], positions[0]}
			reorderedConfig := config
			reorderedConfig.RootSeeds = []int64{404, 303, 202, 101}
			reorderedBatcher, err := NewBatchingEvaluator(context.Background(), evaluator, option)
			if err != nil {
				t.Fatal(err)
			}
			reordered, err := SearchRootsConcurrent(context.Background(), reorderedBatcher, reorderedConfig, reorderedPositions)
			reorderedBatcher.Close()
			if err != nil {
				t.Fatal(err)
			}
			for index := range reordered {
				if !reflect.DeepEqual(reordered[index], want[len(want)-1-index]) {
					t.Fatalf("root order changed result %d", index)
				}
			}
		})
	}
}

func TestConcurrentRootSearchFillsSharedEvaluatorBatch(t *testing.T) {
	underlying := &recordingEvaluator{}
	batcher, err := NewBatchingEvaluator(context.Background(), underlying, BatchingEvaluatorOptions{
		MaxPositions: 4,
		MaxWait:      100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer batcher.Close()
	positions := []Position{
		&toyPosition{spec: immediateChoiceSpec(81)},
		&toyPosition{spec: immediateChoiceSpec(82)},
		&toyPosition{spec: immediateChoiceSpec(83)},
		&toyPosition{spec: immediateChoiceSpec(84)},
	}
	if _, err := SearchRootsConcurrent(context.Background(), batcher, SearchConfig{Simulations: 1}, positions); err != nil {
		t.Fatal(err)
	}
	if got := underlying.batches(); len(got) == 0 || got[0] != 4 {
		t.Fatalf("concurrent roots did not fill shared batch: %v", got)
	}
}

func TestEvaluationCacheAndSearchDoNotMutateParent(t *testing.T) {
	evaluator := &countingEvaluator{}
	search, err := NewMCTS(evaluator, SearchConfig{Simulations: 16})
	if err != nil {
		t.Fatal(err)
	}
	position := &toyPosition{spec: immediateChoiceSpec(30)}
	before := position.CanonicalHash()
	first, err := search.Search(context.Background(), position)
	if err != nil {
		t.Fatal(err)
	}
	requestsAfterFirst := evaluator.requests
	second, err := search.Search(context.Background(), position)
	if err != nil {
		t.Fatal(err)
	}
	if evaluator.requests != requestsAfterFirst {
		t.Fatalf("cache miss on repeated search: %d -> %d requests", requestsAfterFirst, evaluator.requests)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("cached search changed deterministic result")
	}
	if position.CanonicalHash() != before || position.node != 0 {
		t.Fatal("search mutated its parent position")
	}
}

func TestRootDirichletNoiseIsSeededAndRootOnly(t *testing.T) {
	config := SearchConfig{
		Simulations: 0, CPUCT: 1.5, DirichletTotalConcentration: 10,
		DirichletEpsilon: 0.25, Seed: 77,
	}
	search1, _ := NewMCTS(UniformEvaluator{}, config)
	search2, _ := NewMCTS(UniformEvaluator{}, config)
	first, err := search1.Search(context.Background(), &toyPosition{spec: immediateChoiceSpec(40)})
	if err != nil {
		t.Fatal(err)
	}
	second, err := search2.Search(context.Background(), &toyPosition{spec: immediateChoiceSpec(40)})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Priors, second.Priors) {
		t.Fatalf("same seed produced different root noise: %v vs %v", first.Priors, second.Priors)
	}
	config.Seed++
	search3, _ := NewMCTS(UniformEvaluator{}, config)
	third, err := search3.Search(context.Background(), &toyPosition{spec: immediateChoiceSpec(40)})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(first.Priors, third.Priors) {
		t.Fatal("different seeds produced identical root noise")
	}
}

func TestRootNoiseIsIndependentOfBatchCompositionAndOrder(t *testing.T) {
	positions := []Position{
		&toyPosition{spec: variedImmediateChoiceSpec(41, 8, 2)},
		&toyPosition{spec: variedImmediateChoiceSpec(42, 8, 5)},
	}
	rootSeeds := []int64{1101, 2202}
	base := SearchConfig{
		Simulations: 32, CPUCT: 1.5,
		DirichletTotalConcentration: 10, DirichletEpsilon: 0.25,
	}
	want := make([]SearchResult, len(positions))
	for index := range positions {
		config := base
		config.RootSeeds = []int64{rootSeeds[index]}
		search, err := NewMCTS(UniformEvaluator{}, config)
		if err != nil {
			t.Fatal(err)
		}
		want[index], err = search.Search(context.Background(), positions[index])
		if err != nil {
			t.Fatal(err)
		}
	}

	config := base
	config.RootSeeds = append([]int64(nil), rootSeeds...)
	batched, err := NewMCTS(UniformEvaluator{}, config)
	if err != nil {
		t.Fatal(err)
	}
	got, err := batched.SearchBatch(context.Background(), positions)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("batch composition changed seeded root searches\ngot=%#v\nwant=%#v", got, want)
	}

	reorderedConfig := base
	reorderedConfig.RootSeeds = []int64{rootSeeds[1], rootSeeds[0]}
	reordered, err := NewMCTS(UniformEvaluator{}, reorderedConfig)
	if err != nil {
		t.Fatal(err)
	}
	reorderedResults, err := reordered.SearchBatch(context.Background(), []Position{positions[1], positions[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reorderedResults[0], want[1]) || !reflect.DeepEqual(reorderedResults[1], want[0]) {
		t.Fatal("reordering roots changed their seeded searches")
	}

	badConfig := base
	badConfig.RootSeeds = []int64{1}
	bad, err := NewMCTS(UniformEvaluator{}, badConfig)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bad.SearchBatch(context.Background(), positions); err == nil {
		t.Fatal("mismatched root seed count was accepted")
	}
}

func TestSelfPlayRootSeedSeparatesActorBatchGameAndPly(t *testing.T) {
	first := selfPlayRootSeed(100, 0)
	if first == 0 {
		t.Fatal("search seed mapped to zero")
	}
	if got := selfPlayRootSeed(100, 0); got != first {
		t.Fatalf("root seed is not deterministic: got %d, want %d", got, first)
	}
	cases := []int64{
		selfPlayRootSeed(101, 0),
		selfPlayRootSeed(100, 1),
	}
	for index, got := range cases {
		if got == first {
			t.Fatalf("root seed collision for distinct input case %d", index)
		}
	}
}

func TestLockstepAndConcurrentSelfPlayProduceIdenticalTrajectories(t *testing.T) {
	generate := func(mode RootSearchMode) []Trajectory {
		positions := make([]*GamePosition, 2)
		var err error
		positions[0], err = NewBaseGame(7310, models.FactionNomads, models.FactionWitches)
		if err != nil {
			t.Fatal(err)
		}
		positions[1], err = NewBaseGame(7311, models.FactionEngineers, models.FactionGiants)
		if err != nil {
			t.Fatal(err)
		}
		batcher, err := NewBatchingEvaluator(context.Background(), UniformEvaluator{}, BatchingEvaluatorOptions{
			MaxPositions: 8,
			MaxWait:      time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		trajectories, err := GenerateSelfPlay(context.Background(), positions, []int64{7310, 7311}, batcher, SelfPlayConfig{
			Search: SearchConfig{
				Simulations:                 1,
				DirichletTotalConcentration: 10,
				DirichletEpsilon:            0.25,
			},
			RootSearchMode: mode,
			SearchSeeds:    []int64{9001, 9002},
			Temperature:    1,
			MaxPlies:       500,
			EngineCommit:   "test",
		})
		batcher.Close()
		if err != nil {
			t.Fatal(err)
		}
		return trajectories
	}
	want := generate(RootSearchLockstep)
	got := generate(RootSearchConcurrent)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("concurrent self-play changed same-seed trajectories")
	}
}

func TestRootNoiseScalesWithObservedBranching(t *testing.T) {
	config := SearchConfig{DirichletTotalConcentration: 12, DirichletEpsilon: 0.25}
	for branches, wantAlpha := range map[int]float64{2: 6, 12: 1, 24: 0.5} {
		gotAlpha := dirichletAlpha(config, branches)
		if gotAlpha != wantAlpha {
			t.Fatalf("branches=%d alpha=%v want=%v", branches, gotAlpha, wantAlpha)
		}
	}
}

func TestZeroSimulationGreedySelectionUsesNetworkPrior(t *testing.T) {
	result := SearchResult{
		Actions: []SearchAction{toyAction(1), toyAction(2)},
		Priors:  []float32{0.1, 0.9}, Visits: []int{0, 0},
	}
	action, err := result.SelectAction(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if action.Key() != toyAction(2).Key() {
		t.Fatalf("zero-simulation greedy selection ignored prior: %s", action.Key())
	}
}

func TestVisitPolicyIsFiniteAtTinyPositiveTemperature(t *testing.T) {
	result := SearchResult{
		Actions: []SearchAction{toyAction(1), toyAction(2), toyAction(3)},
		Priors:  []float32{0.2, 0.3, 0.5}, Visits: []int{1, 2, 2},
	}
	policy, err := result.VisitPolicy(1e-300)
	if err != nil {
		t.Fatal(err)
	}
	total := float32(0)
	for _, probability := range policy {
		if math.IsNaN(float64(probability)) || math.IsInf(float64(probability), 0) || probability < 0 || probability > 1 {
			t.Fatalf("non-finite/invalid probability %v in %v", probability, policy)
		}
		total += probability
	}
	if total < 0.999999 || total > 1.000001 {
		t.Fatalf("policy sums to %v: %v", total, policy)
	}
	if policy[0] != 0 || policy[1] != 0.5 || policy[2] != 0.5 {
		t.Fatalf("unexpected tiny-temperature limit %v", policy)
	}
}

func TestMoreSimulationsImprovePairedToyStrength(t *testing.T) {
	lowWins, highWins, improved, regressed := 0, 0, 0, 0
	for seed := int64(0); seed < 64; seed++ {
		setupRNG := rand.New(rand.NewSource(seed))
		branches := 2 + setupRNG.Intn(8)
		winningAction := 1 + setupRNG.Intn(branches)
		spec := variedImmediateChoiceSpec(byte(50+seed), branches, winningAction)
		outcomes := make(map[int]bool, 2)
		for _, simulations := range []int{1, 64} {
			search, err := NewMCTS(UniformEvaluator{}, SearchConfig{Simulations: simulations, Seed: seed})
			if err != nil {
				t.Fatal(err)
			}
			position := &toyPosition{spec: spec}
			result, err := search.Search(context.Background(), position)
			if err != nil {
				t.Fatal(err)
			}
			action, err := result.SelectAction(0, rand.New(rand.NewSource(seed)))
			if err != nil {
				t.Fatal(err)
			}
			if err := position.Apply(action); err != nil {
				t.Fatal(err)
			}
			if position.Outcome("A") > 0 {
				outcomes[simulations] = true
			}
		}
		if outcomes[1] {
			lowWins++
		}
		if outcomes[64] {
			highWins++
		}
		if !outcomes[1] && outcomes[64] {
			improved++
		}
		if outcomes[1] && !outcomes[64] {
			regressed++
		}
	}
	// With no discordant regressions, the exact paired sign-test probability is
	// 2^-improved; five or more improvements is significant at p < 0.05.
	if highWins < lowWins || improved < 5 || regressed != 0 {
		t.Fatalf("simulation scaling failed paired sign test: low=%d/64 high=%d/64 improved=%d regressed=%d", lowWins, highWins, improved, regressed)
	}
}

func TestPUCTRunsOnConstructedTerraMysticaReactionState(t *testing.T) {
	state := forcedActionPosition(t, 900, models.FactionWitches, models.FactionEngineers).StateClone()
	state.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{
		Amount: 2, VPCost: 1, FromPlayerID: "p0", EventID: 1,
	}}
	position, err := NewPosition(state)
	if err != nil {
		t.Fatal(err)
	}
	before := position.CanonicalHash()
	search, err := NewMCTS(UniformEvaluator{}, SearchConfig{Simulations: 4, CPUCT: 1.5})
	if err != nil {
		t.Fatal(err)
	}
	result, err := search.Search(context.Background(), position)
	if err != nil {
		t.Fatal(err)
	}
	if result.DecisionPlayer != "p1" {
		t.Fatalf("root owner = %q, want leech responder p1", result.DecisionPlayer)
	}
	totalVisits := 0
	for _, visits := range result.Visits {
		totalVisits += visits
	}
	if totalVisits != 4 {
		t.Fatalf("root visits = %d, want 4", totalVisits)
	}
	action, err := result.SelectAction(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	next := position.Clone()
	if err := next.Apply(action); err != nil {
		t.Fatalf("search returned invalid TM action %s: %v", action.Key(), err)
	}
	if position.CanonicalHash() != before {
		t.Fatal("TM search mutated its parent position")
	}
}

type constantValueEvaluator struct {
	value        float32
	preferredKey string
}

func (e constantValueEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	results := make([]EvaluationResult, len(requests))
	for i, request := range requests {
		results[i] = EvaluationResult{PolicyLogits: make([]float32, len(request.Actions)), Value: e.value}
		for j, action := range request.Actions {
			if action.Key() == e.preferredKey {
				results[i].PolicyLogits[j] = 10
			}
		}
	}
	return results, nil
}

func TestBackupUsesDecisionOwnerIdentityOnTerraMysticaStates(t *testing.T) {
	const leafValue = float32(0.625)

	t.Run("free_conversion_keeps_perspective", func(t *testing.T) {
		state := forcedActionPosition(t, 901, models.FactionChaosMagicians, models.FactionEngineers).StateClone()
		player := state.GetPlayer("p0")
		player.Resources.Coins = 0
		player.Resources.Workers = 0
		player.Resources.Priests = 0
		player.Resources.Power.Bowl1 = 0
		player.Resources.Power.Bowl2 = 0
		player.Resources.Power.Bowl3 = 2
		state.PendingChaosMagiciansDoubleTurn = &game.PendingChaosMagiciansDoubleTurn{
			PlayerID: "p0", ActionsRemaining: 1,
		}
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		actions := position.LegalActions()
		if len(actions) == 0 || actions[0].Kind != game.ActionConversion {
			t.Fatalf("constructed root does not start with a free conversion: %#v", actions)
		}
		child := position.Clone()
		if err := child.Apply(actions[0]); err != nil {
			t.Fatal(err)
		}
		if child.DecisionPlayer() != "p0" {
			t.Fatalf("conversion child owner = %q, want p0", child.DecisionPlayer())
		}
		search, err := NewMCTS(constantValueEvaluator{value: leafValue, preferredKey: actions[0].Key()}, SearchConfig{Simulations: 1})
		if err != nil {
			t.Fatal(err)
		}
		result, err := search.Search(context.Background(), position)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.QValues) == 0 || result.QValues[0] != leafValue {
			t.Fatalf("same-owner conversion Q=%v, want +%v", result.QValues, leafValue)
		}
	})

	t.Run("leech_response_changes_perspective", func(t *testing.T) {
		state := forcedActionPosition(t, 902, models.FactionWitches, models.FactionEngineers).StateClone()
		// The builder retains its turn while the opponent reacts. Once leech
		// resolves, p0 owns optional post-action choices before FinishTurn.
		state.CurrentPlayerIndex = 0
		state.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{
			Amount: 1, VPCost: 0, FromPlayerID: "p0", EventID: 1,
		}}
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		actions := position.LegalActions()
		if len(actions) == 0 || actions[0].Kind != game.ActionAcceptPowerLeech {
			t.Fatalf("constructed root does not start with a leech response: %#v", actions)
		}
		child := position.Clone()
		if err := child.Apply(actions[0]); err != nil {
			t.Fatal(err)
		}
		if child.DecisionPlayer() != "p0" {
			t.Fatalf("leech child owner = %q, want p0", child.DecisionPlayer())
		}
		search, err := NewMCTS(constantValueEvaluator{value: leafValue, preferredKey: actions[0].Key()}, SearchConfig{Simulations: 1})
		if err != nil {
			t.Fatal(err)
		}
		result, err := search.Search(context.Background(), position)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.QValues) == 0 || result.QValues[0] != -leafValue {
			t.Fatalf("opponent-owner leech Q=%v, want -%v", result.QValues, leafValue)
		}
	})
}
