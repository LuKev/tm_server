package game

import (
	"fmt"
	"sort"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
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

// NewBaseTownTileState creates the five town-tile types from the original
// Terra Mystica rules. Fire & Ice adds the shipping, 11 VP, and 2 VP tiles.
func NewBaseTownTileState() *TownTileState {
	return &TownTileState{
		Available: map[models.TownTileType]int{
			models.TownTile5Points: 2,
			models.TownTile6Points: 2,
			models.TownTile7Points: 2,
			models.TownTile8Points: 2,
			models.TownTile9Points: 2,
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
	if player.Faction.GetType() == models.FactionMermaids {
		choices := gs.TownFormationChoices(playerID)
		gs.PendingTownFormations[playerID] = choices
		for _, choice := range choices {
			for _, h := range choice.Hexes {
				if h == hex {
					return choice.Hexes
				}
			}
		}
		return nil
	}

	// Find all connected buildings for this player.
	var connected []board.Hex

	if player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
		connected = gs.getConnectedBuildingsForPlayer(playerID, hex)
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
		gs.createPendingTown(playerID, connected, nil, player.Faction.GetType())
		return connected
	}

	return nil
}

// TownFormationChoices derives Mermaid opportunities from the current board.
// Delaying a river town reserves neither its buildings nor its river: every
// possible river is a player choice, and overlap with an existing town makes
// the entire connected group ineligible. Other factions have mandatory rewards
// already recorded by the action that founded their towns.
func (gs *GameState) TownFormationChoices(playerID string) []*PendingTownFormation {
	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction.GetType() != models.FactionMermaids {
		return gs.PendingTownFormations[playerID]
	}
	var choices []*PendingTownFormation
	eligible := func(hexes []board.Hex) bool {
		sort.Slice(hexes, func(i, j int) bool {
			if hexes[i].R != hexes[j].R {
				return hexes[i].R < hexes[j].R
			}
			return hexes[i].Q < hexes[j].Q
		})
		for _, h := range hexes {
			if gs.Map.GetHex(h).PartOfTown {
				return false
			}
		}
		return gs.CanFormTown(playerID, hexes)
	}
	seen := make(map[board.Hex]bool)
	mandatory := make(map[board.Hex]bool)
	for _, hex := range gs.getPlayerBuildingHexes(playerID) {
		if seen[hex] {
			continue
		}
		component := gs.Map.GetConnectedBuildingsIncludingBridges(hex, playerID)
		for _, h := range component {
			seen[h] = true
		}
		if eligible(component) {
			choices = append(choices, &PendingTownFormation{PlayerID: playerID, Hexes: component})
			for _, h := range component {
				mandatory[h] = true
			}
		}
	}
	for river, mapHex := range gs.Map.Hexes {
		if mapHex.Terrain != models.TerrainRiver || mapHex.HasTownTile {
			continue
		}
		for _, adjacent := range river.Neighbors() {
			neighbor := gs.Map.GetHex(adjacent)
			if neighbor == nil || neighbor.Building == nil || neighbor.Building.PlayerID != playerID {
				continue
			}
			component := gs.Map.GetConnectedBuildingsForMermaidsUsingRiver(adjacent, playerID, river)
			base := gs.Map.GetConnectedBuildingsIncludingBridges(adjacent, playerID)
			if len(component) == len(base) {
				break
			} // The river must actually connect groups.
			blocked := false
			for _, h := range component {
				if mandatory[h] {
					blocked = true
				}
			}
			if !blocked && eligible(component) {
				r := river
				choices = append(choices, &PendingTownFormation{PlayerID: playerID, Hexes: component, SkippedRiverHex: &r, CanBeDelayed: true})
			}
			break // Every owned neighbor of this river yields the same union.
		}
	}
	sort.Slice(choices, func(i, j int) bool {
		if choices[i].CanBeDelayed != choices[j].CanBeDelayed {
			return !choices[i].CanBeDelayed
		}
		a, b := gs.defaultTownAnchorHex(playerID, choices[i]), gs.defaultTownAnchorHex(playerID, choices[j])
		if a.R != b.R {
			return a.R < b.R
		}
		return a.Q < b.Q
	})
	return choices
}

func (gs *GameState) createPendingTown(playerID string, connected []board.Hex, skippedRiver *board.Hex, factionType models.FactionType) {
	// A founded town's reward may still be pending while another immediate
	// effect grows or merges its component (e.g. Halflings' dwelling). Keep all
	// existing rewards/keys, but do not award another town for that same group.
	reserved := make(map[board.Hex]bool)
	var overlap *PendingTownFormation
	for _, existing := range gs.PendingTownFormations[playerID] {
		if existing == nil || existing.CanBeDelayed {
			continue
		}
		for _, h := range existing.Hexes {
			reserved[h] = true
			if overlap == nil {
				for _, candidate := range connected {
					if h == candidate {
						overlap = existing
						break
					}
				}
			}
		}
	}
	if overlap != nil {
		for _, h := range connected {
			if !reserved[h] {
				overlap.Hexes = append(overlap.Hexes, h)
			}
		}
		return
	}
	// Avoid duplicate pending formations for the same connected component.
	// This can happen when a full-board recheck (e.g. after taking Fire+2) touches
	// multiple buildings within the same component before the town is claimed.
	for _, existing := range gs.PendingTownFormations[playerID] {
		if existing == nil {
			continue
		}
		if !sameTownComponent(existing.Hexes, connected) {
			continue
		}
		// Preserve a discovered skipped-river placement for Mermaids if the
		// existing pending formation doesn't already have one.
		if existing.SkippedRiverHex == nil && skippedRiver != nil {
			existing.SkippedRiverHex = skippedRiver
			existing.CanBeDelayed = factionType == models.FactionMermaids
		}
		return
	}

	// For Mermaids: determine if town can be delayed
	// - If river was skipped (skippedRiver != nil), can be delayed
	// - If only land tiles (skippedRiver == nil), must claim immediately
	canBeDelayed := false
	if factionType == models.FactionMermaids && skippedRiver != nil {
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
}

func sameTownComponent(a, b []board.Hex) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	counts := make(map[board.Hex]int, len(a))
	for _, h := range a {
		counts[h]++
	}
	for _, h := range b {
		count, ok := counts[h]
		if !ok || count == 0 {
			return false
		}
		counts[h] = count - 1
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
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
	hasStronghold := false
	hexSet := make(map[board.Hex]bool, len(hexes))

	for _, h := range hexes {
		hexSet[h] = true
		mapHex := gs.Map.GetHex(h)
		if mapHex != nil && mapHex.Building != nil {
			buildingCount++
			if mapHex.Building.PowerValue > 0 {
				totalPower += mapHex.Building.PowerValue
			} else {
				totalPower += GetPowerValue(mapHex.Building.Type)
			}
			if mapHex.Building.Type == models.BuildingSanctuary {
				hasSanctuary = true
			}
			if mapHex.Building.Type == models.BuildingStronghold {
				hasStronghold = true
			}
		}
	}

	if isArchitects(player) {
		totalPower += gs.Map.CountPlayerBridgesWithinHexSet(playerID, hexSet)
	}

	// Check building count requirement
	minBuildings := 4
	if hasSanctuary {
		minBuildings = 3 // Sanctuary allows town with 3 buildings
	}
	if player.Faction.GetType() == models.FactionDynionGeifr && hasStronghold {
		minBuildings = 3
	}

	if buildingCount < minBuildings {
		return false
	}

	// Check power requirement (6 with Fire 2 favor tile, 7 otherwise)
	minPower := gs.GetTownPowerRequirement(playerID)

	return totalPower >= minPower
}

func (gs *GameState) defaultTownAnchorHex(playerID string, pending *PendingTownFormation) *board.Hex {
	if pending == nil {
		return nil
	}
	if pending.SkippedRiverHex != nil {
		anchor := *pending.SkippedRiverHex
		return &anchor
	}
	for _, hex := range pending.Hexes {
		mapHex := gs.Map.GetHex(hex)
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			anchor := hex
			return &anchor
		}
	}
	return nil
}

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

func (gs *GameState) FormTown(playerID string, hexes []board.Hex, tileType models.TownTileType, skippedRiverHex *board.Hex) error {
	return gs.FormTownWithAnchor(playerID, hexes, tileType, skippedRiverHex, gs.defaultTownAnchorHex(playerID, &PendingTownFormation{
		PlayerID:        playerID,
		Hexes:           hexes,
		SkippedRiverHex: skippedRiverHex,
	}))
}

// FormTownWithAnchor marks the buildings as part of a town, stores the chosen town
// anchor hex, and applies town tile benefits.
func (gs *GameState) FormTownWithAnchor(playerID string, hexes []board.Hex, tileType models.TownTileType, skippedRiverHex *board.Hex, anchorHex *board.Hex) error {
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

	townTileHex := anchorHex
	if skippedRiverHex != nil {
		townTileHex = skippedRiverHex
	} else {
		if anchorHex == nil {
			return fmt.Errorf("town anchor hex is required")
		}
		isValidAnchor := false
		for _, hex := range hexes {
			if hex != *anchorHex {
				continue
			}
			mapHex := gs.Map.GetHex(hex)
			if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
				isValidAnchor = true
			}
			break
		}
		if !isValidAnchor {
			return fmt.Errorf("town anchor must be one of the town's building hexes")
		}
	}

	if townTileHex == nil {
		return fmt.Errorf("town tile hex is required")
	}
	townTileMapHex := gs.Map.GetHex(*townTileHex)
	if townTileMapHex == nil {
		return fmt.Errorf("town tile hex does not exist")
	}
	townTileMapHex.HasTownTile = true
	townTileMapHex.TownTileType = tileType
	townTileMapHex.TownTileOwnerPlayerID = playerID

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

	gs.applyTownTileSpecifics(player, tileType, skipResources)
}

func (gs *GameState) applyTownTileSpecifics(player *Player, tileType models.TownTileType, skipResources bool) {
	switch tileType {
	case models.TownTile5Points:
		player.VictoryPoints += 5
		if !skipResources {
			player.Resources.Coins += 6
		}
		player.Keys++

	case models.TownTile6Points:
		player.VictoryPoints += 6
		if !skipResources {
			player.Resources.Power.GainPower(8)
		}
		player.Keys++

	case models.TownTile7Points:
		player.VictoryPoints += 7
		if !skipResources {
			player.Resources.Workers += 2
		}
		player.Keys++

	case models.TownTile4Points:
		player.VictoryPoints += 4
		player.Keys++

		// Fakirs get carpet flight range upgrade instead of shipping
		if fakirs, ok := player.Faction.(*factions.Fakirs); ok {
			fakirs.IncrementFlightRange()
		} else {
			// Advance shipping level by 1 and award VP
			_ = gs.AdvanceShippingLevel(player.ID)
		}

	case models.TownTile8Points:
		player.VictoryPoints += 8
		player.Keys++
		gs.applyTownCultBonusWithPotentialTopChoice(player, 1)

	case models.TownTile9Points:
		player.VictoryPoints += 9
		if !skipResources {
			gs.GainPriests(player.ID, 1)
		}
		player.Keys++

	case models.TownTile11Points:
		player.VictoryPoints += 11
		player.Keys++

	case models.TownTile2Points:
		player.VictoryPoints += 2
		player.Keys += 2
		gs.applyTownCultBonusWithPotentialTopChoice(player, 2)
	}
}

func (gs *GameState) applyTownCultBonusWithPotentialTopChoice(player *Player, advanceAmount int) {
	if player == nil || gs.CultTracks == nil {
		return
	}

	candidates := gs.CultTracks.GetTownCultTopCandidates(player.ID, advanceAmount, player, gs)
	// Every simultaneous mandatory town already grants a key, even when its
	// tile reward is chosen later. AdvancePlayer records borrowed keys as debt.
	maxSelections := player.Keys + countBorrowablePendingTownKeys(gs, player.ID)
	if maxSelections < 0 {
		maxSelections = 0
	}

	if len(candidates) > maxSelections {
		gs.PendingTownCultTopChoice = &PendingTownCultTopChoice{
			PlayerID:        player.ID,
			AdvanceAmount:   advanceAmount,
			CandidateTracks: candidates,
			MaxSelections:   maxSelections,
		}
		return
	}

	gs.CultTracks.ApplyTownCultBonusWithTopChoice(player.ID, advanceAmount, player, gs, nil)
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
	case models.FactionGoblins:
		if player.HasStrongholdAbility {
			player.GoblinTreasureTokens++
		}
	}
}

func (gs *GameState) updateAtlanteansStrongholdTown(playerID string) {
	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil || player.Faction.GetType() != models.FactionAtlanteans {
		return
	}
	if len(player.AtlanteansTownHexes) == 0 {
		return
	}
	if player.AtlanteansTownRewards == nil {
		player.AtlanteansTownRewards = make(map[int]bool)
	}

	known := make(map[board.Hex]bool, len(player.AtlanteansTownHexes))
	queue := make([]board.Hex, 0, len(player.AtlanteansTownHexes))
	for _, hex := range player.AtlanteansTownHexes {
		known[hex] = true
		queue = append(queue, hex)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range gs.atlanteansTownTraversalNeighbors(playerID, current) {
			if known[neighbor] {
				continue
			}
			mapHex := gs.Map.GetHex(neighbor)
			if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
				continue
			}
			known[neighbor] = true
			player.AtlanteansTownHexes = append(player.AtlanteansTownHexes, neighbor)
			queue = append(queue, neighbor)
		}
	}

	totalPower := 0
	for _, hex := range player.AtlanteansTownHexes {
		mapHex := gs.Map.GetHex(hex)
		if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
			continue
		}
		mapHex.PartOfTown = true
		if mapHex.Building.PowerValue > 0 {
			totalPower += mapHex.Building.PowerValue
		} else {
			totalPower += GetPowerValue(mapHex.Building.Type)
		}
	}

	if totalPower >= 7 && !player.AtlanteansTownRewards[7] {
		player.AtlanteansTownRewards[7] = true
		_ = gs.AdvanceShippingLevel(playerID)
	}
	if totalPower >= 10 && !player.AtlanteansTownRewards[10] {
		player.AtlanteansTownRewards[10] = true
		for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
			_, _ = gs.AdvanceCultTrack(playerID, track, 2)
		}
	}
	if totalPower >= 16 && !player.AtlanteansTownRewards[16] {
		player.AtlanteansTownRewards[16] = true
		player.VictoryPoints += 20
	}
}

// CheckAtlanteansStrongholdTown refreshes the persistent Atlanteans starting
// stronghold town after actions that may connect new structures to it.
func (gs *GameState) CheckAtlanteansStrongholdTown(playerID string) {
	gs.updateAtlanteansStrongholdTown(playerID)
}

func (gs *GameState) atlanteansTownTraversalNeighbors(playerID string, current board.Hex) []board.Hex {
	neighbors := make([]board.Hex, 0)
	for _, neighbor := range current.Neighbors() {
		if gs.Map.IsValidHex(neighbor) && gs.Map.IsDirectlyAdjacent(current, neighbor) {
			neighbors = append(neighbors, neighbor)
		}
	}
	for bridgeKey, owner := range gs.Map.Bridges {
		if owner == "" || owner != playerID {
			continue
		}
		if bridgeKey.H1 == current {
			neighbors = append(neighbors, bridgeKey.H2)
		} else if bridgeKey.H2 == current {
			neighbors = append(neighbors, bridgeKey.H1)
		}
	}
	return neighbors
}

// CheckAllTownFormations checks for town formation for all of a player's buildings
// This is useful when a condition changes (e.g. Fire+2 favor tile) that might allow
// existing clusters to form towns
func (gs *GameState) CheckAllTownFormations(playerID string) {
	if player := gs.GetPlayer(playerID); player != nil && player.Faction.GetType() == models.FactionMermaids {
		gs.PendingTownFormations[playerID] = gs.TownFormationChoices(playerID)
		return
	}
	hexes := make([]board.Hex, 0, len(gs.Map.Hexes))
	for hex := range gs.Map.Hexes {
		hexes = append(hexes, hex)
	}
	sort.Slice(hexes, func(i, j int) bool {
		if hexes[i].R != hexes[j].R {
			return hexes[i].R < hexes[j].R
		}
		return hexes[i].Q < hexes[j].Q
	})
	for _, hex := range hexes {
		mapHex := gs.Map.Hexes[hex]
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID && !mapHex.PartOfTown {
			gs.CheckForTownFormation(playerID, hex)
		}
	}
}
