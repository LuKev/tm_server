package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ApplyHalflingsSpadeAction represents applying one of the 3 stronghold spades
type ApplyHalflingsSpadeAction struct {
	BaseAction
	TargetHex board.Hex
}

// GetType returns the action type
func (a *ApplyHalflingsSpadeAction) GetType() ActionType {
	return ActionApplyHalflingsSpade
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

	// Check if spades remain
	if gs.PendingHalflingsSpades.SpadesRemaining <= 0 {
		return fmt.Errorf("no spades remaining to apply")
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

	// Check if already transformed to home terrain
	if targetHex.Terrain == player.Faction.GetHomeTerrain() {
		return fmt.Errorf("hex is already your home terrain")
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

	// Transform the terrain
	targetTerrain := halflings.GetHomeTerrain()
	if err := gs.Map.TransformTerrain(a.TargetHex, targetTerrain); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Award VP for spade (Halflings get +1 VP per spade)
	player.VictoryPoints++

	// Award VP from scoring tile (if applicable)
	gs.AwardActionVP(a.PlayerID, ScoringActionSpades)

	// Update pending spades
	gs.PendingHalflingsSpades.SpadesRemaining--
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

	// Check if all spades have been applied
	if gs.PendingHalflingsSpades.SpadesRemaining > 0 {
		return fmt.Errorf("must apply all 3 spades before building dwelling")
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

	// Check if player can afford dwelling
	cost := player.Faction.GetDwellingCost()
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
	cost := player.Faction.GetDwellingCost()
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
	gs.PendingHalflingsSpades = nil

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

	// Check if all spades have been applied
	if gs.PendingHalflingsSpades.SpadesRemaining > 0 {
		return fmt.Errorf("must apply all 3 spades before skipping dwelling")
	}

	return nil
}

// Execute performs the action
func (a *SkipHalflingsDwellingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	// Clear pending Halflings spades
	gs.PendingHalflingsSpades = nil

	return nil
}
