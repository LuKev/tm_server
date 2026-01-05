package game

import (
	"sort"

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
	CultVP             int    `json:"cultVp"`
	ResourceVP         int    `json:"resourceVp"`
	TotalVP            int    `json:"totalVp"`
	LargestAreaSize    int    `json:"largestAreaSize"`
	TotalResourceValue int    `json:"totalResourceValue"`
}

// CalculateFinalScoring calculates all end-game scoring
// Should be called after round 6 cleanup
func (gs *GameState) CalculateFinalScoring() map[string]*PlayerFinalScore {
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

	// 2. Calculate cult track majority bonuses
	gs.calculateCultBonuses(scores)

	// 3. Calculate resource conversion VP
	gs.calculateResourceConversion(scores)

	// Calculate totals
	for _, score := range scores {
		score.TotalVP = score.BaseVP + score.AreaVP + score.CultVP + score.ResourceVP
	}

	return scores
}

// calculateAreaBonuses awards VP for largest connected area
// 1st: 18 VP, 2nd: 12 VP, 3rd: 6 VP
// Ties: VP is split (rounded down) among tied players.
// If there's a tie for 1st, 1st and 2nd place VP are summed and split.
// If there's a tie for 2nd, 2nd and 3rd place VP are summed and split.
func (gs *GameState) calculateAreaBonuses(scores map[string]*PlayerFinalScore) {
	// Calculate largest area for each player (faction-specific adjacency + shipping)
	for playerID, player := range gs.Players {
		largestArea := gs.Map.GetLargestConnectedArea(playerID, player.Faction, player.ShippingLevel)
		scores[playerID].LargestAreaSize = largestArea
	}

	// Group players by area size
	type playerArea struct {
		playerID string
		size     int
	}
	var ranked []playerArea
	for id, score := range scores {
		ranked = append(ranked, playerArea{id, score.LargestAreaSize})
	}

	// Sort by size descending
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].size > ranked[j].size
	})

	// Define awards
	awards := []int{18, 12, 6}
	currentAwardIndex := 0

	// Process groups of tied players
	for i := 0; i < len(ranked); {
		// Find the group of tied players
		groupSize := 1
		for i+groupSize < len(ranked) && ranked[i+groupSize].size == ranked[i].size {
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
			playerID := ranked[i+k].playerID
			scores[playerID].AreaVP = vpPerPlayer
		}

		// Advance indices
		i += groupSize
		currentAwardIndex += awardsConsumed
	}
}

// calculateCultBonuses awards VP for cult track majorities
// Each track: 8 VP for 1st, 4 VP for 2nd, 2 VP for 3rd
// Ties: VP is split (rounded down)
func (gs *GameState) calculateCultBonuses(scores map[string]*PlayerFinalScore) {
	tracks := []CultTrack{CultFire, CultWater, CultEarth, CultAir}

	for _, track := range tracks {
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

		// Award VP for top 3 positions (handling ties)
		if len(positions) == 0 {
			continue
		}

		// Group by position to handle ties
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
}

// calculateResourceConversion converts remaining resources to VP
// 3 coins = 1 VP (or 2 coins = 1 VP for Alchemists), 1 worker = 1 VP, 1 priest = 1 VP
// Power is automatically converted optimally to coins first:
//   - Burn Bowl 2 power to move to Bowl 3 (2 Bowl 2 → 1 Bowl 3)
//   - Convert Bowl 3 to coins (1 Bowl 3 → 1 coin)
//   - Result: coins from power = Bowl 3 + floor(Bowl 2 / 2)
//
// Also tracks total resource value for tiebreaker
func (gs *GameState) calculateResourceConversion(scores map[string]*PlayerFinalScore) {
	for playerID, player := range gs.Players {
		// First, convert power to coins optimally
		// Players can: 1 power in Bowl 3 → 1 coin
		// Players can: burn 1 power in Bowl 2 to move 1 power from Bowl 2 to Bowl 3
		// Optimal: Convert all Bowl 2 power to Bowl 3 (2 Bowl 2 → 1 Bowl 3), then convert Bowl 3 to coins
		// Result: coins = Bowl 3 + floor(Bowl 2 / 2)
		powerCoins := player.Resources.Power.Bowl3 + (player.Resources.Power.Bowl2 / 2)
		totalCoins := player.Resources.Coins + powerCoins

		// Check if player is Alchemists (2 coins = 1 VP instead of 3 coins = 1 VP)
		coinsPerVP := 3
		if player.Faction.GetType() == models.FactionAlchemists {
			coinsPerVP = 2
		}

		// Convert coins to VP
		coinVP := totalCoins / coinsPerVP

		// Convert workers (1 worker = 1 VP)
		workerVP := player.Resources.Workers

		// Convert priests (1 priest = 1 VP)
		priestVP := player.Resources.Priests

		scores[playerID].ResourceVP = coinVP + workerVP + priestVP

		// Track total resource value for tiebreaker
		// Tiebreaker uses: coins + power + workers + priests (not converted)
		scores[playerID].TotalResourceValue = totalCoins +
			player.Resources.Workers +
			player.Resources.Priests
	}
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
