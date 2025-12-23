package game

import (
	"fmt"
)

// PowerSystem manages the 3-bowl power cycle for a player
// Bowl 1: Inactive power
// Bowl 2: Power ready to be cycled
// Bowl 3: Active power that can be spent
type PowerSystem struct {
	Bowl1 int `json:"powerI"`   // Inactive power
	Bowl2 int `json:"powerII"`  // Power ready to cycle
	Bowl3 int `json:"powerIII"` // Active power (spendable)
}

// NewPowerSystem creates a new power system with starting power distribution
func NewPowerSystem(bowl1, bowl2, bowl3 int) *PowerSystem {
	return &PowerSystem{
		Bowl1: bowl1,
		Bowl2: bowl2,
		Bowl3: bowl3,
	}
}

// TotalPower returns the total power across all three bowls
func (ps *PowerSystem) TotalPower() int {
	return ps.Bowl1 + ps.Bowl2 + ps.Bowl3
}

// GainPower adds power to the system
// If there is power in bowl 1, it moves to bowl 2.
// If there is power in bowl 2 but not bowl 1, then power cycles through Bowl 2 to Bowl 3
// amount: the amount of power to gain
// Returns the actual amount of power gained (may be less if bowls are full)
func (ps *PowerSystem) GainPower(amount int) int {
	if amount <= 0 {
		return 0
	}

	remaining := amount

	// Moving from bowl 1 to bowl 2
	if ps.Bowl1 >= amount {
		ps.Bowl1 -= amount
		ps.Bowl2 += amount
		return amount
	}

	// Not enough; move all power from bowl 1 to bowl 2
	oneToTwo := ps.Bowl1
	ps.Bowl1 = 0
	ps.Bowl2 += oneToTwo
	remaining -= oneToTwo

	// Move power from bowl 2 to bowl 3
	twoToThree := remaining
	if ps.Bowl2 < remaining {
		twoToThree = ps.Bowl2
	}
	ps.Bowl2 -= twoToThree
	ps.Bowl3 += twoToThree

	return oneToTwo + twoToThree
}

// SpendPower spends power from Bowl 3
// Spent power goes back to Bowl 1
// amount: the amount of power to spend
// Returns error if not enough power in Bowl 3
func (ps *PowerSystem) SpendPower(amount int) error {
	if amount < 0 {
		return fmt.Errorf("cannot spend negative power")
	}

	if amount > ps.Bowl3 {
		return fmt.Errorf("cannot spend %d power, only %d available in Bowl 3", amount, ps.Bowl3)
	}

	ps.Bowl3 -= amount
	ps.Bowl1 += amount

	return nil
}

// CanSpend checks if the player has enough power in Bowl 3 to spend
func (ps *PowerSystem) CanSpend(amount int) bool {
	return ps.Bowl3 >= amount
}

// BurnPower converts power from Bowl 2 to Bowl 3 at a 2:1 ratio
// This is a special action where players can "burn" 2 power from Bowl 2 to get 1 power in Bowl 3
// amount: the amount of power to gain in Bowl 3 (will cost 2x this from Bowl 2)
// Returns error if not enough power in Bowl 2
func (ps *PowerSystem) BurnPower(amount int) error {
	if amount < 0 {
		return fmt.Errorf("cannot burn negative power")
	}

	cost := amount * 2
	if cost > ps.Bowl2 {
		return fmt.Errorf("cannot burn for %d power, need %d in Bowl 2 but only have %d", amount, cost, ps.Bowl2)
	}

	ps.Bowl2 -= cost
	ps.Bowl3 += amount

	return nil
}

// CanBurn checks if the player can burn power to get the specified amount in Bowl 3
func (ps *PowerSystem) CanBurn(amount int) bool {
	return ps.Bowl2 >= (amount * 2)
}

// Clone creates a deep copy of the power system
func (ps *PowerSystem) Clone() *PowerSystem {
	return &PowerSystem{
		Bowl1: ps.Bowl1,
		Bowl2: ps.Bowl2,
		Bowl3: ps.Bowl3,
	}
}
