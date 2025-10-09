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
	
	// Building costs
	GetDwellingCost() Cost
	GetTradingHouseCost() Cost
	GetTempleCost() Cost
	GetSanctuaryCost() Cost
	GetStrongholdCost() Cost
	
	// Terraform costs (returns workers needed per spade)
	GetTerraformCost(distance int) int
	
	// Shipping and digging
	GetShippingCost(currentLevel int) Cost
	GetDiggingCost(currentLevel int) Cost
	
	// Special abilities
	HasSpecialAbility(ability SpecialAbility) bool
	CanUseSpecialAction(action string, gameState interface{}) bool
	ExecuteSpecialAction(action string, gameState interface{}) error
	
	// Stronghold ability
	GetStrongholdAbility() string
	ExecuteStrongholdAbility(gameState interface{}) error
	
	// Income modifiers
	ModifyIncome(baseIncome Resources) Resources
	
	// Action modifiers
	CanBuildOnWater() bool // For Fakirs, Mermaids
	CanFly() bool          // For Witches
	GetBridgeCost() Cost   // For Engineers (reduced cost)
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

// Cost represents the cost of an action
type Cost struct {
	Coins   int
	Workers int
	Priests int
	Power   int // Power spent from bowl 3
}

// SpecialAbility represents unique faction abilities
type SpecialAbility int

const (
	AbilityNone SpecialAbility = iota
	AbilityFlying
	AbilityWaterBuilding
	AbilityBridgeBuilding
	AbilityTunnelDigging
	AbilitySandstorm
	AbilityCarpetFlying
	AbilityFavorTransform
	AbilityCheapDwellings
	AbilityTownBonus
	AbilityFavorBenefits
	AbilitySpadeEfficiency
	AbilityCultBonus
	AbilityConversionEfficiency
	AbilityPriestBenefits
)

// Standard building costs (can be overridden by factions)
var (
	StandardDwellingCost = Cost{
		Coins:   0,
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
		Coins:   8,
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
	// Digging 0->1: 5 workers, 2 coins
	// Digging 1->2: 5 workers, 2 coins
	return Cost{
		Coins:   2,
		Workers: 5,
		Priests: 0,
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

func (f *BaseFaction) GetShippingCost(currentLevel int) Cost {
	return StandardShippingCost(currentLevel)
}

func (f *BaseFaction) GetDiggingCost(currentLevel int) Cost {
	return StandardDiggingCost(currentLevel)
}

func (f *BaseFaction) HasSpecialAbility(ability SpecialAbility) bool {
	return false // Override in specific factions
}

func (f *BaseFaction) CanUseSpecialAction(action string, gameState interface{}) bool {
	return false // Override in specific factions
}

func (f *BaseFaction) ExecuteSpecialAction(action string, gameState interface{}) error {
	return nil // Override in specific factions
}

func (f *BaseFaction) GetStrongholdAbility() string {
	return "" // Override in specific factions
}

func (f *BaseFaction) ExecuteStrongholdAbility(gameState interface{}) error {
	return nil // Override in specific factions
}

func (f *BaseFaction) ModifyIncome(baseIncome Resources) Resources {
	return baseIncome // Override in specific factions
}

func (f *BaseFaction) CanBuildOnWater() bool {
	return false
}

func (f *BaseFaction) CanFly() bool {
	return false
}

func (f *BaseFaction) GetBridgeCost() Cost {
	// Standard bridge cost: 3 workers
	return Cost{
		Coins:   0,
		Workers: 3,
		Priests: 0,
		Power:   0,
	}
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
