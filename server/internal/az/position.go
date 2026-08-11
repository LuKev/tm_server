package az

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

type PlayerID string
type Hash128 [16]byte

// StateSchemaVersion changes whenever CanonicalJSON's search-state semantics
// change independently of the rules or SearchAction schema.
const StateSchemaVersion = 1

func (h Hash128) String() string { return hex.EncodeToString(h[:]) }

// Position is the complete interface consumed by MCTS.
type Position interface {
	Clone() Position
	DecisionPlayer() PlayerID
	LegalActions() []SearchAction
	Apply(SearchAction) error
	IsTerminal() bool
	Outcome(PlayerID) float32
	CanonicalHash() Hash128
}

// GamePosition adapts the authoritative game state to Position.
type GamePosition struct {
	state           *game.GameState
	canonicalRecord []byte
	cachedHash      Hash128
	canonicalReady  bool
}

func NewPosition(gs *game.GameState) (*GamePosition, error) {
	if gs == nil {
		return nil, fmt.Errorf("game state is nil")
	}
	if len(gs.Players) != 2 {
		return nil, fmt.Errorf("AlphaZero profile requires exactly two players, got %d", len(gs.Players))
	}
	if len(gs.TurnOrder) != 2 || gs.TurnOrder[0] == gs.TurnOrder[1] || gs.GetPlayer(gs.TurnOrder[0]) == nil || gs.GetPlayer(gs.TurnOrder[1]) == nil {
		return nil, fmt.Errorf("AlphaZero profile requires a complete two-player turn order")
	}
	if gs.EnableFanFactions || gs.EnableFireIceFactions {
		return nil, fmt.Errorf("AlphaZero v0 supports base factions and base rules only")
	}
	if gs.Map == nil || gs.Map.ID != board.MapBase {
		return nil, fmt.Errorf("AlphaZero v0 requires the base map")
	}
	if gs.SetupMode != game.SetupModeSnellman {
		return nil, fmt.Errorf("AlphaZero v0 requires Snellman setup")
	}
	if gs.TurnOrderPolicy != game.TurnOrderPolicyPassOrder {
		return nil, fmt.Errorf("AlphaZero v0 requires base pass-order turn order")
	}
	if gs.FireIceFinalScoringSetting != game.FireIceFinalScoringOff || gs.FireIceFinalScoringTile != game.FireIceFinalScoringTileNone {
		return nil, fmt.Errorf("AlphaZero v0 excludes Fire & Ice final scoring")
	}
	if gs.TownTiles == nil {
		return nil, fmt.Errorf("AlphaZero v0 requires town tiles")
	}
	if gs.BonusCards == nil {
		return nil, fmt.Errorf("AlphaZero v0 requires bonus cards")
	}
	for card := range gs.BonusCards.Available {
		if !isBaseBonusCard(card) {
			return nil, fmt.Errorf("AlphaZero v0 excludes expansion bonus card %v", card)
		}
	}
	for _, card := range gs.BonusCards.PlayerCards {
		if !isBaseBonusCard(card) {
			return nil, fmt.Errorf("AlphaZero v0 excludes held expansion bonus card %v", card)
		}
	}
	for _, cards := range gs.BonusCards.PlayerExtraCards {
		if len(cards) > 0 {
			return nil, fmt.Errorf("AlphaZero v0 excludes extra bonus-card holdings")
		}
	}
	for tile := range gs.TownTiles.Available {
		if !isBaseTownTile(tile) {
			return nil, fmt.Errorf("AlphaZero v0 excludes expansion town tile %v", tile)
		}
	}
	homeTerrains := make(map[models.TerrainType]bool, 2)
	for _, player := range gs.Players {
		if player.Faction == nil {
			if gs.Phase != game.PhaseFactionSelection {
				return nil, fmt.Errorf("AlphaZero v0 requires assigned factions outside faction selection")
			}
			continue
		}
		if !isBaseFaction(player.Faction.GetType()) {
			return nil, fmt.Errorf("AlphaZero v0 excludes faction %s", player.Faction.GetType())
		}
		home := player.Faction.GetHomeTerrain()
		if homeTerrains[home] {
			return nil, fmt.Errorf("AlphaZero v0 requires factions with distinct home terrain")
		}
		homeTerrains[home] = true
		if len(player.AtlanteansTownHexes) > 0 || len(player.AtlanteansTownRewards) > 0 {
			return nil, fmt.Errorf("AlphaZero v0 excludes expansion-faction player state")
		}
		if player.ChashIncomeTrackLevel != 0 || player.GoblinTreasureTokens != 0 ||
			player.DjinniLampTokens != 0 || player.TreasuryCoins != 0 ||
			player.TreasuryWorkers != 0 || player.TreasuryPriests != 0 ||
			player.HasStartingTerrain || len(player.UnlockedTerrains) > 0 ||
			player.FirewalkersBlockerVP != 0 {
			return nil, fmt.Errorf("AlphaZero v0 excludes fan-faction player state")
		}
		for _, tile := range player.TownTiles {
			if !isBaseTownTile(tile) {
				return nil, fmt.Errorf("AlphaZero v0 excludes claimed expansion town tile %v", tile)
			}
		}
	}
	for _, mapHex := range gs.Map.Hexes {
		if mapHex != nil && mapHex.HasTownTile && !isBaseTownTile(mapHex.TownTileType) {
			return nil, fmt.Errorf("AlphaZero v0 excludes placed expansion town tile %v", mapHex.TownTileType)
		}
	}
	if gs.TurnTimer != nil {
		return nil, fmt.Errorf("AlphaZero profile does not support turn timers")
	}
	if gs.HasPendingTurnConfirmation() {
		return nil, fmt.Errorf("cannot adapt a UX turn-confirmation checkpoint")
	}
	for _, enabled := range gs.ReplayMode {
		if enabled {
			return nil, fmt.Errorf("AlphaZero profile rejects replay tolerance mode")
		}
	}
	// These fields are intentionally omitted from the public GameState JSON.
	// They belong to fan factions, replay import, or in-action scratch state and
	// must never cross a v0 search-position boundary where canonical hashing
	// could otherwise miss them.
	if gs.PendingGoblinsCultSteps != nil ||
		gs.PendingWispsStrongholdDwelling != nil ||
		gs.PendingDjinniStartingCultChoice != nil ||
		gs.PendingArchivistsBonusSelection != nil ||
		gs.PendingTreasurersDeposit != nil ||
		gs.PendingRiverwalkersPriestChoice != nil ||
		len(gs.PendingShapeshiftersLeech) > 0 ||
		len(gs.PendingTreasurersDepositQueue) > 0 ||
		len(gs.PendingWispsTradingPostSpade) > 0 ||
		len(gs.PendingPostActionSpecialActions) > 0 ||
		len(gs.ReplayAcolytesCultTracks) > 0 ||
		len(gs.ReplayAcolytesCultTrackIndex) > 0 ||
		len(gs.ReplayRiverBuildHexes) > 0 ||
		len(gs.ReplayRiverBuildHexIndex) > 0 ||
		len(gs.ReplayCultSpadeBuildHexes) > 0 ||
		len(gs.PendingSnowShamansPassUpgrade) > 0 ||
		gs.SuppressTurnAdvance || gs.RiverTownHex != nil {
		return nil, fmt.Errorf("AlphaZero profile rejects replay, fan-faction, and in-action scratch state")
	}
	clone := gs.CloneForUndo()
	clone.TurnTimer = nil
	clone.PendingTurnConfirmationPlayerID = ""
	clone.PendingTurnConfirmationSnapshot = nil
	for _, p := range clone.Players {
		p.Name = ""
		p.Options.ConfirmActions = false
		p.Options.AutoConvertOnPass = false
		p.Options.AutoLeechMode = game.LeechAutoModeOff
	}
	normalizeSemanticSets(clone, nil)
	return &GamePosition{state: clone}, nil
}

func isBaseTownTile(tile models.TownTileType) bool {
	switch tile {
	case models.TownTile5Points, models.TownTile6Points, models.TownTile7Points,
		models.TownTile8Points, models.TownTile9Points:
		return true
	default:
		return false
	}
}

func isBaseBonusCard(card game.BonusCardType) bool {
	return card >= game.BonusCardPriest && card <= game.BonusCardStrongholdSanctuary
}

func (p *GamePosition) Clone() Position {
	return &GamePosition{
		state:           p.state.CloneForUndo(),
		canonicalRecord: p.canonicalRecord,
		cachedHash:      p.cachedHash,
		canonicalReady:  p.canonicalReady,
	}
}

func (p *GamePosition) DecisionPlayer() PlayerID {
	if p == nil || p.state == nil || p.IsTerminal() {
		return ""
	}
	ids := game.DecisionPlayerIDs(p.state)
	if len(ids) != 1 {
		return ""
	}
	return PlayerID(ids[0])
}

func (p *GamePosition) LegalActions() []SearchAction {
	if p == nil || p.state == nil || p.IsTerminal() {
		return nil
	}
	return legalActions(p.state)
}

func (p *GamePosition) Apply(searchAction SearchAction) error {
	if p == nil || p.state == nil {
		return fmt.Errorf("position is nil")
	}
	playerID := string(p.DecisionPlayer())
	if playerID == "" {
		return fmt.Errorf("position has no unique decision player")
	}
	action, err := searchAction.GameAction(playerID)
	if err != nil {
		return err
	}
	next := p.state.CloneForUndo()
	if err := game.ApplyActionToState(next, action, game.StateActionOptions{}); err != nil {
		return err
	}
	normalizeSemanticSets(next, nil)
	p.state = next
	p.canonicalRecord = nil
	p.cachedHash = Hash128{}
	p.canonicalReady = false
	return nil
}

func (p *GamePosition) IsTerminal() bool {
	return p != nil && p.state != nil && p.state.Phase == game.PhaseEnd
}

func (p *GamePosition) Outcome(playerID PlayerID) float32 {
	if !p.IsTerminal() {
		return 0
	}
	scores := p.state.CloneForUndo().CalculateFinalScoring()
	mine := scores[string(playerID)]
	if mine == nil || len(scores) != 2 {
		return 0
	}
	for id, other := range scores {
		if id == string(playerID) {
			continue
		}
		if mine.TotalVP > other.TotalVP {
			return 1
		}
		if mine.TotalVP < other.TotalVP {
			return -1
		}
		return 0
	}
	return 0
}

func (p *GamePosition) FinalVP(playerID PlayerID) (int, bool) {
	if !p.IsTerminal() {
		return 0, false
	}
	score := p.state.CloneForUndo().CalculateFinalScoring()[string(playerID)]
	if score == nil {
		return 0, false
	}
	return score.TotalVP, true
}

func (p *GamePosition) CanonicalHash() Hash128 {
	_, err := p.canonicalBytes()
	if err != nil {
		panic(err)
	}
	return p.cachedHash
}

// CanonicalJSON returns a deterministic, ID-relative state record for
// trajectory storage and hashing. UI options and replay-only fields are absent.
func (p *GamePosition) CanonicalJSON() ([]byte, error) {
	record, err := p.canonicalBytes()
	if err != nil {
		return nil, err
	}
	// Keep the cached record immutable even when a caller retains or mutates the
	// returned byte slice.
	return append([]byte(nil), record...), nil
}

func (p *GamePosition) canonicalBytes() ([]byte, error) {
	if p == nil || p.state == nil {
		return nil, fmt.Errorf("position is nil")
	}
	if p.canonicalReady {
		return p.canonicalRecord, nil
	}
	clone := p.state.CloneForUndo()
	clone.TurnTimer = nil
	clone.ReplayMode = nil
	clone.PendingTurnConfirmationPlayerID = ""
	clone.PendingTurnConfirmationSnapshot = nil
	for _, player := range clone.Players {
		player.Name = ""
		player.Options = game.PlayerOptions{}
	}

	ids := make([]string, 0, 2)
	decision := string(p.DecisionPlayer())
	if decision != "" {
		ids = append(ids, decision)
	}
	for _, id := range clone.TurnOrder {
		if id != decision {
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 {
		for id := range clone.Players {
			if id != decision {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) != 2 {
		return nil, fmt.Errorf("expected two canonical player IDs, got %v", ids)
	}
	replacements := map[string]string{ids[0]: "self", ids[1]: "opponent"}
	normalizeSemanticSets(clone, replacements)
	raw, err := json.Marshal(clone)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	value = normalizePlayerIDs(value, replacements, "", false, false)
	factionState := []factions.SearchState{
		factions.StateForSearch(clone.GetPlayer(ids[0]).Faction),
		factions.StateForSearch(clone.GetPlayer(ids[1]).Faction),
	}
	record, err := json.Marshal(struct {
		RulesVersion  int                    `json:"rules_version"`
		StateVersion  int                    `json:"state_version"`
		ActionVersion int                    `json:"action_version"`
		State         any                    `json:"state"`
		FactionState  []factions.SearchState `json:"faction_state"`
	}{RulesVersion: 1, StateVersion: StateSchemaVersion, ActionVersion: ActionSchemaVersion, State: value, FactionState: factionState})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(record)
	copy(p.cachedHash[:], sum[:16])
	p.canonicalRecord = record
	p.canonicalReady = true
	return p.canonicalRecord, nil
}

// normalizeSemanticSets removes incidental construction order from fields the
// rules consume as sets. Keep this list explicit so a genuinely ordered slice
// cannot silently lose meaning in the canonical state.
func normalizeSemanticSets(gs *game.GameState, playerIDs map[string]string) {
	for _, formations := range gs.PendingTownFormations {
		for _, formation := range formations {
			if formation == nil {
				continue
			}
			sortHexSet(formation.Hexes)
		}
		sort.Slice(formations, func(i, j int) bool { return pendingTownFormationLess(formations[i], formations[j]) })
	}
	if choice := gs.PendingTownCultTopChoice; choice != nil {
		sort.Slice(choice.CandidateTracks, func(i, j int) bool {
			return choice.CandidateTracks[i] < choice.CandidateTracks[j]
		})
	}
	if pending := gs.PendingFavorTileSelection; pending != nil {
		sort.Slice(pending.SelectedTiles, func(i, j int) bool { return pending.SelectedTiles[i] < pending.SelectedTiles[j] })
	}
	if pending := gs.PendingHalflingsSpades; pending != nil {
		sortHexSet(pending.TransformedHexes)
	}
	for _, player := range gs.Players {
		sort.Slice(player.TownTiles, func(i, j int) bool { return player.TownTiles[i] < player.TownTiles[j] })
	}
	if gs.FavorTiles != nil {
		for _, tiles := range gs.FavorTiles.PlayerTiles {
			sort.Slice(tiles, func(i, j int) bool { return tiles[i] < tiles[j] })
		}
	}
	if gs.CultTracks != nil {
		for _, bySpace := range gs.CultTracks.PriestsOnTrack {
			for _, occupants := range bySpace {
				sort.Slice(occupants, func(i, j int) bool {
					return canonicalPlayerSortKey(occupants[i], playerIDs) < canonicalPlayerSortKey(occupants[j], playerIDs)
				})
			}
		}
	}
}

func sortHexSet(hexes []board.Hex) {
	sort.Slice(hexes, func(i, j int) bool {
		if hexes[i].R != hexes[j].R {
			return hexes[i].R < hexes[j].R
		}
		return hexes[i].Q < hexes[j].Q
	})
}

func pendingTownFormationLess(a, b *game.PendingTownFormation) bool {
	if a == nil || b == nil {
		return a != nil
	}
	if a.CanBeDelayed != b.CanBeDelayed {
		return !a.CanBeDelayed
	}
	if a.SkippedRiverHex != nil || b.SkippedRiverHex != nil {
		if a.SkippedRiverHex == nil || b.SkippedRiverHex == nil {
			return a.SkippedRiverHex == nil
		}
		if *a.SkippedRiverHex != *b.SkippedRiverHex {
			return hexLess(*a.SkippedRiverHex, *b.SkippedRiverHex)
		}
	}
	for i := 0; i < len(a.Hexes) && i < len(b.Hexes); i++ {
		if a.Hexes[i] != b.Hexes[i] {
			return hexLess(a.Hexes[i], b.Hexes[i])
		}
	}
	return len(a.Hexes) < len(b.Hexes)
}

func hexLess(a, b board.Hex) bool {
	return a.R < b.R || (a.R == b.R && a.Q < b.Q)
}

func canonicalPlayerSortKey(playerID string, replacements map[string]string) string {
	if replacement, ok := replacements[playerID]; ok {
		return replacement
	}
	return playerID
}

var playerIDKeyedMaps = map[string]bool{
	"players": true, "setupPlacedDwellings": true,
	"pendingLeechOffers": true, "pendingTownFormations": true,
	"pendingSpades": true, "pendingSpadeBuildAllowed": true,
	"pendingCultRewardSpades": true, "skipAbilityUsedThisAction": true,
	"finalScoring": true, "playerPositions": true,
	"bonusPositionsClaimed": true, "priestsOnActionSpaces": true,
	"playerTiles": true, "playerCards": true, "playerExtraCards": true,
	"playerHasCard": true, "priestsSent": true,
}

var playerIDCollections = map[string]bool{
	"turnOrder": true, "passOrder": true, "setupDwellingOrder": true,
	"setupBonusOrder": true, "bridges": true, "position10Occupied": true,
	"priestsOnTrack": true,
}

// normalizePlayerIDs is deliberately schema-aware. Replacing every matching
// JSON string would corrupt unrelated enum values when a player happens to be
// named "0", "base", or "pass_order".
func normalizePlayerIDs(value any, replacements map[string]string, context string, playerObject, stringsArePlayerIDs bool) any {
	switch typed := value.(type) {
	case string:
		fieldIsPlayerID := strings.Contains(strings.ToLower(context), "playerid") || (playerObject && context == "id")
		if stringsArePlayerIDs || fieldIsPlayerID {
			if replacement, ok := replacements[typed]; ok {
				return replacement
			}
		}
		return typed
	case []any:
		for i := range typed {
			typed[i] = normalizePlayerIDs(typed[i], replacements, context, playerObject, stringsArePlayerIDs || playerIDCollections[context])
		}
		return typed
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(typed))
		for _, key := range keys {
			newKey := key
			if playerIDKeyedMaps[context] {
				if replacement, ok := replacements[key]; ok {
					newKey = replacement
				}
			}
			childPlayerObject := playerObject || context == "players"
			childStringsArePlayerIDs := stringsArePlayerIDs || playerIDCollections[context] || playerIDCollections[key]
			out[newKey] = normalizePlayerIDs(typed[key], replacements, key, childPlayerObject, childStringsArePlayerIDs)
		}
		return out
	default:
		return value
	}
}

// StateClone is intentionally read-only-by-convention and used by the encoder
// and trajectory materializer without exposing mutable search state.
func (p *GamePosition) StateClone() *game.GameState {
	if p == nil || p.state == nil {
		return nil
	}
	return p.state.CloneForUndo()
}
