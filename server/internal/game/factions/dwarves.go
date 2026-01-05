package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Dwarves faction - Gray/Mountain
// Ability: When taking "Transform and Build" action, may skip one Terrain or River space by paying 2 more Workers (Tunneling)
//
//	Get 4 VP each time when Tunneling
//	Have no Shipping (cannot increase shipping level)
//	In final Area scoring, Structures reachable via Tunneling are considered connected
//
// Stronghold: After building, only pay 1 more Worker instead of 2 when Tunneling
// Special: Cannot increase shipping level
type Dwarves struct {
	BaseFaction
	hasStronghold bool
}

// NewDwarves creates a new Dwarves faction
func NewDwarves() *Dwarves {
	return &Dwarves{
		BaseFaction: BaseFaction{
			Type:        models.FactionDwarves,
			HomeTerrain: models.TerrainMountain,
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

// GetStartingCultPositions returns Dwarves starting cult track positions
func (f *Dwarves) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 0, Water: 0, Earth: 2, Air: 0}
}

// GetShippingCost overrides the base method
// Dwarves cannot increase shipping (they have no shipping)
func (f *Dwarves) GetShippingCost(currentLevel int) Cost {
	// Dwarves cannot upgrade shipping
	return Cost{
		Coins:   0,
		Workers: 0,
		Priests: 0,
		Power:   0,
	}
}

// BuildStronghold marks that the stronghold has been built
func (f *Dwarves) BuildStronghold() {
	f.hasStronghold = true
}

// HasStronghold returns whether the stronghold has been built
func (f *Dwarves) HasStronghold() bool {
	return f.hasStronghold
}

// CanTunnel returns whether Dwarves can use tunneling
// Tunneling is always available for Dwarves (it's their core ability)
func (f *Dwarves) CanTunnel() bool {
	return true
}

// Income methods (Dwarves-specific)

// GetTradingHouseIncome returns the income for trading houses
func (f *Dwarves) GetTradingHouseIncome(tradingHouseCount int) Income {
	// Dwarves: 1st: 3c+1pw, 2nd: 2c+1pw, 3rd: 2c+2pw, 4th: 3c+2pw
	income := Income{}
	for i := 1; i <= tradingHouseCount && i <= 4; i++ {
		switch i {
		case 1:
			income.Coins += 3
			income.Power++
		case 2:
			income.Coins += 2
			income.Power++
		case 3:
			income.Coins += 2
			income.Power += 2
		case 4:
			income.Coins += 3
			income.Power += 2
		}
	}
	return income
}
