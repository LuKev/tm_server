package notation

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

func TestLogCompoundAction_RejectsAuxiliaryOnlySequence(t *testing.T) {
	compound := &LogCompoundAction{
		Actions: []game.Action{
			&LogBurnAction{PlayerID: "p1", Amount: 1},
			&LogConversionAction{
				PlayerID: "p1",
				Cost: map[models.ResourceType]int{
					models.ResourcePower: 1,
				},
				Reward: map[models.ResourceType]int{
					models.ResourceCoin: 1,
				},
			},
		},
	}

	err := compound.Execute(game.NewGameState())
	if err == nil {
		t.Fatalf("compound.Execute(auxiliary-only) expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no legal main action") {
		t.Fatalf("compound.Execute(auxiliary-only) error = %v, want main-action error", err)
	}
}

