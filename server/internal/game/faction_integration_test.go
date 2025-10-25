package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// This file contains integration tests for faction-specific abilities
// that interact with the game state and actions.
//
// Factions tested:
// - Halflings: Spade VP bonuses
// - Swarmlings: Upgrade scoring
// - Alchemists: VP/Coin conversion, power per spade
// - Cultists: Power leech bonuses

// ============================================================================
// HALFLINGS TESTS
// ============================================================================

func TestHalflings_RegularTransformScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Give player resources
	player.Resources.Workers = 20
	
	// Find a hex that needs transformation (not already home terrain)
	targetHex := NewHex(0, 0)
	// Make sure it's not already home terrain
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Distance 3 from Plains
	
	initialVP := player.VictoryPoints
	
	// Transform (Halflings use 3 spades for distance 3)
	action := NewTransformAndBuildAction("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to transform: %v", err)
	}
	
	// Should get: 2 VP per spade (scoring tile) + 1 VP per spade (Halflings) = 3 VP per spade
	// Distance 3 = 3 spades, so 3 * 3 = 9 VP total
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 3 * 3 // 3 spades * 3 VP per spade
	if vpGained != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, vpGained)
	}
}

func TestHalflings_BonusCardSpadeScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player the spade bonus card
	gs.BonusCards.PlayerCards["player1"] = BonusCardSpade
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	player.Resources.Workers = 20
	
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Distance 3 from Plains
	
	initialVP := player.VictoryPoints
	
	// Use bonus card spade (1 free spade)
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
		t.Fatalf("failed to use bonus card spade: %v", err)
	}
	
	// Distance 3 = 3 spades, should get 3 * 3 = 9 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 3 * 3
	if vpGained != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, vpGained)
	}
}

func TestHalflings_CultSpadeScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Place a dwelling first to make hex adjacent
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
	
	targetHex := NewHex(1, 0) // Adjacent hex
	initialVP := player.VictoryPoints
	
	// Use cult spade
	action := NewUseCultSpadeAction("player1", targetHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use cult spade: %v", err)
	}
	
	// 1 spade: 2 VP (scoring tile) + 1 VP (Halflings) = 3 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 3
	if vpGained != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, vpGained)
	}
}

func TestHalflings_StrongholdSpadesScoring(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold on the faction
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Check that Halflings can use stronghold spades
	if !faction.CanUseStrongholdSpades() {
		t.Fatal("Halflings should be able to use stronghold spades")
	}
	
	spades := faction.UseStrongholdSpades()
	if spades != 3 {
		t.Errorf("expected 3 spades from stronghold, got %d", spades)
	}
	
	// Note: The actual stronghold action implementation is TODO
	// This test just verifies the faction method works
}

// ============================================================================
// SWARMLINGS TESTS
// ============================================================================

func TestSwarmlings_UpgradeWithScoringTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Set up scoring tile: 3 VP per trading house
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringTradingHouseWater,
				ActionType: ScoringActionTradingHouse,
				ActionVP:   3,
			},
		},
	}
	
	// Place a dwelling first
	upgradeHex := NewHex(0, 0)
	gs.Map.GetHex(upgradeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(upgradeHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	initialVP := player.VictoryPoints
	
	// Use Swarmlings upgrade (free D→TH)
	action := NewSwarmlingsUpgradeAction("player1", upgradeHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade: %v", err)
	}
	
	// Should get 3 VP from scoring tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 3 {
		t.Errorf("expected 3 VP from scoring tile, got %d", vpGained)
	}
	
	// Verify building was upgraded
	mapHex := gs.Map.GetHex(upgradeHex)
	if mapHex.Building.Type != models.BuildingTradingHouse {
		t.Error("building should be upgraded to trading house")
	}
}

func TestSwarmlings_UpgradeWithWater1FavorTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Give player Water+1 favor tile
	gs.FavorTiles = NewFavorTileState()
	gs.FavorTiles.PlayerTiles["player1"] = []FavorTileType{FavorWater1}
	
	// Place a dwelling
	upgradeHex := NewHex(0, 0)
	gs.Map.GetHex(upgradeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(upgradeHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	initialVP := player.VictoryPoints
	
	// Use Swarmlings upgrade
	action := NewSwarmlingsUpgradeAction("player1", upgradeHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade: %v", err)
	}
	
	// Should get 3 VP from Water+1 favor tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 3 {
		t.Errorf("expected 3 VP from Water+1 favor tile, got %d", vpGained)
	}
}

func TestSwarmlings_UpgradeWithBothScoringAndFavor(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Set up scoring tile
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringTradingHouseWater,
				ActionType: ScoringActionTradingHouse,
				ActionVP:   3,
			},
		},
	}
	
	// Give player Water+1 favor tile
	gs.FavorTiles = NewFavorTileState()
	gs.FavorTiles.PlayerTiles["player1"] = []FavorTileType{FavorWater1}
	
	// Place a dwelling
	upgradeHex := NewHex(0, 0)
	gs.Map.GetHex(upgradeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(upgradeHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	initialVP := player.VictoryPoints
	
	// Use Swarmlings upgrade
	action := NewSwarmlingsUpgradeAction("player1", upgradeHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade: %v", err)
	}
	
	// Should get 3 VP (scoring tile) + 3 VP (Water+1) = 6 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 6
	if vpGained != expectedVP {
		t.Errorf("expected %d VP (scoring + favor), got %d", expectedVP, vpGained)
	}
}

// ============================================================================
// ALCHEMISTS TESTS
// ============================================================================

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

// ============================================================================
// CULTISTS TESTS
// ============================================================================

func TestCultists_PowerWhenAllDecline(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	swarmlingsFaction := factions.NewSwarmlings() // Using Swarmlings (Lake) not Halflings (Plains)
	
	gs.AddPlayer("cultists", cultistsFaction)
	gs.AddPlayer("swarmlings", swarmlingsFaction)
	
	cultistsPlayer := gs.GetPlayer("cultists")
	swarmlingsPlayer := gs.GetPlayer("swarmlings")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	swarmlingsPlayer.Resources.Power.Bowl1 = 10
	
	// Place a Swarmlings dwelling adjacent to where Cultists will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = swarmlingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    swarmlingsFaction.GetType(),
		PlayerID:   "swarmlings",
		PowerValue: 1,
	})
	
	// Cultists build a dwelling
	cultistsHex := NewHex(0, 0)
	gs.Map.GetHex(cultistsHex).Terrain = cultistsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(cultistsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    cultistsFaction.GetType(),
		PlayerID:   "cultists",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := cultistsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(cultistsHex, "cultists")
	
	// Verify leech offer was created for Swarmlings
	offers := gs.GetPendingLeechOffers("swarmlings")
	if len(offers) != 1 {
		t.Fatalf("expected 1 leech offer, got %d", len(offers))
	}
	
	// Swarmlings decline the offer
	declineAction := NewDeclinePowerLeechAction("swarmlings", 0)
	err := declineAction.Execute(gs)
	if err != nil {
		t.Fatalf("decline failed: %v", err)
	}
	
	// Cultists should gain 1 power (all opponents declined)
	powerGained := cultistsPlayer.Resources.Power.Bowl1 - initialPower
	expectedPower := 1
	if powerGained != expectedPower {
		t.Errorf("expected %d power gained, got %d", expectedPower, powerGained)
	}
	
	// Verify the pending bonus was resolved
	if gs.PendingCultistsLeech["cultists"] != nil {
		t.Error("Cultists leech bonus should be resolved")
	}
}

func TestCultists_CultAdvanceWhenOneAccepts(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	swarmlingsFaction := factions.NewSwarmlings()
	
	gs.AddPlayer("cultists", cultistsFaction)
	gs.AddPlayer("swarmlings", swarmlingsFaction)
	
	cultistsPlayer := gs.GetPlayer("cultists")
	swarmlingsPlayer := gs.GetPlayer("swarmlings")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	swarmlingsPlayer.Resources.Power.Bowl1 = 10
	swarmlingsPlayer.VictoryPoints = 10
	
	// Place a Swarmlings dwelling adjacent to where Cultists will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = swarmlingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    swarmlingsFaction.GetType(),
		PlayerID:   "swarmlings",
		PowerValue: 1,
	})
	
	// Cultists build a dwelling
	cultistsHex := NewHex(0, 0)
	gs.Map.GetHex(cultistsHex).Terrain = cultistsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(cultistsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    cultistsFaction.GetType(),
		PlayerID:   "cultists",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := cultistsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(cultistsHex, "cultists")
	
	// Verify leech offer was created
	offers := gs.GetPendingLeechOffers("swarmlings")
	if len(offers) != 1 {
		t.Fatalf("expected 1 leech offer, got %d", len(offers))
	}
	
	// Swarmlings accept the offer
	acceptAction := NewAcceptPowerLeechAction("swarmlings", 0)
	err := acceptAction.Execute(gs)
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	
	// Cultists should NOT gain power (opponent accepted)
	powerGained := cultistsPlayer.Resources.Power.Bowl1 - initialPower
	if powerGained != 0 {
		t.Errorf("expected 0 power gained (opponent accepted), got %d", powerGained)
	}
	
	// Verify the pending bonus was resolved
	if gs.PendingCultistsLeech["cultists"] != nil {
		t.Error("Cultists leech bonus should be resolved")
	}
	
	// TODO: When cult track selection is implemented, verify cult advance happened
}

func TestCultists_MultipleOpponents_MixedResponses(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	swarmlingsFaction := factions.NewSwarmlings() // Using Swarmlings (Lake) instead of Halflings (Plains)
	nomadsFaction := factions.NewNomads()
	
	gs.AddPlayer("cultists", cultistsFaction)
	gs.AddPlayer("swarmlings", swarmlingsFaction)
	gs.AddPlayer("nomads", nomadsFaction)
	
	cultistsPlayer := gs.GetPlayer("cultists")
	swarmlingsPlayer := gs.GetPlayer("swarmlings")
	nomadsPlayer := gs.GetPlayer("nomads")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	swarmlingsPlayer.Resources.Power.Bowl1 = 10
	swarmlingsPlayer.VictoryPoints = 10
	nomadsPlayer.Resources.Power.Bowl1 = 10
	nomadsPlayer.VictoryPoints = 10
	
	// Place Swarmlings dwelling adjacent to Cultists
	hex1 := NewHex(1, 0)
	gs.Map.GetHex(hex1).Terrain = swarmlingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    swarmlingsFaction.GetType(),
		PlayerID:   "swarmlings",
		PowerValue: 1,
	})
	
	// Place Nomads dwelling adjacent to Cultists
	hex2 := NewHex(0, 1)
	gs.Map.GetHex(hex2).Terrain = nomadsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex2, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    nomadsFaction.GetType(),
		PlayerID:   "nomads",
		PowerValue: 1,
	})
	
	// Cultists build a dwelling
	cultistsHex := NewHex(0, 0)
	gs.Map.GetHex(cultistsHex).Terrain = cultistsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(cultistsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    cultistsFaction.GetType(),
		PlayerID:   "cultists",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := cultistsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(cultistsHex, "cultists")
	
	// Verify leech offers were created
	swarmlingsOffers := gs.GetPendingLeechOffers("swarmlings")
	nomadsOffers := gs.GetPendingLeechOffers("nomads")
	if len(swarmlingsOffers) != 1 || len(nomadsOffers) != 1 {
		t.Fatalf("expected 1 offer each, got %d and %d", len(swarmlingsOffers), len(nomadsOffers))
	}
	
	// Swarmlings accept
	acceptAction := NewAcceptPowerLeechAction("swarmlings", 0)
	err := acceptAction.Execute(gs)
	if err != nil {
		t.Fatalf("swarmlings accept failed: %v", err)
	}
	
	// Nomads decline
	declineAction := NewDeclinePowerLeechAction("nomads", 0)
	err = declineAction.Execute(gs)
	if err != nil {
		t.Fatalf("nomads decline failed: %v", err)
	}
	
	// Cultists should NOT gain power (at least one opponent accepted)
	powerGained := cultistsPlayer.Resources.Power.Bowl1 - initialPower
	if powerGained != 0 {
		t.Errorf("expected 0 power gained (one accepted), got %d", powerGained)
	}
	
	// Verify the pending bonus was resolved
	if gs.PendingCultistsLeech["cultists"] != nil {
		t.Error("Cultists leech bonus should be resolved")
	}
}

func TestCultists_OnlyCultistsGetBonus(t *testing.T) {
	gs := NewGameState()
	halflingsFaction := factions.NewHalflings()
	nomadsFaction := factions.NewNomads()
	
	gs.AddPlayer("halflings", halflingsFaction)
	gs.AddPlayer("nomads", nomadsFaction)
	
	halflingsPlayer := gs.GetPlayer("halflings")
	nomadsPlayer := gs.GetPlayer("nomads")
	
	// Set up power
	halflingsPlayer.Resources.Power.Bowl1 = 10
	nomadsPlayer.Resources.Power.Bowl1 = 10
	
	// Place a Nomads dwelling adjacent to where Halflings will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = nomadsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    nomadsFaction.GetType(),
		PlayerID:   "nomads",
		PowerValue: 1,
	})
	
	// Halflings build a dwelling
	halflingsHex := NewHex(0, 0)
	gs.Map.GetHex(halflingsHex).Terrain = halflingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(halflingsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    halflingsFaction.GetType(),
		PlayerID:   "halflings",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := halflingsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(halflingsHex, "halflings")
	
	// Nomads decline
	declineAction := NewDeclinePowerLeechAction("nomads", 0)
	err := declineAction.Execute(gs)
	if err != nil {
		t.Fatalf("decline failed: %v", err)
	}
	
	// Halflings should NOT gain power (not Cultists)
	powerGained := halflingsPlayer.Resources.Power.Bowl1 - initialPower
	if powerGained != 0 {
		t.Errorf("expected 0 power gained (not Cultists), got %d", powerGained)
	}
	
	// Verify no pending bonus was created
	if gs.PendingCultistsLeech["halflings"] != nil {
		t.Error("non-Cultists should not have pending leech bonus")
	}
}

// ============================================================================
// GIANTS TESTS
// ============================================================================

func TestGiants_TransformAwardsScoringTileVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold to enable special action
	player.HasStrongholdAbility = true
	
	// Set up scoring tile: 2 VP per spade
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				Type:       ScoringSpades,
				ActionType: ScoringActionSpades,
				ActionVP:   2,
			},
		},
	}
	
	// Find a hex to transform
	targetHex := NewHex(0, 0)
	
	initialVP := player.VictoryPoints
	
	// Execute Giants Transform (2 free spades)
	action := NewGiantsTransformAction("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute Giants transform: %v", err)
	}
	
	// Should get 2 VP per spade * 2 spades = 4 VP
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 2 * 2 // 2 spades * 2 VP per spade
	if vpGained != expectedVP {
		t.Errorf("expected %d VP from Giants transform, got %d", expectedVP, vpGained)
	}
	
	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != faction.GetHomeTerrain() {
		t.Errorf("terrain not transformed to home terrain")
	}
}

func TestGiants_TransformWithDwelling(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	player.HasStrongholdAbility = true
	
	// Give player resources for dwelling
	player.Resources.Coins = 10
	player.Resources.Workers = 5
	
	targetHex := NewHex(0, 0)
	
	// Execute Giants Transform with dwelling
	action := NewGiantsTransformAction("player1", targetHex, true)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to execute Giants transform with dwelling: %v", err)
	}
	
	// Verify dwelling was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Error("dwelling not placed")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
	
	// Verify resources were spent
	dwellingCost := faction.GetDwellingCost()
	if player.Resources.Workers != 5-dwellingCost.Workers {
		t.Errorf("workers not spent correctly")
	}
}

func TestGiants_TransformOncePerRound(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewGiants()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	player.HasStrongholdAbility = true
	
	targetHex1 := NewHex(0, 0)
	targetHex2 := NewHex(1, 0)
	
	// First use should succeed
	action1 := NewGiantsTransformAction("player1", targetHex1, false)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("first Giants transform should succeed: %v", err)
	}
	
	// Second use in same round should fail
	action2 := NewGiantsTransformAction("player1", targetHex2, false)
	err = action2.Execute(gs)
	if err == nil {
		t.Error("second Giants transform in same round should fail")
	}
}

// ============================================================================
// ENGINEERS TESTS
// ============================================================================

func TestEngineers_VPPerBridgeOnPass(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold to enable 3 VP per bridge ability
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Simulate building 2 bridges
	player.BridgesBuilt = 2
	
	initialVP := player.VictoryPoints
	
	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardPriest})
	
	// Pass action should award 3 VP per bridge = 6 VP
	bonusCard := BonusCardPriest
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("pass action failed: %v", err)
	}
	
	vpGained := player.VictoryPoints - initialVP
	expectedVP := 6 // 2 bridges * 3 VP
	if vpGained != expectedVP {
		t.Errorf("expected %d VP from bridges on pass, got %d", expectedVP, vpGained)
	}
}

func TestEngineers_VPPerBridgeBeforeStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// No stronghold built
	
	// Simulate building 2 bridges
	player.BridgesBuilt = 2
	
	initialVP := player.VictoryPoints
	
	// Set up bonus cards
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardPriest})
	
	// Pass action should NOT award VP for bridges (no stronghold)
	bonusCard := BonusCardPriest
	action := NewPassAction("player1", &bonusCard)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("pass action failed: %v", err)
	}
	
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 0 {
		t.Errorf("expected 0 VP from bridges (no stronghold), got %d", vpGained)
	}
}

func TestEngineers_BridgePowerAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Engineers can build bridges using power action (like all factions)
	// They also have a special action to build bridges for 2 workers
	
	// Reset starting power and give player resources for power action
	player.Resources.Power.Bowl1 = 0
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 3
	
	// Build a bridge using power action
	// NOTE: Bridge placement requires specifying hex coordinates
	// For now, just verify the counter increments
	action := NewPowerAction("player1", PowerActionBridge)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("bridge power action failed: %v", err)
	}
	
	// Verify bridge counter was incremented
	if player.BridgesBuilt != 1 {
		t.Errorf("expected 1 bridge built, got %d", player.BridgesBuilt)
	}
	
	// Verify power was spent (3 power moved from Bowl3 to Bowl1)
	if player.Resources.Power.Bowl3 != 0 {
		t.Errorf("expected 0 power in Bowl3 after spending, got %d", player.Resources.Power.Bowl3)
	}
	if player.Resources.Power.Bowl1 != 3 {
		t.Errorf("expected 3 power in Bowl1 after spending, got %d", player.Resources.Power.Bowl1)
	}
}

func TestEngineers_BridgeAndTownFormation(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up a valid bridge scenario using base orientation from Terra Mystica rules
	// Bridge from (0,0) to (1,-2) with midpoints (0,-1) and (1,-1) both being river
	// This creates two groups of buildings that will form a town when connected
	
	// Group 1: Buildings at and near (0,0)
	hex1 := NewHex(0, 0)
	hex2 := NewHex(-1, 0)
	
	// River hexes that separate the groups
	river1 := NewHex(0, -1)
	river2 := NewHex(1, -1)
	
	// Group 2: Buildings at and near (1,-2)
	hex3 := NewHex(1, -2)
	hex4 := NewHex(2, -2)
	
	// Ensure hexes exist in map and set up terrain
	if gs.Map.GetHex(hex1) == nil {
		gs.Map.Hexes[hex1] = &MapHex{Coord: hex1}
	}
	if gs.Map.GetHex(hex2) == nil {
		gs.Map.Hexes[hex2] = &MapHex{Coord: hex2}
	}
	if gs.Map.GetHex(river1) == nil {
		gs.Map.Hexes[river1] = &MapHex{Coord: river1}
	}
	if gs.Map.GetHex(river2) == nil {
		gs.Map.Hexes[river2] = &MapHex{Coord: river2}
	}
	if gs.Map.GetHex(hex3) == nil {
		gs.Map.Hexes[hex3] = &MapHex{Coord: hex3}
	}
	if gs.Map.GetHex(hex4) == nil {
		gs.Map.Hexes[hex4] = &MapHex{Coord: hex4}
	}
	
	// Set up terrain
	gs.Map.GetHex(hex1).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(hex2).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(river1).Terrain = models.TerrainRiver
	gs.Map.GetHex(river2).Terrain = models.TerrainRiver
	gs.Map.GetHex(hex3).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(hex4).Terrain = faction.GetHomeTerrain()
	
	// Mark river hexes
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	
	// Place buildings with total power = 7 to form a town
	// Dwelling (1) + Dwelling (1) + Trading House (2) + Trading House (2) = 6 (not enough)
	// Need at least 7 power
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	gs.Map.PlaceBuilding(hex2, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.PlaceBuilding(hex3, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.PlaceBuilding(hex4, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	// Total power = 1 + 2 + 2 + 2 = 7 (exactly 7, meets requirement)
	
	// Before bridge: buildings are not connected (separated by river)
	connectedBefore := gs.CheckForTownFormation("player1", hex1)
	if connectedBefore != nil {
		t.Error("expected no town formation before bridge (groups separated by river)")
	}
	
	// Give player resources for power action
	player.Resources.Power.Bowl3 = 3
	
	// Build a bridge using power action
	action := NewPowerActionWithBridge("player1", hex1, hex3)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to build bridge: %v", err)
	}
	
	// Verify bridge was built
	if player.BridgesBuilt != 1 {
		t.Errorf("expected 1 bridge built, got %d", player.BridgesBuilt)
	}
	
	// Verify bridge exists on map
	if !gs.Map.HasBridge(hex1, hex3) {
		t.Error("expected bridge to exist on map")
	}
	
	// Verify that town formation was detected and pending
	if gs.PendingTownFormations["player1"] == nil {
		t.Error("expected pending town formation after bridge connects buildings")
	} else {
		pendingTown := gs.PendingTownFormations["player1"]
		if len(pendingTown.Hexes) != 4 {
			t.Errorf("expected 4 connected buildings in town, got %d", len(pendingTown.Hexes))
		}
	}
}
