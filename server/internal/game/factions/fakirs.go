package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Fakirs faction - Yellow/Desert
// Ability: When taking "Transform and Build" action, may skip one Terrain or River space by paying 1 more Priest (Carpet Flight)
//
//	Get 4 VP each time doing Carpet Flight
//	In final Area scoring, Structures reachable via Carpet Flight are considered connected
//
// Stronghold: After building, may skip 2 Terrain, or 2 River spaces, or one each when doing Carpet Flight
// Special: Start with 7/5 power (not standard 5/7)
//
//	Cannot upgrade Shipping
//	Can only upgrade Digging by 1 level (max level 1, not 2)
//	Expensive Stronghold (4 workers, 10 coins vs standard 4 workers, 6 coins)
//	Shipping town tile increases Carpet Flight range by 1 (can get multiple)
type Fakirs struct {
	BaseFaction
	hasStronghold bool
	flightRange   int // Base flight range (starts at 1, +1 for stronghold, +1 per shipping town tile)
}

// NewFakirs creates a new Fakirs faction
func NewFakirs() *Fakirs {
	return &Fakirs{
		BaseFaction: BaseFaction{
			Type:        models.FactionFakirs,
			HomeTerrain: models.TerrainDesert,
			StartingRes: Resources{
				Coins:   15,
				Workers: 3,
				Priests: 0,
				Power1:  7, // Fakirs start with 7/5 power (not standard 5/7)
				Power2:  5,
				Power3:  0,
			},
			DiggingLevel: 0,
		},
		hasStronghold: false,
		flightRange:   1, // Base flight range of 1
	}
}

// GetStartingCultPositions returns Fakirs starting cult track positions
func (f *Fakirs) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 1, Water: 0, Earth: 0, Air: 1}
}

// GetStrongholdCost returns the expensive stronghold cost for Fakirs
func (f *Fakirs) GetStrongholdCost() Cost {
	return Cost{
		Coins:   10, // More expensive than standard (6)
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
}

// GetShippingCost overrides the base method
// Fakirs cannot increase shipping
func (f *Fakirs) GetShippingCost(currentLevel int) Cost {
	// Fakirs cannot upgrade shipping
	return Cost{
		Coins:   0,
		Workers: 0,
		Priests: 0,
		Power:   0,
	}
}

// GetDiggingCost overrides the base method
// Fakirs can only upgrade digging once (to level 1)
func (f *Fakirs) GetDiggingCost(currentLevel int) Cost {
	if currentLevel >= 1 {
		// Cannot upgrade past level 1
		return Cost{
			Coins:   0,
			Workers: 0,
			Priests: 0,
			Power:   0,
		}
	}
	// Standard cost for first upgrade (0 -> 1)
	return StandardDiggingCost(currentLevel)
}

// GetMaxDiggingLevel returns the maximum digging level for Fakirs
func (f *Fakirs) GetMaxDiggingLevel() int {
	return 1 // Fakirs can only reach digging level 1 (not 2)
}

// BuildStronghold marks that the stronghold has been built and increases flight range
func (f *Fakirs) BuildStronghold() {
	if !f.hasStronghold {
		f.hasStronghold = true
		f.flightRange++ // Stronghold adds +1 flight range
	}
}

// HasStronghold returns whether the stronghold has been built
func (f *Fakirs) HasStronghold() bool {
	return f.hasStronghold
}

// GetFlightRange returns the current carpet flight range
// Range 1 = can skip 1 hex (connect buildings 2 apart)
// Range 2 = can skip 2 hexes (connect buildings 3 apart)
// etc.
func (f *Fakirs) GetFlightRange() int {
	return f.flightRange
}

// IncrementFlightRange increases the flight range by 1 (e.g., from shipping town tile)
func (f *Fakirs) IncrementFlightRange() {
	f.flightRange++
}

// Income methods (Fakirs-specific)

// GetStrongholdIncome returns the income for the stronghold
func (f *Fakirs) GetStrongholdIncome() Income {
	// Fakirs: ONLY stronghold that gives priest income (1 priest, no power)
	return Income{Priests: 1}
}
