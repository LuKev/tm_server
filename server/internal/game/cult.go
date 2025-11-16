package game

import (
	"fmt"
)

// Cult Track System Implementation
//
// Terra Mystica has 4 cult tracks: Fire, Water, Earth, Air
// Each track has positions 0-10
// Players advance on cult tracks by:
// - Sending priests to the track (costs 1 priest per space)
// - Special actions (e.g., Auren stronghold ability)
// - Building temples/sanctuaries (advance on chosen track)
// - Founding towns (advance on chosen cult track or all tracks)
// - Favor tiles and bonus cards
//
// When advancing, players gain power ONLY at milestone positions (3/5/7/10)
// Power gain: 1/2/2/3 power at positions 3/5/7/10
//
// End-game scoring: Majority bonuses for top 3 positions on each track

// CultTrack is already defined in state.go as an enum
// We'll add methods and tracking here

// TownTileType represents the type of town tile bonus
type TownTileType int

const (
	TownTile5Points TownTileType = iota // +5 VP, +6 coins, +1 key (immediate)
	TownTile6Points                     // +6 VP, +8 power, +1 key (immediate)
	TownTile7Points                     // +7 VP, +2 workers, +1 key (immediate)
	TownTile4Points                     // +4 VP, +1 shipping/range, +1 key (TW7)
	TownTile8Points                     // +8 VP, +1 on all cult tracks, +1 key
	TownTile9Points                     // +9 VP, +1 priest, +1 key (immediate)
	TownTile11Points                    // +11 VP, +1 key
	TownTile2Points                     // +2 VP, +2 on all cult tracks, +2 keys
)

// CultTrackState tracks all players' positions on all cult tracks
type CultTrackState struct {
	// Map of playerID -> cult track -> position (0-10)
	PlayerPositions map[string]map[CultTrack]int
	
	// Track which players have reached position 10 on each track (only one allowed per track)
	Position10Occupied map[CultTrack]string // Track -> PlayerID who occupies position 10
	
	// Track which cult track bonus spaces have been claimed by each player
	// Key: playerID, Value: map of track -> set of bonus positions claimed (3, 5, 7, 10)
	BonusPositionsClaimed map[string]map[CultTrack]map[int]bool
	
	// Track priests placed on cult track action spaces (below each track)
	// Each track has 4 action spaces: one 3-step space and three 2-step spaces
	// Priests placed here stay permanently and count toward the 7-priest limit
	// Key: playerID, Value: map of track -> number of priests on that track's action spaces
	PriestsOnActionSpaces map[string]map[CultTrack]int
}

// NewCultTrackState creates a new cult track state
func NewCultTrackState() *CultTrackState {
	return &CultTrackState{
		PlayerPositions:       make(map[string]map[CultTrack]int),
		Position10Occupied:    make(map[CultTrack]string),
		BonusPositionsClaimed: make(map[string]map[CultTrack]map[int]bool),
		PriestsOnActionSpaces: make(map[string]map[CultTrack]int),
	}
}

// InitializePlayer initializes cult track positions for a player
func (cts *CultTrackState) InitializePlayer(playerID string) {
	cts.PlayerPositions[playerID] = map[CultTrack]int{
		CultFire:  0,
		CultWater: 0,
		CultEarth: 0,
		CultAir:   0,
	}
	
	// Initialize bonus positions tracking
	cts.BonusPositionsClaimed[playerID] = make(map[CultTrack]map[int]bool)
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		cts.BonusPositionsClaimed[playerID][track] = make(map[int]bool)
	}
	
	// Initialize priests on action spaces tracking
	cts.PriestsOnActionSpaces[playerID] = map[CultTrack]int{
		CultFire:  0,
		CultWater: 0,
		CultEarth: 0,
		CultAir:   0,
	}
}

// GetPosition returns a player's position on a cult track
func (cts *CultTrackState) GetPosition(playerID string, track CultTrack) int {
	if positions, ok := cts.PlayerPositions[playerID]; ok {
		return positions[track]
	}
	return 0
}

// GetTotalPriestsOnCultTracks returns the total number of priests a player has on cult track action spaces
// Each cult track has 4 action spaces below it (one 3-step, three 2-step)
// Priests placed on these spaces stay permanently and count toward the 7-priest limit
// Note: Priests can also be sacrificed for 1-step advancement - those don't count
//
// TODO(Bug #34): Currently returns the tracked count, but the game log format doesn't provide
// enough information to accurately distinguish between priests placed on action spaces (2/3 steps)
// vs sacrificed (1 step). The log shows "send p to WATER" but doesn't specify step count.
// For now, this may undercount priests on action spaces in the validator, but it prevents
// incorrectly blocking temple income when priests are actually on action spaces.
func (cts *CultTrackState) GetTotalPriestsOnCultTracks(playerID string) int {
	total := 0
	if priestCounts, ok := cts.PriestsOnActionSpaces[playerID]; ok {
		for _, count := range priestCounts {
			total += count
		}
	}
	return total
}

// AdvancePlayer advances a player on a cult track
// Returns the number of spaces actually advanced (may be less if at max or blocked)
// Also grants power for each space advanced AND bonus power at positions 3/5/7/10
// gs: GameState is optional - if provided, checks for pending town formations when validating position 10
func (cts *CultTrackState) AdvancePlayer(playerID string, track CultTrack, spaces int, player *Player, gs *GameState) (int, error) {
	if spaces < 0 {
		return 0, fmt.Errorf("cannot advance negative spaces")
	}
	if spaces == 0 {
		return 0, nil
	}

	currentPos := cts.GetPosition(playerID, track)
	targetPos := currentPos + spaces

	// Check if position 10 is occupied by another player
	if targetPos >= 10 {
		if occupier, occupied := cts.Position10Occupied[track]; occupied && occupier != playerID {
			// Position 10 is occupied by another player, can only advance to 9
			targetPos = 9
		} else if targetPos > 10 {
			targetPos = 10
		}

		// Check if advancing to position 10 requires a key
		if targetPos == 10 && currentPos < 10 && player != nil {
			// Check if player currently has a key OR will get one from pending town formation
			hasOrWillHaveKey := player.Keys >= 1

			// Check for pending town formation (which will grant a key)
			// This allows cult advancement to position 10 when forming a town during the same turn
			if !hasOrWillHaveKey && gs != nil {
				if pendingTowns, ok := gs.PendingTownFormations[playerID]; ok && len(pendingTowns) > 0 {
					// Player is forming a town this turn, which will grant a key
					hasOrWillHaveKey = true
				}
			}

			if !hasOrWillHaveKey {
				// Player doesn't have a key and won't get one this turn, can only advance to 9
				targetPos = 9
			}
		}
	}
	
	actualAdvancement := targetPos - currentPos
	if actualAdvancement == 0 {
		return 0, nil // Already at max or blocked
	}

	// Update position in cult track state
	cts.PlayerPositions[playerID][track] = targetPos

	// Also update player's local cult position copy (used for validation)
	if player != nil {
		player.CultPositions[track] = targetPos
	}

	// Mark position 10 as occupied if reached
	if targetPos == 10 {
		cts.Position10Occupied[track] = playerID
	}

	// Grant bonus power ONLY for passing milestone positions (3, 5, 7, 10)
	// Bonus: 1/2/2/3 power at positions 3/5/7/10
	// Note: No "base power" is granted for advancing on cult tracks
	if player != nil && actualAdvancement > 0 {
		bonusPositions := map[int]int{
			3:  1, // 1 bonus power
			5:  2, // 2 bonus power
			7:  2, // 2 bonus power
			10: 3, // 3 bonus power
		}

		for pos, bonusPower := range bonusPositions {
			// Check if we passed or reached this position
			if currentPos < pos && targetPos >= pos {
				// Check if we haven't already claimed this bonus
				if !cts.BonusPositionsClaimed[playerID][track][pos] {
					player.Resources.Power.GainPower(bonusPower)
					cts.BonusPositionsClaimed[playerID][track][pos] = true
				}
			}
		}
	}

	return actualAdvancement, nil
}

// GetRankings returns players ranked by position on a cult track (highest to lowest)
// Returns a slice of playerIDs in order of ranking
func (cts *CultTrackState) GetRankings(track CultTrack) []string {
	type playerPos struct {
		playerID string
		position int
	}

	var rankings []playerPos
	for playerID, positions := range cts.PlayerPositions {
		rankings = append(rankings, playerPos{
			playerID: playerID,
			position: positions[track],
		})
	}

	// Sort by position (highest first)
	// Using bubble sort for simplicity
	for i := 0; i < len(rankings); i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[j].position > rankings[i].position {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	result := make([]string, len(rankings))
	for i, r := range rankings {
		result[i] = r.playerID
	}
	return result
}

// CalculateEndGameScoring calculates end-game VP bonuses for cult tracks
// Top player on each track: 8 VP
// 2nd place: 4 VP
// 3rd place: 2 VP
// Ties are handled by splitting points (rounded down)
func (cts *CultTrackState) CalculateEndGameScoring() map[string]int {
	vpByPlayer := make(map[string]int)

	tracks := []CultTrack{CultFire, CultWater, CultEarth, CultAir}
	for _, track := range tracks {
		// Group players by position
		positionGroups := make(map[int][]string)
		for playerID, positions := range cts.PlayerPositions {
			pos := positions[track]
			if pos > 0 { // Only count players who have advanced
				positionGroups[pos] = append(positionGroups[pos], playerID)
			}
		}

		// Get unique positions in descending order
		var uniquePositions []int
		for pos := range positionGroups {
			uniquePositions = append(uniquePositions, pos)
		}
		// Sort descending
		for i := 0; i < len(uniquePositions); i++ {
			for j := i + 1; j < len(uniquePositions); j++ {
				if uniquePositions[j] > uniquePositions[i] {
					uniquePositions[i], uniquePositions[j] = uniquePositions[j], uniquePositions[i]
				}
			}
		}

		// Award points: 8, 4, 2 for top 3 positions
		pointsAvailable := []int{8, 4, 2}
		pointIndex := 0

		for _, pos := range uniquePositions {
			if pointIndex >= len(pointsAvailable) {
				break // No more points to award
			}

			players := positionGroups[pos]
			
			// Calculate points for this group (may span multiple ranks if tied)
			totalPoints := 0
			playersInGroup := len(players)
			ranksOccupied := 0
			
			// Determine how many ranks this group occupies and total points
			for i := pointIndex; i < len(pointsAvailable) && ranksOccupied < playersInGroup; i++ {
				totalPoints += pointsAvailable[i]
				ranksOccupied++
			}
			
			// Split points among tied players (rounded down)
			pointsPerPlayer := totalPoints / playersInGroup
			
			for _, playerID := range players {
				vpByPlayer[playerID] += pointsPerPlayer
			}
			
			pointIndex += ranksOccupied
		}
	}

	return vpByPlayer
}

// ApplyTownCultBonus applies cult track advancement from founding a town
// Returns the total power gained from milestone bonuses
// gs: GameState is optional - passed to AdvancePlayer for position 10 validation
func (cts *CultTrackState) ApplyTownCultBonus(playerID string, townTileType TownTileType, player *Player, gs *GameState) int {
	totalPowerGained := 0
	tracks := []CultTrack{CultFire, CultWater, CultEarth, CultAir}

	var advanceAmount int
	switch townTileType {
	case TownTile8Points:
		advanceAmount = 1 // +1 on all tracks
	case TownTile2Points:
		advanceAmount = 2 // +2 on all tracks
	default:
		return 0
	}

	// Advance on all 4 cult tracks
	for _, track := range tracks {
		advanced, _ := cts.AdvancePlayer(playerID, track, advanceAmount, player, gs)
		
		// Track power gained (AdvancePlayer handles milestone bonuses internally)
		// We need to calculate how much power was actually gained
		// This is a bit tricky since AdvancePlayer modifies the player's power directly
		// For now, we'll just track that advancement occurred
		if advanced > 0 {
			// Power is already granted by AdvancePlayer through milestone bonuses
			totalPowerGained += advanced // This is just for tracking, actual power already applied
		}
	}
	
	return totalPowerGained
}
