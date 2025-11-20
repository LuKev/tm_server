package board

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestTerrainDistance(t *testing.T) {
	tests := []struct {
		name     string
		from     models.TerrainType
		to       models.TerrainType
		expected int
	}{
		{"Same terrain", models.TerrainPlains, models.TerrainPlains, 0},
		{"Adjacent forward", models.TerrainPlains, models.TerrainSwamp, 1},
		{"Adjacent backward", models.TerrainSwamp, models.TerrainPlains, 1},
		{"Opposite sides", models.TerrainPlains, models.TerrainMountain, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TerrainDistance(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("TerrainDistance(%v, %v) = %d, want %d", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestCalculateTerraformCost(t *testing.T) {
	tests := []struct {
		name         string
		from         models.TerrainType
		to           models.TerrainType
		diggingLevel int
		expected     int
	}{
		{"No terraform needed", models.TerrainPlains, models.TerrainPlains, 0, 0},
		{"1 spade, no digging", models.TerrainPlains, models.TerrainSwamp, 0, 3},
		{"1 spade, level 1 digging", models.TerrainPlains, models.TerrainSwamp, 1, 2},
		{"1 spade, level 2 digging", models.TerrainPlains, models.TerrainSwamp, 2, 1},
		{"3 spades, no digging", models.TerrainPlains, models.TerrainMountain, 0, 9},
		{"3 spades, level 1 digging", models.TerrainPlains, models.TerrainMountain, 1, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTerraformCost(tt.from, tt.to, tt.diggingLevel)
			if result != tt.expected {
				t.Errorf("CalculateTerraformCost(%v, %v, %d) = %d, want %d",
					tt.from, tt.to, tt.diggingLevel, result, tt.expected)
			}
		})
	}
}

func TestCanTerraform(t *testing.T) {
	m := NewTerraMysticaMap()

	// Test valid terraform
	h := NewHex(0, 0) // Plains
	if err := m.CanTerraform(h); err != nil {
		t.Errorf("expected valid terraform, got error: %v", err)
	}

	// Test river hex (invalid)
	hRiver := NewHex(1, 1) // River
	if err := m.CanTerraform(hRiver); err == nil {
		t.Errorf("expected error for river hex terraform")
	}

	// Test hex with building (invalid)
	hWithBuilding := NewHex(2, 0) // Forest
	m.PlaceBuilding(hWithBuilding, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionNomads,
		PowerValue: 1,
	})
	if err := m.CanTerraform(hWithBuilding); err == nil {
		t.Errorf("expected error for hex with building")
	}
}

func TestTerraform(t *testing.T) {
	m := NewTerraMysticaMap()
	h := NewHex(0, 0) // Plains

	// Terraform to Swamp
	if err := m.Terraform(h, models.TerrainSwamp); err != nil {
		t.Fatalf("terraform failed: %v", err)
	}

	// Verify terrain changed
	mapHex := m.GetHex(h)
	if mapHex.Terrain != models.TerrainSwamp {
		t.Errorf("expected terrain to be Swamp, got %v", mapHex.Terrain)
	}

	// Try to terraform to same terrain (should error)
	if err := m.Terraform(h, models.TerrainSwamp); err == nil {
		t.Errorf("expected error when terraforming to same terrain")
	}
}

func TestCanUpgradeBuilding(t *testing.T) {
	tests := []struct {
		name    string
		current models.BuildingType
		target  models.BuildingType
		wantErr bool
	}{
		{"Dwelling to TP", models.BuildingDwelling, models.BuildingTradingHouse, false},
		{"Dwelling to Temple", models.BuildingDwelling, models.BuildingTemple, true},
		{"TP to Temple", models.BuildingTradingHouse, models.BuildingTemple, false},
		{"TP to Stronghold", models.BuildingTradingHouse, models.BuildingStronghold, false},
		{"TP to Sanctuary", models.BuildingTradingHouse, models.BuildingSanctuary, false},
		{"TP to Dwelling", models.BuildingTradingHouse, models.BuildingDwelling, true},
		{"Temple upgrade", models.BuildingTemple, models.BuildingStronghold, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanUpgradeBuilding(tt.current, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanUpgradeBuilding(%v, %v) error = %v, wantErr %v",
					tt.current, tt.target, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBuildingPlacement(t *testing.T) {
	m := NewTerraMysticaMap()
	faction := models.FactionNomads
	homeTerrain := models.TerrainDesert

	// Find a desert hex for first dwelling
	var desertHex Hex
	for h, mh := range m.Hexes {
		if mh.Terrain == models.TerrainDesert {
			desertHex = h
			break
		}
	}

	// First dwelling should be valid on home terrain
	if err := m.ValidateBuildingPlacement(desertHex, faction, homeTerrain, true); err != nil {
		t.Errorf("first dwelling placement failed: %v", err)
	}

	// Place first dwelling
	m.PlaceBuilding(desertHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction,
		PowerValue: 1,
	})

	// Second dwelling must be adjacent
	// Find an adjacent desert hex
	var adjacentDesert Hex
	found := false
	for _, neighbor := range m.GetDirectNeighbors(desertHex) {
		if mh := m.GetHex(neighbor); mh != nil && mh.Terrain == models.TerrainDesert && mh.Building == nil {
			adjacentDesert = neighbor
			found = true
			break
		}
	}

	if found {
		if err := m.ValidateBuildingPlacement(adjacentDesert, faction, homeTerrain, false); err != nil {
			t.Errorf("adjacent dwelling placement failed: %v", err)
		}
	}

	// Non-adjacent desert should fail
	var nonAdjacentDesert Hex
	for h, mh := range m.Hexes {
		if mh.Terrain == models.TerrainDesert && mh.Building == nil && !m.IsDirectlyAdjacent(h, desertHex) {
			nonAdjacentDesert = h
			break
		}
	}

	if !nonAdjacentDesert.Equals(Hex{}) {
		if err := m.ValidateBuildingPlacement(nonAdjacentDesert, faction, homeTerrain, false); err == nil {
			t.Errorf("expected error for non-adjacent dwelling placement")
		}
	}
}
