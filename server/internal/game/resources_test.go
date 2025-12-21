package game

import (
	"github.com/lukev/tm_server/internal/game/factions"
	"testing"
)

func TestNewResourcePool(t *testing.T) {
	startingRes := factions.Resources{
		Coins:   15,
		Workers: 3,
		Priests: 0,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	}

	rp := NewResourcePool(startingRes)

	if rp.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", rp.Coins)
	}
	if rp.Workers != 3 {
		t.Errorf("expected 3 workers, got %d", rp.Workers)
	}
	if rp.Priests != 0 {
		t.Errorf("expected 0 priests, got %d", rp.Priests)
	}
	if rp.Power.Bowl1 != 5 || rp.Power.Bowl2 != 7 || rp.Power.Bowl3 != 0 {
		t.Errorf("expected power (5,7,0), got (%d,%d,%d)", rp.Power.Bowl1, rp.Power.Bowl2, rp.Power.Bowl3)
	}
}

func TestCanAfford(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  0,
		Power2:  0,
		Power3:  6,
	})

	// Can afford
	cost1 := factions.Cost{Coins: 5, Workers: 3, Priests: 1, Power: 4}
	if !rp.CanAfford(cost1) {
		t.Errorf("should be able to afford cost1")
	}

	// Cannot afford - not enough coins
	cost2 := factions.Cost{Coins: 15, Workers: 3, Priests: 1, Power: 4}
	if rp.CanAfford(cost2) {
		t.Errorf("should not be able to afford cost2 (not enough coins)")
	}

	// Cannot afford - not enough power
	cost3 := factions.Cost{Coins: 5, Workers: 3, Priests: 1, Power: 10}
	if rp.CanAfford(cost3) {
		t.Errorf("should not be able to afford cost3 (not enough power)")
	}
}

func TestSpend(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  0,
		Power2:  0,
		Power3:  6,
	})

	cost := factions.Cost{Coins: 5, Workers: 3, Priests: 1, Power: 4}
	err := rp.Spend(cost)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Coins != 5 {
		t.Errorf("expected 5 coins remaining, got %d", rp.Coins)
	}
	if rp.Workers != 2 {
		t.Errorf("expected 2 workers remaining, got %d", rp.Workers)
	}
	if rp.Priests != 1 {
		t.Errorf("expected 1 priest remaining, got %d", rp.Priests)
	}
	if rp.Power.Bowl3 != 2 {
		t.Errorf("expected 2 power in bowl 3, got %d", rp.Power.Bowl3)
	}
	// Spent power should return to bowl 1
	if rp.Power.Bowl1 != 4 {
		t.Errorf("expected 4 power in bowl 1 (returned), got %d", rp.Power.Bowl1)
	}
}

func TestSpendInsufficientResources(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   5,
		Workers: 2,
		Priests: 0,
		Power1:  0,
		Power2:  0,
		Power3:  3,
	})

	cost := factions.Cost{Coins: 10, Workers: 3, Priests: 1, Power: 2}
	err := rp.Spend(cost)

	if err == nil {
		t.Errorf("expected error when spending more than available")
	}

	// Resources should be unchanged
	if rp.Coins != 5 {
		t.Errorf("coins should be unchanged, got %d", rp.Coins)
	}
}

func TestGain(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	})

	rp.Gain(5, 3, 1)

	if rp.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", rp.Coins)
	}
	if rp.Workers != 8 {
		t.Errorf("expected 8 workers, got %d", rp.Workers)
	}
	if rp.Priests != 3 {
		t.Errorf("expected 3 priests, got %d", rp.Priests)
	}
}

func TestResourcePool_GainPower(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	})

	gained := rp.GainPower(3)

	if gained != 3 {
		t.Errorf("expected to gain 3 power, got %d", gained)
	}
	// Power should move from bowl 1 to bowl 2
	if rp.Power.Bowl1 != 2 || rp.Power.Bowl2 != 10 {
		t.Errorf("expected power (2,10,0), got (%d,%d,%d)", rp.Power.Bowl1, rp.Power.Bowl2, rp.Power.Bowl3)
	}
}

func TestConvertPriestToWorker(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 3,
		Priests: 3,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	})

	// Convert 2 priests to 2 workers
	err := rp.ConvertPriestToWorker(2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Workers != 5 {
		t.Errorf("expected 5 workers, got %d", rp.Workers)
	}
	if rp.Priests != 1 {
		t.Errorf("expected 1 priest remaining, got %d", rp.Priests)
	}
}

func TestConvertWorkerToCoin(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 0,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	})

	err := rp.ConvertWorkerToCoin(3)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Coins != 13 {
		t.Errorf("expected 13 coins, got %d", rp.Coins)
	}
	if rp.Workers != 2 {
		t.Errorf("expected 2 workers remaining, got %d", rp.Workers)
	}
}

func TestConvertPowerToCoins(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 3,
		Priests: 0,
		Power1:  0,
		Power2:  0,
		Power3:  8,
	})

	err := rp.ConvertPowerToCoins(5)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Coins != 15 {
		t.Errorf("expected 15 coins, got %d", rp.Coins)
	}
	if rp.Power.Bowl3 != 3 {
		t.Errorf("expected 3 power in bowl 3, got %d", rp.Power.Bowl3)
	}
	if rp.Power.Bowl1 != 5 {
		t.Errorf("expected 5 power in bowl 1 (returned), got %d", rp.Power.Bowl1)
	}
}

func TestConvertPowerToWorkers(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 3,
		Priests: 0,
		Power1:  0,
		Power2:  0,
		Power3:  9,
	})

	// Convert 6 power to 2 workers
	err := rp.ConvertPowerToWorkers(2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Workers != 5 {
		t.Errorf("expected 5 workers, got %d", rp.Workers)
	}
	if rp.Power.Bowl3 != 3 {
		t.Errorf("expected 3 power in bowl 3, got %d", rp.Power.Bowl3)
	}
}

func TestConvertPowerToPriests(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 3,
		Priests: 1,
		Power1:  0,
		Power2:  0,
		Power3:  10,
	})

	// Convert 10 power to 2 priests (5 power per priest)
	err := rp.ConvertPowerToPriests(2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Priests != 3 {
		t.Errorf("expected 3 priests, got %d", rp.Priests)
	}
	if rp.Power.Bowl3 != 0 {
		t.Errorf("expected 0 power in bowl 3, got %d", rp.Power.Bowl3)
	}
	if rp.Power.Bowl1 != 10 {
		t.Errorf("expected 10 power in bowl 1 (returned), got %d", rp.Power.Bowl1)
	}
}

func TestResourcePool_BurnPower(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 3,
		Priests: 0,
		Power1:  0,
		Power2:  10,
		Power3:  0,
	})

	// Burn to get 3 power in bowl 3 (costs 6 from bowl 2)
	err := rp.BurnPower(3)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Power.Bowl2 != 4 {
		t.Errorf("expected 4 power in bowl 2, got %d", rp.Power.Bowl2)
	}
	if rp.Power.Bowl3 != 3 {
		t.Errorf("expected 3 power in bowl 3, got %d", rp.Power.Bowl3)
	}
}

func TestResourcePool_Clone(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  5,
		Power2:  7,
		Power3:  3,
	})

	clone := rp.Clone()

	// Clone should have same values
	if clone.Coins != 10 || clone.Workers != 5 || clone.Priests != 2 {
		t.Errorf("clone has wrong resources")
	}
	if clone.Power.Bowl1 != 5 || clone.Power.Bowl2 != 7 || clone.Power.Bowl3 != 3 {
		t.Errorf("clone has wrong power")
	}

	// Modifying clone should not affect original
	clone.Coins = 20
	clone.Power.Bowl1 = 10
	if rp.Coins != 10 {
		t.Errorf("modifying clone affected original coins")
	}
	if rp.Power.Bowl1 != 5 {
		t.Errorf("modifying clone affected original power")
	}
}

func TestToResources(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  5,
		Power2:  7,
		Power3:  3,
	})

	res := rp.ToResources()

	if res.Coins != 10 || res.Workers != 5 || res.Priests != 2 {
		t.Errorf("ToResources returned wrong values")
	}
	if res.Power1 != 5 || res.Power2 != 7 || res.Power3 != 3 {
		t.Errorf("ToResources returned wrong power values")
	}
}

func TestNewPowerLeechOffer(t *testing.T) {
	// Player with plenty of power capacity
	powerSystem := NewPowerSystem(5, 7, 0)

	// Dwelling (value 1)
	offer1 := NewPowerLeechOffer(1, "player1", powerSystem)
	if offer1.Amount != 1 || offer1.VPCost != 0 {
		t.Errorf("dwelling leech should be 1 power, 0 VP cost, got %d power, %d VP", offer1.Amount, offer1.VPCost)
	}

	// Trading House (value 2)
	offer2 := NewPowerLeechOffer(2, "player1", powerSystem)
	if offer2.Amount != 2 || offer2.VPCost != 1 {
		t.Errorf("trading house leech should be 2 power, 1 VP cost, got %d power, %d VP", offer2.Amount, offer2.VPCost)
	}

	// Stronghold (value 3)
	offer3 := NewPowerLeechOffer(3, "player1", powerSystem)
	if offer3.Amount != 3 || offer3.VPCost != 2 {
		t.Errorf("stronghold leech should be 3 power, 2 VP cost, got %d power, %d VP", offer3.Amount, offer3.VPCost)
	}

	// Invalid value
	offer4 := NewPowerLeechOffer(0, "player1", powerSystem)
	if offer4 != nil {
		t.Errorf("expected nil offer for 0 value")
	}
}

func TestNewPowerLeechOfferLimited(t *testing.T) {
	// Player with limited power capacity: 0 in bowl 1, 3 in bowl 2, 9 in bowl 3
	// Can only gain 3 power maximum
	powerSystem := NewPowerSystem(0, 3, 9)

	// Building value 5, but can only gain 3
	offer := NewPowerLeechOffer(5, "player1", powerSystem)
	if offer.Amount != 3 {
		t.Errorf("expected amount 3 (limited by capacity), got %d", offer.Amount)
	}
	if offer.VPCost != 2 {
		t.Errorf("expected VP cost 2 (amount - 1), got %d", offer.VPCost)
	}
}

func TestNewPowerLeechOfferNoPower(t *testing.T) {
	// Player with no power capacity: all power in bowl 3
	powerSystem := NewPowerSystem(0, 0, 12)

	// Building value 3, but cannot gain any power
	offer := NewPowerLeechOffer(3, "player1", powerSystem)
	if offer != nil {
		t.Errorf("expected nil offer when player cannot gain power")
	}
}

func TestAcceptPowerLeech(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	})

	offer := NewPowerLeechOffer(2, "player1", rp.Power)
	vpCost := rp.AcceptPowerLeech(offer)

	if vpCost != 1 {
		t.Errorf("expected VP cost of 1, got %d", vpCost)
	}

	// Power should be gained (moved from bowl 1 to bowl 2)
	if rp.Power.Bowl1 != 3 || rp.Power.Bowl2 != 9 {
		t.Errorf("expected power (3,9,0) after leech, got (%d,%d,%d)", rp.Power.Bowl1, rp.Power.Bowl2, rp.Power.Bowl3)
	}
}

func TestDeclinePowerLeech(t *testing.T) {
	rp := NewResourcePool(factions.Resources{
		Coins:   10,
		Workers: 5,
		Priests: 2,
		Power1:  5,
		Power2:  7,
		Power3:  0,
	})

	offer := NewPowerLeechOffer(2, "player1", rp.Power)
	rp.DeclinePowerLeech(offer)

	// Resources should be unchanged
	if rp.Power.Bowl1 != 5 || rp.Power.Bowl2 != 7 {
		t.Errorf("power should be unchanged after declining leech")
	}
}
