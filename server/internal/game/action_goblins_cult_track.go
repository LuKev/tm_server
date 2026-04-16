package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

type SelectGoblinsCultTrackAction struct {
	BaseAction
	CultTrack CultTrack
}

func NewSelectGoblinsCultTrackAction(playerID string, cultTrack CultTrack) *SelectGoblinsCultTrackAction {
	return &SelectGoblinsCultTrackAction{
		BaseAction: BaseAction{
			Type:     ActionSelectGoblinsCultTrack,
			PlayerID: playerID,
		},
		CultTrack: cultTrack,
	}
}

func (a *SelectGoblinsCultTrackAction) GetType() ActionType {
	return ActionSelectGoblinsCultTrack
}

func (a *SelectGoblinsCultTrackAction) Validate(gs *GameState) error {
	if gs.PendingGoblinsCultSteps == nil {
		return fmt.Errorf("no pending goblins cult steps")
	}
	if gs.PendingGoblinsCultSteps.PlayerID != a.PlayerID {
		return fmt.Errorf("goblins cult steps are not for this player")
	}

	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if player.Faction == nil || player.Faction.GetType() != models.FactionGoblins {
		return fmt.Errorf("only Goblins can resolve goblins cult steps")
	}
	if a.CultTrack != CultFire && a.CultTrack != CultWater && a.CultTrack != CultEarth && a.CultTrack != CultAir {
		return fmt.Errorf("invalid cult track")
	}
	if player.CultPositions[a.CultTrack] >= 10 {
		return fmt.Errorf("already at maximum position on cult track")
	}
	return nil
}

func (a *SelectGoblinsCultTrackAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	if _, err := gs.AdvanceCultTrack(a.PlayerID, a.CultTrack, 1); err != nil {
		return fmt.Errorf("failed to advance cult track: %w", err)
	}

	gs.PendingGoblinsCultSteps.StepsRemaining--
	if gs.PendingGoblinsCultSteps.StepsRemaining <= 0 {
		gs.PendingGoblinsCultSteps = nil
	}

	if gs.AllPlayersPassed() && !gs.HasLateRoundPendingDecisions() {
		advanceAfterRoundComplete(gs)
		return nil
	}
	if !gs.HasBlockingPendingLeechOffers() {
		if current := gs.GetCurrentPlayer(); current != nil && current.ID == a.PlayerID {
			gs.NextTurn()
		}
	}

	return nil
}
