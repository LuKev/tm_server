package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestEngineersBridgeAction_Execute(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewEngineers())
	player := gs.GetPlayer("player1")
	player.HasStrongholdAbility = true
	player.Resources.Workers = 5

	hex1 := board.NewHex(0, 0)
	river1 := board.NewHex(0, -1)
	river2 := board.NewHex(1, -1)
	hex2 := board.NewHex(1, -2)

	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: models.TerrainMountain}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: models.TerrainMountain}
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	gs.Map.Hexes[hex1].Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionEngineers,
		PlayerID:   "player1",
		PowerValue: 1,
	}

	action := NewEngineersBridgeAction("player1", hex1, hex2)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("expected engineers bridge action to succeed, got: %v", err)
	}

	if player.Resources.Workers != 3 {
		t.Fatalf("expected workers to decrease by 2, got %d", player.Resources.Workers)
	}
	if !gs.Map.HasBridge(hex1, hex2) {
		t.Fatalf("expected bridge to be present on map")
	}
	if player.BridgesBuilt != 1 {
		t.Fatalf("expected bridge count 1, got %d", player.BridgesBuilt)
	}
}

func TestEngineersBridgeAction_RequiresStronghold(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewEngineers())
	player := gs.GetPlayer("player1")
	player.Resources.Workers = 5

	hex1 := board.NewHex(0, 0)
	hex2 := board.NewHex(1, -2)
	river1 := board.NewHex(0, -1)
	river2 := board.NewHex(1, -1)

	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: models.TerrainMountain}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: models.TerrainMountain}
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	gs.Map.Hexes[hex1].Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionEngineers,
		PlayerID:   "player1",
		PowerValue: 1,
	}

	action := NewEngineersBridgeAction("player1", hex1, hex2)
	if err := action.Validate(gs); err == nil {
		t.Fatalf("expected validation error without stronghold ability")
	}
}
