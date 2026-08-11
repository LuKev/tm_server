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
	first := selfPlayRootSeed(100, 100, 0)
	if first == 0 {
		t.Fatal("matching actor and game seed cancelled to zero")
	}
	if got := selfPlayRootSeed(100, 100, 0); got != first {
		t.Fatalf("root seed is not deterministic: got %d, want %d", got, first)
	}
	cases := []int64{
		selfPlayRootSeed(101, 101, 0),
		selfPlayRootSeed(100, 101, 0),
		selfPlayRootSeed(100, 100, 1),
	}
	for index, got := range cases {
		if got == first {
			t.Fatalf("root seed collision for distinct input case %d", index)
		}
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
	value float32
}

func (e constantValueEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	results := make([]EvaluationResult, len(requests))
	for i, request := range requests {
		results[i] = EvaluationResult{PolicyLogits: make([]float32, len(request.Actions)), Value: e.value}
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
		search, err := NewMCTS(constantValueEvaluator{value: leafValue}, SearchConfig{Simulations: 1})
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
		state.CurrentPlayerIndex = 1
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
		search, err := NewMCTS(constantValueEvaluator{value: leafValue}, SearchConfig{Simulations: 1})
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
