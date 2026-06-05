package env

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/replay"
)

// Position is the AlphaZero-facing game environment wrapper.
type Position struct {
	State        *game.GameState  `json:"-"`
	RootPlayerID string           `json:"rootPlayerId"`
	Metadata     ScenarioMetadata `json:"metadata,omitempty"`
	legal        []actions.Option `json:"-"`
	legalLoaded  bool             `json:"-"`
}

const ObservationSchema = "tm_az_board_v1"

// ScenarioMetadata carries the matchup identity through cloned positions and
// training records so matrix runs can prove faction coverage after the fact.
type ScenarioMetadata struct {
	Scenario         string   `json:"scenario,omitempty"`
	Factions         []string `json:"factions,omitempty"`
	RootFaction      string   `json:"rootFaction,omitempty"`
	OrderedMatchup   string   `json:"orderedMatchup,omitempty"`
	UnorderedMatchup string   `json:"unorderedMatchup,omitempty"`
}

// Observation is the flat feature vector plus enough metadata for training
// tools to verify they are consuming the same schema that self-play emitted.
type Observation struct {
	Schema       string    `json:"schema"`
	Shape        []int     `json:"shape"`
	FeatureNames []string  `json:"featureNames,omitempty"`
	Features     []float64 `json:"features"`
}

// Scenario defines a deterministic 1v1 start state for self-play.
type Scenario struct {
	Name        string                `json:"name"`
	Players     [2]string             `json:"players"`
	Factions    [2]models.FactionType `json:"factions"`
	StartingHex [2][]board.Hex        `json:"startingHexes"`
}

type SnapshotSeed struct {
	Name         string   `json:"name"`
	RootPlayerID string   `json:"rootPlayerId"`
	Snapshot     string   `json:"snapshot"`
	Source       string   `json:"source,omitempty"`
	ActionIndex  int      `json:"actionIndex,omitempty"`
	Round        int      `json:"round,omitempty"`
	Phase        string   `json:"phase,omitempty"`
	PlayerCount  int      `json:"playerCount,omitempty"`
	RootFaction  string   `json:"rootFaction,omitempty"`
	Factions     []string `json:"factions,omitempty"`
}

var baseScenarioTemplates = []Scenario{
	{
		Name:     "base_nomads_witches",
		Players:  [2]string{"p1", "p2"},
		Factions: [2]models.FactionType{models.FactionNomads, models.FactionWitches},
		StartingHex: [2][]board.Hex{
			{board.NewHex(0, 1), board.NewHex(3, 1)},
			{board.NewHex(0, 0), board.NewHex(2, 1)},
		},
	},
	{
		Name:     "base_giants_mermaids",
		Players:  [2]string{"p1", "p2"},
		Factions: [2]models.FactionType{models.FactionGiants, models.FactionMermaids},
		StartingHex: [2][]board.Hex{
			{board.NewHex(0, 1), board.NewHex(3, 1)},
			{board.NewHex(0, 0), board.NewHex(2, 1)},
		},
	},
	{
		Name:     "base_engineers_auren",
		Players:  [2]string{"p1", "p2"},
		Factions: [2]models.FactionType{models.FactionEngineers, models.FactionAuren},
		StartingHex: [2][]board.Hex{
			{board.NewHex(0, 1), board.NewHex(3, 1)},
			{board.NewHex(0, 0), board.NewHex(2, 1)},
		},
	},
	{
		Name:     "base_halflings_alchemists",
		Players:  [2]string{"p1", "p2"},
		Factions: [2]models.FactionType{models.FactionHalflings, models.FactionAlchemists},
		StartingHex: [2][]board.Hex{
			{board.NewHex(0, 1), board.NewHex(3, 1)},
			{board.NewHex(0, 0), board.NewHex(2, 1)},
		},
	},
}

// NewPosition wraps a cloned state so callers can mutate through Apply without
// touching the source game.
func NewPosition(gs *game.GameState, rootPlayerID string) *Position {
	var clone *game.GameState
	if gs != nil {
		clone = gs.CloneForUndo()
	}
	return &Position{State: clone, RootPlayerID: rootPlayerID, Metadata: metadataFromState("", clone, rootPlayerID)}
}

// LegalActions returns the legal action surface for this position.
func (p *Position) LegalActions() []actions.Option {
	if p == nil || p.State == nil {
		return nil
	}
	if !p.legalLoaded {
		p.legal = actions.LegalActions(p.State)
		p.legalLoaded = true
	}
	return append([]actions.Option(nil), p.legal...)
}

// Apply returns a new position after applying option.
func (p *Position) Apply(option actions.Option) (*Position, error) {
	if p == nil || p.State == nil {
		return nil, fmt.Errorf("nil position")
	}
	next, err := actions.ApplyToClone(p.State, option.Action)
	if err != nil {
		return nil, err
	}
	return &Position{State: next, RootPlayerID: p.RootPlayerID, Metadata: p.Metadata}, nil
}

// IsTerminal reports whether the state has ended or the legal action surface is empty.
func (p *Position) IsTerminal() bool {
	if p == nil || p.State == nil {
		return true
	}
	if p.State.Phase == game.PhaseEnd || p.State.IsGameOver() {
		return true
	}
	return len(p.LegalActions()) == 0
}

// ValueFor returns a normalized value in [-1, 1] from playerID's perspective.
func (p *Position) ValueFor(playerID string) float64 {
	if p == nil || p.State == nil {
		return 0
	}
	players := sortedPlayerIDs(p.State)
	if len(players) == 0 || playerID == "" {
		return 0
	}
	scoreFor := playerScore(p.State, playerID)
	bestOpponent := scoreFor
	for _, id := range players {
		if id == playerID {
			continue
		}
		if s := playerScore(p.State, id); s > bestOpponent || bestOpponent == scoreFor {
			bestOpponent = s
		}
	}
	margin := scoreFor - bestOpponent
	return math.Max(-1, math.Min(1, float64(margin)/80.0))
}

// CurrentPlayerID returns the player who owns the next blocking decision when known.
func (p *Position) CurrentPlayerID() string {
	if p == nil || p.State == nil {
		return ""
	}
	if options := p.LegalActions(); len(options) > 0 {
		return options[0].PlayerID
	}
	if current := p.State.GetCurrentPlayer(); current != nil {
		return current.ID
	}
	return ""
}

// Encode returns the numeric observation vector consumed by search and
// training. Keep this as the compatibility hook when evolving the schema.
func (p *Position) Encode() []float64 {
	return p.Observation().Features
}

// Observation returns a deterministic board-aware observation. The vector is
// flat for the baseline trainers, while Shape records the logical layout:
// [global feature count, board hex count, per-hex feature count].
func (p *Position) Observation() Observation {
	obs := Observation{Schema: ObservationSchema, Shape: []int{0, 0, 0}}
	if p == nil || p.State == nil {
		return obs
	}
	gs := p.State
	currentPlayerID := p.CurrentPlayerID()
	playerIDs := sortedPlayerIDs(gs)
	rootPlayerID := p.RootPlayerID
	if rootPlayerID == "" {
		rootPlayerID = currentPlayerID
	}
	features := make([]float64, 0, 128)
	names := make([]string, 0, 128)
	appendFeature := func(name string, value float64) {
		names = append(names, name)
		features = append(features, value)
	}
	appendFeature("global.round", float64(gs.Round)/6.0)
	appendFeature("global.phase", float64(gs.Phase)/5.0)
	appendFeature("global.current_player_index", normalizeIndex(gs.CurrentPlayerIndex, len(playerIDs)))
	appendFeature("global.player_count", float64(len(playerIDs))/5.0)
	appendFeature("global.pending_free_actions", boolFeature(gs.PendingFreeActionsPlayerID != ""))
	appendFeature("global.pending_turn_confirmation", boolFeature(gs.PendingTurnConfirmationPlayerID != ""))
	appendFeature("global.pending_spades_total", float64(sumIntMap(gs.PendingSpades))/12.0)
	appendFeature("global.pass_order_count", float64(len(gs.PassOrder))/float64(maxInt(1, len(playerIDs))))
	for order, id := range playerIDs {
		player := gs.GetPlayer(id)
		if player == nil || player.Resources == nil {
			continue
		}
		prefix := fmt.Sprintf("player.%d", order)
		appendFeature(prefix+".is_root", boolFeature(id == rootPlayerID))
		appendFeature(prefix+".is_current", boolFeature(id == currentPlayerID))
		appendFeature(prefix+".has_passed", boolFeature(player.HasPassed))
		if player.Faction != nil {
			appendFeature(prefix+".faction", float64(player.Faction.GetType())/float64(models.FactionSnowShamans))
			appendTerrainOneHot(prefix+".home_terrain", player.Faction.GetHomeTerrain(), appendFeature)
		} else {
			appendFeature(prefix+".faction", 0)
			appendTerrainOneHot(prefix+".home_terrain", models.TerrainTypeUnknown, appendFeature)
		}
		appendFeature(prefix+".victory_points", float64(player.VictoryPoints)/200.0)
		appendFeature(prefix+".coins", float64(player.Resources.Coins)/40.0)
		appendFeature(prefix+".workers", float64(player.Resources.Workers)/20.0)
		appendFeature(prefix+".priests", float64(player.Resources.Priests)/7.0)
		appendFeature(prefix+".power_bowl1", float64(player.Resources.Power.Bowl1)/20.0)
		appendFeature(prefix+".power_bowl2", float64(player.Resources.Power.Bowl2)/20.0)
		appendFeature(prefix+".power_bowl3", float64(player.Resources.Power.Bowl3)/20.0)
		appendFeature(prefix+".shipping", float64(player.ShippingLevel)/5.0)
		appendFeature(prefix+".digging", float64(player.DiggingLevel)/3.0)
		appendFeature(prefix+".bridges_built", float64(player.BridgesBuilt)/3.0)
		appendFeature(prefix+".keys", float64(player.Keys)/4.0)
		appendFeature(prefix+".towns", float64(player.TownsFormed)/5.0)
		for _, track := range []game.CultTrack{game.CultFire, game.CultWater, game.CultEarth, game.CultAir} {
			appendFeature(prefix+".cult."+cultTrackName(track), float64(gs.CultTracks.GetPosition(id, track))/10.0)
		}
	}
	globalCount := len(features)
	hexes := orderedHexes(gs.Map)
	perHexNames := perHexFeatureNames()
	bridgeCounts := bridgeCountsByHex(gs.Map)
	for _, hex := range hexes {
		mh := gs.Map.GetHex(hex)
		prefix := fmt.Sprintf("hex.%d.%d", hex.Q, hex.R)
		appendFeature(prefix+".q", float64(hex.Q)/13.0)
		appendFeature(prefix+".r", float64(hex.R)/9.0)
		if mh != nil {
			appendTerrainOneHot(prefix+".terrain", mh.Terrain, appendFeature)
			appendFeature(prefix+".is_river", boolFeature(gs.Map.IsRiver(hex)))
			appendBuildingFeatures(prefix+".building", mh.Building, rootPlayerID, currentPlayerID, appendFeature)
			appendFeature(prefix+".part_of_town", boolFeature(mh.PartOfTown))
			appendFeature(prefix+".has_town_tile", boolFeature(mh.HasTownTile))
			appendFeature(prefix+".bridge_count", float64(bridgeCounts[hex])/3.0)
			appendOwnerFeatures(prefix+".power_token_owner", mh.PowerTokenOwnerPlayerID, rootPlayerID, currentPlayerID, appendFeature)
		} else {
			appendTerrainOneHot(prefix+".terrain", models.TerrainTypeUnknown, appendFeature)
			appendFeature(prefix+".is_river", 0)
			appendBuildingFeatures(prefix+".building", nil, rootPlayerID, currentPlayerID, appendFeature)
			appendFeature(prefix+".part_of_town", 0)
			appendFeature(prefix+".has_town_tile", 0)
			appendFeature(prefix+".bridge_count", 0)
			appendOwnerFeatures(prefix+".power_token_owner", "", rootPlayerID, currentPlayerID, appendFeature)
		}
	}
	obs.Shape = []int{globalCount, len(hexes), len(perHexNames)}
	obs.FeatureNames = names
	obs.Features = features
	return obs
}

// SnapshotJSON emits a deterministic JSON snapshot for training records.
func (p *Position) SnapshotJSON() string {
	return p.SnapshotJSONWithObservation(p.Observation())
}

// SnapshotJSONWithObservation emits the same compact debug snapshot as
// SnapshotJSON while reusing an observation the caller already computed.
func (p *Position) SnapshotJSONWithObservation(observation Observation) string {
	if p == nil || p.State == nil {
		return "{}"
	}
	payload := map[string]interface{}{
		"rootPlayerId":      p.RootPlayerID,
		"round":             p.State.Round,
		"phase":             p.State.Phase,
		"turn":              p.CurrentPlayerID(),
		"observationSchema": ObservationSchema,
		"observationShape":  observation.Shape,
		"encoding":          observation.Features,
	}
	raw, _ := json.Marshal(payload)
	return string(raw)
}

// BuiltInScenario returns a named deterministic 1v1 self-play scenario.
func BuiltInScenario(name string) (*Position, error) {
	if name == "" {
		name = "base_nomads_witches"
	}
	for _, scenario := range baseScenarioTemplates {
		if scenario.Name == name {
			return scenarioPosition(scenario, rand.New(rand.NewSource(1)), false)
		}
	}
	if name == "randomized_base" || name == "training_mix" {
		return randomizedScenarioPosition(rand.New(rand.NewSource(1)))
	}
	if strings.HasPrefix(name, "snapshots:") {
		position, _, err := sampleSnapshotScenario(name, rand.New(rand.NewSource(1)))
		return position, err
	}
	return nil, fmt.Errorf("unknown scenario: %s", name)
}

// ScenarioNames returns the deterministic self-play scenarios built into the
// first training pipeline.
func ScenarioNames() []string {
	names := make([]string, 0, len(baseScenarioTemplates)+1)
	for _, scenario := range baseScenarioTemplates {
		names = append(names, scenario.Name)
	}
	names = append(names, "random_base")
	names = append(names, "randomized_base")
	names = append(names, "matrix:base_ordered")
	names = append(names, "matchup:Nomads:Witches")
	names = append(names, "snapshots:/path/to/snapshot_seeds.jsonl")
	names = append(names, "training_mix")
	sort.Strings(names)
	return names
}

// ScenarioSet expands named matrix suites into explicit scenario names. Callers
// can schedule these by episode/game index for deterministic faction coverage.
func ScenarioSet(name string) []string {
	switch strings.TrimSpace(name) {
	case "matrix:base_ordered", "base_ordered_matrix":
		return BaseOrderedMatchupScenarios()
	default:
		return nil
	}
}

// ScheduledScenario returns the concrete scenario for an episode or arena game.
func ScheduledScenario(name string, index int) string {
	scenarios := ScenarioSet(name)
	if len(scenarios) == 0 {
		return name
	}
	if index < 0 {
		index = 0
	}
	return scenarios[index%len(scenarios)]
}

// BaseOrderedMatchupScenarios returns every legal ordered base-faction pair.
// Same-faction and same-home-terrain pairs are excluded, matching auction rules.
func BaseOrderedMatchupScenarios() []string {
	scenarios := make([]string, 0, len(baseFactionPool)*len(baseFactionPool))
	for _, first := range baseFactionPool {
		for _, second := range baseFactionPool {
			if !validBaseFactionPair(first, second) {
				continue
			}
			scenarios = append(scenarios, matchupScenarioName(first, second))
		}
	}
	return scenarios
}

// SampleScenario returns a deterministic scenario by name, a sampled scenario
// from a comma-separated set, or a randomized base-faction pair.
func SampleScenario(name string, rng *rand.Rand) (*Position, string, error) {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "base_nomads_witches"
	}
	if strings.Contains(name, ",") {
		parts := strings.Split(name, ",")
		candidates := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				candidates = append(candidates, trimmed)
			}
		}
		if len(candidates) == 0 {
			return nil, "", fmt.Errorf("empty scenario set")
		}
		return SampleScenario(candidates[rng.Intn(len(candidates))], rng)
	}
	if name == "random" || name == "random_base" {
		scenario := baseScenarioTemplates[rng.Intn(len(baseScenarioTemplates))]
		if rng.Intn(2) == 1 {
			scenario.Factions[0], scenario.Factions[1] = scenario.Factions[1], scenario.Factions[0]
		}
		position, err := scenarioPosition(scenario, rng, true)
		return position, scenario.Name, err
	}
	if name == "randomized_base" {
		position, err := randomizedScenarioPosition(rng)
		return position, "randomized_base", err
	}
	if name == "training_mix" {
		if rng.Intn(2) == 0 {
			return SampleScenario("random_base", rng)
		}
		position, err := randomizedScenarioPosition(rng)
		return position, "randomized_base", err
	}
	if strings.HasPrefix(name, "snapshots:") {
		return sampleSnapshotScenario(name, rng)
	}
	if strings.HasPrefix(name, "matchup:") {
		position, err := matchupScenarioPosition(name, rng)
		return position, name, err
	}
	if scheduled := ScheduledScenario(name, 0); scheduled != name {
		return SampleScenario(scheduled, rng)
	}
	position, err := BuiltInScenario(name)
	return position, name, err
}

func scenarioPosition(s Scenario, rng *rand.Rand, randomizeRoundAssets bool) (*Position, error) {
	gs := game.NewGameState()
	gs.Phase = game.PhaseAction
	gs.Round = 1
	gs.TurnOrder = []string{s.Players[0], s.Players[1]}
	gs.CurrentPlayerIndex = 0
	for i, playerID := range s.Players {
		faction := factions.NewFaction(s.Factions[i])
		if err := gs.AddPlayer(playerID, faction); err != nil {
			return nil, err
		}
		player := gs.GetPlayer(playerID)
		player.Options.ConfirmActions = false
		player.Resources.Coins += 8
		player.Resources.Workers += 4
		player.Resources.Priests += 1
		for _, hex := range s.StartingHex[i] {
			if mh := gs.Map.GetHex(hex); mh == nil {
				return nil, fmt.Errorf("scenario hex missing: %v", hex)
			}
			if err := gs.Map.TransformTerrain(hex, faction.GetHomeTerrain()); err != nil {
				return nil, err
			}
			if err := gs.Map.PlaceBuilding(hex, &models.Building{
				Type:       models.BuildingDwelling,
				Faction:    faction.GetType(),
				PlayerID:   playerID,
				PowerValue: 1,
			}); err != nil {
				return nil, err
			}
		}
	}
	configureRoundAssets(gs, s.Players[:], rng, randomizeRoundAssets)
	return &Position{
		State:        gs,
		RootPlayerID: s.Players[0],
		Metadata:     metadataFromScenario(s, s.Name),
	}, nil
}

func matchupScenarioPosition(name string, rng *rand.Rand) (*Position, error) {
	first, second, err := parseMatchupScenario(name)
	if err != nil {
		return nil, err
	}
	startingHexes, err := sampleStartingHexes(rng)
	if err != nil {
		return nil, err
	}
	return scenarioPosition(Scenario{
		Name:        matchupScenarioName(first, second),
		Players:     [2]string{"p1", "p2"},
		Factions:    [2]models.FactionType{first, second},
		StartingHex: startingHexes,
	}, rng, true)
}

func randomizedScenarioPosition(rng *rand.Rand) (*Position, error) {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	factions := sampleFactionPair(rng)
	players := [2]string{"p1", "p2"}
	startingHexes, err := sampleStartingHexes(rng)
	if err != nil {
		return nil, err
	}
	scenario := Scenario{
		Name:        "randomized_base",
		Players:     players,
		Factions:    factions,
		StartingHex: startingHexes,
	}
	if rng.Intn(2) == 1 {
		scenario.Factions[0], scenario.Factions[1] = scenario.Factions[1], scenario.Factions[0]
		scenario.StartingHex[0], scenario.StartingHex[1] = scenario.StartingHex[1], scenario.StartingHex[0]
	}
	return scenarioPosition(scenario, rng, true)
}

func sampleSnapshotScenario(name string, rng *rand.Rand) (*Position, string, error) {
	path := strings.TrimSpace(strings.TrimPrefix(name, "snapshots:"))
	if path == "" {
		return nil, "", fmt.Errorf("snapshot seed path is required")
	}
	seeds, err := loadSnapshotSeeds(path)
	if err != nil {
		return nil, "", err
	}
	if len(seeds) == 0 {
		return nil, "", fmt.Errorf("no snapshot seeds in %s", path)
	}
	seed := seeds[rng.Intn(len(seeds))]
	gs, err := replay.ParseSnapshot(seed.Snapshot)
	if err != nil {
		return nil, "", fmt.Errorf("parse snapshot seed %s: %w", seed.Name, err)
	}
	normalizeSnapshotState(gs)
	rootPlayerID := strings.TrimSpace(seed.RootPlayerID)
	if rootPlayerID == "" {
		rootPlayerID = currentOrFirstPlayerID(gs)
	}
	scenarioName := seed.Name
	if scenarioName == "" {
		scenarioName = "snapshot"
	}
	position := NewPosition(gs, rootPlayerID)
	position.Metadata = metadataFromSnapshotSeed(scenarioName, seed, gs, rootPlayerID)
	return position, scenarioName, nil
}

func normalizeSnapshotState(gs *game.GameState) {
	if gs == nil {
		return
	}
	if gs.PendingLeechOffers == nil {
		gs.PendingLeechOffers = make(map[string][]*game.PowerLeechOffer)
	}
	if gs.PendingTownFormations == nil {
		gs.PendingTownFormations = make(map[string][]*game.PendingTownFormation)
	}
	if gs.PendingSpades == nil {
		gs.PendingSpades = make(map[string]int)
	}
	if gs.PendingSpadeBuildAllowed == nil {
		gs.PendingSpadeBuildAllowed = make(map[string]bool)
	}
	if gs.PendingCultistsLeech == nil {
		gs.PendingCultistsLeech = make(map[int]*game.CultistsLeechBonus)
	}
	if gs.PendingShapeshiftersLeech == nil {
		gs.PendingShapeshiftersLeech = make(map[int]*game.CultistsLeechBonus)
	}
	if gs.SkipAbilityUsedThisAction == nil {
		gs.SkipAbilityUsedThisAction = make(map[string][]board.Hex)
	}
	for id, player := range gs.Players {
		if player == nil {
			continue
		}
		player.ID = id
		player.Options.ConfirmActions = false
		if player.SpecialActionsUsed == nil {
			player.SpecialActionsUsed = make(map[game.SpecialActionType]bool)
		}
		if player.CultPositions == nil {
			player.CultPositions = make(map[game.CultTrack]int)
		}
	}
}

func loadSnapshotSeeds(path string) ([]SnapshotSeed, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var seeds []SnapshotSeed
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 64*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var seed SnapshotSeed
		if err := json.Unmarshal([]byte(raw), &seed); err != nil {
			return nil, fmt.Errorf("parse snapshot seed line %d: %w", line, err)
		}
		if strings.TrimSpace(seed.Snapshot) == "" {
			return nil, fmt.Errorf("snapshot seed line %d has empty snapshot", line)
		}
		seeds = append(seeds, seed)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return seeds, nil
}

func currentOrFirstPlayerID(gs *game.GameState) string {
	if gs == nil {
		return ""
	}
	if current := gs.GetCurrentPlayer(); current != nil {
		return current.ID
	}
	ids := sortedPlayerIDs(gs)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func configureRoundAssets(gs *game.GameState, players []string, rng *rand.Rand, randomize bool) {
	if gs == nil || rng == nil {
		return
	}
	if randomize {
		gs.ScoringTiles.Tiles = randomScoringTiles(rng)
		gs.BonusCards.SetAvailableBonusCards(randomBonusCards(rng, len(players)+3))
	} else if len(gs.ScoringTiles.Tiles) == 0 {
		gs.ScoringTiles.Tiles = game.GetAllScoringTiles()[:6]
		gs.BonusCards.SetAvailableBonusCards([]game.BonusCardType{
			game.BonusCardPriest,
			game.BonusCardShipping,
			game.BonusCardDwellingVP,
			game.BonusCardWorkerPower,
			game.BonusCardSpade,
		})
	}
	for i, playerID := range players {
		available := sortedAvailableBonusCards(gs)
		if len(available) == 0 {
			return
		}
		card := available[i%len(available)]
		if randomize {
			card = available[rng.Intn(len(available))]
		}
		_, _ = gs.BonusCards.TakeBonusCard(playerID, card)
	}
	gs.BonusCards.AddCoinsToLeftoverCards()
	gs.BonusCards.PlayerHasCard = make(map[string]bool)
}

func playerScore(gs *game.GameState, playerID string) int {
	if gs == nil {
		return 0
	}
	if gs.Phase == game.PhaseEnd || gs.IsGameOver() {
		if score := gs.CalculateFinalScoring()[playerID]; score != nil {
			return score.TotalVP
		}
	}
	player := gs.GetPlayer(playerID)
	if player == nil {
		return 0
	}
	return player.VictoryPoints
}

func sortedPlayerIDs(gs *game.GameState) []string {
	ids := make([]string, 0, len(gs.Players))
	for id := range gs.Players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

var baseFactionPool = []models.FactionType{
	models.FactionNomads,
	models.FactionFakirs,
	models.FactionChaosMagicians,
	models.FactionGiants,
	models.FactionSwarmlings,
	models.FactionMermaids,
	models.FactionWitches,
	models.FactionAuren,
	models.FactionHalflings,
	models.FactionCultists,
	models.FactionAlchemists,
	models.FactionDarklings,
	models.FactionEngineers,
	models.FactionDwarves,
}

func sampleFactionPair(rng *rand.Rand) [2]models.FactionType {
	first := baseFactionPool[rng.Intn(len(baseFactionPool))]
	candidates := make([]models.FactionType, 0, len(baseFactionPool))
	for _, candidate := range baseFactionPool {
		if !validBaseFactionPair(first, candidate) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	second := candidates[rng.Intn(len(candidates))]
	return [2]models.FactionType{first, second}
}

func validBaseFactionPair(first, second models.FactionType) bool {
	if first == second || first == models.FactionUnknown || second == models.FactionUnknown {
		return false
	}
	return factions.NewFaction(first).GetHomeTerrain() != factions.NewFaction(second).GetHomeTerrain()
}

func parseMatchupScenario(name string) (models.FactionType, models.FactionType, error) {
	parts := strings.Split(strings.TrimSpace(name), ":")
	if len(parts) != 3 || parts[0] != "matchup" {
		return models.FactionUnknown, models.FactionUnknown, fmt.Errorf("matchup scenario must be matchup:FactionA:FactionB")
	}
	first := parseBaseFaction(parts[1])
	second := parseBaseFaction(parts[2])
	if !validBaseFactionPair(first, second) {
		return first, second, fmt.Errorf("invalid base faction matchup: %s vs %s", parts[1], parts[2])
	}
	return first, second, nil
}

func parseBaseFaction(name string) models.FactionType {
	normalized := strings.ReplaceAll(strings.TrimSpace(name), " ", "")
	for _, faction := range baseFactionPool {
		if strings.EqualFold(normalized, strings.ReplaceAll(faction.String(), " ", "")) {
			return faction
		}
	}
	return models.FactionUnknown
}

func matchupScenarioName(first, second models.FactionType) string {
	return fmt.Sprintf("matchup:%s:%s", first.String(), second.String())
}

func metadataFromScenario(s Scenario, scenarioName string) ScenarioMetadata {
	factionNames := []string{s.Factions[0].String(), s.Factions[1].String()}
	return ScenarioMetadata{
		Scenario:         scenarioName,
		Factions:         factionNames,
		RootFaction:      factionNames[0],
		OrderedMatchup:   orderedMatchup(factionNames),
		UnorderedMatchup: unorderedMatchup(factionNames),
	}
}

func metadataFromSnapshotSeed(scenarioName string, seed SnapshotSeed, gs *game.GameState, rootPlayerID string) ScenarioMetadata {
	metadata := metadataFromState(scenarioName, gs, rootPlayerID)
	if len(seed.Factions) > 0 {
		metadata.Factions = append([]string(nil), seed.Factions...)
		metadata.OrderedMatchup = orderedMatchup(metadata.Factions)
		metadata.UnorderedMatchup = unorderedMatchup(metadata.Factions)
	}
	if strings.TrimSpace(seed.RootFaction) != "" {
		metadata.RootFaction = strings.TrimSpace(seed.RootFaction)
	}
	return metadata
}

func metadataFromState(scenarioName string, gs *game.GameState, rootPlayerID string) ScenarioMetadata {
	metadata := ScenarioMetadata{Scenario: scenarioName}
	if gs == nil {
		return metadata
	}
	ids := sortedPlayerIDs(gs)
	metadata.Factions = make([]string, 0, len(ids))
	for _, id := range ids {
		player := gs.GetPlayer(id)
		if player == nil || player.Faction == nil {
			continue
		}
		name := player.Faction.GetType().String()
		metadata.Factions = append(metadata.Factions, name)
		if id == rootPlayerID {
			metadata.RootFaction = name
		}
	}
	metadata.OrderedMatchup = orderedMatchup(metadata.Factions)
	metadata.UnorderedMatchup = unorderedMatchup(metadata.Factions)
	return metadata
}

func orderedMatchup(factions []string) string {
	if len(factions) < 2 {
		return ""
	}
	return fmt.Sprintf("%s_vs_%s", factions[0], factions[1])
}

func unorderedMatchup(factionNames []string) string {
	if len(factionNames) < 2 {
		return ""
	}
	names := append([]string(nil), factionNames[:2]...)
	sort.Strings(names)
	return fmt.Sprintf("%s_vs_%s", names[0], names[1])
}

func sampleStartingHexes(rng *rand.Rand) ([2][]board.Hex, error) {
	gs := game.NewGameState()
	hexes := orderedHexes(gs.Map)
	land := make([]board.Hex, 0, len(hexes))
	for _, hex := range hexes {
		if !gs.Map.IsRiver(hex) {
			land = append(land, hex)
		}
	}
	rng.Shuffle(len(land), func(i, j int) {
		land[i], land[j] = land[j], land[i]
	})
	if len(land) < 4 {
		return [2][]board.Hex{}, fmt.Errorf("not enough land hexes for randomized scenario")
	}
	return [2][]board.Hex{
		{land[0], land[1]},
		{land[2], land[3]},
	}, nil
}

func randomScoringTiles(rng *rand.Rand) []game.ScoringTile {
	tiles := game.GetAllScoringTiles()
	rng.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
	selected := make([]game.ScoringTile, 0, 6)
	for _, tile := range tiles {
		if len(selected) >= 6 {
			break
		}
		if tile.Type == game.ScoringSpades && len(selected) >= 4 {
			continue
		}
		selected = append(selected, tile)
	}
	if len(selected) < 6 {
		for _, tile := range tiles {
			if len(selected) >= 6 {
				break
			}
			if containsScoringTile(selected, tile.Type) {
				continue
			}
			selected = append(selected, tile)
		}
	}
	return selected
}

func containsScoringTile(tiles []game.ScoringTile, tileType game.ScoringTileType) bool {
	for _, tile := range tiles {
		if tile.Type == tileType {
			return true
		}
	}
	return false
}

func randomBonusCards(rng *rand.Rand, count int) []game.BonusCardType {
	all := make([]game.BonusCardType, 0, len(game.GetAllBonusCards()))
	for card := range game.GetAllBonusCards() {
		all = append(all, card)
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	rng.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})
	if count > len(all) {
		count = len(all)
	}
	return append([]game.BonusCardType(nil), all[:count]...)
}

func sortedAvailableBonusCards(gs *game.GameState) []game.BonusCardType {
	if gs == nil || gs.BonusCards == nil {
		return nil
	}
	out := make([]game.BonusCardType, 0, len(gs.BonusCards.Available))
	for card := range gs.BonusCards.Available {
		out = append(out, card)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

var observationTerrains = []models.TerrainType{
	models.TerrainPlains,
	models.TerrainSwamp,
	models.TerrainLake,
	models.TerrainForest,
	models.TerrainMountain,
	models.TerrainWasteland,
	models.TerrainDesert,
	models.TerrainRiver,
	models.TerrainIce,
	models.TerrainVolcano,
}

var observationBuildings = []models.BuildingType{
	models.BuildingDwelling,
	models.BuildingTradingHouse,
	models.BuildingTemple,
	models.BuildingSanctuary,
	models.BuildingStronghold,
}

func perHexFeatureNames() []string {
	names := []string{"q", "r"}
	for _, terrain := range observationTerrains {
		names = append(names, "terrain."+terrain.String())
	}
	names = append(names, "is_river")
	for _, building := range observationBuildings {
		names = append(names, "building.type."+building.String())
	}
	names = append(names,
		"building.owner.any",
		"building.owner.root",
		"building.owner.current",
		"building.owner.opponent",
		"building.power",
		"part_of_town",
		"has_town_tile",
		"bridge_count",
		"power_token_owner.any",
		"power_token_owner.root",
		"power_token_owner.current",
		"power_token_owner.opponent",
	)
	return names
}

func appendTerrainOneHot(prefix string, terrain models.TerrainType, appendFeature func(string, float64)) {
	for _, candidate := range observationTerrains {
		appendFeature(prefix+"."+candidate.String(), boolFeature(terrain == candidate))
	}
}

func appendBuildingFeatures(prefix string, building *models.Building, rootPlayerID, currentPlayerID string, appendFeature func(string, float64)) {
	for _, candidate := range observationBuildings {
		appendFeature(prefix+".type."+candidate.String(), boolFeature(building != nil && building.Type == candidate))
	}
	owner := ""
	power := 0
	if building != nil {
		owner = building.PlayerID
		power = building.PowerValue
	}
	appendOwnerFeatures(prefix+".owner", owner, rootPlayerID, currentPlayerID, appendFeature)
	appendFeature(prefix+".power", float64(power)/4.0)
}

func appendOwnerFeatures(prefix, ownerPlayerID, rootPlayerID, currentPlayerID string, appendFeature func(string, float64)) {
	appendFeature(prefix+".any", boolFeature(ownerPlayerID != ""))
	appendFeature(prefix+".root", boolFeature(ownerPlayerID != "" && ownerPlayerID == rootPlayerID))
	appendFeature(prefix+".current", boolFeature(ownerPlayerID != "" && ownerPlayerID == currentPlayerID))
	appendFeature(prefix+".opponent", boolFeature(ownerPlayerID != "" && ownerPlayerID != rootPlayerID))
}

func orderedHexes(m *board.TerraMysticaMap) []board.Hex {
	if m == nil {
		return nil
	}
	hexes := make([]board.Hex, 0, len(m.Hexes))
	for hex := range m.Hexes {
		hexes = append(hexes, hex)
	}
	sort.Slice(hexes, func(i, j int) bool {
		if hexes[i].R != hexes[j].R {
			return hexes[i].R < hexes[j].R
		}
		return hexes[i].Q < hexes[j].Q
	})
	return hexes
}

func bridgeCountsByHex(m *board.TerraMysticaMap) map[board.Hex]int {
	counts := make(map[board.Hex]int)
	if m == nil {
		return counts
	}
	for bridge := range m.Bridges {
		counts[bridge.H1]++
		counts[bridge.H2]++
	}
	return counts
}

func cultTrackName(track game.CultTrack) string {
	switch track {
	case game.CultFire:
		return "fire"
	case game.CultWater:
		return "water"
	case game.CultEarth:
		return "earth"
	case game.CultAir:
		return "air"
	default:
		return "unknown"
	}
}

func boolFeature(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func normalizeIndex(index, count int) float64 {
	if count <= 1 {
		return 0
	}
	return float64(index) / float64(count-1)
}

func sumIntMap(values map[string]int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
