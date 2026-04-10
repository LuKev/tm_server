package factions

import "testing"

func TestFanFactionIncome_Archivists(t *testing.T) {
	f := NewArchivists()

	if got := f.GetBaseFactionIncome(); got != (Income{Workers: 2}) {
		t.Fatalf("expected archivists base income %+v, got %+v", Income{Workers: 2}, got)
	}

	if got := f.GetDwellingIncome(4); got != (Income{Workers: 3}) {
		t.Fatalf("expected archivists dwelling income after 4 dwellings %+v, got %+v", Income{Workers: 3}, got)
	}

	if got := f.GetDwellingIncome(8); got != (Income{Workers: 6}) {
		t.Fatalf("expected archivists dwelling income after 8 dwellings %+v, got %+v", Income{Workers: 6}, got)
	}
}

func TestFanFactionIncome_TheEnlightened(t *testing.T) {
	f := NewTheEnlightened()

	if got := f.GetBaseFactionIncome(); got != (Income{Power: 3}) {
		t.Fatalf("expected enlightened base income %+v, got %+v", Income{Power: 3}, got)
	}

	if got := f.GetDwellingIncome(3); got != (Income{Power: 7}) {
		t.Fatalf("expected enlightened dwelling income after 3 dwellings %+v, got %+v", Income{Power: 7}, got)
	}

	if got := f.GetStrongholdIncome(); got != (Income{}) {
		t.Fatalf("expected enlightened stronghold income %+v, got %+v", Income{}, got)
	}

	if got := f.GetTempleCost(); got != (Cost{Coins: 5, Priests: 1}) {
		t.Fatalf("expected enlightened temple cost %+v, got %+v", Cost{Coins: 5, Priests: 1}, got)
	}

	if got := f.GetSanctuaryCost(); got != (Cost{Coins: 6, Priests: 2}) {
		t.Fatalf("expected enlightened sanctuary cost %+v, got %+v", Cost{Coins: 6, Priests: 2}, got)
	}

	if got := f.GetStrongholdCost(); got != (Cost{Coins: 4, Priests: 1}) {
		t.Fatalf("expected enlightened stronghold cost %+v, got %+v", Cost{Coins: 4, Priests: 1}, got)
	}
}

func TestFanFactionCosts_UserCorrections(t *testing.T) {
	tests := []struct {
		name           string
		faction        Faction
		templeCost     *Cost
		sanctuaryCost  *Cost
		strongholdCost *Cost
	}{
		{
			name:           "Children of the Wyrm",
			faction:        NewChildrenOfTheWyrm(),
			sanctuaryCost:  &Cost{Coins: 5, Workers: 4},
			strongholdCost: &Cost{Coins: 10, Workers: 4},
		},
		{
			name:           "Goblins",
			faction:        NewGoblins(),
			templeCost:     &Cost{Coins: 6, Workers: 2},
			sanctuaryCost:  &Cost{Coins: 6, Workers: 4},
			strongholdCost: &Cost{Coins: 6, Workers: 4},
		},
		{
			name:           "Time Travelers",
			faction:        NewTimeTravelers(),
			sanctuaryCost:  &Cost{Coins: 8, Workers: 4},
			strongholdCost: &Cost{Coins: 8, Workers: 4},
		},
		{
			name:           "Djinni",
			faction:        NewDjinni(),
			strongholdCost: &Cost{Coins: 6, Workers: 3},
		},
		{
			name:           "Chash Dallah",
			faction:        NewChashDallah(),
			strongholdCost: &Cost{Coins: 4, Workers: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.templeCost != nil {
				if got := tt.faction.GetTempleCost(); got != *tt.templeCost {
					t.Fatalf("expected temple cost %+v, got %+v", *tt.templeCost, got)
				}
			}
			if tt.sanctuaryCost != nil {
				if got := tt.faction.GetSanctuaryCost(); got != *tt.sanctuaryCost {
					t.Fatalf("expected sanctuary cost %+v, got %+v", *tt.sanctuaryCost, got)
				}
			}
			if tt.strongholdCost != nil {
				if got := tt.faction.GetStrongholdCost(); got != *tt.strongholdCost {
					t.Fatalf("expected stronghold cost %+v, got %+v", *tt.strongholdCost, got)
				}
			}
		})
	}
}

func TestFanFactionIncome_Treasurers(t *testing.T) {
	f := NewTreasurers()

	if got := f.GetBaseFactionIncome(); got != (Income{}) {
		t.Fatalf("expected treasurers base income %+v, got %+v", Income{}, got)
	}

	if got := f.GetDwellingIncome(8); got != (Income{Workers: 5}) {
		t.Fatalf("expected treasurers dwelling income %+v, got %+v", Income{Workers: 5}, got)
	}

	if got := f.GetTempleIncome(2); got != (Income{Priests: 1, Power: 4}) {
		t.Fatalf("expected treasurers temple income %+v, got %+v", Income{Priests: 1, Power: 4}, got)
	}

	if got := f.GetStrongholdIncome(); got != (Income{Power: 4}) {
		t.Fatalf("expected treasurers stronghold income %+v, got %+v", Income{Power: 4}, got)
	}
}
