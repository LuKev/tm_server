package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestGetPowerValue(t *testing.T) {
	tests := []struct {
		building models.BuildingType
		expected int
	}{
		{models.BuildingDwelling, 1},
		{models.BuildingTradingHouse, 2},
		{models.BuildingTemple, 2},
		{models.BuildingSanctuary, 3},
		{models.BuildingStronghold, 3},
	}

	for _, tt := range tests {
		t.Run(string(tt.building), func(t *testing.T) {
			result := GetPowerValue(tt.building)
			if result != tt.expected {
				t.Errorf("GetPowerValue(%v) = %d, want %d", tt.building, result, tt.expected)
			}
		})
	}
}

func TestIsTown(t *testing.T) {
	m := NewTerraMysticaMap()
	faction := models.FactionNomads

	// Create a group of 4 dwellings (power value 1 each = 4 total)
	hexes := []Hex{
		NewHex(0, 0),
		NewHex(1, 0),
		NewHex(2, 0),
		NewHex(3, 0),
	}

	// Place dwellings
	for _, h := range hexes {
		m.PlaceBuilding(h, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction,
			PowerValue: 1,
		})
	}

	// 4 dwellings with total power 4 < 7, not a town
	if IsTown(hexes, m) {
		t.Errorf("expected not a town (power < 7)")
	}

	// Upgrade one to trading house (power 2), total = 5, still not a town
	m.GetHex(hexes[0]).Building.Type = models.BuildingTradingHouse
	m.GetHex(hexes[0]).Building.PowerValue = 2
	if IsTown(hexes, m) {
		t.Errorf("expected not a town (power = 5 < 7)")
	}

	// Upgrade another to trading house, total = 6, still not a town
	m.GetHex(hexes[1]).Building.Type = models.BuildingTradingHouse
	m.GetHex(hexes[1]).Building.PowerValue = 2
	if IsTown(hexes, m) {
		t.Errorf("expected not a town (power = 6 < 7)")
	}

	// Upgrade third to trading house, total = 7, now it's a town
	m.GetHex(hexes[2]).Building.Type = models.BuildingTradingHouse
	m.GetHex(hexes[2]).Building.PowerValue = 2
	if !IsTown(hexes, m) {
		t.Errorf("expected a town (4 buildings, power = 7)")
	}
}

func TestIsTown_MinimumBuildings(t *testing.T) {
	m := NewTerraMysticaMap()
	faction := models.FactionNomads

	// Only 3 buildings, even with high power
	hexes := []Hex{
		NewHex(0, 0),
		NewHex(1, 0),
		NewHex(2, 0),
	}

	for _, h := range hexes {
		m.PlaceBuilding(h, &models.Building{
			Type:       models.BuildingSanctuary,
			Faction:    faction,
			PowerValue: 3,
		})
	}

	// 3 buildings with power 9 >= 7, but < 4 buildings, not a town
	if IsTown(hexes, m) {
		t.Errorf("expected not a town (only 3 buildings)")
	}
}

func TestCalculateAdjacencyBonus(t *testing.T) {
	m := NewTerraMysticaMap()
	faction1 := models.FactionNomads
	faction2 := models.FactionFakirs

	// Place building at (0,0)
	h := NewHex(0, 0)

	// Place opponent buildings adjacent
	neighbors := m.GetDirectNeighbors(h)
	if len(neighbors) < 2 {
		t.Skip("not enough neighbors for test")
	}

	// Place 2 opponent buildings
	m.PlaceBuilding(neighbors[0], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2,
		PowerValue: 1,
	})
	m.PlaceBuilding(neighbors[1], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2,
		PowerValue: 1,
	})

	// Calculate adjacency bonus
	bonus := m.CalculateAdjacencyBonus(h, faction1)
	if bonus != 2 {
		t.Errorf("expected adjacency bonus of 2, got %d", bonus)
	}

	// Place own faction building
	if len(neighbors) > 2 {
		m.PlaceBuilding(neighbors[2], &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction1,
			PowerValue: 1,
		})

		// Bonus should still be 2 (own buildings don't count)
		bonus = m.CalculateAdjacencyBonus(h, faction1)
		if bonus != 2 {
			t.Errorf("expected adjacency bonus of 2 (own buildings don't count), got %d", bonus)
		}
	}
}

func TestGetPowerLeechTargets(t *testing.T) {
	m := NewTerraMysticaMap()
	faction1 := models.FactionNomads
	faction2 := models.FactionFakirs
	faction3 := models.FactionGiants

	h := NewHex(0, 0)
	neighbors := m.GetDirectNeighbors(h)
	if len(neighbors) < 3 {
		t.Skip("not enough neighbors for test")
	}

	// Place buildings from 2 different opponent factions
	m.PlaceBuilding(neighbors[0], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2,
		PowerValue: 1,
	})
	m.PlaceBuilding(neighbors[1], &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction3,
		PowerValue: 2,
	})

	// Place a temple (power value 2) at h
	powerValue := 2
	targets := m.GetPowerLeechTargets(h, faction1, powerValue)

	// Both opponents should be able to leech 2 power
	if len(targets) != 2 {
		t.Errorf("expected 2 leech targets, got %d", len(targets))
	}

	if targets[faction2] != 2 {
		t.Errorf("expected faction2 to leech 2 power, got %d", targets[faction2])
	}

	if targets[faction3] != 2 {
		t.Errorf("expected faction3 to leech 2 power, got %d", targets[faction3])
	}

	// Own faction should not be in targets
	if _, ok := targets[faction1]; ok {
		t.Errorf("own faction should not be able to leech power")
	}
}

func TestDetectTowns(t *testing.T) {
	m := NewTerraMysticaMap()
	faction := models.FactionNomads

	// Create a connected group of 4 buildings with total power >= 7
	hexes := []Hex{
		NewHex(0, 0), // Plains
		NewHex(1, 0), // Mountain (adjacent)
		NewHex(2, 0), // Forest (adjacent)
		NewHex(3, 0), // Lake (adjacent)
	}

	// Place 2 dwellings and 2 trading houses (total power = 1+1+2+2 = 6, not enough)
	m.PlaceBuilding(hexes[0], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction,
		PowerValue: 1,
	})
	m.PlaceBuilding(hexes[1], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction,
		PowerValue: 1,
	})
	m.PlaceBuilding(hexes[2], &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction,
		PowerValue: 2,
	})
	m.PlaceBuilding(hexes[3], &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction,
		PowerValue: 2,
	})

	// Should not detect a town yet (power = 6 < 7)
	towns := m.DetectTowns(faction)
	if len(towns) != 0 {
		t.Errorf("expected 0 towns (power < 7), got %d", len(towns))
	}

	// Upgrade one dwelling to trading house (power = 7)
	m.GetHex(hexes[0]).Building.Type = models.BuildingTradingHouse
	m.GetHex(hexes[0]).Building.PowerValue = 2

	// Should detect 1 town now
	towns = m.DetectTowns(faction)
	if len(towns) != 1 {
		t.Errorf("expected 1 town, got %d", len(towns))
	}

	if len(towns) > 0 {
		if len(towns[0].Hexes) != 4 {
			t.Errorf("expected town with 4 hexes, got %d", len(towns[0].Hexes))
		}
		if towns[0].TotalPower != 7 {
			t.Errorf("expected town power of 7, got %d", towns[0].TotalPower)
		}
		if towns[0].Faction != faction {
			t.Errorf("expected town faction %v, got %v", faction, towns[0].Faction)
		}
	}
}
