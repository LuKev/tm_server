package game

import (
	"sort"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// Final Scoring (End of Game - After Round 6)
// 1. Largest connected area of buildings (18 VP for largest)
// 2. Cult track majority bonuses (8/4/2 VP for top 3 per track)
// 3. Resource conversion (3 coins = 1 VP, 1 worker = 1 VP, 1 priest = 1 VP)

// PlayerFinalScore represents a player's final score breakdown
type PlayerFinalScore struct {
	PlayerID           string `json:"playerId"`
	PlayerName         string `json:"playerName"`
	BaseVP             int    `json:"baseVp"`
	AreaVP             int    `json:"areaVp"`
	FireIceVP          int    `json:"fireIceVp"`
	FireIceMetricValue int    `json:"fireIceMetricValue"`
	CultVP             int    `json:"cultVp"`
	ResourceVP         int    `json:"resourceVp"`
	TotalVP            int    `json:"totalVp"`
	LargestAreaSize    int    `json:"largestAreaSize"`
	TotalResourceValue int    `json:"totalResourceValue"`
}

// CalculateFinalScoring calculates all end-game scoring
// Should be called after round 6 cleanup
func (gs *GameState) CalculateFinalScoring() map[string]*PlayerFinalScore {
	// If final scoring has already been finalized for this end-state snapshot,
	// return it directly to avoid double-counting when callers recompute.
	if gs.Phase == PhaseEnd && gs.FinalScoring != nil {
		return gs.FinalScoring
	}

	scores := make(map[string]*PlayerFinalScore)

	// Initialize scores with base VP
	for playerID, player := range gs.Players {
		name := player.Name
		if name == "" {
			name = playerID
		}
		scores[playerID] = &PlayerFinalScore{
			PlayerID:   playerID,
			PlayerName: name,
			BaseVP:     player.VictoryPoints,
		}
	}

	// 1. Calculate largest connected area bonuses
	gs.calculateAreaBonuses(scores)

	// 2. Calculate optional Fire & Ice final scoring bonus
	gs.calculateFireIceBonuses(scores)

	// 3. Calculate cult track majority bonuses
	gs.calculateCultBonuses(scores)

	// 4. Calculate resource conversion VP
	gs.calculateResourceConversion(scores)

	// Calculate totals
	for _, score := range scores {
		score.TotalVP = score.BaseVP + score.AreaVP + score.FireIceVP + score.CultVP + score.ResourceVP
	}

	return scores
}

// calculateAreaBonuses awards VP for largest connected area
// 1st: 18 VP, 2nd: 12 VP, 3rd: 6 VP
// Ties: VP is split (rounded down) among tied players.
// If there's a tie for 1st, 1st and 2nd place VP are summed and split.
// If there's a tie for 2nd, 2nd and 3rd place VP are summed and split.
func (gs *GameState) calculateAreaBonuses(scores map[string]*PlayerFinalScore) {
	// Calculate largest area for each player using state-level connectivity so
	// fan-faction adjacency rules such as Children river-token networks apply.
	for playerID := range gs.Players {
		largestArea := gs.getLargestConnectedAreaForPlayer(playerID)
		scores[playerID].LargestAreaSize = largestArea
	}

	ranked := gs.getRankedAreas(scores)
	gs.distributeAreaVP(scores, ranked)
}

type playerArea struct {
	playerID string
	size     int
}

type playerMetric struct {
	playerID string
	value    int
}

func (gs *GameState) getRankedAreas(scores map[string]*PlayerFinalScore) []playerArea {
	var ranked []playerArea
	for id, score := range scores {
		ranked = append(ranked, playerArea{id, score.LargestAreaSize})
	}

	// Sort by size descending
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].size > ranked[j].size
	})
	return ranked
}

func (gs *GameState) distributeAreaVP(scores map[string]*PlayerFinalScore, ranked []playerArea) {
	metrics := make([]playerMetric, 0, len(ranked))
	for _, item := range ranked {
		metrics = append(metrics, playerMetric{playerID: item.playerID, value: item.size})
	}
	distributeRankedVP(metrics, func(playerID string, vp int) {
		scores[playerID].AreaVP = vp
	})
}

func distributeRankedVP(ranked []playerMetric, assign func(playerID string, vp int)) {
	// Define awards
	awards := []int{18, 12, 6}
	currentAwardIndex := 0

	// Process groups of tied players
	for i := 0; i < len(ranked); {
		// Find the group of tied players
		groupSize := 1
		for i+groupSize < len(ranked) && ranked[i+groupSize].value == ranked[i].value {
			groupSize++
		}

		// Calculate total VP available for this group based on how many awards they consume
		totalVP := 0
		awardsConsumed := 0
		for k := 0; k < groupSize; k++ {
			if currentAwardIndex+k < len(awards) {
				totalVP += awards[currentAwardIndex+k]
				awardsConsumed++
			}
		}

		// Distribute VP
		vpPerPlayer := 0
		if groupSize > 0 {
			vpPerPlayer = totalVP / groupSize
		}

		for k := 0; k < groupSize; k++ {
			assign(ranked[i+k].playerID, vpPerPlayer)
		}

		// Advance indices
		i += groupSize
		currentAwardIndex += awardsConsumed
	}
}

func sortPlayerMetricsDescending(metrics []playerMetric) {
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].value != metrics[j].value {
			return metrics[i].value > metrics[j].value
		}
		return metrics[i].playerID < metrics[j].playerID
	})
}

func (gs *GameState) calculateFireIceBonuses(scores map[string]*PlayerFinalScore) {
	tile := gs.FireIceFinalScoringTile
	if tile == FireIceFinalScoringTileNone {
		return
	}

	ranked := make([]playerMetric, 0, len(gs.Players))
	for playerID := range gs.Players {
		value := gs.fireIceMetricForPlayer(playerID, tile)
		scores[playerID].FireIceMetricValue = value
		if value > 0 {
			ranked = append(ranked, playerMetric{playerID: playerID, value: value})
		}
	}
	sortPlayerMetricsDescending(ranked)
	distributeRankedVP(ranked, func(playerID string, vp int) {
		scores[playerID].FireIceVP = vp
	})
}

func (gs *GameState) fireIceMetricForPlayer(playerID string, tile FireIceFinalScoringTile) int {
	components := gs.getConnectedStructureClustersForPlayer(playerID)
	if len(components) == 0 {
		return 0
	}

	switch tile {
	case FireIceFinalScoringTileGreatestDistance:
		best := 0
		for _, component := range components {
			if distance := gs.greatestDistanceForComponent(component); distance > best {
				best = distance
			}
		}
		return best
	case FireIceFinalScoringTileStrongholdSanctuary:
		return gs.strongholdSanctuaryDistance(playerID, components)
	case FireIceFinalScoringTileOutposts:
		best := 0
		for _, component := range components {
			count := 0
			for _, hex := range component {
				if gs.isBorderMapHex(hex) {
					count++
				}
			}
			if count > best {
				best = count
			}
		}
		return best
	case FireIceFinalScoringTileSettlements:
		best := 0
		for _, component := range components {
			if settlements := gs.countSettlementsInComponent(playerID, component); settlements > best {
				best = settlements
			}
		}
		return best
	default:
		return 0
	}
}

func (gs *GameState) getConnectedStructureClustersForPlayer(playerID string) [][]board.Hex {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return nil
	}

	buildingHexes := gs.getPlayerBuildingHexes(playerID)
	if len(buildingHexes) == 0 {
		return nil
	}

	visited := make(map[board.Hex]bool, len(buildingHexes))
	components := make([][]board.Hex, 0, len(buildingHexes))

	for _, start := range buildingHexes {
		if visited[start] {
			continue
		}

		component := []board.Hex{}
		queue := []board.Hex{start}
		visited[start] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			component = append(component, current)

			for _, candidate := range buildingHexes {
				if visited[candidate] || candidate == current {
					continue
				}
				if gs.areHexesConnectedForAreaScoring(playerID, player, current, candidate) {
					visited[candidate] = true
					queue = append(queue, candidate)
				}
			}
		}

		components = append(components, component)
	}

	return components
}

func (gs *GameState) greatestDistanceForComponent(component []board.Hex) int {
	if len(component) < 2 {
		return 0
	}

	best := 0
	for i := 0; i < len(component); i++ {
		for j := i + 1; j < len(component); j++ {
			distance := gs.shortestMapDistance(component[i], component[j])
			if distance > best {
				best = distance
			}
		}
	}
	return best
}

func (gs *GameState) strongholdSanctuaryDistance(playerID string, components [][]board.Hex) int {
	var strongholdHex *board.Hex
	var sanctuaryHex *board.Hex

	for _, component := range components {
		var componentStronghold *board.Hex
		var componentSanctuary *board.Hex
		for _, hex := range component {
			mapHex := gs.Map.GetHex(hex)
			if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
				continue
			}
			switch mapHex.Building.Type {
			case models.BuildingStronghold:
				h := hex
				componentStronghold = &h
			case models.BuildingSanctuary:
				h := hex
				componentSanctuary = &h
			}
		}
		if componentStronghold != nil && componentSanctuary != nil {
			strongholdHex = componentStronghold
			sanctuaryHex = componentSanctuary
			break
		}
	}

	if strongholdHex == nil || sanctuaryHex == nil {
		return 0
	}
	return gs.shortestMapDistance(*strongholdHex, *sanctuaryHex)
}

func (gs *GameState) countSettlementsInComponent(playerID string, component []board.Hex) int {
	if len(component) == 0 {
		return 0
	}

	componentSet := make(map[board.Hex]bool, len(component))
	for _, hex := range component {
		componentSet[hex] = true
	}

	visited := make(map[board.Hex]bool, len(component))
	settlements := 0
	for _, start := range component {
		if visited[start] {
			continue
		}
		settlements++
		queue := []board.Hex{start}
		visited[start] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for candidate := range componentSet {
				if visited[candidate] || candidate == current {
					continue
				}
				if gs.areHexesInSameSettlement(playerID, current, candidate) {
					visited[candidate] = true
					queue = append(queue, candidate)
				}
			}
		}
	}

	return settlements
}

func (gs *GameState) areHexesInSameSettlement(playerID string, h1, h2 board.Hex) bool {
	if gs.areHexesDirectlyAdjacentForPlayer(playerID, h1, h2) {
		return true
	}

	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil || player.Faction.GetType() != models.FactionMermaids {
		return false
	}

	for riverHex, mapHex := range gs.Map.Hexes {
		if mapHex == nil || !mapHex.HasTownTile || mapHex.TownTileOwnerPlayerID != playerID || mapHex.Terrain != models.TerrainRiver {
			continue
		}
		if h1.IsDirectlyAdjacent(riverHex) && h2.IsDirectlyAdjacent(riverHex) {
			return true
		}
	}

	return false
}

func (gs *GameState) isBorderMapHex(h board.Hex) bool {
	for _, neighbor := range h.Neighbors() {
		if !gs.Map.IsValidHex(neighbor) {
			return true
		}
	}
	return false
}

func (gs *GameState) shortestMapDistance(from, to board.Hex) int {
	if from == to {
		return 0
	}

	type queueItem struct {
		hex      board.Hex
		distance int
	}

	visited := map[board.Hex]bool{from: true}
	queue := []queueItem{{hex: from, distance: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range current.hex.Neighbors() {
			if visited[neighbor] || !gs.Map.IsValidHex(neighbor) {
				continue
			}
			if neighbor == to {
				return current.distance + 1
			}
			visited[neighbor] = true
			queue = append(queue, queueItem{hex: neighbor, distance: current.distance + 1})
		}
	}

	return 0
}

// calculateCultBonuses awards VP for cult track majorities
// Each track: 8 VP for 1st, 4 VP for 2nd, 2 VP for 3rd
// Ties: VP is split (rounded down)
func (gs *GameState) calculateCultBonuses(scores map[string]*PlayerFinalScore) {
	tracks := []CultTrack{CultFire, CultWater, CultEarth, CultAir}

	for _, track := range tracks {
		positionGroups := gs.getRankedCultPositions(track)
		gs.distributeCultVP(scores, positionGroups)
	}
}

func (gs *GameState) getRankedCultPositions(track CultTrack) [][]string {
	// Get all players and their positions on this track
	type playerPosition struct {
		playerID string
		position int
	}

	positions := []playerPosition{}
	for playerID := range gs.Players {
		pos := gs.CultTracks.GetPosition(playerID, track)
		if pos > 0 { // Only include players who advanced on this track
			positions = append(positions, playerPosition{playerID, pos})
		}
	}

	// Sort by position (descending)
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].position > positions[j].position
	})

	// Group by position to handle ties
	if len(positions) == 0 {
		return nil
	}

	positionGroups := [][]string{}
	currentPos := -1
	var currentGroup []string

	for _, pp := range positions {
		if pp.position != currentPos {
			if len(currentGroup) > 0 {
				positionGroups = append(positionGroups, currentGroup)
			}
			currentGroup = []string{pp.playerID}
			currentPos = pp.position
		} else {
			currentGroup = append(currentGroup, pp.playerID)
		}
	}
	if len(currentGroup) > 0 {
		positionGroups = append(positionGroups, currentGroup)
	}
	return positionGroups
}

func (gs *GameState) distributeCultVP(scores map[string]*PlayerFinalScore, positionGroups [][]string) {
	if len(positionGroups) == 0 {
		return
	}

	// Award VP: 8 for 1st place, 4 for 2nd, 2 for 3rd
	vpAwards := []int{8, 4, 2}
	awardIndex := 0

	for _, group := range positionGroups {
		if awardIndex >= len(vpAwards) {
			break // No more awards to give
		}

		// Calculate VP for this group (may span multiple award levels if tied)
		totalVP := 0
		awardsUsed := 0
		for i := awardIndex; i < len(vpAwards) && awardsUsed < len(group); i++ {
			totalVP += vpAwards[i]
			awardsUsed++
		}

		// Split VP among tied players (rounded down)
		vpPerPlayer := totalVP / len(group)

		for _, playerID := range group {
			scores[playerID].CultVP += vpPerPlayer
		}

		// Move to next award level
		awardIndex += len(group)
	}
}

// calculateResourceConversion converts remaining resources to VP
// Terra Mystica end-game conversion:
// 1. Workers → Coins (1:1)
// 2. Priests → Coins (1:1)
// 3. Power in Bowl 2: burn optimally (2 Bowl 2 → 1 Bowl 3)
// 4. Power in Bowl 3 → Coins (1:1)
// 5. All Coins → VP at 3:1 (or 2:1 for Alchemists)
func (gs *GameState) calculateResourceConversion(scores map[string]*PlayerFinalScore) {
	for playerID, player := range gs.Players {
		// Step 1 & 2: Convert workers and priests to coins
		workerCoins := player.Resources.Workers
		priestCoins := player.Resources.Priests

		// Step 3 & 4: Convert power to coins optimally.
		bowl2Coins := finalBowl2CoinValue(player)
		bowl3Coins := player.Resources.Power.Bowl3 // 1 Bowl 3 → 1 coin
		powerCoins := bowl2Coins + bowl3Coins

		// Step 5: Sum all coins
		totalCoins := player.Resources.Coins + workerCoins + priestCoins + powerCoins

		// Check if player is Alchemists (2 coins = 1 VP instead of 3 coins = 1 VP)
		coinsPerVP := 3
		if player.Faction.GetType() == models.FactionAlchemists {
			coinsPerVP = 2
		}

		// Convert all coins to VP
		scores[playerID].ResourceVP = totalCoins / coinsPerVP

		// Track total resource value (in coins) for tiebreaker
		scores[playerID].TotalResourceValue = totalCoins
	}
}

func finalBowl2CoinValue(player *Player) int {
	bowl2 := player.Resources.Power.Bowl2
	if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
		// Children sacrifice one Bowl II token to move up to two more tokens to
		// Bowl III. BGA also applies this during final resource conversion.
		return (bowl2 * 2) / 3
	}
	return bowl2 / 2
}

// GetWinner returns the player ID of the winner
// Tiebreaker: highest total resource value (coins + workers + priests)
func (gs *GameState) GetWinner(scores map[string]*PlayerFinalScore) string {
	var winner string
	maxVP := -1
	maxResources := -1

	for playerID, score := range scores {
		if score.TotalVP > maxVP {
			winner = playerID
			maxVP = score.TotalVP
			maxResources = score.TotalResourceValue
		} else if score.TotalVP == maxVP {
			// Tiebreaker: highest resource value
			if score.TotalResourceValue > maxResources {
				winner = playerID
				maxResources = score.TotalResourceValue
			}
		}
	}

	return winner
}

// GetRankedPlayers returns players sorted by final score (descending)
func GetRankedPlayers(scores map[string]*PlayerFinalScore) []*PlayerFinalScore {
	ranked := make([]*PlayerFinalScore, 0, len(scores))
	for _, score := range scores {
		ranked = append(ranked, score)
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].TotalVP != ranked[j].TotalVP {
			return ranked[i].TotalVP > ranked[j].TotalVP
		}
		// Tiebreaker: resource value
		return ranked[i].TotalResourceValue > ranked[j].TotalResourceValue
	})

	return ranked
}
