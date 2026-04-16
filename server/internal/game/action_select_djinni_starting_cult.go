package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// SelectDjinniStartingCultTrackAction resolves the Djinni setup cult choice.
type SelectDjinniStartingCultTrackAction struct {
	BaseAction
	CultTrack CultTrack
}

func NewSelectDjinniStartingCultTrackAction(playerID string, cultTrack CultTrack) *SelectDjinniStartingCultTrackAction {
	return &SelectDjinniStartingCultTrackAction{
		BaseAction: BaseAction{Type: ActionSelectDjinniStartingCultTrack, PlayerID: playerID},
		CultTrack:  cultTrack,
	}
}

func (a *SelectDjinniStartingCultTrackAction) GetType() ActionType {
	return ActionSelectDjinniStartingCultTrack
}

func (a *SelectDjinniStartingCultTrackAction) Validate(gs *GameState) error {
	if gs.PendingDjinniStartingCultChoice == nil {
		return fmt.Errorf("no pending djinni starting cult choice")
	}
	if gs.PendingDjinniStartingCultChoice.PlayerID != a.PlayerID {
		return fmt.Errorf("djinni starting cult choice required from player %s", gs.PendingDjinniStartingCultChoice.PlayerID)
	}
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if player.Faction == nil || player.Faction.GetType() != models.FactionDjinni {
		return fmt.Errorf("only Djinni can choose a starting cult track")
	}
	if a.CultTrack != CultFire && a.CultTrack != CultWater && a.CultTrack != CultEarth && a.CultTrack != CultAir {
		return fmt.Errorf("invalid cult track")
	}
	if player.CultPositions[a.CultTrack] >= 10 {
		return fmt.Errorf("already at maximum position on cult track")
	}
	return nil
}

func (a *SelectDjinniStartingCultTrackAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	if _, err := gs.AdvanceCultTrack(a.PlayerID, a.CultTrack, 2); err != nil {
		return fmt.Errorf("failed to advance Djinni starting cult track: %w", err)
	}
	gs.PendingDjinniStartingCultChoice = nil
	return nil
}
