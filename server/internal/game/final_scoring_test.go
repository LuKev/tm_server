package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestCalculateFinalScoring_Complete(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	gs.AddPlayer("player2", faction)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	// Set base VP
	player1.VictoryPoints = 50
	player2.VictoryPoints = 45
	
	// Build some buildings for player1 (connected area of 3)
	hex1 := NewHex(0, 0)
	hex2 := NewHex(1, 0)
	hex3 := NewHex(2, 0)
	gs.Map.GetHex(hex1).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(hex2).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(hex3).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	gs.Map.PlaceBuilding(hex2, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	gs.Map.PlaceBuilding(hex3, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Build for player2 (connected area of 2)
	hex4 := NewHex(5, 5)
	hex5 := NewHex(6, 5)
	gs.Map.GetHex(hex4).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(hex5).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex4, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player2",
		PowerValue: 1,
	})
	gs.Map.PlaceBuilding(hex5, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player2",
		PowerValue: 1,
	})
	
	// Set cult positions
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 8, player2)
	
	// Set resources
	player1.Resources.Coins = 9  // 3 VP
	player1.Resources.Workers = 2 // 2 VP
	player1.Resources.Priests = 1 // 1 VP
	
	player2.Resources.Coins = 6  // 2 VP
	player2.Resources.Workers = 3 // 3 VP
	player2.Resources.Priests = 0 // 0 VP
	
	// Calculate final scoring
	scores := gs.CalculateFinalScoring()
	
	// Verify player1
	if scores["player1"].BaseVP != 50 {
		t.Errorf("player1 BaseVP: expected 50, got %d", scores["player1"].BaseVP)
	}
	if scores["player1"].AreaVP != 18 {
		t.Errorf("player1 AreaVP: expected 18, got %d", scores["player1"].AreaVP)
	}
	if scores["player1"].CultVP != 8 {
		t.Errorf("player1 CultVP: expected 8, got %d", scores["player1"].CultVP)
	}
	if scores["player1"].ResourceVP != 6 {
		t.Errorf("player1 ResourceVP: expected 6 (3+2+1), got %d", scores["player1"].ResourceVP)
	}
	if scores["player1"].TotalVP != 82 {
		t.Errorf("player1 TotalVP: expected 82, got %d", scores["player1"].TotalVP)
	}
	
	// Verify player2
	if scores["player2"].CultVP != 4 {
		t.Errorf("player2 CultVP: expected 4, got %d", scores["player2"].CultVP)
	}
	if scores["player2"].ResourceVP != 5 {
		t.Errorf("player2 ResourceVP: expected 5 (2+3+0), got %d", scores["player2"].ResourceVP)
	}
}

func TestAreaBonus_SingleWinner(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	gs.AddPlayer("player2", faction)
	
	// Player1: 5 connected buildings
	for i := 0; i < 5; i++ {
		hex := NewHex(i, 0)
		gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		})
	}
	
	// Player2: 3 connected buildings
	for i := 0; i < 3; i++ {
		hex := NewHex(i, 5)
		gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player2",
			PowerValue: 1,
		})
	}
	
	scores := make(map[string]*PlayerFinalScore)
	scores["player1"] = &PlayerFinalScore{PlayerID: "player1"}
	scores["player2"] = &PlayerFinalScore{PlayerID: "player2"}
	
	gs.calculateAreaBonuses(scores)
	
	// Player1 should get 18 VP
	if scores["player1"].AreaVP != 18 {
		t.Errorf("player1: expected 18 VP, got %d", scores["player1"].AreaVP)
	}
	if scores["player1"].LargestAreaSize != 5 {
		t.Errorf("player1: expected area size 5, got %d", scores["player1"].LargestAreaSize)
	}
	
	// Player2 should get 0 VP
	if scores["player2"].AreaVP != 0 {
		t.Errorf("player2: expected 0 VP, got %d", scores["player2"].AreaVP)
	}
}

func TestAreaBonus_Tie(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	gs.AddPlayer("player2", faction)
	gs.AddPlayer("player3", faction)
	
	// All players: 4 connected buildings each
	for playerID := range gs.Players {
		row := 0
		if playerID == "player2" {
			row = 3
		} else if playerID == "player3" {
			row = 6
		}
		
		for i := 0; i < 4; i++ {
			hex := NewHex(i, row)
			gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
			gs.Map.PlaceBuilding(hex, &models.Building{
				Type:       models.BuildingDwelling,
				Faction:    faction.GetType(),
				PlayerID:   playerID,
				PowerValue: 1,
			})
		}
	}
	
	scores := make(map[string]*PlayerFinalScore)
	for playerID := range gs.Players {
		scores[playerID] = &PlayerFinalScore{PlayerID: playerID}
	}
	
	gs.calculateAreaBonuses(scores)
	
	// All players tied: 18 / 3 = 6 VP each
	for playerID, score := range scores {
		if score.AreaVP != 6 {
			t.Errorf("%s: expected 6 VP, got %d", playerID, score.AreaVP)
		}
	}
}

func TestCultBonus_SingleTrack(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	gs.AddPlayer("player2", faction)
	gs.AddPlayer("player3", faction)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")
	
	// Fire track: player1=10, player2=8, player3=5
	player1.Keys = 1
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 8, player2)
	gs.CultTracks.AdvancePlayer("player3", CultFire, 5, player3)
	
	scores := make(map[string]*PlayerFinalScore)
	for playerID := range gs.Players {
		scores[playerID] = &PlayerFinalScore{PlayerID: playerID}
	}
	
	gs.calculateCultBonuses(scores)
	
	// Player1: 8 VP (1st place)
	if scores["player1"].CultVP != 8 {
		t.Errorf("player1: expected 8 VP, got %d", scores["player1"].CultVP)
	}
	
	// Player2: 4 VP (2nd place)
	if scores["player2"].CultVP != 4 {
		t.Errorf("player2: expected 4 VP, got %d", scores["player2"].CultVP)
	}
	
	// Player3: 2 VP (3rd place)
	if scores["player3"].CultVP != 2 {
		t.Errorf("player3: expected 2 VP, got %d", scores["player3"].CultVP)
	}
}

func TestCultBonus_TieForFirst(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	gs.AddPlayer("player2", faction)
	gs.AddPlayer("player3", faction)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")
	
	// Fire track: player1=9, player2=9, player3=5 (both tied for 1st)
	// Note: Position 10 can only be occupied by one player
	gs.CultTracks.AdvancePlayer("player1", CultFire, 9, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 9, player2)
	gs.CultTracks.AdvancePlayer("player3", CultFire, 5, player3)
	
	scores := make(map[string]*PlayerFinalScore)
	for playerID := range gs.Players {
		scores[playerID] = &PlayerFinalScore{PlayerID: playerID}
	}
	
	gs.calculateCultBonuses(scores)
	
	// Player1 and Player2 tied for 1st: (8+4)/2 = 6 VP each
	if scores["player1"].CultVP != 6 {
		t.Errorf("player1: expected 6 VP, got %d", scores["player1"].CultVP)
	}
	if scores["player2"].CultVP != 6 {
		t.Errorf("player2: expected 6 VP, got %d", scores["player2"].CultVP)
	}
	
	// Player3: 2 VP (3rd place)
	if scores["player3"].CultVP != 2 {
		t.Errorf("player3: expected 2 VP, got %d", scores["player3"].CultVP)
	}
}

func TestCultBonus_MultipleTracks(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	gs.AddPlayer("player2", faction)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	// Player1: 1st on Fire, 2nd on Water
	player1.Keys = 2
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 7, player1)
	
	// Player2: 1st on Water, 2nd on Fire
	player2.Keys = 1
	gs.CultTracks.AdvancePlayer("player2", CultFire, 8, player2)
	gs.CultTracks.AdvancePlayer("player2", CultWater, 10, player2)
	
	scores := make(map[string]*PlayerFinalScore)
	scores["player1"] = &PlayerFinalScore{PlayerID: "player1"}
	scores["player2"] = &PlayerFinalScore{PlayerID: "player2"}
	
	gs.calculateCultBonuses(scores)
	
	// Player1: 8 (Fire 1st) + 4 (Water 2nd) = 12 VP
	if scores["player1"].CultVP != 12 {
		t.Errorf("player1: expected 12 VP, got %d", scores["player1"].CultVP)
	}
	
	// Player2: 4 (Fire 2nd) + 8 (Water 1st) = 12 VP
	if scores["player2"].CultVP != 12 {
		t.Errorf("player2: expected 12 VP, got %d", scores["player2"].CultVP)
	}
}

func TestResourceConversion(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set resources
	player.Resources.Coins = 10  // 10/3 = 3 VP
	player.Resources.Workers = 5 // 5 VP
	player.Resources.Priests = 2 // 2 VP
	
	scores := make(map[string]*PlayerFinalScore)
	scores["player1"] = &PlayerFinalScore{PlayerID: "player1"}
	
	gs.calculateResourceConversion(scores)
	
	// Total: 3 + 5 + 2 = 10 VP
	if scores["player1"].ResourceVP != 10 {
		t.Errorf("expected 10 VP, got %d", scores["player1"].ResourceVP)
	}
	
	// Tiebreaker value: 10 + 5 + 2 = 17
	if scores["player1"].TotalResourceValue != 17 {
		t.Errorf("expected resource value 17, got %d", scores["player1"].TotalResourceValue)
	}
}

func TestGetWinner_Clear(t *testing.T) {
	gs := NewGameState()
	
	scores := map[string]*PlayerFinalScore{
		"player1": {PlayerID: "player1", TotalVP: 100, TotalResourceValue: 10},
		"player2": {PlayerID: "player2", TotalVP: 95, TotalResourceValue: 15},
		"player3": {PlayerID: "player3", TotalVP: 90, TotalResourceValue: 20},
	}
	
	winner := gs.GetWinner(scores)
	if winner != "player1" {
		t.Errorf("expected player1 to win, got %s", winner)
	}
}

func TestGetWinner_Tiebreaker(t *testing.T) {
	gs := NewGameState()
	
	scores := map[string]*PlayerFinalScore{
		"player1": {PlayerID: "player1", TotalVP: 100, TotalResourceValue: 10},
		"player2": {PlayerID: "player2", TotalVP: 100, TotalResourceValue: 15},
		"player3": {PlayerID: "player3", TotalVP: 95, TotalResourceValue: 20},
	}
	
	winner := gs.GetWinner(scores)
	if winner != "player2" {
		t.Errorf("expected player2 to win (tiebreaker), got %s", winner)
	}
}

func TestGetRankedPlayers(t *testing.T) {
	scores := map[string]*PlayerFinalScore{
		"player1": {PlayerID: "player1", TotalVP: 95, TotalResourceValue: 10},
		"player2": {PlayerID: "player2", TotalVP: 100, TotalResourceValue: 15},
		"player3": {PlayerID: "player3", TotalVP: 100, TotalResourceValue: 20},
	}
	
	ranked := GetRankedPlayers(scores)
	
	if len(ranked) != 3 {
		t.Fatalf("expected 3 players, got %d", len(ranked))
	}
	
	// 1st: player3 (100 VP, 20 resources)
	if ranked[0].PlayerID != "player3" {
		t.Errorf("1st place: expected player3, got %s", ranked[0].PlayerID)
	}
	
	// 2nd: player2 (100 VP, 15 resources)
	if ranked[1].PlayerID != "player2" {
		t.Errorf("2nd place: expected player2, got %s", ranked[1].PlayerID)
	}
	
	// 3rd: player1 (95 VP)
	if ranked[2].PlayerID != "player1" {
		t.Errorf("3rd place: expected player1, got %s", ranked[2].PlayerID)
	}
}

func TestGetLargestConnectedArea(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Create two separate areas: 3 buildings and 2 buildings
	// Area 1: (0,0), (1,0), (2,0)
	for i := 0; i < 3; i++ {
		hex := NewHex(i, 0)
		gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		})
	}
	
	// Area 2: (5,5), (6,5) - not connected to Area 1
	for i := 5; i < 7; i++ {
		hex := NewHex(i, 5)
		gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		})
	}
	
	largestArea := gs.Map.GetLargestConnectedArea("player1")
	if largestArea != 3 {
		t.Errorf("expected largest area 3, got %d", largestArea)
	}
}
