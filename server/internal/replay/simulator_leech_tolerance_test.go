package replay

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
)

func TestStepForward_IgnoresDeclineWithoutOfferWhenPowerIsFull(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("Darklings", factions.NewFaction(models.FactionDarklings)); err != nil {
		t.Fatalf("add player: %v", err)
	}
	gs.Phase = game.PhaseAction
	p := gs.GetPlayer("Darklings")
	p.Resources.Power.Bowl1 = 0
	p.Resources.Power.Bowl2 = 0
	p.Resources.Power.Bowl3 = 12

	actions := []notation.LogItem{
		notation.ActionItem{Action: game.NewDeclinePowerLeechAction("Darklings", 0)},
	}
	sim := NewGameSimulator(gs, actions)

	if err := sim.StepForward(); err != nil {
		t.Fatalf("StepForward() error = %v, want nil", err)
	}
	if sim.CurrentIndex != 1 {
		t.Fatalf("expected simulator to advance to index 1, got %d", sim.CurrentIndex)
	}
}

func TestStepForward_DeclineWithoutOfferStillFailsWhenPowerNotFull(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("Darklings", factions.NewFaction(models.FactionDarklings)); err != nil {
		t.Fatalf("add player: %v", err)
	}
	gs.Phase = game.PhaseAction
	p := gs.GetPlayer("Darklings")
	p.Resources.Power.Bowl1 = 0
	p.Resources.Power.Bowl2 = 1
	p.Resources.Power.Bowl3 = 11

	actions := []notation.LogItem{
		notation.ActionItem{Action: game.NewDeclinePowerLeechAction("Darklings", 0)},
	}
	sim := NewGameSimulator(gs, actions)

	err := sim.StepForward()
	if err == nil {
		t.Fatalf("StepForward() error = nil, want missing offer error")
	}
	if !strings.Contains(err.Error(), "no pending leech offers") {
		t.Fatalf("expected missing offer error, got: %v", err)
	}
}
