package game

import (
	"fmt"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ActionType represents the type of action a player can take
type ActionType int

const (
	ActionTransformAndBuild ActionType = iota
	ActionUpgradeBuilding
	ActionAdvanceShipping
	ActionAdvanceDigging
	ActionSendPriestToCult
	ActionPowerAction
	ActionSpecialAction
	ActionPass
	ActionUseCultSpade         // Use a spade from cult track reward (cleanup phase)
	ActionAcceptPowerLeech     // Accept a power leech offer
	ActionDeclinePowerLeech    // Decline a power leech offer
)

// Action represents a player action
type Action interface {
	GetType() ActionType
	GetPlayerID() string
	Validate(gs *GameState) error
	Execute(gs *GameState) error
}

// BaseAction provides common fields for all actions
type BaseAction struct {
	Type     ActionType
	PlayerID string
}

func (a *BaseAction) GetType() ActionType {
	return a.Type
}

func (a *BaseAction) GetPlayerID() string {
	return a.PlayerID
}

// TransformAndBuildAction represents terraforming a hex and optionally building a dwelling
// Per rulebook: "First, you may change the type of one Terrain space. Then, if you have 
// changed its type to your Home terrain, you may immediately build a Dwelling on that space."
type TransformAndBuildAction struct {
	BaseAction
	TargetHex      Hex
	BuildDwelling  bool // Whether to build a dwelling after transforming
}

func NewTransformAndBuildAction(playerID string, targetHex Hex, buildDwelling bool) *TransformAndBuildAction {
	return &TransformAndBuildAction{
		BaseAction: BaseAction{
			Type:     ActionTransformAndBuild,
			PlayerID: playerID,
		},
		TargetHex:      targetHex,
		BuildDwelling:  buildDwelling,
	}
}

func (a *TransformAndBuildAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if hex exists and is empty (no building)
	mapHex := gs.Map.GetHex(a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}
	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building: %v", a.TargetHex)
	}

	// Check adjacency - required for both transforming and building
	// "Even when transforming a Terrain space without building a Dwelling, the transformed 
	// Terrain space needs to be directly or indirectly adjacent to one of your Structures"
	if !gs.IsAdjacentToPlayerBuilding(a.TargetHex, a.PlayerID) {
		// TODO: Check for special abilities (Witches flying, Fakirs carpet, etc.)
		return fmt.Errorf("hex is not adjacent to player's buildings")
	}

	// Check if terrain needs transformation to home terrain
	needsTransform := mapHex.Terrain != player.Faction.GetHomeTerrain()
	
	totalWorkersNeeded := 0
	if needsTransform {
		// Calculate terraform cost
		distance := gs.Map.GetTerrainDistance(mapHex.Terrain, player.Faction.GetHomeTerrain())
		if distance == 0 {
			return fmt.Errorf("terrain distance calculation failed")
		}
		
		// GetTerraformCost returns total workers needed (already accounts for distance)
		totalWorkersNeeded = player.Faction.GetTerraformCost(distance)
	}

	// If building a dwelling, check requirements
	if a.BuildDwelling {
		// Check building limit (max 8 dwellings)
		if err := checkBuildingLimit(gs, a.PlayerID, models.BuildingDwelling); err != nil {
			return err
		}
		
		// After transformation (if any), hex must be player's home terrain
		if needsTransform {
			// Will be home terrain after transform
		} else if mapHex.Terrain != player.Faction.GetHomeTerrain() {
			return fmt.Errorf("cannot build dwelling: hex is not home terrain")
		}
		
		// Check if player can afford dwelling (coins and priests)
		dwellingCost := player.Faction.GetDwellingCost()
		if player.Resources.Coins < dwellingCost.Coins {
			return fmt.Errorf("not enough coins for dwelling: need %d, have %d", dwellingCost.Coins, player.Resources.Coins)
		}
		if player.Resources.Priests < dwellingCost.Priests {
			return fmt.Errorf("not enough priests for dwelling: need %d, have %d", dwellingCost.Priests, player.Resources.Priests)
		}
		
		// Add dwelling workers to total needed (checked separately below)
		totalWorkersNeeded += dwellingCost.Workers
	}
	
	// Check total workers needed (terraform + dwelling)
	if player.Resources.Workers < totalWorkersNeeded {
		return fmt.Errorf("not enough workers: need %d, have %d", totalWorkersNeeded, player.Resources.Workers)
	}

	return nil
}

func (a *TransformAndBuildAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	mapHex := gs.Map.GetHex(a.TargetHex)

	// Step 1: Transform terrain to home terrain if needed
	needsTransform := mapHex.Terrain != player.Faction.GetHomeTerrain()
	if needsTransform {
		distance := gs.Map.GetTerrainDistance(mapHex.Terrain, player.Faction.GetHomeTerrain())
		// GetTerraformCost returns total workers needed (already accounts for distance)
		totalWorkers := player.Faction.GetTerraformCost(distance)

		// Pay workers for terraform (spades)
		player.Resources.Workers -= totalWorkers

		// Transform terrain to home terrain
		if err := gs.Map.TransformTerrain(a.TargetHex, player.Faction.GetHomeTerrain()); err != nil {
			return fmt.Errorf("failed to transform terrain: %w", err)
		}
		
		// Award VP from scoring tile (per spade used)
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
	}

	// Step 2: Build dwelling if requested
	if a.BuildDwelling {
		// Pay for dwelling
		dwellingCost := player.Faction.GetDwellingCost()
		if err := player.Resources.Spend(dwellingCost); err != nil {
			return fmt.Errorf("failed to pay for dwelling: %w", err)
		}

		// Place dwelling
		dwelling := &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    player.Faction.GetType(),
			PlayerID:   a.PlayerID,
			PowerValue: 1, // Dwellings provide 1 power to neighbors
		}
		mapHex.Building = dwelling

		// Award VP from Earth+1 favor tile (+2 VP when building Dwelling)
		playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
		if HasFavorTile(playerTiles, FavorEarth1) {
			player.VictoryPoints += 2
		}

		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionDwelling)

		// Trigger power leech for adjacent players
		gs.TriggerPowerLeech(a.TargetHex, a.PlayerID)
		
		// Check for town formation after building dwelling
		connected := gs.CheckForTownFormation(a.PlayerID, a.TargetHex)
		if connected != nil {
			// Town can be formed - create pending town formation
			gs.PendingTownFormations[a.PlayerID] = &PendingTownFormation{
				PlayerID: a.PlayerID,
				Hexes:    connected,
			}
		}
	}

	return nil
}

// UpgradeBuildingAction represents upgrading a building
type UpgradeBuildingAction struct {
	BaseAction
	TargetHex      Hex
	NewBuildingType models.BuildingType
}

func NewUpgradeBuildingAction(playerID string, targetHex Hex, newType models.BuildingType) *UpgradeBuildingAction {
	return &UpgradeBuildingAction{
		BaseAction: BaseAction{
			Type:     ActionUpgradeBuilding,
			PlayerID: playerID,
		},
		TargetHex:      targetHex,
		NewBuildingType: newType,
	}
}

func (a *UpgradeBuildingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if hex has player's building
	mapHex := gs.Map.GetHex(a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}
	if mapHex.Building == nil {
		return fmt.Errorf("no building at hex: %v", a.TargetHex)
	}
	if mapHex.Building.PlayerID != a.PlayerID {
		return fmt.Errorf("building does not belong to player")
	}

	// Validate upgrade path
	if !isValidUpgrade(mapHex.Building.Type, a.NewBuildingType) {
		return fmt.Errorf("invalid upgrade: cannot upgrade %v to %v", mapHex.Building.Type, a.NewBuildingType)
	}

	// Check building limits
	if err := checkBuildingLimit(gs, a.PlayerID, a.NewBuildingType); err != nil {
		return err
	}

	// Get upgrade cost (may be reduced if adjacent to opponent)
	cost := getUpgradeCost(gs, player, mapHex, a.NewBuildingType)

	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford upgrade to %v", a.NewBuildingType)
	}

	return nil
}

func (a *UpgradeBuildingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	mapHex := gs.Map.GetHex(a.TargetHex)

	// Get upgrade cost (may be reduced if adjacent to opponent)
	cost := getUpgradeCost(gs, player, mapHex, a.NewBuildingType)

	// Pay for upgrade
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for upgrade: %w", err)
	}

	// Return old building to faction board (reduces income)
	// Buildings are returned to the rightmost position on their track
	// This is handled by the faction board state (not implemented yet)

	// Get new power value
	var newPowerValue int
	switch a.NewBuildingType {
	case models.BuildingTradingHouse:
		newPowerValue = 2
	case models.BuildingTemple:
		newPowerValue = 2
	case models.BuildingSanctuary:
		newPowerValue = 3
	case models.BuildingStronghold:
		newPowerValue = 3
	}

	// Upgrade building
	mapHex.Building = &models.Building{
		Type:       a.NewBuildingType,
		Faction:    player.Faction.GetType(),
		PlayerID:   a.PlayerID,
		PowerValue: newPowerValue,
	}

	// Handle special rewards based on upgrade type
	switch a.NewBuildingType {
	case models.BuildingTradingHouse:
		// Award VP from Water+1 favor tile (+3 VP when upgrading Dwelling→Trading House)
		playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
		if HasFavorTile(playerTiles, FavorWater1) {
			player.VictoryPoints += 3
		}
		
		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionTradingHouse)
		break
	case models.BuildingTemple, models.BuildingSanctuary:
		// Player must select a Favor tile
		// Chaos Magicians get 2 tiles instead of 1 (special passive ability)
		// This will be handled by a separate action/prompt system
		// For now, we just note that favor tile selection is pending
		// Number of tiles to select:
		//   - Chaos Magicians: 2 favor tiles
		//   - All other factions: 1 favor tile
		// TODO: Implement favor tile selection prompt/action (Phase 7+)
		
		// Award VP from scoring tile for Sanctuary
		if a.NewBuildingType == models.BuildingSanctuary {
			gs.AwardActionVP(a.PlayerID, ScoringActionStronghold)
		}
		break
	case models.BuildingStronghold:
		// Grant stronghold special ability
		player.HasStrongholdAbility = true
		
		// Auren gets an immediate favor tile when building stronghold
		// TODO: Implement favor tile selection prompt/action for Auren
		if player.Faction.GetType() == models.FactionAuren {
			// TODO: Award 1 Favor tile immediately
		}
		
		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionStronghold)
		break
	}

	// Trigger power leech when upgrading (adjacent players leech based on their adjacent buildings)
	gs.TriggerPowerLeech(a.TargetHex, a.PlayerID)
	
	// Check for town formation after upgrading
	connected := gs.CheckForTownFormation(a.PlayerID, a.TargetHex)
	if connected != nil {
		// Town can be formed - create pending town formation
		gs.PendingTownFormations[a.PlayerID] = &PendingTownFormation{
			PlayerID: a.PlayerID,
			Hexes:    connected,
		}
	}

	return nil
}

// isValidUpgrade checks if an upgrade path is valid
func isValidUpgrade(from, to models.BuildingType) bool {
	validUpgrades := map[models.BuildingType][]models.BuildingType{
		models.BuildingDwelling: {
			models.BuildingTradingHouse,
		},
		models.BuildingTradingHouse: {
			models.BuildingTemple,
			models.BuildingStronghold,
		},
		models.BuildingTemple: {
			models.BuildingSanctuary,
		},
	}

	allowed, exists := validUpgrades[from]
	if !exists {
		return false
	}

	for _, validTo := range allowed {
		if validTo == to {
			return true
		}
	}
	return false
}

// checkBuildingLimit checks if player has reached the building limit for a type
// Limits: 8 dwellings, 4 trading houses, 3 temples, 1 sanctuary, 1 stronghold
func checkBuildingLimit(gs *GameState, playerID string, buildingType models.BuildingType) error {
	// Count existing buildings of this type
	count := 0
	for _, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID && mapHex.Building.Type == buildingType {
			count++
		}
	}

	// Check limits
	var limit int
	switch buildingType {
	case models.BuildingDwelling:
		limit = 8
	case models.BuildingTradingHouse:
		limit = 4
	case models.BuildingTemple:
		limit = 3
	case models.BuildingSanctuary, models.BuildingStronghold:
		limit = 1
	default:
		return nil
	}

	if count >= limit {
		return fmt.Errorf("building limit reached: cannot have more than %d %v", limit, buildingType)
	}

	return nil
}

// getUpgradeCost calculates the upgrade cost, applying discount if adjacent to opponent
func getUpgradeCost(gs *GameState, player *Player, mapHex *MapHex, newBuildingType models.BuildingType) factions.Cost {
	var baseCost factions.Cost

	switch newBuildingType {
	case models.BuildingTradingHouse:
		baseCost = player.Faction.GetTradingHouseCost()
	case models.BuildingTemple:
		baseCost = player.Faction.GetTempleCost()
	case models.BuildingSanctuary:
		baseCost = player.Faction.GetSanctuaryCost()
	case models.BuildingStronghold:
		baseCost = player.Faction.GetStrongholdCost()
	default:
		return baseCost
	}

	// Apply discount for Trading House if adjacent to opponent
	if newBuildingType == models.BuildingTradingHouse {
		if hasAdjacentOpponent(gs, mapHex.Coord, player.ID) {
			// Reduce coin cost by half (6 -> 3 for most factions)
			baseCost.Coins = baseCost.Coins / 2
		}
	}

	return baseCost
}

// hasAdjacentOpponent checks if there's an opponent building adjacent to the hex
func hasAdjacentOpponent(gs *GameState, hex Hex, playerID string) bool {
	neighbors := hex.Neighbors()
	for _, neighbor := range neighbors {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID != playerID {
			return true
		}
	}
	return false
}

// AdvanceShippingAction represents advancing shipping level
type AdvanceShippingAction struct {
	BaseAction
}

func NewAdvanceShippingAction(playerID string) *AdvanceShippingAction {
	return &AdvanceShippingAction{
		BaseAction: BaseAction{
			Type:     ActionAdvanceShipping,
			PlayerID: playerID,
		},
	}
}

func (a *AdvanceShippingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player can advance shipping (some factions like Dwarves cannot)
	// TODO: Add faction-specific check

	// Check if already at max level
	if player.ShippingLevel >= 5 {
		return fmt.Errorf("shipping already at max level")
	}

	// Check if player can afford shipping upgrade
	cost := player.Faction.GetShippingCost(player.ShippingLevel)
	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford shipping upgrade")
	}

	return nil
}

func (a *AdvanceShippingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	cost := player.Faction.GetShippingCost(player.ShippingLevel)

	// Pay for upgrade
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for shipping: %w", err)
	}

	// Advance shipping
	player.ShippingLevel++

	return nil
}

// AdvanceDiggingAction represents advancing digging level
type AdvanceDiggingAction struct {
	BaseAction
}

func NewAdvanceDiggingAction(playerID string) *AdvanceDiggingAction {
	return &AdvanceDiggingAction{
		BaseAction: BaseAction{
			Type:     ActionAdvanceDigging,
			PlayerID: playerID,
		},
	}
}

func (a *AdvanceDiggingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if already at max level
	if player.DiggingLevel >= 2 {
		return fmt.Errorf("digging already at max level")
	}

	// Check if player can afford digging upgrade
	cost := player.Faction.GetDiggingCost(player.DiggingLevel)
	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford digging upgrade")
	}

	return nil
}

func (a *AdvanceDiggingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	cost := player.Faction.GetDiggingCost(player.DiggingLevel)

	// Pay for upgrade
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for digging: %w", err)
	}

	// Advance digging
	player.DiggingLevel++

	return nil
}

// PassAction represents passing for the round
type PassAction struct {
	BaseAction
	BonusCard *BonusCardType // Bonus card selection (required)
}

func NewPassAction(playerID string, bonusCard *BonusCardType) *PassAction {
	return &PassAction{
		BaseAction: BaseAction{
			Type:     ActionPass,
			PlayerID: playerID,
		},
		BonusCard: bonusCard,
	}
}

func (a *PassAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Validate bonus card selection
	if a.BonusCard == nil {
		return fmt.Errorf("bonus card selection is required when passing")
	}

	if !gs.BonusCards.IsAvailable(*a.BonusCard) {
		return fmt.Errorf("bonus card %v is not available", *a.BonusCard)
	}

	return nil
}

func (a *PassAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	player.HasPassed = true
	
	// Record pass order (determines turn order for next round)
	gs.PassOrder = append(gs.PassOrder, a.PlayerID)

	// Take bonus card and get coins from it
	coins, err := gs.BonusCards.TakeBonusCard(a.PlayerID, *a.BonusCard)
	if err != nil {
		return fmt.Errorf("failed to take bonus card: %w", err)
	}
	player.Resources.Coins += coins

	// Award VP from Air+1 favor tile (VP based on Trading House count)
	playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
	if HasFavorTile(playerTiles, FavorAir1) {
		// Count trading houses on the map
		tradingHouseCount := 0
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil && 
			   mapHex.Building.PlayerID == a.PlayerID && 
			   mapHex.Building.Type == models.BuildingTradingHouse {
				tradingHouseCount++
			}
		}
		
		vp := GetAir1PassVP(playerTiles, tradingHouseCount)
		player.VictoryPoints += vp
	}

	// Award VP from bonus card (based on buildings/shipping)
	bonusVP := GetBonusCardPassVP(*a.BonusCard, gs, a.PlayerID)
	player.VictoryPoints += bonusVP

	return nil
}

// SendPriestToCultAction represents sending a priest to a cult track
type SendPriestToCultAction struct {
	BaseAction
	Track         CultTrack
	UsePriestSlot bool // If true, priest is placed on board (2 or 3 spaces). If false, returns to supply (1 space)
	SlotValue     int  // 2 or 3, only used if UsePriestSlot is true
}

func (a *SendPriestToCultAction) GetType() ActionType {
	return ActionSendPriestToCult
}

func (a *SendPriestToCultAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has passed
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Check if player has a priest available
	if player.Resources.Priests < 1 {
		return fmt.Errorf("no priests available")
	}

	// Validate slot value if using priest slot
	if a.UsePriestSlot {
		if a.SlotValue != 2 && a.SlotValue != 3 {
			return fmt.Errorf("invalid priest slot value: %d (must be 2 or 3)", a.SlotValue)
		}
		
		// TODO: Check if the specific priest slot is available (not already occupied)
		// For now, we'll allow it - this can be tracked in CultTrackState if needed
	}

	return nil
}

func (a *SendPriestToCultAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)

	// Determine how many spaces to advance
	spacesToAdvance := 1 // Default: return priest to supply
	if a.UsePriestSlot {
		spacesToAdvance = a.SlotValue // 2 or 3 spaces
	}

	// Remove priest from player's supply
	player.Resources.Priests--

	// Advance on cult track (with bonus power at milestones)
	// Note: It's valid to sacrifice a priest even if you can't advance (no refund)
	// Position 10 requires a key (checked in AdvancePlayer)
	gs.CultTracks.AdvancePlayer(a.PlayerID, a.Track, spacesToAdvance, player)

	// Record priest sent for scoring tile #5 (Trading House + Priest)
	if gs.ScoringTiles != nil {
		gs.ScoringTiles.RecordPriestSent(a.PlayerID)
	}

	// Note: If UsePriestSlot is true, the priest is permanently placed on the board
	// If false, the priest is returned to supply (already handled by not placing it)

	return nil
}
