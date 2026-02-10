package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Alchemists faction - Black/Swamp
// Ability: Philosopher's Stone - Trade 1 VP for 1 Coin, or 2 Coins for 1 VP anytime, any number of times
// Stronghold: After building, immediately gain 12 Power (once)
//
//	From now on, gain 2 Power for each Spade throughout remainder of game
type Alchemists struct {
	BaseFaction
}

// NewAlchemists creates a new Alchemists faction
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
	}
}

// GetStartingCultPositions returns Alchemists starting cult track positions
func (f *Alchemists) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 1, Earth: 0, Air: 0}
}

// BuildStronghold marks that the stronghold has been built
// Returns the one-time power bonus (12 power gained via GainPower)
func (f *Alchemists) BuildStronghold() int {
	return 12 // 12 power gained via GainPower (cycles through bowls)
}

// ConvertVPToCoins trades Victory Points for Coins (Philosopher's Stone)
// Rate: 1 VP -> 1 Coin
func (f *Alchemists) ConvertVPToCoins(vp int) (coins int, err error) {
	if vp < 1 {
		return 0, fmt.Errorf("must convert at least 1 VP")
	}
	// 1 VP = 1 Coin
	return vp, nil
}

// ConvertCoinsToVP trades Coins for Victory Points (Philosopher's Stone)
// Rate: 2 Coins -> 1 VP
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

// Income methods (Alchemists-specific)

// GetTradingHouseIncome returns the income for trading houses
func (f *Alchemists) GetTradingHouseIncome(tradingHouseCount int) Income {
	// Alchemists: 1st-2nd: 2c+1pw, 3rd: 3c+1pw, 4th: 4c+1pw
	income := Income{}
	for i := 1; i <= tradingHouseCount && i <= 4; i++ {
		switch i {
		case 1, 2:
			income.Coins += 2
			income.Power++
		case 3:
			income.Coins += 3
			income.Power++
		case 4:
			income.Coins += 4
			income.Power++
		}
	}
	return income
}

// GetStrongholdIncome returns the income for the stronghold
func (f *Alchemists) GetStrongholdIncome() Income {
	// Alchemists: 6 coins, NO priest
	return Income{Coins: 6}
}
