package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Engineers faction - Gray/Mountain
// Ability: As an Action, build a Bridge for 2 Workers (any number of times per round)
//
//	Can still build Bridges via Power action
//
// Stronghold: After building, each round when passing get 3 VP for each Bridge connecting two of your Structures
// Special: Cheaper building costs across the board
type Engineers struct {
	BaseFaction
	hasStronghold bool
}

// NewEngineers creates a new Engineers faction
func NewEngineers() *Engineers {
	return &Engineers{
		BaseFaction: BaseFaction{
			Type:        models.FactionEngineers,
			HomeTerrain: models.TerrainMountain,
			StartingRes: Resources{
				Coins:   10, // Engineers start with 10 coins (not standard 15)
				Workers: 2,  // Engineers start with 2 workers (not standard 3)
				Priests: 0,
				Power1:  3, // Engineers start with 3/9 power (not standard 5/7)
				Power2:  9,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold: false,
	}
}

// GetDwellingCost returns the cheaper dwelling cost for Engineers
func (f *Engineers) GetDwellingCost() Cost {
	return Cost{
		Coins:   1, // Cheaper than standard (2)
		Workers: 1,
		Priests: 0,
		Power:   0,
	}
}

// GetTradingHouseCost returns the cheaper trading house cost for Engineers
func (f *Engineers) GetTradingHouseCost() Cost {
	return Cost{
		Coins:   4, // Cheaper than standard (6)
		Workers: 1, // Cheaper than standard (2)
		Priests: 0,
		Power:   0,
	}
}

// GetTempleCost returns the cheaper temple cost for Engineers
func (f *Engineers) GetTempleCost() Cost {
	return Cost{
		Coins:   4, // Cheaper than standard (5)
		Workers: 1, // Cheaper than standard (2)
		Priests: 0,
		Power:   0,
	}
}

// GetSanctuaryCost returns the cheaper sanctuary cost for Engineers
func (f *Engineers) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   6, // Same as standard
		Workers: 3, // Cheaper than standard (4)
		Priests: 0,
		Power:   0,
	}
}

// GetStrongholdCost returns the cheaper stronghold cost for Engineers
func (f *Engineers) GetStrongholdCost() Cost {
	return Cost{
		Coins:   6, // Same as standard
		Workers: 3, // Cheaper than standard (4)
		Priests: 0,
		Power:   0,
	}
}

// BuildStronghold marks that the stronghold has been built
func (f *Engineers) BuildStronghold() {
	f.hasStronghold = true
}

// Income methods (Engineers-specific)

// GetBaseFactionIncome returns the base income for the faction
func (f *Engineers) GetBaseFactionIncome() Income {
	// Engineers: 0 base income
	return Income{}
}

// GetDwellingIncome returns the income for dwellings
func (f *Engineers) GetDwellingIncome(dwellingCount int) Income {
	// Engineers: dwellings 1, 2, 4, 5, 7, 8 give income (skip 3rd and 6th)
	workers := 0
	for i := 1; i <= dwellingCount && i <= 8; i++ {
		if i != 3 && i != 6 {
			workers++
		}
	}
	return Income{Workers: workers}
}

// GetTempleIncome returns the income for temples
func (f *Engineers) GetTempleIncome(templeCount int) Income {
	// Engineers: 1st and 3rd temples give 1 priest, 2nd temple gives 5 power
	income := Income{}
	for i := 1; i <= templeCount; i++ {
		if i == 2 {
			income.Power += 5 // 2nd temple: 5 power, no priest
		} else {
			income.Priests++ // 1st and 3rd temples: 1 priest, no power
		}
	}
	return income
}

// HasStronghold returns whether the stronghold has been built
func (f *Engineers) HasStronghold() bool {
	return f.hasStronghold
}
