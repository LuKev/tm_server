package factions

import "github.com/lukev/tm_server/internal/models"

type configuredDiggingFaction struct {
	*configuredFaction
	diggingCost Cost
}

func (f *configuredDiggingFaction) GetDiggingCost(currentLevel int) Cost {
	return f.diggingCost
}

type configuredShippingFaction struct {
	*configuredFaction
	shippingLevel int
}

func (f *configuredShippingFaction) GetShippingLevel() int {
	return f.shippingLevel
}

func iceStartingResources(power1, power2 int) Resources {
	return Resources{Coins: 15, Workers: 3, Power1: power1, Power2: power2}
}

func volcanoStartingResources(power1, power2 int) Resources {
	return Resources{Coins: 15, Workers: 3, Power1: power1, Power2: power2}
}

var (
	iceDiggingCost       = Cost{Coins: 5, Workers: 1, Priests: 1}
	iceStrongholdCost    = Cost{Coins: 6, Workers: 4}
	iceSanctuaryCost     = Cost{Coins: 6, Workers: 4}
	volcanoStronghold8C  = Cost{Coins: 8, Workers: 4}
	volcanoSanctuary8C   = Cost{Coins: 8, Workers: 4}
	colorlessStronghold6 = Cost{Coins: 6, Workers: 4}
	shapeshifterSHCost   = Cost{Coins: 6, Workers: 3}
	firewalkersSHCost    = Cost{Coins: 8, Workers: 4}
)

var (
	allDwellingIncomeSeq = []Income{
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
	}
	noBaseFourthDwellingIncomeSeq = []Income{
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
	}
	firewalkersDwellingIncomeSeq = []Income{
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{},
		{Workers: 1},
		{Workers: 1},
		{Workers: 1},
		{},
	}
	firewalkersTempleIncomeSeq = []Income{
		{Priests: 1},
		{VictoryPoints: 2},
		{Priests: 1},
	}
	yetisTradingIncomeSeq = []Income{
		{Coins: 2, Power: 2},
		{Coins: 2, Power: 2},
		{Coins: 2, Power: 2},
		{Coins: 2, Power: 2},
	}
	yetisTempleIncomeSeq = []Income{
		{Priests: 1},
		{Power: 5},
		{Priests: 1},
	}
	riverwalkersDwellingIncomeSeq = []Income{
		{Workers: 1},
		{Workers: 1},
		{},
		{Workers: 1},
		{Workers: 1},
		{},
		{Workers: 1},
		{Workers: 1},
	}
	riverwalkersTempleIncomeSeq = []Income{
		{Priests: 1},
		{Power: 5},
		{Priests: 1},
	}
	snowShamansTempleIncomeSeq = []Income{
		{Power: 4},
		{Priests: 1},
		{Power: 4},
	}
)

func NewIceMaidens() Faction {
	return &configuredDiggingFaction{
		configuredFaction: newConfiguredFaction(
			models.FactionIceMaidens,
			models.TerrainIce,
			iceStartingResources(6, 6),
			CultPositions{Water: 1, Air: 1},
			Income{Workers: 1},
			allDwellingIncomeSeq,
			standardTradingIncomeSeq,
			standardTempleIncomeSeq,
			Income{Priests: 1},
			Income{Power: 4},
			nil,
			nil, nil, nil, &iceSanctuaryCost, &iceStrongholdCost,
		),
		diggingCost: iceDiggingCost,
	}
}

func NewYetis() Faction {
	return &configuredDiggingFaction{
		configuredFaction: newConfiguredFaction(
			models.FactionYetis,
			models.TerrainIce,
			iceStartingResources(0, 12),
			CultPositions{Earth: 1, Air: 1},
			Income{Workers: 1},
			allDwellingIncomeSeq,
			yetisTradingIncomeSeq,
			yetisTempleIncomeSeq,
			Income{Priests: 1},
			Income{Power: 4},
			nil,
			nil, nil, nil, &iceSanctuaryCost, &iceStrongholdCost,
		),
		diggingCost: iceDiggingCost,
	}
}

func NewAcolytes() Faction {
	return newConfiguredFaction(
		models.FactionAcolytes,
		models.TerrainVolcano,
		volcanoStartingResources(6, 6),
		CultPositions{Fire: 3, Water: 3, Earth: 3, Air: 3},
		Income{},
		noBaseFourthDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 2},
		nil,
		nil, nil, nil, &volcanoSanctuary8C, &volcanoStronghold8C,
	)
}

func NewDragonlords() Faction {
	return newConfiguredFaction(
		models.FactionDragonlords,
		models.TerrainVolcano,
		volcanoStartingResources(4, 4),
		CultPositions{Fire: 2},
		Income{},
		noBaseFourthDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 2},
		nil,
		nil, nil, nil, &volcanoSanctuary8C, &volcanoStronghold8C,
	)
}

func NewShapeshifters() Faction {
	return newConfiguredFaction(
		models.FactionShapeshifters,
		models.TerrainPlains,
		iceStartingResources(4, 4),
		CultPositions{Fire: 1, Water: 1},
		Income{Workers: 1},
		standardDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 4},
		nil,
		nil, nil, nil, &iceSanctuaryCost, &shapeshifterSHCost,
	)
}

func NewRiverwalkers() Faction {
	return &configuredShippingFaction{
		configuredFaction: newConfiguredFaction(
			models.FactionRiverwalkers,
			models.TerrainTypeUnknown,
			Resources{Coins: 15, Workers: 3, Power1: 10, Power2: 2},
			CultPositions{Fire: 1, Air: 1},
			Income{Workers: 1},
			riverwalkersDwellingIncomeSeq,
			standardTradingIncomeSeq,
			riverwalkersTempleIncomeSeq,
			Income{Priests: 1},
			Income{Power: 2},
			nil,
			nil, nil, nil, &iceSanctuaryCost, &colorlessStronghold6,
		),
		shippingLevel: 1,
	}
}

func NewFirewalkers() Faction {
	return newConfiguredFaction(
		models.FactionFirewalkers,
		models.TerrainVolcano,
		standardFanFactionStartingResources(3, 15),
		CultPositions{Fire: 1, Air: 1},
		Income{},
		firewalkersDwellingIncomeSeq,
		standardTradingIncomeSeq,
		firewalkersTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 2},
		nil,
		nil, nil, nil, nil, &firewalkersSHCost,
	)
}

func NewSelkies() Faction {
	return newConfiguredFaction(
		models.FactionSelkies,
		models.TerrainIce,
		standardFanFactionStartingResources(3, 15),
		CultPositions{Water: 2},
		Income{Workers: 1},
		allDwellingIncomeSeq,
		standardTradingIncomeSeq,
		standardTempleIncomeSeq,
		Income{Priests: 1},
		Income{Power: 4},
		nil,
		nil, nil, nil, &iceSanctuaryCost, &iceStrongholdCost,
	)
}

func NewSnowShamans() Faction {
	return newConfiguredFaction(
		models.FactionSnowShamans,
		models.TerrainIce,
		standardFanFactionStartingResources(3, 15),
		CultPositions{Water: 1, Earth: 1},
		Income{Workers: 1},
		allDwellingIncomeSeq,
		standardTradingIncomeSeq,
		snowShamansTempleIncomeSeq,
		Income{Priests: 1},
		Income{},
		nil,
		nil, nil, nil, &iceSanctuaryCost, &iceStrongholdCost,
	)
}
