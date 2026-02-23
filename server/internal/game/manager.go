package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/models"
)

// CreateGameOptions controls game creation behavior.
type CreateGameOptions struct {
	RandomizeTurnOrder bool
	SetupMode          SetupMode
}

// ActionMeta provides metadata for action execution.
type ActionMeta struct {
	ActionID         string
	ExpectedRevision int
	SeatID           string
}

// ActionResult reports action execution outcome.
type ActionResult struct {
	Revision  int
	Duplicate bool
}

// RevisionMismatchError indicates stale optimistic concurrency data.
type RevisionMismatchError struct {
	Expected int
	Current  int
}

func (e *RevisionMismatchError) Error() string {
	return fmt.Sprintf("revision mismatch: expected %d, current %d", e.Expected, e.Current)
}

// Manager handles multiple in-memory game instances.
type Manager struct {
	mu              sync.RWMutex
	games           map[string]*GameState
	revisions       map[string]int
	appliedActionID map[string]map[string]int
}

// NewManager creates a new game manager.
func NewManager() *Manager {
	return &Manager{
		games:           make(map[string]*GameState),
		revisions:       make(map[string]int),
		appliedActionID: make(map[string]map[string]int),
	}
}

// CreateGameWithState creates a game with an existing GameState.
func (m *Manager) CreateGameWithState(id string, gs *GameState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.games[id] = gs
	m.revisions[id] = 0
	m.appliedActionID[id] = make(map[string]int)
}

// GetGame retrieves a game by ID.
func (m *Manager) GetGame(id string) (*GameState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.games[id]
	return g, ok
}

// GetRevision returns the current revision for a game.
func (m *Manager) GetRevision(id string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.games[id]
	if !ok {
		return 0, false
	}
	return m.revisions[id], true
}

// ListGames returns all active games.
func (m *Manager) ListGames() []*GameState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*GameState, 0, len(m.games))
	for _, g := range m.games {
		out = append(out, g)
	}
	return out
}

// ExecuteAction executes an action in the specified game (legacy API).
func (m *Manager) ExecuteAction(gameID string, action Action) error {
	_, err := m.ExecuteActionWithMeta(gameID, action, ActionMeta{ExpectedRevision: -1})
	return err
}

// ExecuteActionWithMeta executes an action with revision/idempotency checks.
func (m *Manager) ExecuteActionWithMeta(gameID string, action Action, meta ActionMeta) (*ActionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	gs := m.games[gameID]
	if gs == nil {
		return nil, fmt.Errorf("game %s not found", gameID)
	}

	currentRevision := m.revisions[gameID]
	if meta.ActionID != "" {
		if _, exists := m.appliedActionID[gameID][meta.ActionID]; exists {
			return &ActionResult{Revision: currentRevision, Duplicate: true}, nil
		}
	}

	if meta.ExpectedRevision >= 0 && meta.ExpectedRevision != currentRevision {
		return nil, &RevisionMismatchError{Expected: meta.ExpectedRevision, Current: currentRevision}
	}

	if err := validateActionTurnAndPendingState(gs, action); err != nil {
		return nil, fmt.Errorf("action turn validation failed: %w", err)
	}

	if err := action.Validate(gs); err != nil {
		return nil, fmt.Errorf("action validation failed: %w", err)
	}

	if err := action.Execute(gs); err != nil {
		return nil, fmt.Errorf("action execution failed: %w", err)
	}

	currentRevision++
	m.revisions[gameID] = currentRevision
	if meta.ActionID != "" {
		if m.appliedActionID[gameID] == nil {
			m.appliedActionID[gameID] = make(map[string]int)
		}
		m.appliedActionID[gameID][meta.ActionID] = currentRevision
	}

	return &ActionResult{Revision: currentRevision, Duplicate: false}, nil
}

func validateActionTurnAndPendingState(gs *GameState, action Action) error {
	actionType := action.GetType()
	playerID := action.GetPlayerID()

	if gs.HasPendingLeechOffers() {
		expected := gs.GetNextLeechResponder()
		if actionType != ActionAcceptPowerLeech && actionType != ActionDeclinePowerLeech {
			return fmt.Errorf("leech response pending for player %s", expected)
		}
		if expected != "" && expected != playerID {
			return fmt.Errorf("leech response required from player %s", expected)
		}
		return nil
	}

	if gs.PendingCultistsCultSelection != nil {
		if actionType != ActionSelectCultistsCultTrack {
			return fmt.Errorf("cultists cult selection pending for player %s", gs.PendingCultistsCultSelection.PlayerID)
		}
		if playerID != gs.PendingCultistsCultSelection.PlayerID {
			return fmt.Errorf("cultists cult selection required from player %s", gs.PendingCultistsCultSelection.PlayerID)
		}
		return nil
	}

	if gs.PendingFavorTileSelection != nil {
		if actionType != ActionSelectFavorTile {
			return fmt.Errorf("favor tile selection pending for player %s", gs.PendingFavorTileSelection.PlayerID)
		}
		if playerID != gs.PendingFavorTileSelection.PlayerID {
			return fmt.Errorf("favor tile selection required from player %s", gs.PendingFavorTileSelection.PlayerID)
		}
		return nil
	}

	if gs.PendingDarklingsPriestOrdination != nil {
		if actionType != ActionUseDarklingsPriestOrdination {
			return fmt.Errorf("darklings priest ordination pending for player %s", gs.PendingDarklingsPriestOrdination.PlayerID)
		}
		if playerID != gs.PendingDarklingsPriestOrdination.PlayerID {
			return fmt.Errorf("darklings priest ordination required from player %s", gs.PendingDarklingsPriestOrdination.PlayerID)
		}
		return nil
	}

	if gs.PendingHalflingsSpades != nil {
		if playerID != gs.PendingHalflingsSpades.PlayerID {
			return fmt.Errorf("halflings spade follow-up required from player %s", gs.PendingHalflingsSpades.PlayerID)
		}
		if actionType != ActionApplyHalflingsSpade && actionType != ActionBuildHalflingsDwelling && actionType != ActionSkipHalflingsDwelling {
			return fmt.Errorf("halflings spade follow-up pending for player %s", gs.PendingHalflingsSpades.PlayerID)
		}
		return nil
	}

	if requiredPlayer, _ := gs.GetPendingSpadeFollowupPlayer(); requiredPlayer != "" {
		if playerID != requiredPlayer {
			return fmt.Errorf("spade follow-up required from player %s", requiredPlayer)
		}
		if actionType != ActionTransformAndBuild && actionType != ActionDiscardPendingSpade {
			return fmt.Errorf("spade follow-up pending for player %s", requiredPlayer)
		}
		return nil
	}

	if requiredPlayer, _ := gs.GetPendingCultRewardSpadePlayer(); requiredPlayer != "" {
		if playerID != requiredPlayer {
			return fmt.Errorf("cult reward spade follow-up required from player %s", requiredPlayer)
		}
		if actionType != ActionUseCultSpade {
			return fmt.Errorf("cult reward spade follow-up pending for player %s", requiredPlayer)
		}
		return nil
	}

	if gs.PendingTownCultTopChoice != nil {
		if actionType != ActionSelectTownCultTop {
			return fmt.Errorf("town cult-top choice pending for player %s", gs.PendingTownCultTopChoice.PlayerID)
		}
		if playerID != gs.PendingTownCultTopChoice.PlayerID {
			return fmt.Errorf("town cult-top choice required from player %s", gs.PendingTownCultTopChoice.PlayerID)
		}
		return nil
	}

	if townPlayer := gs.GetPendingTownSelectionPlayer(); townPlayer != "" {
		if actionType != ActionSelectTownTile {
			return fmt.Errorf("town tile selection pending for player %s", townPlayer)
		}
		if playerID != townPlayer {
			return fmt.Errorf("town tile selection required from player %s", townPlayer)
		}
		return nil
	}

	if actionType == ActionSelectTownTile {
		if pendingTowns, ok := gs.PendingTownFormations[playerID]; ok && len(pendingTowns) > 0 {
			return nil
		}
		return fmt.Errorf("no pending town formation for player %s", playerID)
	}

	if gs.Phase == PhaseFactionSelection {
		switch gs.SetupMode {
		case SetupModeAuction:
			if gs.AuctionState == nil || !gs.AuctionState.Active {
				return fmt.Errorf("regular auction setup is not active")
			}
			if gs.AuctionState.NominationPhase {
				if actionType != ActionAuctionNominateFaction {
					return fmt.Errorf("auction nomination is required")
				}
			} else if actionType != ActionAuctionPlaceBid {
				return fmt.Errorf("regular auction bid is required")
			}
		case SetupModeFastAuction:
			if gs.AuctionState == nil || !gs.AuctionState.Active {
				return fmt.Errorf("fast auction setup is not active")
			}
			if gs.AuctionState.NominationPhase {
				if actionType != ActionAuctionNominateFaction {
					return fmt.Errorf("auction nomination is required")
				}
			} else if actionType != ActionFastAuctionSubmitBids {
				return fmt.Errorf("fast auction bid submission is required")
			}
		default:
			if actionType != ActionSelectFaction {
				return fmt.Errorf("faction selection is required")
			}
		}
	}

	if gs.Phase == PhaseSetup && gs.SetupSubphase == SetupSubphaseBonusCards && actionType != ActionSetupBonusCard {
		return fmt.Errorf("setup bonus card selection is required")
	}

	if actionType == ActionAcceptPowerLeech || actionType == ActionDeclinePowerLeech {
		return fmt.Errorf("no pending leech offer for player")
	}

	if actionType == ActionSelectTownCultTop || actionType == ActionSelectFavorTile || actionType == ActionUseDarklingsPriestOrdination || actionType == ActionApplyHalflingsSpade || actionType == ActionBuildHalflingsDwelling || actionType == ActionSkipHalflingsDwelling || actionType == ActionSelectCultistsCultTrack || actionType == ActionDiscardPendingSpade {
		return fmt.Errorf("no pending decision for requested action")
	}

	if actionRequiresTurnOwnership(actionType) {
		current := gs.GetCurrentPlayer()
		if current == nil {
			return fmt.Errorf("no current player")
		}
		if current.ID != playerID {
			return fmt.Errorf("not your turn")
		}
	}

	return nil
}

func actionRequiresTurnOwnership(actionType ActionType) bool {
	switch actionType {
	case ActionAcceptPowerLeech,
		ActionDeclinePowerLeech,
		ActionSelectFavorTile,
		ActionSelectTownTile,
		ActionSelectTownCultTop,
		ActionUseDarklingsPriestOrdination,
		ActionApplyHalflingsSpade,
		ActionBuildHalflingsDwelling,
		ActionSkipHalflingsDwelling,
		ActionSetupBonusCard,
		ActionSelectCultistsCultTrack,
		ActionDiscardPendingSpade:
		return false
	default:
		return true
	}
}

// CreateGame initializes a new game state with the given ID and players.
func (m *Manager) CreateGame(id string, playerIDs []string) error {
	return m.CreateGameWithOptions(id, playerIDs, CreateGameOptions{RandomizeTurnOrder: true, SetupMode: SetupModeSnellman})
}

// CreateGameWithOptions initializes a new game with explicit options.
func (m *Manager) CreateGameWithOptions(id string, playerIDs []string, opts CreateGameOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.games[id]; exists {
		return fmt.Errorf("game already exists")
	}

	gs := NewGameState()
	setupMode := opts.SetupMode
	if setupMode == "" {
		setupMode = SetupModeSnellman
	}
	switch setupMode {
	case SetupModeSnellman, SetupModeAuction, SetupModeFastAuction:
	default:
		return fmt.Errorf("invalid setup mode: %s", setupMode)
	}
	gs.SetupMode = setupMode

	if err := gs.ScoringTiles.InitializeForGame(); err != nil {
		return fmt.Errorf("failed to initialize scoring tiles: %w", err)
	}

	gs.BonusCards.SelectRandomBonusCards(len(playerIDs))

	turnOrder := make([]string, len(playerIDs))
	copy(turnOrder, playerIDs)
	if opts.RandomizeTurnOrder {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		rng.Shuffle(len(turnOrder), func(i, j int) {
			turnOrder[i], turnOrder[j] = turnOrder[j], turnOrder[i]
		})
	}

	for _, pid := range turnOrder {
		if err := gs.AddPlayer(pid, nil); err != nil {
			return fmt.Errorf("failed to add player %s: %w", pid, err)
		}
	}

	gs.Phase = PhaseFactionSelection
	gs.TurnOrder = turnOrder
	gs.CurrentPlayerIndex = 0
	if setupMode == SetupModeAuction || setupMode == SetupModeFastAuction {
		gs.AuctionState = NewAuctionStateWithMode(turnOrder, setupMode)
	}

	m.games[id] = gs
	m.revisions[id] = 0
	m.appliedActionID[id] = make(map[string]int)
	return nil
}

// SerializeGameState converts GameState to a JSON-friendly format for the frontend.
func (m *Manager) SerializeGameState(gameID string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gs := m.games[gameID]
	if gs == nil {
		return nil
	}
	return SerializeStateWithRevision(gs, gameID, m.revisions[gameID])
}

// SerializeState converts the game state to a map for JSON response.
func SerializeState(gs *GameState, gameID string) map[string]interface{} {
	return SerializeStateWithRevision(gs, gameID, 0)
}

// SerializeStateWithRevision converts game state to JSON-friendly map including revision.
func SerializeStateWithRevision(gs *GameState, gameID string, revision int) map[string]interface{} {
	players := make(map[string]interface{})
	for playerID, player := range gs.Players {
		var factionType models.FactionType
		if player.Faction != nil {
			factionType = player.Faction.GetType()
		}

		players[playerID] = map[string]interface{}{
			"id":      playerID,
			"name":    playerID,
			"faction": factionType,
			"resources": map[string]interface{}{
				"coins":   player.Resources.Coins,
				"workers": player.Resources.Workers,
				"priests": player.Resources.Priests,
				"power": map[string]interface{}{
					"powerI":   player.Resources.Power.Bowl1,
					"powerII":  player.Resources.Power.Bowl2,
					"powerIII": player.Resources.Power.Bowl3,
				},
			},
			"shipping":             player.ShippingLevel,
			"digging":              player.DiggingLevel,
			"hasPassed":            player.HasPassed,
			"hasStrongholdAbility": player.HasStrongholdAbility,
			"victoryPoints":        player.VictoryPoints,
			"keys":                 player.Keys,
			"townsFormed":          player.TownsFormed,
			"townTiles":            player.TownTiles,
			"specialActionsUsed":   player.SpecialActionsUsed,
			"cults": map[string]interface{}{
				"0": player.CultPositions[CultFire],
				"1": player.CultPositions[CultWater],
				"2": player.CultPositions[CultEarth],
				"3": player.CultPositions[CultAir],
			},
		}
	}

	hexes := make(map[string]interface{})
	for _, mapHex := range gs.Map.Hexes {
		key := fmt.Sprintf("%d,%d", mapHex.Coord.Q, mapHex.Coord.R)
		hexData := map[string]interface{}{
			"coord": map[string]int{
				"q": mapHex.Coord.Q,
				"r": mapHex.Coord.R,
			},
			"terrain": mapHex.Terrain,
		}

		if mapHex.Building != nil {
			hexData["building"] = map[string]interface{}{
				"ownerPlayerId": mapHex.Building.PlayerID,
				"faction":       mapHex.Building.Faction,
				"type":          mapHex.Building.Type,
			}
		}

		hexes[key] = hexData
	}

	bridges := make([]map[string]interface{}, 0)
	for bridgeKey, ownerID := range gs.Map.Bridges {
		var factionType models.FactionType
		if p, ok := gs.Players[ownerID]; ok && p.Faction != nil {
			factionType = p.Faction.GetType()
		}

		bridges = append(bridges, map[string]interface{}{
			"ownerPlayerId": ownerID,
			"faction":       factionType,
			"fromCoord": map[string]int{
				"q": bridgeKey.H1.Q,
				"r": bridgeKey.H1.R,
			},
			"toCoord": map[string]int{
				"q": bridgeKey.H2.Q,
				"r": bridgeKey.H2.R,
			},
		})
	}

	return map[string]interface{}{
		"id":                   gameID,
		"revision":             revision,
		"phase":                gs.Phase,
		"setupMode":            gs.SetupMode,
		"setupSubphase":        gs.SetupSubphase,
		"setupDwellingOrder":   gs.SetupDwellingOrder,
		"setupDwellingIndex":   gs.SetupDwellingIndex,
		"setupBonusOrder":      gs.SetupBonusOrder,
		"setupBonusIndex":      gs.SetupBonusIndex,
		"setupPlacedDwellings": gs.SetupPlacedDwellings,
		"currentTurn":          gs.CurrentPlayerIndex,
		"players":              players,
		"map": map[string]interface{}{
			"hexes":   hexes,
			"bridges": bridges,
		},
		"turnOrder": gs.TurnOrder,
		"passOrder": gs.PassOrder,
		"round": map[string]interface{}{
			"round": gs.Round,
		},
		"started":  gs.Phase != PhaseSetup,
		"finished": gs.Phase == PhaseEnd,
		"scoringTiles": func() interface{} {
			if gs.ScoringTiles == nil {
				return nil
			}
			return gs.ScoringTiles
		}(),
		"bonusCards": func() interface{} {
			if gs.BonusCards == nil {
				return nil
			}
			return gs.BonusCards
		}(),
		"townTiles": func() interface{} {
			if gs.TownTiles == nil {
				return nil
			}
			return gs.TownTiles
		}(),
		"favorTiles": func() interface{} {
			if gs.FavorTiles == nil {
				return nil
			}
			return map[string]interface{}{
				"available":   gs.FavorTiles.Available,
				"playerTiles": gs.FavorTiles.PlayerTiles,
			}
		}(),
		"powerActions": func() interface{} {
			if gs.PowerActions == nil {
				return nil
			}
			return gs.PowerActions
		}(),
		"cultTracks": func() interface{} {
			if gs.CultTracks == nil {
				return nil
			}
			return gs.CultTracks
		}(),
		"pendingLeechOffers":               gs.PendingLeechOffers,
		"pendingTownFormations":            gs.PendingTownFormations,
		"pendingSpades":                    gs.PendingSpades,
		"pendingSpadeBuildAllowed":         gs.PendingSpadeBuildAllowed,
		"pendingCultRewardSpades":          gs.PendingCultRewardSpades,
		"pendingFavorTileSelection":        gs.PendingFavorTileSelection,
		"pendingHalflingsSpades":           gs.PendingHalflingsSpades,
		"pendingDarklingsPriestOrdination": gs.PendingDarklingsPriestOrdination,
		"pendingCultistsCultSelection":     gs.PendingCultistsCultSelection,
		"pendingTownCultTopChoice":         gs.PendingTownCultTopChoice,
		"pendingDecision":                  serializePendingDecision(gs),
		"auctionState":                     serializeAuctionState(gs.AuctionState),
		"finalScoring": func() interface{} {
			if gs.FinalScoring == nil {
				return nil
			}
			return gs.FinalScoring
		}(),
	}
}

func serializePendingDecision(gs *GameState) interface{} {
	if gs == nil {
		return nil
	}

	if gs.Phase == PhaseFactionSelection && gs.AuctionState != nil && gs.AuctionState.Active {
		factions := make([]string, 0, len(gs.AuctionState.NominationOrder))
		for _, faction := range gs.AuctionState.NominationOrder {
			factions = append(factions, faction.String())
		}

		if gs.AuctionState.NominationPhase {
			return map[string]interface{}{
				"type":              "auction_nomination",
				"playerId":          gs.AuctionState.GetCurrentBidder(),
				"setupMode":         gs.SetupMode,
				"nominatedFactions": factions,
			}
		}

		if gs.SetupMode == SetupModeFastAuction {
			return map[string]interface{}{
				"type":              "fast_auction_bid_matrix",
				"playerId":          gs.AuctionState.GetCurrentBidder(),
				"setupMode":         gs.SetupMode,
				"nominatedFactions": factions,
			}
		}

		return map[string]interface{}{
			"type":              "auction_bid",
			"playerId":          gs.AuctionState.GetCurrentBidder(),
			"setupMode":         gs.SetupMode,
			"nominatedFactions": factions,
		}
	}

	if gs.Phase == PhaseSetup && gs.SetupSubphase == SetupSubphaseBonusCards {
		if playerID := gs.currentSetupBonusPlayerID(); playerID != "" {
			return map[string]interface{}{
				"type":     "setup_bonus_card",
				"playerId": playerID,
			}
		}
	}

	if gs.HasPendingLeechOffers() {
		if playerID := gs.GetNextLeechResponder(); playerID != "" {
			offers := gs.PendingLeechOffers[playerID]
			return map[string]interface{}{
				"type":     "leech_offer",
				"playerId": playerID,
				"offers":   offers,
			}
		}
	}

	if gs.PendingCultistsCultSelection != nil {
		return map[string]interface{}{
			"type":     "cultists_cult_choice",
			"playerId": gs.PendingCultistsCultSelection.PlayerID,
		}
	}

	if gs.PendingFavorTileSelection != nil {
		return map[string]interface{}{
			"type":     "favor_tile_selection",
			"playerId": gs.PendingFavorTileSelection.PlayerID,
			"count":    gs.PendingFavorTileSelection.Count,
		}
	}

	if gs.PendingDarklingsPriestOrdination != nil {
		return map[string]interface{}{
			"type":     "darklings_ordination",
			"playerId": gs.PendingDarklingsPriestOrdination.PlayerID,
		}
	}

	if gs.PendingHalflingsSpades != nil {
		return map[string]interface{}{
			"type":            "halflings_spades",
			"playerId":        gs.PendingHalflingsSpades.PlayerID,
			"spadesRemaining": gs.PendingHalflingsSpades.SpadesRemaining,
		}
	}

	if playerID, count := gs.GetPendingSpadeFollowupPlayer(); playerID != "" {
		canBuildDwelling := true
		if allowed, ok := gs.PendingSpadeBuildAllowed[playerID]; ok {
			canBuildDwelling = allowed
		}
		return map[string]interface{}{
			"type":             "spade_followup",
			"playerId":         playerID,
			"spadesRemaining":  count,
			"canBuildDwelling": canBuildDwelling,
		}
	}

	if playerID, count := gs.GetPendingCultRewardSpadePlayer(); playerID != "" {
		return map[string]interface{}{
			"type":            "cult_reward_spade",
			"playerId":        playerID,
			"spadesRemaining": count,
		}
	}

	if gs.PendingTownCultTopChoice != nil {
		return map[string]interface{}{
			"type":            "town_cult_top_choice",
			"playerId":        gs.PendingTownCultTopChoice.PlayerID,
			"candidateTracks": gs.PendingTownCultTopChoice.CandidateTracks,
			"maxSelections":   gs.PendingTownCultTopChoice.MaxSelections,
			"advanceAmount":   gs.PendingTownCultTopChoice.AdvanceAmount,
		}
	}

	if townPlayer := gs.GetPendingTownSelectionPlayer(); townPlayer != "" {
		return map[string]interface{}{
			"type":     "town_tile_selection",
			"playerId": townPlayer,
		}
	}

	return nil
}

func serializeAuctionState(as *AuctionState) interface{} {
	if as == nil {
		return nil
	}

	nominationOrder := make([]string, 0, len(as.NominationOrder))
	for _, faction := range as.NominationOrder {
		nominationOrder = append(nominationOrder, faction.String())
	}

	currentBids := make(map[string]int)
	for faction, bid := range as.CurrentBids {
		currentBids[faction.String()] = bid
	}

	factionHolders := make(map[string]string)
	for faction, playerID := range as.FactionHolders {
		factionHolders[faction.String()] = playerID
	}

	fastBids := make(map[string]map[string]int)
	for playerID, bids := range as.FastBids {
		playerBids := make(map[string]int)
		for faction, bid := range bids {
			playerBids[faction.String()] = bid
		}
		fastBids[playerID] = playerBids
	}

	return map[string]interface{}{
		"active":              as.Active,
		"mode":                as.Mode,
		"nominationPhase":     as.NominationPhase,
		"currentBidder":       as.GetCurrentBidder(),
		"currentBidderIndex":  as.CurrentBidderIndex,
		"nominationsComplete": as.NominationsComplete,
		"nominationOrder":     nominationOrder,
		"currentBids":         currentBids,
		"factionHolders":      factionHolders,
		"seatOrder":           as.SeatOrder,
		"playerHasFaction":    as.PlayerHasFaction,
		"fastSubmitted":       as.FastSubmitted,
		"fastBids":            fastBids,
	}
}
