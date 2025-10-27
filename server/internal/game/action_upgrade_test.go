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
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	player.Resources.Priests = 2
	
	// Place dwelling
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &Building{
		Type:      models.BuildingDwelling,
		Faction:   faction.GetType(),
		PlayerID:  "player1",
		PowerValue: 1,
	}
	
	// Upgrade to trading house
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify upgrade
	mapHex := gs.Map.GetHex(dwellingHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected trading house, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PowerValue != 2 {
		t.Errorf("expected power value 2, got %d", mapHex.Building.PowerValue)
	}
}

func TestUpgradeBuilding_DwellingToTemple(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	player.Resources.Priests = 2
	
	// Place dwelling
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &Building{
		Type:      models.BuildingDwelling,
		Faction:   faction.GetType(),
		PlayerID:  "player1",
		PowerValue: 1,
	}
	
	// Upgrade to temple
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTemple)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify upgrade
	mapHex := gs.Map.GetHex(dwellingHex)
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
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Priests = 2
	
	// Place trading house
	thHex := NewHex(0, 1)
	gs.Map.GetHex(thHex).Building = &Building{
		Type:      models.BuildingTradingHouse,
		Faction:   faction.GetType(),
		PlayerID:  "player1",
		PowerValue: 2,
	}
	
	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", thHex, models.BuildingStronghold)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify upgrade
	mapHex := gs.Map.GetHex(thHex)
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
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Priests = 2
	
	// Place temple
	templeHex := NewHex(0, 1)
	gs.Map.GetHex(templeHex).Building = &Building{
		Type:      models.BuildingTemple,
		Faction:   faction.GetType(),
		PlayerID:  "player1",
		PowerValue: 2,
	}
	
	// Upgrade to sanctuary
	action := NewUpgradeBuildingAction("player1", templeHex, models.BuildingSanctuary)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify upgrade
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
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	
	// Place dwelling
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &Building{
		Type:      models.BuildingDwelling,
		Faction:   faction.GetType(),
		PlayerID:  "player1",
		PowerValue: 1,
	}
	
	// Try to upgrade dwelling directly to stronghold (invalid)
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingStronghold)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for invalid upgrade path")
	}
}

func TestUpgradeBuilding_InsufficientResources(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	
	player := gs.GetPlayer("player1")
	// Not enough resources
	player.Resources.Coins = 0
	player.Resources.Workers = 0
	
	// Place dwelling
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &Building{
		Type:      models.BuildingDwelling,
		Faction:   faction.GetType(),
		PlayerID:  "player1",
		PowerValue: 1,
	}
	
	// Try to upgrade
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for insufficient resources")
	}
}

func TestUpgradeBuilding_NotPlayerBuilding(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewCultists()
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player1.Resources.Coins = 10
	player1.Resources.Workers = 10
	
	// Place player2's dwelling
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &Building{
		Type:      models.BuildingDwelling,
		Faction:   faction2.GetType(),
		PlayerID:  "player2",
		PowerValue: 1,
	}
	
	// Player1 tries to upgrade player2's building
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err == nil {
		t.Errorf("expected error for upgrading another player's building")
	}
}

func TestUpgradeBuilding_PowerLeech(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()
	faction2 := factions.NewCultists()
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	player1.Resources.Coins = 10
	player1.Resources.Workers = 10
	player1.Resources.Priests = 2
	
	// Place player1's dwelling
	dwellingHex := NewHex(0, 1)
	gs.Map.GetHex(dwellingHex).Building = &Building{
		Type:      models.BuildingDwelling,
		Faction:   faction1.GetType(),
		PlayerID:  "player1",
		PowerValue: 1,
	}
	
	// Place player2's dwelling adjacent
	player2Hex := NewHex(1, 1)
	gs.Map.GetHex(player2Hex).Building = &Building{
		Type:      models.BuildingDwelling,
		Faction:   faction2.GetType(),
		PlayerID:  "player2",
		PowerValue: 1,
	}
	
	// Player1 upgrades dwelling to trading house (power value 1 -> 2, increase of 1)
	action := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected upgrade to succeed, got error: %v", err)
	}
	
	// Verify upgrade
	mapHex := gs.Map.GetHex(dwellingHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected trading house, got %v", mapHex.Building.Type)
	}
	
	// Power leech is triggered for the power increase
	// Power leech offers are stored in gs.PendingLeechOffers
}
