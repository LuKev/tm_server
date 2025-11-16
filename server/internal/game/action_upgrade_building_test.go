package game

import (
	"testing"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestUpgradeBuilding_DwellingToTradingHouse(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Place a dwelling at (0, 1)
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Give player enough resources for upgrade
	tradingHouseCost := player.Faction.GetTradingHouseCost()
	player.Resources.Coins = tradingHouseCost.Coins + 10
	player.Resources.Workers = tradingHouseCost.Workers + 10
	
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify building was upgraded
	mapHex := gs.Map.GetHex(dwellingHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected trading house, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PowerValue != 2 {
		t.Errorf("expected power value 2, got %d", mapHex.Building.PowerValue)
	}
}

func TestUpgradeBuilding_TradingHouseDiscount(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewSwarmlings() // Changed from Cultists (Plains) to Swarmlings (Lake)
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	
	// Place player1's dwelling at (0, 1)
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction1.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Place player2's dwelling adjacent at (1, 0)
	player2Hex := NewHex(1, 0)
	gs.Map.GetHex(player2Hex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 1,
	}
	
	// Get base cost
	tradingHouseCost := player1.Faction.GetTradingHouseCost()
	baseCoinCost := tradingHouseCost.Coins
	discountedCoinCost := baseCoinCost / 2
	
	// Give player just enough for discounted cost
	player1.Resources.Coins = discountedCoinCost
	player1.Resources.Workers = tradingHouseCost.Workers
	
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed with discount, got error: %v", err)
	}
	
	// Verify building was upgraded
	mapHex := gs.Map.GetHex(dwellingHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected trading house, got %v", mapHex.Building.Type)
	}
	
	// Verify coins were spent (should be 0 now)
	if player1.Resources.Coins != 0 {
		t.Errorf("expected 0 coins remaining, got %d", player1.Resources.Coins)
	}
}

func TestUpgradeBuilding_TradingHouseToTemple(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Place a trading house at (0, 1)
	tradingHouseHex := NewHex(0, 1)
	gs.Map.GetHex(tradingHouseHex).Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}
	
	// Give player enough resources for upgrade
	templeCost := player.Faction.GetTempleCost()
	player.Resources.Coins = templeCost.Coins + 10
	player.Resources.Workers = templeCost.Workers + 10
	
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingTemple)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify building was upgraded
	mapHex := gs.Map.GetHex(tradingHouseHex)
	if mapHex.Building.Type != models.BuildingTemple {
		t.Errorf("expected temple, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PowerValue != 2 {
		t.Errorf("expected power value 2, got %d", mapHex.Building.PowerValue)
	}
}

func TestUpgradeBuilding_TradingHouseToStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Place a trading house at (0, 1)
	tradingHouseHex := NewHex(0, 1)
	gs.Map.GetHex(tradingHouseHex).Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}
	
	// Give player enough resources for upgrade
	strongholdCost := player.Faction.GetStrongholdCost()
	player.Resources.Coins = strongholdCost.Coins + 10
	player.Resources.Workers = strongholdCost.Workers + 10
	
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify building was upgraded
	mapHex := gs.Map.GetHex(tradingHouseHex)
	if mapHex.Building.Type != models.BuildingStronghold {
		t.Errorf("expected stronghold, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PowerValue != 3 {
		t.Errorf("expected power value 3, got %d", mapHex.Building.PowerValue)
	}
}

func TestUpgradeBuilding_TempleToSanctuary(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Place a temple at (0, 1)
	templeHex := NewHex(0, 1)
	gs.Map.GetHex(templeHex).Building = &models.Building{
		Type:       models.BuildingTemple,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}
	
	// Give player enough resources for upgrade
	sanctuaryCost := player.Faction.GetSanctuaryCost()
	player.Resources.Coins = sanctuaryCost.Coins + 10
	player.Resources.Workers = sanctuaryCost.Workers + 10
	
	action := NewUpgradeBuildingAction("player1", templeHex, models.BuildingSanctuary)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify building was upgraded
	mapHex := gs.Map.GetHex(templeHex)
	if mapHex.Building.Type != models.BuildingSanctuary {
		t.Errorf("expected sanctuary, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PowerValue != 3 {
		t.Errorf("expected power value 3, got %d", mapHex.Building.PowerValue)
	}
}

func TestUpgradeBuilding_InvalidUpgradePath(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Place a dwelling at (0, 1)
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Give player resources
	player.Resources.Coins = 100
	player.Resources.Workers = 100
	
	// Try to upgrade dwelling directly to stronghold (invalid)
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingStronghold)
	
	err := action.Execute(gs)
	if err == nil {
		t.Fatalf("expected error for invalid upgrade path")
	}
}

func TestUpgradeBuilding_InsufficientResources(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	
	// Place a dwelling at (0, 1)
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Give player insufficient resources
	player.Resources.Coins = 1
	player.Resources.Workers = 1
	
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err == nil {
		t.Fatalf("expected error for insufficient resources")
	}
}

func TestUpgradeBuilding_BuildingLimitTradingHouse(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 1000
	player.Resources.Workers = 1000
	
	// Place 4 trading houses (the limit)
	for i := 0; i < 4; i++ {
		hex := NewHex(i, 1)
		gs.Map.GetHex(hex).Building = &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		}
	}
	
	// Try to upgrade a 5th dwelling to trading house
	dwellingHex := NewHex(5, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err == nil {
		t.Fatalf("expected error for building limit reached")
	}
}

func TestUpgradeBuilding_BuildingLimitStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 1000
	player.Resources.Workers = 1000
	
	// Place 1 stronghold (the limit)
	hex1 := NewHex(0, 1)
	gs.Map.GetHex(hex1).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	}
	
	// Try to upgrade another trading house to stronghold
	tradingHouseHex := NewHex(2, 1)
	gs.Map.GetHex(tradingHouseHex).Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}
	
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	
	err := action.Execute(gs)
	if err == nil {
		t.Fatalf("expected error for stronghold limit reached")
	}
}

func TestUpgradeBuilding_PowerLeech(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewSwarmlings() // Changed from Cultists (Plains) to Swarmlings (Lake)
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	_ = gs.GetPlayer("player2") // player2 exists but we don't need to use it
	
	// Place player1's dwelling at (0, 1)
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction1.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Place player2's dwelling adjacent at (1, 0)
	player2Hex := NewHex(1, 0)
	gs.Map.GetHex(player2Hex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 1,
	}
	
	// Give player1 resources
	tradingHouseCost := player1.Faction.GetTradingHouseCost()
	player1.Resources.Coins = tradingHouseCost.Coins
	player1.Resources.Workers = tradingHouseCost.Workers
	
	// Upgrade player1's dwelling to trading house
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify player2 has a pending leech offer
	offers := gs.GetPendingLeechOffers("player2")
	if len(offers) == 0 {
		t.Fatalf("expected player2 to have a pending leech offer")
	}
	
	offer := offers[0]
	// Player2's dwelling (power 1) is adjacent to the upgraded building
	if offer.Amount != 1 {
		t.Errorf("expected offer amount of 1, got %d", offer.Amount)
	}
	if offer.VPCost != 0 {
		t.Errorf("expected VP cost of 0, got %d", offer.VPCost)
	}
	if offer.FromPlayerID != "player1" {
		t.Errorf("expected offer from player1, got %s", offer.FromPlayerID)
	}
}

func TestUpgradeBuilding_PowerLeechMultipleBuildings(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewSwarmlings() // Changed from Cultists (Plains) to Swarmlings (Lake)
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	_ = gs.GetPlayer("player2") // player2 exists but we don't need to use it
	
	// Place player1's dwelling at (1, 2)
	dwellingHex := NewHex(1, 2)
	gs.Map.GetHex(dwellingHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction1.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	
	// Place player2's Temple at (1, 1) - adjacent to (1,2)
	player2Temple := NewHex(1, 1)
	gs.Map.GetHex(player2Temple).Building = &models.Building{
		Type:       models.BuildingTemple,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 2,
	}
	
	// Place player2's Stronghold at (2, 2) - also adjacent to (1,2)
	player2Stronghold := NewHex(2, 2)
	gs.Map.GetHex(player2Stronghold).Building = &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 3,
	}
	
	// Give player1 resources
	tradingHouseCost := player1.Faction.GetTradingHouseCost()
	player1.Resources.Coins = tradingHouseCost.Coins / 2 // Discounted due to adjacent opponent
	player1.Resources.Workers = tradingHouseCost.Workers
	
	// Upgrade player1's dwelling to trading house
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify player2 has ONE leech offer with TOTAL power from both buildings
	offers := gs.GetPendingLeechOffers("player2")
	if len(offers) != 1 {
		t.Fatalf("expected player2 to have exactly 1 leech offer, got %d", len(offers))
	}
	
	offer := offers[0]
	// Total power should be Temple (2) + Stronghold (3) = 5
	if offer.Amount != 5 {
		t.Errorf("expected offer amount of 5 (2 from temple + 3 from stronghold), got %d", offer.Amount)
	}
	if offer.VPCost != 4 {
		t.Errorf("expected VP cost of 4, got %d", offer.VPCost)
	}
}

func TestUpgradeBuilding_FreesUpDwellingSlot(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 1000
	player.Resources.Workers = 1000
	player.Resources.Priests = 5
	
	// Place 8 dwellings (the limit)
	dwellingHexes := []Hex{
		NewHex(0, 1),
		NewHex(1, 1),
		NewHex(2, 1),
		NewHex(3, 1),
		NewHex(4, 1),
		NewHex(0, 2),
		NewHex(1, 2),
		NewHex(2, 2),
	}
	
	for _, hex := range dwellingHexes {
		gs.Map.GetHex(hex).Building = &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		}
	}
	
	// Verify we cannot build another dwelling (limit reached)
	newDwellingHex := NewHex(3, 2)
	gs.Map.TransformTerrain(newDwellingHex, faction.GetHomeTerrain())
	buildAction := NewTransformAndBuildAction("player1", newDwellingHex, true)
	
	err := buildAction.Execute(gs)
	if err == nil {
		t.Fatalf("expected error when building 9th dwelling")
	}
	
	// Upgrade one dwelling to trading house
	upgradeHex := dwellingHexes[0]
	upgradeAction := NewUpgradeBuildingAction("player1", upgradeHex, models.BuildingTradingHouse)
	
	err = upgradeAction.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify the building was upgraded
	mapHex := gs.Map.GetHex(upgradeHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected trading house, got %v", mapHex.Building.Type)
	}
	
	// Count dwellings - should be 7 now
	dwellingCount := 0
	for _, hex := range gs.Map.Hexes {
		if hex.Building != nil && hex.Building.PlayerID == "player1" && hex.Building.Type == models.BuildingDwelling {
			dwellingCount++
		}
	}
	if dwellingCount != 7 {
		t.Errorf("expected 7 dwellings after upgrade, got %d", dwellingCount)
	}
	
	// Now we should be able to build another dwelling (slot freed up)
	buildAction2 := NewTransformAndBuildAction("player1", newDwellingHex, true)
	
	err = buildAction2.Execute(gs)
	if err != nil {
		t.Fatalf("expected to build dwelling after upgrade freed up slot, got error: %v", err)
	}
	
	// Verify the new dwelling was built
	newMapHex := gs.Map.GetHex(newDwellingHex)
	if newMapHex.Building == nil {
		t.Errorf("expected dwelling to be built")
	}
	if newMapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", newMapHex.Building.Type)
	}
	
	// Count dwellings - should be 8 again
	dwellingCount = 0
	for _, hex := range gs.Map.Hexes {
		if hex.Building != nil && hex.Building.PlayerID == "player1" && hex.Building.Type == models.BuildingDwelling {
			dwellingCount++
		}
	}
	if dwellingCount != 8 {
		t.Errorf("expected 8 dwellings after building new one, got %d", dwellingCount)
	}
}
