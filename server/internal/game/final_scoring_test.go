package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func placeFinalScoringTestBuilding(t *testing.T, gs *GameState, playerID string, hex board.Hex, buildingType models.BuildingType) {
	t.Helper()

	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil {
		t.Fatalf("player %s missing faction", playerID)
	}

	mapHex := gs.Map.GetHex(hex)
	if mapHex == nil {
		t.Fatalf("hex %v is not valid on the test map", hex)
	}
	mapHex.Terrain = player.Faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex, &models.Building{
		Type:       buildingType,
		Faction:    player.Faction.GetType(),
		PlayerID:   playerID,
		PowerValue: getStructurePowerValue(player, buildingType),
	})
}

func pickConnectedBorderHexes(t *testing.T, gs *GameState, count int, used map[board.Hex]bool) []board.Hex {
	t.Helper()

	for start := range gs.Map.Hexes {
		if used[start] || !gs.isBorderMapHex(start) {
			continue
		}

		seen := map[board.Hex]bool{start: true}
		queue := []board.Hex{start}
		component := []board.Hex{start}

		for len(queue) > 0 && len(component) < count {
			current := queue[0]
			queue = queue[1:]
			for _, neighbor := range current.Neighbors() {
				if seen[neighbor] || used[neighbor] || !gs.Map.IsValidHex(neighbor) || !gs.isBorderMapHex(neighbor) {
					continue
				}
				seen[neighbor] = true
				queue = append(queue, neighbor)
				component = append(component, neighbor)
				if len(component) == count {
					return component
				}
			}
		}
		if len(component) == count {
			return component
		}
	}

	t.Fatalf("unable to find %d connected border hexes", count)
	return nil
}

func pickStraightHexRun(t *testing.T, gs *GameState, count int, used map[board.Hex]bool) []board.Hex {
	t.Helper()

	for start := range gs.Map.Hexes {
		if used[start] {
			continue
		}
		for direction := 0; direction < len(board.DirectionVectors); direction++ {
			run := make([]board.Hex, 0, count)
			current := start
			valid := true
			for len(run) < count {
				if used[current] || !gs.Map.IsValidHex(current) {
					valid = false
					break
				}
				run = append(run, current)
				current = current.Neighbor(direction)
			}
			if valid && len(run) == count {
				return run
			}
		}
	}

	t.Fatalf("unable to find a straight run of %d hexes", count)
	return nil
}

func TestCalculateFinalScoring_Complete(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren()      // Forest
	faction2 := factions.NewSwarmlings() // Lake - different from Auren, no special resource conversion
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")

	// Set base VP
	player1.VictoryPoints = 50
	player2.VictoryPoints = 45

	// Build some buildings for player1 (connected area of 3)
	hex1 := board.NewHex(0, 0)
	hex2 := board.NewHex(1, 0)
	hex3 := board.NewHex(2, 0)
	gs.Map.GetHex(hex1).Terrain = faction1.GetHomeTerrain()
	gs.Map.GetHex(hex2).Terrain = faction1.GetHomeTerrain()
	gs.Map.GetHex(hex3).Terrain = faction1.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction1.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	gs.Map.PlaceBuilding(hex2, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction1.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	gs.Map.PlaceBuilding(hex3, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction1.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// Build for player2 (connected area of 2)
	hex4 := board.NewHex(5, 5)
	hex5 := board.NewHex(6, 5)
	gs.Map.GetHex(hex4).Terrain = faction2.GetHomeTerrain()
	gs.Map.GetHex(hex5).Terrain = faction2.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex4, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 1,
	})
	gs.Map.PlaceBuilding(hex5, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2.GetType(),
		PlayerID:   "player2",
		PowerValue: 1,
	})

	// Reset cult positions to 0
	for _, p := range gs.Players {
		p.CultPositions = map[CultTrack]int{
			CultFire: 0, CultWater: 0, CultEarth: 0, CultAir: 0,
		}
		gs.CultTracks.PlayerPositions[p.ID][CultFire] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultWater] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultEarth] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultAir] = 0
	}

	// Set cult positions
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1, gs)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 8, player2, gs)

	// Set resources (clear power bowls first since Auren starts with power)
	player1.Resources.Power.Bowl1 = 0
	player1.Resources.Power.Bowl2 = 0
	player1.Resources.Power.Bowl3 = 0
	// Resources convert: workers/priests -> coins, then coins -> VP (3:1)
	player1.Resources.Coins = 9   // 9 coins
	player1.Resources.Workers = 2 // 2 coins
	player1.Resources.Priests = 1 // 1 coin
	// Total: 12 coins -> 4 VP

	player2.Resources.Power.Bowl1 = 0
	player2.Resources.Power.Bowl2 = 0
	player2.Resources.Power.Bowl3 = 0
	player2.Resources.Coins = 6   // 6 coins
	player2.Resources.Workers = 3 // 3 coins
	player2.Resources.Priests = 0 // 0 coins
	// Total: 9 coins -> 3 VP

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
	if scores["player1"].ResourceVP != 4 {
		t.Errorf("player1 ResourceVP: expected 4 (12 coins / 3), got %d", scores["player1"].ResourceVP)
	}
	if scores["player1"].TotalVP != 80 {
		t.Errorf("player1 TotalVP: expected 80, got %d", scores["player1"].TotalVP)
	}

	// Verify player2
	if scores["player2"].CultVP != 4 {
		t.Errorf("player2 CultVP: expected 4, got %d", scores["player2"].CultVP)
	}
	if scores["player2"].ResourceVP != 3 {
		t.Errorf("player2 ResourceVP: expected 3 (9 coins / 3), got %d", scores["player2"].ResourceVP)
	}
}

func TestAreaBonus_SingleWinner(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren()      // Forest
	faction2 := factions.NewSwarmlings() // Lake
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)

	// Player1: 5 connected buildings
	for i := 0; i < 5; i++ {
		hex := board.NewHex(i, 0)
		gs.Map.GetHex(hex).Terrain = faction1.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction1.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		})
	}

	// Player2: 3 connected buildings
	for i := 0; i < 3; i++ {
		hex := board.NewHex(i, 5)
		gs.Map.GetHex(hex).Terrain = faction2.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction2.GetType(),
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

	// Player2 should get 12 VP (2nd place)
	if scores["player2"].AreaVP != 12 {
		t.Errorf("player2: expected 12 VP, got %d", scores["player2"].AreaVP)
	}
}

func TestAreaBonus_Tie(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren()      // Forest
	faction2 := factions.NewSwarmlings() // Lake
	faction3 := factions.NewHalflings()  // Plains
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	gs.AddPlayer("player3", faction3)

	// All players: 4 connected buildings each
	for playerID, player := range gs.Players {
		row := 0
		if playerID == "player2" {
			row = 3
		} else if playerID == "player3" {
			row = 6
		}

		for i := 0; i < 4; i++ {
			hex := board.NewHex(i, row)
			gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
			gs.Map.PlaceBuilding(hex, &models.Building{
				Type:       models.BuildingDwelling,
				Faction:    player.Faction.GetType(),
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

	// All players tied for 1st: (18 + 12 + 6) / 3 = 12 VP each
	for playerID, score := range scores {
		if score.AreaVP != 12 {
			t.Errorf("%s: expected 12 VP, got %d", playerID, score.AreaVP)
		}
	}
}

func TestCultBonus_SingleTrack(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren()      // Forest
	faction2 := factions.NewSwarmlings() // Lake
	faction3 := factions.NewHalflings()  // Plains
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	gs.AddPlayer("player3", faction3)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")

	// Reset cult positions to 0 to avoid interference from starting cults
	for _, p := range gs.Players {
		p.CultPositions = map[CultTrack]int{
			CultFire: 0, CultWater: 0, CultEarth: 0, CultAir: 0,
		}
		gs.CultTracks.PlayerPositions[p.ID][CultFire] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultWater] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultEarth] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultAir] = 0
	}

	// Fire track: player1=10, player2=8, player3=5
	player1.Keys = 1
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1, gs)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 8, player2, gs)
	gs.CultTracks.AdvancePlayer("player3", CultFire, 5, player3, gs)

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
	faction1 := factions.NewAuren()      // Forest
	faction2 := factions.NewSwarmlings() // Lake
	faction3 := factions.NewHalflings()  // Plains
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	gs.AddPlayer("player3", faction3)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")

	// Reset cult positions to 0
	for _, p := range gs.Players {
		p.CultPositions = map[CultTrack]int{
			CultFire: 0, CultWater: 0, CultEarth: 0, CultAir: 0,
		}
		gs.CultTracks.PlayerPositions[p.ID][CultFire] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultWater] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultEarth] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultAir] = 0
	}

	// Fire track: player1=9, player2=9, player3=5 (both tied for 1st)
	// Note: Position 10 can only be occupied by one player
	gs.CultTracks.AdvancePlayer("player1", CultFire, 9, player1, gs)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 9, player2, gs)
	gs.CultTracks.AdvancePlayer("player3", CultFire, 5, player3, gs)

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
	faction1 := factions.NewAuren()      // Forest
	faction2 := factions.NewSwarmlings() // Lake
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")

	// Reset cult positions to 0
	for _, p := range gs.Players {
		p.CultPositions = map[CultTrack]int{
			CultFire: 0, CultWater: 0, CultEarth: 0, CultAir: 0,
		}
		gs.CultTracks.PlayerPositions[p.ID][CultFire] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultWater] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultEarth] = 0
		gs.CultTracks.PlayerPositions[p.ID][CultAir] = 0
	}

	// Player1: 1st on Fire, 2nd on Water
	player1.Keys = 2
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1, gs)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 7, player1, gs)

	// Player2: 1st on Water, 2nd on Fire
	player2.Keys = 1
	gs.CultTracks.AdvancePlayer("player2", CultFire, 8, player2, gs)
	gs.CultTracks.AdvancePlayer("player2", CultWater, 10, player2, gs)

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

	// Set resources - all convert to coins first, then coins -> VP (3:1)
	player.Resources.Coins = 10      // 10 coins
	player.Resources.Power.Bowl2 = 6 // 6/2 = 3 coins (burn to Bowl3, convert)
	player.Resources.Power.Bowl3 = 0
	player.Resources.Workers = 5 // 5 coins
	player.Resources.Priests = 2 // 2 coins
	// Total: 10 + 3 + 5 + 2 = 20 coins -> 6 VP

	scores := make(map[string]*PlayerFinalScore)
	scores["player1"] = &PlayerFinalScore{PlayerID: "player1"}

	gs.calculateResourceConversion(scores)

	if scores["player1"].ResourceVP != 6 {
		t.Errorf("expected 6 VP (20 coins / 3), got %d", scores["player1"].ResourceVP)
	}

	// Tiebreaker value: 20 coins total
	if scores["player1"].TotalResourceValue != 20 {
		t.Errorf("expected resource value 20, got %d", scores["player1"].TotalResourceValue)
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
	player := gs.GetPlayer("player1")

	// Create two separate areas: 3 buildings and 2 buildings
	// Area 1: (0,0), (1,0), (2,0)
	for i := 0; i < 3; i++ {
		hex := board.NewHex(i, 0)
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
		hex := board.NewHex(i, 5)
		gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		})
	}

	largestArea := gs.Map.GetLargestConnectedArea("player1", player.Faction, player.ShippingLevel)
	if largestArea != 3 {
		t.Errorf("expected largest area 3, got %d", largestArea)
	}
}

func TestResourceConversion_WithPower(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set resources - all convert to coins, then coins -> VP (3:1)
	player.Resources.Coins = 3        // 3 coins
	player.Resources.Power.Bowl2 = 10 // 10/2 = 5 coins (burn to Bowl 3, then convert)
	player.Resources.Power.Bowl3 = 4  // 4 coins (direct conversion)
	player.Resources.Workers = 2      // 2 coins
	player.Resources.Priests = 1      // 1 coin
	// Total: 3 + 5 + 4 + 2 + 1 = 15 coins -> 5 VP

	scores := make(map[string]*PlayerFinalScore)
	scores["player1"] = &PlayerFinalScore{PlayerID: "player1"}

	gs.calculateResourceConversion(scores)

	if scores["player1"].ResourceVP != 5 {
		t.Errorf("expected 5 VP (15 coins / 3), got %d", scores["player1"].ResourceVP)
	}
}

func TestResourceConversion_ChildrenOfTheWyrmUsesSpecialFinalBurn(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("player1", factions.NewChildrenOfTheWyrm()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 5
	player.Resources.Power.Bowl2 = 3
	player.Resources.Power.Bowl3 = 0
	player.Resources.Workers = 0
	player.Resources.Priests = 0

	scores := map[string]*PlayerFinalScore{
		"player1": {PlayerID: "player1"},
	}
	gs.calculateResourceConversion(scores)

	if scores["player1"].TotalResourceValue != 7 {
		t.Fatalf("resource value = %d, want 7", scores["player1"].TotalResourceValue)
	}
	if scores["player1"].ResourceVP != 2 {
		t.Fatalf("resource VP = %d, want 2", scores["player1"].ResourceVP)
	}
}

func TestResourceConversion_ChildrenOfTheWyrmFinalBurnAllowsPartialMove(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("player1", factions.NewChildrenOfTheWyrm()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 0
	player.Resources.Power.Bowl2 = 2
	player.Resources.Power.Bowl3 = 0
	player.Resources.Workers = 0
	player.Resources.Priests = 0

	scores := map[string]*PlayerFinalScore{
		"player1": {PlayerID: "player1"},
	}
	gs.calculateResourceConversion(scores)

	if scores["player1"].TotalResourceValue != 1 {
		t.Fatalf("resource value = %d, want 1", scores["player1"].TotalResourceValue)
	}
	if scores["player1"].ResourceVP != 0 {
		t.Fatalf("resource VP = %d, want 0", scores["player1"].ResourceVP)
	}
}

func TestResourceConversion_Alchemists(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Alchemists: all resources -> coins -> VP at 2:1 rate
	player.Resources.Coins = 8       // 8 coins
	player.Resources.Power.Bowl2 = 4 // 4/2 = 2 coins
	player.Resources.Power.Bowl3 = 0 // 0 coins
	player.Resources.Workers = 1     // 1 coin
	player.Resources.Priests = 0     // 0 coins
	// Total: 8 + 2 + 1 = 11 coins -> 5 VP (Alchemists 2:1)

	scores := make(map[string]*PlayerFinalScore)
	scores["player1"] = &PlayerFinalScore{PlayerID: "player1"}

	gs.calculateResourceConversion(scores)

	if scores["player1"].ResourceVP != 5 {
		t.Errorf("expected 5 VP (11 coins / 2), got %d", scores["player1"].ResourceVP)
	}
}

func TestResourceConversion_AlchemistsWithPower(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Alchemists: all resources -> coins -> VP at 2:1 rate
	player.Resources.Coins = 2       // 2 coins
	player.Resources.Power.Bowl2 = 6 // 6/2 = 3 coins
	player.Resources.Power.Bowl3 = 1 // 1 coin
	player.Resources.Workers = 1     // 1 coin
	player.Resources.Priests = 0
	// Total: 2 + 3 + 1 + 1 = 7 coins -> 3 VP (Alchemists 2:1)

	scores := make(map[string]*PlayerFinalScore)
	scores["player1"] = &PlayerFinalScore{PlayerID: "player1"}

	gs.calculateResourceConversion(scores)

	if scores["player1"].ResourceVP != 3 {
		t.Errorf("expected 3 VP (7 coins / 2), got %d", scores["player1"].ResourceVP)
	}
}

func TestCalculateFinalScoring_FireIceGreatestDistance(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("player1", factions.NewAuren()); err != nil {
		t.Fatalf("add player1: %v", err)
	}
	if err := gs.AddPlayer("player2", factions.NewSwarmlings()); err != nil {
		t.Fatalf("add player2: %v", err)
	}
	if err := gs.AddPlayer("player3", factions.NewHalflings()); err != nil {
		t.Fatalf("add player3: %v", err)
	}

	used := make(map[board.Hex]bool)
	player1Run := pickStraightHexRun(t, gs, 4, used)
	for _, hex := range player1Run {
		used[hex] = true
	}
	player2Run := pickStraightHexRun(t, gs, 5, used)
	for _, hex := range player2Run {
		used[hex] = true
	}
	player3Run := pickStraightHexRun(t, gs, 2, used)

	for _, hex := range player1Run {
		placeFinalScoringTestBuilding(t, gs, "player1", hex, models.BuildingDwelling)
	}
	for _, hex := range player2Run {
		placeFinalScoringTestBuilding(t, gs, "player2", hex, models.BuildingDwelling)
	}
	for _, hex := range player3Run {
		placeFinalScoringTestBuilding(t, gs, "player3", hex, models.BuildingDwelling)
	}

	gs.FireIceFinalScoringTile = FireIceFinalScoringTileGreatestDistance

	scores := gs.CalculateFinalScoring()

	if scores["player2"].FireIceMetricValue != 4 || scores["player2"].FireIceVP != 18 {
		t.Fatalf("player2 Fire & Ice score = (%d, %d), want (4, 18)", scores["player2"].FireIceMetricValue, scores["player2"].FireIceVP)
	}
	if scores["player1"].FireIceMetricValue != 3 || scores["player1"].FireIceVP != 12 {
		t.Fatalf("player1 Fire & Ice score = (%d, %d), want (3, 12)", scores["player1"].FireIceMetricValue, scores["player1"].FireIceVP)
	}
	if scores["player3"].FireIceMetricValue != 1 || scores["player3"].FireIceVP != 6 {
		t.Fatalf("player3 Fire & Ice score = (%d, %d), want (1, 6)", scores["player3"].FireIceMetricValue, scores["player3"].FireIceVP)
	}
}

func TestCalculateFinalScoring_FireIceStrongholdSanctuary(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("player1", factions.NewAuren()); err != nil {
		t.Fatalf("add player1: %v", err)
	}
	if err := gs.AddPlayer("player2", factions.NewSwarmlings()); err != nil {
		t.Fatalf("add player2: %v", err)
	}
	if err := gs.AddPlayer("player3", factions.NewHalflings()); err != nil {
		t.Fatalf("add player3: %v", err)
	}

	placeFinalScoringTestBuilding(t, gs, "player1", board.NewHex(0, 0), models.BuildingStronghold)
	placeFinalScoringTestBuilding(t, gs, "player1", board.NewHex(1, 0), models.BuildingDwelling)
	placeFinalScoringTestBuilding(t, gs, "player1", board.NewHex(2, 0), models.BuildingDwelling)
	placeFinalScoringTestBuilding(t, gs, "player1", board.NewHex(3, 0), models.BuildingSanctuary)

	placeFinalScoringTestBuilding(t, gs, "player2", board.NewHex(0, 4), models.BuildingStronghold)
	placeFinalScoringTestBuilding(t, gs, "player2", board.NewHex(1, 4), models.BuildingSanctuary)

	placeFinalScoringTestBuilding(t, gs, "player3", board.NewHex(0, 7), models.BuildingStronghold)

	gs.FireIceFinalScoringTile = FireIceFinalScoringTileStrongholdSanctuary

	scores := gs.CalculateFinalScoring()

	if scores["player1"].FireIceMetricValue != 3 || scores["player1"].FireIceVP != 18 {
		t.Fatalf("player1 Fire & Ice score = (%d, %d), want (3, 18)", scores["player1"].FireIceMetricValue, scores["player1"].FireIceVP)
	}
	if scores["player2"].FireIceMetricValue != 1 || scores["player2"].FireIceVP != 12 {
		t.Fatalf("player2 Fire & Ice score = (%d, %d), want (1, 12)", scores["player2"].FireIceMetricValue, scores["player2"].FireIceVP)
	}
	if scores["player3"].FireIceMetricValue != 0 || scores["player3"].FireIceVP != 0 {
		t.Fatalf("player3 Fire & Ice score = (%d, %d), want (0, 0)", scores["player3"].FireIceMetricValue, scores["player3"].FireIceVP)
	}
}

func TestCalculateFinalScoring_FireIceOutposts(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("player1", factions.NewAuren()); err != nil {
		t.Fatalf("add player1: %v", err)
	}
	if err := gs.AddPlayer("player2", factions.NewSwarmlings()); err != nil {
		t.Fatalf("add player2: %v", err)
	}
	if err := gs.AddPlayer("player3", factions.NewHalflings()); err != nil {
		t.Fatalf("add player3: %v", err)
	}

	used := make(map[board.Hex]bool)
	player1Hexes := pickConnectedBorderHexes(t, gs, 3, used)
	for _, hex := range player1Hexes {
		used[hex] = true
	}
	player2Hexes := pickConnectedBorderHexes(t, gs, 2, used)
	for _, hex := range player2Hexes {
		used[hex] = true
	}
	player3Hexes := pickConnectedBorderHexes(t, gs, 1, used)

	for _, hex := range player1Hexes {
		if !gs.isBorderMapHex(hex) {
			t.Fatalf("expected %v to be a border hex", hex)
		}
		placeFinalScoringTestBuilding(t, gs, "player1", hex, models.BuildingDwelling)
	}
	for _, hex := range player2Hexes {
		if !gs.isBorderMapHex(hex) {
			t.Fatalf("expected %v to be a border hex", hex)
		}
		placeFinalScoringTestBuilding(t, gs, "player2", hex, models.BuildingDwelling)
	}
	if !gs.isBorderMapHex(player3Hexes[0]) {
		t.Fatalf("expected %v to be a border hex", player3Hexes[0])
	}
	placeFinalScoringTestBuilding(t, gs, "player3", player3Hexes[0], models.BuildingDwelling)

	gs.FireIceFinalScoringTile = FireIceFinalScoringTileOutposts

	scores := gs.CalculateFinalScoring()

	if scores["player1"].FireIceMetricValue != 3 || scores["player1"].FireIceVP != 18 {
		t.Fatalf("player1 Fire & Ice score = (%d, %d), want (3, 18)", scores["player1"].FireIceMetricValue, scores["player1"].FireIceVP)
	}
	if scores["player2"].FireIceMetricValue != 2 || scores["player2"].FireIceVP != 12 {
		t.Fatalf("player2 Fire & Ice score = (%d, %d), want (2, 12)", scores["player2"].FireIceMetricValue, scores["player2"].FireIceVP)
	}
	if scores["player3"].FireIceMetricValue != 1 || scores["player3"].FireIceVP != 6 {
		t.Fatalf("player3 Fire & Ice score = (%d, %d), want (1, 6)", scores["player3"].FireIceMetricValue, scores["player3"].FireIceVP)
	}
}

func TestCalculateFinalScoring_FireIceSettlements(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("player1", factions.NewDwarves()); err != nil {
		t.Fatalf("add player1: %v", err)
	}
	if err := gs.AddPlayer("player2", factions.NewFakirs()); err != nil {
		t.Fatalf("add player2: %v", err)
	}
	if err := gs.AddPlayer("player3", factions.NewMermaids()); err != nil {
		t.Fatalf("add player3: %v", err)
	}

	for _, hex := range []board.Hex{board.NewHex(0, 0), board.NewHex(2, 0), board.NewHex(4, 0)} {
		placeFinalScoringTestBuilding(t, gs, "player1", hex, models.BuildingDwelling)
	}
	for _, hex := range []board.Hex{board.NewHex(0, 4), board.NewHex(2, 4)} {
		placeFinalScoringTestBuilding(t, gs, "player2", hex, models.BuildingDwelling)
	}
	for _, hex := range []board.Hex{board.NewHex(0, 7), board.NewHex(1, 7)} {
		placeFinalScoringTestBuilding(t, gs, "player3", hex, models.BuildingDwelling)
	}

	gs.FireIceFinalScoringTile = FireIceFinalScoringTileSettlements

	scores := gs.CalculateFinalScoring()

	if scores["player1"].FireIceMetricValue != 3 || scores["player1"].FireIceVP != 18 {
		t.Fatalf("player1 Fire & Ice score = (%d, %d), want (3, 18)", scores["player1"].FireIceMetricValue, scores["player1"].FireIceVP)
	}
	if scores["player2"].FireIceMetricValue != 2 || scores["player2"].FireIceVP != 12 {
		t.Fatalf("player2 Fire & Ice score = (%d, %d), want (2, 12)", scores["player2"].FireIceMetricValue, scores["player2"].FireIceVP)
	}
	if scores["player3"].FireIceMetricValue != 1 || scores["player3"].FireIceVP != 6 {
		t.Fatalf("player3 Fire & Ice score = (%d, %d), want (1, 6)", scores["player3"].FireIceMetricValue, scores["player3"].FireIceVP)
	}
}
