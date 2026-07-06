package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestUpgradeBuildingAction_ReplayAutoConvertsPowerToWorker(t *testing.T) {
	gs := NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true, "__bga__": true}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"acolytes"}

	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("acolytes")
	player.Resources.Coins = 8
	player.Resources.Workers = 3
	player.Resources.Priests = 0
	player.Resources.Power = NewPowerSystem(0, 2, 8)

	hex := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(hex, models.TerrainVolcano); err != nil {
		t.Fatalf("set terrain: %v", err)
	}
	gs.Map.PlaceBuilding(hex, &models.Building{
		Type:       models.BuildingTemple,
		PlayerID:   "acolytes",
		Faction:    models.FactionAcolytes,
		PowerValue: 2,
	})

	action := NewUpgradeBuildingAction("acolytes", hex, models.BuildingSanctuary)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("execute sanctuary upgrade: %v", err)
	}

	if got := gs.Map.GetHex(hex).Building.Type; got != models.BuildingSanctuary {
		t.Fatalf("building type = %v, want sanctuary", got)
	}
	if got := player.Resources.Coins; got != 0 {
		t.Fatalf("coins after replay funding = %d, want 0", got)
	}
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers after replay funding = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl3; got != 5 {
		t.Fatalf("bowl III after replay funding = %d, want 5", got)
	}
}

func TestUpgradeBuildingAction_NormalModeStillRequiresExplicitConversions(t *testing.T) {
	gs := NewGameState()
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"acolytes"}

	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("acolytes")
	player.Resources.Coins = 8
	player.Resources.Workers = 3
	player.Resources.Priests = 0
	player.Resources.Power = NewPowerSystem(0, 2, 8)

	hex := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(hex, models.TerrainVolcano); err != nil {
		t.Fatalf("set terrain: %v", err)
	}
	gs.Map.PlaceBuilding(hex, &models.Building{
		Type:       models.BuildingTemple,
		PlayerID:   "acolytes",
		Faction:    models.FactionAcolytes,
		PowerValue: 2,
	})

	action := NewUpgradeBuildingAction("acolytes", hex, models.BuildingSanctuary)
	if err := action.Validate(gs); err == nil {
		t.Fatal("expected sanctuary upgrade to require explicit conversions outside replay mode")
	}
}

func TestUpgradeBuildingAction_ReplayDoesNotAutoConvertAlchemistsVPWhenAffordable(t *testing.T) {
	gs := NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true, "__bga__": true}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"alchemists"}

	if err := gs.AddPlayer("alchemists", factions.NewAlchemists()); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("alchemists")
	player.VictoryPoints = 30
	player.Resources.Coins = 4
	player.Resources.Workers = 2
	player.Resources.Priests = 1

	hex := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(hex, models.TerrainSwamp); err != nil {
		t.Fatalf("set terrain: %v", err)
	}
	gs.Map.PlaceBuilding(hex, &models.Building{
		Type:       models.BuildingTradingHouse,
		PlayerID:   "alchemists",
		Faction:    models.FactionAlchemists,
		PowerValue: 2,
	})

	action := NewUpgradeBuildingAction("alchemists", hex, models.BuildingTemple)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("execute temple upgrade: %v", err)
	}

	if got := player.VictoryPoints; got != 30 {
		t.Fatalf("VP after replay funding = %d, want 30", got)
	}
	if got := player.Resources.Coins; got != 0 {
		t.Fatalf("coins after replay funding = %d, want 0", got)
	}
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers after replay funding = %d, want 0", got)
	}
	if got := player.Resources.Priests; got != 0 {
		t.Fatalf("priests after replay funding = %d, want 0", got)
	}
}

func TestTransformAndBuildAction_ReplayAutoConvertsPowerToWorker(t *testing.T) {
	gs := NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true, "__bga__": true}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"witches"}

	if err := gs.AddPlayer("witches", factions.NewWitches()); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("witches")
	player.Resources.Coins = 2
	player.Resources.Workers = 0
	player.Resources.Priests = 0
	player.Resources.Power = NewPowerSystem(0, 0, 3)

	existingHex := board.NewHex(0, 0)
	targetHex := board.NewHex(1, 0)
	if err := gs.Map.TransformTerrain(existingHex, models.TerrainForest); err != nil {
		t.Fatalf("set existing terrain: %v", err)
	}
	if err := gs.Map.TransformTerrain(targetHex, models.TerrainForest); err != nil {
		t.Fatalf("set target terrain: %v", err)
	}
	gs.Map.PlaceBuilding(existingHex, &models.Building{
		Type:       models.BuildingDwelling,
		PlayerID:   "witches",
		Faction:    models.FactionWitches,
		PowerValue: 1,
	})

	action := NewTransformAndBuildAction("witches", targetHex, true, models.TerrainTypeUnknown)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("execute replay build: %v", err)
	}

	if building := gs.Map.GetHex(targetHex).Building; building == nil || building.Type != models.BuildingDwelling {
		t.Fatalf("expected dwelling at target, got %+v", building)
	}
	if got := player.Resources.Coins; got != 0 {
		t.Fatalf("coins after replay build = %d, want 0", got)
	}
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers after replay build = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl1; got != 3 {
		t.Fatalf("bowl I after replay build = %d, want 3", got)
	}
}

func TestPlanReplayAutoCostSkipsConversionsWhenAlreadyAffordable(t *testing.T) {
	player := replayFundingPlayer(t, factions.NewWitches())
	player.Resources.Coins = 5
	player.Resources.Workers = 3
	player.Resources.Priests = 1
	player.Resources.Power = NewPowerSystem(0, 0, 3)

	plan, ok := planReplayAutoCost(player, factions.Cost{Coins: 5, Workers: 2})
	if !ok {
		t.Fatal("expected cost to be affordable")
	}
	if plan != (replayAutoCostPlan{}) {
		t.Fatalf("plan = %+v, want no conversions", plan)
	}
}

func TestPlanReplayAutoCostUsesPriestToWorkerOnlyForWorkerShortfall(t *testing.T) {
	player := replayFundingPlayer(t, factions.NewWitches())
	player.Resources.Coins = 10
	player.Resources.Workers = 2
	player.Resources.Priests = 1
	player.Resources.Power = NewPowerSystem(0, 0, 3)

	plan, ok := planReplayAutoCost(player, factions.Cost{Coins: 10, Workers: 3})
	if !ok {
		t.Fatal("expected cost to be affordable with priest conversion")
	}
	if plan.priestsToWorker != 1 {
		t.Fatalf("plan = %+v, want priest_to_worker only for worker shortfall", plan)
	}
}

func TestPlanReplayAutoCostUsesWorkerToCoinOnlyForCoinShortfall(t *testing.T) {
	player := replayFundingPlayer(t, factions.NewWitches())
	player.Resources.Coins = 4
	player.Resources.Workers = 3
	player.Resources.Priests = 0
	player.Resources.Power = NewPowerSystem(0, 0, 1)

	plan, ok := planReplayAutoCost(player, factions.Cost{Coins: 5, Workers: 2})
	if !ok {
		t.Fatal("expected cost to be affordable with worker conversion")
	}
	if plan.workersToCoins != 1 {
		t.Fatalf("plan = %+v, want worker_to_coin only for coin shortfall", plan)
	}
}

func TestPlanReplayAutoCostAlchemistsDoNotSpendVPWhenSurplusWorkersCanFundCoins(t *testing.T) {
	player := replayFundingPlayer(t, factions.NewAlchemists())
	player.VictoryPoints = 20
	player.Resources.Coins = 4
	player.Resources.Workers = 3
	player.Resources.Priests = 0
	player.Resources.Power = NewPowerSystem(0, 0, 0)

	plan, ok := planReplayAutoCost(player, factions.Cost{Coins: 5, Workers: 2})
	if !ok {
		t.Fatal("expected cost to be affordable with worker conversion")
	}
	if plan.vpToCoins != 0 || plan.workersToCoins != 1 {
		t.Fatalf("plan = %+v, want worker_to_coin and no VP_to_coin", plan)
	}
}

func TestPlanReplayAutoCostAlchemistsSpendVPOnlyWhenNeededForCoins(t *testing.T) {
	player := replayFundingPlayer(t, factions.NewAlchemists())
	player.VictoryPoints = 20
	player.Resources.Coins = 4
	player.Resources.Workers = 2
	player.Resources.Priests = 0
	player.Resources.Power = NewPowerSystem(0, 0, 0)

	plan, ok := planReplayAutoCost(player, factions.Cost{Coins: 5, Workers: 2})
	if !ok {
		t.Fatal("expected cost to be affordable with VP conversion")
	}
	if plan.vpToCoins != 1 || plan.workersToCoins != 0 {
		t.Fatalf("plan = %+v, want VP_to_coin only as last resort", plan)
	}
}

func replayFundingPlayer(t *testing.T, faction factions.Faction) *Player {
	t.Helper()
	gs := NewGameState()
	if err := gs.AddPlayer("p1", faction); err != nil {
		t.Fatalf("add player: %v", err)
	}
	return gs.GetPlayer("p1")
}
