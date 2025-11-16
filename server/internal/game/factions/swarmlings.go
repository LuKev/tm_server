package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Swarmlings faction - Blue/Lake
// Ability: Collect 3 additional Workers when founding a Town
// Stronghold: After building, take an Action token
//             Special action (once per Action phase): Upgrade Dwelling to Trading House for free (no coins, no workers)
// Special: All buildings are more expensive
//          Start with 12 workers and 20 coins (not standard 3 workers and 15 coins)
type Swarmlings struct {
	BaseFaction
	hasStronghold                  bool
	tradingHouseUpgradeUsedThisRound bool // Special action usage tracking
}

func NewSwarmlings() *Swarmlings {
	return &Swarmlings{
		BaseFaction: BaseFaction{
			Type:        models.FactionSwarmlings,
			HomeTerrain: models.TerrainLake,
			StartingRes: Resources{
				Coins:   20, // Swarmlings start with 20 coins (not standard 15)
				Workers: 8,  // Swarmlings start with 8 workers (not standard 3)
				Priests: 0,
				Power1:  3, // Swarmlings start with 3/9 power (not standard 5/7)
				Power2:  9,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:                  false,
		tradingHouseUpgradeUsedThisRound: false,
	}
}

// GetStartingCultPositions returns Swarmlings starting cult track positions
func (f *Swarmlings) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 1, Earth: 1, Air: 1}
}

// GetDwellingCost returns the expensive dwelling cost for Swarmlings
func (f *Swarmlings) GetDwellingCost() Cost {
	return Cost{
		Coins:   3, // More expensive than standard (2)
		Workers: 2, // More expensive than standard (1)
		Priests: 0,
		Power:   0,
	}
}

// GetTradingHouseCost returns the expensive trading house cost for Swarmlings
func (f *Swarmlings) GetTradingHouseCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 3, // More expensive than standard (2)
		Priests: 0,
		Power:   0,
	}
}

// GetTempleCost returns the expensive temple cost for Swarmlings
func (f *Swarmlings) GetTempleCost() Cost {
	return Cost{
		Coins:   6, // More expensive than standard (5)
		Workers: 3, // More expensive than standard (2)
		Priests: 0,
		Power:   0,
	}
}

// GetSanctuaryCost returns the expensive sanctuary cost for Swarmlings
func (f *Swarmlings) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 5, // More expensive than standard (4)
		Priests: 0,
		Power:   0,
	}
}

// GetStrongholdCost returns the expensive stronghold cost for Swarmlings
func (f *Swarmlings) GetStrongholdCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 5, // More expensive than standard (4)
		Priests: 0,
		Power:   0,
	}
}

// HasSpecialAbility returns true for cheap dwellings (they get workers from towns)
func (f *Swarmlings) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityCheapDwellings
}

// BuildStronghold marks that the stronghold has been built
func (f *Swarmlings) BuildStronghold() {
	f.hasStronghold = true
}

// GetTownFoundingWorkerBonus returns the worker bonus for founding a town
// NOTE: Phase 3.2 (Town Formation) and Phase 6.2 (Action System) apply this
func (f *Swarmlings) GetTownFoundingWorkerBonus() int {
	return 3 // Swarmlings get +3 workers when founding a town
}

// Income methods (Swarmlings-specific)

func (f *Swarmlings) GetBaseFactionIncome() Income {
	// Swarmlings: 2 workers base income
	return Income{Workers: 2}
}

func (f *Swarmlings) GetTradingHouseIncome(tradingHouseCount int) Income {
	// Swarmlings: 1st-3rd: 2c+2pw, 4th: 3c+2pw
	income := Income{}
	for i := 1; i <= tradingHouseCount && i <= 4; i++ {
		if i <= 3 {
			income.Coins += 2
			income.Power += 2
		} else {
			income.Coins += 3
			income.Power += 2
		}
	}
	return income
}

func (f *Swarmlings) GetSanctuaryIncome() Income {
	// Swarmlings: 2 priests per sanctuary
	return Income{Priests: 2}
}

func (f *Swarmlings) GetStrongholdIncome() Income {
	// Swarmlings: 4 power, NO priest
	return Income{Power: 4}
}
