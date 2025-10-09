package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// TerraMysticaMap represents the game board
// Terra Mystica uses a pointy-top hex grid with 9 rows alternating 13/12 hexagons
type TerraMysticaMap struct {
	Hexes     map[Hex]*MapHex
	Bridges   map[BridgeKey]bool // Tracks built bridges between hexes
	RiverHexes map[Hex]bool      // Tracks which hexes are rivers
}

// MapHex represents a single hex on the map
type MapHex struct {
	Coord    Hex
	Terrain  models.TerrainType
	Building *models.Building // nil if no building
}

// BridgeKey represents a bridge between two hexes (order-independent)
type BridgeKey struct {
	H1, H2 Hex
}

// NewBridgeKey creates a normalized bridge key (smaller hex first)
func NewBridgeKey(h1, h2 Hex) BridgeKey {
	if h1.Q < h2.Q || (h1.Q == h2.Q && h1.R < h2.R) {
		return BridgeKey{H1: h1, H2: h2}
	}
	return BridgeKey{H1: h2, H2: h1}
}

// NewTerraMysticaMap creates a new game map
func NewTerraMysticaMap() *TerraMysticaMap {
	m := &TerraMysticaMap{
		Hexes:      make(map[Hex]*MapHex),
		Bridges:    make(map[BridgeKey]bool),
		RiverHexes: make(map[Hex]bool),
	}
	m.initializeBaseMap()
	return m
}

// initializeBaseMap sets up the standard Terra Mystica base game map
// 9 rows alternating 13/12 hexagons, pointy-top orientation
func (m *TerraMysticaMap) initializeBaseMap() {
	// Terra Mystica map layout:
	// Row 0: 13 hexes (q: 0-12)
	// Row 1: 12 hexes (q: 0-11, offset by 0.5)
	// Row 2: 13 hexes (q: 0-12)
	// ... alternating pattern for 9 rows total

	// For now, create a placeholder map with all plains terrain
	// TODO: Replace with actual Terra Mystica terrain layout from rulebook
	for row := 0; row < 9; row++ {
		hexCount := 13
		if row%2 == 1 {
			hexCount = 12
		}

		for col := 0; col < hexCount; col++ {
			// In axial coordinates for pointy-top with offset rows:
			// Even rows (0,2,4,6,8): q = col, r = row
			// Odd rows (1,3,5,7): q = col, r = row (but visually offset)
			q := col
			r := row

			hex := NewHex(q, r)
			m.Hexes[hex] = &MapHex{
				Coord:    hex,
				Terrain:  models.TerrainPlains, // Placeholder
				Building: nil,
			}
		}
	}
}

// GetHex returns the MapHex at the given coordinate, or nil if out of bounds
func (m *TerraMysticaMap) GetHex(h Hex) *MapHex {
	return m.Hexes[h]
}

// IsValidHex checks if a hex coordinate is on the map
func (m *TerraMysticaMap) IsValidHex(h Hex) bool {
	_, exists := m.Hexes[h]
	return exists
}

// IsRiver checks if a hex is a river space
func (m *TerraMysticaMap) IsRiver(h Hex) bool {
	return m.RiverHexes[h]
}

// HasBridge checks if there is a bridge between two hexes
func (m *TerraMysticaMap) HasBridge(h1, h2 Hex) bool {
	return m.Bridges[NewBridgeKey(h1, h2)]
}

// BuildBridge creates a bridge between two hexes
func (m *TerraMysticaMap) BuildBridge(h1, h2 Hex) error {
	if !m.IsValidHex(h1) || !m.IsValidHex(h2) {
		return fmt.Errorf("cannot build bridge: invalid hex coordinates")
	}
	if !h1.IsDirectlyAdjacent(h2) {
		return fmt.Errorf("cannot build bridge: hexes are not adjacent")
	}
	m.Bridges[NewBridgeKey(h1, h2)] = true
	return nil
}

// IsDirectlyAdjacent checks if two hexes are directly adjacent according to Terra Mystica rules:
// 1. They share a hex edge (distance = 1), OR
// 2. They are separated by a river but connected via a bridge
func (m *TerraMysticaMap) IsDirectlyAdjacent(h1, h2 Hex) bool {
	// Check if they share an edge
	if h1.IsDirectlyAdjacent(h2) {
		// If there's a river between them, they need a bridge
		// For simplicity, we'll check if either hex is a river
		if m.IsRiver(h1) || m.IsRiver(h2) {
			return m.HasBridge(h1, h2)
		}
		return true
	}
	return false
}

// IsIndirectlyAdjacent checks if two hexes are indirectly adjacent according to Terra Mystica rules:
// - Separated by one or more river spaces
// - Reachable via shipping value
// - OR reachable via special abilities (tunneling, carpet flight)
func (m *TerraMysticaMap) IsIndirectlyAdjacent(h1, h2 Hex, shippingValue int) bool {
	// If directly adjacent, not indirectly adjacent
	if m.IsDirectlyAdjacent(h1, h2) {
		return false
	}

	// Check if reachable via shipping
	// Count river spaces between the two hexes
	distance := h1.Distance(h2)
	if distance <= shippingValue {
		// TODO: Verify path only crosses rivers, not land
		return true
	}

	return false
}

// GetDirectNeighbors returns all directly adjacent hexes (including bridges)
func (m *TerraMysticaMap) GetDirectNeighbors(h Hex) []Hex {
	neighbors := []Hex{}
	for _, neighbor := range h.Neighbors() {
		if m.IsValidHex(neighbor) && m.IsDirectlyAdjacent(h, neighbor) {
			neighbors = append(neighbors, neighbor)
		}
	}
	return neighbors
}

// GetIndirectNeighbors returns all indirectly adjacent hexes within shipping range
func (m *TerraMysticaMap) GetIndirectNeighbors(h Hex, shippingValue int) []Hex {
	neighbors := []Hex{}
	// Check all hexes within range
	for candidate := range m.Hexes {
		if m.IsIndirectlyAdjacent(h, candidate, shippingValue) {
			neighbors = append(neighbors, candidate)
		}
	}
	return neighbors
}

// CanPlaceBuilding checks if a building can be placed at the given hex
func (m *TerraMysticaMap) CanPlaceBuilding(h Hex, faction models.FactionType) error {
	mapHex := m.GetHex(h)
	if mapHex == nil {
		return fmt.Errorf("hex %s is not on the map", h)
	}

	if mapHex.Building != nil {
		return fmt.Errorf("hex %s already has a building", h)
	}

	// TODO: Check if terrain matches faction's home terrain or has been terraformed
	// TODO: Check adjacency requirements for first dwelling

	return nil
}

// PlaceBuilding places a building on the map
func (m *TerraMysticaMap) PlaceBuilding(h Hex, building *models.Building) error {
	mapHex := m.GetHex(h)
	if mapHex == nil {
		return fmt.Errorf("hex %s is not on the map", h)
	}

	if mapHex.Building != nil {
		return fmt.Errorf("hex %s already has a building", h)
	}

	mapHex.Building = building
	return nil
}

// RemoveBuilding removes a building from the map
func (m *TerraMysticaMap) RemoveBuilding(h Hex) error {
	mapHex := m.GetHex(h)
	if mapHex == nil {
		return fmt.Errorf("hex %s is not on the map", h)
	}

	if mapHex.Building == nil {
		return fmt.Errorf("hex %s has no building to remove", h)
	}

	mapHex.Building = nil
	return nil
}

// TransformTerrain changes the terrain type of a hex
func (m *TerraMysticaMap) TransformTerrain(h Hex, newTerrain models.TerrainType) error {
	mapHex := m.GetHex(h)
	if mapHex == nil {
		return fmt.Errorf("hex %s is not on the map", h)
	}

	if mapHex.Building != nil {
		return fmt.Errorf("cannot transform terrain: hex %s has a building", h)
	}

	mapHex.Terrain = newTerrain
	return nil
}

// GetBuildingsInRange returns all buildings within a given range of a hex
func (m *TerraMysticaMap) GetBuildingsInRange(h Hex, distance int) []*models.Building {
	buildings := []*models.Building{}
	for candidate := range m.Hexes {
		if h.IsWithinRange(candidate, distance) {
			if mapHex := m.GetHex(candidate); mapHex != nil && mapHex.Building != nil {
				buildings = append(buildings, mapHex.Building)
			}
		}
	}
	return buildings
}

// FindConnectedBuildings returns all buildings directly connected to the given hex
// Used for town formation detection
func (m *TerraMysticaMap) FindConnectedBuildings(h Hex, faction models.FactionType) []Hex {
	visited := make(map[Hex]bool)
	connected := []Hex{}

	var dfs func(Hex)
	dfs = func(current Hex) {
		if visited[current] {
			return
		}
		visited[current] = true

		mapHex := m.GetHex(current)
		if mapHex == nil || mapHex.Building == nil {
			return
		}

		// Only include buildings of the same faction
		if mapHex.Building.Faction != faction {
			return
		}

		connected = append(connected, current)

		// Explore direct neighbors
		for _, neighbor := range m.GetDirectNeighbors(current) {
			dfs(neighbor)
		}
	}

	dfs(h)
	return connected
}
