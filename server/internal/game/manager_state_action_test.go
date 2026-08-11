package game

import (
	"encoding/json"
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func TestApplyActionToStateMatchesManagerLifecycle(t *testing.T) {
	base := NewGameState()
	if err := base.AddPlayer("actor", factions.NewWitches()); err != nil {
		t.Fatal(err)
	}
	if err := base.AddPlayer("opponent", factions.NewEngineers()); err != nil {
		t.Fatal(err)
	}
	base.TurnOrder = []string{"actor", "opponent"}
	base.Phase = PhaseAction
	base.SetupSubphase = SetupSubphaseComplete
	base.GetPlayer("actor").Options.ConfirmActions = false
	base.GetPlayer("opponent").Options.ConfirmActions = false

	direct := base.CloneForUndo()
	managed := base.CloneForUndo()
	action := &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "actor"},
		ConversionType: ConversionPriestToWorker,
		Amount:         1,
	}
	direct.GetPlayer("actor").Resources.Priests = 1
	managed.GetPlayer("actor").Resources.Priests = 1
	if err := ApplyActionToState(direct, action, StateActionOptions{}); err != nil {
		t.Fatalf("direct apply failed: %v", err)
	}

	mgr := NewManager()
	mgr.CreateGameWithState("g", managed)
	if _, err := mgr.ExecuteActionWithMeta("g", action, ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("manager apply failed: %v", err)
	}
	got, _ := json.Marshal(managed)
	want, _ := json.Marshal(direct)
	if string(got) != string(want) {
		t.Fatalf("manager and direct state differ\nmanager=%s\ndirect=%s", got, want)
	}
}

func TestApplyActionToStateMatchesManagerForTurnAndPendingActions(t *testing.T) {
	base := NewGameState()
	if err := base.AddPlayer("actor", factions.NewWitches()); err != nil {
		t.Fatal(err)
	}
	if err := base.AddPlayer("opponent", factions.NewEngineers()); err != nil {
		t.Fatal(err)
	}
	base.TurnOrder = []string{"actor", "opponent"}
	base.Phase = PhaseAction
	base.SetupSubphase = SetupSubphaseComplete
	base.GetPlayer("actor").Options.ConfirmActions = false
	base.GetPlayer("opponent").Options.ConfirmActions = false
	base.GetPlayer("actor").Resources.Coins = 10
	base.GetPlayer("actor").Resources.Priests = 2

	t.Run("turn-advancing action", func(t *testing.T) {
		assertDirectMatchesManager(t, base,
			NewAdvanceShippingAction("actor"),
			NewAdvanceShippingAction("actor"),
		)
	})

	t.Run("pending decision", func(t *testing.T) {
		pending := base.CloneForUndo()
		pending.PendingFavorTileSelection = &PendingFavorTileSelection{PlayerID: "actor", Count: 1}
		newAction := func() Action {
			return &SelectFavorTileAction{
				BaseAction: BaseAction{Type: ActionSelectFavorTile, PlayerID: "actor"},
				TileType:   FavorFire3,
			}
		}
		assertDirectMatchesManager(t, pending, newAction(), newAction())
	})
}

func assertDirectMatchesManager(t *testing.T, base *GameState, directAction, managedAction Action) {
	t.Helper()
	direct := base.CloneForUndo()
	managed := base.CloneForUndo()
	if err := ApplyActionToState(direct, directAction, StateActionOptions{}); err != nil {
		t.Fatalf("direct apply failed: %v", err)
	}
	mgr := NewManager()
	mgr.CreateGameWithState("g", managed)
	if _, err := mgr.ExecuteActionWithMeta("g", managedAction, ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("manager apply failed: %v", err)
	}
	got, _ := json.Marshal(managed)
	want, _ := json.Marshal(direct)
	if string(got) != string(want) {
		t.Fatalf("manager and direct state differ\nmanager=%s\ndirect=%s", got, want)
	}
}

func TestCloneForUndoDoesNotShareMutableFactionState(t *testing.T) {
	base := NewGameState()
	if err := base.AddPlayer("actor", factions.NewAuren()); err != nil {
		t.Fatal(err)
	}
	if err := base.AddPlayer("opponent", factions.NewEngineers()); err != nil {
		t.Fatal(err)
	}
	base.TurnOrder = []string{"actor", "opponent"}
	base.Phase = PhaseAction
	base.GetPlayer("actor").Resources.Coins = 20
	base.GetPlayer("actor").Resources.Workers = 10
	base.GetPlayer("actor").Resources.Priests = 5

	before, _ := json.Marshal(base.GetPlayer("actor").Faction)
	clone := base.CloneForUndo()
	if err := NewAdvanceDiggingAction("actor").Execute(clone); err != nil {
		t.Fatalf("advance digging on clone failed: %v", err)
	}
	after, _ := json.Marshal(base.GetPlayer("actor").Faction)
	if string(before) != string(after) {
		t.Fatalf("clone action mutated original faction: before=%s after=%s", before, after)
	}
	if base.GetPlayer("actor").DiggingLevel != 0 {
		t.Fatalf("clone action mutated original player digging level: %d", base.GetPlayer("actor").DiggingLevel)
	}
}

func TestCloneForUndoPreservesMapIdentity(t *testing.T) {
	base := NewGameState()
	clone := base.CloneForUndo()
	if clone.Map == nil || clone.Map.ID != base.Map.ID {
		t.Fatalf("clone map ID = %q, want %q", clone.Map.ID, base.Map.ID)
	}
}

func TestPassedPlayerLeechBecomesBlockingAtRoundEnd(t *testing.T) {
	state := NewGameState()
	if err := state.AddPlayer("actor", factions.NewWitches()); err != nil {
		t.Fatal(err)
	}
	if err := state.AddPlayer("responder", factions.NewEngineers()); err != nil {
		t.Fatal(err)
	}
	state.TurnOrder = []string{"actor", "responder"}
	state.Phase = PhaseAction
	state.Round = 1
	state.SetupSubphase = SetupSubphaseComplete
	state.GetPlayer("actor").HasPassed = true
	state.GetPlayer("responder").HasPassed = true
	state.PendingLeechOffers["responder"] = []*PowerLeechOffer{{
		FromPlayerID: "actor",
		Amount:       2,
		VPCost:       1,
	}}

	if got := state.GetNextBlockingLeechResponder(); got != "responder" {
		t.Fatalf("round-end blocking responder = %q, want responder", got)
	}
	owners := DecisionPlayerIDs(state)
	if len(owners) != 1 || owners[0] != "responder" {
		t.Fatalf("decision owners = %v, want responder", owners)
	}
	if err := ApplyActionToState(state, NewDeclinePowerLeechAction("responder", 0), StateActionOptions{}); err != nil {
		t.Fatalf("decline round-end leech: %v", err)
	}
	if state.HasPendingLeechOffers() {
		t.Fatal("round-end leech offer was not removed")
	}
	if state.Round != 2 || state.Phase != PhaseAction {
		t.Fatalf("round did not resume normally: round=%d phase=%d", state.Round, state.Phase)
	}
}
