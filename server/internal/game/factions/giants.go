package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Giants faction - Red/Wasteland
// Ability: Always pay exactly 2 Spades to transform terrain (even for Mountains and Desert)
//
//	(A single Spade will be forfeit when gained in Phase III as a Cult bonus)
//
// Stronghold: After building, take an Action token
//
//	Special action (once per Action phase): Get 2 free Spades to transform a reachable space
//	May immediately build a Dwelling on that space by paying its costs
//
// Special: All standard building costs
type Giants struct {
	BaseFaction
	hasStronghold           bool
	freeSpadesUsedThisRound bool // Special action usage tracking
}

func NewGiants() *Giants {
	return &Giants{
		BaseFaction: BaseFaction{
			Type:        models.FactionGiants,
			HomeTerrain: models.TerrainWasteland,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  5, // Standard 5/7 power
				Power2:  7,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold:           false,
		freeSpadesUsedThisRound: false,
	}
}

// GetStartingCultPositions returns Giants starting cult track positions
func (f *Giants) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 0, Earth: 0, Air: 1}
}

// GetTerraformCost overrides the base method
// Giants always pay exactly 2 spades (regardless of terrain distance)
func (f *Giants) GetTerraformCost(distance int) int {
	// Giants always pay 2 spades, regardless of distance
	// This is in terms of workers needed
	// Base terraform cost is 3 workers per spade, reduced by digging level
	workersPerSpade := 3 - f.DiggingLevel
	if workersPerSpade < 1 {
		workersPerSpade = 1
	}
	return 2 * workersPerSpade // Always 2 spades worth of workers
}

// BuildStronghold marks that the stronghold has been built
func (f *Giants) BuildStronghold() {
	f.hasStronghold = true
}

// Income methods (Giants-specific)

func (f *Giants) GetStrongholdIncome() Income {
	// Giants: 4 power
	return Income{Power: 4}
}
