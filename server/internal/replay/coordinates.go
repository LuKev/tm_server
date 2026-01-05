package replay

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// ConvertLogCoordToAxial converts log notation (e.g., "D5") to axial (q, r)
// Letter = row (A=0, B=1, ..., I=8)
// Number = nth non-river hex in that row (1-indexed)
func ConvertLogCoordToAxial(coord string) (board.Hex, error) {
	if len(coord) < 2 {
		return board.Hex{}, fmt.Errorf("invalid coordinate: %s", coord)
	}

	// Convert to uppercase for case-insensitive parsing
	coord = strings.ToUpper(coord)

	// Parse row letter
	row := int(coord[0] - 'A')
	if row < 0 || row > 8 {
		return board.Hex{}, fmt.Errorf("invalid row: %c (must be A-I)", coord[0])
	}

	// Parse hex number (1-indexed)
	var hexNum int
	_, err := fmt.Sscanf(coord[1:], "%d", &hexNum)
	if err != nil {
		return board.Hex{}, fmt.Errorf("invalid hex number in %s: %w", coord, err)
	}
	if hexNum < 1 {
		return board.Hex{}, fmt.Errorf("hex number must be >= 1, got %d", hexNum)
	}

	// Get terrain layout
	layout := board.BaseGameTerrainLayout()

	// Find the starting q for this row
	// Pattern from terrain_layout.go:
	// Row 0: q starts at 0
	// Row 1: q starts at 0
	// Row 2: q starts at -1
	// Row 3: q starts at -1
	// Row 4: q starts at -2
	// Row 5: q starts at -2
	// Row 6: q starts at -3
	// Row 7: q starts at -3
	// Row 8: q starts at -4
	startQ := -row / 2

	// Count non-river hexes until we reach the nth one
	count := 0
	r := row
	for q := startQ; ; q++ {
		h := board.NewHex(q, r)
		terrain, exists := layout[h]
		if !exists {
			break // End of row
		}
		if terrain != models.TerrainRiver {
			count++
			if count == hexNum {
				return h, nil
			}
		}
	}

	return board.Hex{}, fmt.Errorf("hex %s not found (row %d, hex %d): only %d non-river hexes in row", coord, row, hexNum, count)
}

// ValidateCoordinateConversion tests the coordinate conversion logic
func ValidateCoordinateConversion() error {
	layout := board.BaseGameTerrainLayout()

	// Test D5 - should be Lake terrain (blue)
	hex, err := ConvertLogCoordToAxial("D5")
	if err != nil {
		return fmt.Errorf("failed to convert D5: %w", err)
	}

	terrain := layout[hex]
	if terrain != models.TerrainLake {
		return fmt.Errorf("D5 should be Lake but got %s (coords: q=%d, r=%d)", terrain, hex.Q, hex.R)
	}

	return nil
}

// ValidateTerrainLayout verifies our terrain layout against known setup positions from game logs
func ValidateTerrainLayout() error {
	layout := board.BaseGameTerrainLayout()

	// From game log setup - these are home terrain builds (no transformation)
	expected := map[string]models.TerrainType{
		// Engineers (gray/mountain)
		"E7": models.TerrainMountain,
		"F1": models.TerrainMountain,
		// Darklings (black/swamp)
		"G5": models.TerrainSwamp,
		"E5": models.TerrainSwamp,
		// Cultists (brown/plains)
		"E6": models.TerrainPlains,
		"F5": models.TerrainPlains,
		// Witches (green/forest)
		"F4": models.TerrainForest,
		"E9": models.TerrainForest,
	}

	for coord, expectedTerrain := range expected {
		hex, err := ConvertLogCoordToAxial(coord)
		if err != nil {
			return fmt.Errorf("failed to convert %s: %w", coord, err)
		}
		actualTerrain := layout[hex]
		if actualTerrain != expectedTerrain {
			return fmt.Errorf("%s (q=%d, r=%d): expected %s, got %s",
				coord, hex.Q, hex.R, expectedTerrain, actualTerrain)
		}
	}

	return nil
}
