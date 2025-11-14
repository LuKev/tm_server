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
	// Note: We do NOT clear PendingSpades here
	// Cult reward spades from the previous round's cleanup phase persist into the new round
	// and can be used during the action phase
	// Example: A player reaches position 7 on a cult track at end of round 5,
	// gets 1 spade reward, and can use it in round 6

	for _, player := range gs.Players {
		income := calculatePlayerIncome(gs, player)
		applyIncome(gs, player, income)
	}
}

// calculatePlayerIncome calculates the total income for a player
func calculatePlayerIncome(gs *GameState, player *Player) BaseIncome {
	income := BaseIncome{}
	faction := player.Faction

	// 1. Base faction income (uses faction method)
	baseIncome := faction.GetBaseFactionIncome()
	income.Coins += baseIncome.Coins
	income.Workers += baseIncome.Workers
	income.Priests += baseIncome.Priests
	income.Power += baseIncome.Power

	// 2. Income from buildings on the map (uses faction methods)
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

// calculateBuildingIncome calculates income from buildings on the map
// Uses faction methods for income calculations
func calculateBuildingIncome(gs *GameState, player *Player) BaseIncome {
	income := BaseIncome{}
	faction := player.Faction

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

	// Dwelling income (uses faction method)
	dwellingIncome := faction.GetDwellingIncome(dwellings)
	income.Workers += dwellingIncome.Workers

	// Trading house income (uses faction method)
	thIncome := faction.GetTradingHouseIncome(tradingHouses)
	income.Coins += thIncome.Coins
	income.Power += thIncome.Power

	// Temple income (uses faction method)
	templeIncome := faction.GetTempleIncome(temples)
	income.Priests += templeIncome.Priests
	income.Power += templeIncome.Power

	// Sanctuary income (uses faction method, only 1 per faction)
	if sanctuaries > 0 {
		sanctuaryIncome := faction.GetSanctuaryIncome()
		income.Priests += sanctuaryIncome.Priests
	}

	// Stronghold income (uses faction method)
	if strongholds > 0 {
		strongholdIncome := faction.GetStrongholdIncome()
		income.Coins += strongholdIncome.Coins
		income.Workers += strongholdIncome.Workers
		income.Priests += strongholdIncome.Priests
		income.Power += strongholdIncome.Power
	}

	return income
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
