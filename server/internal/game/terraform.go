package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// TerrainDistance returns the number of spades needed to transform from one terrain to another
// Terra Mystica terrain wheel: Plains -> Swamp -> Lake -> Forest -> Mountain -> Wasteland -> Desert -> Plains
func TerrainDistance(from, to models.TerrainType) int {
	if from == to {
		return 0
	}
	
	// Define terrain wheel order
	wheel := []models.TerrainType{
		models.TerrainPlains,
		models.TerrainSwamp,
		models.TerrainLake,
		models.TerrainForest,
		models.TerrainMountain,
		models.TerrainWasteland,
		models.TerrainDesert,
	}
	
	// Find positions
	fromIdx := -1
	toIdx := -1
	for i, t := range wheel {
		if t == from {
			fromIdx = i
		}
		if t == to {
			toIdx = i
		}
	}
	
	if fromIdx == -1 || toIdx == -1 {
		return -1 // Invalid terrain type
	}
	
	// Calculate shortest distance around the wheel
	forward := (toIdx - fromIdx + len(wheel)) % len(wheel)
	backward := (fromIdx - toIdx + len(wheel)) % len(wheel)
	
	if forward < backward {
		return forward
	}
	return backward
}

// CalculateTerraformCost calculates the worker cost to terraform a hex
// Base cost is 3 workers per spade, modified by faction digging level
func CalculateTerraformCost(from, to models.TerrainType, diggingLevel int) int {
	distance := TerrainDistance(from, to)
	if distance <= 0 {
		return 0
	}
	
	// Base cost: 3 workers per spade
	baseCost := 3
	
	// Digging level reduces cost (each level reduces by 1 worker per spade)
	costPerSpade := baseCost - diggingLevel
	if costPerSpade < 1 {
		costPerSpade = 1 // Minimum 1 worker per spade
	}
	
	return distance * costPerSpade
}

// CanTerraform checks if a hex can be terraformed
func (m *TerraMysticaMap) CanTerraform(h Hex) error {
	mapHex := m.GetHex(h)
	if mapHex == nil {
		return fmt.Errorf("hex %s is not on the map", h)
	}
	
	if mapHex.Terrain == models.TerrainRiver {
		return fmt.Errorf("cannot terraform river hexes")
	}
	
	if mapHex.Building != nil {
		return fmt.Errorf("hex %s already has a building", h)
	}
	
	return nil
}

// Terraform changes the terrain of a hex
func (m *TerraMysticaMap) Terraform(h Hex, newTerrain models.TerrainType) error {
	if err := m.CanTerraform(h); err != nil {
		return err
	}
	
	mapHex := m.GetHex(h)
	if mapHex.Terrain == newTerrain {
		return fmt.Errorf("hex is already %s terrain", newTerrain)
	}
	
	// Update terrain
	mapHex.Terrain = newTerrain
	
	// Remove from river hexes if it was a river
	if m.RiverHexes[h] {
		delete(m.RiverHexes, h)
	}
	
	return nil
}

// ValidateBuildingPlacement checks if a building can be placed at a hex
func (m *TerraMysticaMap) ValidateBuildingPlacement(h Hex, faction models.FactionType, homeTerrain models.TerrainType, isFirstDwelling bool) error {
	mapHex := m.GetHex(h)
	if mapHex == nil {
		return fmt.Errorf("hex %s is not on the map", h)
	}
	
	if mapHex.Building != nil {
		return fmt.Errorf("hex %s already has a building", h)
	}
	
	if mapHex.Terrain == models.TerrainRiver {
		return fmt.Errorf("cannot build on river hexes")
	}
	
	// Terrain must match faction's home terrain
	if mapHex.Terrain != homeTerrain {
		return fmt.Errorf("terrain must be %s for this faction", homeTerrain)
	}
	
	// First dwelling can be placed anywhere on home terrain
	if isFirstDwelling {
		return nil
	}
	
	// Subsequent dwellings must be adjacent (directly or indirectly via shipping) to existing buildings
	hasAdjacentBuilding := false
	for hex := range m.Hexes {
		if mh := m.GetHex(hex); mh != nil && mh.Building != nil && mh.Building.Faction == faction {
			// Check direct adjacency
			if m.IsDirectlyAdjacent(h, hex) {
				hasAdjacentBuilding = true
				break
			}
			// TODO: Check indirect adjacency via shipping (requires player shipping level)
		}
	}
	
	if !hasAdjacentBuilding {
		return fmt.Errorf("building must be adjacent to an existing building of your faction")
	}
	
	return nil
}

// CanUpgradeBuilding checks if a building can be upgraded
func CanUpgradeBuilding(current models.BuildingType, target models.BuildingType) error {
	// Valid upgrade paths:
	// Dwelling -> Trading House
	// Trading House -> Temple or Stronghold or Sanctuary
	
	switch current {
	case models.BuildingDwelling:
		if target != models.BuildingTradingHouse {
			return fmt.Errorf("dwelling can only upgrade to trading house")
		}
	case models.BuildingTradingHouse:
		if target != models.BuildingTemple && target != models.BuildingStronghold && target != models.BuildingSanctuary {
			return fmt.Errorf("trading house can upgrade to temple, stronghold, or sanctuary")
		}
	case models.BuildingTemple, models.BuildingStronghold, models.BuildingSanctuary:
		return fmt.Errorf("cannot upgrade %s further", current)
	default:
		return fmt.Errorf("unknown building type: %s", current)
	}
	
	return nil
}
