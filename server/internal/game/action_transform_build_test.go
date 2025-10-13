package game

import (
	"testing"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Helper to create a building for tests
func testBuilding(playerID string, faction models.FactionType, buildingType models.BuildingType) *models.Building {
	powerValue := 1
	if buildingType == models.BuildingTradingHouse || buildingType == models.BuildingTemple {
		powerValue = 2
	} else if buildingType == models.BuildingSanctuary || buildingType == models.BuildingStronghold {
		powerValue = 3
	}
	return &models.Building{
		Type:       buildingType,
		Faction:    faction,
		PlayerID:   playerID,
		PowerValue: powerValue,
	}
}

func TestTransformAndBuild_ValidAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Give player enough resources
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling to establish adjacency
	initialHex := NewHex(0, 1) // Desert terrain
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Transform and build adjacent
	// Neighbors of (0,1) include (1,1), (0,2), (-1,2), (-1,1), (0,0), (1,0)
	// (1,1) is Plains (already home terrain), so use (0,0) which is Forest
	targetHex := NewHex(0, 0) // Forest terrain - directly adjacent to (0,1)
	initialTerrain := gs.Map.GetHex(targetHex).Terrain
	action := NewTransformAndBuildAction("player1", targetHex, true) // true = build dwelling
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Verify building was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Errorf("expected building to be placed")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
	
	// Verify terrain was transformed to Plains (Halflings home terrain)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be transformed from %v to Plains, got %v", initialTerrain, mapHex.Terrain)
	}
}

func TestTransformAndBuild_NotAdjacent(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Try to build without any adjacent buildings
	targetHex := NewHex(5, 5)
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for non-adjacent building")
	}
}

func TestTransformAndBuild_InsufficientResources(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	// Don't give enough resources
	player.Resources.Coins = 0
	player.Resources.Workers = 0
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Use adjacent hex (0,0) which is Forest
	targetHex := NewHex(0, 0)
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for insufficient resources")
	}
}

func TestTransformAndBuild_SkipTerraform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Build on home terrain (Plains for Halflings) - no transform needed
	// Use adjacent hex (1,1) which is Plains
	targetHex := NewHex(1, 1) // Plains terrain
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	initialWorkers := player.Resources.Workers
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Verify no workers were spent on terraform (only dwelling cost)
	dwellingCost := faction.GetDwellingCost()
	expectedWorkers := initialWorkers - dwellingCost.Workers
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
}

func TestTransformAndBuild_PowerLeech(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewCultists()
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	player1.Resources.Coins = 10
	player1.Resources.Workers = 10
	
	// Place player1's initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction1.GetType(), models.BuildingDwelling)
	
	// Place player2's dwelling adjacent to where player1 will build
	player2Hex := NewHex(2, 2)
	gs.Map.GetHex(player2Hex).Building = testBuilding("player2", faction2.GetType(), models.BuildingDwelling)
	
	// Player1 builds adjacent to both buildings
	// Use (1,1) which is adjacent to (0,1) and close to (2,2)
	targetHex := NewHex(1, 1)
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	// Record player2's initial power
	initialPower2Bowl1 := player2.Resources.Power.Bowl1
	initialPower2Bowl2 := player2.Resources.Power.Bowl2
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected action to succeed, got error: %v", err)
	}
	
	// Power leech is triggered but not automatically accepted
	// The offer is created in TriggerPowerLeech
	// For now, we just verify the action executed successfully
	// TODO: When Phase 6.1 is complete, verify power leech offers are stored
	
	// Verify building was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Errorf("expected building to be placed")
	}
	
	// Power should not have changed yet (player2 hasn't accepted/declined)
	if player2.Resources.Power.Bowl1 != initialPower2Bowl1 {
		t.Errorf("power should not change until leech is accepted")
	}
	if player2.Resources.Power.Bowl2 != initialPower2Bowl2 {
		t.Errorf("power should not change until leech is accepted")
	}
}

func TestTransformAndBuild_HexAlreadyOccupied(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Try to build on occupied hex
	action := NewTransformAndBuildAction("player1", initialHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for building on occupied hex")
	}
}

func TestTransformAndBuild_TransformOnly(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Transform adjacent hex WITHOUT building (buildDwelling = false)
	// Use (0,0) which is Forest
	targetHex := NewHex(0, 0)
	action := NewTransformAndBuildAction("player1", targetHex, false)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected transform-only to succeed, got error: %v", err)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be transformed to Plains, got %v", mapHex.Terrain)
	}
	
	// Verify NO building was placed
	if mapHex.Building != nil {
		t.Errorf("expected no building, but found %v", mapHex.Building.Type)
	}
}

func TestTransformAndBuild_BuildOnNonHomeTerrain(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	
	// Place initial dwelling
	initialHex := NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = testBuilding("player1", faction.GetType(), models.BuildingDwelling)
	
	// Try to build on non-home terrain without transforming
	// (This should fail because dwelling requires home terrain)
	targetHex := NewHex(0, 0) // Forest terrain (not Plains)
	// First transform it to something else (not home terrain)
	gs.Map.TransformTerrain(targetHex, models.TerrainDesert)
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for building on non-home terrain")
	}
}
