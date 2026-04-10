package game

import (
	"strings"
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func postActionPendingDecisionMap(t *testing.T, gs *GameState) map[string]interface{} {
	t.Helper()
	raw := serializePendingDecision(gs)
	pending, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("serializePendingDecision() = %T, want map[string]interface{}", raw)
	}
	return pending
}

func TestManager_PostActionFreeWindow_AllowsActorConversionAfterMainAction(t *testing.T) {
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

	next := gs.GetPlayer("next")
	if next == nil || next.Resources == nil {
		t.Fatal("missing next player")
	}
	next.Resources.Priests = 1

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "actor"},
		Track:         CultFire,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor send_priest: %v", err)
	}

	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "next" {
		t.Fatalf("current player after main action = %v, want next", current)
	}
	if got := strings.TrimSpace(gs.PendingFreeActionsPlayerID); got != "actor" {
		t.Fatalf("pending free-actions player = %q, want actor", got)
	}
	if !gs.HasPendingTurnConfirmation() {
		t.Fatal("expected pending turn confirmation after main action")
	}
	if pending := postActionPendingDecisionMap(t, gs); pending["type"] != "post_action_free_actions" || pending["playerId"] != "actor" {
		t.Fatalf("pending decision = %v, want post_action_free_actions for actor", pending)
	}

	startCoins := actor.Resources.Coins
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
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "next" {
		t.Fatalf("current player after free conversion = %v, want next", current)
	}
	if got := strings.TrimSpace(gs.PendingFreeActionsPlayerID); got != "actor" {
		t.Fatalf("pending free-actions player after conversion = %q, want actor", got)
	}
}

func TestManager_PostActionFreeWindow_SkipsWindowWhenConfirmTurnDisabled(t *testing.T) {
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
	next := gs.GetPlayer("next")
	if actor == nil || next == nil || actor.Resources == nil || next.Resources == nil {
		t.Fatal("missing player resources")
	}
	actor.Options.ConfirmActions = false
	actor.Resources.Priests = 1
	next.Resources.Priests = 1

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "actor"},
		Track:         CultFire,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor send_priest: %v", err)
	}

	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "next" {
		t.Fatalf("current player after main action = %v, want next", current)
	}
	if got := strings.TrimSpace(gs.PendingFreeActionsPlayerID); got != "" {
		t.Fatalf("pending free-actions player = %q, want empty", got)
	}
	if gs.HasPendingTurnConfirmation() {
		t.Fatal("expected no pending turn confirmation when confirm turn is disabled")
	}
	if pending := serializePendingDecision(gs); pending != nil {
		t.Fatalf("pending decision = %v, want nil", pending)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "next"},
		Track:         CultWater,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("next send_priest without confirm window: %v", err)
	}
}

func TestManager_PostActionFreeWindow_BlocksNextPlayerUntilActorConfirms(t *testing.T) {
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
	next := gs.GetPlayer("next")
	if actor == nil || next == nil || actor.Resources == nil || next.Resources == nil || actor.Resources.Power == nil {
		t.Fatal("missing player resources")
	}
	actor.Resources.Priests = 1
	actor.Resources.Power.Bowl3 = 1
	next.Resources.Priests = 1

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "actor"},
		Track:         CultFire,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor send_priest: %v", err)
	}

	_, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "next"},
		Track:         CultWater,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 1})
	if err == nil {
		t.Fatal("expected next player action to be blocked before confirm")
	}
	if !strings.Contains(err.Error(), "turn confirmation pending for player actor") {
		t.Fatalf("unexpected blocked-action error: %v", err)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", NewConfirmTurnAction("actor"), ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("actor confirm: %v", err)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "next"},
		Track:         CultWater,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 2}); err != nil {
		t.Fatalf("next send_priest after confirm: %v", err)
	}

	if got := strings.TrimSpace(gs.PendingFreeActionsPlayerID); got != "next" {
		t.Fatalf("pending free-actions player after next action = %q, want next", got)
	}

	_, err = mgr.ExecuteActionWithMeta("g1", &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "actor"},
		ConversionType: ConversionPowerToCoin,
		Amount:         1,
	}, ActionMeta{ExpectedRevision: 3})
	if err == nil {
		t.Fatal("expected actor conversion to be rejected after next player acted")
	}
	if !strings.Contains(err.Error(), "turn confirmation pending for player next") {
		t.Fatalf("unexpected actor conversion error: %v", err)
	}
}

func TestManager_PostActionFreeWindow_PendingResolutionAdvancesTurnAndKeepsFreeWindow(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("actor", factions.NewAuren()); err != nil {
		t.Fatalf("add actor: %v", err)
	}
	if err := gs.AddPlayer("next", factions.NewNomads()); err != nil {
		t.Fatalf("add next: %v", err)
	}
	gs.TurnOrder = []string{"actor", "next"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = PhaseAction
	gs.PendingFavorTileSelection = &PendingFavorTileSelection{
		PlayerID:      "actor",
		Count:         1,
		SelectedTiles: []FavorTileType{},
	}
	gs.PendingFreeActionsPlayerID = "actor"
	gs.BeginPendingTurnConfirmation("actor", gs.CloneForUndo())

	actor := gs.GetPlayer("actor")
	next := gs.GetPlayer("next")
	if actor == nil || next == nil || actor.Resources == nil || next.Resources == nil {
		t.Fatal("missing player resources")
	}
	actor.Resources.Priests = 1
	next.Resources.Priests = 1

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", &SelectFavorTileAction{
		BaseAction: BaseAction{Type: ActionSelectFavorTile, PlayerID: "actor"},
		TileType:   FavorWater2,
	}, ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("actor select_favor_tile: %v", err)
	}

	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "next" {
		t.Fatalf("current player after pending resolution = %v, want next", current)
	}
	if pending := postActionPendingDecisionMap(t, gs); pending["type"] != "post_action_free_actions" || pending["playerId"] != "actor" {
		t.Fatalf("pending decision after favor = %v, want post_action_free_actions for actor", pending)
	}
	if !gs.HasPendingTurnConfirmation() {
		t.Fatal("expected turn confirmation to remain pending after favor resolution")
	}

	actor.Resources.Power.Bowl3 = 1
	startCoins := actor.Resources.Coins
	if _, err := mgr.ExecuteActionWithMeta("g1", &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "actor"},
		ConversionType: ConversionPowerToCoin,
		Amount:         1,
	}, ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("actor conversion after favor: %v", err)
	}
	if actor.Resources.Coins != startCoins+1 {
		t.Fatalf("actor coins after conversion = %d, want %d", actor.Resources.Coins, startCoins+1)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "next" {
		t.Fatalf("current player after actor conversion = %v, want next", current)
	}

	_, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "next"},
		Track:         CultWater,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 2})
	if err == nil {
		t.Fatal("expected next player action to be blocked before confirm")
	}
	if !strings.Contains(err.Error(), "turn confirmation pending for player actor") {
		t.Fatalf("unexpected blocked-action error: %v", err)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", NewConfirmTurnAction("actor"), ActionMeta{ExpectedRevision: 2}); err != nil {
		t.Fatalf("actor confirm after favor: %v", err)
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", &SendPriestToCultAction{
		BaseAction:    BaseAction{Type: ActionSendPriestToCult, PlayerID: "next"},
		Track:         CultWater,
		SpacesToClimb: 1,
	}, ActionMeta{ExpectedRevision: 3}); err != nil {
		t.Fatalf("next send_priest after pending resolution confirm: %v", err)
	}
}
