package replay

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
)

type simulatorFailingAction struct {
	playerID string
}

func (a simulatorFailingAction) GetType() game.ActionType { return game.ActionSpecialAction }
func (a simulatorFailingAction) GetPlayerID() string      { return a.playerID }
func (a simulatorFailingAction) Validate(*game.GameState) error {
	return nil
}
func (a simulatorFailingAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.playerID)
	if player != nil {
		player.VictoryPoints += 10
	}
	return fmt.Errorf("forced replay failure")
}

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

func TestStepForward_RestoresActionStateAfterExecutionFailure(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("Darklings", factions.NewFaction(models.FactionDarklings)); err != nil {
		t.Fatalf("add player: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"Darklings"}
	gs.GetPlayer("Darklings").VictoryPoints = 20

	actions := []notation.LogItem{
		notation.ActionItem{Action: simulatorFailingAction{playerID: "Darklings"}},
	}
	sim := NewGameSimulator(gs, actions)

	err := sim.StepForward()
	if err == nil {
		t.Fatalf("StepForward() error = nil, want forced failure")
	}
	player := sim.CurrentState.GetPlayer("Darklings")
	if player == nil {
		t.Fatalf("player missing after failed StepForward")
	}
	if player.VictoryPoints != 20 {
		t.Fatalf("VictoryPoints = %d, want restored value 20", player.VictoryPoints)
	}
	if sim.CurrentIndex != 0 {
		t.Fatalf("CurrentIndex = %d, want 0 after failed StepForward", sim.CurrentIndex)
	}
}
