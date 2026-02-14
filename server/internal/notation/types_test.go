package notation

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

func TestLogCompoundAction_AllowsAuxiliaryOnlySequence(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}
	// Ensure the auxiliary actions are executable.
	gs.Players[playerID].Resources.Power = game.NewPowerSystem(0, 2, 1)

	compound := &LogCompoundAction{
		Actions: []game.Action{
			&LogBurnAction{PlayerID: playerID, Amount: 1},
			&LogConversionAction{
				PlayerID: playerID,
				Cost: map[models.ResourceType]int{
					models.ResourcePower: 1,
				},
				Reward: map[models.ResourceType]int{
					models.ResourceCoin: 1,
				},
			},
		},
	}

	if err := compound.Execute(gs); err != nil {
		t.Fatalf("compound.Execute(auxiliary-only) error = %v, want nil", err)
	}
}

func TestLogDeclineLeechAction_NoPendingOffers_NoError(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}

	action := &LogDeclineLeechAction{PlayerID: playerID}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogDeclineLeechAction.Execute(no pending) error = %v, want nil", err)
	}
}
