package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// TownTileState tracks available town tiles
type TownTileState struct {
	Available map[models.TownTileType]int `json:"available"` // How many of each tile remain
}

// NewTownTileState creates a new town tile state with all tiles available
func NewTownTileState() *TownTileState {
	return &TownTileState{
		Available: map[models.TownTileType]int{
			models.TownTile5Points:  2, // 2 copies
			models.TownTile6Points:  2, // 2 copies
			models.TownTile7Points:  2, // 2 copies
			models.TownTile4Points:  2, // 2 copies (shipping/range upgrade, TW7)
			models.TownTile8Points:  2, // 2 copies
			models.TownTile9Points:  2, // 2 copies
			models.TownTile11Points: 1, // 1 copy
			models.TownTile2Points:  1, // 1 copy
		},
	}
}

// IsAvailable checks if a town tile is still available
func (tts *TownTileState) IsAvailable(tileType models.TownTileType) bool {
	count, ok := tts.Available[tileType]
	return ok && count > 0
}

// TakeTile removes a town tile from the available pool
func (tts *TownTileState) TakeTile(tileType models.TownTileType) error {
	if !tts.IsAvailable(tileType) {
		return fmt.Errorf("town tile %v is not available", tileType)
	}
	tts.Available[tileType]--
	return nil
}

// GetAvailableTiles returns a list of all available town tile types
func (tts *TownTileState) GetAvailableTiles() []models.TownTileType {
	tiles := []models.TownTileType{}
	for tileType, count := range tts.Available {
		if count > 0 {
			tiles = append(tiles, tileType)
		}
	}
	return tiles
}

// Town represents a connected group of buildings
type Town struct {
	Hexes       []board.Hex
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
func calculateAdjacencyBonus(m *board.TerraMysticaMap, h board.Hex, faction models.FactionType) int {
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
func getPowerLeechTargets(m *board.TerraMysticaMap, h board.Hex, placedFaction models.FactionType, powerValue int) map[models.FactionType]int {
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
func (gs *GameState) CheckForTownFormation(playerID string, hex board.Hex) []board.Hex {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return nil
	}

	mapHex := gs.Map.GetHex(hex)
	if mapHex == nil || mapHex.Building == nil {
		return nil
	}

	// Find all connected buildings for this player
	// Mermaids can skip one river hex when founding towns
	var connected []board.Hex
	var skippedRiver *board.Hex

	if player.Faction.GetType() == models.FactionMermaids {
		connected, skippedRiver = gs.Map.GetConnectedBuildingsForMermaids(hex, playerID)
	} else {
		connected = gs.Map.GetConnectedBuildingsIncludingBridges(hex, playerID)
	}

	// Check if any building in the component is already part of a town
	for _, h := range connected {
		mh := gs.Map.GetHex(h)
		if mh != nil && mh.PartOfTown {
			return nil // Already part of a town, cannot form another
		}
	}

	// Check if requirements are met
	if gs.CanFormTown(playerID, connected) {
		// For Mermaids: determine if town can be delayed
		// - If river was skipped (skippedRiver != nil), can be delayed
		// - If only land tiles (skippedRiver == nil), must claim immediately
		canBeDelayed := false
		if player.Faction.GetType() == models.FactionMermaids && skippedRiver != nil {
			canBeDelayed = true
		}

		// Append new pending town formation (supports multiple simultaneous towns)
		newTown := &PendingTownFormation{
			PlayerID:        playerID,
			Hexes:           connected,
			SkippedRiverHex: skippedRiver,
			CanBeDelayed:    canBeDelayed,
		}
		gs.PendingTownFormations[playerID] = append(gs.PendingTownFormations[playerID], newTown)

		return connected
	}

	return nil
}

// CanFormTown checks if the given connected buildings meet town requirements
func (gs *GameState) CanFormTown(playerID string, hexes []board.Hex) bool {
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

	if totalPower < minPower {
		return false
	}

	return true
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
// For Mermaids: if skippedRiverHex is provided, places the town tile on that river hex
func (gs *GameState) FormTown(playerID string, hexes []board.Hex, tileType models.TownTileType, skippedRiverHex *board.Hex) error {
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

	// For Mermaids: Place town tile on the skipped river hex
	if skippedRiverHex != nil {
		riverMapHex := gs.Map.GetHex(*skippedRiverHex)
		if riverMapHex != nil {
			riverMapHex.HasTownTile = true
			riverMapHex.TownTileType = tileType
		}
	}
	// For other factions: Town tile placement is tracked per player, not on the map

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
func (gs *GameState) ApplyTownTileBenefits(playerID string, tileType models.TownTileType) {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return
	}

	// Check if we're in replay mode and should skip resource grants
	skipResources := gs.ReplayMode != nil && gs.ReplayMode[playerID]
	if skipResources {
		// Clear the flag after use
		delete(gs.ReplayMode, playerID)
	}

	switch tileType {
	case models.TownTile5Points:
		player.VictoryPoints += 5
		if !skipResources {
			player.Resources.Coins += 6
		}
		player.Keys += 1

	case models.TownTile6Points:
		player.VictoryPoints += 6
		if !skipResources {
			player.Resources.Power.GainPower(8)
		}
		player.Keys += 1

	case models.TownTile7Points:
		player.VictoryPoints += 7
		if !skipResources {
			player.Resources.Workers += 2
		}
		player.Keys += 1

	case models.TownTile4Points:
		player.VictoryPoints += 4
		player.Keys += 1
		// Advance shipping level by 1 and award VP
		gs.AdvanceShippingLevel(playerID)

	case models.TownTile8Points:
		player.VictoryPoints += 8
		player.Keys += 1
		// Advance 1 on all cult tracks
		gs.CultTracks.ApplyTownCultBonus(playerID, models.TownTile8Points, player, gs)

	case models.TownTile9Points:
		player.VictoryPoints += 9
		if !skipResources {
			gs.GainPriests(playerID, 1)
		}
		player.Keys += 1

	case models.TownTile11Points:
		player.VictoryPoints += 11
		player.Keys += 1

	case models.TownTile2Points:
		player.VictoryPoints += 2
		player.Keys += 2
		// Advance 2 on all cult tracks
		gs.CultTracks.ApplyTownCultBonus(playerID, models.TownTile2Points, player, gs)
	}
}

// ApplyFactionTownBonus applies faction-specific bonuses when forming a town
func (gs *GameState) ApplyFactionTownBonus(playerID string) {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return
	}

	// Apply town founding bonuses
	// Only Witches and Swarmlings get bonuses when founding towns
	switch player.Faction.GetType() {
	case models.FactionWitches:
		// Witches get +5 VP per town formed
		// Witches get +5 VP per town formed
		player.VictoryPoints += 5
	case models.FactionSwarmlings:
		// Swarmlings get +3 workers per town formed
		player.Resources.Workers += 3
	}
}

// GetConnectedBuildingsIncludingBridges finds all buildings connected to the starting hex
// This includes connections via bridges
func getConnectedBuildingsIncludingBridges(m *board.TerraMysticaMap, start board.Hex, playerID string) []board.Hex {
	visited := make(map[board.Hex]bool)
	connected := []board.Hex{}
	queue := []board.Hex{start}
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
func getConnectedBuildingsForMermaids(m *board.TerraMysticaMap, start board.Hex, playerID string) ([]board.Hex, *board.Hex) {
	visited := make(map[board.Hex]bool)
	connected := []board.Hex{}
	queue := []board.Hex{start}
	visited[start] = true
	var skippedRiver *board.Hex

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

// CheckAllTownFormations checks for town formation for all of a player's buildings
// This is useful when a condition changes (e.g. Fire+2 favor tile) that might allow
// existing clusters to form towns
func (gs *GameState) CheckAllTownFormations(playerID string) {
	for hex, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID && !mapHex.PartOfTown {
			gs.CheckForTownFormation(playerID, hex)
		}
	}
}
