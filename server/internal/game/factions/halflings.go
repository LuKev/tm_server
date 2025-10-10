package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Halflings faction - Brown/Plains
// Ability: Get 1 additional VP for each Spade throughout the game
// Stronghold: After building, immediately and only once get 3 Spades to apply on Terrain spaces
//             May build a Dwelling on exactly one of these spaces by paying its costs
// Special: Digging upgrade costs 2 workers, 1 coin, 1 priest (cheaper than standard 2 workers, 5 coins, 1 priest)
//          Stronghold costs 4 workers, 8 coins (more expensive than standard 4 workers, 6 coins)
type Halflings struct {
	BaseFaction
	hasStronghold              bool
	hasUsedStrongholdSpades    bool // One-time 3 spades bonus
}

func NewHalflings() *Halflings {
	return &Halflings{
		BaseFaction: BaseFaction{
			Type:        models.FactionHalflings,
			HomeTerrain: models.TerrainPlains,
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
		hasUsedStrongholdSpades: false,
	}
}

// GetDiggingCost returns the cheaper digging upgrade cost for Halflings
func (f *Halflings) GetDiggingCost(currentLevel int) Cost {
	// Halflings pay less for digging upgrades
	// Standard: 5 workers, 2 coins, 1 priest
	// Halflings: 2 workers, 1 coin, 1 priest
	return Cost{
		Coins:   1, // Cheaper than standard (5)
		Workers: 2,
		Priests: 1,
		Power:   0,
	}
}

// GetStrongholdCost returns the expensive stronghold cost for Halflings
func (f *Halflings) GetStrongholdCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// HasSpecialAbility returns true for spade efficiency
func (f *Halflings) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilitySpadeEfficiency
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Halflings) GetStrongholdAbility() string {
	return "Immediately and only once: Get 3 Spades to apply on Terrain spaces. May build a Dwelling on exactly one of these spaces by paying its costs"
}

// BuildStronghold marks that the stronghold has been built
func (f *Halflings) BuildStronghold() {
	f.hasStronghold = true
}

// CanUseStrongholdSpades checks if the stronghold spades can be used
func (f *Halflings) CanUseStrongholdSpades() bool {
	return f.hasStronghold && !f.hasUsedStrongholdSpades
}

// UseStrongholdSpades marks the stronghold spades as used
// Returns the number of spades granted (3)
// NOTE: Phase 6.2 (Action System) handles applying spades and optional dwelling placement
func (f *Halflings) UseStrongholdSpades() int {
	if !f.hasStronghold {
		return 0
	}
	
	if f.hasUsedStrongholdSpades {
		return 0
	}
	
	f.hasUsedStrongholdSpades = true
	return 3 // Grant 3 spades
}

// GetVPPerSpade returns the VP bonus for each spade
// NOTE: Phase 8 (Scoring System) tracks VP
// NOTE: Phase 6.2 (Action System) must apply this bonus whenever Halflings get spades
func (f *Halflings) GetVPPerSpade() int {
	return 1 // Halflings get +1 VP per spade
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Halflings) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is the one-time 3 spades
	// This is handled by UseStrongholdSpades()
	return nil
}
