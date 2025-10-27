package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Alchemists faction - Black/Swamp
// Ability: Philosopher's Stone - Trade 1 VP for 1 Coin, or 2 Coins for 1 VP anytime, any number of times
// Stronghold: After building, immediately gain 12 Power (once)
//             From now on, gain 2 Power for each Spade throughout remainder of game
type Alchemists struct {
	BaseFaction
	hasStronghold           bool
	hasReceivedStrongholdPower bool // One-time 12 power bonus
}

func NewAlchemists() *Alchemists {
	return &Alchemists{
		BaseFaction: BaseFaction{
			Type:        models.FactionAlchemists,
			HomeTerrain: models.TerrainSwamp,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  5,
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:           false,
		hasReceivedStrongholdPower: false,
	}
}

// GetStartingCultPositions returns Alchemists starting cult track positions
func (f *Alchemists) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 1, Earth: 0, Air: 0}
}

// HasSpecialAbility returns true for conversion efficiency
func (f *Alchemists) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityConversionEfficiency
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Alchemists) GetStrongholdAbility() string {
	return "On building: Gain 12 Power (once). From now on: Gain 2 Power for each Spade throughout remainder of game"
}

// BuildStronghold marks that the stronghold has been built
// Returns the one-time power bonus (12 power to bowl 1)
// NOTE: Power system implementation in Phase 5.1
func (f *Alchemists) BuildStronghold() int {
	f.hasStronghold = true
	
	// Return one-time power bonus
	if !f.hasReceivedStrongholdPower {
		f.hasReceivedStrongholdPower = true
		return 12 // Power added to bowl 1 (Phase 5.1: Power System)
	}
	return 0
}

// GetPowerPerSpade returns how much power to gain per spade
// NOTE: This bonus must be applied in Phase 6.2 (Action System) when spades are gained
func (f *Alchemists) GetPowerPerSpade() int {
	if f.hasStronghold {
		return 2 // After stronghold, gain 2 power per spade (Phase 5.1: Power System)
	}
	return 0 // Before stronghold, no bonus power
}

// ConvertVPToCoins trades Victory Points for Coins (Philosopher's Stone)
// Rate: 1 VP -> 1 Coin
// NOTE: Full validation and VP tracking will be in Phase 6.2 (Action System)
// NOTE: Phase 8 (Scoring System) tracks VP
func (f *Alchemists) ConvertVPToCoins(vp int) (coins int, err error) {
	if vp < 1 {
		return 0, fmt.Errorf("must convert at least 1 VP")
	}
	// 1 VP = 1 Coin
	return vp, nil
}

// ConvertCoinsToVP trades Coins for Victory Points (Philosopher's Stone)
// Rate: 2 Coins -> 1 VP
// NOTE: Full validation and VP tracking will be in Phase 6.2 (Action System)
// NOTE: Phase 8 (Scoring System) tracks VP
func (f *Alchemists) ConvertCoinsToVP(coins int) (vp int, err error) {
	if coins < 2 {
		return 0, fmt.Errorf("must convert at least 2 coins")
	}
	if coins%2 != 0 {
		return 0, fmt.Errorf("must convert an even number of coins (2 coins = 1 VP)")
	}
	// 2 Coins = 1 VP
	return coins / 2, nil
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Alchemists) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is passive (power per spade)
	// The one-time power bonus is handled in BuildStronghold()
	return nil
}
