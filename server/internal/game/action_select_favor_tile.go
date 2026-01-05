package game

import (
	"fmt"
)

// SelectFavorTileAction represents selecting a favor tile after building Temple/Sanctuary/Auren Stronghold
type SelectFavorTileAction struct {
	BaseAction
	TileType FavorTileType
}

func (a *SelectFavorTileAction) GetType() ActionType {
	return ActionSelectFavorTile
}

func (a *SelectFavorTileAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has passed
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Check if there's a pending favor tile selection
	if gs.PendingFavorTileSelection == nil {
		return fmt.Errorf("no pending favor tile selection")
	}

	// Check if this is the correct player
	if gs.PendingFavorTileSelection.PlayerID != a.PlayerID {
		return fmt.Errorf("pending favor tile selection is for player %s, not %s",
			gs.PendingFavorTileSelection.PlayerID, a.PlayerID)
	}

	// Check if player has already selected all tiles
	if len(gs.PendingFavorTileSelection.SelectedTiles) >= gs.PendingFavorTileSelection.Count {
		return fmt.Errorf("player has already selected all favor tiles")
	}

	// Check if tile is available
	if !gs.FavorTiles.IsAvailable(a.TileType) {
		return fmt.Errorf("favor tile %v is not available", a.TileType)
	}

	// Check if player already has this tile type
	if gs.FavorTiles.HasTileType(a.PlayerID, a.TileType) {
		return fmt.Errorf("player already has a %v favor tile", a.TileType)
	}

	return nil
}

func (a *SelectFavorTileAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	// Take the favor tile
	if err := gs.FavorTiles.TakeFavorTile(a.PlayerID, a.TileType); err != nil {
		return fmt.Errorf("failed to take favor tile: %w", err)
	}

	// Add to selected tiles
	gs.PendingFavorTileSelection.SelectedTiles = append(gs.PendingFavorTileSelection.SelectedTiles, a.TileType)

	// Check if all tiles have been selected
	if len(gs.PendingFavorTileSelection.SelectedTiles) >= gs.PendingFavorTileSelection.Count {
		// Clear pending selection
		gs.PendingFavorTileSelection = nil

		// After selecting favor tiles, re-check town formation for all player buildings
		// This is especially important for Fire+2 which reduces town power requirement from 7 to 6
		// A building cluster that previously didn't meet the power requirement may now form a town
		// IMPORTANT: Do this BEFORE applying immediate cult advancement so that PendingTownFormation
		// is set when cult advancement checks for position 10 key requirement
		player := gs.GetPlayer(a.PlayerID)
		if player != nil {
			// Check all hexes where player has buildings that aren't already part of a town
			// Note: With Fire+2 (reducing requirement from 7 to 6), multiple towns can form simultaneously
			gs.CheckAllTownFormations(a.PlayerID)
		}
	}

	// Apply immediate effects (cult advancement for +3 tiles)
	// This happens AFTER town check so that pending town formation exists during cult advancement
	if err := ApplyFavorTileImmediate(gs, a.PlayerID, a.TileType); err != nil {
		return fmt.Errorf("failed to apply favor tile effect: %w", err)
	}

	gs.NextTurn()
	return nil
}
