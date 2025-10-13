package game

import (
	"testing"
)

func TestNewPowerSystem(t *testing.T) {
	ps := NewPowerSystem(5, 7, 0)
	
	if ps.Bowl1 != 5 {
		t.Errorf("expected Bowl1 = 5, got %d", ps.Bowl1)
	}
	if ps.Bowl2 != 7 {
		t.Errorf("expected Bowl2 = 7, got %d", ps.Bowl2)
	}
	if ps.Bowl3 != 0 {
		t.Errorf("expected Bowl3 = 0, got %d", ps.Bowl3)
	}
}

func TestTotalPower(t *testing.T) {
	ps := NewPowerSystem(5, 7, 3)
	
	total := ps.TotalPower()
	if total != 15 {
		t.Errorf("expected total power = 15, got %d", total)
	}
}

func TestGainPower(t *testing.T) {
	ps := NewPowerSystem(5, 7, 0)
	
	// Gain 3 power
	gained := ps.GainPower(3)
	
	if gained != 3 {
		t.Errorf("expected to gain 3 power, got %d", gained)
	}
	if ps.Bowl1 != 2 {
		t.Errorf("expected Bowl1 = 2, got %d", ps.Bowl1)
	}
	if ps.Bowl2 != 10 {
		t.Errorf("expected Bowl2 = 10, got %d", ps.Bowl2)
	}
}

func TestGainPowerZero(t *testing.T) {
	ps := NewPowerSystem(5, 7, 0)
	
	gained := ps.GainPower(0)
	
	if gained != 0 {
		t.Errorf("expected to gain 0 power, got %d", gained)
	}
	if ps.Bowl1 != 5 {
		t.Errorf("expected Bowl1 = 5 (unchanged), got %d", ps.Bowl1)
	}
}

func TestGainPowerOverflow(t *testing.T) {
	ps := NewPowerSystem(0, 3, 9)
	
	gained := ps.GainPower(10)
	
	if gained != 3 {
		t.Errorf("expected to gain 3 power, got %d", gained)
	}
	if ps.Bowl1 != 0 {
		t.Errorf("expected Bowl1 = 0, got %d", ps.Bowl1)
	}
	if ps.Bowl2 != 0 {
		t.Errorf("expected Bowl2 = 0, got %d", ps.Bowl2)
	}
	if ps.Bowl3 != 0 {
		t.Errorf("expected Bowl3 = 0, got %d", ps.Bowl3)
	}
}

func TestSpendPower(t *testing.T) {
	ps := NewPowerSystem(5, 7, 6)
	
	// Spend 4 power
	err := ps.SpendPower(4)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps.Bowl3 != 2 {
		t.Errorf("expected Bowl3 = 2, got %d", ps.Bowl3)
	}
	if ps.Bowl1 != 9 {
		t.Errorf("expected Bowl1 = 9 (spent power returns), got %d", ps.Bowl1)
	}
}

func TestSpendPowerTooMuch(t *testing.T) {
	ps := NewPowerSystem(5, 7, 3)
	
	// Try to spend more than available
	err := ps.SpendPower(5)
	
	if err == nil {
		t.Errorf("expected error when spending too much power")
	}
	// State should be unchanged
	if ps.Bowl3 != 3 {
		t.Errorf("expected Bowl3 = 3 (unchanged), got %d", ps.Bowl3)
	}
	if ps.Bowl1 != 5 {
		t.Errorf("expected Bowl1 = 5 (unchanged), got %d", ps.Bowl1)
	}
}

func TestCanSpend(t *testing.T) {
	ps := NewPowerSystem(5, 7, 6)
	
	if !ps.CanSpend(6) {
		t.Errorf("should be able to spend 6 power")
	}
	if !ps.CanSpend(3) {
		t.Errorf("should be able to spend 3 power")
	}
	if ps.CanSpend(7) {
		t.Errorf("should not be able to spend 7 power")
	}
}

func TestBurnPower(t *testing.T) {
	ps := NewPowerSystem(5, 10, 0)
	
	// Burn to get 3 power in Bowl 3 (costs 6 from Bowl 2)
	err := ps.BurnPower(3)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps.Bowl2 != 4 {
		t.Errorf("expected Bowl2 = 4 (10 - 6), got %d", ps.Bowl2)
	}
	if ps.Bowl3 != 3 {
		t.Errorf("expected Bowl3 = 3, got %d", ps.Bowl3)
	}
}

func TestBurnPowerNotEnough(t *testing.T) {
	ps := NewPowerSystem(5, 5, 0)
	
	// Try to burn for 3 power (would cost 6 from Bowl 2, but only have 5)
	err := ps.BurnPower(3)
	
	if err == nil {
		t.Errorf("expected error when burning without enough power")
	}
	// State should be unchanged
	if ps.Bowl2 != 5 {
		t.Errorf("expected Bowl2 = 5 (unchanged), got %d", ps.Bowl2)
	}
	if ps.Bowl3 != 0 {
		t.Errorf("expected Bowl3 = 0 (unchanged), got %d", ps.Bowl3)
	}
}

func TestCanBurn(t *testing.T) {
	ps := NewPowerSystem(5, 10, 0)
	
	if !ps.CanBurn(5) {
		t.Errorf("should be able to burn for 5 power (costs 10)")
	}
	if !ps.CanBurn(3) {
		t.Errorf("should be able to burn for 3 power (costs 6)")
	}
	if ps.CanBurn(6) {
		t.Errorf("should not be able to burn for 6 power (would cost 12)")
	}
}

// This logic is wrong, please fix it.
func TestPowerCycle(t *testing.T) {
	// Test a full power cycle
	ps := NewPowerSystem(5, 7, 0)
	
	// 1. Gain power (goes to Bowl 1)
	ps.GainPower(3)
	if ps.Bowl1 != 8 || ps.Bowl2 != 7 || ps.Bowl3 != 0 {
		t.Errorf("after gain: expected (8,7,0), got (%d,%d,%d)", ps.Bowl1, ps.Bowl2, ps.Bowl3)
	}
	
	// 2. Income phase (Bowl 1 → Bowl 2)
	ps.IncomePhase()
	if ps.Bowl1 != 0 || ps.Bowl2 != 15 || ps.Bowl3 != 0 {
		t.Errorf("after income: expected (0,15,0), got (%d,%d,%d)", ps.Bowl1, ps.Bowl2, ps.Bowl3)
	}
	
	// 3. Cycle power (Bowl 2 → Bowl 3)
	ps.CyclePower(10)
	if ps.Bowl1 != 0 || ps.Bowl2 != 5 || ps.Bowl3 != 10 {
		t.Errorf("after cycle: expected (0,5,10), got (%d,%d,%d)", ps.Bowl1, ps.Bowl2, ps.Bowl3)
	}
	
	// 4. Spend power (Bowl 3 → Bowl 1)
	ps.SpendPower(6)
	if ps.Bowl1 != 6 || ps.Bowl2 != 5 || ps.Bowl3 != 4 {
		t.Errorf("after spend: expected (6,5,4), got (%d,%d,%d)", ps.Bowl1, ps.Bowl2, ps.Bowl3)
	}
	
	// Total power should remain constant
	if ps.TotalPower() != 15 {
		t.Errorf("total power should remain 15, got %d", ps.TotalPower())
	}
}

func TestClone(t *testing.T) {
	ps := NewPowerSystem(5, 7, 3)
	
	clone := ps.Clone()
	
	// Clone should have same values
	if clone.Bowl1 != 5 || clone.Bowl2 != 7 || clone.Bowl3 != 3 {
		t.Errorf("clone has wrong values: (%d,%d,%d)", clone.Bowl1, clone.Bowl2, clone.Bowl3)
	}
	
	// Modifying clone should not affect original
	clone.Bowl1 = 10
	if ps.Bowl1 != 5 {
		t.Errorf("modifying clone affected original")
	}
}
