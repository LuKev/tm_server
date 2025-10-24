package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestStrongholdIncome_Auren(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Place a stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.GetHex(strongholdHex).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Calculate income
	income := calculateBuildingIncome(gs, player)

	// Auren stronghold: 2 power + 1 priest
	if income.Power != 2 {
		t.Errorf("expected 2 power from Auren stronghold, got %d", income.Power)
	}
	if income.Priests != 1 {
		t.Errorf("expected 1 priest from Auren stronghold, got %d", income.Priests)
	}
}

func TestStrongholdIncome_Swarmlings(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Place a stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.GetHex(strongholdHex).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Calculate income
	income := calculateBuildingIncome(gs, player)

	// Swarmlings stronghold: 4 power + 2 priests
	if income.Power != 4 {
		t.Errorf("expected 4 power from Swarmlings stronghold, got %d", income.Power)
	}
	if income.Priests != 2 {
		t.Errorf("expected 2 priests from Swarmlings stronghold, got %d", income.Priests)
	}
}

func TestStrongholdIncome_Alchemists(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Place a stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.GetHex(strongholdHex).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Calculate income
	income := calculateBuildingIncome(gs, player)

	// Alchemists stronghold: 6 coins + 1 priest
	if income.Coins != 6 {
		t.Errorf("expected 6 coins from Alchemists stronghold, got %d", income.Coins)
	}
	if income.Priests != 1 {
		t.Errorf("expected 1 priest from Alchemists stronghold, got %d", income.Priests)
	}
}

func TestStrongholdIncome_ChaosMagicians(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewChaosMagicians()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Place a stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.GetHex(strongholdHex).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Calculate income
	income := calculateBuildingIncome(gs, player)

	// Chaos Magicians stronghold: 2 workers + 1 priest
	if income.Workers != 2 {
		t.Errorf("expected 2 workers from Chaos Magicians stronghold, got %d", income.Workers)
	}
	if income.Priests != 1 {
		t.Errorf("expected 1 priest from Chaos Magicians stronghold, got %d", income.Priests)
	}
}

func TestStrongholdIncome_Fakirs(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewFakirs()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Place a stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.GetHex(strongholdHex).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Calculate income
	income := calculateBuildingIncome(gs, player)

	// Fakirs stronghold: 1 priest only
	if income.Priests != 1 {
		t.Errorf("expected 1 priest from Fakirs stronghold, got %d", income.Priests)
	}
	if income.Power != 0 {
		t.Errorf("expected 0 power from Fakirs stronghold, got %d", income.Power)
	}
	if income.Workers != 0 {
		t.Errorf("expected 0 workers from Fakirs stronghold, got %d", income.Workers)
	}
	if income.Coins != 0 {
		t.Errorf("expected 0 coins from Fakirs stronghold, got %d", income.Coins)
	}
}

func TestBuildingIncome_Mixed(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Place various buildings
	// 2 dwellings
	gs.Map.GetHex(NewHex(0, 0)).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	gs.Map.GetHex(NewHex(0, 1)).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// 1 trading house
	gs.Map.GetHex(NewHex(1, 0)).Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}

	// 1 temple
	gs.Map.GetHex(NewHex(1, 1)).Building = &models.Building{
		Type:       models.BuildingTemple,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}

	// 1 stronghold (Halflings: 2 power + 1 priest)
	gs.Map.GetHex(NewHex(2, 0)).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Calculate income
	income := calculateBuildingIncome(gs, player)

	// Expected:
	// 2 dwellings: 2 workers
	// 1 trading house: 2 coins + 1 power
	// 1 temple: 1 priest + 1 power
	// 1 stronghold: 2 power + 1 priest
	// Total: 2 workers, 2 coins, 2 priests, 4 power

	if income.Workers != 2 {
		t.Errorf("expected 2 workers, got %d", income.Workers)
	}
	if income.Coins != 2 {
		t.Errorf("expected 2 coins, got %d", income.Coins)
	}
	if income.Priests != 2 {
		t.Errorf("expected 2 priests, got %d", income.Priests)
	}
	if income.Power != 4 {
		t.Errorf("expected 4 power, got %d", income.Power)
	}
}

func TestBaseFactionIncome_Standard(t *testing.T) {
	// Most factions get 1 worker base income
	income := getBaseFactionIncome(models.FactionHalflings)
	if income.Workers != 1 {
		t.Errorf("expected 1 worker base income for Halflings, got %d", income.Workers)
	}
}

func TestBaseFactionIncome_Engineers(t *testing.T) {
	// Engineers get 0 base income
	income := getBaseFactionIncome(models.FactionEngineers)
	if income.Workers != 0 {
		t.Errorf("expected 0 worker base income for Engineers, got %d", income.Workers)
	}
}

func TestBaseFactionIncome_Swarmlings(t *testing.T) {
	// Swarmlings get 2 workers base income
	income := getBaseFactionIncome(models.FactionSwarmlings)
	if income.Workers != 2 {
		t.Errorf("expected 2 workers base income for Swarmlings, got %d", income.Workers)
	}
}

func TestDwellingIncome_Standard(t *testing.T) {
	// Standard factions: 1 worker per dwelling, except 8th gives no income
	tests := []struct {
		dwellings int
		expected  int
	}{
		{0, 0},
		{1, 1},
		{5, 5},
		{7, 7},
		{8, 7}, // 8th dwelling gives no income
	}

	for _, tt := range tests {
		income := calculateDwellingIncome(tt.dwellings, models.FactionHalflings)
		if income != tt.expected {
			t.Errorf("expected %d workers from %d dwellings, got %d", tt.expected, tt.dwellings, income)
		}
	}
}

func TestDwellingIncome_Engineers(t *testing.T) {
	// Engineers: dwellings 1, 2, 4, 5, 7, 8 give income (not 3rd or 6th)
	tests := []struct {
		dwellings int
		expected  int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 2}, // 3rd dwelling gives no income
		{4, 3},
		{5, 4},
		{6, 4}, // 6th dwelling gives no income
		{7, 5},
		{8, 6},
	}

	for _, tt := range tests {
		income := calculateDwellingIncome(tt.dwellings, models.FactionEngineers)
		if income != tt.expected {
			t.Errorf("expected %d workers from %d Engineers dwellings, got %d", tt.expected, tt.dwellings, income)
		}
	}
}

func TestTempleIncome_Standard(t *testing.T) {
	// Standard: 1 priest + 1 power per temple
	income := calculateTempleIncome(2, models.FactionHalflings)
	if income.Priests != 2 {
		t.Errorf("expected 2 priests from 2 temples, got %d", income.Priests)
	}
	if income.Power != 2 {
		t.Errorf("expected 2 power from 2 temples, got %d", income.Power)
	}
}

func TestTempleIncome_Engineers(t *testing.T) {
	// Engineers: 1st temple = 1 priest + 1 power, 2nd temple = 5 power, 3rd temple = 1 priest + 1 power
	tests := []struct {
		temples        int
		expectedPriest int
		expectedPower  int
	}{
		{1, 1, 1}, // 1st temple: 1 priest + 1 power
		{2, 1, 6}, // 1st temple: 1 priest + 1 power, 2nd temple: 5 power
		{3, 2, 7}, // 1st: 1p+1pw, 2nd: 5pw, 3rd: 1p+1pw
	}

	for _, tt := range tests {
		income := calculateTempleIncome(tt.temples, models.FactionEngineers)
		if income.Priests != tt.expectedPriest {
			t.Errorf("expected %d priests from %d Engineers temples, got %d", tt.expectedPriest, tt.temples, income.Priests)
		}
		if income.Power != tt.expectedPower {
			t.Errorf("expected %d power from %d Engineers temples, got %d", tt.expectedPower, tt.temples, income.Power)
		}
	}
}

func TestTradingHouseIncome_Standard(t *testing.T) {
	// Standard: 1st-2nd: 2c+1pw, 3rd-4th: 2c+2pw
	tests := []struct {
		count         int
		expectedCoins int
		expectedPower int
	}{
		{1, 2, 1},
		{2, 4, 2},
		{3, 6, 4}, // 2+2+2 coins, 1+1+2 power
		{4, 8, 6}, // 2+2+2+2 coins, 1+1+2+2 power
	}

	for _, tt := range tests {
		income := calculateTradingHouseIncome(tt.count, models.FactionHalflings)
		if income.Coins != tt.expectedCoins {
			t.Errorf("expected %d coins from %d TH, got %d", tt.expectedCoins, tt.count, income.Coins)
		}
		if income.Power != tt.expectedPower {
			t.Errorf("expected %d power from %d TH, got %d", tt.expectedPower, tt.count, income.Power)
		}
	}
}

func TestTradingHouseIncome_Nomads(t *testing.T) {
	// Nomads: 1st-2nd: 2c+1pw, 3rd: 3c+1pw, 4th: 4c+1pw
	tests := []struct {
		count         int
		expectedCoins int
		expectedPower int
	}{
		{1, 2, 1},
		{2, 4, 2},
		{3, 7, 3}, // 2+2+3 coins, 1+1+1 power
		{4, 11, 4}, // 2+2+3+4 coins, 1+1+1+1 power
	}

	for _, tt := range tests {
		income := calculateTradingHouseIncome(tt.count, models.FactionNomads)
		if income.Coins != tt.expectedCoins {
			t.Errorf("expected %d coins from %d Nomads TH, got %d", tt.expectedCoins, tt.count, income.Coins)
		}
		if income.Power != tt.expectedPower {
			t.Errorf("expected %d power from %d Nomads TH, got %d", tt.expectedPower, tt.count, income.Power)
		}
	}
}

func TestTradingHouseIncome_Dwarves(t *testing.T) {
	// Dwarves: 1st: 3c+1pw, 2nd: 2c+1pw, 3rd: 2c+2pw, 4th: 3c+2pw
	tests := []struct {
		count         int
		expectedCoins int
		expectedPower int
	}{
		{1, 3, 1},
		{2, 5, 2},
		{3, 7, 4}, // 3+2+2 coins, 1+1+2 power
		{4, 10, 6}, // 3+2+2+3 coins, 1+1+2+2 power
	}

	for _, tt := range tests {
		income := calculateTradingHouseIncome(tt.count, models.FactionDwarves)
		if income.Coins != tt.expectedCoins {
			t.Errorf("expected %d coins from %d Dwarves TH, got %d", tt.expectedCoins, tt.count, income.Coins)
		}
		if income.Power != tt.expectedPower {
			t.Errorf("expected %d power from %d Dwarves TH, got %d", tt.expectedPower, tt.count, income.Power)
		}
	}
}

func TestTradingHouseIncome_Swarmlings(t *testing.T) {
	// Swarmlings: 1st-3rd: 2c+2pw, 4th: 3c+2pw
	tests := []struct {
		count         int
		expectedCoins int
		expectedPower int
	}{
		{1, 2, 2},
		{2, 4, 4},
		{3, 6, 6}, // 2+2+2 coins, 2+2+2 power
		{4, 9, 8}, // 2+2+2+3 coins, 2+2+2+2 power
	}

	for _, tt := range tests {
		income := calculateTradingHouseIncome(tt.count, models.FactionSwarmlings)
		if income.Coins != tt.expectedCoins {
			t.Errorf("expected %d coins from %d Swarmlings TH, got %d", tt.expectedCoins, tt.count, income.Coins)
		}
		if income.Power != tt.expectedPower {
			t.Errorf("expected %d power from %d Swarmlings TH, got %d", tt.expectedPower, tt.count, income.Power)
		}
	}
}

func TestGrantIncome_AppliesCorrectly(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Set initial resources - power in Bowl1 to test GainPower properly
	player.Resources.Coins = 5
	player.Resources.Workers = 3
	player.Resources.Priests = 1
	player.Resources.Power.Bowl1 = 10
	player.Resources.Power.Bowl2 = 2
	player.Resources.Power.Bowl3 = 3

	// Place a dwelling and stronghold
	gs.Map.GetHex(NewHex(0, 0)).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	gs.Map.GetHex(NewHex(0, 1)).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}

	// Grant income
	gs.GrantIncome()

	// Expected income:
	// Base income (Giants): 1 worker
	// 1 dwelling: 1 worker
	// 1 stronghold (Giants): 4 power + 1 priest
	// Total: 2 workers, 1 priest, 4 power
	// Power cycles: 4 power from Bowl1 -> Bowl2

	expectedCoins := 5 // No change
	expectedWorkers := 3 + 2 // Base 1 + dwelling 1
	expectedPriests := 1 + 1
	expectedBowl1 := 10 - 4 // 4 power moved from Bowl1
	expectedBowl2 := 2 + 4 // 4 power moved to Bowl2

	if player.Resources.Coins != expectedCoins {
		t.Errorf("expected %d coins, got %d", expectedCoins, player.Resources.Coins)
	}
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
	if player.Resources.Priests != expectedPriests {
		t.Errorf("expected %d priests, got %d", expectedPriests, player.Resources.Priests)
	}
	if player.Resources.Power.Bowl1 != expectedBowl1 {
		t.Errorf("expected %d power in bowl 1, got %d", expectedBowl1, player.Resources.Power.Bowl1)
	}
	if player.Resources.Power.Bowl2 != expectedBowl2 {
		t.Errorf("expected %d power in bowl 2, got %d", expectedBowl2, player.Resources.Power.Bowl2)
	}
}

func TestIncome_WithFavorTiles(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up initial resources
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	player.Resources.Priests = 2
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player favor tiles: Fire+1 (3 coins), Earth+2 (1 worker, 1 power), Air+2 (4 power)
	gs.FavorTiles.TakeFavorTile("player1", FavorFire1)
	gs.FavorTiles.TakeFavorTile("player1", FavorEarth2)
	gs.FavorTiles.TakeFavorTile("player1", FavorAir2)

	// Grant income
	gs.GrantIncome()

	// Expected income:
	// Base income (Auren): 1 worker
	// Favor tiles: +3 coins (Fire+1), +1 worker (Earth+2), +5 power (Earth+2 + Air+2)
	// Total: +3 coins, +2 workers, +5 power

	expectedCoins := 10 + 3
	expectedWorkers := 5 + 1 + 1 // Base 1 + Earth+2 1
	expectedPriests := 2
	expectedBowl1 := 20 - 5
	expectedBowl2 := 0 + 5

	if player.Resources.Coins != expectedCoins {
		t.Errorf("expected %d coins, got %d", expectedCoins, player.Resources.Coins)
	}
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers, got %d", expectedWorkers, player.Resources.Workers)
	}
	if player.Resources.Priests != expectedPriests {
		t.Errorf("expected %d priests, got %d", expectedPriests, player.Resources.Priests)
	}
	if player.Resources.Power.Bowl1 != expectedBowl1 {
		t.Errorf("expected %d power in bowl 1, got %d", expectedBowl1, player.Resources.Power.Bowl1)
	}
	if player.Resources.Power.Bowl2 != expectedBowl2 {
		t.Errorf("expected %d power in bowl 2, got %d", expectedBowl2, player.Resources.Power.Bowl2)
	}
}
