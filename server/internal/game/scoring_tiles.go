package game

import (
	"fmt"
	"math/rand"
)

// ScoringTileType represents the type of scoring tile
type ScoringTileType int

const (
	ScoringDwellingWater     ScoringTileType = iota // 2 VP per dwelling | 4 steps Water = 1 priest
	ScoringDwellingFire                             // 2 VP per dwelling | 4 steps Fire = 4 power
	ScoringTradingHouseWater                        // 3 VP per trading house | 4 steps Water = 1 spade
	ScoringTradingHouseAir                          // 3 VP per trading house | 4 steps Air = 1 spade
	ScoringTemplePriest                             // 4 VP per temple | 2 coins per priest sent to cult (SCORE5)
	ScoringStrongholdFire                           // 5 VP per SH/SA | 2 steps Fire = 1 worker
	ScoringStrongholdAir                            // 5 VP per SH/SA | 2 steps Air = 1 worker
	ScoringSpades                                   // 2 VP per spade | 1 step Earth = 1 coin
	ScoringTown                                     // 5 VP per town | 4 steps Earth = 1 spade
	ScoringTileUnknown       ScoringTileType = -1
)

func ScoringTileTypeFromString(s string) ScoringTileType {
	switch s {
	case "Dwelling (Water)":
		return ScoringDwellingWater
	case "Dwelling (Fire)":
		return ScoringDwellingFire
	case "Trading House (Water)":
		return ScoringTradingHouseWater
	case "Trading House (Air)":
		return ScoringTradingHouseAir
	case "Temple (Priest)":
		return ScoringTemplePriest
	case "Stronghold (Fire)":
		return ScoringStrongholdFire
	case "Stronghold (Air)":
		return ScoringStrongholdAir
	case "Spades":
		return ScoringSpades
	case "Town":
		return ScoringTown
	default:
		return ScoringTileUnknown
	}
}

// ActionType for scoring
type ScoringActionType int

const (
	ScoringActionDwelling ScoringActionType = iota
	ScoringActionTradingHouse
	ScoringActionStronghold
	ScoringActionTemple // Only used for SCORE5 (Temple+Priest tile)
	ScoringActionSpades
	ScoringActionTown
	ScoringActionPriestToCult
)

// CultRewardType represents what resource is awarded for cult milestones
type CultRewardType int

const (
	CultRewardPriest CultRewardType = iota
	CultRewardPower
	CultRewardSpade
	CultRewardWorker
	CultRewardCoin
)

// ScoringTile represents a scoring tile with action rewards and cult rewards
type ScoringTile struct {
	Type             ScoringTileType   `json:"type"`
	ActionType       ScoringActionType `json:"actionType"`
	ActionVP         int               `json:"actionVP"`
	CultTrack        CultTrack         `json:"cultTrack"`
	CultThreshold    int               `json:"cultThreshold"`
	CultRewardType   CultRewardType    `json:"cultRewardType"`
	CultRewardAmount int               `json:"cultRewardAmount"`
}

// GetAllScoringTiles returns all 9 scoring tiles
func GetAllScoringTiles() []ScoringTile {
	return []ScoringTile{
		{
			Type:             ScoringDwellingWater,
			ActionType:       ScoringActionDwelling,
			ActionVP:         2,
			CultTrack:        CultWater,
			CultThreshold:    4,
			CultRewardType:   CultRewardPriest,
			CultRewardAmount: 1,
		},
		{
			Type:             ScoringDwellingFire,
			ActionType:       ScoringActionDwelling,
			ActionVP:         2,
			CultTrack:        CultFire,
			CultThreshold:    4,
			CultRewardType:   CultRewardPower,
			CultRewardAmount: 4,
		},
		{
			Type:             ScoringTradingHouseWater,
			ActionType:       ScoringActionTradingHouse,
			ActionVP:         3,
			CultTrack:        CultWater,
			CultThreshold:    4,
			CultRewardType:   CultRewardSpade,
			CultRewardAmount: 1,
		},
		{
			Type:             ScoringTradingHouseAir,
			ActionType:       ScoringActionTradingHouse,
			ActionVP:         3,
			CultTrack:        CultAir,
			CultThreshold:    4,
			CultRewardType:   CultRewardSpade,
			CultRewardAmount: 1,
		},
		{
			Type:             ScoringTemplePriest,
			ActionType:       ScoringActionTemple,
			ActionVP:         4,
			CultTrack:        CultFire, // Not used for this tile
			CultThreshold:    0,        // Special: 2 coins per priest sent to cult
			CultRewardType:   CultRewardCoin,
			CultRewardAmount: 2,
		},
		{
			Type:             ScoringStrongholdFire,
			ActionType:       ScoringActionStronghold,
			ActionVP:         5,
			CultTrack:        CultFire,
			CultThreshold:    2,
			CultRewardType:   CultRewardWorker,
			CultRewardAmount: 1,
		},
		{
			Type:             ScoringStrongholdAir,
			ActionType:       ScoringActionStronghold,
			ActionVP:         5,
			CultTrack:        CultAir,
			CultThreshold:    2,
			CultRewardType:   CultRewardWorker,
			CultRewardAmount: 1,
		},
		{
			Type:             ScoringSpades,
			ActionType:       ScoringActionSpades,
			ActionVP:         2,
			CultTrack:        CultEarth,
			CultThreshold:    1,
			CultRewardType:   CultRewardCoin,
			CultRewardAmount: 1,
		},
		{
			Type:             ScoringTown,
			ActionType:       ScoringActionTown,
			ActionVP:         5,
			CultTrack:        CultEarth,
			CultThreshold:    4,
			CultRewardType:   CultRewardSpade,
			CultRewardAmount: 1,
		},
	}
}

// ScoringTileState tracks the scoring tiles for the game
type ScoringTileState struct {
	Tiles       []ScoringTile  `json:"tiles"`
	PriestsSent map[string]int `json:"priestsSent"`
}

// NewScoringTileState creates a new scoring tile state with random selection
func NewScoringTileState() *ScoringTileState {
	return &ScoringTileState{
		Tiles:       []ScoringTile{},
		PriestsSent: make(map[string]int),
	}
}

// InitializeForGame randomly selects 6 scoring tiles for the game
// Spades tile cannot be in rounds 5 or 6
func (sts *ScoringTileState) InitializeForGame() error {
	allTiles := GetAllScoringTiles()

	// Shuffle tiles
	rand.Shuffle(len(allTiles), func(i, j int) {
		allTiles[i], allTiles[j] = allTiles[j], allTiles[i]
	})

	// Select 6 tiles, ensuring spades is not in rounds 5 or 6
	selected := make([]ScoringTile, 0, 6)

	// If spades tile is selected, ensure it's not in the last 2 rounds
	for _, tile := range allTiles {
		if len(selected) >= 6 {
			break
		}

		// If this is the spades tile and we're filling rounds 5 or 6, skip it
		if tile.Type == ScoringSpades && len(selected) >= 4 {
			continue
		}

		selected = append(selected, tile)
	}

	// If we don't have 6 tiles yet (because we skipped spades), add remaining tiles
	if len(selected) < 6 {
		for _, tile := range allTiles {
			if len(selected) >= 6 {
				break
			}

			// Check if this tile is already selected
			found := false
			for _, s := range selected {
				if s.Type == tile.Type {
					found = true
					break
				}
			}

			if !found {
				selected = append(selected, tile)
			}
		}
	}

	if len(selected) != 6 {
		return fmt.Errorf("failed to select 6 scoring tiles, got %d", len(selected))
	}

	sts.Tiles = selected
	return nil
}

// GetTileForRound returns the scoring tile for a given round (1-6)
func (sts *ScoringTileState) GetTileForRound(round int) *ScoringTile {
	if round < 1 || round > 6 {
		return nil
	}
	return &sts.Tiles[round-1]
}

// RecordPriestSent records that a player sent a priest to a cult track
// This is used for scoring tile #5 (Trading House + Priest)
func (sts *ScoringTileState) RecordPriestSent(playerID string) {
	sts.PriestsSent[playerID]++
}

// GetPriestsSent returns the number of priests a player sent to cult tracks this round
func (sts *ScoringTileState) GetPriestsSent(playerID string) int {
	return sts.PriestsSent[playerID]
}

// ResetPriestsSent resets the priest count for all players (called at end of round)
func (sts *ScoringTileState) ResetPriestsSent() {
	sts.PriestsSent = make(map[string]int)
}

// AwardActionVP awards VP for performing an action based on the current round's scoring tile
func (gs *GameState) AwardActionVP(playerID string, actionType ScoringActionType) {
	if gs.ScoringTiles == nil || len(gs.ScoringTiles.Tiles) == 0 {
		return
	}

	tile := gs.ScoringTiles.GetTileForRound(gs.Round)
	if tile == nil {
		return
	}

	// Check if this action matches the scoring tile
	if tile.ActionType == actionType {
		player := gs.GetPlayer(playerID)
		if player != nil {
			player.VictoryPoints += tile.ActionVP
		}
	}
}

// AwardCultRewards awards cult rewards at the end of the round
func (gs *GameState) AwardCultRewards() {
	if gs.ScoringTiles == nil {
		return
	}

	tile := gs.ScoringTiles.GetTileForRound(gs.Round)
	if tile == nil {
		return
	}

	// Special case: Temple + Priest tile (SCORE5)
	if tile.Type == ScoringTemplePriest {
		for playerID, priestCount := range gs.ScoringTiles.PriestsSent {
			player := gs.GetPlayer(playerID)
			if player != nil {
				coins := priestCount * tile.CultRewardAmount
				player.Resources.Coins += coins
			}
		}
		gs.ScoringTiles.ResetPriestsSent()
		return
	}

	// Regular cult threshold rewards
	// Award rewards based on how many thresholds the player has crossed
	// e.g., "2 steps = 1 worker" means position 8 gives 4 workers (8/2 = 4)
	for playerID, player := range gs.Players {
		position := gs.CultTracks.GetPosition(playerID, tile.CultTrack)

		if tile.CultThreshold == 0 {
			continue // Skip if no threshold (shouldn't happen for regular tiles)
		}

		// Calculate how many times the threshold was crossed
		rewardCount := position / tile.CultThreshold

		if rewardCount > 0 {
			totalReward := rewardCount * tile.CultRewardAmount

			switch tile.CultRewardType {
			case CultRewardPriest:
				gs.GainPriests(playerID, totalReward)
			case CultRewardPower:
				player.Resources.Power.GainPower(totalReward)
			case CultRewardSpade:
				// Spades must be used immediately - track as pending
				// Cult reward spades don't count for VP (unlike BON1 or paid spades)
				// Faction bonuses (e.g., Alchemists +2 power) are granted when spades are USED
				if gs.PendingCultRewardSpades == nil {
					gs.PendingCultRewardSpades = make(map[string]int)
				}
				gs.PendingCultRewardSpades[playerID] += totalReward
			case CultRewardWorker:
				player.Resources.Workers += totalReward
			case CultRewardCoin:
				player.Resources.Coins += totalReward
			}
		}
	}
}
