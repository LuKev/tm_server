package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestBuildingPowerValue(t *testing.T) {
	tests := []struct {
		buildingType models.BuildingType
		expected     int
	}{
		{models.BuildingDwelling, 1},
		{models.BuildingTradingHouse, 2},
		{models.BuildingTemple, 2},
		{models.BuildingSanctuary, 3},
		{models.BuildingStronghold, 3},
	}
	
	for _, tt := range tests {
		result := BuildingPowerValue(tt.buildingType)
		if result != tt.expected {
			t.Errorf("BuildingPowerValue(%v) = %d, expected %d", tt.buildingType, result, tt.expected)
		}
	}
}

func TestTownDetectionMinimumBuildings(t *testing.T) {
	m := NewTerraMysticaMap()
	playerID := "player1"
	faction := models.FactionNomads
	
	// Place only 3 dwellings (not enough for town)
	hexes := []Hex{
		NewHex(5, 5),
		NewHex(6, 5),
		NewHex(7, 5),
	}
	
	for _, h := range hexes {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          models.BuildingDwelling,
		}
		m.PlaceBuilding(h, building)
	}
	
	// Should not form a town (need at least 4)
	town := m.DetectTown(hexes[0])
	if town != nil {
		t.Errorf("3 buildings should not form a town")
	}
}

func TestTownDetectionMinimumPower(t *testing.T) {
	m := NewTerraMysticaMap()
	playerID := "player1"
	faction := models.FactionNomads
	
	// Place 4 dwellings (4 buildings, but only 4 power - need 7)
	hexes := []Hex{
		NewHex(5, 5),
		NewHex(6, 5),
		NewHex(7, 5),
		NewHex(8, 5),
	}
	
	for _, h := range hexes {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          models.BuildingDwelling,
		}
		m.PlaceBuilding(h, building)
	}
	
	// Should not form a town (only 4 power, need 7)
	town := m.DetectTown(hexes[0])
	if town != nil {
		t.Errorf("4 dwellings (4 power) should not form a town (need 7 power)")
	}
}

func TestTownDetectionValidTown(t *testing.T) {
	m := NewTerraMysticaMap()
	playerID := "player1"
	faction := models.FactionNomads
	
	// Place 4 buildings with total power >= 7
	// 1 Stronghold (3) + 2 Trading Houses (2+2) + 1 Dwelling (1) = 8 power
	buildings := []struct {
		hex  Hex
		bType models.BuildingType
	}{
		{NewHex(5, 5), models.BuildingStronghold},
		{NewHex(6, 5), models.BuildingTradingHouse},
		{NewHex(7, 5), models.BuildingTradingHouse},
		{NewHex(8, 5), models.BuildingDwelling},
	}
	
	for _, b := range buildings {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          b.bType,
		}
		m.PlaceBuilding(b.hex, building)
	}
	
	// Should form a valid town
	town := m.DetectTown(buildings[0].hex)
	if town == nil {
		t.Fatalf("Should detect a valid town")
	}
	
	if len(town.Hexes) != 4 {
		t.Errorf("Town should have 4 hexes, got %d", len(town.Hexes))
	}
	
	if town.TotalPower != 8 {
		t.Errorf("Town should have 8 power, got %d", town.TotalPower)
	}
	
	if town.PlayerID != playerID {
		t.Errorf("Town playerID should be %s, got %s", playerID, town.PlayerID)
	}
	
	if town.Faction != faction {
		t.Errorf("Town faction should be %v, got %v", faction, town.Faction)
	}
}

func TestTownDetectionWithBridge(t *testing.T) {
	m := NewTerraMysticaMap()
	playerID := "player1"
	faction := models.FactionNomads
	
	// Place 4 buildings in a line
	hexes := []Hex{
		NewHex(5, 5),
		NewHex(6, 5),
		NewHex(7, 5),
		NewHex(8, 5),
	}
	
	// Place stronghold and temples (3+2+2+2 = 9 power)
	buildingTypes := []models.BuildingType{
		models.BuildingStronghold,
		models.BuildingTemple,
		models.BuildingTemple,
		models.BuildingTemple,
	}
	
	for i, h := range hexes {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          buildingTypes[i],
		}
		m.PlaceBuilding(h, building)
	}
	
	// Without bridge, all should be connected
	town := m.DetectTown(hexes[0])
	if town == nil {
		t.Fatalf("Should detect town without bridge")
	}
	
	// Now test that bridges can be used to connect buildings for town formation
	// Create a simple case: 4 buildings in a line, all connected
	m2 := NewTerraMysticaMap()
	
	// Place 4 buildings that form a valid town
	connectedHexes := []Hex{
		NewHex(5, 5),
		NewHex(6, 5),
		NewHex(6, 4), // Northeast of (6,5)
		NewHex(7, 4), // East of (6,4)
	}
	
	for i, h := range connectedHexes {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          buildingTypes[i],
		}
		m2.PlaceBuilding(h, building)
	}
	
	// Build a bridge to enhance connectivity
	// Bridge between (5,5) and (5,6) - just testing bridge functionality
	err := m2.BuildBridge(NewHex(5, 5), NewHex(5, 6), playerID)
	if err != nil {
		t.Fatalf("Failed to build bridge: %v", err)
	}
	
	// Should still form town (buildings are connected)
	townWithBridge := m2.DetectTown(connectedHexes[0])
	if townWithBridge == nil {
		t.Errorf("Should detect town with bridge present")
	} else if len(townWithBridge.Hexes) != 4 {
		t.Errorf("Town with bridge should have 4 hexes, got %d", len(townWithBridge.Hexes))
	}
}

func TestTownDetectionDisconnectedBuildings(t *testing.T) {
	m := NewTerraMysticaMap()
	playerID := "player1"
	faction := models.FactionNomads
	
	// Place 2 groups of buildings far apart
	group1 := []Hex{
		NewHex(2, 2),
		NewHex(3, 2),
	}
	
	group2 := []Hex{
		NewHex(8, 8),
		NewHex(9, 8),
	}
	
	// Place strongholds in both groups (3+3 = 6 power each group)
	for _, h := range append(group1, group2...) {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          models.BuildingStronghold,
		}
		m.PlaceBuilding(h, building)
	}
	
	// Each group alone should not form a town (only 2 buildings each)
	town1 := m.DetectTown(group1[0])
	if town1 != nil {
		t.Errorf("Disconnected group 1 should not form a town (only 2 buildings)")
	}
	
	town2 := m.DetectTown(group2[0])
	if town2 != nil {
		t.Errorf("Disconnected group 2 should not form a town (only 2 buildings)")
	}
}

func TestTownDetectionMultiplePlayers(t *testing.T) {
	m := NewTerraMysticaMap()
	
	// Player 1 places 4 buildings
	player1Hexes := []Hex{
		NewHex(5, 5),
		NewHex(6, 5),
		NewHex(7, 5),
		NewHex(8, 5),
	}
	
	for _, h := range player1Hexes {
		building := &models.Building{
			OwnerPlayerID: "player1",
			Faction:       models.FactionNomads,
			Type:          models.BuildingTemple,
		}
		m.PlaceBuilding(h, building)
	}
	
	// Player 2 places buildings adjacent to player 1
	player2Hexes := []Hex{
		NewHex(5, 6),
		NewHex(6, 6),
	}
	
	for _, h := range player2Hexes {
		building := &models.Building{
			OwnerPlayerID: "player2",
			Faction:       models.FactionGiants,
			Type:          models.BuildingStronghold,
		}
		m.PlaceBuilding(h, building)
	}
	
	// Player 1 should form a town (4 temples = 8 power)
	town1 := m.DetectTown(player1Hexes[0])
	if town1 == nil {
		t.Errorf("Player 1 should form a town")
	} else {
		// Town should only include player 1's buildings
		if len(town1.Hexes) != 4 {
			t.Errorf("Player 1 town should have 4 hexes, got %d", len(town1.Hexes))
		}
		for _, hex := range town1.Hexes {
			mapHex := m.GetHex(hex)
			if mapHex.Building.OwnerPlayerID != "player1" {
				t.Errorf("Town should only include player 1's buildings")
			}
		}
	}
	
	// Player 2 should not form a town (only 2 buildings)
	town2 := m.DetectTown(player2Hexes[0])
	if town2 != nil {
		t.Errorf("Player 2 should not form a town (only 2 buildings)")
	}
}

func TestDetectAllTowns(t *testing.T) {
	m := NewTerraMysticaMap()
	playerID := "player1"
	faction := models.FactionNomads
	
	// Create two separate towns for the same player
	// Town 1: hexes (2,2) to (5,2)
	town1Hexes := []Hex{
		NewHex(2, 2),
		NewHex(3, 2),
		NewHex(4, 2),
		NewHex(5, 2),
	}
	
	// Town 2: hexes (8,8) to (11,8)
	town2Hexes := []Hex{
		NewHex(8, 8),
		NewHex(9, 8),
		NewHex(10, 8),
		NewHex(11, 8),
	}
	
	// Place temples in both towns (2 power each, 4 buildings = 8 power)
	for _, h := range append(town1Hexes, town2Hexes...) {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          models.BuildingTemple,
		}
		m.PlaceBuilding(h, building)
	}
	
	// Detect all towns
	towns := m.DetectAllTowns(playerID)
	
	if len(towns) != 2 {
		t.Errorf("Should detect 2 towns, got %d", len(towns))
	}
	
	// Each town should have 4 hexes and 8 power
	for i, town := range towns {
		if len(town.Hexes) != 4 {
			t.Errorf("Town %d should have 4 hexes, got %d", i, len(town.Hexes))
		}
		if town.TotalPower != 8 {
			t.Errorf("Town %d should have 8 power, got %d", i, town.TotalPower)
		}
	}
}

func TestTownDetectionExactly7Power(t *testing.T) {
	m := NewTerraMysticaMap()
	playerID := "player1"
	faction := models.FactionNomads
	
	// Place buildings with exactly 7 power
	// 1 Sanctuary (3) + 2 Trading Houses (2+2) = 7 power, but only 3 buildings
	// Need to add 1 more dwelling to get 4 buildings
	// 1 Sanctuary (3) + 1 Trading House (2) + 2 Dwellings (1+1) = 7 power, 4 buildings
	buildings := []struct {
		hex   Hex
		bType models.BuildingType
	}{
		{NewHex(5, 5), models.BuildingSanctuary},
		{NewHex(6, 5), models.BuildingTradingHouse},
		{NewHex(7, 5), models.BuildingDwelling},
		{NewHex(8, 5), models.BuildingDwelling},
	}
	
	for _, b := range buildings {
		building := &models.Building{
			OwnerPlayerID: playerID,
			Faction:       faction,
			Type:          b.bType,
		}
		m.PlaceBuilding(b.hex, building)
	}
	
	// Should form a valid town with exactly 7 power
	town := m.DetectTown(buildings[0].hex)
	if town == nil {
		t.Fatalf("Should detect a town with exactly 7 power")
	}
	
	if town.TotalPower != 7 {
		t.Errorf("Town should have exactly 7 power, got %d", town.TotalPower)
	}
}
