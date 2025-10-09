package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestMapInitialization(t *testing.T) {
	m := NewTerraMysticaMap()
	
	// Check that we have the correct number of hexes
	// 9 rows alternating 13/12: 13+12+13+12+13+12+13+12+13 = 113 hexes
	expectedHexCount := 13 + 12 + 13 + 12 + 13 + 12 + 13 + 12 + 13
	if len(m.Hexes) != expectedHexCount {
		t.Errorf("Expected %d hexes, got %d", expectedHexCount, len(m.Hexes))
	}
	
	// Check that some specific hexes exist with correct terrain
	testHexes := []struct {
		hex     Hex
		terrain models.TerrainType
	}{
		{NewHex(0, 0), models.TerrainForest},
		{NewHex(6, 0), models.TerrainWasteland},
		{NewHex(12, 0), models.TerrainForest},
		{NewHex(0, 4), models.TerrainSwamp},
		{NewHex(6, 4), models.TerrainForest},
	}
	
	for _, tt := range testHexes {
		mapHex := m.GetHex(tt.hex)
		if mapHex == nil {
			t.Errorf("Hex %v should exist", tt.hex)
			continue
		}
		if mapHex.Terrain != tt.terrain {
			t.Errorf("Hex %v: expected terrain %v, got %v", tt.hex, tt.terrain, mapHex.Terrain)
		}
	}
}

func TestDirectAdjacency(t *testing.T) {
	m := NewTerraMysticaMap()
	
	// Test basic adjacency (no rivers/bridges)
	h1 := NewHex(5, 5)
	h2 := NewHex(6, 5) // East neighbor
	
	if !m.IsDirectlyAdjacent(h1, h2) {
		t.Errorf("Adjacent hexes %v and %v should be directly adjacent", h1, h2)
	}
	
	// Test non-adjacent hexes
	h3 := NewHex(7, 5) // Distance 2
	if m.IsDirectlyAdjacent(h1, h3) {
		t.Errorf("Non-adjacent hexes %v and %v should not be directly adjacent", h1, h3)
	}
}

func TestBridgeBuilding(t *testing.T) {
	m := NewTerraMysticaMap()
	
	h1 := NewHex(5, 5)
	h2 := NewHex(6, 5)
	playerID := "player1"
	
	// Build first bridge
	err := m.BuildBridge(h1, h2, playerID)
	if err != nil {
		t.Errorf("Failed to build first bridge: %v", err)
	}
	
	// Check bridge exists
	if !m.HasBridge(h1, h2) {
		t.Errorf("Bridge should exist between %v and %v", h1, h2)
	}
	
	// Check bridge count
	if m.PlayerBridges[playerID] != 1 {
		t.Errorf("Player should have 1 bridge, got %d", m.PlayerBridges[playerID])
	}
	
	// Try to build duplicate bridge
	err = m.BuildBridge(h1, h2, playerID)
	if err == nil {
		t.Errorf("Should not be able to build duplicate bridge")
	}
	
	// Build second and third bridges
	h3 := NewHex(5, 6)
	h4 := NewHex(4, 5)
	h5 := NewHex(4, 6)
	
	m.BuildBridge(h1, h3, playerID)
	m.BuildBridge(h1, h4, playerID)
	
	if m.PlayerBridges[playerID] != 3 {
		t.Errorf("Player should have 3 bridges, got %d", m.PlayerBridges[playerID])
	}
	
	// Try to build fourth bridge (should fail)
	err = m.BuildBridge(h4, h5, playerID)
	if err == nil {
		t.Errorf("Should not be able to build more than 3 bridges")
	}
	
	if m.PlayerBridges[playerID] != 3 {
		t.Errorf("Player should still have 3 bridges after failed attempt, got %d", m.PlayerBridges[playerID])
	}
}

func TestBridgeBetweenPlayers(t *testing.T) {
	m := NewTerraMysticaMap()
	
	h1 := NewHex(5, 5)
	h2 := NewHex(6, 5)
	h3 := NewHex(5, 6)
	h4 := NewHex(6, 6)
	
	// Player 1 builds 3 bridges
	m.BuildBridge(h1, h2, "player1")
	m.BuildBridge(h1, h3, "player1")
	m.BuildBridge(h2, h4, "player1")
	
	// Player 2 should be able to build their own bridges
	h5 := NewHex(7, 5)
	h6 := NewHex(8, 5)
	err := m.BuildBridge(h5, h6, "player2")
	if err != nil {
		t.Errorf("Player 2 should be able to build bridges: %v", err)
	}
	
	if m.PlayerBridges["player1"] != 3 {
		t.Errorf("Player 1 should have 3 bridges")
	}
	if m.PlayerBridges["player2"] != 1 {
		t.Errorf("Player 2 should have 1 bridge")
	}
}

func TestIndirectAdjacencyShipping(t *testing.T) {
	m := NewTerraMysticaMap()
	
	// Test various shipping levels
	tests := []struct {
		h1           Hex
		h2           Hex
		shippingLevel int
		expected     bool
		description  string
	}{
		// Adjacent hexes are not indirectly adjacent
		{NewHex(5, 5), NewHex(6, 5), 1, false, "Adjacent hexes not indirect"},
		
		// Distance 2, shipping 1 - should be false
		{NewHex(5, 5), NewHex(7, 5), 1, false, "Distance 2, shipping 1"},
		
		// Distance 2, shipping 2 - should be true
		{NewHex(5, 5), NewHex(7, 5), 2, true, "Distance 2, shipping 2"},
		
		// Distance 3, shipping 2 - should be false
		{NewHex(5, 5), NewHex(8, 5), 2, false, "Distance 3, shipping 2"},
		
		// Distance 3, shipping 3 - should be true
		{NewHex(5, 5), NewHex(8, 5), 3, true, "Distance 3, shipping 3"},
		
		// Distance 4, shipping 4
		{NewHex(5, 5), NewHex(9, 5), 4, true, "Distance 4, shipping 4"},
		
		// Distance 5, shipping 5
		{NewHex(5, 5), NewHex(10, 5), 5, true, "Distance 5, shipping 5"},
		
		// Distance 6, shipping 6
		{NewHex(5, 5), NewHex(11, 5), 6, true, "Distance 6, shipping 6"},
		
		// Distance 6, shipping 5 - should be false
		{NewHex(5, 5), NewHex(11, 5), 5, false, "Distance 6, shipping 5"},
	}
	
	for _, tt := range tests {
		result := m.IsIndirectlyAdjacent(tt.h1, tt.h2, tt.shippingLevel)
		if result != tt.expected {
			t.Errorf("%s: IsIndirectlyAdjacent(%v, %v, %d) = %v, expected %v",
				tt.description, tt.h1, tt.h2, tt.shippingLevel, result, tt.expected)
		}
	}
}

func TestBuildingPlacement(t *testing.T) {
	m := NewTerraMysticaMap()
	
	h := NewHex(5, 5)
	building := &models.Building{
		OwnerPlayerID: "player1",
		Faction:       models.FactionNomads,
		Type:          models.BuildingDwelling,
	}
	
	// Place building
	err := m.PlaceBuilding(h, building)
	if err != nil {
		t.Errorf("Failed to place building: %v", err)
	}
	
	// Check building exists
	mapHex := m.GetHex(h)
	if mapHex.Building == nil {
		t.Errorf("Building should exist at %v", h)
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("Expected dwelling, got %v", mapHex.Building.Type)
	}
	
	// Try to place another building (should fail)
	building2 := &models.Building{
		OwnerPlayerID: "player2",
		Faction:       models.FactionGiants,
		Type:          models.BuildingDwelling,
	}
	err = m.PlaceBuilding(h, building2)
	if err == nil {
		t.Errorf("Should not be able to place building on occupied hex")
	}
}

func TestTerrainTransformation(t *testing.T) {
	m := NewTerraMysticaMap()
	
	h := NewHex(5, 5)
	originalTerrain := m.GetHex(h).Terrain
	newTerrain := models.TerrainDesert
	
	// Transform terrain
	err := m.TransformTerrain(h, newTerrain)
	if err != nil {
		t.Errorf("Failed to transform terrain: %v", err)
	}
	
	// Check terrain changed
	if m.GetHex(h).Terrain != newTerrain {
		t.Errorf("Terrain should be %v, got %v", newTerrain, m.GetHex(h).Terrain)
	}
	
	// Place building
	building := &models.Building{
		OwnerPlayerID: "player1",
		Faction:       models.FactionNomads,
		Type:          models.BuildingDwelling,
	}
	m.PlaceBuilding(h, building)
	
	// Try to transform terrain with building (should fail)
	err = m.TransformTerrain(h, originalTerrain)
	if err == nil {
		t.Errorf("Should not be able to transform terrain with building present")
	}
}

func TestGetDirectNeighbors(t *testing.T) {
	m := NewTerraMysticaMap()
	
	// Test hex in middle of map
	h := NewHex(5, 5)
	neighbors := m.GetDirectNeighbors(h)
	
	// Should have 6 neighbors (assuming no rivers blocking)
	if len(neighbors) != 6 {
		t.Errorf("Expected 6 neighbors, got %d", len(neighbors))
	}
	
	// Test corner hex (should have fewer valid neighbors)
	corner := NewHex(0, 0)
	cornerNeighbors := m.GetDirectNeighbors(corner)
	
	// Corner should have fewer than 6 neighbors (some are off-map)
	if len(cornerNeighbors) >= 6 {
		t.Errorf("Corner hex should have fewer than 6 valid neighbors, got %d", len(cornerNeighbors))
	}
	
	// All returned neighbors should be valid hexes
	for _, neighbor := range cornerNeighbors {
		if !m.IsValidHex(neighbor) {
			t.Errorf("GetDirectNeighbors returned invalid hex %v", neighbor)
		}
	}
}

func TestGetBuildingsInRange(t *testing.T) {
	m := NewTerraMysticaMap()
	
	center := NewHex(5, 5)
	
	// Place buildings at various distances
	buildings := []struct {
		hex      Hex
		distance int
	}{
		{NewHex(5, 5), 0}, // Center
		{NewHex(6, 5), 1}, // Distance 1
		{NewHex(7, 5), 2}, // Distance 2
		{NewHex(8, 5), 3}, // Distance 3
	}
	
	for i, b := range buildings {
		building := &models.Building{
			OwnerPlayerID: "player1",
			Faction:       models.FactionNomads,
			Type:          models.BuildingDwelling,
		}
		m.PlaceBuilding(b.hex, building)
		
		// Check buildings in range
		inRange := m.GetBuildingsInRange(center, b.distance)
		expectedCount := i + 1
		if len(inRange) != expectedCount {
			t.Errorf("At distance %d, expected %d buildings in range, got %d",
				b.distance, expectedCount, len(inRange))
		}
	}
}
