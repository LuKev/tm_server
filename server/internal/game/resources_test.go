package game

import (
	"testing"
)

func TestNewPowerLeechOffer_CapacityCalculation(t *testing.T) {
	tests := []struct {
		name          string
		bowl1         int
		bowl2         int
		bowl3         int
		buildingValue int
		expectedOffer int
		description   string
	}{
		{
			name:          "Standard case - full capacity",
			bowl1:         5,
			bowl2:         7,
			bowl3:         0,
			buildingValue: 2,
			expectedOffer: 2,
			description:   "Offer amount is the (uncapped) leech amount; capacity limits apply at acceptance time",
		},
		{
			name:          "Auren bug case - Bowl1=1, Bowl2=0",
			bowl1:         1,
			bowl2:         0,
			bowl3:         4,
			buildingValue: 2,
			expectedOffer: 2,
			description:   "Offer is still created when there is at least some charging capacity",
		},
		{
			name:          "Only Bowl2 available",
			bowl1:         0,
			bowl2:         3,
			bowl3:         2,
			buildingValue: 2,
			expectedOffer: 2,
			description:   "Offer amount is not capped by remaining capacity",
		},
		{
			name:          "Limited by Bowl1+Bowl2",
			bowl1:         1,
			bowl2:         1,
			bowl3:         0,
			buildingValue: 5,
			expectedOffer: 5,
			description:   "Offer is not capped; actual power gained is limited by current bowls when accepting",
		},
		{
			name:          "No capacity",
			bowl1:         0,
			bowl2:         0,
			bowl3:         12,
			buildingValue: 2,
			expectedOffer: 0,
			description:   "Player with 0/0/12 (full Bowl3) cannot receive power",
		},
		{
			name:          "Capacity of 1",
			bowl1:         0,
			bowl2:         1,
			bowl3:         4,
			buildingValue: 2,
			expectedOffer: 2,
			description:   "Offer amount is not capped; acceptance will only gain up to 1 power here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			power := NewPowerSystem(tt.bowl1, tt.bowl2, tt.bowl3)
			offer := NewPowerLeechOffer(tt.buildingValue, "testPlayer", power)

			if tt.expectedOffer == 0 {
				if offer != nil {
					t.Errorf("%s: expected nil offer, got %d", tt.description, offer.Amount)
				}
			} else {
				if offer == nil {
					t.Errorf("%s: expected offer of %d, got nil", tt.description, tt.expectedOffer)
				} else if offer.Amount != tt.expectedOffer {
					t.Errorf("%s: expected offer of %d, got %d", tt.description, tt.expectedOffer, offer.Amount)
				}
			}
		})
	}
}

func TestAcceptPowerLeech_ChargesByActualGain(t *testing.T) {
	t.Run("no capacity at accept time costs zero VP", func(t *testing.T) {
		rp := &ResourcePool{
			Power: NewPowerSystem(0, 0, 12),
		}
		offer := &PowerLeechOffer{
			Amount:       2,
			VPCost:       1, // stale snapshot cost from offer creation
			FromPlayerID: "neighbor",
		}

		vpCost := rp.AcceptPowerLeech(offer)
		if vpCost != 0 {
			t.Fatalf("expected vp cost 0 when no power can be gained, got %d", vpCost)
		}
		if rp.Power.Bowl1 != 0 || rp.Power.Bowl2 != 0 || rp.Power.Bowl3 != 12 {
			t.Fatalf("expected power bowls unchanged at 0/0/12, got %d/%d/%d", rp.Power.Bowl1, rp.Power.Bowl2, rp.Power.Bowl3)
		}
	})

	t.Run("partial gain recomputes VP cost", func(t *testing.T) {
		rp := &ResourcePool{
			Power: NewPowerSystem(0, 1, 11),
		}
		offer := &PowerLeechOffer{
			Amount:       2,
			VPCost:       1, // stale snapshot cost from offer creation
			FromPlayerID: "neighbor",
		}

		vpCost := rp.AcceptPowerLeech(offer)
		if vpCost != 0 {
			t.Fatalf("expected vp cost 0 when only 1 power is gained, got %d", vpCost)
		}
		if rp.Power.Bowl1 != 0 || rp.Power.Bowl2 != 0 || rp.Power.Bowl3 != 12 {
			t.Fatalf("expected final bowls 0/0/12 after gaining 1 power, got %d/%d/%d", rp.Power.Bowl1, rp.Power.Bowl2, rp.Power.Bowl3)
		}
	})
}
