package factions

import "github.com/lukev/tm_server/internal/models"

type configuredFaction struct {
	BaseFaction
	startingCult       CultPositions
	baseIncome         Income
	dwellingIncomeSeq  []Income
	tradingIncomeSeq   []Income
	templeIncomeSeq    []Income
	sanctuaryIncome    Income
	strongholdIncome   Income
	fixedTerraformCost *int
	dwellingCost       *Cost
	tradingHouseCost   *Cost
	templeCost         *Cost
	sanctuaryCost      *Cost
	strongholdCost     *Cost
}

func (f *configuredFaction) GetStartingCultPositions() CultPositions {
	return f.startingCult
}

func (f *configuredFaction) GetBaseFactionIncome() Income {
	return f.baseIncome
}

func (f *configuredFaction) GetDwellingIncome(dwellingCount int) Income {
	return incomeFromSequence(f.dwellingIncomeSeq, dwellingCount)
}

func (f *configuredFaction) GetTradingHouseIncome(tradingHouseCount int) Income {
	return incomeFromSequence(f.tradingIncomeSeq, tradingHouseCount)
}

func (f *configuredFaction) GetTempleIncome(templeCount int) Income {
	return incomeFromSequence(f.templeIncomeSeq, templeCount)
}

func (f *configuredFaction) GetSanctuaryIncome() Income {
	return f.sanctuaryIncome
}

func (f *configuredFaction) GetStrongholdIncome() Income {
	return f.strongholdIncome
}

func (f *configuredFaction) GetTerraformCost(distance int) int {
	if f.fixedTerraformCost != nil {
		return distance * *f.fixedTerraformCost
	}
	return f.BaseFaction.GetTerraformCost(distance)
}

func (f *configuredFaction) GetDwellingCost() Cost {
	if f.dwellingCost != nil {
		return *f.dwellingCost
	}
	return f.BaseFaction.GetDwellingCost()
}

func (f *configuredFaction) GetTradingHouseCost() Cost {
	if f.tradingHouseCost != nil {
		return *f.tradingHouseCost
	}
	return f.BaseFaction.GetTradingHouseCost()
}

func (f *configuredFaction) GetTempleCost() Cost {
	if f.templeCost != nil {
		return *f.templeCost
	}
	return f.BaseFaction.GetTempleCost()
}

func (f *configuredFaction) GetSanctuaryCost() Cost {
	if f.sanctuaryCost != nil {
		return *f.sanctuaryCost
	}
	return f.BaseFaction.GetSanctuaryCost()
}

func (f *configuredFaction) GetStrongholdCost() Cost {
	if f.strongholdCost != nil {
		return *f.strongholdCost
	}
	return f.BaseFaction.GetStrongholdCost()
}

func incomeFromSequence(seq []Income, count int) Income {
	total := Income{}
	for i := 0; i < count && i < len(seq); i++ {
		total.Coins += seq[i].Coins
		total.Workers += seq[i].Workers
		total.Priests += seq[i].Priests
		total.Power += seq[i].Power
	}
	return total
}

func repeatedIncome(n int, income Income) []Income {
	result := make([]Income, n)
	for i := range result {
		result[i] = income
	}
	return result
}

func copyIncomeSeq(seq []Income) []Income {
	result := make([]Income, len(seq))
	copy(result, seq)
	return result
}

var (
	standardDwellingIncomeSeq = []Income{
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{},
	}
	standardTradingIncomeSeq = []Income{
		{Coins: 2, Power: 1},
		{Coins: 2, Power: 1},
		{Coins: 2, Power: 2},
		{Coins: 2, Power: 2},
	}
	standardTempleIncomeSeq = []Income{
		{Priests: 1},
		{Priests: 1},
		{Priests: 1},
	}
)

func newConfiguredFaction(
	factionType models.FactionType,
	home models.TerrainType,
	startingRes Resources,
	startingCult CultPositions,
	baseIncome Income,
	dwellingSeq []Income,
	tradingSeq []Income,
	templeSeq []Income,
	sanctuaryIncome Income,
	strongholdIncome Income,
	fixedTerraformCost *int,
	dwellingCost *Cost,
	tradingHouseCost *Cost,
	templeCost *Cost,
	sanctuaryCost *Cost,
	strongholdCost *Cost,
) *configuredFaction {
	return &configuredFaction{
		BaseFaction: BaseFaction{
			Type:         factionType,
			HomeTerrain:  home,
			StartingRes:  startingRes,
			DiggingLevel: 0,
		},
		startingCult:       startingCult,
		baseIncome:         baseIncome,
		dwellingIncomeSeq:  copyIncomeSeq(dwellingSeq),
		tradingIncomeSeq:   copyIncomeSeq(tradingSeq),
		templeIncomeSeq:    copyIncomeSeq(templeSeq),
		sanctuaryIncome:    sanctuaryIncome,
		strongholdIncome:   strongholdIncome,
		fixedTerraformCost: fixedTerraformCost,
		dwellingCost:       dwellingCost,
		tradingHouseCost:   tradingHouseCost,
		templeCost:         templeCost,
		sanctuaryCost:      sanctuaryCost,
		strongholdCost:     strongholdCost,
	}
}

func fanFactionStartingResources(workers, coins, power1, power2 int) Resources {
	return Resources{
		Coins:   coins,
		Workers: workers,
		Priests: 0,
		Power1:  power1,
		Power2:  power2,
		Power3:  0,
	}
}

func standardFanFactionStartingResources(workers, coins int) Resources {
	return fanFactionStartingResources(workers, coins, 5, 7)
}

func fixedTerraformCost(perSpade int) *int {
	return &perSpade
}

func NewArchitects() Faction {
	return newConfiguredFaction(
		models.FactionArchitects,
		models.TerrainWasteland,
		fanFactionStartingResources(3, 15, 3, 9),
		CultPositions{Fire: 1, Air: 1},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 4},
		nil,
		nil, nil, nil, nil, nil,
	)
}

func NewArchivists() Faction {
	return newConfiguredFaction(
		models.FactionArchivists,
		models.TerrainDesert,
		standardFanFactionStartingResources(3, 15),
		CultPositions{},
		Income{Workers: 2},
		[]Income{{Workers: 1}, {Workers: 1}, {Workers: 1}, {}, {Workers: 1}, {Workers: 1}, {Workers: 1}, {}},
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 2},
		nil,
		nil, nil, nil, nil, nil,
	)
}

func NewAtlanteans() Faction {
	sanctuaryCost := Cost{Coins: 8, Workers: 4}
	return newConfiguredFaction(
		models.FactionAtlanteans,
		models.TerrainLake,
		fanFactionStartingResources(3, 15, 1, 11),
		CultPositions{Fire: 1, Water: 1},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 6},
		nil,
		nil, nil, nil, &sanctuaryCost, nil,
	)
}

func NewChashDallah() Faction {
	strongholdCost := Cost{Coins: 4, Workers: 4}
	return newConfiguredFaction(
		models.FactionChashDallah,
		models.TerrainForest,
		standardFanFactionStartingResources(3, 15),
		CultPositions{Earth: 1, Air: 1},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		[]Income{{Coins: 2, Power: 1}, {Coins: 2, Power: 1}, {Coins: 3, Power: 1}, {Coins: 4, Power: 1}},
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Coins: 4},
		fixedTerraformCost(3),
		nil, nil, nil, nil, &strongholdCost,
	)
}

func NewChildrenOfTheWyrm() Faction {
	sanctuaryCost := Cost{Coins: 5, Workers: 4}
	strongholdCost := Cost{Coins: 10, Workers: 4}
	return newConfiguredFaction(
		models.FactionChildrenOfTheWyrm,
		models.TerrainSwamp,
		standardFanFactionStartingResources(3, 12),
		CultPositions{Water: 1, Earth: 1},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 4},
		nil,
		nil, nil, nil, &sanctuaryCost, &strongholdCost,
	)
}

func NewConspirators() Faction {
	return newConfiguredFaction(
		models.FactionConspirators,
		models.TerrainMountain,
		standardFanFactionStartingResources(3, 15),
		CultPositions{},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		[]Income{{Coins: 3, Power: 1}, {Coins: 2, Power: 1}, {Coins: 2, Power: 2}, {Coins: 3, Power: 2}},
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{},
		nil,
		nil, nil, nil, nil, nil,
	)
}

func NewDjinni() Faction {
	strongholdCost := Cost{Coins: 6, Workers: 3}
	return newConfiguredFaction(
		models.FactionDjinni,
		models.TerrainDesert,
		standardFanFactionStartingResources(3, 15),
		CultPositions{},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 4},
		nil,
		nil, nil, nil, nil, &strongholdCost,
	)
}

func NewDynionGeifr() Faction {
	dwellingCost := Cost{Coins: 2, Workers: 2}
	return newConfiguredFaction(
		models.FactionDynionGeifr,
		models.TerrainMountain,
		standardFanFactionStartingResources(2, 15),
		CultPositions{Earth: 1, Air: 1},
		Income{Workers: 2},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		[]Income{{Priests: 1}, {Power: 5}, {Priests: 1}},
		Income{Priests: 1},
		Income{Power: 4},
		nil,
		&dwellingCost, nil, nil, nil, nil,
	)
}

func NewGoblins() Faction {
	sanctuaryCost := Cost{Coins: 6, Workers: 4}
	strongholdCost := Cost{Coins: 6, Workers: 4}
	templeCost := Cost{Coins: 6, Workers: 2}
	return newConfiguredFaction(
		models.FactionGoblins,
		models.TerrainSwamp,
		standardFanFactionStartingResources(3, 15),
		CultPositions{Earth: 1, Air: 1},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 4},
		nil,
		nil, nil, &templeCost, &sanctuaryCost, &strongholdCost,
	)
}

func NewProspectors() Faction {
	sanctuaryCost := Cost{Coins: 8, Workers: 4}
	strongholdCost := Cost{Coins: 11, Workers: 4}
	return newConfiguredFaction(
		models.FactionProspectors,
		models.TerrainPlains,
		fanFactionStartingResources(2, 15, 8, 4),
		CultPositions{Earth: 3},
		Income{Workers: 1},
		[]Income{{Workers: 1}, {Workers: 1}, {Workers: 1}, {}, {Workers: 1}, {}, {Workers: 1}, {}},
		[]Income{{Coins: 4, Power: 1}, {Coins: 3, Power: 1}, {Coins: 2, Power: 2}, {Coins: 1, Power: 2}},
		[]Income{{Coins: 3}, {Priests: 1}, {Coins: 4}},
		Income{Priests: 1},
		Income{Power: 3},
		nil,
		nil, nil, nil, &sanctuaryCost, &strongholdCost,
	)
}

func NewTheEnlightened() Faction {
	base := standardFanFactionStartingResources(3, 15)
	templeCost := Cost{Coins: 5, Priests: 1}
	sanctuaryCost := Cost{Coins: 6, Priests: 2}
	strongholdCost := Cost{Coins: 4, Priests: 1}
	return newConfiguredFaction(
		models.FactionTheEnlightened,
		models.TerrainForest,
		base,
		CultPositions{Air: 2},
		Income{Power: 3},
		[]Income{{Power: 2}, {Power: 3}, {Power: 2}, {Power: 3}, {Power: 2}, {Power: 3}, {Power: 2}, {Power: 3}},
		[]Income{{Coins: 2, Power: 1}, {Coins: 2, Power: 1}, {Coins: 3, Power: 1}, {Coins: 4, Power: 1}},
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{},
		nil,
		nil, nil, &templeCost, &sanctuaryCost, &strongholdCost,
	)
}

func NewTimeTravelers() Faction {
	sanctuaryCost := Cost{Coins: 8, Workers: 4}
	strongholdCost := Cost{Coins: 8, Workers: 4}
	return newConfiguredFaction(
		models.FactionTimeTravelers,
		models.TerrainPlains,
		standardFanFactionStartingResources(3, 15),
		CultPositions{Fire: 2},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 2},
		nil,
		nil, nil, nil, &sanctuaryCost, &strongholdCost,
	)
}

func NewTreasurers() Faction {
	strongholdIncome := Income{Power: 4}
	return newConfiguredFaction(
		models.FactionTreasurers,
		models.TerrainWasteland,
		fanFactionStartingResources(4, 15, 4, 8),
		CultPositions{Fire: 2},
		Income{},
		[]Income{{Workers: 1}, {Workers: 1}, {}, {Workers: 1}, {}, {Workers: 1}, {Workers: 1}, {}},
		[]Income{{Coins: 2, Power: 1}, {}, {Coins: 2, Power: 2}, {Coins: 2, Power: 2}},
		[]Income{{Priests: 1}, {Power: 4}, {Priests: 1}},
		Income{Priests: 1},
		strongholdIncome,
		nil,
		nil, nil, nil, nil, nil,
	)
}

func NewWisps() Faction {
	strongholdCost := Cost{Coins: 4, Workers: 4}
	return newConfiguredFaction(
		models.FactionWisps,
		models.TerrainLake,
		fanFactionStartingResources(3, 15, 7, 5),
		CultPositions{Water: 1, Air: 1},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 3},
		nil,
		nil, nil, nil, nil, &strongholdCost,
	)
}
