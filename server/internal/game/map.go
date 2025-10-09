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
    // Load predefined layout
    layout := BaseGameTerrainLayout()
    m.RiverHexes = make(map[Hex]bool)
    for h, t := range layout {
        m.Hexes[h] = &MapHex{Coord: h, Terrain: t}
        if t == models.TerrainRiver {
            m.RiverHexes[h] = true
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

// BuildBridge creates a bridge between two land hexes.
// A valid bridge must:
// - Connect two non-river hexes
// - Span across the edge of a river hex: the vector (h2 - h1) must be one of the 6 allowed
//   distance-2 offsets: (1,-2), (2,-1), (2,0), (0,2), (-2,2), (-2,0) up to rotation,
//   and the two intermediate hexes along that edge must both be river hexes.
func (m *TerraMysticaMap) BuildBridge(h1, h2 Hex) error {
	if !m.IsValidHex(h1) || !m.IsValidHex(h2) {
		return fmt.Errorf("cannot build bridge: invalid hex coordinates")
	}
	// Endpoints must be non-river
	if m.isRiver(h1) || m.isRiver(h2) {
		return fmt.Errorf("cannot build bridge: endpoints must be land hexes")
	}

	// Validate against allowed bridge geometry
	if ok := m.validateBridgeGeometry(h1, h2); !ok {
		return fmt.Errorf("cannot build bridge: not a valid river-spanning bridge")
	}

	key := NewBridgeKey(h1, h2)
	if m.Bridges[key] {
		return fmt.Errorf("cannot build bridge: bridge already exists")
	}
	m.Bridges[key] = true
	return nil
}

// IsDirectlyAdjacent checks if two hexes are directly adjacent according to Terra Mystica rules:
// 1. They share a hex edge (distance = 1), OR
// 2. They are separated by a river but connected via a bridge
func (m *TerraMysticaMap) IsDirectlyAdjacent(h1, h2 Hex) bool {
	// Natural adjacency (shared edge)
	if h1.IsDirectlyAdjacent(h2) {
		return true
	}
	// Bridge-based adjacency per rules
	if m.HasBridge(h1, h2) {
		return true
	}
	return false
}

// isRiver returns true if the hex is a river hex according to either explicit map or terrain type.
func (m *TerraMysticaMap) isRiver(h Hex) bool {
	if m.RiverHexes[h] {
		return true
	}
	if mx := m.GetHex(h); mx != nil {
		return mx.Terrain == models.TerrainRiver
	}
	return false
}

// validateBridgeGeometry checks if h1->h2 is a valid bridge per the precise rule:
// vector must be one of the 6 allowed distance-2 offsets and the two intermediate
// hexes must both be river hexes.
func (m *TerraMysticaMap) validateBridgeGeometry(h1, h2 Hex) bool {
	dq := h2.Q - h1.Q
	dr := h2.R - h1.R
	delta := Hex{Q: dq, R: dr}

	// Base pattern (target and its two midpoints) for one orientation
	baseTarget := Hex{Q: 1, R: -2}
	midA := Hex{Q: 0, R: -1}
	midB := Hex{Q: 1, R: -1}

	for rot := 0; rot < 6; rot++ {
		rt := rotate60(baseTarget, rot)
		if delta.Equals(rt) {
			ra := rotate60(midA, rot)
			rb := rotate60(midB, rot)
			a := h1.Add(ra)
			b := h1.Add(rb)
			return m.isRiver(a) && m.isRiver(b)
		}
	}
	return false
}

// rotate60 rotates an axial coordinate around origin by k*60 degrees (k in [0..5])
func rotate60(h Hex, k int) Hex {
	// axial -> cube
	x := h.Q
	z := h.R
	y := -x - z
	for i := 0; i < k%6; i++ {
		// 60° rotation: (x,y,z) -> (-z, -x, -y)
		x, y, z = -z, -x, -y
	}
	// cube -> axial
	return Hex{Q: x, R: z}
}

// IsIndirectlyAdjacent checks if two hexes are indirectly adjacent according to Terra Mystica rules:
// 1. They share a hex edge (distance = 1), OR
// 2. They are separated by a river but connected via a bridge
func (m *TerraMysticaMap) IsIndirectlyAdjacent(h1, h2 Hex, shippingValue int) bool {
    // If directly adjacent, not indirectly adjacent
    if m.IsDirectlyAdjacent(h1, h2) {
        return false
    }
    // Endpoints must be land
    if m.isRiver(h1) || m.isRiver(h2) {
        return false
    }

    // River-only BFS: from river neighbors of h1, walk through river hexes
    // up to 'shippingValue' steps; success if any frontier river hex is
    // directly adjacent (edge-sharing) to h2.
    if shippingValue <= 0 {
        return false
    }

    // Seed with river neighbors of h1
    start := m.riverNeighbors(h1)
    if len(start) == 0 {
        return false
    }

    visited := make(map[Hex]bool)
    frontier := start
    steps := 1
    for _, v := range frontier { visited[v] = true }

    for steps <= shippingValue {
        // Check if any river in frontier touches h2
        for _, rv := range frontier {
            if rv.IsDirectlyAdjacent(h2) { // river hex shares edge with h2
                return true
            }
        }
        // Expand frontier if we have remaining steps
        if steps == shippingValue { break }
        next := []Hex{}
        for _, rv := range frontier {
            for _, nbr := range rv.Neighbors() {
                if !m.IsValidHex(nbr) || !m.isRiver(nbr) { continue }
                if visited[nbr] { continue }
                visited[nbr] = true
                next = append(next, nbr)
            }
        }
        frontier = next
        steps++
    }
    return false
}

// riverNeighbors returns all river hexes sharing an edge with h.
func (m *TerraMysticaMap) riverNeighbors(h Hex) []Hex {
    out := []Hex{}
    for _, nbr := range h.Neighbors() {
        if m.IsValidHex(nbr) && m.isRiver(nbr) {
            out = append(out, nbr)
        }
    }
    return out
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
