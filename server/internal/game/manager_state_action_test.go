package game

import (
	"encoding/json"
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
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

func TestDelayedMermaidsTownUsesPostActionWindowAsOwnTurn(t *testing.T) {
	state := NewGameState()
	if err := state.AddPlayer("mermaids", factions.NewMermaids()); err != nil {
		t.Fatal(err)
	}
	if err := state.AddPlayer("opponent", factions.NewEngineers()); err != nil {
		t.Fatal(err)
	}
	state.TurnOrder = []string{"mermaids", "opponent"}
	state.CurrentPlayerIndex = 1
	state.Phase = PhaseAction
	river := setupMermaidChoiceBuildings(state, "mermaids")
	action := &SelectTownTileAction{BaseAction: BaseAction{Type: ActionSelectTownTile, PlayerID: "mermaids"}, AnchorHex: &river}

	if err := validateActionTurnAndPendingState(state, action); err == nil {
		t.Fatal("delayed town was accepted outside the Mermaid player's action window")
	}
	state.PendingFreeActionsPlayerID = "mermaids"
	if err := validateActionTurnAndPendingState(state, action); err != nil {
		t.Fatalf("delayed town rejected during Mermaid post-action window: %v", err)
	}
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

// Designer-referenced FAQ 2.10: neighbors decide on construction power before
// the builder chooses favor/town rewards. Passing does not waive that response.
func TestLeechPrecedesBuilderRewardsIncludingPassedResponder(t *testing.T) {
	for _, reward := range []string{"favor", "town"} {
		for _, passed := range []bool{false, true} {
			state := NewGameState()
			state.AddPlayer("actor", factions.NewWitches())
			state.AddPlayer("responder", factions.NewEngineers())
			state.TurnOrder = []string{"actor", "responder"}
			state.Phase = PhaseAction
			state.SetupSubphase = SetupSubphaseComplete
			state.GetPlayer("responder").HasPassed = passed
			state.PendingLeechOffers["responder"] = []*PowerLeechOffer{{FromPlayerID: "actor", Amount: 2}}
			var premature Action
			if reward == "favor" {
				state.PendingFavorTileSelection = &PendingFavorTileSelection{PlayerID: "actor", Count: 1}
				premature = &SelectFavorTileAction{BaseAction: BaseAction{Type: ActionSelectFavorTile, PlayerID: "actor"}, TileType: FavorFire3}
			} else {
				state.PendingTownFormations["actor"] = []*PendingTownFormation{{PlayerID: "actor"}}
				premature = &SelectTownTileAction{BaseAction: BaseAction{Type: ActionSelectTownTile, PlayerID: "actor"}}
			}
			if owners := DecisionPlayerIDs(state); len(owners) != 1 || owners[0] != "responder" {
				t.Fatalf("%s passed=%v: wrong decision owner %v", reward, passed, owners)
			}
			if pending := serializePendingDecision(state).(map[string]interface{}); pending["type"] != "leech_offer" || pending["playerId"] != "responder" {
				t.Fatalf("wrong serialized reaction: %v", pending)
			}
			if err := ApplyActionToState(state, premature, StateActionOptions{}); err == nil {
				t.Fatalf("builder chose %s before leech", reward)
			}
			if err := ApplyActionToState(state, NewDeclinePowerLeechAction("responder", 0), StateActionOptions{}); err != nil {
				t.Fatalf("leech resolution blocked by %s: %v", reward, err)
			}
			if owners := DecisionPlayerIDs(state); len(owners) != 1 || owners[0] != "actor" {
				t.Fatalf("%s decision not restored after leech: %v", reward, owners)
			}
		}
	}
}

func TestLeechRespondersFollowCurrentTurnOrder(t *testing.T) {
	state := NewGameState()
	state.AddPlayer("actor", factions.NewWitches())
	state.AddPlayer("first", factions.NewEngineers())
	state.AddPlayer("second", factions.NewNomads())
	state.TurnOrder = []string{"actor", "first", "second"}
	state.Phase = PhaseAction
	state.SetupSubphase = SetupSubphaseComplete
	for _, id := range []string{"first", "second"} {
		state.PendingLeechOffers[id] = []*PowerLeechOffer{{FromPlayerID: "actor", Amount: 1}}
	}
	if err := ApplyActionToState(state, NewDeclinePowerLeechAction("second", 0), StateActionOptions{}); err == nil {
		t.Fatal("second responder bypassed first")
	}
	if err := ApplyActionToState(state, NewDeclinePowerLeechAction("first", 0), StateActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := state.GetNextBlockingLeechResponder(); got != "second" {
		t.Fatalf("next responder=%s", got)
	}
	if err := ApplyActionToState(state, NewDeclinePowerLeechAction("second", 0), StateActionOptions{}); err != nil {
		t.Fatal(err)
	}
}

func simultaneousRewardFixture(t *testing.T, faction models.FactionType, sixPower bool) (*GameState, board.Hex) {
	t.Helper()
	state := NewGameState()
	state.AddPlayer("actor", factions.NewFaction(faction))
	state.AddPlayer("other", factions.NewNomads())
	state.TurnOrder = []string{"actor", "other"}
	state.Phase = PhaseAction
	state.SetupSubphase = SetupSubphaseComplete
	for q := 0; q < 4; q++ {
		h := board.NewHex(q, 0)
		kind, power := models.BuildingTradingHouse, 2
		if q == 3 || (sixPower && q == 2) {
			kind, power = models.BuildingDwelling, 1
		}
		state.Map.GetHex(h).Terrain = state.GetPlayer("actor").Faction.GetHomeTerrain()
		if err := state.Map.PlaceBuilding(h, &models.Building{Type: kind, PowerValue: power, Faction: faction, PlayerID: "actor"}); err != nil {
			t.Fatal(err)
		}
	}
	state.CheckAllTownFormations("actor")
	return state, board.NewHex(0, 0)
}

func TestSimultaneousDarklingsTownAndOrdinationEitherOrder(t *testing.T) {
	// FAQ 2.10 expressly allows the town's two workers to feed SH ordination.
	for _, townFirst := range []bool{false, true} {
		state, anchor := simultaneousRewardFixture(t, models.FactionDarklings, false)
		player := state.GetPlayer("actor")
		player.Resources.Workers, player.Resources.Priests = 1, 0
		state.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{PlayerID: "actor"}
		town := &SelectTownTileAction{BaseAction: BaseAction{Type: ActionSelectTownTile, PlayerID: "actor"}, TileType: models.TownTile7Points, AnchorHex: &anchor}
		amount := 1
		if townFirst {
			amount = 3
			if err := ApplyActionToState(state, town, StateActionOptions{}); err != nil {
				t.Fatal(err)
			}
		}
		ordination := &UseDarklingsPriestOrdinationAction{BaseAction: BaseAction{Type: ActionUseDarklingsPriestOrdination, PlayerID: "actor"}, WorkersToConvert: amount}
		if err := ApplyActionToState(state, ordination, StateActionOptions{}); err != nil {
			t.Fatalf("townFirst=%v: %v", townFirst, err)
		}
		if !townFirst {
			if err := ApplyActionToState(state, town, StateActionOptions{}); err != nil {
				t.Fatal(err)
			}
		}
		if player.Resources.Priests != amount || player.Resources.Workers != 3-amount {
			t.Fatalf("wrong reward ordering outcome: workers=%d priests=%d", player.Resources.Workers, player.Resources.Priests)
		}
	}
}

func TestChaosFirstFireFavorCreatesImmediateTownKey(t *testing.T) {
	state, anchor := simultaneousRewardFixture(t, models.FactionChaosMagicians, true)
	player := state.GetPlayer("actor")
	state.CultTracks.PlayerPositions["actor"][CultFire] = 8
	player.CultPositions[CultFire] = 8
	state.PendingFavorTileSelection = &PendingFavorTileSelection{PlayerID: "actor", Count: 2}
	first := &SelectFavorTileAction{BaseAction: BaseAction{Type: ActionSelectFavorTile, PlayerID: "actor"}, TileType: FavorFire2}
	if err := ApplyActionToState(state, first, StateActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(state.PendingTownFormations["actor"]) != 1 || state.CultTracks.GetPosition("actor", CultFire) != 10 || player.Keys != -1 {
		t.Fatal("first Fire2 must immediately create and borrow town key to reach Fire10")
	}
	// The second favor remains legal before selecting the simultaneously founded town.
	second := &SelectFavorTileAction{BaseAction: BaseAction{Type: ActionSelectFavorTile, PlayerID: "actor"}, TileType: FavorAir3}
	if err := ApplyActionToState(state, second, StateActionOptions{}); err != nil {
		t.Fatal(err)
	}
	player.Resources.Priests = 7
	// Rulebook pp.14,18 grants VP/key separately from a capped priest reward.
	town := &SelectTownTileAction{BaseAction: BaseAction{Type: ActionSelectTownTile, PlayerID: "actor"}, TileType: models.TownTile9Points, AnchorHex: &anchor}
	if err := ApplyActionToState(state, town, StateActionOptions{}); err != nil {
		t.Fatal(err)
	}
	if player.Keys != 0 || player.Resources.Priests != 7 {
		t.Fatalf("town key debt/priest cap wrong: keys=%d priests=%d", player.Keys, player.Resources.Priests)
	}
}
