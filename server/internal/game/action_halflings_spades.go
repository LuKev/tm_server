package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ApplyHalflingsSpadeAction represents applying spades from the Halflings stronghold ability
// The stronghold grants 3 spades total, which can be distributed across 1-3 hexes
// Each spade moves one step on the terrain wheel
type ApplyHalflingsSpadeAction struct {
	BaseAction
	TargetHex     board.Hex          // The hex to transform
	TargetTerrain models.TerrainType // The terrain to transform to (determines spades needed)
}

// GetType returns the action type
func (a *ApplyHalflingsSpadeAction) GetType() ActionType {
	return ActionApplyHalflingsSpade
}

// GetSpadesNeeded calculates how many spades are required for this transform
func (a *ApplyHalflingsSpadeAction) GetSpadesNeeded(gs *GameState) int {
	mapHex := gs.Map.GetHex(a.TargetHex)
	if mapHex == nil {
		return 0
	}
	steps, err := TerraformSpadeCount(gs.GetPlayer(a.PlayerID), mapHex.Terrain, a.TargetTerrain, 0)
	if err != nil {
		return 0
	}
	return steps
}

// Validate checks if the action is valid
func (a *ApplyHalflingsSpadeAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has passed
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Check if there's a pending Halflings spades application
	if gs.PendingHalflingsSpades == nil {
		return fmt.Errorf("no pending Halflings spades to apply")
	}

	// Check if this is the correct player
	if gs.PendingHalflingsSpades.PlayerID != a.PlayerID {
		return fmt.Errorf("pending spades are for player %s, not %s",
			gs.PendingHalflingsSpades.PlayerID, a.PlayerID)
	}

	// Check if player is Halflings
	if player.Faction.GetType() != models.FactionHalflings {
		return fmt.Errorf("only Halflings can use this action")
	}
	if gs.PendingHalflingsSpades.SpadesRemaining <= 0 {
		return fmt.Errorf("no free spades remaining")
	}
	// FAQ 2.1: finish a space at home before distributing leftovers. Each
	// application selects a final destination, so revisiting a hex is an alias
	// of selecting that destination initially, not another legal decision.
	for _, hex := range gs.PendingHalflingsSpades.TransformedHexes {
		if hex == a.TargetHex {
			return fmt.Errorf("terrain space already transformed in this action")
		}
		if gs.Map.GetHex(hex).Terrain != effectiveHomeTerrain(player) {
			return fmt.Errorf("must finish previous terrain at home before distributing spades")
		}
	}
	if !gs.IsAdjacentToPlayerBuilding(a.TargetHex, a.PlayerID) {
		return fmt.Errorf("terrain must be directly or indirectly adjacent")
	}

	// Check if hex is valid (on the map)
	targetHex := gs.Map.GetHex(a.TargetHex)
	if targetHex == nil {
		return fmt.Errorf("invalid hex: %v", a.TargetHex)
	}

	// Check if hex already has a building
	if targetHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	// Check if terrain can be transformed
	if targetHex.Terrain == models.TerrainRiver {
		return fmt.Errorf("cannot terraform river hexes")
	}

	// Check if target terrain is valid (not river)
	if a.TargetTerrain == models.TerrainRiver {
		return fmt.Errorf("cannot transform to river")
	}

	// Check if already at target terrain
	if targetHex.Terrain == a.TargetTerrain {
		return fmt.Errorf("hex is already target terrain")
	}

	// Calculate spades needed and check if player has enough
	spadesNeeded := a.GetSpadesNeeded(gs)
	if spadesNeeded <= 0 {
		return fmt.Errorf("invalid transform: no spades needed")
	}
	if gs.PendingHalflingsSpades.SpadesRemaining < spadesNeeded {
		if len(gs.PendingHalflingsSpades.TransformedHexes) > 0 {
			return fmt.Errorf("leftover stronghold spades cannot be supplemented on another terrain space")
		}
		home := effectiveHomeTerrain(player)
		homeSteps, _ := TerraformSpadeCount(player, targetHex.Terrain, home, 0)
		remainingSteps, _ := TerraformSpadeCount(player, a.TargetTerrain, home, 0)
		if homeSteps != spadesNeeded+remainingSteps {
			return fmt.Errorf("paid extra spades must follow the shortest route toward home")
		}
		workers := player.Faction.GetTerraformCost(spadesNeeded - gs.PendingHalflingsSpades.SpadesRemaining)
		if player.Resources.Workers < workers {
			return fmt.Errorf("cannot afford extra spades")
		}
	}

	return nil
}

// Execute performs the action
func (a *ApplyHalflingsSpadeAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	halflings, ok := player.Faction.(*factions.Halflings)
	if !ok {
		return fmt.Errorf("player is not Halflings")
	}

	// Calculate spades needed for this transform
	spadesNeeded := a.GetSpadesNeeded(gs)
	freeSpades := min(spadesNeeded, gs.PendingHalflingsSpades.SpadesRemaining)
	player.Resources.Workers -= player.Faction.GetTerraformCost(spadesNeeded - freeSpades)

	// Transform the terrain to the specified target (not necessarily home terrain)
	if err := gs.Map.TransformTerrain(a.TargetHex, a.TargetTerrain); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Award VP for each spade used (Halflings get +1 VP per spade)
	player.VictoryPoints += spadesNeeded

	// Award VP from scoring tile for each spade (if applicable)
	for i := 0; i < spadesNeeded; i++ {
		gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
	}

	// Update pending spades - decrement by actual spades used
	gs.PendingHalflingsSpades.SpadesRemaining -= freeSpades
	gs.PendingHalflingsSpades.TransformedHexes = append(gs.PendingHalflingsSpades.TransformedHexes, a.TargetHex)

	// If all spades have been applied, mark as used
	if gs.PendingHalflingsSpades.SpadesRemaining == 0 {
		// Mark the faction method as used
		halflings.UseStrongholdSpades()
		// Keep the pending state for optional dwelling placement
		// It will be cleared when player passes or builds a dwelling
	}

	return nil
}

// BuildHalflingsDwellingAction represents building a dwelling on one of the transformed hexes
type BuildHalflingsDwellingAction struct {
	BaseAction
	TargetHex board.Hex
}

// GetType returns the action type
func (a *BuildHalflingsDwellingAction) GetType() ActionType {
	return ActionBuildHalflingsDwelling
}

// Validate checks if the action is valid
func (a *BuildHalflingsDwellingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has passed
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Check if there's a pending Halflings spades application
	if gs.PendingHalflingsSpades == nil {
		return fmt.Errorf("no pending Halflings spades")
	}

	// Check if this is the correct player
	if gs.PendingHalflingsSpades.PlayerID != a.PlayerID {
		return fmt.Errorf("pending spades are for player %s, not %s",
			gs.PendingHalflingsSpades.PlayerID, a.PlayerID)
	}

	// Check if player is Halflings
	if player.Faction.GetType() != models.FactionHalflings {
		return fmt.Errorf("only Halflings can use this action")
	}

	// Check if hex is one of the transformed hexes
	isTransformed := false
	for _, hex := range gs.PendingHalflingsSpades.TransformedHexes {
		if hex == a.TargetHex {
			isTransformed = true
			break
		}
	}
	if !isTransformed {
		return fmt.Errorf("can only build dwelling on one of the 3 transformed hexes")
	}

	// Check if hex already has a building
	targetHex := gs.Map.GetHex(a.TargetHex)
	if targetHex == nil {
		return fmt.Errorf("invalid hex: %v", a.TargetHex)
	}
	if targetHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}
	if targetHex.Terrain != effectiveHomeTerrain(player) {
		return fmt.Errorf("dwelling requires home terrain")
	}
	if err := gs.CheckBuildingLimit(a.PlayerID, models.BuildingDwelling); err != nil {
		return err
	}

	// Check if player can afford dwelling
	cost := getDwellingBuildCost(gs, player, a.TargetHex)
	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford dwelling")
	}

	return nil
}

// Execute performs the action
func (a *BuildHalflingsDwellingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)

	// Pay for dwelling
	cost := getDwellingBuildCost(gs, player, a.TargetHex)
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for dwelling: %w", err)
	}

	// Place dwelling
	dwelling := &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    player.Faction.GetType(),
		PlayerID:   a.PlayerID,
		PowerValue: 1,
	}
	if err := gs.Map.PlaceBuilding(a.TargetHex, dwelling); err != nil {
		return fmt.Errorf("failed to place building: %w", err)
	}

	// Award VP from Earth+1 favor tile (+2 VP when building Dwelling)
	playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
	if HasFavorTile(playerTiles, FavorEarth1) {
		player.VictoryPoints += 2
	}

	// Award VP from scoring tile
	gs.AwardActionVP(a.PlayerID, ScoringActionDwelling)

	// Trigger power leech for adjacent players
	gs.TriggerPowerLeech(a.TargetHex, a.PlayerID)

	// Check for town formation
	gs.CheckForTownFormation(a.PlayerID, a.TargetHex)

	// Clear pending Halflings spades
	player.Faction.(*factions.Halflings).UseStrongholdSpades()
	gs.PendingHalflingsSpades = nil
	gs.NextTurn()

	return nil
}

// SkipHalflingsDwellingAction represents choosing not to build the optional dwelling
type SkipHalflingsDwellingAction struct {
	BaseAction
}

// GetType returns the action type
func (a *SkipHalflingsDwellingAction) GetType() ActionType {
	return ActionSkipHalflingsDwelling
}

// Validate checks if the action is valid
func (a *SkipHalflingsDwellingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if there's a pending Halflings spades application
	if gs.PendingHalflingsSpades == nil {
		return fmt.Errorf("no pending Halflings spades")
	}

	// Check if this is the correct player
	if gs.PendingHalflingsSpades.PlayerID != a.PlayerID {
		return fmt.Errorf("pending spades are for player %s, not %s",
			gs.PendingHalflingsSpades.PlayerID, a.PlayerID)
	}

	if player.HasPassed || player.Faction.GetType() != models.FactionHalflings {
		return fmt.Errorf("only active Halflings may resolve their spades")
	}

	return nil
}

// Execute performs the action
func (a *SkipHalflingsDwellingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	// Clear pending Halflings spades
	gs.GetPlayer(a.PlayerID).Faction.(*factions.Halflings).UseStrongholdSpades()
	gs.PendingHalflingsSpades = nil
	gs.NextTurn()

	return nil
}
