package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Nomads faction - Yellow/Desert
// Ability: Start with 3 Dwellings instead of 2
//
//	Place third Dwelling after all players have placed their second ones (but before Chaos Magicians)
//
// Stronghold: After building, take an Action token
//
//	Special action (once per Action phase): Transform a Terrain space directly adjacent to one of your Structures
//	into your Home terrain (Sandstorm). May immediately build a Dwelling on that space by paying its costs.
//	(Not applicable past a River space or Bridge. Sandstorm is not considered a Spade.)
//
// Special: Start with 2 workers, 15 coins (not standard 3 workers, 15 coins)
type Nomads struct {
	BaseFaction
	hasStronghold          bool
	sandstormUsedThisRound bool // Special action usage tracking
}

// NewNomads creates a new Nomads faction
func NewNomads() *Nomads {
	return &Nomads{
		BaseFaction: BaseFaction{
			Type:        models.FactionNomads,
			HomeTerrain: models.TerrainDesert,
			StartingRes: Resources{
				Coins:   15,
				Workers: 2, // Nomads start with 2 workers (not standard 3)
				Priests: 0,
				Power1:  5, // Standard 5/7 power
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:          false,
		sandstormUsedThisRound: false,
	}
}

// GetStartingCultPositions returns Nomads starting cult track positions
func (f *Nomads) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 0, Earth: 1, Air: 0}
}

// BuildStronghold marks that the stronghold has been built
func (f *Nomads) BuildStronghold() {
	f.hasStronghold = true
}

// GetStrongholdCost returns the expensive stronghold cost for Nomads (8 coins, 4 workers)
func (f *Nomads) GetStrongholdCost() Cost {
	return Cost{
		Coins:   8,
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// Income methods (Nomads-specific)

// GetTradingHouseIncome returns the income for trading houses
func (f *Nomads) GetTradingHouseIncome(tradingHouseCount int) Income {
	// Nomads: 1st-2nd: 2c+1pw, 3rd: 3c+1pw, 4th: 4c+1pw
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
