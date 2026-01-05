package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Darklings faction - Black/Swamp
// Ability: Pay Priests (instead of Workers) when transforming Terrain
//
//	Pay 1 Priest and get 2 VP for each step of transformation
//	IMPORTANT: Darklings can NEVER upgrade digging (cannot use workers for spades)
//
// Stronghold: After building, immediately and only once trade up to 3 Workers for 1 Priest each
// Special: Sanctuary costs 4 workers/10 coins (more expensive than standard)
//
//	Start with 1 Priest, 1 Worker, 15 Coins
type Darklings struct {
	BaseFaction
	hasStronghold bool
}

// NewDarklings creates a new Darklings faction
func NewDarklings() *Darklings {
	return &Darklings{
		BaseFaction: BaseFaction{
			Type:        models.FactionDarklings,
			HomeTerrain: models.TerrainSwamp,
			StartingRes: Resources{
				Coins:   15,
				Workers: 1, // Only 1 worker (instead of 3)
				Priests: 1, // Start with 1 priest (instead of 0)
				Power1:  5,
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold: false,
	}
}

// GetStartingCultPositions returns Darklings starting cult track positions
func (f *Darklings) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 0, Water: 1, Earth: 1, Air: 0}
}

// GetSanctuaryCost returns the Darklings' expensive sanctuary cost
func (f *Darklings) GetSanctuaryCost() Cost {
	return Cost{
		Coins:   10, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// BuildStronghold marks that the stronghold has been built
func (f *Darklings) BuildStronghold() {
	f.hasStronghold = true
}

// CanUsePriestOrdination checks if the priest ordination can be used
func (f *Darklings) CanUsePriestOrdination() bool {
	return f.hasStronghold
}

// UsePriestOrdination trades workers for priests (0-3 workers)
// Returns the number of priests gained
// NOTE: The pending ordination system ensures this is only used once
func (f *Darklings) UsePriestOrdination(workersToTrade int) (int, error) {
	if workersToTrade < 0 || workersToTrade > 3 {
		return 0, fmt.Errorf("can only trade 0-3 workers")
	}

	// 1 Worker = 1 Priest
	return workersToTrade, nil
}

// GetTerraformCost overrides the base method
// Darklings don't use workers for terraform, they use priests
func (f *Darklings) GetTerraformCost(distance int) int {
	// Return 0 workers (Darklings use priests instead)
	return 0
}

// GetDiggingCost overrides the base method
// Darklings can NEVER upgrade digging (they use priests, not workers for spades)
func (f *Darklings) GetDiggingCost(currentLevel int) Cost {
	// Darklings cannot upgrade digging
	return Cost{
		Coins:   0,
		Workers: 0,
		Priests: 0,
		Power:   0,
	}
}

// Income methods (Darklings-specific)

// GetSanctuaryIncome returns the income for the sanctuary
func (f *Darklings) GetSanctuaryIncome() Income {
	// Darklings: 2 priests per sanctuary
	return Income{Priests: 2}
}
