package notation

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// ConvertLogCoordToAxialForMap converts log notation (e.g., "D5") to axial (q, r)
// for the supplied map. Letter = row (A=0, B=1, ..., I=8), number = nth non-river
// hex in that row (1-indexed).
func ConvertLogCoordToAxialForMap(mapID board.MapID, coord string) (board.Hex, error) {
	if len(coord) < 2 {
		return board.Hex{}, fmt.Errorf("invalid coordinate: %s", coord)
	}

	coord = strings.ToUpper(coord)
	normalizedMapID := board.NormalizeMapID(string(mapID))
	if hex, ok := board.HexForDisplayCoordinate(normalizedMapID, coord); ok {
		return hex, nil
	}

	return board.Hex{}, fmt.Errorf("hex %s not found on map %s", coord, normalizedMapID)
}

// ConvertLogCoordToAxial converts log notation using the base-map coordinate index.
func ConvertLogCoordToAxial(coord string) (board.Hex, error) {
	return ConvertLogCoordToAxialForMap(board.MapBase, coord)
}

// ConvertRiverCoordToAxialForMap converts BGA river notation (e.g., "R~C3") to axial
// coordinates for the supplied map.
func ConvertRiverCoordToAxialForMap(mapID board.MapID, riverCoord string) (board.Hex, error) {
	if !strings.HasPrefix(riverCoord, "R~") {
		return board.Hex{}, fmt.Errorf("invalid river coordinate format: %s (expected R~[Coord])", riverCoord)
	}

	coord := strings.ToUpper(strings.TrimSpace(riverCoord[2:]))
	if len(coord) < 2 {
		return board.Hex{}, fmt.Errorf("invalid river coordinate: %s", riverCoord)
	}

	normalizedMapID := board.NormalizeMapID(string(mapID))
	if landHex, ok := board.HexForDisplayCoordinate(normalizedMapID, coord); ok {
		layout, err := board.LayoutForMap(normalizedMapID)
		if err != nil {
			return board.Hex{}, err
		}

		for q := landHex.Q + 1; ; q++ {
			h := board.NewHex(q, landHex.R)
			terrain, exists := layout[h]
			if !exists {
				break
			}
			if terrain == models.TerrainRiver {
				return h, nil
			}
			break
		}

		riverNeighbors := make([]board.Hex, 0, 2)
		for _, neighbor := range landHex.Neighbors() {
			terrain, exists := layout[neighbor]
			if !exists || terrain != models.TerrainRiver {
				continue
			}
			riverNeighbors = append(riverNeighbors, neighbor)
		}
		if len(riverNeighbors) == 1 {
			return riverNeighbors[0], nil
		}
	}

	return convertRiverCoordToAxialByRowCountForMap(normalizedMapID, riverCoord)
}

// ConvertRiverCoordToAxial converts BGA river notation using the base-map coordinate index.
func ConvertRiverCoordToAxial(riverCoord string) (board.Hex, error) {
	return ConvertRiverCoordToAxialForMap(board.MapBase, riverCoord)
}

func convertRiverCoordToAxialByRowCountForMap(mapID board.MapID, riverCoord string) (board.Hex, error) {
	coord := strings.ToUpper(strings.TrimSpace(riverCoord))
	if strings.HasPrefix(coord, "R~") {
		coord = strings.TrimSpace(coord[2:])
	}
	if len(coord) < 2 {
		return board.Hex{}, fmt.Errorf("invalid river coordinate: %s", riverCoord)
	}

	row := int(coord[0] - 'A')
	if row < 0 || row > 8 {
		return board.Hex{}, fmt.Errorf("invalid river row: %c (must be A-I)", coord[0])
	}

	var riverNum int
	if _, err := fmt.Sscanf(coord[1:], "%d", &riverNum); err != nil {
		return board.Hex{}, fmt.Errorf("invalid river number in %s: %w", riverCoord, err)
	}
	if riverNum < 1 {
		return board.Hex{}, fmt.Errorf("river number must be >= 1, got %d", riverNum)
	}

	layout, err := board.LayoutForMap(board.NormalizeMapID(string(mapID)))
	if err != nil {
		return board.Hex{}, err
	}
	startQ := 0
	for candidate := range layout {
		if candidate.R != row {
			continue
		}
		if candidate.Q < startQ {
			startQ = candidate.Q
		}
	}
	count := 0
	for q := startQ; ; q++ {
		h := board.NewHex(q, row)
		terrain, exists := layout[h]
		if !exists {
			break
		}
		if terrain != models.TerrainRiver {
			continue
		}
		count++
		if count == riverNum {
			return h, nil
		}
	}

	return board.Hex{}, fmt.Errorf("river %s not found: only %d river hexes in row", riverCoord, count)
}
