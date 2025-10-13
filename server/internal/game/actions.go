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
	ActionPowerAction
	ActionSpecialAction
	ActionPass
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
		
		// TODO: Award VP if current scoring tile rewards spades
		// TODO: Award VP if current scoring tile rewards terraform
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

		// TODO: Award VP if current scoring tile rewards dwellings

		// Trigger power leech for adjacent players
		gs.TriggerPowerLeech(a.TargetHex, a.PlayerID, dwelling.PowerValue)
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

	// Check if player can afford upgrade
	var cost factions.Cost
	switch a.NewBuildingType {
	case models.BuildingTradingHouse:
		cost = player.Faction.GetTradingHouseCost()
	case models.BuildingTemple:
		cost = player.Faction.GetTempleCost()
	case models.BuildingSanctuary:
		cost = player.Faction.GetSanctuaryCost()
	case models.BuildingStronghold:
		cost = player.Faction.GetStrongholdCost()
	default:
		return fmt.Errorf("invalid building type for upgrade: %v", a.NewBuildingType)
	}

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
	oldBuilding := mapHex.Building

	// Get upgrade cost
	var cost factions.Cost
	var newPowerValue int
	switch a.NewBuildingType {
	case models.BuildingTradingHouse:
		cost = player.Faction.GetTradingHouseCost()
		newPowerValue = 2
	case models.BuildingTemple:
		cost = player.Faction.GetTempleCost()
		newPowerValue = 2
	case models.BuildingSanctuary:
		cost = player.Faction.GetSanctuaryCost()
		newPowerValue = 3
	case models.BuildingStronghold:
		cost = player.Faction.GetStrongholdCost()
		newPowerValue = 3
	}

	// Pay for upgrade
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for upgrade: %w", err)
	}

	// Upgrade building
	mapHex.Building = &models.Building{
		Type:       a.NewBuildingType,
		Faction:    player.Faction.GetType(),
		PlayerID:   a.PlayerID,
		PowerValue: newPowerValue,
	}

	// Trigger power leech for the power increase (new value - old value)
	powerIncrease := newPowerValue - oldBuilding.PowerValue
	if powerIncrease > 0 {
		gs.TriggerPowerLeech(a.TargetHex, a.PlayerID, powerIncrease)
	}

	return nil
}

// isValidUpgrade checks if an upgrade path is valid
func isValidUpgrade(from, to models.BuildingType) bool {
	validUpgrades := map[models.BuildingType][]models.BuildingType{
		models.BuildingDwelling: {
			models.BuildingTradingHouse,
			models.BuildingTemple,
		},
		models.BuildingTradingHouse: {
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

// PowerActionType represents different power actions
type PowerActionType int

const (
	PowerActionCoins PowerActionType = iota
	PowerActionWorkers
	PowerActionPriests
)

// PowerAction represents using power for resources
type PowerAction struct {
	BaseAction
	PowerType PowerActionType
	Amount    int // Amount of resource to gain
}

func NewPowerAction(playerID string, powerType PowerActionType, amount int) *PowerAction {
	return &PowerAction{
		BaseAction: BaseAction{
			Type:     ActionPowerAction,
			PlayerID: playerID,
		},
		PowerType: powerType,
		Amount:    amount,
	}
}

func (a *PowerAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	if a.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Check if player has enough power
	var powerNeeded int
	switch a.PowerType {
	case PowerActionCoins:
		powerNeeded = a.Amount // 1 power = 1 coin
	case PowerActionWorkers:
		powerNeeded = a.Amount * 3 // 3 power = 1 worker
	case PowerActionPriests:
		powerNeeded = a.Amount * 5 // 5 power = 1 priest
	default:
		return fmt.Errorf("invalid power action type")
	}

	if !player.Resources.Power.CanSpend(powerNeeded) {
		return fmt.Errorf("not enough power: need %d, have %d", powerNeeded, player.Resources.Power.Bowl3)
	}

	return nil
}

func (a *PowerAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)

	// Execute the appropriate conversion
	var err error
	switch a.PowerType {
	case PowerActionCoins:
		err = player.Resources.ConvertPowerToCoins(a.Amount)
	case PowerActionWorkers:
		err = player.Resources.ConvertPowerToWorkers(a.Amount)
	case PowerActionPriests:
		err = player.Resources.ConvertPowerToPriests(a.Amount)
	}

	if err != nil {
		return fmt.Errorf("failed to execute power action: %w", err)
	}

	return nil
}

// PassAction represents passing for the round
type PassAction struct {
	BaseAction
	BonusCardID string // Optional bonus card selection
}

func NewPassAction(playerID string, bonusCardID string) *PassAction {
	return &PassAction{
		BaseAction: BaseAction{
			Type:     ActionPass,
			PlayerID: playerID,
		},
		BonusCardID: bonusCardID,
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

	// TODO: Validate bonus card selection

	return nil
}

func (a *PassAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	player.HasPassed = true

	// TODO: Handle bonus card selection and benefits

	return nil
}
