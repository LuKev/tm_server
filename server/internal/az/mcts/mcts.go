package mcts

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/model"
)

type Config struct {
	Simulations     int     `json:"simulations"`
	BatchSize       int     `json:"batchSize"`
	CPUCT           float64 `json:"cpuct"`
	Temperature     float64 `json:"temperature"`
	DirichletAlpha  float64 `json:"dirichletAlpha"`
	DirichletWeight float64 `json:"dirichletWeight"`
	MaxDepth        int     `json:"maxDepth"`
	RandomSeed      int64   `json:"randomSeed"`
}

type Result struct {
	RootPlayerID string         `json:"rootPlayerId"`
	Selected     RankedAction   `json:"selected"`
	Actions      []RankedAction `json:"actions"`
	Simulations  int            `json:"simulations"`
}

type RankedAction struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`
	Label  string            `json:"label"`
	Player string            `json:"playerId"`
	Visits int               `json:"visits"`
	Prior  float64           `json:"prior"`
	Q      float64           `json:"q"`
	Prob   float64           `json:"prob"`
	Meta   map[string]string `json:"meta,omitempty"`
	Params map[string]any    `json:"params,omitempty"`
}

type node struct {
	position *env.Position
	parent   *node
	playerID string
	prior    float64
	visits   int
	valueSum float64
	action   actions.Option
	children []*node
	expanded bool
	noisy    bool
}

// Tree keeps the selected subtree between real moves. It is intended for
// self-play where the same evaluator controls both players.
type Tree struct {
	root *node
}

func NewTree(position *env.Position) *Tree {
	return &Tree{root: newRootNode(position)}
}

func (t *Tree) Search(evaluator model.Evaluator, config Config) Result {
	if evaluator == nil {
		evaluator = model.NewHeuristicEvaluator()
	}
	if t == nil || t.root == nil || t.root.position == nil {
		return Search(nil, evaluator, config)
	}
	prepareConfig(&config)
	rng := rand.New(rand.NewSource(config.RandomSeed))
	rootPlayer := rootPlayerID(t.root.position)
	if !t.root.expanded {
		expand(t.root, evaluator, rootPlayer)
	}
	if !t.root.noisy {
		applyRootNoise(t.root, config, rng)
		t.root.noisy = true
	}
	ranked := runSearch(t.root, evaluator, rootPlayer, config)
	return searchResult(rootPlayer, ranked, config.Simulations)
}

func (t *Tree) Advance(_ string, position *env.Position) {
	if t == nil {
		return
	}
	// Reusing expanded child subtrees can surface stale pending-decision actions
	// when live-engine resolution has equivalent-looking but not identical
	// state. Keep the API safe by advancing to a fresh root until state
	// identity/revision checks are available for subtree reuse.
	t.root = newRootNode(position)
}

func Search(position *env.Position, evaluator model.Evaluator, config Config) Result {
	if evaluator == nil {
		evaluator = model.NewHeuristicEvaluator()
	}
	prepareConfig(&config)
	rng := rand.New(rand.NewSource(config.RandomSeed))
	rootPlayer := rootPlayerID(position)
	root := newRootNode(position)
	expand(root, evaluator, rootPlayer)
	applyRootNoise(root, config, rng)
	ranked := runSearch(root, evaluator, rootPlayer, config)
	return searchResult(rootPlayer, ranked, config.Simulations)
}

func prepareConfig(config *Config) {
	if config.Simulations < 0 {
		config.Simulations = 64
	}
	if config.CPUCT <= 0 {
		config.CPUCT = 1.5
	}
	if config.Temperature < 0 {
		config.Temperature = 0
	}
	if config.MaxDepth <= 0 {
		config.MaxDepth = 200
	}
	if config.RandomSeed == 0 {
		config.RandomSeed = time.Now().UnixNano()
	}
}

func newRootNode(position *env.Position) *node {
	playerID := ""
	if position != nil {
		playerID = position.CurrentPlayerID()
	}
	return &node{position: position, playerID: playerID, prior: 1}
}

func rootPlayerID(position *env.Position) string {
	if position == nil {
		return ""
	}
	rootPlayer := position.RootPlayerID
	if rootPlayer == "" {
		rootPlayer = position.CurrentPlayerID()
	}
	return rootPlayer
}

func runSearch(root *node, evaluator model.Evaluator, rootPlayer string, config Config) []RankedAction {
	var ranked []RankedAction
	if config.Simulations == 0 {
		ranked = rankedPolicyChildren(root)
	} else if batchEvaluator, ok := evaluator.(model.BatchEvaluator); ok && config.BatchSize > 1 {
		for i := 0; i < config.Simulations; i += config.BatchSize {
			limit := config.BatchSize
			if remaining := config.Simulations - i; remaining < limit {
				limit = remaining
			}
			runBatchSimulations(root, batchEvaluator, rootPlayer, config, limit)
		}
	} else {
		for i := 0; i < config.Simulations; i++ {
			runSimulation(root, evaluator, rootPlayer, config, 0)
		}
	}
	if ranked == nil {
		ranked = rankedChildren(root, config.Temperature)
	}
	return ranked
}

func searchResult(rootPlayer string, ranked []RankedAction, simulations int) Result {
	result := Result{
		RootPlayerID: rootPlayer,
		Actions:      ranked,
		Simulations:  simulations,
	}
	if len(ranked) > 0 {
		result.Selected = ranked[0]
	}
	return result
}

func runSimulation(n *node, evaluator model.Evaluator, rootPlayer string, config Config, depth int) float64 {
	if n == nil || n.position == nil {
		return 0
	}
	if depth >= config.MaxDepth || n.position.IsTerminal() {
		value := n.position.ValueFor(rootPlayer)
		n.visits++
		n.valueSum += value
		return value
	}
	if !n.expanded {
		value := expand(n, evaluator, rootPlayer)
		n.visits++
		n.valueSum += value
		return value
	}
	child := selectChild(n, rootPlayer, config.CPUCT)
	if child == nil {
		value := n.position.ValueFor(rootPlayer)
		n.visits++
		n.valueSum += value
		return value
	}
	if child.position == nil {
		next, err := n.position.Apply(child.action)
		if err != nil {
			value := n.position.ValueFor(rootPlayer)
			child.visits++
			child.valueSum += value
			n.visits++
			n.valueSum += value
			return value
		}
		child.position = next
		child.playerID = next.CurrentPlayerID()
	}
	value := runSimulation(child, evaluator, rootPlayer, config, depth+1)
	n.visits++
	n.valueSum += value
	return value
}

type selectedLeaf struct {
	node     *node
	path     []*node
	legal    []actions.Option
	terminal bool
	value    float64
}

func runBatchSimulations(root *node, evaluator model.BatchEvaluator, rootPlayer string, config Config, batchSize int) {
	if batchSize <= 0 {
		return
	}
	leaves := make([]selectedLeaf, 0, batchSize)
	positions := make([]*env.Position, 0, batchSize)
	legals := make([][]actions.Option, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		leaf := selectLeaf(root, rootPlayer, config)
		if leaf.node == nil {
			continue
		}
		if leaf.terminal {
			backpropagate(leaf.path, leaf.value)
			continue
		}
		leaves = append(leaves, leaf)
		positions = append(positions, leaf.node.position)
		legals = append(legals, leaf.legal)
	}
	if len(leaves) == 0 {
		return
	}
	uniqueLeaves := make([]selectedLeaf, 0, len(leaves))
	uniquePositions := make([]*env.Position, 0, len(positions))
	uniqueLegals := make([][]actions.Option, 0, len(legals))
	uniqueIndex := make(map[*node]int, len(leaves))
	leafIndexes := make([]int, 0, len(leaves))
	for _, leaf := range leaves {
		index, ok := uniqueIndex[leaf.node]
		if !ok {
			index = len(uniqueLeaves)
			uniqueIndex[leaf.node] = index
			uniqueLeaves = append(uniqueLeaves, leaf)
			uniquePositions = append(uniquePositions, leaf.node.position)
			uniqueLegals = append(uniqueLegals, leaf.legal)
		}
		leafIndexes = append(leafIndexes, index)
	}
	evals := evaluator.EvaluateBatch(uniquePositions, uniqueLegals, rootPlayer)
	evalByLeaf := make([]model.Evaluation, len(uniqueLeaves))
	for i, leaf := range uniqueLeaves {
		eval := model.Evaluation{Value: leaf.node.position.ValueFor(rootPlayer)}
		if i < len(evals) {
			eval = evals[i]
		}
		evalByLeaf[i] = eval
		expandWithEvaluation(leaf.node, leaf.legal, eval)
	}
	for i, leaf := range leaves {
		eval := evalByLeaf[leafIndexes[i]]
		backpropagate(leaf.path, eval.Value)
	}
}

func selectLeaf(root *node, rootPlayer string, config Config) selectedLeaf {
	n := root
	path := []*node{root}
	for depth := 0; n != nil && n.position != nil; depth++ {
		if depth >= config.MaxDepth || n.position.IsTerminal() {
			return selectedLeaf{node: n, path: path, terminal: true, value: n.position.ValueFor(rootPlayer)}
		}
		if !n.expanded {
			return selectedLeaf{node: n, path: path, legal: n.position.LegalActions()}
		}
		child := selectChild(n, rootPlayer, config.CPUCT)
		if child == nil {
			return selectedLeaf{node: n, path: path, terminal: true, value: n.position.ValueFor(rootPlayer)}
		}
		if child.position == nil {
			next, err := n.position.Apply(child.action)
			if err != nil {
				path = append(path, child)
				return selectedLeaf{node: child, path: path, terminal: true, value: n.position.ValueFor(rootPlayer)}
			}
			child.position = next
			child.playerID = next.CurrentPlayerID()
		}
		n = child
		path = append(path, n)
	}
	return selectedLeaf{}
}

func backpropagate(path []*node, value float64) {
	for _, n := range path {
		if n == nil {
			continue
		}
		n.visits++
		n.valueSum += value
	}
}

func expand(n *node, evaluator model.Evaluator, rootPlayer string) float64 {
	if n == nil || n.position == nil {
		return 0
	}
	legal := n.position.LegalActions()
	eval := evaluator.Evaluate(n.position, legal, rootPlayer)
	expandWithEvaluation(n, legal, eval)
	return eval.Value
}

func expandWithEvaluation(n *node, legal []actions.Option, eval model.Evaluation) {
	if n == nil || n.position == nil {
		return
	}
	n.children = make([]*node, 0, len(legal))
	for _, option := range legal {
		child := &node{
			parent:   n,
			playerID: option.PlayerID,
			prior:    eval.Priors[option.ID],
			action:   option,
		}
		n.children = append(n.children, child)
	}
	if len(n.children) == 0 {
		n.expanded = true
		return
	}
	normalizePriors(n.children)
	n.expanded = true
}

func selectChild(n *node, rootPlayer string, cpuct float64) *node {
	if n == nil || len(n.children) == 0 {
		return nil
	}
	parentVisits := math.Sqrt(float64(max(1, n.visits)))
	maximize := n.playerID == "" || n.playerID == rootPlayer
	var best *node
	bestScore := math.Inf(-1)
	for _, child := range n.children {
		q := 0.0
		if child.visits > 0 {
			q = child.valueSum / float64(child.visits)
		}
		if !maximize {
			q = -q
		}
		u := cpuct * child.prior * parentVisits / (1.0 + float64(child.visits))
		score := q + u
		if score > bestScore {
			bestScore = score
			best = child
		}
	}
	return best
}

func rankedChildren(root *node, temperature float64) []RankedAction {
	if root == nil || len(root.children) == 0 {
		return nil
	}
	sum := 0.0
	probs := make(map[string]float64, len(root.children))
	if temperature <= 0 {
		bestVisits := -1
		for _, child := range root.children {
			if child.visits > bestVisits {
				bestVisits = child.visits
			}
		}
		for _, child := range root.children {
			if child.visits == bestVisits {
				probs[child.action.ID] = 1
				sum++
			}
		}
	} else {
		power := 1.0 / temperature
		for _, child := range root.children {
			v := math.Pow(float64(child.visits), power)
			probs[child.action.ID] = v
			sum += v
		}
	}
	out := make([]RankedAction, 0, len(root.children))
	for _, child := range root.children {
		q := 0.0
		if child.visits > 0 {
			q = child.valueSum / float64(child.visits)
		}
		prob := 0.0
		if sum > 0 {
			prob = probs[child.action.ID] / sum
		}
		out = append(out, RankedAction{
			ID:     child.action.ID,
			Type:   child.action.Type,
			Label:  child.action.Label,
			Player: child.action.PlayerID,
			Visits: child.visits,
			Prior:  child.prior,
			Q:      q,
			Prob:   prob,
			Meta:   child.action.Meta,
			Params: child.action.Params,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Visits != out[j].Visits {
			return out[i].Visits > out[j].Visits
		}
		if out[i].Prob != out[j].Prob {
			return out[i].Prob > out[j].Prob
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func rankedPolicyChildren(root *node) []RankedAction {
	if root == nil || len(root.children) == 0 {
		return nil
	}
	out := make([]RankedAction, 0, len(root.children))
	for _, child := range root.children {
		out = append(out, RankedAction{
			ID:     child.action.ID,
			Type:   child.action.Type,
			Label:  child.action.Label,
			Player: child.action.PlayerID,
			Visits: 0,
			Prior:  child.prior,
			Q:      0,
			Prob:   child.prior,
			Meta:   child.action.Meta,
			Params: child.action.Params,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Prob != out[j].Prob {
			return out[i].Prob > out[j].Prob
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func applyRootNoise(root *node, config Config, rng *rand.Rand) {
	if root == nil || len(root.children) == 0 || config.DirichletWeight <= 0 || config.DirichletAlpha <= 0 {
		return
	}
	noises := make([]float64, len(root.children))
	sum := 0.0
	for i := range noises {
		noises[i] = gammaSample(config.DirichletAlpha, rng)
		sum += noises[i]
	}
	if sum <= 0 {
		return
	}
	for i, child := range root.children {
		noise := noises[i] / sum
		child.prior = (1-config.DirichletWeight)*child.prior + config.DirichletWeight*noise
	}
	normalizePriors(root.children)
}

func normalizePriors(children []*node) {
	total := 0.0
	for _, child := range children {
		if child.prior < 0 {
			child.prior = 0
		}
		total += child.prior
	}
	if total <= 0 {
		uniform := 1.0 / float64(len(children))
		for _, child := range children {
			child.prior = uniform
		}
		return
	}
	for _, child := range children {
		child.prior /= total
	}
}

func gammaSample(alpha float64, rng *rand.Rand) float64 {
	// Marsaglia and Tsang's method. Enough for root exploration noise.
	if alpha < 1 {
		return gammaSample(alpha+1, rng) * math.Pow(rng.Float64(), 1/alpha)
	}
	d := alpha - 1.0/3.0
	c := 1.0 / math.Sqrt(9*d)
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
