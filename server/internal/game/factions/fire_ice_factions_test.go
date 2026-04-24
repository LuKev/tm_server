package factions

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestFireIceFactionBasics(t *testing.T) {
	tests := []struct {
		name      string
		faction   Faction
		fType     models.FactionType
		home      models.TerrainType
		resources Resources
		cults     CultPositions
	}{
		{
			name:      "Ice Maidens",
			faction:   NewIceMaidens(),
			fType:     models.FactionIceMaidens,
			home:      models.TerrainIce,
			resources: Resources{Coins: 15, Workers: 3, Power1: 6, Power2: 6},
			cults:     CultPositions{Water: 1, Air: 1},
		},
		{
			name:      "Yetis",
			faction:   NewYetis(),
			fType:     models.FactionYetis,
			home:      models.TerrainIce,
			resources: Resources{Coins: 15, Workers: 3, Power1: 0, Power2: 12},
			cults:     CultPositions{Earth: 1, Air: 1},
		},
		{
			name:      "Dragonlords",
			faction:   NewDragonlords(),
			fType:     models.FactionDragonlords,
			home:      models.TerrainVolcano,
			resources: Resources{Coins: 15, Workers: 3, Power1: 4, Power2: 4},
			cults:     CultPositions{Fire: 2},
		},
		{
			name:      "Acolytes",
			faction:   NewAcolytes(),
			fType:     models.FactionAcolytes,
			home:      models.TerrainVolcano,
			resources: Resources{Coins: 15, Workers: 3, Power1: 6, Power2: 6},
			cults:     CultPositions{Fire: 3, Water: 3, Earth: 3, Air: 3},
		},
		{
			name:      "Riverwalkers",
			faction:   NewRiverwalkers(),
			fType:     models.FactionRiverwalkers,
			home:      models.TerrainTypeUnknown,
			resources: Resources{Coins: 15, Workers: 3, Power1: 10, Power2: 2},
			cults:     CultPositions{Fire: 1, Air: 1},
		},
		{
			name:      "Shapeshifters",
			faction:   NewShapeshifters(),
			fType:     models.FactionShapeshifters,
			home:      models.TerrainPlains,
			resources: Resources{Coins: 15, Workers: 3, Power1: 4, Power2: 4},
			cults:     CultPositions{Fire: 1, Water: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.faction.GetType(); got != tt.fType {
				t.Fatalf("type = %v, want %v", got, tt.fType)
			}
			if got := tt.faction.GetHomeTerrain(); got != tt.home {
				t.Fatalf("home = %v, want %v", got, tt.home)
			}
			if got := tt.faction.GetStartingResources(); got != tt.resources {
				t.Fatalf("resources = %+v, want %+v", got, tt.resources)
			}
			if got := tt.faction.GetStartingCultPositions(); got != tt.cults {
				t.Fatalf("cults = %+v, want %+v", got, tt.cults)
			}
		})
	}
}

func TestFireIceFactionIncomeAndCosts(t *testing.T) {
	if got := NewIceMaidens().GetDiggingCost(0); got != (Cost{Coins: 5, Workers: 1, Priests: 1}) {
		t.Fatalf("ice maidens digging cost = %+v", got)
	}
	if got := NewYetis().GetTradingHouseIncome(4); got != (Income{Coins: 8, Power: 8}) {
		t.Fatalf("yetis trading house income = %+v", got)
	}
	if got := NewYetis().GetDwellingIncome(8); got != (Income{Workers: 8}) {
		t.Fatalf("yetis dwelling income at 8 = %+v", got)
	}
	if got := NewYetis().GetTempleIncome(2); got != (Income{Priests: 1, Power: 5}) {
		t.Fatalf("yetis temple income at 2 = %+v", got)
	}
	if got := NewAcolytes().GetBaseFactionIncome(); got != (Income{}) {
		t.Fatalf("acolytes base income = %+v", got)
	}
	if got := NewDragonlords().GetDwellingIncome(4); got != (Income{Workers: 3}) {
		t.Fatalf("dragonlords dwelling income at 4 = %+v", got)
	}
	if got := NewRiverwalkers().GetTempleIncome(2); got != (Income{Priests: 1, Power: 5}) {
		t.Fatalf("riverwalkers temple income at 2 = %+v", got)
	}
	if got := NewFirewalkers().GetBaseFactionIncome(); got != (Income{}) {
		t.Fatalf("firewalkers base income = %+v", got)
	}
	if got := NewFirewalkers().GetDwellingIncome(4); got != (Income{Workers: 3}) {
		t.Fatalf("firewalkers dwelling income at 4 = %+v", got)
	}
	if got := NewFirewalkers().GetDwellingIncome(8); got != (Income{Workers: 6}) {
		t.Fatalf("firewalkers dwelling income at 8 = %+v", got)
	}
	if got := NewFirewalkers().GetTempleIncome(2); got != (Income{Priests: 1, VictoryPoints: 2}) {
		t.Fatalf("firewalkers temple income at 2 = %+v", got)
	}
	if got := NewFirewalkers().GetStrongholdIncome(); got != (Income{Power: 2}) {
		t.Fatalf("firewalkers stronghold income = %+v", got)
	}
	if got := NewFirewalkers().GetStrongholdCost(); got != (Cost{Coins: 8, Workers: 4}) {
		t.Fatalf("firewalkers stronghold cost = %+v", got)
	}
	if got := NewSelkies().GetStrongholdIncome(); got != (Income{Power: 4}) {
		t.Fatalf("selkies stronghold income = %+v", got)
	}
}

func TestRequestedFanFactionBasics(t *testing.T) {
	tests := []struct {
		name    string
		faction Faction
		fType   models.FactionType
		home    models.TerrainType
	}{
		{name: "Firewalkers", faction: NewFirewalkers(), fType: models.FactionFirewalkers, home: models.TerrainVolcano},
		{name: "Selkies", faction: NewSelkies(), fType: models.FactionSelkies, home: models.TerrainIce},
		{name: "Snow Shamans", faction: NewSnowShamans(), fType: models.FactionSnowShamans, home: models.TerrainIce},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.faction.GetType(); got != tt.fType {
				t.Fatalf("type = %v, want %v", got, tt.fType)
			}
			if got := tt.faction.GetHomeTerrain(); got != tt.home {
				t.Fatalf("home = %v, want %v", got, tt.home)
			}
			if got := tt.faction.GetStartingResources(); got != (Resources{Coins: 15, Workers: 3, Power1: 5, Power2: 7}) {
				t.Fatalf("resources = %+v", got)
			}
		})
	}
}
