package game

import (
	"fmt"
)

// SelectFavorTileAction represents selecting a favor tile after building Temple/Sanctuary/Auren Stronghold
type SelectFavorTileAction struct {
	BaseAction
	TileType FavorTileType
}

// GetType returns the action type
func (a *SelectFavorTileAction) GetType() ActionType {
	return ActionSelectFavorTile
}

// Validate checks if the action is valid
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

// Execute performs the action
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
	}
	// Each favor resolves immediately, including the FIRST Chaos Magicians tile.
	// Fire+2 can found a town whose key is already usable for its own cult gain.
	gs.CheckAllTownFormations(a.PlayerID)

	// Apply immediate effects (cult advancement for +3 tiles)
	// This happens AFTER town check so that pending town formation exists during cult advancement
	if err := ApplyFavorTileImmediate(gs, a.PlayerID, a.TileType); err != nil {
		return fmt.Errorf("failed to apply favor tile effect: %w", err)
	}

	gs.NextTurn()
	return nil
}
