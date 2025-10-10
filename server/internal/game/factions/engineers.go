package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Engineers faction - Gray/Mountain
// Ability: As an Action, build a Bridge for 2 Workers (any number of times per round)
//          Can still build Bridges via Power action
// Stronghold: After building, each round when passing get 3 VP for each Bridge connecting two of your Structures
// Special: Cheaper building costs across the board
type Engineers struct {
	BaseFaction
	hasStronghold bool
}

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
		Coins:   1, // Cheaper than standard (0)
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

// GetBridgeCost returns the reduced bridge cost for Engineers
// NOTE: Phase 6.2 (Action System) implements bridge building action
func (f *Engineers) GetBridgeCost() Cost {
	return Cost{
		Coins:   0,
		Workers: 2, // Cheaper than standard (3)
		Priests: 0,
		Power:   0,
	}
}

// HasSpecialAbility returns true for bridge building
func (f *Engineers) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityBridgeBuilding
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Engineers) GetStrongholdAbility() string {
	return "Each round when passing: Get 3 VP for each Bridge connecting two of your Structures"
}

// BuildStronghold marks that the stronghold has been built
func (f *Engineers) BuildStronghold() {
	f.hasStronghold = true
}

// HasStronghold returns whether the stronghold has been built
func (f *Engineers) HasStronghold() bool {
	return f.hasStronghold
}

// GetVPPerBridgeOnPass returns the VP bonus per bridge when passing
// NOTE: Phase 8 (Scoring System) tracks VP
// NOTE: Phase 6.2 (Action System) must apply this when Engineers pass
func (f *Engineers) GetVPPerBridgeOnPass() int {
	if f.hasStronghold {
		return 3 // 3 VP per bridge connecting two structures
	}
	return 0
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Engineers) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is passive (VP on passing)
	return nil
}
