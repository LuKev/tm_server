package game

import "github.com/lukev/tm_server/internal/models"

// Based on the official base game map
// Rivers are represented as TerrainRiver hexes
func BaseGameTerrainLayout() map[Hex]models.TerrainType {
	layout := make(map[Hex]models.TerrainType)
	
	// Row 0 (13 hexes) - Top row
	layout[NewHex(0, 0)] = models.TerrainPlains
	layout[NewHex(1, 0)] = models.TerrainMountain
	layout[NewHex(2, 0)] = models.TerrainForest
	layout[NewHex(3, 0)] = models.TerrainLake
	layout[NewHex(4, 0)] = models.TerrainDesert
	layout[NewHex(5, 0)] = models.TerrainWasteland
	layout[NewHex(6, 0)] = models.TerrainPlains
	layout[NewHex(7, 0)] = models.TerrainSwamp
	layout[NewHex(8, 0)] = models.TerrainWasteland
	layout[NewHex(9, 0)] = models.TerrainForest
	layout[NewHex(10, 0)] = models.TerrainLake
	layout[NewHex(11, 0)] = models.TerrainWasteland
	layout[NewHex(12, 0)] = models.TerrainSwamp
	
	// Row 1 (12 hexes)
	layout[NewHex(0, 1)] = models.TerrainDesert
	layout[NewHex(1, 1)] = models.TerrainRiver
	layout[NewHex(2, 1)] = models.TerrainRiver
	layout[NewHex(3, 1)] = models.TerrainPlains
	layout[NewHex(4, 1)] = models.TerrainSwamp
	layout[NewHex(5, 1)] = models.TerrainRiver
	layout[NewHex(6, 1)] = models.TerrainRiver
	layout[NewHex(7, 1)] = models.TerrainDesert
	layout[NewHex(8, 1)] = models.TerrainSwamp
	layout[NewHex(9, 1)] = models.TerrainRiver
	layout[NewHex(10, 1)] = models.TerrainRiver
	layout[NewHex(11, 1)] = models.TerrainDesert
	
	// Row 2 (13 hexes)
	layout[NewHex(-1, 2)] = models.TerrainRiver
	layout[NewHex(0, 2)] = models.TerrainRiver
	layout[NewHex(1, 2)] = models.TerrainSwamp
	layout[NewHex(2, 2)] = models.TerrainRiver
	layout[NewHex(3, 2)] = models.TerrainMountain
	layout[NewHex(4, 2)] = models.TerrainRiver
	layout[NewHex(5, 2)] = models.TerrainForest
	layout[NewHex(6, 2)] = models.TerrainRiver
	layout[NewHex(7, 2)] = models.TerrainForest
	layout[NewHex(8, 2)] = models.TerrainRiver
	layout[NewHex(9, 2)] = models.TerrainMountain
	layout[NewHex(10, 2)] = models.TerrainRiver
	layout[NewHex(11, 2)] = models.TerrainRiver
	
	// Row 3 (12 hexes)
	layout[NewHex(-1, 3)] = models.TerrainForest
	layout[NewHex(0, 3)] = models.TerrainLake
	layout[NewHex(1, 3)] = models.TerrainDesert
	layout[NewHex(2, 3)] = models.TerrainRiver
	layout[NewHex(3, 3)] = models.TerrainRiver
	layout[NewHex(4, 3)] = models.TerrainWasteland
	layout[NewHex(5, 3)] = models.TerrainLake
	layout[NewHex(6, 3)] = models.TerrainRiver
	layout[NewHex(7, 3)] = models.TerrainWasteland
	layout[NewHex(8, 3)] = models.TerrainRiver
	layout[NewHex(9, 3)] = models.TerrainWasteland
	layout[NewHex(10, 3)] = models.TerrainPlains
	
	// Row 4 (13 hexes)
	layout[NewHex(-2, 4)] = models.TerrainSwamp
	layout[NewHex(-1, 4)] = models.TerrainPlains
	layout[NewHex(0, 4)] = models.TerrainWasteland
	layout[NewHex(1, 4)] = models.TerrainLake
	layout[NewHex(2, 4)] = models.TerrainSwamp
	layout[NewHex(3, 4)] = models.TerrainPlains      // E6 - cultists initial
	layout[NewHex(4, 4)] = models.TerrainMountain    // E7 - engineers initial
	layout[NewHex(5, 4)] = models.TerrainDesert
	layout[NewHex(6, 4)] = models.TerrainRiver
	layout[NewHex(7, 4)] = models.TerrainRiver
	layout[NewHex(8, 4)] = models.TerrainForest
	layout[NewHex(9, 4)] = models.TerrainSwamp
	layout[NewHex(10, 4)] = models.TerrainLake
	
	// Row 5 (12 hexes)
	layout[NewHex(-2, 5)] = models.TerrainMountain     // F1 - engineers initial
	layout[NewHex(-1, 5)] = models.TerrainForest
	layout[NewHex(0, 5)] = models.TerrainRiver
	layout[NewHex(1, 5)] = models.TerrainRiver
	layout[NewHex(2, 5)] = models.TerrainDesert
	layout[NewHex(3, 5)] = models.TerrainForest      // F4 - witches initial
	layout[NewHex(4, 5)] = models.TerrainRiver
	layout[NewHex(5, 5)] = models.TerrainRiver
	layout[NewHex(6, 5)] = models.TerrainRiver
	layout[NewHex(7, 5)] = models.TerrainPlains      // F5 - cultists initial
	layout[NewHex(8, 5)] = models.TerrainMountain
	layout[NewHex(9, 5)] = models.TerrainPlains
	
	// Row 6 (13 hexes)
	layout[NewHex(-3, 6)] = models.TerrainRiver
	layout[NewHex(-2, 6)] = models.TerrainRiver
	layout[NewHex(-1, 6)] = models.TerrainRiver
	layout[NewHex(0, 6)] = models.TerrainMountain
	layout[NewHex(1, 6)] = models.TerrainRiver
	layout[NewHex(2, 6)] = models.TerrainWasteland
	layout[NewHex(3, 6)] = models.TerrainRiver
	layout[NewHex(4, 6)] = models.TerrainForest
	layout[NewHex(5, 6)] = models.TerrainRiver
	layout[NewHex(6, 6)] = models.TerrainDesert
	layout[NewHex(7, 6)] = models.TerrainSwamp
	layout[NewHex(8, 6)] = models.TerrainLake
	layout[NewHex(9, 6)] = models.TerrainDesert
	
	// Row 7 (12 hexes)
	// Start (-3, 7)
	layout[NewHex(-3, 7)] = models.TerrainDesert
	layout[NewHex(-2, 7)] = models.TerrainLake
	layout[NewHex(-1, 7)] = models.TerrainPlains
	layout[NewHex(0, 7)] = models.TerrainRiver
	layout[NewHex(1, 7)] = models.TerrainRiver
	layout[NewHex(2, 7)] = models.TerrainRiver
	layout[NewHex(3, 7)] = models.TerrainLake
	layout[NewHex(4, 7)] = models.TerrainSwamp
	layout[NewHex(5, 7)] = models.TerrainRiver
	layout[NewHex(6, 7)] = models.TerrainMountain
	layout[NewHex(7, 7)] = models.TerrainPlains
	layout[NewHex(8, 7)] = models.TerrainMountain
	
	// Row 8 (13 hexes)
	layout[NewHex(-4, 8)] = models.TerrainWasteland
	layout[NewHex(-3, 8)] = models.TerrainSwamp
	layout[NewHex(-2, 8)] = models.TerrainMountain
	layout[NewHex(-1, 8)] = models.TerrainLake
	layout[NewHex(0, 8)] = models.TerrainWasteland
	layout[NewHex(1, 8)] = models.TerrainForest
	layout[NewHex(2, 8)] = models.TerrainDesert
	layout[NewHex(3, 8)] = models.TerrainPlains
	layout[NewHex(4, 8)] = models.TerrainMountain
	layout[NewHex(5, 8)] = models.TerrainRiver
	layout[NewHex(6, 8)] = models.TerrainLake
	layout[NewHex(7, 8)] = models.TerrainForest
	layout[NewHex(8, 8)] = models.TerrainWasteland

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
