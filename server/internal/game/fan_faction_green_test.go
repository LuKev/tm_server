package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestChashDallahIncomeTrack_StartAndAdvance(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChashDallah()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	gs.GrantIncome()
	if got := player.Resources.Coins; got != 17 {
		t.Fatalf("round-one coins = %d, want 17", got)
	}

	player.Resources.Coins = 15
	player.Resources.Workers = 3
	action := NewAdvanceChashTrackAction("p1")
	if err := action.Execute(gs); err != nil {
		t.Fatalf("AdvanceChashTrackAction.Execute failed: %v", err)
	}

	if got := player.ChashIncomeTrackLevel; got != 1 {
		t.Fatalf("track level = %d, want 1", got)
	}
	if got := player.VictoryPoints; got != 21 {
		t.Fatalf("victory points = %d, want 21", got)
	}

	player.Resources.Coins = 15
	player.Resources.Workers = 3
	player.ChashIncomeTrackLevel = 1
	gs.GrantIncome()
	if got := player.Resources.Coins; got != 15 {
		t.Fatalf("coins after level-1 income = %d, want 15", got)
	}
	if got := player.Resources.Workers; got != 5 {
		t.Fatalf("workers after level-1 income = %d, want 5", got)
	}
}

func TestChashDallahCannotAdvanceDigging(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChashDallah()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	err := NewAdvanceDiggingAction("p1").Validate(gs)
	if err == nil || err.Error() != "chash dallah cannot advance digging level" {
		t.Fatalf("Validate error = %v, want chash digging rejection", err)
	}
}

func TestChashDallahPowerActionUsesCoinsAfterStronghold(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewChashDallah()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Coins = 10
	player.Resources.Power = NewPowerSystem(0, 0, 0)

	action := NewPowerAction("p1", PowerActionCoins)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("PowerAction.Execute failed: %v", err)
	}

	if got := player.Resources.Coins; got != 13 {
		t.Fatalf("coins after coin-paid power action = %d, want 13", got)
	}
	if got := player.Resources.Power.Bowl3; got != 0 {
		t.Fatalf("bowl III = %d, want 0", got)
	}
}

func TestEnlightenedCoinToPowerConversion(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTheEnlightened()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	startCoins := player.Resources.Coins
	startBowl1 := player.Resources.Power.Bowl1

	action := &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "p1"},
		ConversionType: ConversionCoinToPower,
		Amount:         1,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("ConversionAction.Execute failed: %v", err)
	}

	if got := player.Resources.Coins; got != startCoins-1 {
		t.Fatalf("coins = %d, want %d", got, startCoins-1)
	}
	if got := player.Resources.Power.Bowl1; got != startBowl1+1 {
		t.Fatalf("bowl I = %d, want %d", got, startBowl1+1)
	}
}

func TestEnlightenedStrongholdConversionsAndAction(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTheEnlightened()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Power = NewPowerSystem(2, 2, 6)

	workerConversion := &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "p1"},
		ConversionType: ConversionPowerToWorker,
		Amount:         1,
	}
	if err := workerConversion.Execute(gs); err != nil {
		t.Fatalf("worker conversion failed: %v", err)
	}
	if got := player.Resources.Workers; got != 5 {
		t.Fatalf("workers = %d, want 5", got)
	}

	player.SpecialActionsUsed = make(map[SpecialActionType]bool)
	player.Resources.Power = NewPowerSystem(2, 2, 0)
	if err := NewEnlightenedGainPowerAction("p1").Execute(gs); err != nil {
		t.Fatalf("Enlightened gain-power action failed: %v", err)
	}
	if player.Resources.Power.Bowl1 != 0 || player.Resources.Power.Bowl2 != 2 || player.Resources.Power.Bowl3 != 2 {
		t.Fatalf("unexpected power state after SH action: %d/%d/%d", player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
	}
}

func TestEnlightenedTerraformUsesPower(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTheEnlightened()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 10
	player.Resources.Workers = 1
	player.Resources.Power = NewPowerSystem(0, 0, 9)

	initialHex := board.NewHex(0, 0)
	targetHex := board.NewHex(0, 1)
	gs.Map.GetHex(initialHex).Terrain = models.TerrainForest
	gs.Map.GetHex(initialHex).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)

	distance := gs.Map.GetTerrainDistance(gs.Map.GetHex(targetHex).Terrain, player.Faction.GetHomeTerrain())
	expectedPowerSpend := player.Faction.GetTerraformCost(distance)

	action := NewTransformAndBuildAction("p1", targetHex, false, models.TerrainTypeUnknown)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("TransformAndBuildAction.Execute failed: %v", err)
	}

	if got := player.Resources.Workers; got != 1 {
		t.Fatalf("workers = %d, want 1", got)
	}
	if got := player.Resources.Power.Bowl1; got != expectedPowerSpend {
		t.Fatalf("bowl I = %d, want %d", got, expectedPowerSpend)
	}
	if got := player.Resources.Power.Bowl3; got != 9-expectedPowerSpend {
		t.Fatalf("bowl III = %d, want %d", got, 9-expectedPowerSpend)
	}
}
