package game

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func TestConversionAction_RejectsWorkerToPriest(t *testing.T) {
	gs := NewGameState()
	playerID := "p1"
	if err := gs.AddPlayer(playerID, factions.NewDarklings()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	action := &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: playerID},
		ConversionType: ConversionWorkerToPriest,
		Amount:         1,
	}

	err := action.Validate(gs)
	if err == nil {
		t.Fatalf("ConversionAction.Validate() expected error")
	}
	if !strings.Contains(err.Error(), "worker to priest conversion is only allowed through Darklings priest ordination") {
		t.Fatalf("unexpected validation error: %v", err)
	}

	err = action.Execute(gs)
	if err == nil {
		t.Fatalf("ConversionAction.Execute() expected error")
	}
	if !strings.Contains(err.Error(), "worker to priest conversion is only allowed through Darklings priest ordination") {
		t.Fatalf("unexpected execute error: %v", err)
	}
}
