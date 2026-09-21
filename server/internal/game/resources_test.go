package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

// Rulebook p. 12: accept the entire offer, except for charging capacity or
// avoiding a negative score; the cost is actual power gained minus one.
func TestLeechRulebookAcceptanceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                                                   string
		vp, amount, b1, b2, b3, wantVP, wantB1, wantB2, wantB3 int
	}{
		{"three costs two", 20, 3, 0, 12, 0, 18, 0, 9, 3},
		{"zero VP gains one", 0, 3, 0, 12, 0, 0, 0, 11, 1},
		{"one VP gains two", 1, 3, 0, 12, 0, 0, 0, 10, 2},
		{"capacity caps cost", 20, 5, 1, 0, 11, 19, 0, 0, 12},
		{"full bowls cost nothing", 20, 3, 0, 0, 12, 20, 0, 0, 12},
		{"charge bowl one first", 20, 3, 2, 10, 0, 18, 0, 11, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := NewGameState()
			gs.AddPlayer("receiver", factions.NewWitches())
			gs.AddPlayer("source", factions.NewCultists())
			p := gs.GetPlayer("receiver")
			p.VictoryPoints = tc.vp
			p.Resources.Power = NewPowerSystem(tc.b1, tc.b2, tc.b3)
			gs.PendingLeechOffers[p.ID] = []*PowerLeechOffer{{Amount: tc.amount, FromPlayerID: "source", EventID: 1}}
			if err := gs.AcceptLeechOffer(p.ID, 0); err != nil {
				t.Fatal(err)
			}
			power := p.Resources.Power
			if p.VictoryPoints != tc.wantVP || power.Bowl1 != tc.wantB1 || power.Bowl2 != tc.wantB2 || power.Bowl3 != tc.wantB3 {
				t.Fatalf("got VP %d bowls %d/%d/%d, want VP %d bowls %d/%d/%d", p.VictoryPoints, power.Bowl1, power.Bowl2, power.Bowl3, tc.wantVP, tc.wantB1, tc.wantB2, tc.wantB3)
			}
			if len(gs.PendingLeechOffers[p.ID]) != 0 {
				t.Fatal("offer must resolve once without remainder")
			}
		})
	}
}

func TestLeechRejectsFragmentWithoutMutation(t *testing.T) {
	for _, amount := range []int{-1, 1, 2, 4} {
		gs := NewGameState()
		gs.AddPlayer("receiver", factions.NewWitches())
		gs.AddPlayer("source", factions.NewCultists())
		p := gs.GetPlayer("receiver")
		p.VictoryPoints = 20
		p.Resources.Power = NewPowerSystem(0, 12, 0)
		offer := &PowerLeechOffer{Amount: 3, FromPlayerID: "source", EventID: 1}
		gs.PendingLeechOffers[p.ID] = []*PowerLeechOffer{offer}
		gs.PendingCultistsLeech[1] = &CultistsLeechBonus{PlayerID: "source", OffersCreated: 1}
		action := NewAcceptPowerLeechAmountAction(p.ID, 0, amount)
		if err := action.Validate(gs); err == nil {
			t.Errorf("Validate accepted amount %d", amount)
		}
		if err := action.Execute(gs); err == nil {
			t.Errorf("Execute accepted amount %d", amount)
		}
		if p.VictoryPoints != 20 || p.Resources.Power.Bowl2 != 12 || len(gs.PendingLeechOffers[p.ID]) != 1 || offer.Amount != 3 || gs.PendingCultistsLeech[1].ResolvedCount != 0 {
			t.Fatalf("rejected amount %d mutated state", amount)
		}
	}
}

func TestLeechCappedAcceptanceResolvesCultistsOnce(t *testing.T) {
	for _, explicit := range []int{0, 2} {
		gs := NewGameState()
		gs.AddPlayer("receiver", factions.NewWitches())
		gs.AddPlayer("source", factions.NewCultists())
		p := gs.GetPlayer("receiver")
		p.VictoryPoints = 1
		p.Resources.Power = NewPowerSystem(0, 12, 0)
		gs.PendingLeechOffers[p.ID] = []*PowerLeechOffer{{Amount: 3, FromPlayerID: "source", EventID: 1}}
		bonus := &CultistsLeechBonus{PlayerID: "source", OffersCreated: 1}
		gs.PendingCultistsLeech[1] = bonus
		if err := NewAcceptPowerLeechAmountAction(p.ID, 0, explicit).Execute(gs); err != nil {
			t.Fatal(err)
		}
		if bonus.ResolvedCount != 1 || bonus.AcceptedCount != 1 || len(gs.PendingLeechOffers[p.ID]) != 0 {
			t.Fatalf("original offer not resolved exactly once: %+v", bonus)
		}
		if gs.PendingCultistsCultSelection == nil || gs.PendingCultistsCultSelection.PlayerID != "source" {
			t.Fatal("accepted offer must yield one Cultists cult choice")
		}
		if err := NewAcceptPowerLeechAction(p.ID, 0).Execute(gs); err == nil {
			t.Fatal("resolved offer was accepted again")
		}
	}
}

func TestLeechDistinctOffersRecomputeCapacityAndAffordability(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("receiver", factions.NewWitches())
	gs.AddPlayer("source", factions.NewCultists())
	gs.AddPlayer("other", factions.NewNomads())
	p := gs.GetPlayer("receiver")
	p.VictoryPoints = 1
	p.Resources.Power = NewPowerSystem(0, 3, 9)
	gs.PendingLeechOffers[p.ID] = []*PowerLeechOffer{
		{Amount: 3, FromPlayerID: "source", EventID: 1},
		{Amount: 3, FromPlayerID: "other", EventID: 2},
	}
	if err := gs.AcceptLeechOffer(p.ID, 0); err != nil {
		t.Fatal(err)
	}
	if p.VictoryPoints != 0 || p.Resources.Power.Bowl3 != 11 || len(gs.PendingLeechOffers[p.ID]) != 1 || gs.PendingLeechOffers[p.ID][0].EventID != 2 {
		t.Fatal("first offer corrupted unrelated source offer")
	}
	if err := gs.AcceptLeechOffer(p.ID, 0); err != nil {
		t.Fatal(err)
	}
	if p.VictoryPoints != 0 || p.Resources.Power.Bowl3 != 12 || len(gs.PendingLeechOffers[p.ID]) != 0 {
		t.Fatal("second original offer must charge remaining one power for zero VP")
	}
}

func TestLeechResourceAndVPBoundsExhaustive(t *testing.T) {
	// Independent arithmetic oracle: each token in bowl I can move twice;
	// each in bowl II once, and one gained power is free for base factions.
	for b1 := 0; b1 <= 12; b1++ {
		for b2 := 0; b2 <= 12-b1; b2++ {
			for vp := 0; vp <= 5; vp++ {
				for offered := 1; offered <= 12; offered++ {
					p := &Player{VictoryPoints: vp, Resources: &ResourcePool{Power: NewPowerSystem(b1, b2, 12-b1-b2)}}
					gain := offered
					if gain > 2*b1+b2 {
						gain = 2*b1 + b2
					}
					if gain > vp+1 {
						gain = vp + 1
					}
					cost := gain - 1
					if cost < 0 {
						cost = 0
					}
					p.AcceptPowerLeech(&PowerLeechOffer{Amount: offered})
					power := p.Resources.Power
					if p.VictoryPoints != vp-cost || p.VictoryPoints < 0 || power.Bowl1+power.Bowl2+power.Bowl3 != 12 || 2*power.Bowl1+power.Bowl2 != 2*b1+b2-gain {
						t.Fatalf("bounds/accounting failed for bowls %d/%d VP %d offer %d: %+v VP %d", b1, b2, vp, offered, power, p.VictoryPoints)
					}
				}
			}
		}
	}
}

func TestLeechNoCapacityDoesNotRewardCultists(t *testing.T) {
	for _, accept := range []bool{false, true} {
		gs := NewGameState()
		gs.AddPlayer("receiver", factions.NewWitches())
		gs.AddPlayer("source", factions.NewCultists())
		p := gs.GetPlayer("receiver")
		p.Resources.Power = NewPowerSystem(0, 0, 12)
		source := gs.GetPlayer("source")
		source.Resources.Power = NewPowerSystem(0, 12, 0)
		gs.PendingLeechOffers[p.ID] = []*PowerLeechOffer{{Amount: 3, FromPlayerID: "source", EventID: 1}}
		gs.PendingCultistsLeech[1] = &CultistsLeechBonus{PlayerID: "source", OffersCreated: 1}
		var err error
		if accept {
			err = gs.AcceptLeechOffer(p.ID, 0)
		} else {
			err = gs.DeclineLeechOffer(p.ID, 0)
		}
		if err != nil {
			t.Fatal(err)
		}
		if source.Resources.Power.Bowl2 != 12 || gs.PendingCultistsCultSelection != nil || len(gs.PendingCultistsLeech) != 0 {
			t.Fatal("no-capacity response must resolve without Cultists reward")
		}
	}
}

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
