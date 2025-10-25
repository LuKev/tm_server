package game

import (
	"fmt"
)

// UseCultSpadeAction represents using a free spade from cult track rewards
// These spades can only be used on directly or indirectly adjacent hexes
// The temporary shipping bonus from bonus cards does NOT extend the range
type UseCultSpadeAction struct {
	BaseAction
	TargetHex Hex
}

// NewUseCultSpadeAction creates a new cult spade action
func NewUseCultSpadeAction(playerID string, targetHex Hex) *UseCultSpadeAction {
	return &UseCultSpadeAction{
		BaseAction: BaseAction{
			Type:     ActionUseCultSpade,
			PlayerID: playerID,
		},
		TargetHex: targetHex,
	}
}

func (a *UseCultSpadeAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has pending spades
	if gs.PendingSpades == nil || gs.PendingSpades[a.PlayerID] <= 0 {
		return fmt.Errorf("player has no pending spades from cult rewards")
	}

	// Check if hex exists
	mapHex := gs.Map.GetHex(a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}

	// Check if hex is already occupied by a building
	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	// Check if hex is adjacent (directly or indirectly) to player's territory
	// Note: Temporary shipping bonus does NOT apply to cult reward spades
	if !gs.IsAdjacentToPlayerBuilding(a.TargetHex, a.PlayerID) {
		return fmt.Errorf("hex is not adjacent to your territory (cult spades can only be used on adjacent hexes)")
	}

	// Check if terrain can be transformed
	if mapHex.Terrain == player.Faction.GetHomeTerrain() {
		return fmt.Errorf("hex is already your home terrain")
	}

	return nil
}

func (a *UseCultSpadeAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)

	// Transform terrain to home terrain (free - no workers needed)
	if err := gs.Map.TransformTerrain(a.TargetHex, player.Faction.GetHomeTerrain()); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Use one pending spade
	if !gs.UseSpadeFromReward(a.PlayerID) {
		return fmt.Errorf("failed to use pending spade")
	}

	// Award VP from scoring tile for spades used
	// Even though this is a free spade, it still counts for scoring
	spadesUsed := 1 // Cult reward spades are always 1 spade at a time
	for i := 0; i < spadesUsed; i++ {
		gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
	}

	return nil
}
