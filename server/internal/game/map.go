package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// TerraMysticaMap represents the game board
// Terra Mystica uses a pointy-top hex grid with 9 rows alternating 13/12 hexagons
type TerraMysticaMap struct {
	Hexes         map[Hex]*MapHex
	Bridges       map[BridgeKey]*Bridge  // Tracks built bridges with ownership
	PlayerBridges map[string]int         // Tracks bridge count per player (max 3)
	RiverHexes    map[Hex]bool           // Tracks which hexes are rivers
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

// Bridge represents a bridge built by a player
type Bridge struct {
	Key      BridgeKey
	PlayerID string
}

// NewTerraMysticaMap creates a new game map
func NewTerraMysticaMap() *TerraMysticaMap {
	m := &TerraMysticaMap{
		Hexes:         make(map[Hex]*MapHex),
		Bridges:       make(map[BridgeKey]*Bridge),
		PlayerBridges: make(map[string]int),
		RiverHexes:    make(map[Hex]bool),
	}
	m.initializeBaseMap()
	return m
}

// initializeBaseMap sets up the standard Terra Mystica base game map
// 9 rows alternating 13/12 hexagons, pointy-top orientation
func (m *TerraMysticaMap) initializeBaseMap() {
	// Load the actual Terra Mystica base game terrain layout
	terrainLayout := BaseGameTerrainLayout()
	riverHexes := BaseGameRiverHexes()
	
	// Create all hexes with their terrain
	for hex, terrain := range terrainLayout {
		m.Hexes[hex] = &MapHex{
			Coord:    hex,
			Terrain:  terrain,
			Building: nil,
		}
	}
	
	// Mark river hexes
	for hex := range riverHexes {
		m.RiverHexes[hex] = true
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
	_, exists := m.Bridges[NewBridgeKey(h1, h2)]
	return exists
}

// BuildBridge creates a bridge between two hexes for a player
// Each player can build a maximum of 3 bridges
func (m *TerraMysticaMap) BuildBridge(h1, h2 Hex, playerID string) error {
	if !m.IsValidHex(h1) || !m.IsValidHex(h2) {
		return fmt.Errorf("cannot build bridge: invalid hex coordinates")
	}
	if !h1.IsDirectlyAdjacent(h2) {
		return fmt.Errorf("cannot build bridge: hexes are not adjacent")
	}
	
	key := NewBridgeKey(h1, h2)
	if _, exists := m.Bridges[key]; exists {
		return fmt.Errorf("cannot build bridge: bridge already exists")
	}
	
	// Check player bridge limit (max 3)
	if m.PlayerBridges[playerID] >= 3 {
		return fmt.Errorf("cannot build bridge: player has reached maximum of 3 bridges")
	}
	
	m.Bridges[key] = &Bridge{Key: key, PlayerID: playerID}
	m.PlayerBridges[playerID]++
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

