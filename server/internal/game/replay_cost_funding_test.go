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
