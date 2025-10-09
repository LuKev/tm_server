package game

import (
	"github.com/lukev/tm_server/internal/models"
)

// Town represents a connected group of buildings
type Town struct {
	Hexes       []Hex
	TotalPower  int
	Faction     models.FactionType
	TownTileKey string // Empty if no town tile selected yet
}

// IsTown checks if a group of hexes forms a valid town
// Requirements: 4+ buildings, total power value >= 7
func IsTown(hexes []Hex, m *TerraMysticaMap) bool {
	if len(hexes) < 4 {
		return false
	}
	
	totalPower := 0
	for _, h := range hexes {
		mapHex := m.GetHex(h)
		if mapHex == nil || mapHex.Building == nil {
			return false
		}
		totalPower += mapHex.Building.PowerValue
	}
	
	return totalPower >= 7
}

// DetectTowns finds all towns for a given faction on the map
func (m *TerraMysticaMap) DetectTowns(faction models.FactionType) []Town {
	visited := make(map[Hex]bool)
	towns := []Town{}
	
	// Find all hexes with buildings of this faction
	for hex, mapHex := range m.Hexes {
		if mapHex.Building != nil && mapHex.Building.Faction == faction && !visited[hex] {
			// Find connected component
			connected := m.FindConnectedBuildings(hex, faction)
			
			// Mark all as visited
			for _, h := range connected {
				visited[h] = true
			}
			
			// Check if it forms a town
			if IsTown(connected, m) {
				totalPower := 0
				for _, h := range connected {
					if mh := m.GetHex(h); mh != nil && mh.Building != nil {
						totalPower += mh.Building.PowerValue
					}
				}
				
				towns = append(towns, Town{
					Hexes:      connected,
					TotalPower: totalPower,
					Faction:    faction,
				})
			}
		}
	}
	
	return towns
}

// GetPowerValue returns the power value of a building type
func GetPowerValue(buildingType models.BuildingType) int {
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
