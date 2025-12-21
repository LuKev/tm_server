package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Auren faction - Green/Forest
// Ability: None (no passive ability)
// Stronghold: After building, immediately get 1 Favor tile (once)
//
//	Special action (once per Action phase): Advance 2 spaces on a Cult track
//	(only advancing to space 10 if you have a key)
//
// Special: Sanctuary costs 4 workers/8 coins (more expensive than standard)
type Auren struct {
	BaseFaction
	hasStronghold bool
}

func NewAuren() *Auren {
	return &Auren{
		BaseFaction: BaseFaction{
			Type:        models.FactionAuren,
			HomeTerrain: models.TerrainForest,
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
		hasStronghold: false,
	}
}

// GetStartingCultPositions returns Auren starting cult track positions
func (f *Auren) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 0, Water: 1, Earth: 0, Air: 1}
}

// GetSanctuaryCost returns the Auren's expensive sanctuary cost
func (f *Auren) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   8, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// BuildStronghold marks that the stronghold has been built
// Returns true to indicate the player should receive a favor tile
func (f *Auren) BuildStronghold() bool {
	f.hasStronghold = true
	return true // Grant favor tile
}
