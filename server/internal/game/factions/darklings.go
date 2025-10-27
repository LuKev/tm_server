package factions

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Darklings faction - Black/Swamp
// Ability: Pay Priests (instead of Workers) when transforming Terrain
//          Pay 1 Priest and get 2 VP for each step of transformation
//          IMPORTANT: Darklings can NEVER upgrade digging (cannot use workers for spades)
// Stronghold: After building, immediately and only once trade up to 3 Workers for 1 Priest each
// Special: Sanctuary costs 4 workers/10 coins (more expensive than standard)
//          Start with 1 Priest, 1 Worker, 15 Coins
type Darklings struct {
	BaseFaction
	hasStronghold              bool
	hasUsedPriestOrdination    bool // One-time worker->priest conversion
}

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
		hasStronghold:           false,
		hasUsedPriestOrdination: false,
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

// HasSpecialAbility returns true for priest benefits
func (f *Darklings) HasSpecialAbility(ability SpecialAbility) bool {
	return ability == AbilityPriestBenefits
}

// GetStrongholdAbility returns the description of the stronghold ability
func (f *Darklings) GetStrongholdAbility() string {
	return "Ordination of Priests: Immediately and only once, trade up to 3 Workers for 1 Priest each"
}

// BuildStronghold marks that the stronghold has been built
func (f *Darklings) BuildStronghold() {
	f.hasStronghold = true
}

// CanUsePriestOrdination checks if the priest ordination can be used
func (f *Darklings) CanUsePriestOrdination() bool {
	return f.hasStronghold && !f.hasUsedPriestOrdination
}

// UsePriestOrdination trades workers for priests (0-3 workers)
// Returns the number of priests gained
// Player can choose 0 to decline the conversion, but ability is still used up
func (f *Darklings) UsePriestOrdination(workersToTrade int) (int, error) {
	if !f.hasStronghold {
		return 0, fmt.Errorf("must build stronghold before using priest ordination")
	}

	if f.hasUsedPriestOrdination {
		return 0, fmt.Errorf("priest ordination already used")
	}

	if workersToTrade < 0 || workersToTrade > 3 {
		return 0, fmt.Errorf("can only trade 0-3 workers")
	}

	f.hasUsedPriestOrdination = true

	// 1 Worker = 1 Priest
	return workersToTrade, nil
}

// GetTerraformCostInPriests returns the priest cost for terraforming
// Darklings pay priests instead of workers
// NOTE: Phase 6.2 (Action System) must use this instead of GetTerraformCost()
func (f *Darklings) GetTerraformCostInPriests(distance int) int {
	// Darklings pay 1 priest per spade (instead of 3 workers per spade)
	return distance
}

// GetTerraformVPBonus returns the VP bonus for terraforming
// Darklings get 2 VP per step of transformation
// NOTE: Phase 8 (Scoring System) tracks VP
// NOTE: Phase 6.2 (Action System) must apply this bonus when Darklings terraform
func (f *Darklings) GetTerraformVPBonus(distance int) int {
	return distance * 2 // 2 VP per spade
}

// GetTerraformCost overrides the base method
// Darklings don't use workers for terraform, they use priests
// NOTE: Phase 6.2 (Action System) must use GetTerraformCostInPriests() for Darklings
func (f *Darklings) GetTerraformCost(distance int) int {
	// Return 0 workers (Darklings use priests instead)
	return 0
}

// GetDiggingCost overrides the base method
// Darklings can NEVER upgrade digging (they use priests, not workers for spades)
// NOTE: Phase 6.2 (Action System) must prevent Darklings from taking digging actions
func (f *Darklings) GetDiggingCost(currentLevel int) Cost {
	// Darklings cannot upgrade digging
	return Cost{
		Coins:   0,
		Workers: 0,
		Priests: 0,
		Power:   0,
	}
}

// ExecuteStrongholdAbility implements the Faction interface
func (f *Darklings) ExecuteStrongholdAbility(gameState interface{}) error {
	// Stronghold ability is the one-time priest ordination
	// This is handled by UsePriestOrdination()
	return nil
}
