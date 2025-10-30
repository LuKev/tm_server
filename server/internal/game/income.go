package game

import (
	"github.com/lukev/tm_server/internal/models"
)

// Income Phase Implementation
//
// Terra Mystica Income Phase occurs at the start of each round (rounds 2-6).
// Round 1 has no income phase (players start with initial resources).
//
// Income Sources:
// 1. Base faction income
// 2. Buildings on the map (coins, workers, priests, power)
// 3. Bonus tiles (implemented)
// 4. Favor tiles (implemented)
//
// Income is granted simultaneously to all players at the start of the round.

// BaseIncome represents the standard income for each faction
type BaseIncome struct {
	Coins   int
	Workers int
	Priests int
	Power   int // Power cycles through bowls using GainPower()
}

// GrantIncome grants income to all players at the start of a round
func (gs *GameState) GrantIncome() {
	for _, player := range gs.Players {
		income := calculatePlayerIncome(gs, player)
		applyIncome(gs, player, income)
	}
}

// calculatePlayerIncome calculates the total income for a player
func calculatePlayerIncome(gs *GameState, player *Player) BaseIncome {
	income := BaseIncome{}

	// 1. Base faction income
	baseIncome := getBaseFactionIncome(player.Faction.GetType())
	income.Coins += baseIncome.Coins
	income.Workers += baseIncome.Workers
	income.Priests += baseIncome.Priests
	income.Power += baseIncome.Power
	// 2. Income from buildings on the map
	buildingIncome := calculateBuildingIncome(gs, player)
	income.Coins += buildingIncome.Coins
	income.Workers += buildingIncome.Workers
	income.Priests += buildingIncome.Priests
	income.Power += buildingIncome.Power

	// 3. Income from favor tiles
	playerTiles := gs.FavorTiles.GetPlayerTiles(player.ID)
	favorCoins, favorWorkers, favorPower := GetFavorTileIncomeBonus(playerTiles)
	income.Coins += favorCoins
	income.Workers += favorWorkers
	income.Power += favorPower

	// 4. Income from bonus cards
	if bonusCard, ok := gs.BonusCards.GetPlayerCard(player.ID); ok {
		bonusCoins, bonusWorkers, bonusPriests, bonusPower := GetBonusCardIncomeBonus(bonusCard)
		income.Coins += bonusCoins
		income.Workers += bonusWorkers
		income.Priests += bonusPriests
		income.Power += bonusPower
	}

	return income
}

// getBaseFactionIncome returns the base income for each faction (before buildings)
func getBaseFactionIncome(factionType models.FactionType) BaseIncome {
	switch factionType {
	// 0 base income
	case models.FactionEngineers:
		return BaseIncome{}

	// 2 workers base income
	case models.FactionSwarmlings:
		return BaseIncome{Workers: 2}

	// All other factions: 1 worker base income
	default:
		return BaseIncome{Workers: 1}
	}
}

// calculateBuildingIncome calculates income from buildings on the map
func calculateBuildingIncome(gs *GameState, player *Player) BaseIncome {
	income := BaseIncome{}

	// Count buildings of each type
	dwellings := 0
	tradingHouses := 0
	temples := 0
	sanctuaries := 0
	strongholds := 0

	for _, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == player.ID {
			switch mapHex.Building.Type {
			case models.BuildingDwelling:
				dwellings++
			case models.BuildingTradingHouse:
				tradingHouses++
			case models.BuildingTemple:
				temples++
			case models.BuildingSanctuary:
				sanctuaries++
			case models.BuildingStronghold:
				strongholds++
			}
		}
	}

	factionType := player.Faction.GetType()

	// Dwelling income: 1 worker per dwelling (with exceptions)
	income.Workers += calculateDwellingIncome(dwellings, factionType)

	// Trading house income: varies by faction
	tradingHouseIncome := calculateTradingHouseIncome(tradingHouses, factionType)
	income.Coins += tradingHouseIncome.Coins
	income.Power += tradingHouseIncome.Power

	// Temple income: 1 priest per temple (standard), Engineers 2nd temple gives 5 power instead
	// Standard temples provide cult advancement abilities, not power income
	// Engineers exception: 2nd temple gives 5 power instead of priest
	templeIncome := calculateTempleIncome(temples, factionType)
	income.Priests += templeIncome.Priests
	income.Power += templeIncome.Power // Engineers 2nd temple gives 5 power

	// Sanctuary income: 1 priest (standard), 2 priests for Darklings/Swarmlings
	sanctuaryIncome := calculateSanctuaryIncome(sanctuaries, factionType)
	income.Priests += sanctuaryIncome

	// Stronghold income is faction-specific
	if strongholds > 0 {
		strongholdIncome := getStrongholdIncome(player.Faction.GetType())
		income.Coins += strongholdIncome.Coins
		income.Workers += strongholdIncome.Workers
		income.Priests += strongholdIncome.Priests
		income.Power += strongholdIncome.Power
	}

	return income
}

// calculateDwellingIncome calculates worker income from dwellings
// Standard: 1 worker per dwelling, except 8th dwelling gives no income
// Engineers: Workers from dwellings 1, 2, 4, 5, 7, 8 (not 3rd or 6th)
func calculateDwellingIncome(dwellingCount int, factionType models.FactionType) int {
	if dwellingCount == 0 {
		return 0
	}

	if factionType == models.FactionEngineers {
		// Engineers: dwellings 1, 2, 4, 5, 7, 8 give income
		// Pattern: skip 3rd and 6th
		income := 0
		for i := 1; i <= dwellingCount && i <= 8; i++ {
			if i != 3 && i != 6 {
				income++
			}
		}
		return income
	}

	// Standard factions: 1 worker per dwelling, except 8th gives no income
	if dwellingCount >= 8 {
		return 7 // Max 7 workers from dwellings
	}
	return dwellingCount
}

// calculateTradingHouseIncome calculates income from trading houses
// Standard: 1st-2nd: 2c+1pw, 3rd-4th: 2c+2pw
// Nomads/Alchemists: 1st-2nd: 2c+1pw, 3rd: 3c+1pw, 4th: 4c+1pw
// Dwarves: 1st: 3c+1pw, 2nd: 2c+1pw, 3rd: 2c+2pw, 4th: 3c+2pw
// Swarmlings: 1st-3rd: 2c+2pw, 4th: 3c+2pw
func calculateTradingHouseIncome(tradingHouseCount int, factionType models.FactionType) BaseIncome {
	income := BaseIncome{}

	if tradingHouseCount == 0 {
		return income
	}

	// Nomads and Alchemists have special coin progression
	if factionType == models.FactionNomads || factionType == models.FactionAlchemists {
		for i := 1; i <= tradingHouseCount && i <= 4; i++ {
			switch i {
			case 1, 2:
				income.Coins += 2
				income.Power += 1
			case 3:
				income.Coins += 3
				income.Power += 1
			case 4:
				income.Coins += 4
				income.Power += 1
			}
		}
		return income
	}

	// Dwarves have special pattern
	if factionType == models.FactionDwarves {
		for i := 1; i <= tradingHouseCount && i <= 4; i++ {
			switch i {
			case 1:
				income.Coins += 3
				income.Power += 1
			case 2:
				income.Coins += 2
				income.Power += 1
			case 3:
				income.Coins += 2
				income.Power += 2
			case 4:
				income.Coins += 3
				income.Power += 2
			}
		}
		return income
	}

	// Swarmlings have special pattern
	if factionType == models.FactionSwarmlings {
		for i := 1; i <= tradingHouseCount && i <= 4; i++ {
			switch i {
			case 1, 2, 3:
				income.Coins += 2
				income.Power += 2
			case 4:
				income.Coins += 3
				income.Power += 2
			}
		}
		return income
	}

	// Standard factions: 1st-2nd: 2c+1pw, 3rd-4th: 2c+2pw
	for i := 1; i <= tradingHouseCount && i <= 4; i++ {
		if i <= 2 {
			income.Coins += 2
			income.Power += 1
		} else {
			income.Coins += 2
			income.Power += 2
		}
	}
	return income
}

// calculateTempleIncome calculates income from temples
// Standard: 1 priest + 1 power per temple
// Engineers: 2nd temple gives 5 power instead of 1 priest + 1 power
func calculateTempleIncome(templeCount int, factionType models.FactionType) BaseIncome {
	income := BaseIncome{}

	if factionType == models.FactionEngineers {
		// Engineers: 1st and 3rd temples give 1 priest (NO power), 2nd temple gives 5 power (NO priest)
		for i := 1; i <= templeCount; i++ {
			if i == 2 {
				income.Power += 5 // 2nd temple: 5 power, no priest
			} else {
				income.Priests++ // 1st and 3rd temples: 1 priest, no power
			}
		}
		return income
	}

	// Standard: 1 priest per temple (NO power for standard temples)
	// Temples provide cult advancement abilities, not power income
	income.Priests = templeCount
	// No power income for standard temples
	return income
}

// calculateSanctuaryIncome calculates priest income from sanctuaries
// Standard: 1 priest per sanctuary
// Darklings/Swarmlings: 2 priests per sanctuary
func calculateSanctuaryIncome(sanctuaryCount int, factionType models.FactionType) int {
	if sanctuaryCount == 0 {
		return 0
	}

	// Darklings and Swarmlings get 2 priests per sanctuary
	if factionType == models.FactionDarklings || factionType == models.FactionSwarmlings {
		return sanctuaryCount * 2
	}

	// Standard: 1 priest per sanctuary
	return sanctuaryCount
}

// getStrongholdIncome returns the income for a faction's stronghold
// Most strongholds give their special resource (power/coins/workers) + 1 priest
// Swarmlings: special resource + 2 priests
// Fakirs: 1 priest only (no power)
func getStrongholdIncome(factionType models.FactionType) BaseIncome {
	switch factionType {
	// 2 power + 1 priest
	case models.FactionAuren, models.FactionWitches, models.FactionNomads,
		models.FactionDarklings, models.FactionCultists, models.FactionHalflings,
		models.FactionEngineers, models.FactionDwarves:
		return BaseIncome{Power: 2, Priests: 1}

	// 4 power + 1 priest
	case models.FactionMermaids, models.FactionGiants:
		return BaseIncome{Power: 4, Priests: 1}

	// 4 power + 2 priests (Swarmlings)
	case models.FactionSwarmlings:
		return BaseIncome{Power: 4, Priests: 2}

	// 1 priest only (Fakirs)
	case models.FactionFakirs:
		return BaseIncome{Priests: 1}

	// 2 workers + 1 priest (Chaos Magicians)
	case models.FactionChaosMagicians:
		return BaseIncome{Workers: 2, Priests: 1}

	// 6 coins + 1 priest (Alchemists)
	case models.FactionAlchemists:
		return BaseIncome{Coins: 6, Priests: 1}

	default:
		// Default: 0 income (should not happen if all factions are covered)
		return BaseIncome{}
	}
}

// applyIncome applies the calculated income to a player's resources
func applyIncome(gs *GameState, player *Player, income BaseIncome) {
	player.Resources.Coins += income.Coins
	player.Resources.Workers += income.Workers

	// Apply priest income with 7-priest limit enforcement
	if income.Priests > 0 {
		gs.GainPriests(player.ID, income.Priests)
	}

	// Use GainPower to properly cycle power through bowls
	if income.Power > 0 {
		player.Resources.Power.GainPower(income.Power)
	}
}
