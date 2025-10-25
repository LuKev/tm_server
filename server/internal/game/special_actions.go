package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Special Actions Implementation Notes:
//
// IMPLEMENTED STRONGHOLD SPECIAL ACTIONS:
//
// AUREN:
// - Stronghold: Advance 2 spaces on any cult track (once per round)
// - TODO: Finalize cult system (Phase 7) - power gains, keys, global tracking
//
// WITCHES:
// - Stronghold: Witches' Ride - build dwelling on any Forest (once per round)
// - Passive: +5 VP when founding a town (TODO: Phase 7/8)
//
// SWARMLINGS:
// - Stronghold: Upgrade dwelling to trading house for free (once per round)
// - Passive: +3 workers when founding a town (TODO: Phase 7/8)
//
// CHAOS MAGICIANS:
// - Stronghold: Take a double-turn (any 2 actions) (once per round)
// - Passive: 2 favor tiles instead of 1 for Temple/Sanctuary (TODO: Phase 7)
// - Setup: Start with only 1 dwelling, placed last (TODO: Phase 1)
//
// GIANTS:
// - Stronghold: 2 free spades to transform + optional build (once per round)
// - Passive: Always pay exactly 2 spades for any transform (TODO: integrate with terraform)
//
// NOMADS:
// - Stronghold: Sandstorm - transform adjacent hex + optional build (once per round)
// - Passive: Start with 3 dwellings instead of 2 (TODO: Phase 1)
//
// PASSIVE ABILITIES & IMMEDIATE BONUSES (TODO: Implement when needed):
//
// ALCHEMISTS:
// - Passive: Trade 1 VP <-> 1 Coin or 2 Coins <-> 1 VP anytime (Philosopher's Stone)
// - Stronghold: Immediate 12 power + ongoing 2 power per spade gained
//
// DARKLINGS:
// - Passive: Pay priests (not workers) for terraform, get 2 VP per step
// - Stronghold: Immediate trade up to 3 workers for 1 priest each
//
// HALFLINGS:
// - Passive: +1 VP for each spade gained
// - Stronghold: Immediate 3 spades to apply + optional build on one
//
// CULTISTS:
// - Passive: When opponent takes power leech, advance 1 cult space (or gain 1 power if all refuse)
// - Stronghold: Immediate 7 VP
//
// ENGINEERS:
// - Passive: Build bridge for 2 workers as an action (unlimited per round)
// - Stronghold: On pass, get 3 VP per bridge connecting your structures
//
// DWARVES:
// - Passive: Tunneling - skip 1 terrain/river for 2 workers, get 4 VP (no shipping)
// - Stronghold: Tunneling costs only 1 worker instead of 2
//
// MERMAIDS:
// - Passive: Skip 1 river when founding town
// - Stronghold: Immediate +1 shipping level for free
//
// FAKIRS:
// - Passive: Carpet Flight - skip 1 terrain/river for 1 priest, get 4 VP (no shipping upgrades)
// - Stronghold: Carpet Flight can now skip 2 spaces instead of 1
//
// TODO: Implement 2 bonus card special actions (Phase 7)

// SpecialActionType represents different special actions
type SpecialActionType int

const (
	SpecialActionAurenCultAdvance SpecialActionType = iota
	SpecialActionWitchesRide
	SpecialActionAlchemistsConvert
	SpecialActionSwarmlingsUpgrade
	SpecialActionChaosMagiciansDoubleTurn
	SpecialActionGiantsTransform
	SpecialActionNomadsSandstorm
	SpecialActionWater2CultAdvance      // Water+2 favor tile: Advance 1 on any cult track
	SpecialActionBonusCardSpade         // Bonus card: 1 free spade
	SpecialActionBonusCardCultAdvance   // Bonus card: Advance 1 on any cult track
	// TODO: Add other faction special actions
)

// SpecialAction represents a faction-specific special action
type SpecialAction struct {
	BaseAction
	ActionType SpecialActionType
	// For Auren cult advance
	CultTrack *CultTrack
	// For Witches' Ride, Giants, Nomads
	TargetHex     *Hex
	BuildDwelling bool // For Giants and Nomads - whether to build dwelling after transform
	// For Alchemists conversion
	ConvertVPToCoins bool // true = VP->Coins, false = Coins->VP
	Amount          int  // Number of conversions
	// For Swarmlings upgrade
	UpgradeHex *Hex
	// For Chaos Magicians double turn
	FirstAction  Action
	SecondAction Action
}

func NewSpecialAction(playerID string, actionType SpecialActionType) *SpecialAction {
	return &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: playerID,
		},
		ActionType: actionType,
	}
}

// NewAurenCultAdvanceAction creates an Auren cult advance special action
func NewAurenCultAdvanceAction(playerID string, cultTrack CultTrack) *SpecialAction {
	return &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: playerID,
		},
		ActionType: SpecialActionAurenCultAdvance,
		CultTrack:  &cultTrack,
	}
}

// NewWitchesRideAction creates a Witches' Ride special action
func NewWitchesRideAction(playerID string, targetHex Hex) *SpecialAction {
	return &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: playerID,
		},
		ActionType: SpecialActionWitchesRide,
		TargetHex:  &targetHex,
	}
}

// NewSwarmlingsUpgradeAction creates a Swarmlings free upgrade action
func NewSwarmlingsUpgradeAction(playerID string, upgradeHex Hex) *SpecialAction {
	return &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: playerID,
		},
		ActionType: SpecialActionSwarmlingsUpgrade,
		UpgradeHex: &upgradeHex,
	}
}

// NewChaosMagiciansDoubleTurnAction creates a Chaos Magicians double-turn action
func NewChaosMagiciansDoubleTurnAction(playerID string, firstAction, secondAction Action) *SpecialAction {
	return &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: playerID,
		},
		ActionType:   SpecialActionChaosMagiciansDoubleTurn,
		FirstAction:  firstAction,
		SecondAction: secondAction,
	}
}

// NewGiantsTransformAction creates a Giants 2-spade transform action
func NewGiantsTransformAction(playerID string, targetHex Hex, buildDwelling bool) *SpecialAction {
	return &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: playerID,
		},
		ActionType:    SpecialActionGiantsTransform,
		TargetHex:     &targetHex,
		BuildDwelling: buildDwelling,
	}
}

// NewNomadsSandstormAction creates a Nomads sandstorm action
func NewNomadsSandstormAction(playerID string, targetHex Hex, buildDwelling bool) *SpecialAction {
	return &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: playerID,
		},
		ActionType:    SpecialActionNomadsSandstorm,
		TargetHex:     &targetHex,
		BuildDwelling: buildDwelling,
	}
}

func (a *SpecialAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if this specific special action has already been used this round
	if player.SpecialActionsUsed[a.ActionType] {
		return fmt.Errorf("special action %v already used this round", a.ActionType)
	}

	// Stronghold actions require the stronghold ability
	strongholdActions := []SpecialActionType{
		SpecialActionAurenCultAdvance,
		SpecialActionWitchesRide,
		SpecialActionSwarmlingsUpgrade,
		SpecialActionChaosMagiciansDoubleTurn,
		SpecialActionGiantsTransform,
		SpecialActionNomadsSandstorm,
	}
	
	isStrongholdAction := false
	for _, sa := range strongholdActions {
		if a.ActionType == sa {
			isStrongholdAction = true
			break
		}
	}
	
	if isStrongholdAction && !player.HasStrongholdAbility {
		return fmt.Errorf("player does not have stronghold special ability")
	}

	switch a.ActionType {
	case SpecialActionAurenCultAdvance:
		return a.validateAurenCultAdvance(gs, player)
	case SpecialActionWitchesRide:
		return a.validateWitchesRide(gs, player)
	case SpecialActionSwarmlingsUpgrade:
		return a.validateSwarmlingsUpgrade(gs, player)
	case SpecialActionChaosMagiciansDoubleTurn:
		return a.validateChaosMagiciansDoubleTurn(gs, player)
	case SpecialActionGiantsTransform:
		return a.validateGiantsTransform(gs, player)
	case SpecialActionNomadsSandstorm:
		return a.validateNomadsSandstorm(gs, player)
	case SpecialActionWater2CultAdvance:
		return a.validateWater2CultAdvance(gs, player)
	case SpecialActionBonusCardSpade:
		return a.validateBonusCardSpade(gs, player)
	case SpecialActionBonusCardCultAdvance:
		return a.validateBonusCardCultAdvance(gs, player)
	default:
		return fmt.Errorf("unknown special action type")
	}
}

func (a *SpecialAction) validateAurenCultAdvance(gs *GameState, player *Player) error {
	// Verify player is Auren
	if player.Faction.GetType() != models.FactionAuren {
		return fmt.Errorf("only Auren can use cult advance special action")
	}

	if a.CultTrack == nil {
		return fmt.Errorf("cult track must be specified")
	}

	// Check current position on cult track
	currentPos := player.CultPositions[*a.CultTrack]
	
	// Can advance 2 spaces, but cannot go beyond 10
	if currentPos >= 10 {
		return fmt.Errorf("already at maximum position on cult track")
	}

	// TODO: Check if advancing to space 10 requires a key

	return nil
}

func (a *SpecialAction) validateWitchesRide(gs *GameState, player *Player) error {
	// Verify player is Witches
	if player.Faction.GetType() != models.FactionWitches {
		return fmt.Errorf("only Witches can use Witches' Ride special action")
	}

	if a.TargetHex == nil {
		return fmt.Errorf("target hex must be specified")
	}

	mapHex := gs.Map.GetHex(*a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}

	// Must be an unoccupied Forest space
	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	if mapHex.Terrain != models.TerrainForest {
		return fmt.Errorf("Witches' Ride can only build on Forest spaces")
	}

	// Check building limit (max 8 dwellings)
	if err := checkBuildingLimit(gs, a.PlayerID, models.BuildingDwelling); err != nil {
		return err
	}

	// Note: Adjacency rule is ignored for Witches' Ride

	return nil
}

func (a *SpecialAction) validateSwarmlingsUpgrade(gs *GameState, player *Player) error {
	// Verify player is Swarmlings
	if player.Faction.GetType() != models.FactionSwarmlings {
		return fmt.Errorf("only Swarmlings can use free upgrade special action")
	}

	if a.UpgradeHex == nil {
		return fmt.Errorf("upgrade hex must be specified")
	}

	mapHex := gs.Map.GetHex(*a.UpgradeHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.UpgradeHex)
	}

	if mapHex.Building == nil {
		return fmt.Errorf("no building at hex: %v", a.UpgradeHex)
	}

	if mapHex.Building.PlayerID != a.PlayerID {
		return fmt.Errorf("building does not belong to player")
	}

	if mapHex.Building.Type != models.BuildingDwelling {
		return fmt.Errorf("can only upgrade dwelling to trading house")
	}

	// Check trading house limit
	if err := checkBuildingLimit(gs, a.PlayerID, models.BuildingTradingHouse); err != nil {
		return err
	}

	return nil
}

func (a *SpecialAction) validateChaosMagiciansDoubleTurn(gs *GameState, player *Player) error {
	// Verify player is Chaos Magicians
	if player.Faction.GetType() != models.FactionChaosMagicians {
		return fmt.Errorf("only Chaos Magicians can use double-turn special action")
	}

	if a.FirstAction == nil || a.SecondAction == nil {
		return fmt.Errorf("both actions must be specified for double-turn")
	}

	// Validate both actions
	if err := a.FirstAction.Validate(gs); err != nil {
		return fmt.Errorf("first action invalid: %w", err)
	}

	if err := a.SecondAction.Validate(gs); err != nil {
		return fmt.Errorf("second action invalid: %w", err)
	}

	return nil
}

func (a *SpecialAction) validateGiantsTransform(gs *GameState, player *Player) error {
	// Verify player is Giants
	if player.Faction.GetType() != models.FactionGiants {
		return fmt.Errorf("only Giants can use 2-spade transform special action")
	}

	if a.TargetHex == nil {
		return fmt.Errorf("target hex must be specified")
	}

	mapHex := gs.Map.GetHex(*a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}

	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	// Check adjacency to player's buildings
	if !gs.IsAdjacentToPlayerBuilding(*a.TargetHex, a.PlayerID) {
		return fmt.Errorf("hex is not adjacent to player's buildings")
	}

	// If building dwelling, check limit
	if a.BuildDwelling {
		if err := checkBuildingLimit(gs, a.PlayerID, models.BuildingDwelling); err != nil {
			return err
		}
	}

	return nil
}

func (a *SpecialAction) validateNomadsSandstorm(gs *GameState, player *Player) error {
	// Verify player is Nomads
	if player.Faction.GetType() != models.FactionNomads {
		return fmt.Errorf("only Nomads can use sandstorm special action")
	}

	if a.TargetHex == nil {
		return fmt.Errorf("target hex must be specified")
	}

	mapHex := gs.Map.GetHex(*a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}

	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	// Must be directly adjacent (not via bridge or shipping)
	neighbors := a.TargetHex.Neighbors()
	hasAdjacentBuilding := false
	for _, neighbor := range neighbors {
		neighborHex := gs.Map.GetHex(neighbor)
		if neighborHex != nil && neighborHex.Building != nil && neighborHex.Building.PlayerID == a.PlayerID {
			hasAdjacentBuilding = true
			break
		}
	}

	if !hasAdjacentBuilding {
		return fmt.Errorf("sandstorm requires direct adjacency to player's structure")
	}

	// If building dwelling, check limit
	if a.BuildDwelling {
		if err := checkBuildingLimit(gs, a.PlayerID, models.BuildingDwelling); err != nil {
			return err
		}
	}

	return nil
}

func (a *SpecialAction) validateWater2CultAdvance(gs *GameState, player *Player) error {
	// Check if player has Water+2 favor tile
	playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
	if !HasFavorTile(playerTiles, FavorWater2) {
		return fmt.Errorf("player does not have Water+2 favor tile")
	}

	if a.CultTrack == nil {
		return fmt.Errorf("cult track must be specified")
	}

	return nil
}

func (a *SpecialAction) validateBonusCardSpade(gs *GameState, player *Player) error {
	// Check if player has the spade bonus card
	if bonusCard, ok := gs.BonusCards.GetPlayerCard(a.PlayerID); !ok || bonusCard != BonusCardSpade {
		return fmt.Errorf("player does not have the spade bonus card")
	}

	// Validate the transform action (hex must exist, be empty or transformable, etc.)
	if a.TargetHex == nil {
		return fmt.Errorf("target hex must be specified")
	}

	mapHex := gs.Map.GetHex(*a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}

	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	// Check adjacency
	if !gs.IsAdjacentToPlayerBuilding(*a.TargetHex, a.PlayerID) {
		return fmt.Errorf("hex is not adjacent to player's buildings")
	}

	return nil
}

func (a *SpecialAction) validateBonusCardCultAdvance(gs *GameState, player *Player) error {
	// Check if player has the cult advance bonus card
	if bonusCard, ok := gs.BonusCards.GetPlayerCard(a.PlayerID); !ok || bonusCard != BonusCardCultAdvance {
		return fmt.Errorf("player does not have the cult advance bonus card")
	}

	if a.CultTrack == nil {
		return fmt.Errorf("cult track must be specified")
	}

	return nil
}

func (a *SpecialAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)

	// Mark this specific special action as used
	player.SpecialActionsUsed[a.ActionType] = true

	switch a.ActionType {
	case SpecialActionAurenCultAdvance:
		return a.executeAurenCultAdvance(gs, player)
	case SpecialActionWitchesRide:
		return a.executeWitchesRide(gs, player)
	case SpecialActionSwarmlingsUpgrade:
		return a.executeSwarmlingsUpgrade(gs, player)
	case SpecialActionChaosMagiciansDoubleTurn:
		return a.executeChaosMagiciansDoubleTurn(gs, player)
	case SpecialActionGiantsTransform:
		return a.executeGiantsTransform(gs, player)
	case SpecialActionNomadsSandstorm:
		return a.executeNomadsSandstorm(gs, player)
	case SpecialActionWater2CultAdvance:
		return a.executeWater2CultAdvance(gs, player)
	case SpecialActionBonusCardSpade:
		return a.executeBonusCardSpade(gs, player)
	case SpecialActionBonusCardCultAdvance:
		return a.executeBonusCardCultAdvance(gs, player)
	default:
		return fmt.Errorf("unknown special action type")
	}
}

func (a *SpecialAction) executeAurenCultAdvance(gs *GameState, player *Player) error {
	// Advance 2 spaces on the chosen cult track
	currentPos := player.CultPositions[*a.CultTrack]
	newPos := currentPos + 2
	if newPos > 10 {
		newPos = 10
	}

	player.CultPositions[*a.CultTrack] = newPos

	// TODO: Finalize cult system implementation (Phase 7)
	// - Grant power for advancing on cult track (varies by space: 2/4/7 power at certain positions)
	// - Handle reaching space 10 (requires a key, only one player can reach 10 per track)
	// - Track cult track state globally (which spaces are occupied, who has keys)
	// - Award end-game cult majority bonuses

	return nil
}

func (a *SpecialAction) executeWitchesRide(gs *GameState, player *Player) error {
	mapHex := gs.Map.GetHex(*a.TargetHex)

	// Build dwelling without paying workers or coins
	// (Priests cost is still paid if required, but standard dwelling has no priest cost)
	dwelling := &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    player.Faction.GetType(),
		PlayerID:   a.PlayerID,
		PowerValue: 1,
	}
	mapHex.Building = dwelling

	// Trigger power leech for adjacent players
	gs.TriggerPowerLeech(*a.TargetHex, a.PlayerID)

	return nil
}

func (a *SpecialAction) executeSwarmlingsUpgrade(gs *GameState, player *Player) error {
	mapHex := gs.Map.GetHex(*a.UpgradeHex)

	// Upgrade dwelling to trading house without paying coins or workers
	mapHex.Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    player.Faction.GetType(),
		PlayerID:   a.PlayerID,
		PowerValue: 2,
	}
	
	// Award VP from Water+1 favor tile (+3 VP when upgrading Dwelling→Trading House)
	playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
	if HasFavorTile(playerTiles, FavorWater1) {
		player.VictoryPoints += 3
	}
	
	// Award VP from scoring tile
	gs.AwardActionVP(a.PlayerID, ScoringActionTradingHouse)

	// Trigger power leech for adjacent players
	gs.TriggerPowerLeech(*a.UpgradeHex, a.PlayerID)

	return nil
}

func (a *SpecialAction) executeChaosMagiciansDoubleTurn(gs *GameState, player *Player) error {
	// Execute first action
	if err := a.FirstAction.Execute(gs); err != nil {
		return fmt.Errorf("first action failed: %w", err)
	}

	// Execute second action
	if err := a.SecondAction.Execute(gs); err != nil {
		return fmt.Errorf("second action failed: %w", err)
	}

	return nil
}

func (a *SpecialAction) executeGiantsTransform(gs *GameState, player *Player) error {
	mapHex := gs.Map.GetHex(*a.TargetHex)

	// Transform terrain to home terrain (2 free spades)
	targetTerrain := player.Faction.GetHomeTerrain()
	gs.Map.TransformTerrain(*a.TargetHex, targetTerrain)
	
	// Award VP from scoring tile (2 spades used)
	// Giants always use 2 spades, so award VP twice
	gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
	gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
	
	// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
	// Note: Giants can't be Halflings, but this is here for consistency
	if halflings, ok := player.Faction.(*factions.Halflings); ok {
		vpBonus := halflings.GetVPPerSpade() * 2 // 2 spades
		player.VictoryPoints += vpBonus
	}
	
	// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
	// Note: Giants can't be Alchemists, but this is here for consistency
	if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
		powerBonus := alchemists.GetPowerPerSpade() * 2 // 2 spades
		if powerBonus > 0 {
			player.Resources.Power.Bowl1 += powerBonus
		}
	}

	// Build dwelling if requested
	if a.BuildDwelling {
		dwellingCost := player.Faction.GetDwellingCost()
		
		// Pay for dwelling
		if player.Resources.Coins < dwellingCost.Coins {
			return fmt.Errorf("not enough coins for dwelling: need %d, have %d", dwellingCost.Coins, player.Resources.Coins)
		}
		if player.Resources.Workers < dwellingCost.Workers {
			return fmt.Errorf("not enough workers for dwelling: need %d, have %d", dwellingCost.Workers, player.Resources.Workers)
		}
		if player.Resources.Priests < dwellingCost.Priests {
			return fmt.Errorf("not enough priests for dwelling: need %d, have %d", dwellingCost.Priests, player.Resources.Priests)
		}

		player.Resources.Coins -= dwellingCost.Coins
		player.Resources.Workers -= dwellingCost.Workers
		player.Resources.Priests -= dwellingCost.Priests

		// Place dwelling
		dwelling := &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    player.Faction.GetType(),
			PlayerID:   a.PlayerID,
			PowerValue: 1,
		}
		mapHex.Building = dwelling

		// Trigger power leech
		gs.TriggerPowerLeech(*a.TargetHex, a.PlayerID)
	}

	return nil
}

func (a *SpecialAction) executeNomadsSandstorm(gs *GameState, player *Player) error {
	mapHex := gs.Map.GetHex(*a.TargetHex)

	// Transform terrain to home terrain (Sandstorm - not considered a spade)
	targetTerrain := player.Faction.GetHomeTerrain()
	gs.Map.TransformTerrain(*a.TargetHex, targetTerrain)

	// Build dwelling if requested
	if a.BuildDwelling {
		dwellingCost := player.Faction.GetDwellingCost()
		
		// Pay for dwelling
		if player.Resources.Coins < dwellingCost.Coins {
			return fmt.Errorf("not enough coins for dwelling: need %d, have %d", dwellingCost.Coins, player.Resources.Coins)
		}
		if player.Resources.Workers < dwellingCost.Workers {
			return fmt.Errorf("not enough workers for dwelling: need %d, have %d", dwellingCost.Workers, player.Resources.Workers)
		}
		if player.Resources.Priests < dwellingCost.Priests {
			return fmt.Errorf("not enough priests for dwelling: need %d, have %d", dwellingCost.Priests, player.Resources.Priests)
		}

		player.Resources.Coins -= dwellingCost.Coins
		player.Resources.Workers -= dwellingCost.Workers
		player.Resources.Priests -= dwellingCost.Priests

		// Place dwelling
		dwelling := &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    player.Faction.GetType(),
			PlayerID:   a.PlayerID,
			PowerValue: 1,
		}
		mapHex.Building = dwelling

		// Trigger power leech
		gs.TriggerPowerLeech(*a.TargetHex, a.PlayerID)
	}

	return nil
}

func (a *SpecialAction) executeWater2CultAdvance(gs *GameState, player *Player) error {
	// Advance 1 space on the chosen cult track
	// This uses the cult track system which handles power bonuses automatically
	gs.CultTracks.AdvancePlayer(a.PlayerID, *a.CultTrack, 1, player)

	return nil
}

func (a *SpecialAction) executeBonusCardSpade(gs *GameState, player *Player) error {
	mapHex := gs.Map.GetHex(*a.TargetHex)

	// Get 1 free spade to transform terrain
	// Calculate terraform cost (but we get 1 free spade)
	distance := gs.Map.GetTerrainDistance(mapHex.Terrain, player.Faction.GetHomeTerrain())
	if distance == 0 {
		return fmt.Errorf("terrain distance calculation failed")
	}

	totalWorkers := player.Faction.GetTerraformCost(distance)
	
	// Calculate workers per spade (to subtract for the free spade)
	workersPerSpade := player.Faction.GetTerraformCost(1)
	
	// Subtract workers for 1 free spade (minimum 0)
	workersNeeded := totalWorkers - workersPerSpade
	if workersNeeded < 0 {
		workersNeeded = 0
	}

	// Pay remaining workers if needed
	if workersNeeded > 0 {
		if player.Resources.Workers < workersNeeded {
			return fmt.Errorf("not enough workers: need %d, have %d", workersNeeded, player.Resources.Workers)
		}
		player.Resources.Workers -= workersNeeded
	}

	// Transform terrain to home terrain
	if err := gs.Map.TransformTerrain(*a.TargetHex, player.Faction.GetHomeTerrain()); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Award VP from scoring tile for spades used
	// Even though we get 1 free spade, we still used spades for the transformation
	spadesUsed := player.Faction.GetTerraformSpades(distance)
	for i := 0; i < spadesUsed; i++ {
		gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
	}
	
	// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
	if halflings, ok := player.Faction.(*factions.Halflings); ok {
		vpBonus := halflings.GetVPPerSpade() * spadesUsed
		player.VictoryPoints += vpBonus
	}
	
	// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
	if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
		powerBonus := alchemists.GetPowerPerSpade() * spadesUsed
		if powerBonus > 0 {
			player.Resources.Power.Bowl1 += powerBonus
		}
	}

	// Optionally build dwelling if requested
	if a.BuildDwelling {
		dwellingCost := player.Faction.GetDwellingCost()
		if err := player.Resources.Spend(dwellingCost); err != nil {
			return fmt.Errorf("failed to pay for dwelling: %w", err)
		}

		dwelling := &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    player.Faction.GetType(),
			PlayerID:   a.PlayerID,
			PowerValue: 1,
		}
		mapHex.Building = dwelling

		// Award VP from scoring tile for dwelling
		gs.AwardActionVP(a.PlayerID, ScoringActionDwelling)

		// Trigger power leech
		gs.TriggerPowerLeech(*a.TargetHex, a.PlayerID)
		
		// Check for town formation
		connected := gs.CheckForTownFormation(a.PlayerID, *a.TargetHex)
		if connected != nil {
			gs.PendingTownFormations[a.PlayerID] = &PendingTownFormation{
				PlayerID: a.PlayerID,
				Hexes:    connected,
			}
		}
	}

	return nil
}

func (a *SpecialAction) executeBonusCardCultAdvance(gs *GameState, player *Player) error {
	// Advance 1 space on the chosen cult track (free)
	// This uses the cult track system which handles power bonuses automatically
	gs.CultTracks.AdvancePlayer(a.PlayerID, *a.CultTrack, 1, player)

	return nil
}
