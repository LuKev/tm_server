package notation

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

// ConvertRiverCoordToAxial converts river notation (e.g., "R~D5") to axial coordinates
// The format is R~[Coord] where Coord is a land hex adjacent to the river hex
// Returns the river hex closest to the given land hex
func ConvertRiverCoordToAxial(riverCoord string) (board.Hex, error) {
	// Parse "R~D5" format
	if !strings.HasPrefix(riverCoord, "R~") {
		return board.Hex{}, fmt.Errorf("invalid river coordinate format: %s (expected R~[Coord])", riverCoord)
	}

	landCoord := riverCoord[2:] // Remove "R~" prefix
	landHex, err := ConvertLogCoordToAxial(landCoord)
	if err != nil {
		return board.Hex{}, fmt.Errorf("invalid land coordinate in river ref: %w", err)
	}

	// Get terrain layout
	layout := board.BaseGameTerrainLayout()

	// Find adjacent river hexes
	neighbors := landHex.Neighbors()
	for _, neighbor := range neighbors {
		terrain, exists := layout[neighbor]
		if exists && terrain == models.TerrainRiver {
			return neighbor, nil
		}
	}

	return board.Hex{}, fmt.Errorf("no river hex found adjacent to %s", landCoord)
}
