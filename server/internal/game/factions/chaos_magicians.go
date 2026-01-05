package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// ChaosMagicians faction - Red/Wasteland
// Ability: Start with only 1 Dwelling (place after all other players)
//
//	Get 2 Favor tiles instead of 1 when building Temple or Sanctuary
//
// Stronghold: After building, take an Action token
//
//	Special action (once per Action phase): Take a double-turn (any 2 Actions one after another)
//
// Special: Start with 4 workers, 15 coins (not standard 3 workers, 15 coins)
//
//	Cheap Stronghold (4 workers, 4 coins vs standard 4 workers, 6 coins)
//	Expensive Sanctuary (4 workers, 8 coins vs standard 4 workers, 6 coins)
type ChaosMagicians struct {
	BaseFaction
	hasStronghold           bool
	doubleTurnUsedThisRound bool // Special action usage tracking
}

// NewChaosMagicians creates a new Chaos Magicians faction
func NewChaosMagicians() *ChaosMagicians {
	return &ChaosMagicians{
		BaseFaction: BaseFaction{
			Type:        models.FactionChaosMagicians,
			HomeTerrain: models.TerrainWasteland,
			StartingRes: Resources{
				Coins:   15,
				Workers: 4, // Chaos Magicians start with 4 workers (not standard 3)
				Priests: 0,
				Power1:  5, // Standard 5/7 power
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:           false,
		doubleTurnUsedThisRound: false,
	}
}

// GetStartingCultPositions returns Chaos Magicians starting cult track positions
func (f *ChaosMagicians) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 2, Water: 0, Earth: 0, Air: 0}
}

// GetSanctuaryCost returns the expensive sanctuary cost for Chaos Magicians
func (f *ChaosMagicians) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// GetStrongholdCost returns the cheap stronghold cost for Chaos Magicians
func (f *ChaosMagicians) GetStrongholdCost() Cost {
	return Cost{
		Coins:   4, // Cheaper than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// BuildStronghold marks that the stronghold has been built
func (f *ChaosMagicians) BuildStronghold() {
	f.hasStronghold = true
}

// Income methods (Chaos Magicians-specific)

// GetStrongholdIncome returns the income for the stronghold
func (f *ChaosMagicians) GetStrongholdIncome() Income {
	// Chaos Magicians: 2 workers, NO priest
	return Income{Workers: 2}
}
