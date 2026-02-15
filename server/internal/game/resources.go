package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
)

// ResourcePool manages a player's resources including power system
type ResourcePool struct {
	Coins   int          `json:"coins"`
	Workers int          `json:"workers"`
	Priests int          `json:"priests"`
	Power   *PowerSystem `json:"power"`
}

// NewResourcePool creates a new resource pool from starting resources
func NewResourcePool(startingRes factions.Resources) *ResourcePool {
	return &ResourcePool{
		Coins:   startingRes.Coins,
		Workers: startingRes.Workers,
		Priests: startingRes.Priests,
		Power:   NewPowerSystem(startingRes.Power1, startingRes.Power2, startingRes.Power3),
	}
}

// CanAfford checks if the player has enough resources to pay a cost
func (rp *ResourcePool) CanAfford(cost factions.Cost) bool {
	if rp.Coins < cost.Coins {
		return false
	}
	if rp.Workers < cost.Workers {
		return false
	}
	if rp.Priests < cost.Priests {
		return false
	}
	if cost.Power > 0 && !rp.Power.CanSpend(cost.Power) {
		return false
	}
	return true
}

// Spend deducts resources from the pool
// Returns error if not enough resources
func (rp *ResourcePool) Spend(cost factions.Cost) error {
	if !rp.CanAfford(cost) {
		return fmt.Errorf("insufficient resources: need (coins:%d, workers:%d, priests:%d, power:%d), have (coins:%d, workers:%d, priests:%d, power:%d)",
			cost.Coins, cost.Workers, cost.Priests, cost.Power,
			rp.Coins, rp.Workers, rp.Priests, rp.Power.Bowl3)
	}

	rp.Coins -= cost.Coins
	rp.Workers -= cost.Workers
	rp.Priests -= cost.Priests

	if cost.Power > 0 {
		if err := rp.Power.SpendPower(cost.Power); err != nil {
			return fmt.Errorf("failed to spend power: %w", err)
		}
	}

	return nil
}

// Gain adds resources to the pool
func (rp *ResourcePool) Gain(coins, workers, priests int) {
	if coins > 0 {
		rp.Coins += coins
	}
	if workers > 0 {
		rp.Workers += workers
	}
	if priests > 0 {
		rp.Priests += priests
	}
}

// GainPower adds power to the power system (cycles through bowls)
func (rp *ResourcePool) GainPower(amount int) int {
	return rp.Power.GainPower(amount)
}

func (rp *ResourcePool) ConvertPowerToCoins(amount int) error {
	if !rp.Power.CanSpend(amount) {
		return fmt.Errorf("need %d power in bowl 3, only have %d", amount, rp.Power.Bowl3)
	}

	if err := rp.Power.SpendPower(amount); err != nil {
		return err
	}

	rp.Coins += amount
	return nil
}

func (rp *ResourcePool) ConvertPowerToWorkers(numWorkers int) error {
	powerNeeded := numWorkers * 3
	if !rp.Power.CanSpend(powerNeeded) {
		return fmt.Errorf("need %d power in bowl 3 to convert to %d workers, only have %d", powerNeeded, numWorkers, rp.Power.Bowl3)
	}

	if err := rp.Power.SpendPower(powerNeeded); err != nil {
		return err
	}

	rp.Workers += numWorkers
	return nil
}

func (rp *ResourcePool) ConvertPowerToPriests(numPriests int) error {
	powerNeeded := numPriests * 5
	if !rp.Power.CanSpend(powerNeeded) {
		return fmt.Errorf("need %d power in bowl 3 to convert to %d priests, only have %d", powerNeeded, numPriests, rp.Power.Bowl3)
	}

	if err := rp.Power.SpendPower(powerNeeded); err != nil {
		return err
	}

	rp.Priests += numPriests
	return nil
}

func (rp *ResourcePool) ConvertPriestToWorker(numWorkers int) error {
	if rp.Priests < numWorkers {
		return fmt.Errorf("need %d priests, only have %d", numWorkers, rp.Priests)
	}

	rp.Priests -= numWorkers
	rp.Workers += numWorkers
	return nil
}

func (rp *ResourcePool) ConvertWorkerToCoin(numCoins int) error {
	if rp.Workers < numCoins {
		return fmt.Errorf("need %d workers, only have %d", numCoins, rp.Workers)
	}

	rp.Workers -= numCoins
	rp.Coins += numCoins
	return nil
}

// BurnPower converts power from bowl 2 to bowl 3 at 2:1 ratio
func (rp *ResourcePool) BurnPower(amount int) error {
	return rp.Power.BurnPower(amount)
}

// Clone creates a deep copy of the resource pool
func (rp *ResourcePool) Clone() *ResourcePool {
	return &ResourcePool{
		Coins:   rp.Coins,
		Workers: rp.Workers,
		Priests: rp.Priests,
		Power:   rp.Power.Clone(),
	}
}

// ToResources converts the resource pool to a Resources struct
func (rp *ResourcePool) ToResources() factions.Resources {
	return factions.Resources{
		Coins:   rp.Coins,
		Workers: rp.Workers,
		Priests: rp.Priests,
		Power1:  rp.Power.Bowl1,
		Power2:  rp.Power.Bowl2,
		Power3:  rp.Power.Bowl3,
	}
}

// PowerLeechOffer represents an offer to gain power from a neighbor's building
type PowerLeechOffer struct {
	// Amount of power offered by the source event. This is not capped by the
	// receiver's current power-cycle capacity; the receiver may only gain as
	// much power as they can charge when accepting.
	Amount       int
	VPCost       int // VP cost to accept (usually Amount - 1)
	FromPlayerID string
	EventID      int `json:"eventId"`
}

// NewPowerLeechOffer creates a power leech offer based on building value and player's power capacity
// Building values: Dwelling=1, Trading House=2, Temple=2, Sanctuary=3, Stronghold=3
func NewPowerLeechOffer(buildingValue int, fromPlayerID string, targetPower *PowerSystem) *PowerLeechOffer {
	if buildingValue <= 0 {
		return nil
	}
	if targetPower == nil {
		return nil
	}

	// Capacity to gain power depends on the power cycle:
	// - Each token in Bowl I can contribute up to 2 gained power (I->II then II->III).
	// - Each token in Bowl II can contribute up to 1 gained power (II->III).
	// Bowl III is already active and does not provide capacity to gain.
	capacity := targetPower.Bowl2 + 2*targetPower.Bowl1
	if capacity <= 0 {
		return nil
	}

	return &PowerLeechOffer{
		Amount:       buildingValue,
		// Snellman leech VP cost model: actualTaken - 1 (minimum 0).
		// Note: this value is informational; actual VP cost is derived from the
		// power actually gained at acceptance time (capacity can change).
		VPCost:       maxInt(0, buildingValue-1),
		FromPlayerID: fromPlayerID,
	}
}

// AcceptPowerLeech accepts a power leech offer
// Player gains power but loses VP
// Returns the VP cost that should be deducted
func (rp *ResourcePool) AcceptPowerLeech(offer *PowerLeechOffer) int {
	if offer == nil {
		return 0
	}

	// VP cost is based on power actually gained now, not on the offer's
	// original snapshot. This matters when multiple pending leeches resolve
	// sequentially and the player no longer has capacity for the full amount.
	actualGained := rp.GainPower(offer.Amount)
	// VP cost is based on power actually gained (capacity can change as offers resolve).
	// Snellman cost curve: actualTaken - 1 (minimum 0).
	if actualGained <= 1 {
		return 0
	}
	return actualGained - 1
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DeclinePowerLeech declines a power leech offer (no effect)
func (rp *ResourcePool) DeclinePowerLeech(offer *PowerLeechOffer) {
	// No effect - player chooses not to gain power
}
