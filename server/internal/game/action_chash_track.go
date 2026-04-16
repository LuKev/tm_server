package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

const chashIncomeTrackMaxLevel = 4

func chashIncomeTrackVP(level int) int {
	switch level {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 4:
		return 4
	default:
		return 0
	}
}

// AdvanceChashTrackAction advances the Chash Dallah income track.
type AdvanceChashTrackAction struct {
	BaseAction
}

func NewAdvanceChashTrackAction(playerID string) *AdvanceChashTrackAction {
	return &AdvanceChashTrackAction{
		BaseAction: BaseAction{
			Type:     ActionAdvanceChashTrack,
			PlayerID: playerID,
		},
	}
}

func (a *AdvanceChashTrackAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
	}
	if player.Faction.GetType() != models.FactionChashDallah {
		return fmt.Errorf("only Chash Dallah can advance the income track")
	}
	if player.ChashIncomeTrackLevel >= chashIncomeTrackMaxLevel {
		return fmt.Errorf("chash dallah income track is already at max level")
	}

	cost := factions.Cost{Coins: 2, Workers: 2}
	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford chash dallah income-track upgrade")
	}
	return nil
}

func (a *AdvanceChashTrackAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	cost := factions.Cost{Coins: 2, Workers: 2}
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay chash income-track cost: %w", err)
	}

	player.ChashIncomeTrackLevel++
	player.VictoryPoints += chashIncomeTrackVP(player.ChashIncomeTrackLevel)
	gs.NextTurn()
	return nil
}
