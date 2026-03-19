package game

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func TestManagerTurnConfirmation_RequiresExplicitConfirmBeforeNextPlayerActs(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("actor", factions.NewCultists()); err != nil {
		t.Fatalf("add actor: %v", err)
	}
	if err := gs.AddPlayer("next", factions.NewWitches()); err != nil {
		t.Fatalf("add next: %v", err)
	}
	gs.TurnOrder = []string{"actor", "next"}
	gs.CurrentPlayerIndex = 0
	gs.Round = 6
	gs.Phase = PhaseAction

	next := gs.GetPlayer("next")
	if next == nil || next.Resources == nil || next.Resources.Power == nil {
		t.Fatal("missing next player resources")
	}
	next.Resources.Power.Bowl3 = 1

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", NewPassAction("actor", nil), ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor pass: %v", err)
	}

	if !gs.HasPendingTurnConfirmation() {
		t.Fatal("expected pending turn confirmation after pass")
	}
	if got := gs.PendingTurnConfirmationPlayerID; got != "actor" {
		t.Fatalf("pending confirmation player = %q, want actor", got)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "next" {
		t.Fatalf("current player after pass = %v, want next", current)
	}
	if pending, ok := serializePendingDecision(gs).(map[string]interface{}); !ok || pending["type"] != "turn_confirmation" || pending["playerId"] != "actor" {
		t.Fatalf("pending decision = %v, want turn_confirmation for actor", pending)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "next"},
		ConversionType: ConversionPowerToCoin,
		Amount:         1,
	}, ActionMeta{ExpectedRevision: 1}); err == nil {
		t.Fatal("expected next player action to be blocked before confirm")
	} else if !strings.Contains(err.Error(), "turn confirmation pending for player actor") {
		t.Fatalf("unexpected blocked-action error: %v", err)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", NewConfirmTurnAction("actor"), ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("actor confirm: %v", err)
	}
	if gs.HasPendingTurnConfirmation() {
		t.Fatal("expected confirmation window to be cleared")
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "next" {
		t.Fatalf("current player after confirm = %v, want next", current)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "next"},
		ConversionType: ConversionPowerToCoin,
		Amount:         1,
	}, ActionMeta{ExpectedRevision: 2}); err != nil {
		t.Fatalf("next conversion after confirm: %v", err)
	}
}

func TestManagerTurnConfirmation_UndoRestoresLastAcceptedLeechCheckpoint(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("actor", factions.NewHalflings()); err != nil {
		t.Fatalf("add actor: %v", err)
	}
	if err := gs.AddPlayer("neighbor", factions.NewWitches()); err != nil {
		t.Fatalf("add neighbor: %v", err)
	}
	gs.TurnOrder = []string{"actor", "neighbor"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = PhaseAction

	actor := gs.GetPlayer("actor")
	neighbor := gs.GetPlayer("neighbor")
	if actor == nil || neighbor == nil || actor.Resources == nil || actor.Resources.Power == nil || neighbor.Resources == nil || neighbor.Resources.Power == nil {
		t.Fatal("missing player resources")
	}
	actor.Resources.Power.Bowl3 = 1
	beforeActorCoins := actor.Resources.Coins

	gs.PendingLeechOffers = map[string][]*PowerLeechOffer{
		"neighbor": {
			{
				Amount:       2,
				FromPlayerID: "actor",
				EventID:      1,
			},
		},
	}
	gs.PendingFreeActionsPlayerID = "actor"
	gs.BeginPendingTurnConfirmation("actor", gs.CloneForUndo())

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", NewAcceptPowerLeechAction("neighbor", 0), ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("neighbor accept leech: %v", err)
	}

	acceptedPower := neighbor.Resources.Power.Clone()
	if !gs.HasPendingTurnConfirmation() {
		t.Fatal("expected turn confirmation to remain pending after accepted leech")
	}
	if got := gs.PendingTurnConfirmationPlayerID; got != "actor" {
		t.Fatalf("pending confirmation player = %q, want actor", got)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "actor"},
		ConversionType: ConversionPowerToCoin,
		Amount:         1,
	}, ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("actor conversion after leech: %v", err)
	}
	if actor.Resources.Coins != beforeActorCoins+1 {
		t.Fatalf("actor coins after conversion = %d, want %d", actor.Resources.Coins, beforeActorCoins+1)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", NewUndoTurnAction("actor"), ActionMeta{ExpectedRevision: 2}); err != nil {
		t.Fatalf("actor undo: %v", err)
	}

	actor = gs.GetPlayer("actor")
	neighbor = gs.GetPlayer("neighbor")
	if actor == nil || neighbor == nil || actor.Resources == nil || neighbor.Resources == nil || neighbor.Resources.Power == nil {
		t.Fatal("missing players after undo")
	}

	if gs.HasPendingTurnConfirmation() {
		t.Fatal("expected confirmation window to be cleared after undo")
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "actor" {
		t.Fatalf("current player after undo = %v, want actor", current)
	}
	if pending, ok := serializePendingDecision(gs).(map[string]interface{}); !ok || pending["type"] != "post_action_free_actions" || pending["playerId"] != "actor" {
		t.Fatalf("pending decision after undo = %v, want post_action_free_actions for actor", pending)
	}
	if actor.Resources.Coins != beforeActorCoins {
		t.Fatalf("actor coins after undo = %d, want %d", actor.Resources.Coins, beforeActorCoins)
	}
	if len(gs.PendingLeechOffers["neighbor"]) != 0 {
		t.Fatalf("expected accepted leech offers to stay resolved, got %d pending", len(gs.PendingLeechOffers["neighbor"]))
	}
	if neighbor.Resources.Power.Bowl1 != acceptedPower.Bowl1 ||
		neighbor.Resources.Power.Bowl2 != acceptedPower.Bowl2 ||
		neighbor.Resources.Power.Bowl3 != acceptedPower.Bowl3 {
		t.Fatalf("neighbor power after undo = %+v, want %+v", neighbor.Resources.Power, acceptedPower)
	}
}
