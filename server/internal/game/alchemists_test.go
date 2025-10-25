package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Test Alchemists VP to Coins conversion
func TestAlchemists_ConvertVPToCoins(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player VP
	player.VictoryPoints = 10
	player.Resources.Coins = 5
	
	// Convert 3 VP to 3 coins (1:1 ratio)
	err := gs.AlchemistsConvertVPToCoins("player1", 3)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	
	// Check results
	if player.VictoryPoints != 7 {
		t.Errorf("expected 7 VP, got %d", player.VictoryPoints)
	}
	if player.Resources.Coins != 8 {
		t.Errorf("expected 8 coins, got %d", player.Resources.Coins)
	}
}

func TestAlchemists_ConvertVPToCoins_NotEnoughVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	player.VictoryPoints = 2
	
	// Try to convert more VP than available
	err := gs.AlchemistsConvertVPToCoins("player1", 5)
	if err == nil {
		t.Error("should fail when not enough VP")
	}
}

func TestAlchemists_ConvertVPToCoins_OnlyAlchemists(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	player.VictoryPoints = 10
	
	// Try to convert as non-Alchemists
	err := gs.AlchemistsConvertVPToCoins("player1", 3)
	if err == nil {
		t.Error("should fail for non-Alchemists faction")
	}
}

// Test Alchemists Coins to VP conversion
func TestAlchemists_ConvertCoinsToVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player coins
	player.Resources.Coins = 10
	player.VictoryPoints = 5
	
	// Convert 6 coins to 3 VP (2:1 ratio)
	err := gs.AlchemistsConvertCoinsToVP("player1", 6)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	
	// Check results
	if player.Resources.Coins != 4 {
		t.Errorf("expected 4 coins, got %d", player.Resources.Coins)
	}
	if player.VictoryPoints != 8 {
		t.Errorf("expected 8 VP, got %d", player.VictoryPoints)
	}
}

func TestAlchemists_ConvertCoinsToVP_NotEnoughCoins(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	player.Resources.Coins = 3
	
	// Try to convert more coins than available
	err := gs.AlchemistsConvertCoinsToVP("player1", 6)
	if err == nil {
		t.Error("should fail when not enough coins")
	}
}

func TestAlchemists_ConvertCoinsToVP_MustBeEven(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	player.Resources.Coins = 10
	
	// Try to convert odd number of coins
	err := gs.AlchemistsConvertCoinsToVP("player1", 5)
	if err == nil {
		t.Error("should fail when converting odd number of coins")
	}
}

// Test Alchemists power per spade after stronghold
func TestAlchemists_PowerPerSpadeAfterStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Give player resources
	player.Resources.Workers = 20
	player.Resources.Power.Bowl1 = 10
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0
	
	// Transform terrain (distance 3 = 2 spades for Alchemists, who use 2 workers per spade)
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Distance 3 from Swamp
	
	initialPower := player.Resources.Power.Bowl1
	
	action := NewTransformAndBuildAction("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	
	// Should gain 2 spades * 2 power = 4 power
	powerGained := player.Resources.Power.Bowl1 - initialPower
	expectedPower := 4
	if powerGained != expectedPower {
		t.Errorf("expected %d power gained, got %d", expectedPower, powerGained)
	}
}

func TestAlchemists_PowerPerSpadeBeforeStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// No stronghold built
	
	// Give player resources
	player.Resources.Workers = 20
	player.Resources.Power.Bowl1 = 10
	
	// Transform terrain
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest
	
	initialPower := player.Resources.Power.Bowl1
	
	action := NewTransformAndBuildAction("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	
	// Should gain 0 power (no stronghold)
	powerGained := player.Resources.Power.Bowl1 - initialPower
	if powerGained != 0 {
		t.Errorf("expected 0 power gained before stronghold, got %d", powerGained)
	}
}

func TestAlchemists_PowerPerSpadeWithCultSpade(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Place a dwelling first
	startHex := NewHex(0, 0)
	gs.Map.GetHex(startHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(startHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Give player a pending spade from cult reward
	gs.PendingSpades = make(map[string]int)
	gs.PendingSpades["player1"] = 1
	
	player.Resources.Power.Bowl1 = 10
	
	targetHex := NewHex(1, 0) // Adjacent hex
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest
	
	initialPower := player.Resources.Power.Bowl1
	
	// Use cult spade
	action := NewUseCultSpadeAction("player1", targetHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("cult spade failed: %v", err)
	}
	
	// Should gain 1 spade * 2 power = 2 power
	powerGained := player.Resources.Power.Bowl1 - initialPower
	expectedPower := 2
	if powerGained != expectedPower {
		t.Errorf("expected %d power gained, got %d", expectedPower, powerGained)
	}
}

func TestAlchemists_PowerPerSpadeWithBonusCard(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	player.Resources.Workers = 20
	player.Resources.Power.Bowl1 = 10
	
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Distance 3 from Swamp
	
	initialPower := player.Resources.Power.Bowl1
	
	// Use bonus card spade
	action := &SpecialAction{
		BaseAction: BaseAction{
			Type:     ActionSpecialAction,
			PlayerID: "player1",
		},
		ActionType:    SpecialActionBonusCardSpade,
		TargetHex:     &targetHex,
		BuildDwelling: false,
	}
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("bonus card spade failed: %v", err)
	}
	
	// Should gain 2 spades * 2 power = 4 power (Alchemists use 2 workers per spade)
	powerGained := player.Resources.Power.Bowl1 - initialPower
	expectedPower := 4
	if powerGained != expectedPower {
		t.Errorf("expected %d power gained, got %d", expectedPower, powerGained)
	}
}

// Test conversion during an action (e.g., convert VP to coins, then build)
func TestAlchemists_ConversionDuringAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Player has 0 coins, 1 worker, 20 VP
	player.Resources.Coins = 0
	player.Resources.Workers = 1
	player.VictoryPoints = 20
	
	// Convert 2 VP to 2 coins
	err := gs.AlchemistsConvertVPToCoins("player1", 2)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	
	// Now build a dwelling (costs 0 coins + 1 worker for Alchemists)
	// First need to transform terrain
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = faction.GetHomeTerrain()
	
	action := NewTransformAndBuildAction("player1", targetHex, true)
	err = action.Execute(gs)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	
	// Verify dwelling was built
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil || mapHex.Building.Type != models.BuildingDwelling {
		t.Error("dwelling should be built")
	}
	
	// Verify resources were spent
	if player.VictoryPoints != 18 {
		t.Errorf("expected 18 VP (20-2), got %d", player.VictoryPoints)
	}
	if player.Resources.Coins != 2 {
		t.Errorf("expected 2 coins remaining, got %d", player.Resources.Coins)
	}
	if player.Resources.Workers != 0 {
		t.Errorf("expected 0 workers, got %d", player.Resources.Workers)
	}
}
