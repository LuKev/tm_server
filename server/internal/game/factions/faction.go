package factions

import (
	"github.com/lukev/tm_server/internal/models"
)

// Faction defines the interface that all faction implementations must satisfy
type Faction interface {
	// Basic properties
	GetType() models.FactionType
	GetHomeTerrain() models.TerrainType
	GetStartingResources() Resources
	GetStartingCultPositions() CultPositions

	// Building costs
	GetDwellingCost() Cost
	GetTradingHouseCost() Cost
	GetTempleCost() Cost
	GetSanctuaryCost() Cost
	GetStrongholdCost() Cost

	// Terraform costs (returns workers needed per spade)
	GetTerraformCost(distance int) int
	GetTerraformSpades(distance int) int // Returns actual spades used (for scoring)

	// Shipping and digging
	GetShippingCost(currentLevel int) Cost
	GetDiggingCost(currentLevel int) Cost

	// Income methods
	GetBaseFactionIncome() Income
	GetDwellingIncome(dwellingCount int) Income
	GetTradingHouseIncome(tradingHouseCount int) Income
	GetTempleIncome(templeCount int) Income
	GetSanctuaryIncome() Income // Only 1 sanctuary per faction, no count parameter
	GetStrongholdIncome() Income

	// Special abilities

	CanUseSpecialAction(action string, gameState interface{}) bool
	ExecuteSpecialAction(action string, gameState interface{}) error
}

// BaseFaction provides default implementations for common faction behavior
type BaseFaction struct {
	Type         models.FactionType
	HomeTerrain  models.TerrainType
	StartingRes  Resources
	DiggingLevel int // Base digging level (0-2)
}

// Resources represents a faction's resources
type Resources struct {
	Coins   int
	Workers int
	Priests int
	Power1  int // Power in bowl 1
	Power2  int // Power in bowl 2
	Power3  int // Power in bowl 3
}

// CultPositions represents starting positions on cult tracks
type CultPositions struct {
	Fire  int
	Water int
	Earth int
	Air   int
}

// Cost represents the cost of an action
type Cost struct {
	Coins   int
	Workers int
	Priests int
	Power   int // Power spent from bowl 3
}

// Income represents resource income per round
type Income struct {
	Coins   int
	Workers int
	Priests int
	Power   int
}

// Standard building costs (can be overridden by factions)
var (
	StandardDwellingCost = Cost{
		Coins:   2,
		Workers: 1,
		Priests: 0,
		Power:   0,
	}

	StandardTradingHouseCost = Cost{
		Coins:   6,
		Workers: 2,
		Priests: 0,
		Power:   0,
	}

	StandardTempleCost = Cost{
		Coins:   5,
		Workers: 2,
		Priests: 0,
		Power:   0,
	}

	StandardSanctuaryCost = Cost{
		Coins:   6,
		Workers: 4,
		Priests: 0,
		Power:   0,
	}

	StandardStrongholdCost = Cost{
		Coins:   6,
		Workers: 4,
		Priests: 0,
		Power:   0,
	}
)

// Standard shipping and digging costs
func StandardShippingCost(currentLevel int) Cost {
	// Shipping 0->1: 4 coins, 1 priest
	// Shipping 1->2: 4 coins, 1 priest
	// Shipping 2->3: 4 coins, 1 priest
	return Cost{
		Coins:   4,
		Workers: 0,
		Priests: 1,
		Power:   0,
	}
}

func StandardDiggingCost(currentLevel int) Cost {
	// Digging 0->1: 2 workers, 5 coins, 1 priest
	// Digging 1->2: 2 workers, 5 coins, 1 priest
	return Cost{
		Coins:   5,
		Workers: 2,
		Priests: 1,
		Power:   0,
	}
}

// BaseFaction method implementations (defaults)

func (f *BaseFaction) GetType() models.FactionType {
	return f.Type
}

func (f *BaseFaction) GetHomeTerrain() models.TerrainType {
	return f.HomeTerrain
}

func (f *BaseFaction) GetStartingResources() Resources {
	return f.StartingRes
}

func (f *BaseFaction) GetStartingCultPositions() CultPositions {
	return CultPositions{Fire: 0, Water: 0, Earth: 0, Air: 0}
}

func (f *BaseFaction) GetDwellingCost() Cost {
	return StandardDwellingCost
}

func (f *BaseFaction) GetTradingHouseCost() Cost {
	return StandardTradingHouseCost
}

func (f *BaseFaction) GetTempleCost() Cost {
	return StandardTempleCost
}

func (f *BaseFaction) GetSanctuaryCost() Cost {
	return StandardSanctuaryCost
}

func (f *BaseFaction) GetStrongholdCost() Cost {
	return StandardStrongholdCost
}

func (f *BaseFaction) GetTerraformCost(distance int) int {
	// Base cost: 3 workers per spade, reduced by digging level
	costPerSpade := 3 - f.DiggingLevel
	if costPerSpade < 1 {
		costPerSpade = 1
	}
	return distance * costPerSpade
}

func (f *BaseFaction) GetTerraformSpades(distance int) int {
	// For most factions, spades = distance
	return distance
}

func (f *BaseFaction) GetShippingCost(currentLevel int) Cost {
	return StandardShippingCost(currentLevel)
}

func (f *BaseFaction) GetDiggingCost(currentLevel int) Cost {
	return StandardDiggingCost(currentLevel)
}

func (f *BaseFaction) CanUseSpecialAction(action string, gameState interface{}) bool {
	return false // Override in specific factions
}

func (f *BaseFaction) ExecuteSpecialAction(action string, gameState interface{}) error {
	return nil // Override in specific factions
}

// Income method implementations (defaults)

func (f *BaseFaction) GetBaseFactionIncome() Income {
	// Standard: 1 worker base income
	return Income{Workers: 1}
}

func (f *BaseFaction) GetDwellingIncome(dwellingCount int) Income {
	// Standard: 1 worker per dwelling, except 8th gives no income
	if dwellingCount >= 8 {
		return Income{Workers: 7}
	}
	return Income{Workers: dwellingCount}
}

func (f *BaseFaction) GetTradingHouseIncome(tradingHouseCount int) Income {
	// Standard: 1st-2nd: 2c+1pw, 3rd-4th: 2c+2pw
	income := Income{}
	for i := 1; i <= tradingHouseCount && i <= 4; i++ {
		income.Coins += 2
		if i <= 2 {
			income.Power += 1
		} else {
			income.Power += 2
		}
	}
	return income
}

func (f *BaseFaction) GetTempleIncome(templeCount int) Income {
	// Standard: 1 priest per temple
	return Income{Priests: templeCount}
}

func (f *BaseFaction) GetSanctuaryIncome() Income {
	// Standard: 1 priest per sanctuary
	return Income{Priests: 1}
}

func (f *BaseFaction) GetStrongholdIncome() Income {
	// Standard: 2 power, NO priest (only Fakirs stronghold gives priest)
	return Income{Power: 2}
}

// Helper functions

// CanAfford checks if resources are sufficient for a cost
func CanAfford(resources Resources, cost Cost) bool {
	return resources.Coins >= cost.Coins &&
		resources.Workers >= cost.Workers &&
		resources.Priests >= cost.Priests &&
		resources.Power3 >= cost.Power
}

// Subtract deducts a cost from resources
func Subtract(resources Resources, cost Cost) Resources {
	return Resources{
		Coins:   resources.Coins - cost.Coins,
		Workers: resources.Workers - cost.Workers,
		Priests: resources.Priests - cost.Priests,
		Power1:  resources.Power1,
		Power2:  resources.Power2,
		Power3:  resources.Power3 - cost.Power,
	}
}

// Add adds resources together
func Add(a, b Resources) Resources {
	return Resources{
		Coins:   a.Coins + b.Coins,
		Workers: a.Workers + b.Workers,
		Priests: a.Priests + b.Priests,
		Power1:  a.Power1 + b.Power1,
		Power2:  a.Power2 + b.Power2,
		Power3:  a.Power3 + b.Power3,
	}
}
