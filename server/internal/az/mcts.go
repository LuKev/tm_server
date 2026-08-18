package az

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/game"
)

// EvaluationRequest is one position and its complete, stable legal-action
// list. Evaluators return values from Position.DecisionPlayer's perspective.
type EvaluationRequest struct {
	Position Position
	Actions  []SearchAction
}

// EvaluationResult contains unnormalized policy logits and a scalar value in
// [-1, 1] from the request position's decision-owner perspective.
type EvaluationResult struct {
	PolicyLogits []float32 `json:"policy_logits"`
	Value        float32   `json:"value"`
}

// Evaluator is batch-shaped from the start even though the first search is
// serial. Milestone 3 can fill larger cross-game batches without changing the
// search/evaluator contract.
type Evaluator interface {
	Evaluate(context.Context, []EvaluationRequest) ([]EvaluationResult, error)
}

// UniformEvaluator is the correctness baseline: uniform priors and zero value.
type UniformEvaluator struct{}

func (UniformEvaluator) ModelID() string { return "uniform" }

func (UniformEvaluator) Evaluate(_ context.Context, requests []EvaluationRequest) ([]EvaluationResult, error) {
	results := make([]EvaluationResult, len(requests))
	for i, request := range requests {
		results[i].PolicyLogits = make([]float32, len(request.Actions))
	}
	return results, nil
}

type SearchConfig struct {
	Simulations                 int
	CPUCT                       float64
	DirichletTotalConcentration float64
	DirichletEpsilon            float64
	Seed                        int64
	RootSeeds                   []int64
	Telemetry                   *SearchTelemetry
}

// SearchStats describes work performed by MCTS independently of the evaluator
// implementation. Counts are additive across searches and safe to publish as
// operational metrics; durations are wall-clock observations, not benchmarks.
type SearchStats struct {
	SearchCalls              int64                           `json:"search_calls"`
	RootPositions            int64                           `json:"root_positions"`
	SimulationTraversals     int64                           `json:"simulation_traversals"`
	ExpandedNodes            int64                           `json:"expanded_nodes"`
	EvaluationBatches        int64                           `json:"evaluation_batches"`
	EvaluationPositions      int64                           `json:"evaluation_positions"`
	EvaluationCacheHits      int64                           `json:"evaluation_cache_hits"`
	EvaluationCacheMisses    int64                           `json:"evaluation_cache_misses"`
	EvaluationBatchDedupHits int64                           `json:"evaluation_batch_dedup_hits"`
	ByPhaseFaction           map[string]SearchDimensionStats `json:"by_phase_faction,omitempty"`
	WallDuration             time.Duration                   `json:"-"`
}

type SearchDimensionStats struct {
	RootPositions            int64 `json:"root_positions"`
	SimulationTraversals     int64 `json:"simulation_traversals"`
	EvaluationPositions      int64 `json:"evaluation_positions"`
	EvaluationCacheHits      int64 `json:"evaluation_cache_hits"`
	EvaluationCacheMisses    int64 `json:"evaluation_cache_misses"`
	EvaluationBatchDedupHits int64 `json:"evaluation_batch_dedup_hits"`
}

func (s *SearchStats) add(other SearchStats) {
	s.SearchCalls += other.SearchCalls
	s.RootPositions += other.RootPositions
	s.SimulationTraversals += other.SimulationTraversals
	s.ExpandedNodes += other.ExpandedNodes
	s.EvaluationBatches += other.EvaluationBatches
	s.EvaluationPositions += other.EvaluationPositions
	s.EvaluationCacheHits += other.EvaluationCacheHits
	s.EvaluationCacheMisses += other.EvaluationCacheMisses
	s.EvaluationBatchDedupHits += other.EvaluationBatchDedupHits
	if len(other.ByPhaseFaction) > 0 && s.ByPhaseFaction == nil {
		s.ByPhaseFaction = make(map[string]SearchDimensionStats, len(other.ByPhaseFaction))
	}
	for dimension, item := range other.ByPhaseFaction {
		current := s.ByPhaseFaction[dimension]
		current.RootPositions += item.RootPositions
		current.SimulationTraversals += item.SimulationTraversals
		current.EvaluationPositions += item.EvaluationPositions
		current.EvaluationCacheHits += item.EvaluationCacheHits
		current.EvaluationCacheMisses += item.EvaluationCacheMisses
		current.EvaluationBatchDedupHits += item.EvaluationBatchDedupHits
		s.ByPhaseFaction[dimension] = current
	}
	s.WallDuration += other.WallDuration
}

// SearchTelemetry aggregates SearchStats across the independently-created
// searches used by self-play. A nil telemetry pointer leaves the hot path with
// only local counter increments.
type SearchTelemetry struct {
	mu    sync.Mutex
	stats SearchStats
}

func (t *SearchTelemetry) add(stats SearchStats) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.stats.add(stats)
	t.mu.Unlock()
}

func (t *SearchTelemetry) Snapshot() SearchStats {
	if t == nil {
		return SearchStats{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	out := t.stats
	if len(t.stats.ByPhaseFaction) > 0 {
		out.ByPhaseFaction = make(map[string]SearchDimensionStats, len(t.stats.ByPhaseFaction))
		for key, value := range t.stats.ByPhaseFaction {
			out.ByPhaseFaction[key] = value
		}
	}
	return out
}

func (c SearchConfig) normalized() (SearchConfig, error) {
	if c.Simulations < 0 {
		return c, fmt.Errorf("simulations must be non-negative")
	}
	if c.CPUCT == 0 {
		c.CPUCT = 1.5
	}
	if c.CPUCT < 0 || math.IsNaN(c.CPUCT) || math.IsInf(c.CPUCT, 0) {
		return c, fmt.Errorf("c_puct must be finite and positive")
	}
	if c.DirichletEpsilon < 0 || c.DirichletEpsilon > 1 || math.IsNaN(c.DirichletEpsilon) {
		return c, fmt.Errorf("Dirichlet epsilon must be in [0, 1]")
	}
	if c.DirichletEpsilon > 0 && (c.DirichletTotalConcentration <= 0 || math.IsNaN(c.DirichletTotalConcentration) || math.IsInf(c.DirichletTotalConcentration, 0)) {
		return c, fmt.Errorf("Dirichlet total concentration must be finite and positive when noise is enabled")
	}
	return c, nil
}

type MCTS struct {
	evaluator Evaluator
	config    SearchConfig
	cache     map[Hash128]EvaluationResult
	stats     SearchStats
}

func NewMCTS(evaluator Evaluator, config SearchConfig) (*MCTS, error) {
	if evaluator == nil {
		return nil, fmt.Errorf("evaluator is nil")
	}
	normalized, err := config.normalized()
	if err != nil {
		return nil, err
	}
	return &MCTS{evaluator: evaluator, config: normalized, cache: make(map[Hash128]EvaluationResult)}, nil
}

type SearchResult struct {
	RootHash       Hash128
	DecisionPlayer PlayerID
	Actions        []SearchAction
	Priors         []float32
	Visits         []int
	QValues        []float32
	RootValue      float32
}

// VisitPolicy returns the AlphaZero policy target. Temperature zero is a
// deterministic one-hot argmax for evaluation; positive temperatures use
// N^(1/tau), falling back to priors before any edge has been visited.
func (r SearchResult) VisitPolicy(temperature float64) ([]float32, error) {
	if temperature < 0 || math.IsNaN(temperature) || math.IsInf(temperature, 0) {
		return nil, fmt.Errorf("temperature must be finite and non-negative")
	}
	if len(r.Actions) != len(r.Visits) || len(r.Actions) != len(r.Priors) {
		return nil, fmt.Errorf("malformed search result")
	}
	policy := make([]float32, len(r.Actions))
	if len(policy) == 0 {
		return policy, nil
	}
	if temperature == 0 {
		best := 0
		totalVisits := r.Visits[0]
		for _, visits := range r.Visits[1:] {
			totalVisits += visits
		}
		for i := 1; i < len(r.Visits); i++ {
			if r.Visits[i] > r.Visits[best] || (totalVisits == 0 && r.Priors[i] > r.Priors[best]) {
				best = i
			}
		}
		policy[best] = 1
		return policy, nil
	}
	weights := make([]float64, len(policy))
	total := 0.0
	maxVisits := 0
	for _, visits := range r.Visits {
		if visits > maxVisits {
			maxVisits = visits
		}
	}
	if maxVisits > 0 {
		for i, visits := range r.Visits {
			if visits == 0 {
				continue
			}
			// Normalize by the maximum before exponentiating. The log ratio is
			// never positive, so arbitrarily small positive temperatures cannot
			// overflow policy targets.
			weights[i] = math.Exp(math.Log(float64(visits)/float64(maxVisits)) / temperature)
			total += weights[i]
		}
	}
	if total == 0 {
		for i, prior := range r.Priors {
			weights[i] = float64(prior)
			total += weights[i]
		}
	}
	if total == 0 {
		total = float64(len(policy))
		for i := range weights {
			weights[i] = 1
		}
	}
	for i := range policy {
		policy[i] = float32(weights[i] / total)
	}
	return policy, nil
}

func (r SearchResult) SelectAction(temperature float64, rng *rand.Rand) (SearchAction, error) {
	policy, err := r.VisitPolicy(temperature)
	if err != nil {
		return SearchAction{}, err
	}
	if len(policy) == 0 {
		return SearchAction{}, fmt.Errorf("search result has no legal actions")
	}
	if temperature == 0 {
		for i, probability := range policy {
			if probability == 1 {
				return r.Actions[i], nil
			}
		}
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(0))
	}
	draw := rng.Float64()
	cumulative := 0.0
	for i, probability := range policy {
		cumulative += float64(probability)
		if draw < cumulative || i == len(policy)-1 {
			return r.Actions[i], nil
		}
	}
	return SearchAction{}, fmt.Errorf("failed to sample search policy")
}

type searchNode struct {
	position Position
	player   PlayerID
	expanded bool
	value    float32
	edges    []*searchEdge
}

type searchEdge struct {
	action       SearchAction
	parentPlayer PlayerID
	prior        float64
	visits       int
	valueSum     float64
	child        *searchNode
}

func (e *searchEdge) q() float64 {
	if e.visits == 0 {
		return 0
	}
	return e.valueSum / float64(e.visits)
}

func (m *MCTS) Search(ctx context.Context, position Position) (SearchResult, error) {
	results, err := m.SearchBatch(ctx, []Position{position})
	if err != nil {
		return SearchResult{}, err
	}
	return results[0], nil
}

// SearchRootsConcurrent runs one independent MCTS per root. Concurrent leaf
// evaluations can be coalesced by a shared BatchingEvaluator, while each root
// retains exactly the serial tree, cache, seed, and backup semantics.
func SearchRootsConcurrent(ctx context.Context, evaluator Evaluator, config SearchConfig, positions []Position) ([]SearchResult, error) {
	if len(positions) == 0 {
		return nil, nil
	}
	if len(config.RootSeeds) != 0 && len(config.RootSeeds) != len(positions) {
		return nil, fmt.Errorf("got %d root seeds for %d positions", len(config.RootSeeds), len(positions))
	}
	results := make([]SearchResult, len(positions))
	errors := make([]error, len(positions))
	var wait sync.WaitGroup
	wait.Add(len(positions))
	for index, position := range positions {
		go func(index int, position Position) {
			defer wait.Done()
			rootConfig := config
			if len(config.RootSeeds) != 0 {
				rootConfig.RootSeeds = []int64{config.RootSeeds[index]}
			}
			search, err := NewMCTS(evaluator, rootConfig)
			if err != nil {
				errors[index] = err
				return
			}
			results[index], errors[index] = search.Search(ctx, position)
		}(index, position)
	}
	wait.Wait()
	for index, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("search root %d: %w", index, err)
		}
	}
	return results, nil
}

// SearchBatch runs roots in lockstep and evaluates one selected leaf per
// non-terminal root in each batch. With a deterministic evaluator and no root
// noise, each root has the same trace as an independent serial Search.
func (m *MCTS) SearchBatch(ctx context.Context, positions []Position) ([]SearchResult, error) {
	if len(positions) == 0 {
		return nil, nil
	}
	if len(m.config.RootSeeds) != 0 && len(m.config.RootSeeds) != len(positions) {
		return nil, fmt.Errorf("got %d root seeds for %d positions", len(m.config.RootSeeds), len(positions))
	}
	m.stats = SearchStats{SearchCalls: 1, RootPositions: int64(len(positions))}
	collectDimensions := m.config.Telemetry != nil
	var rootDimensions []string
	if collectDimensions {
		m.stats.ByPhaseFaction = make(map[string]SearchDimensionStats)
		rootDimensions = make([]string, len(positions))
	}
	started := time.Now()
	defer func() {
		m.stats.WallDuration = time.Since(started)
		m.config.Telemetry.add(m.stats)
	}()
	roots := make([]*searchNode, len(positions))
	for i, position := range positions {
		if position == nil {
			return nil, fmt.Errorf("position %d is nil", i)
		}
		roots[i] = &searchNode{position: position.Clone()}
		if collectDimensions {
			rootDimensions[i] = searchMetricDimension(position)
			item := m.stats.ByPhaseFaction[rootDimensions[i]]
			item.RootPositions++
			m.stats.ByPhaseFaction[rootDimensions[i]] = item
		}
	}
	if err := m.expandNodes(ctx, roots); err != nil {
		return nil, err
	}
	for index, root := range roots {
		seed := m.config.Seed
		if len(m.config.RootSeeds) != 0 {
			seed = m.config.RootSeeds[index]
		}
		seed = mixSearchSeed(seed, root.position.CanonicalHash())
		rng := rand.New(rand.NewSource(seed))
		if err := addDirichletNoise(root, m.config, rng); err != nil {
			return nil, err
		}
	}

	for simulation := 0; simulation < m.config.Simulations; simulation++ {
		paths := make([][]*searchEdge, len(roots))
		leaves := make([]*searchNode, len(roots))
		toExpand := make([]*searchNode, 0, len(roots))
		for i, root := range roots {
			if root.position.IsTerminal() {
				continue
			}
			m.stats.SimulationTraversals++
			if collectDimensions {
				item := m.stats.ByPhaseFaction[rootDimensions[i]]
				item.SimulationTraversals++
				m.stats.ByPhaseFaction[rootDimensions[i]] = item
			}
			leaf, path, err := m.selectLeaf(root)
			if err != nil {
				return nil, err
			}
			leaves[i], paths[i] = leaf, path
			if !leaf.position.IsTerminal() && !leaf.expanded {
				toExpand = append(toExpand, leaf)
			}
		}
		if err := m.expandNodes(ctx, toExpand); err != nil {
			return nil, err
		}
		for i, leaf := range leaves {
			if leaf == nil {
				continue
			}
			backupPath(paths[i], leaf)
		}
	}

	results := make([]SearchResult, len(roots))
	for i, root := range roots {
		results[i] = resultFromRoot(root)
	}
	return results, nil
}

func mixSearchSeed(seed int64, hash Hash128) int64 {
	value := uint64(seed) ^ binary.LittleEndian.Uint64(hash[:8])
	value ^= binary.LittleEndian.Uint64(hash[8:]) + 0x9e3779b97f4a7c15
	value ^= value >> 30
	value *= 0xbf58476d1ce4e5b9
	value ^= value >> 27
	value *= 0x94d049bb133111eb
	value ^= value >> 31
	return int64(value)
}

func (m *MCTS) selectLeaf(root *searchNode) (*searchNode, []*searchEdge, error) {
	node := root
	path := make([]*searchEdge, 0, 16)
	for node.expanded && !node.position.IsTerminal() {
		edge := selectPUCTEdge(node, m.config.CPUCT)
		if edge == nil {
			return nil, nil, fmt.Errorf("expanded non-terminal node has no edges")
		}
		path = append(path, edge)
		if edge.child == nil {
			next := node.position.Clone()
			if err := next.Apply(edge.action); err != nil {
				return nil, nil, fmt.Errorf("search applied emitted action %s: %w", edge.action.Key(), err)
			}
			edge.child = &searchNode{position: next}
		}
		node = edge.child
	}
	return node, path, nil
}

func selectPUCTEdge(node *searchNode, cPUCT float64) *searchEdge {
	totalVisits := 0
	for _, edge := range node.edges {
		totalVisits += edge.visits
	}
	sqrtTotal := math.Sqrt(float64(totalVisits))
	var best *searchEdge
	bestScore := math.Inf(-1)
	for _, edge := range node.edges {
		score := edge.q() + cPUCT*edge.prior*sqrtTotal/float64(1+edge.visits)
		if score > bestScore {
			best, bestScore = edge, score
		}
	}
	return best
}

func (m *MCTS) expandNodes(ctx context.Context, nodes []*searchNode) error {
	type miss struct {
		key     Hash128
		actions []SearchAction
		nodes   []*searchNode
	}
	misses := make([]miss, 0, len(nodes))
	missIndex := make(map[Hash128]int, len(nodes))
	for _, node := range nodes {
		if node == nil || node.expanded {
			continue
		}
		if node.position.IsTerminal() {
			node.expanded = true
			m.stats.ExpandedNodes++
			continue
		}
		node.player = node.position.DecisionPlayer()
		if node.player == "" {
			return fmt.Errorf("non-terminal position has no unique decision player")
		}
		actions := append([]SearchAction(nil), node.position.LegalActions()...)
		sort.Slice(actions, func(i, j int) bool { return actions[i].Key() < actions[j].Key() })
		if len(actions) == 0 {
			return fmt.Errorf("non-terminal position %s has no legal actions", node.position.CanonicalHash())
		}
		key := node.position.CanonicalHash()
		if cached, ok := m.cache[key]; ok {
			if err := expandNode(node, actions, cached); err != nil {
				return err
			}
			m.stats.ExpandedNodes++
			m.stats.EvaluationCacheHits++
			if m.stats.ByPhaseFaction != nil {
				dimension := searchMetricDimension(node.position)
				item := m.stats.ByPhaseFaction[dimension]
				item.EvaluationCacheHits++
				m.stats.ByPhaseFaction[dimension] = item
			}
			continue
		}
		if index, ok := missIndex[key]; ok {
			misses[index].nodes = append(misses[index].nodes, node)
			m.stats.EvaluationBatchDedupHits++
			if m.stats.ByPhaseFaction != nil {
				dimension := searchMetricDimension(node.position)
				item := m.stats.ByPhaseFaction[dimension]
				item.EvaluationBatchDedupHits++
				m.stats.ByPhaseFaction[dimension] = item
			}
			continue
		}
		m.stats.EvaluationCacheMisses++
		if m.stats.ByPhaseFaction != nil {
			dimension := searchMetricDimension(node.position)
			item := m.stats.ByPhaseFaction[dimension]
			item.EvaluationCacheMisses++
			item.EvaluationPositions++
			m.stats.ByPhaseFaction[dimension] = item
		}
		missIndex[key] = len(misses)
		misses = append(misses, miss{key: key, actions: actions, nodes: []*searchNode{node}})
	}
	if len(misses) == 0 {
		return nil
	}
	m.stats.EvaluationBatches++
	m.stats.EvaluationPositions += int64(len(misses))
	requests := make([]EvaluationRequest, len(misses))
	for i, item := range misses {
		requests[i] = EvaluationRequest{Position: item.nodes[0].position.Clone(), Actions: append([]SearchAction(nil), item.actions...)}
	}
	results, err := m.evaluator.Evaluate(ctx, requests)
	if err != nil {
		return fmt.Errorf("evaluate leaves: %w", err)
	}
	if len(results) != len(requests) {
		return fmt.Errorf("evaluator returned %d results for %d requests", len(results), len(requests))
	}
	for i, result := range results {
		if err := validateEvaluation(result, len(misses[i].actions)); err != nil {
			return fmt.Errorf("evaluation %d: %w", i, err)
		}
		result = cloneEvaluation(result)
		m.cache[misses[i].key] = result
		for _, node := range misses[i].nodes {
			if err := expandNode(node, misses[i].actions, result); err != nil {
				return err
			}
			m.stats.ExpandedNodes++
		}
	}
	return nil
}

func searchMetricDimension(position Position) string {
	gamePosition, ok := position.(*GamePosition)
	if !ok || gamePosition == nil || gamePosition.state == nil {
		return "synthetic"
	}
	state := gamePosition.state
	phase := map[game.GamePhase]string{
		game.PhaseSetup: "setup", game.PhaseFactionSelection: "faction_selection",
		game.PhaseIncome: "income", game.PhaseAction: "action",
		game.PhaseCleanup: "cleanup", game.PhaseEnd: "end",
	}[state.Phase]
	if phase == "" {
		phase = fmt.Sprintf("unknown_%d", state.Phase)
	}
	faction := "unassigned"
	if player := state.GetPlayer(string(gamePosition.DecisionPlayer())); player != nil && player.Faction != nil {
		faction = player.Faction.GetType().String()
	}
	return fmt.Sprintf("phase=%s/round=%d/faction=%s", phase, state.Round, faction)
}

func validateEvaluation(result EvaluationResult, actionCount int) error {
	if len(result.PolicyLogits) != actionCount {
		return fmt.Errorf("got %d policy logits for %d legal actions", len(result.PolicyLogits), actionCount)
	}
	if math.IsNaN(float64(result.Value)) || math.IsInf(float64(result.Value), 0) || result.Value < -1 || result.Value > 1 {
		return fmt.Errorf("value %v is outside [-1, 1]", result.Value)
	}
	for _, logit := range result.PolicyLogits {
		if math.IsNaN(float64(logit)) || math.IsInf(float64(logit), 0) {
			return fmt.Errorf("policy contains non-finite logit")
		}
	}
	return nil
}

func expandNode(node *searchNode, actions []SearchAction, evaluation EvaluationResult) error {
	if err := validateEvaluation(evaluation, len(actions)); err != nil {
		return err
	}
	priors := softmax(evaluation.PolicyLogits)
	node.value = evaluation.Value
	node.edges = make([]*searchEdge, len(actions))
	for i, action := range actions {
		node.edges[i] = &searchEdge{action: action, parentPlayer: node.player, prior: priors[i]}
	}
	node.expanded = true
	return nil
}

func softmax(logits []float32) []float64 {
	priors := make([]float64, len(logits))
	if len(logits) == 0 {
		return priors
	}
	maximum := float64(logits[0])
	for _, logit := range logits[1:] {
		if float64(logit) > maximum {
			maximum = float64(logit)
		}
	}
	total := 0.0
	for i, logit := range logits {
		priors[i] = math.Exp(float64(logit) - maximum)
		total += priors[i]
	}
	for i := range priors {
		priors[i] /= total
	}
	return priors
}

func backupPath(path []*searchEdge, leaf *searchNode) {
	for _, edge := range path {
		value := float32(0)
		if leaf.position.IsTerminal() {
			value = leaf.position.Outcome(edge.parentPlayer)
		} else if edge.parentPlayer == leaf.player {
			value = leaf.value
		} else {
			value = -leaf.value
		}
		edge.visits++
		edge.valueSum += float64(value)
	}
}

func resultFromRoot(root *searchNode) SearchResult {
	result := SearchResult{
		RootHash: root.position.CanonicalHash(), DecisionPlayer: root.player,
		RootValue: root.value, Actions: make([]SearchAction, len(root.edges)),
		Priors: make([]float32, len(root.edges)), Visits: make([]int, len(root.edges)),
		QValues: make([]float32, len(root.edges)),
	}
	for i, edge := range root.edges {
		result.Actions[i] = edge.action
		result.Priors[i] = float32(edge.prior)
		result.Visits[i] = edge.visits
		result.QValues[i] = float32(edge.q())
	}
	return result
}

func cloneEvaluation(result EvaluationResult) EvaluationResult {
	return EvaluationResult{PolicyLogits: append([]float32(nil), result.PolicyLogits...), Value: result.Value}
}

func addDirichletNoise(root *searchNode, config SearchConfig, rng *rand.Rand) error {
	if config.DirichletEpsilon == 0 || len(root.edges) == 0 {
		return nil
	}
	noise := make([]float64, len(root.edges))
	alpha := dirichletAlpha(config, len(root.edges))
	total := 0.0
	for i := range noise {
		noise[i] = sampleGamma(rng, alpha)
		total += noise[i]
	}
	if total == 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		return fmt.Errorf("failed to sample Dirichlet noise")
	}
	for i, edge := range root.edges {
		edge.prior = (1-config.DirichletEpsilon)*edge.prior + config.DirichletEpsilon*noise[i]/total
	}
	return nil
}

func dirichletAlpha(config SearchConfig, branching int) float64 {
	if branching <= 0 {
		return 0
	}
	return config.DirichletTotalConcentration / float64(branching)
}

// sampleGamma uses Marsaglia and Tsang's method, with the standard shape<1
// transformation. Dirichlet samples need only a unit scale.
func sampleGamma(rng *rand.Rand, shape float64) float64 {
	if shape < 1 {
		return sampleGamma(rng, shape+1) * math.Pow(rng.Float64(), 1/shape)
	}
	d := shape - 1.0/3.0
	c := 1 / math.Sqrt(9*d)
	for {
		x := rng.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1-0.0331*x*x*x*x || math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v
		}
	}
}
