package factions

import (
	"fmt"

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
				Workers: 12, // Swarmlings start with 12 workers (not standard 3)
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

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Swarmlings) GetStrongholdAbility() string {
	return "Special action (once per Action phase): Upgrade Dwelling to Trading House for free (no coins, no workers)"
}

// BuildStronghold marks that the stronghold has been built
func (f *Swarmlings) BuildStronghold() {
	f.hasStronghold = true
}

// CanUseTradingHouseUpgrade checks if the trading house upgrade special action can be used
func (f *Swarmlings) CanUseTradingHouseUpgrade() bool {
	return f.hasStronghold && !f.tradingHouseUpgradeUsedThisRound
}

// UseTradingHouseUpgrade marks the trading house upgrade special action as used
// NOTE: Phase 6.2 (Action System) implements the actual upgrade logic
func (f *Swarmlings) UseTradingHouseUpgrade() error {
	if !f.hasStronghold {
		return fmt.Errorf("must build stronghold before using trading house upgrade")
	}
	
	if f.tradingHouseUpgradeUsedThisRound {
		return fmt.Errorf("trading house upgrade already used this Action phase")
	}
	
	f.tradingHouseUpgradeUsedThisRound = true
	return nil
}

// ResetTradingHouseUpgrade resets the trading house upgrade for a new Action phase
func (f *Swarmlings) ResetTradingHouseUpgrade() {
	f.tradingHouseUpgradeUsedThisRound = false
}

// GetTownFoundingWorkerBonus returns the worker bonus for founding a town
// NOTE: Phase 3.2 (Town Formation) and Phase 6.2 (Action System) apply this
func (f *Swarmlings) GetTownFoundingWorkerBonus() int {
	return 3 // Swarmlings get +3 workers when founding a town
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Swarmlings) ExecuteStrongholdAbility(gameState interface{}) error {
	return f.UseTradingHouseUpgrade()
}
