package game

import "github.com/lukev/tm_server/internal/models"

// Town represents a formed town on the map
type Town struct {
	Hexes      []Hex
	PlayerID   string
	Faction    models.FactionType
	TotalPower int // Sum of building power values
}

// BuildingPowerValue returns the power value of a building type for town formation
func BuildingPowerValue(buildingType models.BuildingType) int {
	switch buildingType {
	case models.BuildingDwelling:
		return 1
	case models.BuildingTradingHouse:
		return 2
	case models.BuildingTemple:
		return 2
	case models.BuildingSanctuary:
		return 3
	case models.BuildingStronghold:
		return 3
	default:
		return 0
	}
}

// DetectTown checks if a group of connected buildings forms a valid town
// A town requires:
// - At least 4 buildings
// - Total power value >= 7
// - All buildings must be directly connected (including via bridges)
func (m *TerraMysticaMap) DetectTown(startHex Hex) *Town {
	mapHex := m.GetHex(startHex)
	if mapHex == nil || mapHex.Building == nil {
		return nil
	}
	
	playerID := mapHex.Building.OwnerPlayerID
	faction := mapHex.Building.Faction
	
	// Find all connected buildings for this player
	connected := m.FindConnectedBuildings(startHex, faction, playerID)
	
	// Check town requirements
	if len(connected) < 4 {
		return nil
	}
	
	// Calculate total power
	totalPower := 0
	for _, hex := range connected {
		if h := m.GetHex(hex); h != nil && h.Building != nil {
			totalPower += BuildingPowerValue(h.Building.Type)
		}
	}
	
	if totalPower < 7 {
		return nil
	}
	
	return &Town{
		Hexes:      connected,
		PlayerID:   playerID,
		Faction:    faction,
		TotalPower: totalPower,
	}
}

// FindConnectedBuildings returns all buildings directly connected to the given hex
// Used for town formation detection
// Includes connections via bridges (bridges can be used to achieve correct power levels)
func (m *TerraMysticaMap) FindConnectedBuildings(startHex Hex, faction models.FactionType, playerID string) []Hex {
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
		
		// Only include buildings of the same faction and player
		if mapHex.Building.Faction != faction || mapHex.Building.OwnerPlayerID != playerID {
			return
		}
		
		connected = append(connected, current)
		
		// Explore direct neighbors (including via bridges)
		for _, neighbor := range m.GetDirectNeighbors(current) {
			dfs(neighbor)
		}
	}
	
	dfs(startHex)
	return connected
}

// DetectAllTowns finds all towns on the map for a given player
func (m *TerraMysticaMap) DetectAllTowns(playerID string) []*Town {
	visited := make(map[Hex]bool)
	towns := []*Town{}
	
	// Check each hex with a building
	for hex, mapHex := range m.Hexes {
		if mapHex.Building == nil {
			continue
		}
		if mapHex.Building.OwnerPlayerID != playerID {
			continue
		}
		if visited[hex] {
			continue
		}
		
		// Try to detect a town starting from this hex
		town := m.DetectTown(hex)
		if town != nil {
			// Mark all hexes in this town as visited
			for _, townHex := range town.Hexes {
				visited[townHex] = true
			}
			towns = append(towns, town)
		}
	}
	
	return towns
}
