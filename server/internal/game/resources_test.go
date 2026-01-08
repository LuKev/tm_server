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
			description:   "Player with 5/7/0 should be able to receive 2 power",
		},
		{
			name:          "Auren bug case - Bowl1=1, Bowl2=0",
			bowl1:         1,
			bowl2:         0,
			bowl3:         4,
			buildingValue: 2,
			expectedOffer: 2,
			description:   "Player with 1/0/4 should be able to receive 2 power (1 from Bowl1→Bowl2, then 1 from Bowl2→Bowl3)",
		},
		{
			name:          "Only Bowl2 available",
			bowl1:         0,
			bowl2:         3,
			bowl3:         2,
			buildingValue: 2,
			expectedOffer: 2,
			description:   "Player with 0/3/2 should be able to receive 2 power from Bowl2→Bowl3",
		},
		{
			name:          "Limited by Bowl1+Bowl2",
			bowl1:         1,
			bowl2:         1,
			bowl3:         0,
			buildingValue: 5,
			expectedOffer: 3,
			description:   "Player with 1/1/0 can receive max 3 (1*2 + 1)",
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
			expectedOffer: 1,
			description:   "Player with 0/1/4 can only receive 1 power",
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
