package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Halflings faction - Brown/Plains
// Ability: Get 1 additional VP for each Spade throughout the game
// Stronghold: After building, immediately and only once get 3 Spades to apply on Terrain spaces
//
//	May build a Dwelling on exactly one of these spaces by paying its costs
//
// Special: Digging upgrade costs 2 workers, 1 coin, 1 priest (cheaper than standard 2 workers, 5 coins, 1 priest)
//
//	Stronghold costs 4 workers, 8 coins (more expensive than standard 4 workers, 6 coins)
type Halflings struct {
	BaseFaction
	hasStronghold bool
}

// NewHalflings creates a new Halflings faction
func NewHalflings() *Halflings {
	return &Halflings{
		BaseFaction: BaseFaction{
			Type:        models.FactionHalflings,
			HomeTerrain: models.TerrainPlains,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  3, // Halflings start with 3/9 power (not standard 5/7)
				Power2:  9,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold: false,
	}
}

// GetStartingCultPositions returns Halflings starting cult track positions
func (f *Halflings) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 0, Water: 0, Earth: 1, Air: 1}
}

// GetDiggingCost returns the cheaper digging upgrade cost for Halflings
func (f *Halflings) GetDiggingCost(currentLevel int) Cost {
	// Halflings pay less for digging upgrades
	// Standard: 2 workers, 5 coins, 1 priest
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

// BuildStronghold marks that the stronghold has been built
func (f *Halflings) BuildStronghold() {
	f.hasStronghold = true
}

// CanUseStrongholdSpades checks if the stronghold spades can be used
func (f *Halflings) CanUseStrongholdSpades() bool {
	return f.hasStronghold
}

// UseStrongholdSpades returns the number of spades granted (3)
// NOTE: The pending spades system ensures this is only used once
func (f *Halflings) UseStrongholdSpades() int {
	return 3 // Grant 3 spades
}
