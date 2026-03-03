package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func TestDiscardPendingSpadeAction_Execute(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewHalflings()); err != nil {
		t.Fatalf("failed to add p1: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewWitches()); err != nil {
		t.Fatalf("failed to add p2: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 0
	gs.PendingSpades["p1"] = 1
	gs.PendingSpadeBuildAllowed["p1"] = false

	action := NewDiscardPendingSpadeAction("p1", 1)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("discard should succeed: %v", err)
	}

	if _, ok := gs.PendingSpades["p1"]; ok {
		t.Fatalf("expected pending spade entry to be cleared")
	}
	if _, ok := gs.PendingSpadeBuildAllowed["p1"]; ok {
		t.Fatalf("expected pending spade build policy to be cleared")
	}
	if gs.CurrentPlayerIndex != 1 {
		t.Fatalf("expected turn to advance to p2 after resolving pending spade, got index %d", gs.CurrentPlayerIndex)
	}
}

func TestDiscardPendingSpadeAction_ValidateFailsWithoutPending(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewHalflings()); err != nil {
		t.Fatalf("failed to add p1: %v", err)
	}

	action := NewDiscardPendingSpadeAction("p1", 1)
	if err := action.Validate(gs); err == nil {
		t.Fatalf("expected validation to fail with no pending spades")
	}
}

func TestDiscardPendingSpadeAction_ExecuteCultRewardDuringIncome(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewHalflings()); err != nil {
		t.Fatalf("failed to add p1: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewWitches()); err != nil {
		t.Fatalf("failed to add p2: %v", err)
	}

	gs.Phase = PhaseIncome
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 0
	gs.PendingCultRewardSpades = map[string]int{"p1": 1}

	action := NewDiscardPendingSpadeAction("p1", 1)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("discard cult reward spade should succeed: %v", err)
	}

	if count := gs.PendingCultRewardSpades["p1"]; count != 0 {
		t.Fatalf("expected pending cult reward spades for p1 to be cleared, got %d", count)
	}
	if gs.Phase != PhaseAction {
		t.Fatalf("expected phase to advance to action after resolving income cult spades, got %d", gs.Phase)
	}
}
