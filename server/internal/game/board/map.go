package board

import (
	"encoding/json"
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// TerraMysticaMap represents the game board
// Terra Mystica uses a pointy-top hex grid with 9 rows alternating 13/12 hexagons
type TerraMysticaMap struct {
	Hexes      map[Hex]*MapHex
	Bridges    map[BridgeKey]bool // Tracks built bridges between hexes
	RiverHexes map[Hex]bool       // Tracks which hexes are rivers
}

// MarshalJSON implements custom JSON marshaling for TerraMysticaMap
// Go's encoding/json doesn't support struct keys in maps, so we convert them to strings
func (m *TerraMysticaMap) MarshalJSON() ([]byte, error) {

	// Convert Hexes map
	hexes := make(map[string]*MapHex)
	for k, v := range m.Hexes {
		hexes[fmt.Sprintf("%d,%d", k.Q, k.R)] = v
	}

	// Convert Bridges map
	bridges := make(map[string]bool)
	for k, v := range m.Bridges {
		bridges[fmt.Sprintf("%d,%d|%d,%d", k.H1.Q, k.H1.R, k.H2.Q, k.H2.R)] = v
	}

	// Convert RiverHexes map
	riverHexes := make(map[string]bool)
	for k, v := range m.RiverHexes {
		riverHexes[fmt.Sprintf("%d,%d", k.Q, k.R)] = v
	}

	return json.Marshal(&struct {
		Hexes      map[string]*MapHex `json:"hexes"`
		Bridges    map[string]bool    `json:"bridges"`
		RiverHexes map[string]bool    `json:"riverHexes"`
	}{
		Hexes:      hexes,
		Bridges:    bridges,
		RiverHexes: riverHexes,
	})
}

// MapHex represents a single hex on the map
type MapHex struct {
	Coord        Hex
	Terrain      models.TerrainType
	Building     *models.Building    // nil if no building
	PartOfTown   bool                // true if this building is part of a town
	HasTownTile  bool                // For Mermaids: true if a town tile is placed on this hex (river)
	TownTileType models.TownTileType // For Mermaids: the type of town tile placed on this hex
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

// GetHex returns the MapHex at the given coordinates, or nil if not found
func (m *TerraMysticaMap) GetHex(h Hex) *MapHex {
	return m.Hexes[h]
}

// GetTerrainDistance returns the number of spades needed to transform from one terrain to another
func (m *TerraMysticaMap) GetTerrainDistance(from, to models.TerrainType) int {
	return TerrainDistance(from, to)
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
//   - Connect two non-river hexes
//   - Span across the edge of a river hex: the vector (h2 - h1) must be one of the 6 allowed
//     distance-2 offsets: (1,-2), (2,-1), (2,0), (0,2), (-2,2), (-2,0) up to rotation,
//     and the two intermediate hexes along that edge must both be river hexes.
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

// CountBridgesConnectingPlayerStructures counts bridges connecting two of the player's structures
// This is used for Engineers' stronghold ability: 3 VP per bridge connecting two structures when passing
func (m *TerraMysticaMap) CountBridgesConnectingPlayerStructures(playerID string) int {
	count := 0
	for bridgeKey := range m.Bridges {
		// Check if both endpoints have the player's buildings
		hex1 := m.GetHex(bridgeKey.H1)
		hex2 := m.GetHex(bridgeKey.H2)

		if hex1 != nil && hex1.Building != nil && hex1.Building.PlayerID == playerID &&
			hex2 != nil && hex2.Building != nil && hex2.Building.PlayerID == playerID {
			count++
		}
	}
	return count
}

// IsWithinSkipRange checks if a target hex is reachable with skip ability (Fakirs/Dwarves)
// Skip allows reaching a hex by skipping OVER skipRange hexes
// For example, with skipRange=1, you can reach distance 2 (skip 1 hex to get there)
// With skipRange=2, you can reach distance 3 (skip 2 hexes to get there)
func (m *TerraMysticaMap) IsWithinSkipRange(target Hex, playerID string, skipRange int) bool {
	// Find all hexes with player's buildings
	for hex, mapHex := range m.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			// Check if target is within skip range of this building
			// Distance should be skip range + 1 (you skip OVER that many hexes)
			distance := hex.Distance(target)
			if distance <= skipRange+1 {
				return true
			}
		}
	}
	return false
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
	for _, v := range frontier {
		visited[v] = true
	}

	for steps <= shippingValue {
		// Check if any river in frontier touches h2
		for _, rv := range frontier {
			if rv.IsDirectlyAdjacent(h2) { // river hex shares edge with h2
				return true
			}
		}
		// Expand frontier if we have remaining steps
		if steps == shippingValue {
			break
		}
		next := []Hex{}
		for _, rv := range frontier {
			for _, nbr := range rv.Neighbors() {
				if !m.IsValidHex(nbr) || !m.isRiver(nbr) {
					continue
				}
				if visited[nbr] {
					continue
				}
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

	// Add natural neighbors (distance 1)
	for _, neighbor := range h.Neighbors() {
		if m.IsValidHex(neighbor) && m.IsDirectlyAdjacent(h, neighbor) {
			neighbors = append(neighbors, neighbor)
		}
	}

	// Add bridge-connected neighbors (distance 2)
	// Bridges connect hexes that are not naturally adjacent
	for bridgeKey := range m.Bridges {
		if bridgeKey.H1.Equals(h) && m.IsValidHex(bridgeKey.H2) {
			neighbors = append(neighbors, bridgeKey.H2)
		} else if bridgeKey.H2.Equals(h) && m.IsValidHex(bridgeKey.H1) {
			neighbors = append(neighbors, bridgeKey.H1)
		}
	}

	return neighbors
}

// GetIndirectNeighbors returns all indirectly adjacent hexes within shipping range
func (m *TerraMysticaMap) GetIndirectNeighbors(h Hex, shippingValue int) []Hex {
	neighbors := []Hex{}
	// Check all hexes within range
	for candidate := range m.Hexes {
		// Skip the source hex itself
		if candidate == h {
			continue
		}
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

	// Note: Terrain and adjacency checks are performed by the calling action
	// (e.g., TransformAndBuildAction validates terrain, adjacency, etc.)

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

// GetLargestConnectedArea returns the size of the largest connected area for a player
// Used for final scoring (18/12/6 VP for 1st/2nd/3rd place)
// Considers direct adjacency, bridges, and shipping for connectivity
func (m *TerraMysticaMap) GetLargestConnectedArea(playerID string, faction factions.Faction, shippingLevel int) int {
	visited := make(map[Hex]bool)
	maxArea := 0

	// Find all buildings belonging to this player
	for hex, mapHex := range m.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID && !visited[hex] {
			// Start a new connected component search
			component := m.getConnectedComponent(hex, playerID, visited, faction, shippingLevel)
			area := len(component)
			if area > maxArea {
				maxArea = area
			}
		}
	}

	return maxArea
}

// getConnectedComponent returns all hexes in the connected component starting from a hex
func (m *TerraMysticaMap) getConnectedComponent(start Hex, playerID string, visited map[Hex]bool, faction factions.Faction, shippingLevel int) []Hex {
	if visited[start] {
		return nil
	}

	mapHex := m.GetHex(start)
	if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
		return nil
	}

	visited[start] = true
	component := []Hex{start}

	neighbors := m.getNeighborsForAreaScoring(start, faction, shippingLevel)
	for _, neighbor := range neighbors {
		component = append(component, m.getConnectedComponent(neighbor, playerID, visited, faction, shippingLevel)...)
	}

	return component
}

// getConnectedAreaSize returns the size of the connected area starting from a hex
// Uses DFS to explore all connected buildings
// Handles faction-specific adjacency: Fakirs (carpet flight), Dwarves (tunneling), and shipping
func (m *TerraMysticaMap) getConnectedAreaSize(start Hex, playerID string, visited map[Hex]bool, faction factions.Faction, shippingLevel int) int {
	if visited[start] {
		return 0
	}

	mapHex := m.GetHex(start)
	if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
		return 0
	}

	// Mark as visited
	visited[start] = true
	size := 1

	// Get neighbors based on faction-specific movement and shipping
	neighbors := m.getNeighborsForAreaScoring(start, faction, shippingLevel)

	for _, neighbor := range neighbors {
		size += m.getConnectedAreaSize(neighbor, playerID, visited, faction, shippingLevel)
	}

	return size
}

// getNeighborsForAreaScoring returns neighbors based on faction-specific movement and shipping
// For final area scoring: Fakirs use carpet flight, Dwarves use tunneling, others use direct adjacency + shipping
func (m *TerraMysticaMap) getNeighborsForAreaScoring(h Hex, faction factions.Faction, shippingLevel int) []Hex {
	neighbors := []Hex{}

	// Check for Fakirs (carpet flight)
	// Fakirs can connect via carpet flight (range 1-3 depending on upgrades)
	if fakir, ok := faction.(*factions.Fakirs); ok {
		flightRange := 1
		if fakir.HasStronghold() {
			flightRange++
		}
		if fakir.HasShippingTownTile() {
			flightRange++
		}

		// Get all hexes within flight range
		for candidate := range m.Hexes {
			if candidate == h {
				continue
			}

			distance := h.Distance(candidate)
			if distance <= flightRange {
				neighbors = append(neighbors, candidate)
			}
		}
		return neighbors
	}

	// Check for Dwarves (tunneling - always range 2)
	if faction.GetType() == models.FactionDwarves {
		// Dwarves can connect via tunneling (distance 2)
		for candidate := range m.Hexes {
			if candidate == h {
				continue
			}

			distance := h.Distance(candidate)
			if distance <= 2 {
				neighbors = append(neighbors, candidate)
			}
		}
		return neighbors
	}

	// Default: direct adjacency (including bridges) + indirect adjacency via shipping
	neighbors = m.GetDirectNeighbors(h)

	// Add indirect neighbors via shipping (if shipping level > 0)
	if shippingLevel > 0 {
		indirectNeighbors := m.GetIndirectNeighbors(h, shippingLevel)
		neighbors = append(neighbors, indirectNeighbors...)
	}

	return neighbors
}

// CalculateAdjacencyBonus calculates coins gained from adjacent opponent buildings
// when placing a new building
func (m *TerraMysticaMap) CalculateAdjacencyBonus(h Hex, faction models.FactionType) int {
	bonus := 0

	for _, neighbor := range m.GetDirectNeighbors(h) {
		mapHex := m.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil {
			// Adjacent opponent building gives 1 coin
			if mapHex.Building.Faction != faction {
				bonus++
			}
		}
	}

	return bonus
}

// GetPowerLeechTargets returns all players who can leech power from a building placement
// Returns map of faction -> power amount they can leech
func (m *TerraMysticaMap) GetPowerLeechTargets(h Hex, placedFaction models.FactionType, powerValue int) map[models.FactionType]int {
	targets := make(map[models.FactionType]int)

	for _, neighbor := range m.GetDirectNeighbors(h) {
		mapHex := m.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil {
			// Can only leech from opponent buildings
			if mapHex.Building.Faction != placedFaction {
				faction := mapHex.Building.Faction
				// Power leech amount equals the power value of the placed building
				if existing, ok := targets[faction]; ok {
					targets[faction] = existing + powerValue
				} else {
					targets[faction] = powerValue
				}
			}
		}
	}

	return targets
}

// GetConnectedBuildingsIncludingBridges finds all buildings connected to the starting hex
// This includes connections via bridges
func (m *TerraMysticaMap) GetConnectedBuildingsIncludingBridges(start Hex, playerID string) []Hex {
	visited := make(map[Hex]bool)
	connected := []Hex{}
	queue := []Hex{start}
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentHex := m.GetHex(current)
		if currentHex == nil || currentHex.Building == nil {
			continue
		}

		// Only include buildings belonging to this player
		if currentHex.Building.PlayerID != playerID {
			continue
		}

		connected = append(connected, current)

		// Get all adjacent hexes (including via bridges)
		neighbors := m.GetDirectNeighbors(current)

		// Also check for bridge connections
		for bridge := range m.Bridges {
			if bridge.H1 == current {
				neighbors = append(neighbors, bridge.H2)
			} else if bridge.H2 == current {
				neighbors = append(neighbors, bridge.H1)
			}
		}

		// Add unvisited neighbors to queue
		for _, neighbor := range neighbors {
			if !visited[neighbor] {
				neighborHex := m.GetHex(neighbor)
				if neighborHex != nil && neighborHex.Building != nil && neighborHex.Building.PlayerID == playerID {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
	}

	return connected
}

// GetConnectedBuildingsForMermaids finds all buildings connected to the starting hex
// Mermaids can skip ONE river hex when forming towns (town tile goes on the skipped river)
// Returns: connected buildings, and the skipped river hex (if any)
func (m *TerraMysticaMap) GetConnectedBuildingsForMermaids(start Hex, playerID string) ([]Hex, *Hex) {
	visited := make(map[Hex]bool)
	connected := []Hex{}
	queue := []Hex{start}
	visited[start] = true
	var skippedRiver *Hex

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentHex := m.GetHex(current)
		if currentHex == nil || currentHex.Building == nil {
			continue
		}

		// Only include buildings belonging to this player
		if currentHex.Building.PlayerID != playerID {
			continue
		}

		connected = append(connected, current)

		// Get all adjacent hexes (including via bridges)
		neighbors := m.GetDirectNeighbors(current)

		// Also check for bridge connections
		for bridge := range m.Bridges {
			if bridge.H1 == current {
				neighbors = append(neighbors, bridge.H2)
			} else if bridge.H2 == current {
				neighbors = append(neighbors, bridge.H1)
			}
		}

		// Add unvisited neighbors to queue
		for _, neighbor := range neighbors {
			if !visited[neighbor] {
				neighborHex := m.GetHex(neighbor)
				if neighborHex != nil && neighborHex.Building != nil && neighborHex.Building.PlayerID == playerID {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}

		// Mermaids special ability: Check for buildings across ONE river hex
		// Only use this ability once per town formation
		if skippedRiver == nil {
			for _, neighbor := range neighbors {
				neighborHex := m.GetHex(neighbor)
				// Check if this neighbor is a river hex
				if neighborHex != nil && neighborHex.Terrain == models.TerrainRiver && !visited[neighbor] {
					// Check hexes on the other side of this river
					riverNeighbors := neighbor.Neighbors()
					for _, acrossRiver := range riverNeighbors {
						if acrossRiver == current {
							continue // Skip the hex we came from
						}
						if !visited[acrossRiver] {
							acrossHex := m.GetHex(acrossRiver)
							// If there's a player building on the other side, connect it
							if acrossHex != nil && acrossHex.Building != nil && acrossHex.Building.PlayerID == playerID {
								visited[acrossRiver] = true
								queue = append(queue, acrossRiver)
								skippedRiver = &neighbor // Track which river was skipped
							}
						}
					}
				}
			}
		}
	}

	return connected, skippedRiver
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
func (m *TerraMysticaMap) Terraform(h Hex, terrain models.TerrainType) error {
	if err := m.CanTerraform(h); err != nil {
		return err
	}

	mapHex := m.GetHex(h)
	if mapHex.Terrain == terrain {
		return fmt.Errorf("hex is already %s", terrain)
	}

	mapHex.Terrain = terrain
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
			// Check if this building is directly adjacent to the target hex
			if m.IsDirectlyAdjacent(hex, h) {
				hasAdjacentBuilding = true
				break
			}
			// Note: Indirect adjacency via shipping is checked by the calling action (state.CanPlaceBuilding)
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
