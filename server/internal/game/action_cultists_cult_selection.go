package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// SelectCultistsCultTrackAction represents Cultists selecting a cult track for power leech bonus
// This happens when at least one opponent accepts power after Cultists triggers power leech
type SelectCultistsCultTrackAction struct {
	BaseAction
	CultTrack CultTrack
}

func NewSelectCultistsCultTrackAction(playerID string, cultTrack CultTrack) *SelectCultistsCultTrackAction {
	return &SelectCultistsCultTrackAction{
		BaseAction: BaseAction{
			Type:     ActionSelectCultistsCultTrack,
			PlayerID: playerID,
		},
		CultTrack: cultTrack,
	}
}

func (a *SelectCultistsCultTrackAction) Validate(gs *GameState) error {
	// Check if there's a pending cult selection for this player
	if gs.PendingCultistsCultSelection == nil {
		return fmt.Errorf("no pending cult track selection")
	}

	if gs.PendingCultistsCultSelection.PlayerID != a.PlayerID {
		return fmt.Errorf("cult track selection is not for this player")
	}

	// Verify player is Cultists
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}

	if player.Faction.GetType() != models.FactionCultists {
		return fmt.Errorf("only Cultists can use cult track selection")
	}

	// Validate cult track
	if a.CultTrack != CultFire && a.CultTrack != CultWater && a.CultTrack != CultEarth && a.CultTrack != CultAir {
		return fmt.Errorf("invalid cult track")
	}

	// Check if player can advance on this cult track
	currentPos := player.CultPositions[a.CultTrack]
	if currentPos >= 10 {
		return fmt.Errorf("already at maximum position on cult track")
	}

	return nil
}

func (a *SelectCultistsCultTrackAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	// Advance 1 space on the selected cult track
	// Uses gs.AdvanceCultTrack which handles power gains, keys, and position 10 blocking
	_, err := gs.AdvanceCultTrack(a.PlayerID, a.CultTrack, 1)
	if err != nil {
		return fmt.Errorf("failed to advance cult track: %w", err)
	}

	// Clear the pending state
	gs.PendingCultistsCultSelection = nil

	return nil
}
