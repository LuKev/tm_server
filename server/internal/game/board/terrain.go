package board

import "github.com/lukev/tm_server/internal/models"

// Based on the official base game map
// Rivers are represented as TerrainRiver hexes
func BaseGameTerrainLayout() map[Hex]models.TerrainType {
	layout, err := LayoutForMap(MapBase)
	if err != nil {
		panic(err)
	}
	return layout
}

// BaseGameRiverHexes returns the river hexes for the base game
func BaseGameRiverHexes() map[Hex]bool {
	rivers := make(map[Hex]bool)

	// Note: River hex coordinates from the official game board would be added here
	// Rivers separate land hexes and can be crossed via shipping
	// Placeholder implementation - frontend can specify river positions based on board layout

	return rivers
}

// TerrainDistance returns the number of spades needed to transform from one terrain to another
// Uses the standard Terra Mystica terrain cycle
func TerrainDistance(from, to models.TerrainType) int {
	if from == to {
		return 0
	}

	// Terrain cycle: Plains -> Swamp -> Lake -> Forest -> Mountain -> Wasteland -> Desert -> Plains
	// Map terrain types to cycle positions
	cycle := map[models.TerrainType]int{
		models.TerrainPlains:    0,
		models.TerrainSwamp:     1,
		models.TerrainLake:      2,
		models.TerrainForest:    3,
		models.TerrainMountain:  4,
		models.TerrainWasteland: 5,
		models.TerrainDesert:    6,
	}

	pos1, ok1 := cycle[from]
	pos2, ok2 := cycle[to]

	if !ok1 || !ok2 {
		return 0 // Invalid terrain (e.g. River)
	}

	// Calculate distance in both directions and take the shorter one
	diff := abs(pos1 - pos2)
	if diff > 3 {
		return 7 - diff
	}
	return diff
}

// CalculateIntermediateTerrain calculates the terrain that is 'steps' spades towards the target terrain
// For example: if from=Desert, to=Swamp, steps=1, returns Plains (1 step towards Swamp)
func CalculateIntermediateTerrain(from, to models.TerrainType, steps int) models.TerrainType {
	if from == to || steps == 0 {
		return from
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
		return from // Invalid terrain type
	}

	// Calculate shortest direction
	forward := (toIdx - fromIdx + len(wheel)) % len(wheel)
	backward := (fromIdx - toIdx + len(wheel)) % len(wheel)

	// Move 'steps' in the shortest direction
	var newIdx int
	if forward < backward {
		// Move forward
		newIdx = (fromIdx + steps) % len(wheel)
	} else {
		// Move backward
		newIdx = (fromIdx - steps + len(wheel)) % len(wheel)
	}

	return wheel[newIdx]
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
