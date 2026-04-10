package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// CreateGameOptions controls game creation behavior.
type CreateGameOptions struct {
	RandomizeTurnOrder bool
	SetupMode          SetupMode
	TurnTimer          *TurnTimerConfig
	MapID              board.MapID
	EnableFanFactions  bool
	CustomMap          *board.CustomMapDefinition
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
	now             func() time.Time
}

// NewManager creates a new game manager.
func NewManager() *Manager {
	return &Manager{
		games:           make(map[string]*GameState),
		revisions:       make(map[string]int),
		appliedActionID: make(map[string]map[string]int),
		now:             time.Now,
	}
}

// CreateGameWithState creates a game with an existing GameState.
func (m *Manager) CreateGameWithState(id string, gs *GameState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gs != nil && gs.TurnTimer != nil {
		gs.TurnTimer.SyncActivePlayers(activeDecisionPlayerIDs(gs), m.now())
	}
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

// ApplyFixtureSettings sets scoring and bonus card availability for a game and bumps revision.
// This is intended for deterministic integration/golden automation paths.
func (m *Manager) ApplyFixtureSettings(gameID string, scoringTiles []ScoringTile, bonusCards []BonusCardType, turnOrderPolicy TurnOrderPolicy) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	gs := m.games[gameID]
	if gs == nil {
		return 0, fmt.Errorf("game %s not found", gameID)
	}

	gs.ScoringTiles.Tiles = append([]ScoringTile(nil), scoringTiles...)
	gs.BonusCards.SetAvailableBonusCards(bonusCards)
	if turnOrderPolicy != "" {
		gs.TurnOrderPolicy = turnOrderPolicy
	}

	nextRevision := m.revisions[gameID] + 1
	m.revisions[gameID] = nextRevision
	return nextRevision, nil
}

// ApplyConversionWithoutTurnCheck applies a conversion action for test replay and
// UI automation paths without requiring player turn ownership.
func (m *Manager) ApplyConversionWithoutTurnCheck(gameID, playerID string, conversionType ConversionType, amount int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	gs := m.games[gameID]
	if gs == nil {
		return 0, fmt.Errorf("game %s not found", gameID)
	}

	action := &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: playerID},
		ConversionType: conversionType,
		Amount:         amount,
	}

	if err := action.Validate(gs); err != nil {
		return 0, fmt.Errorf("conversion validation failed: %w", err)
	}

	if err := action.Execute(gs); err != nil {
		return 0, fmt.Errorf("conversion execution failed: %w", err)
	}

	if err := gs.ResolveAutoLeechOffers(); err != nil {
		return 0, fmt.Errorf("auto leech resolution failed: %w", err)
	}

	currentRevision := m.revisions[gameID]
	nextRevision := currentRevision + 1
	m.revisions[gameID] = nextRevision
	return nextRevision, nil
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
	now := m.now()
	if gs.TurnTimer != nil {
		gs.TurnTimer.ChargeActivePlayers(now)
	}

	currentRevision := m.revisions[gameID]
	beforeTurn := captureTurnProgress(gs)
	undoSnapshot := gs.CloneForUndo()
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

	maybeExpirePendingFreeActionsWindow(gs, action)

	if err := action.Validate(gs); err != nil {
		return nil, fmt.Errorf("action validation failed: %w", err)
	}

	if err := action.Execute(gs); err != nil {
		return nil, fmt.Errorf("action execution failed: %w", err)
	}
	if err := gs.ResolveAutoLeechOffers(); err != nil {
		return nil, fmt.Errorf("auto leech resolution failed: %w", err)
	}
	updatePendingFreeActionsWindow(gs, action)
	stageTurnConfirmation(gs, action, beforeTurn, undoSnapshot)
	syncTurnConfirmationPreferences(gs, action)
	refreshTurnConfirmationUndoCheckpoint(gs, action)
	if gs.TurnTimer != nil {
		gs.TurnTimer.SyncActivePlayers(activeDecisionPlayerIDs(gs), now)
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
	if actionType == ActionConfirmTurn || actionType == ActionUndoTurn {
		if !gs.HasPendingTurnConfirmation() {
			return fmt.Errorf("no pending turn confirmation")
		}
		if strings.TrimSpace(playerID) != strings.TrimSpace(gs.PendingTurnConfirmationPlayerID) {
			return fmt.Errorf("turn confirmation pending for player %s", gs.PendingTurnConfirmationPlayerID)
		}
		return nil
	}

	if actionType == ActionSetPlayerOptions {
		if gs.GetPlayer(playerID) == nil {
			return fmt.Errorf("player not found")
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

	if gs.PendingDarklingsPriestOrdination != nil {
		if actionType != ActionUseDarklingsPriestOrdination {
			return fmt.Errorf("darklings priest ordination pending for player %s", gs.PendingDarklingsPriestOrdination.PlayerID)
		}
		if playerID != gs.PendingDarklingsPriestOrdination.PlayerID {
			return fmt.Errorf("darklings priest ordination required from player %s", gs.PendingDarklingsPriestOrdination.PlayerID)
		}
		return nil
	}

	if gs.HasPendingLeechOffers() {
		expected := gs.GetNextBlockingLeechResponder()
		if actionType == ActionAcceptPowerLeech || actionType == ActionDeclinePowerLeech {
			if len(gs.PendingLeechOffers[playerID]) == 0 {
				if expected != "" {
					return fmt.Errorf("leech response required from player %s", expected)
				}
				return fmt.Errorf("no pending leech offer for player %s", playerID)
			}
			return nil
		}
		if expected != "" {
			if canCurrentPlayerContinueFreeActionBeforeLeech(gs, action) {
				return nil
			}
			return fmt.Errorf("leech response pending for player %s", expected)
		}
	}

	if gs.PendingCultistsCultSelection != nil {
		pendingPlayer := strings.TrimSpace(gs.PendingCultistsCultSelection.PlayerID)
		if strings.TrimSpace(playerID) == pendingPlayer {
			if actionType != ActionSelectCultistsCultTrack {
				return fmt.Errorf("cultists cult selection pending for player %s", gs.PendingCultistsCultSelection.PlayerID)
			}
			return nil
		}
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

	if gs.PendingHalflingsSpades != nil {
		if playerID != gs.PendingHalflingsSpades.PlayerID {
			return fmt.Errorf("halflings spade follow-up required from player %s", gs.PendingHalflingsSpades.PlayerID)
		}
		if actionType != ActionApplyHalflingsSpade && actionType != ActionBuildHalflingsDwelling && actionType != ActionSkipHalflingsDwelling {
			return fmt.Errorf("halflings spade follow-up pending for player %s", gs.PendingHalflingsSpades.PlayerID)
		}
		return nil
	}

	if gs.PendingWispsStrongholdDwelling != nil {
		if playerID != gs.PendingWispsStrongholdDwelling.PlayerID {
			return fmt.Errorf("wisps stronghold dwelling required from player %s", gs.PendingWispsStrongholdDwelling.PlayerID)
		}
		if actionType != ActionBuildWispsStrongholdDwelling {
			return fmt.Errorf("wisps stronghold dwelling pending for player %s", gs.PendingWispsStrongholdDwelling.PlayerID)
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
		if actionType != ActionUseCultSpade && actionType != ActionDiscardPendingSpade {
			return fmt.Errorf("cult reward spade follow-up pending for player %s", requiredPlayer)
		}
		return nil
	}

	if actionType == ActionSelectTownTile {
		if pendingTowns, ok := gs.PendingTownFormations[playerID]; ok && len(pendingTowns) > 0 {
			return nil
		}
		return fmt.Errorf("no pending town formation for player %s", playerID)
	}

	if pendingPlayerID := strings.TrimSpace(gs.PendingFreeActionsPlayerID); pendingPlayerID != "" &&
		strings.TrimSpace(playerID) == pendingPlayerID {
		if canPlayerUsePendingFreeActionsWindow(gs, action) {
			return nil
		}
		current := gs.GetCurrentPlayer()
		if current != nil && strings.TrimSpace(current.ID) == pendingPlayerID {
			return fmt.Errorf("post-action free actions pending for player %s", pendingPlayerID)
		}
	}

	if canPlayerUsePendingFreeActionsWindow(gs, action) {
		return nil
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

	if actionType == ActionSelectTownCultTop || actionType == ActionSelectFavorTile || actionType == ActionUseDarklingsPriestOrdination || actionType == ActionApplyHalflingsSpade || actionType == ActionBuildHalflingsDwelling || actionType == ActionSkipHalflingsDwelling || actionType == ActionBuildWispsStrongholdDwelling || actionType == ActionSelectCultistsCultTrack || actionType == ActionDiscardPendingSpade {
		return fmt.Errorf("no pending decision for requested action")
	}

	if pendingPlayerID := strings.TrimSpace(gs.PendingTurnConfirmationPlayerID); pendingPlayerID != "" {
		if strings.TrimSpace(playerID) == pendingPlayerID {
			if canPlayerUsePendingFreeActionsWindow(gs, action) || actionType == ActionSetPlayerOptions {
				return nil
			}
			return fmt.Errorf("turn confirmation pending for player %s", pendingPlayerID)
		}
		return fmt.Errorf("turn confirmation pending for player %s", pendingPlayerID)
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

func canCurrentPlayerContinueFreeActionBeforeLeech(gs *GameState, action Action) bool {
	if gs == nil || action == nil {
		return false
	}
	current := gs.GetCurrentPlayer()
	if current == nil || current.ID != action.GetPlayerID() {
		return false
	}
	switch action.GetType() {
	case ActionConversion, ActionBurnPower:
		return true
	default:
		return false
	}
}

func canPlayerUsePendingFreeActionsWindow(gs *GameState, action Action) bool {
	if gs == nil || action == nil {
		return false
	}
	pendingPlayerID := strings.TrimSpace(gs.PendingFreeActionsPlayerID)
	if pendingPlayerID == "" || strings.TrimSpace(action.GetPlayerID()) != pendingPlayerID {
		return false
	}
	switch action.GetType() {
	case ActionConversion, ActionBurnPower:
		return true
	default:
		return false
	}
}

func maybeExpirePendingFreeActionsWindow(gs *GameState, action Action) {
	if gs == nil || action == nil {
		return
	}
	pendingPlayerID := strings.TrimSpace(gs.PendingFreeActionsPlayerID)
	if pendingPlayerID == "" {
		return
	}
	playerID := strings.TrimSpace(action.GetPlayerID())
	if playerID == "" {
		return
	}
	if playerID == pendingPlayerID && canPlayerUsePendingFreeActionsWindow(gs, action) {
		return
	}
	if isPendingResolutionActionType(action.GetType()) {
		return
	}
	current := gs.GetCurrentPlayer()
	if current == nil {
		return
	}
	if strings.TrimSpace(current.ID) == pendingPlayerID && playerID != pendingPlayerID {
		gs.PendingFreeActionsPlayerID = ""
		gs.advanceToNextPlayer()
		return
	}
	if strings.TrimSpace(current.ID) != playerID {
		return
	}
	gs.PendingFreeActionsPlayerID = ""
}

func updatePendingFreeActionsWindow(gs *GameState, action Action) {
	if gs == nil {
		return
	}
	if gs.Phase != PhaseAction {
		gs.PendingFreeActionsPlayerID = ""
		return
	}
	if action == nil {
		return
	}
	if opensPendingFreeActionsWindow(action) {
		playerID := strings.TrimSpace(action.GetPlayerID())
		player := gs.GetPlayer(playerID)
		if player != nil && !player.HasPassed && playerUsesTurnConfirmation(gs, playerID) {
			gs.PendingFreeActionsPlayerID = playerID
			return
		}
	}
	if gs.PendingFreeActionsPlayerID != "" {
		if current := gs.GetCurrentPlayer(); current != nil && strings.TrimSpace(current.ID) == strings.TrimSpace(gs.PendingFreeActionsPlayerID) {
			return
		}
	}
	if action.GetType() == ActionPass {
		gs.PendingFreeActionsPlayerID = ""
	}
}

type turnProgress struct {
	phase         GamePhase
	round         int
	currentTurn   int
	setupSubphase SetupSubphase
}

func captureTurnProgress(gs *GameState) turnProgress {
	if gs == nil {
		return turnProgress{}
	}
	return turnProgress{
		phase:         gs.Phase,
		round:         gs.Round,
		currentTurn:   gs.CurrentPlayerIndex,
		setupSubphase: gs.SetupSubphase,
	}
}

func stageTurnConfirmation(gs *GameState, action Action, before turnProgress, snapshot *GameState) {
	if gs == nil || action == nil || snapshot == nil {
		return
	}
	if action.GetType() == ActionConfirmTurn || action.GetType() == ActionUndoTurn {
		return
	}
	if before.phase != PhaseAction {
		return
	}
	if shouldBeginTurnConfirmation(action) && playerUsesTurnConfirmation(gs, action.GetPlayerID()) && !gs.HasPendingTurnConfirmation() {
		gs.BeginPendingTurnConfirmation(action.GetPlayerID(), snapshot)
	}
}

func syncTurnConfirmationPreferences(gs *GameState, action Action) {
	if gs == nil || action == nil {
		return
	}
	playerID := strings.TrimSpace(action.GetPlayerID())
	if playerID == "" || playerUsesTurnConfirmation(gs, playerID) {
		return
	}
	if strings.TrimSpace(gs.PendingFreeActionsPlayerID) == playerID {
		gs.PendingFreeActionsPlayerID = ""
	}
	if strings.TrimSpace(gs.PendingTurnConfirmationPlayerID) == playerID {
		gs.ClearPendingTurnConfirmation()
	}
}

func refreshTurnConfirmationUndoCheckpoint(gs *GameState, action Action) {
	if gs == nil || action == nil || !gs.HasPendingTurnConfirmation() {
		return
	}
	if action.GetType() != ActionAcceptPowerLeech {
		return
	}
	snapshot := gs.CloneForUndo()
	if snapshot == nil {
		return
	}
	pendingPlayerID := strings.TrimSpace(gs.PendingTurnConfirmationPlayerID)
	if pendingPlayerID != "" {
		snapshot.setCurrentPlayerByID(pendingPlayerID)
	}
	gs.PendingTurnConfirmationSnapshot = snapshot
}

func shouldBeginTurnConfirmation(action Action) bool {
	if action == nil {
		return false
	}
	if opensPendingFreeActionsWindow(action) {
		return true
	}
	return action.GetType() == ActionPass
}

func opensPendingFreeActionsWindow(action Action) bool {
	if action == nil {
		return false
	}
	switch action.GetType() {
	case ActionTransformAndBuild,
		ActionUpgradeBuilding,
		ActionAdvanceShipping,
		ActionAdvanceDigging,
		ActionAdvanceChashTrack,
		ActionSendPriestToCult,
		ActionPowerAction,
		ActionSpecialAction,
		ActionEngineersBridge:
		return true
	default:
		return false
	}
}

func playerUsesTurnConfirmation(gs *GameState, playerID string) bool {
	if gs == nil {
		return false
	}
	player := gs.GetPlayer(strings.TrimSpace(playerID))
	if player == nil {
		return false
	}
	return player.Options.ConfirmActions
}

func isPendingResolutionActionType(actionType ActionType) bool {
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
		ActionBuildWispsStrongholdDwelling,
		ActionSelectCultistsCultTrack,
		ActionUseCultSpade,
		ActionDiscardPendingSpade,
		ActionSetupBonusCard,
		ActionSetPlayerOptions:
		return true
	default:
		return false
	}
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
		ActionBuildWispsStrongholdDwelling,
		ActionSetupBonusCard,
		ActionSelectCultistsCultTrack,
		ActionDiscardPendingSpade,
		ActionSetPlayerOptions,
		ActionConfirmTurn,
		ActionUndoTurn,
		ActionFastAuctionSubmitBids:
		return false
	default:
		return true
	}
}

// CreateGame initializes a new game state with the given ID and players.
func (m *Manager) CreateGame(id string, playerIDs []string) error {
	return m.CreateGameWithOptions(id, playerIDs, CreateGameOptions{
		RandomizeTurnOrder: true,
		SetupMode:          SetupModeSnellman,
		MapID:              board.MapBase,
	})
}

// CreateGameWithOptions initializes a new game with explicit options.
func (m *Manager) CreateGameWithOptions(id string, playerIDs []string, opts CreateGameOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.games[id]; exists {
		return fmt.Errorf("game already exists")
	}

	mapID := opts.MapID
	if mapID == "" {
		mapID = board.MapBase
	}
	var (
		gs  *GameState
		err error
	)
	if mapID == board.MapCustom {
		gs, err = NewGameStateWithCustomMap(opts.CustomMap)
		if err != nil {
			return fmt.Errorf("failed to initialize custom map: %w", err)
		}
	} else {
		if opts.CustomMap != nil {
			return fmt.Errorf("custom map payload is only valid for map %s", board.MapCustom)
		}
		gs, err = NewGameStateWithMap(mapID)
		if err != nil {
			return fmt.Errorf("failed to initialize map %s: %w", mapID, err)
		}
	}
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
	gs.EnableFanFactions = opts.EnableFanFactions

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
	if opts.TurnTimer != nil {
		gs.TurnTimer = NewTurnTimerState(turnOrder, *opts.TurnTimer)
		if gs.TurnTimer != nil {
			gs.TurnTimer.SyncActivePlayers(activeDecisionPlayerIDs(gs), m.now())
		}
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
	state := serializeStateWithRevisionAt(gs, gameID, m.revisions[gameID], m.now())

	// Detach nested mutable maps/slices while the manager read-lock is held so JSON
	// encoding in websocket handlers does not race with concurrent action writes.
	raw, err := json.Marshal(state)
	if err != nil {
		return state
	}
	var detached map[string]interface{}
	if err := json.Unmarshal(raw, &detached); err != nil {
		return state
	}
	return detached
}

// SerializeState converts the game state to a map for JSON response.
func SerializeState(gs *GameState, gameID string) map[string]interface{} {
	return serializeStateWithRevisionAt(gs, gameID, 0, time.Now())
}

// SerializeStateWithRevision converts game state to JSON-friendly map including revision.
func SerializeStateWithRevision(gs *GameState, gameID string, revision int) map[string]interface{} {
	return serializeStateWithRevisionAt(gs, gameID, revision, time.Now())
}

func serializeStateWithRevisionAt(gs *GameState, gameID string, revision int, now time.Time) map[string]interface{} {
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
			"options": player.Options,
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
			"shipping":              player.ShippingLevel,
			"digging":               player.DiggingLevel,
			"chashIncomeTrackLevel": player.ChashIncomeTrackLevel,
			"hasPassed":             player.HasPassed,
			"hasStrongholdAbility":  player.HasStrongholdAbility,
			"victoryPoints":         player.VictoryPoints,
			"keys":                  player.Keys,
			"townsFormed":           player.TownsFormed,
			"townTiles":             player.TownTiles,
			"specialActionsUsed":    player.SpecialActionsUsed,
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
		if displayCoord, ok := gs.Map.DisplayCoordinateForHex(mapHex.Coord); ok {
			hexData["displayCoord"] = displayCoord
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
		"mapId":                gs.Map.ID,
		"enableFanFactions":    gs.EnableFanFactions,
		"phase":                gs.Phase,
		"setupMode":            gs.SetupMode,
		"turnOrderPolicy":      gs.TurnOrderPolicy,
		"setupSubphase":        gs.SetupSubphase,
		"setupDwellingOrder":   gs.SetupDwellingOrder,
		"setupDwellingIndex":   gs.SetupDwellingIndex,
		"setupBonusOrder":      gs.SetupBonusOrder,
		"setupBonusIndex":      gs.SetupBonusIndex,
		"setupPlacedDwellings": gs.SetupPlacedDwellings,
		"currentTurn":          gs.CurrentPlayerIndex,
		"players":              players,
		"map": map[string]interface{}{
			"id":      gs.Map.ID,
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
		"pendingFreeActionsPlayerId":       gs.PendingFreeActionsPlayerID,
		"pendingTurnConfirmationPlayerId":  gs.PendingTurnConfirmationPlayerID,
		"pendingDecision":                  serializePendingDecision(gs),
		"auctionState":                     serializeAuctionState(gs.AuctionState),
		"turnTimer":                        serializeTurnTimer(gs.TurnTimer, now),
		"nextRoundIncome":                  serializeNextRoundIncomePreview(gs),
		"finalScoring": func() interface{} {
			if gs.FinalScoring == nil {
				return nil
			}
			return gs.FinalScoring
		}(),
	}
}

func serializeNextRoundIncomePreview(gs *GameState) interface{} {
	if gs == nil || gs.Round < 1 || gs.Round > 5 {
		return nil
	}
	preview := make(map[string]IncomePreview)
	for playerID := range gs.Players {
		if income, ok := gs.GetNextRoundIncomePreview(playerID); ok {
			preview[playerID] = income
		}
	}
	return preview
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
			pendingPlayers := gs.AuctionState.GetPendingFastSubmitters()
			return map[string]interface{}{
				"type":              "fast_auction_bid_matrix",
				"playerId":          gs.AuctionState.GetCurrentBidder(),
				"playerIds":         pendingPlayers,
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

	if gs.PendingDarklingsPriestOrdination != nil {
		return map[string]interface{}{
			"type":     "darklings_ordination",
			"playerId": gs.PendingDarklingsPriestOrdination.PlayerID,
		}
	}

	if gs.HasPendingLeechOffers() {
		if playerID := gs.GetNextBlockingLeechResponder(); playerID != "" {
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

	if gs.PendingHalflingsSpades != nil {
		return map[string]interface{}{
			"type":            "halflings_spades",
			"playerId":        gs.PendingHalflingsSpades.PlayerID,
			"spadesRemaining": gs.PendingHalflingsSpades.SpadesRemaining,
		}
	}

	if gs.PendingWispsStrongholdDwelling != nil {
		return map[string]interface{}{
			"type":     "wisps_stronghold_dwelling",
			"playerId": gs.PendingWispsStrongholdDwelling.PlayerID,
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

	if playerID := strings.TrimSpace(gs.PendingFreeActionsPlayerID); playerID != "" {
		return map[string]interface{}{
			"type":     "post_action_free_actions",
			"playerId": playerID,
		}
	}

	if playerID := strings.TrimSpace(gs.PendingTurnConfirmationPlayerID); playerID != "" {
		return map[string]interface{}{
			"type":     "turn_confirmation",
			"playerId": playerID,
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
