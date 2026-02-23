package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
)

// UseCultSpadeAction represents using a free spade from cult track rewards
// These spades can only be used on directly or indirectly adjacent hexes
// The temporary shipping bonus from bonus cards does NOT extend the range
type UseCultSpadeAction struct {
	BaseAction
	TargetHex board.Hex
}

// NewUseCultSpadeAction creates a new cult spade action
func NewUseCultSpadeAction(playerID string, targetHex board.Hex) *UseCultSpadeAction {
	return &UseCultSpadeAction{
		BaseAction: BaseAction{
			Type:     ActionUseCultSpade,
			PlayerID: playerID,
		},
		TargetHex: targetHex,
	}
}

// GetType returns the action type
func (a *UseCultSpadeAction) GetType() ActionType {
	return ActionUseCultSpade
}

// Validate checks if the action is valid
func (a *UseCultSpadeAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has pending cult reward spades
	if gs.PendingCultRewardSpades == nil || gs.PendingCultRewardSpades[a.PlayerID] <= 0 {
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

// Execute performs the action
func (a *UseCultSpadeAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	mapHex := gs.Map.GetHex(a.TargetHex)
	currentTerrain := mapHex.Terrain
	homeTerrain := player.Faction.GetHomeTerrain()

	// Cult spade actions transform BY 1 spade towards home terrain
	// Not all the way to home terrain (unless it's only 1 spade away)
	targetTerrain := board.CalculateIntermediateTerrain(currentTerrain, homeTerrain, 1)

	// Transform terrain BY 1 spade (not all the way to home)
	if err := gs.Map.TransformTerrain(a.TargetHex, targetTerrain); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Use one pending cult reward spade
	if gs.PendingCultRewardSpades[a.PlayerID] <= 0 {
		return fmt.Errorf("failed to use pending cult reward spade")
	}
	gs.PendingCultRewardSpades[a.PlayerID]--
	if gs.PendingCultRewardSpades[a.PlayerID] == 0 {
		delete(gs.PendingCultRewardSpades, a.PlayerID)
	}

	// Cult reward spades do NOT award scoring tile VP
	// These are bonus spades from the previous round's cult rewards
	// However, faction-specific bonuses still apply (Halflings, Alchemists)
	spadesUsed := 1 // Cult reward spades are always 1 spade at a time

	// Award faction-specific spade bonuses (Halflings VP, Alchemists power)
	AwardFactionSpadeBonuses(player, spadesUsed)

	// During round-start income interlude, once all pending cult-reward spades
	// are resolved, proceed with normal income and then start the action phase.
	if gs.Phase == PhaseIncome {
		if _, count := gs.GetPendingCultRewardSpadePlayer(); count == 0 {
			gs.GrantIncome()
			gs.StartActionPhase()
		}
	}

	return nil
}
