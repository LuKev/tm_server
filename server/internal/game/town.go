package game

import (
	"fmt"
	
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// TownTileState tracks available town tiles
type TownTileState struct {
	Available map[TownTileType]int // How many of each tile remain
}

// NewTownTileState creates a new town tile state with all tiles available
func NewTownTileState() *TownTileState {
	return &TownTileState{
		Available: map[TownTileType]int{
			TownTile5Points:  2, // 2 copies
			TownTile6Points:  2, // 2 copies
			TownTile7Points:  2, // 2 copies
			TownTile8Points:  2, // 2 copies
			TownTile9Points:  2, // 2 copies
			TownTile11Points: 1, // 1 copy
			TownTile2Points:  1, // 1 copy
		},
	}
}

// IsAvailable checks if a town tile is still available
func (tts *TownTileState) IsAvailable(tileType TownTileType) bool {
	count, ok := tts.Available[tileType]
	return ok && count > 0
}

// TakeTile removes a town tile from the available pool
func (tts *TownTileState) TakeTile(tileType TownTileType) error {
	if !tts.IsAvailable(tileType) {
		return fmt.Errorf("town tile %v is not available", tileType)
	}
	tts.Available[tileType]--
	return nil
}

// GetAvailableTiles returns a list of all available town tile types
func (tts *TownTileState) GetAvailableTiles() []TownTileType {
	tiles := []TownTileType{}
	for tileType, count := range tts.Available {
		if count > 0 {
			tiles = append(tiles, tileType)
		}
	}
	return tiles
}

// Town represents a connected group of buildings
type Town struct {
	Hexes       []Hex
	TotalPower  int
	Faction     models.FactionType
	TownTileKey string // Empty if no town tile selected yet
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

// CheckForTownFormation checks if a town can be formed after building/upgrading at the given hex
// Returns the connected buildings if a town can be formed, nil otherwise
func (gs *GameState) CheckForTownFormation(playerID string, hex Hex) []Hex {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return nil
	}
	
	mapHex := gs.Map.GetHex(hex)
	if mapHex == nil || mapHex.Building == nil {
		return nil
	}
	
	// Find all connected buildings for this player
	connected := gs.Map.GetConnectedBuildingsIncludingBridges(hex, playerID)
	
	// Check if any building in the component is already part of a town
	for _, h := range connected {
		mh := gs.Map.GetHex(h)
		if mh != nil && mh.PartOfTown {
			return nil // Already part of a town, cannot form another
		}
	}
	
	// Check if requirements are met
	if gs.CanFormTown(playerID, connected) {
		return connected
	}
	
	return nil
}

// CanFormTown checks if the given connected buildings meet town requirements
func (gs *GameState) CanFormTown(playerID string, hexes []Hex) bool {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return false
	}
	
	// Count buildings and calculate total power
	buildingCount := 0
	totalPower := 0
	hasSanctuary := false
	
	for _, h := range hexes {
		mapHex := gs.Map.GetHex(h)
		if mapHex != nil && mapHex.Building != nil {
			buildingCount++
			totalPower += GetPowerValue(mapHex.Building.Type)
			if mapHex.Building.Type == models.BuildingSanctuary {
				hasSanctuary = true
			}
		}
	}
	
	// Check building count requirement
	minBuildings := 4
	if hasSanctuary {
		minBuildings = 3 // Sanctuary allows town with 3 buildings
	}
	
	if buildingCount < minBuildings {
		return false
	}
	
	// Check power requirement (6 with Fire 2 favor tile, 7 otherwise)
	minPower := gs.GetTownPowerRequirement(playerID)
	
	return totalPower >= minPower
}

// GetTownPowerRequirement returns the minimum power required for a town
// Returns 6 if player has Fire 2 favor tile, 7 otherwise
func (gs *GameState) GetTownPowerRequirement(playerID string) int {
	// Check if player has Fire 2 favor tile
	playerTiles := gs.FavorTiles.GetPlayerTiles(playerID)
	for _, tile := range playerTiles {
		if tile == FavorFire2 {
			return 6
		}
	}
	return 7
}

// FormTown marks the buildings as part of a town and applies town tile benefits
func (gs *GameState) FormTown(playerID string, hexes []Hex, tileType TownTileType) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}
	
	// Check if tile is available
	if !gs.TownTiles.IsAvailable(tileType) {
		return fmt.Errorf("town tile %v is not available", tileType)
	}
	
	// Mark all buildings as part of a town
	for _, h := range hexes {
		mapHex := gs.Map.GetHex(h)
		if mapHex != nil {
			mapHex.PartOfTown = true
		}
	}
	
	// Take the tile
	if err := gs.TownTiles.TakeTile(tileType); err != nil {
		return err
	}
	
	// Add to player's town tiles
	player.TownTiles = append(player.TownTiles, tileType)
	player.TownsFormed++
	
	// Apply immediate benefits
	gs.ApplyTownTileBenefits(playerID, tileType)
	
	// Apply faction-specific town bonuses
	gs.ApplyFactionTownBonus(playerID)
	
	// Award VP from scoring tile
	gs.AwardActionVP(playerID, ScoringActionTown)
	
	return nil
}

// ApplyTownTileBenefits applies the immediate benefits of a town tile
func (gs *GameState) ApplyTownTileBenefits(playerID string, tileType TownTileType) {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return
	}
	
	switch tileType {
	case TownTile5Points:
		player.VictoryPoints += 5
		player.Resources.Coins += 6
		player.Keys += 1
		
	case TownTile6Points:
		player.VictoryPoints += 6
		player.Resources.Power.GainPower(8)
		player.Keys += 1
		
	case TownTile7Points:
		player.VictoryPoints += 7
		player.Resources.Workers += 2
		player.Keys += 1
		
	case TownTile8Points:
		player.VictoryPoints += 8
		player.Keys += 1
		// Advance 1 on all cult tracks
		gs.CultTracks.ApplyTownCultBonus(playerID, TownTile8Points, player)
		
	case TownTile9Points:
		player.VictoryPoints += 9
		player.Resources.Priests += 1
		player.Keys += 1
		
	case TownTile11Points:
		player.VictoryPoints += 11
		player.Keys += 1
		
	case TownTile2Points:
		player.VictoryPoints += 2
		player.Keys += 2
		// Advance 2 on all cult tracks
		gs.CultTracks.ApplyTownCultBonus(playerID, TownTile2Points, player)
	}
}

// ApplyFactionTownBonus applies faction-specific bonuses when forming a town
func (gs *GameState) ApplyFactionTownBonus(playerID string) {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return
	}
	
	// Check if faction has the town bonus ability
	if player.Faction.HasSpecialAbility(factions.AbilityTownBonus) {
		switch player.Faction.GetType() {
		case models.FactionWitches:
			// Witches get +5 VP per town formed
			if witches, ok := player.Faction.(*factions.Witches); ok {
				player.VictoryPoints += witches.GetTownFoundingBonus()
			}
		case models.FactionMermaids:
			// Mermaids get +3 power per town formed
			player.Resources.Power.GainPower(3)
		}
	}
	
	// Swarmlings get +3 workers per town formed (not part of AbilityTownBonus, separate mechanic)
	if player.Faction.GetType() == models.FactionSwarmlings {
		player.Resources.Workers += 3
	}
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
