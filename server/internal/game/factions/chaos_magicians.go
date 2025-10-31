package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// ChaosMagicians faction - Red/Wasteland
// Ability: Start with only 1 Dwelling (place after all other players)
//          Get 2 Favor tiles instead of 1 when building Temple or Sanctuary
// Stronghold: After building, take an Action token
//             Special action (once per Action phase): Take a double-turn (any 2 Actions one after another)
// Special: Start with 4 workers, 15 coins (not standard 3 workers, 15 coins)
//          Cheap Stronghold (4 workers, 4 coins vs standard 4 workers, 6 coins)
//          Expensive Sanctuary (4 workers, 8 coins vs standard 4 workers, 6 coins)
type ChaosMagicians struct {
	BaseFaction
	hasStronghold           bool
	doubleTurnUsedThisRound bool // Special action usage tracking
}

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

// HasSpecialAbility returns true for favor transform (double favor tiles)
func (f *ChaosMagicians) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityFavorTransform
}

// BuildStronghold marks that the stronghold has been built
func (f *ChaosMagicians) BuildStronghold() {
	f.hasStronghold = true
}

// GetFavorTilesForTemple returns how many favor tiles to get when building Temple
// NOTE: Phase 7.2 (Favor Tiles) uses this
func (f *ChaosMagicians) GetFavorTilesForTemple() int {
	return 2 // Chaos Magicians get 2 favor tiles (not standard 1)
}

// GetFavorTilesForSanctuary returns how many favor tiles to get when building Sanctuary
// NOTE: Phase 7.2 (Favor Tiles) uses this
func (f *ChaosMagicians) GetFavorTilesForSanctuary() int {
	return 2 // Chaos Magicians get 2 favor tiles (not standard 1)
}

// StartsWithOneDwelling returns true - Chaos Magicians start with only 1 dwelling
// NOTE: Game setup (Phase 1) uses this
func (f *ChaosMagicians) StartsWithOneDwelling() bool {
	return true
}

// PlacesDwellingLast returns true - Chaos Magicians place dwelling after all other players
// NOTE: Game setup (Phase 1) uses this
func (f *ChaosMagicians) PlacesDwellingLast() bool {
	return true
}

// Income methods (Chaos Magicians-specific)

func (f *ChaosMagicians) GetStrongholdIncome() Income {
	// Chaos Magicians: 2 workers, NO priest
	return Income{Workers: 2}
}
