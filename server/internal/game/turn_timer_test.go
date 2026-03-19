package game

import (
	"testing"
	"time"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestTurnTimer_ConfirmHandoffAppliesIncrement(t *testing.T) {
	baseTime := time.Unix(1_700_000_000, 0)
	now := baseTime

	mgr := NewManager()
	mgr.now = func() time.Time { return now }

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
	gs.TurnTimer = NewTurnTimerState(gs.TurnOrder, TurnTimerConfig{
		InitialTimeMs: 60_000,
		IncrementMs:   5_000,
	})

	mgr.CreateGameWithState("g1", gs)

	now = now.Add(10 * time.Second)
	if _, err := mgr.ExecuteActionWithMeta("g1", NewPassAction("actor", nil), ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor pass: %v", err)
	}

	actorTimer := gs.TurnTimer.Players["actor"]
	nextTimer := gs.TurnTimer.Players["next"]
	if actorTimer == nil || nextTimer == nil {
		t.Fatal("missing timer state")
	}
	if actorTimer.RemainingMs != 50_000 {
		t.Fatalf("actor remaining after pass = %d, want 50000", actorTimer.RemainingMs)
	}
	if actorTimer.ActiveSinceMs != now.UnixMilli() {
		t.Fatalf("actor activeSince after pass = %d, want %d", actorTimer.ActiveSinceMs, now.UnixMilli())
	}
	if nextTimer.ActiveSinceMs != 0 {
		t.Fatalf("expected next timer to remain stopped during actor confirm window, got %d", nextTimer.ActiveSinceMs)
	}

	now = now.Add(3 * time.Second)
	if _, err := mgr.ExecuteActionWithMeta("g1", NewConfirmTurnAction("actor"), ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("actor confirm: %v", err)
	}

	if actorTimer.RemainingMs != 52_000 {
		t.Fatalf("actor remaining after confirm = %d, want 52000", actorTimer.RemainingMs)
	}
	if actorTimer.ActiveSinceMs != 0 {
		t.Fatalf("expected actor timer to stop after confirm, got %d", actorTimer.ActiveSinceMs)
	}
	if nextTimer.ActiveSinceMs != now.UnixMilli() {
		t.Fatalf("next activeSince after confirm = %d, want %d", nextTimer.ActiveSinceMs, now.UnixMilli())
	}

	serialized := mgr.SerializeGameState("g1")
	turnTimer, ok := serialized["turnTimer"].(map[string]interface{})
	if !ok {
		t.Fatalf("turnTimer serialization = %T, want map[string]interface{}", serialized["turnTimer"])
	}
	activePlayerIds, ok := turnTimer["activePlayerIds"].([]interface{})
	if !ok || len(activePlayerIds) != 1 || activePlayerIds[0] != "next" {
		t.Fatalf("serialized activePlayerIds = %v, want [next]", turnTimer["activePlayerIds"])
	}
}

func TestTurnTimer_FastAuctionTracksAllUnsubmittedPlayers(t *testing.T) {
	baseTime := time.Unix(1_700_000_100, 0)
	now := baseTime

	mgr := NewManager()
	mgr.now = func() time.Time { return now }

	gs := NewGameState()
	for _, playerID := range []string{"p1", "p2", "p3"} {
		if err := gs.AddPlayer(playerID, nil); err != nil {
			t.Fatalf("add %s: %v", playerID, err)
		}
	}
	gs.Phase = PhaseFactionSelection
	gs.SetupMode = SetupModeFastAuction
	gs.TurnOrder = []string{"p1", "p2", "p3"}
	gs.CurrentPlayerIndex = 0
	gs.AuctionState = NewAuctionStateWithMode(gs.TurnOrder, SetupModeFastAuction)
	if err := gs.AuctionState.NominateFaction("p1", models.FactionNomads); err != nil {
		t.Fatalf("nominate p1: %v", err)
	}
	if err := gs.AuctionState.NominateFaction("p2", models.FactionWitches); err != nil {
		t.Fatalf("nominate p2: %v", err)
	}
	if err := gs.AuctionState.NominateFaction("p3", models.FactionEngineers); err != nil {
		t.Fatalf("nominate p3: %v", err)
	}
	gs.TurnTimer = NewTurnTimerState(gs.TurnOrder, TurnTimerConfig{
		InitialTimeMs: 60_000,
		IncrementMs:   2_000,
	})

	mgr.CreateGameWithState("g1", gs)

	now = now.Add(4 * time.Second)
	if _, err := mgr.ExecuteActionWithMeta("g1", NewFastAuctionSubmitBidsAction("p2", map[models.FactionType]int{
		models.FactionNomads:    0,
		models.FactionWitches:   3,
		models.FactionEngineers: 5,
	}), ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("p2 fast auction submit: %v", err)
	}

	p1Timer := gs.TurnTimer.Players["p1"]
	p2Timer := gs.TurnTimer.Players["p2"]
	p3Timer := gs.TurnTimer.Players["p3"]
	if p1Timer.RemainingMs != 56_000 {
		t.Fatalf("p1 remaining after p2 submit = %d, want 56000", p1Timer.RemainingMs)
	}
	if p2Timer.RemainingMs != 58_000 {
		t.Fatalf("p2 remaining after submit = %d, want 58000", p2Timer.RemainingMs)
	}
	if p3Timer.RemainingMs != 56_000 {
		t.Fatalf("p3 remaining after p2 submit = %d, want 56000", p3Timer.RemainingMs)
	}
	if p2Timer.ActiveSinceMs != 0 {
		t.Fatalf("expected p2 timer to stop after submission, got %d", p2Timer.ActiveSinceMs)
	}
	if p1Timer.ActiveSinceMs != now.UnixMilli() || p3Timer.ActiveSinceMs != now.UnixMilli() {
		t.Fatalf("expected p1 and p3 timers to keep running at %d, got %d and %d", now.UnixMilli(), p1Timer.ActiveSinceMs, p3Timer.ActiveSinceMs)
	}

	serialized := mgr.SerializeGameState("g1")
	turnTimer, ok := serialized["turnTimer"].(map[string]interface{})
	if !ok {
		t.Fatalf("turnTimer serialization = %T, want map[string]interface{}", serialized["turnTimer"])
	}
	activePlayerIds, ok := turnTimer["activePlayerIds"].([]interface{})
	if !ok || len(activePlayerIds) != 2 || activePlayerIds[0] != "p1" || activePlayerIds[1] != "p3" {
		t.Fatalf("serialized activePlayerIds = %v, want [p1 p3]", turnTimer["activePlayerIds"])
	}
}
