package game

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
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

func TestManagerTurnConfirmation_UndoRestoresAutoBurnedPowerAction(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("actor", factions.NewHalflings()); err != nil {
		t.Fatalf("add actor: %v", err)
	}
	if err := gs.AddPlayer("next", factions.NewWitches()); err != nil {
		t.Fatalf("add next: %v", err)
	}
	gs.TurnOrder = []string{"actor", "next"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = PhaseAction

	actor := gs.GetPlayer("actor")
	if actor == nil || actor.Resources == nil || actor.Resources.Power == nil {
		t.Fatal("missing actor resources")
	}
	actor.Resources.Power.Bowl1 = 0
	actor.Resources.Power.Bowl2 = 11
	actor.Resources.Power.Bowl3 = 1
	actor.Resources.Coins = 20
	actor.Resources.Workers = 20
	actor.Resources.Priests = 5

	initialHex := board.NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    actor.Faction.GetType(),
		PlayerID:   "actor",
		PowerValue: 1,
	}
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", NewPowerActionWithTransform("actor", PowerActionSpade2, targetHex, true), ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor ACT6: %v", err)
	}

	if actor.Resources.Power.Bowl1 != 6 || actor.Resources.Power.Bowl2 != 1 || actor.Resources.Power.Bowl3 != 0 {
		t.Fatalf("actor power after ACT6 = %+v, want bowl1=6 bowl2=1 bowl3=0", actor.Resources.Power)
	}
	if !gs.HasPendingTurnConfirmation() {
		t.Fatal("expected pending turn confirmation after ACT6")
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", NewUndoTurnAction("actor"), ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("undo ACT6: %v", err)
	}

	actor = gs.GetPlayer("actor")
	if actor == nil || actor.Resources == nil || actor.Resources.Power == nil {
		t.Fatal("missing actor resources after undo")
	}
	if actor.Resources.Power.Bowl1 != 0 || actor.Resources.Power.Bowl2 != 11 || actor.Resources.Power.Bowl3 != 1 {
		t.Fatalf("actor power after undo = %+v, want bowl1=0 bowl2=11 bowl3=1", actor.Resources.Power)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "actor" {
		t.Fatalf("current player after undo = %v, want actor", current)
	}
	if gs.Map.GetHex(targetHex).Building != nil {
		t.Fatal("expected ACT6 dwelling build to be undone")
	}
}

func TestManagerTurnConfirmation_UndoRestoresPostActionConversion(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("actor", factions.NewCultists()); err != nil {
		t.Fatalf("add actor: %v", err)
	}
	if err := gs.AddPlayer("next", factions.NewWitches()); err != nil {
		t.Fatalf("add next: %v", err)
	}
	gs.TurnOrder = []string{"actor", "next"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = PhaseAction

	actor := gs.GetPlayer("actor")
	if actor == nil || actor.Resources == nil || actor.Resources.Power == nil {
		t.Fatal("missing actor resources")
	}
	actor.Resources.Priests = 1
	actor.Resources.Power.Bowl3 = 1
	startCoins := actor.Resources.Coins
	startPriests := actor.Resources.Priests
	startFire := gs.CultTracks.GetPosition("actor", CultFire)

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "actor"},
		Track:         CultFire,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor send_priest: %v", err)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "actor"},
		ConversionType: ConversionPowerToCoin,
		Amount:         1,
	}, ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("actor conversion after main action: %v", err)
	}

	if actor.Resources.Coins != startCoins+1 {
		t.Fatalf("actor coins after conversion = %d, want %d", actor.Resources.Coins, startCoins+1)
	}
	if actor.Resources.Priests != startPriests-1 {
		t.Fatalf("actor priests after send_priest = %d, want %d", actor.Resources.Priests, startPriests-1)
	}
	if got := gs.CultTracks.GetPosition("actor", CultFire); got != startFire+1 {
		t.Fatalf("actor fire cult after send_priest = %d, want %d", got, startFire+1)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", NewUndoTurnAction("actor"), ActionMeta{ExpectedRevision: 2}); err != nil {
		t.Fatalf("actor undo: %v", err)
	}

	actor = gs.GetPlayer("actor")
	if actor == nil || actor.Resources == nil || actor.Resources.Power == nil {
		t.Fatal("missing actor resources after undo")
	}
	if actor.Resources.Coins != startCoins {
		t.Fatalf("actor coins after undo = %d, want %d", actor.Resources.Coins, startCoins)
	}
	if actor.Resources.Priests != startPriests {
		t.Fatalf("actor priests after undo = %d, want %d", actor.Resources.Priests, startPriests)
	}
	if got := gs.CultTracks.GetPosition("actor", CultFire); got != startFire {
		t.Fatalf("actor fire cult after undo = %d, want %d", got, startFire)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "actor" {
		t.Fatalf("current player after undo = %v, want actor", current)
	}
	if gs.HasPendingTurnConfirmation() {
		t.Fatal("expected confirmation window to be cleared after undo")
	}
}
