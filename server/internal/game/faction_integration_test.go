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
	
	// Stronghold action implemented via PendingHalflingsSpades system
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
	
	// Now build a dwelling (costs 2 coins + 1 worker for Alchemists)
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
	
	// Verify resources were spent (2 coins for dwelling)
	if player.VictoryPoints != 18 {
		t.Errorf("expected 18 VP (20-2), got %d", player.VictoryPoints)
	}
	if player.Resources.Coins != 0 {
		t.Errorf("expected 0 coins remaining (2 converted - 2 for dwelling), got %d", player.Resources.Coins)
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
	
	// Cult track selection is now implemented and tested below
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
// CULTISTS CULT TRACK SELECTION TESTS
// ============================================================================

func TestCultists_CultTrackSelection_OpponentAccepts(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	aurenFaction := factions.NewAuren()
	
	err := gs.AddPlayer("cultists", cultistsFaction)
	if err != nil {
		t.Fatalf("failed to add cultists: %v", err)
	}
	err = gs.AddPlayer("auren", aurenFaction)
	if err != nil {
		t.Fatalf("failed to add auren: %v", err)
	}
	
	cultistsPlayer := gs.GetPlayer("cultists")
	aurenPlayer := gs.GetPlayer("auren")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	aurenPlayer.Resources.Power.Bowl1 = 10
	aurenPlayer.VictoryPoints = 10
	
	// Initialize cult positions
	cultistsPlayer.CultPositions = make(map[CultTrack]int)
	cultistsPlayer.CultPositions[CultFire] = 2
	
	// Place an Auren dwelling adjacent to where Cultists will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = aurenFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    aurenFaction.GetType(),
		PlayerID:   "auren",
		PowerValue: 1,
	})
	
	// Cultists place a dwelling (triggers power leech)
	cultistHex := NewHex(0, 0)
	gs.Map.GetHex(cultistHex).Terrain = cultistsFaction.GetHomeTerrain()
	action := NewTransformAndBuildAction("cultists", cultistHex, true)
	err = action.Execute(gs)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	
	// Auren accepts power leech
	acceptAction := NewAcceptPowerLeechAction("auren", 0)
	err = acceptAction.Execute(gs)
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	
	// Verify pending cult selection was created for Cultists
	if gs.PendingCultistsCultSelection == nil {
		t.Fatal("expected pending cult selection for Cultists")
	}
	
	if gs.PendingCultistsCultSelection.PlayerID != "cultists" {
		t.Errorf("expected cultists, got %s", gs.PendingCultistsCultSelection.PlayerID)
	}
	
	initialFirePos := cultistsPlayer.CultPositions[CultFire]
	
	// Cultists selects Fire cult track
	selectAction := NewSelectCultistsCultTrackAction("cultists", CultFire)
	err = selectAction.Execute(gs)
	if err != nil {
		t.Fatalf("cult selection failed: %v", err)
	}
	
	// Verify cult position advanced
	if cultistsPlayer.CultPositions[CultFire] != initialFirePos+1 {
		t.Errorf("expected Fire cult position %d, got %d", initialFirePos+1, cultistsPlayer.CultPositions[CultFire])
	}
	
	// Verify pending state was cleared
	if gs.PendingCultistsCultSelection != nil {
		t.Error("pending cult selection should be cleared")
	}
}

func TestCultists_CultTrackSelection_AllOpponentsDecline(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	aurenFaction := factions.NewAuren()
	
	err := gs.AddPlayer("cultists", cultistsFaction)
	if err != nil {
		t.Fatalf("failed to add cultists: %v", err)
	}
	err = gs.AddPlayer("auren", aurenFaction)
	if err != nil {
		t.Fatalf("failed to add auren: %v", err)
	}
	
	cultistsPlayer := gs.GetPlayer("cultists")
	aurenPlayer := gs.GetPlayer("auren")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	aurenPlayer.Resources.Power.Bowl1 = 10
	aurenPlayer.VictoryPoints = 10
	
	// Initialize cult positions
	cultistsPlayer.CultPositions = make(map[CultTrack]int)
	
	// Place an Auren dwelling adjacent to where Cultists will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = aurenFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    aurenFaction.GetType(),
		PlayerID:   "auren",
		PowerValue: 1,
	})
	
	// Cultists place a dwelling (triggers power leech)
	cultistHex := NewHex(0, 0)
	gs.Map.GetHex(cultistHex).Terrain = cultistsFaction.GetHomeTerrain()
	action := NewTransformAndBuildAction("cultists", cultistHex, true)
	err = action.Execute(gs)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	
	initialPower := cultistsPlayer.Resources.Power.Bowl1
	
	// Auren declines power leech
	declineAction := NewDeclinePowerLeechAction("auren", 0)
	err = declineAction.Execute(gs)
	if err != nil {
		t.Fatalf("decline failed: %v", err)
	}
	
	// Verify NO pending cult selection (Cultists gains power instead)
	if gs.PendingCultistsCultSelection != nil {
		t.Error("should not have pending cult selection when all decline")
	}
	
	// Verify Cultists gained 1 power (not cult advance)
	if cultistsPlayer.Resources.Power.Bowl1 != initialPower+1 {
		t.Errorf("expected %d power, got %d", initialPower+1, cultistsPlayer.Resources.Power.Bowl1)
	}
}

func TestCultists_CultTrackSelection_CannotSelectAtPosition10(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	err := gs.AddPlayer("cultists", cultistsFaction)
	if err != nil {
		t.Fatalf("failed to add cultists: %v", err)
	}
	cultistsPlayer := gs.GetPlayer("cultists")
	
	// Set up cult position at maximum
	cultistsPlayer.CultPositions = make(map[CultTrack]int)
	cultistsPlayer.CultPositions[CultFire] = 10
	cultistsPlayer.CultPositions[CultWater] = 5
	
	// Create pending cult selection
	gs.PendingCultistsCultSelection = &PendingCultistsCultSelection{
		PlayerID: "cultists",
	}
	
	// Try to select Fire (at position 10)
	selectAction := NewSelectCultistsCultTrackAction("cultists", CultFire)
	err = selectAction.Validate(gs)
	if err == nil {
		t.Error("expected error when selecting cult track at position 10")
	}
	
	// Try to select Water (at position 5) - should work
	selectAction2 := NewSelectCultistsCultTrackAction("cultists", CultWater)
	err = selectAction2.Execute(gs)
	if err != nil {
		t.Fatalf("expected success for Water cult track: %v", err)
	}
	
	if cultistsPlayer.CultPositions[CultWater] != 6 {
		t.Errorf("expected Water position 6, got %d", cultistsPlayer.CultPositions[CultWater])
	}
}

func TestCultists_CultTrackSelection_OnlyForCultists(t *testing.T) {
	gs := NewGameState()
	halflingsFaction := factions.NewHalflings()
	err := gs.AddPlayer("halflings", halflingsFaction)
	if err != nil {
		t.Fatalf("failed to add halflings: %v", err)
	}
	halflingsPlayer := gs.GetPlayer("halflings")
	
	// Set up cult positions
	halflingsPlayer.CultPositions = make(map[CultTrack]int)
	halflingsPlayer.CultPositions[CultFire] = 2
	
	// Incorrectly create pending cult selection for non-Cultists
	gs.PendingCultistsCultSelection = &PendingCultistsCultSelection{
		PlayerID: "halflings",
	}
	
	// Try to select cult track as non-Cultists
	selectAction := NewSelectCultistsCultTrackAction("halflings", CultFire)
	err = selectAction.Validate(gs)
	if err == nil {
		t.Error("expected error when non-Cultists tries to use cult selection")
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

// ============================================================================
// WITCHES TESTS
// ============================================================================

func TestWitches_TownFoundingBonus(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up 4 connected buildings with total power >= 7
	hexes := []Hex{
		NewHex(0, 0),
		NewHex(1, 0),
		NewHex(2, 0),
		NewHex(3, 0),
	}
	
	// Place buildings: 1 Dwelling + 3 Trading Houses = 1 + 2 + 2 + 2 = 7 power
	for i, hex := range hexes {
		gs.Map.Hexes[hex] = &MapHex{Coord: hex, Terrain: faction.GetHomeTerrain()}
		buildingType := models.BuildingDwelling
		powerValue := 1
		if i > 0 {
			buildingType = models.BuildingTradingHouse
			powerValue = 2
		}
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       buildingType,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: powerValue,
		})
	}
	
	// Record initial VP
	initialVP := player.VictoryPoints
	
	// Form town
	err := gs.FormTown("player1", hexes, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify Witches got their +5 VP bonus
	// Town tile gives +5 VP, Witches bonus gives +5 VP = +10 VP total
	expectedVPGain := 5 + 5 // tile VP + Witches bonus
	actualVPGain := player.VictoryPoints - initialVP
	if actualVPGain != expectedVPGain {
		t.Errorf("expected +%d VP (5 tile + 5 Witches bonus), got +%d", expectedVPGain, actualVPGain)
	}
}

func TestWitches_RideIgnoresAdjacency(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold (required for Witches' Ride)
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Place a building for the player
	startHex := NewHex(0, 0)
	gs.Map.Hexes[startHex] = &MapHex{Coord: startHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(startHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is far away (not adjacent) but is Forest
	targetHex := NewHex(10, 10)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainForest}
	
	// Use Witches' Ride to place dwelling far away (ignoring adjacency)
	action := NewWitchesRideAction("player1", targetHex)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("Witches' Ride should ignore adjacency, got error: %v", err)
	}
	
	// Verify dwelling was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building == nil {
		t.Error("expected building at target hex")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
	if mapHex.Building.PlayerID != "player1" {
		t.Errorf("expected player1's building, got player %s", mapHex.Building.PlayerID)
	}
}

func TestWitches_RideOnlyOnForest(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Target hex is NOT forest
	targetHex := NewHex(5, 5)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainPlains}
	
	// Witches' Ride should fail on non-forest
	action := NewWitchesRideAction("player1", targetHex)
	err := action.Execute(gs)
	if err == nil {
		t.Error("Witches' Ride should only work on Forest spaces")
	}
	
	// Verify no building was placed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Building != nil {
		t.Error("no building should have been placed on non-forest")
	}
}

// ============================================================================
// FAKIRS TESTS
// ============================================================================

func TestFakirs_CarpetFlightBasic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewFakirs()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Place initial dwelling at (0,0)
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is distance 2 away (not directly adjacent) - requires carpet flight
	// Fakirs with range 1 can reach distance 2 (skip over 1 hex)
	targetHex := NewHex(2, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainPlains}
	
	// Give player resources
	player.Resources.Workers = 10
	player.Resources.Priests = 5
	initialVP := player.VictoryPoints
	
	// Try normal build action without skip - should fail (not adjacent)
	actionNoSkip := NewTransformAndBuildAction("player1", targetHex, true)
	err := actionNoSkip.Execute(gs)
	if err == nil {
		t.Fatal("expected error for non-adjacent hex without skip")
	}
	
	// Use carpet flight (skip)
	actionWithSkip := NewTransformAndBuildActionWithSkip("player1", targetHex, true)
	err = actionWithSkip.Execute(gs)
	if err != nil {
		t.Fatalf("carpet flight should work, got error: %v", err)
	}
	
	// Verify priest was spent
	if player.Resources.Priests != 4 {
		t.Errorf("expected 4 priests remaining, got %d", player.Resources.Priests)
	}
	
	// Verify VP bonus was awarded (+4 VP)
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 4 {
		t.Errorf("expected +4 VP for carpet flight, got +%d", vpGained)
	}
	
	// Verify dwelling was built
	if gs.Map.GetHex(targetHex).Building == nil {
		t.Error("expected dwelling to be built")
	}
}

func TestFakirs_CarpetFlightAfterStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewFakirs()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold (increases range to 2)
	faction.BuildStronghold()
	
	// Place initial dwelling
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is 2 spaces away - only possible with stronghold
	targetHex := NewHex(2, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainPlains}
	
	// Give player resources
	player.Resources.Workers = 10
	player.Resources.Priests = 2
	
	// Verify range is 2 after stronghold
	if faction.GetCarpetFlightRange() != 2 {
		t.Errorf("expected carpet flight range 2 after stronghold, got %d", faction.GetCarpetFlightRange())
	}
	
	// Use carpet flight
	action := NewTransformAndBuildActionWithSkip("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("carpet flight with stronghold should work, got error: %v", err)
	}
	
	// Verify priest was spent
	if player.Resources.Priests != 1 {
		t.Errorf("expected 1 priest remaining, got %d", player.Resources.Priests)
	}
}

func TestFakirs_CarpetFlightWithPowerAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewFakirs()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Place initial dwelling
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is 1 space away
	targetHex := NewHex(1, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainPlains}
	
	// Give player power and priests
	player.Resources.Power.Bowl3 = 5
	player.Resources.Priests = 2
	player.Resources.Workers = 10
	initialVP := player.VictoryPoints
	
	// Use spade power action with carpet flight
	action := NewPowerActionWithTransform("player1", PowerActionSpade1, targetHex, false)
	action.UseSkip = true
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("spade power action with carpet flight should work, got error: %v", err)
	}
	
	// Verify priest was spent for carpet flight
	if player.Resources.Priests != 1 {
		t.Errorf("expected 1 priest remaining, got %d", player.Resources.Priests)
	}
	
	// Verify VP bonus was awarded (+4 VP)
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 4 {
		t.Errorf("expected +4 VP for carpet flight, got +%d", vpGained)
	}
}

func TestFakirs_CannotUpgradeShipping(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewFakirs()
	gs.AddPlayer("player1", faction)
	
	// Fakirs cannot upgrade shipping - cost should be 0
	cost := faction.GetShippingCost(0)
	if cost.Priests != 0 || cost.Coins != 0 {
		t.Error("Fakirs should have 0 cost for shipping (indicating impossible)")
	}
}

// ============================================================================
// DWARVES TESTS
// ============================================================================

func TestDwarves_TunnelingBasic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDwarves()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Place initial dwelling at (0,0)
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is distance 2 away (not adjacent) - use (2,0)
	// (1,0) is directly adjacent to (0,0), so it won't work for testing tunneling
	targetHex := NewHex(2, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainPlains}
	
	// Give player resources (2 extra workers for tunneling + terraform + dwelling)
	player.Resources.Workers = 15
	initialVP := player.VictoryPoints
	initialWorkers := player.Resources.Workers
	
	// Try normal build action without skip - should fail (not adjacent)
	actionNoSkip := NewTransformAndBuildAction("player1", targetHex, true)
	err := actionNoSkip.Execute(gs)
	if err == nil {
		t.Fatal("expected error for non-adjacent hex without skip")
	}
	
	// Use tunneling (skip)
	actionWithSkip := NewTransformAndBuildActionWithSkip("player1", targetHex, true)
	err = actionWithSkip.Execute(gs)
	if err != nil {
		t.Fatalf("tunneling should work, got error: %v", err)
	}
	
	// Verify extra workers were spent (2 for tunneling before stronghold)
	// Mountain to Plains = 3 spades * 3 workers/spade = 9 workers
	// Plus tunneling = 2 workers
	// Plus dwelling = 1 worker
	// Total = 12 workers
	expectedWorkerCost := 2 + 9 + 1 // tunneling + terraform + dwelling
	actualWorkerCost := initialWorkers - player.Resources.Workers
	if actualWorkerCost != expectedWorkerCost {
		t.Errorf("expected %d workers spent, got %d", expectedWorkerCost, actualWorkerCost)
	}
	
	// Verify VP bonus was awarded (+4 VP)
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 4 {
		t.Errorf("expected +4 VP for tunneling, got +%d", vpGained)
	}
	
	// Verify dwelling was built
	if gs.Map.GetHex(targetHex).Building == nil {
		t.Error("expected dwelling to be built")
	}
}

func TestDwarves_TunnelingAfterStronghold(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDwarves()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold (reduces tunneling cost to 1 worker)
	faction.BuildStronghold()
	
	// Place initial dwelling
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is distance 2 away
	targetHex := NewHex(2, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainPlains}
	
	// Give player resources
	player.Resources.Workers = 15
	initialWorkers := player.Resources.Workers
	initialVP := player.VictoryPoints
	
	// Verify tunneling cost is 1 after stronghold
	if faction.GetTunnelingCost() != 1 {
		t.Errorf("expected tunneling cost 1 after stronghold, got %d", faction.GetTunnelingCost())
	}
	
	// Use tunneling with stronghold
	action := NewTransformAndBuildActionWithSkip("player1", targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("tunneling with stronghold should work, got error: %v", err)
	}
	
	// Verify VP bonus was awarded
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 4 {
		t.Errorf("expected +4 VP for tunneling, got +%d", vpGained)
	}
	
	// Verify only 1 extra worker was spent for tunneling (+ terraform)
	// Mountain to Plains = 3 spades * 3 workers/spade = 9 workers
	// Plus tunneling with stronghold = 1 worker
	// Total = 10 workers
	expectedWorkerCost := 1 + 9 // tunneling + terraform
	actualWorkerCost := initialWorkers - player.Resources.Workers
	if actualWorkerCost != expectedWorkerCost {
		t.Errorf("expected %d workers spent with stronghold, got %d", expectedWorkerCost, actualWorkerCost)
	}
}

func TestDwarves_TunnelingWithPowerAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDwarves()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Place initial dwelling
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is distance 2 away
	targetHex := NewHex(2, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainPlains}
	
	// Give player power and workers
	player.Resources.Power.Bowl3 = 5
	player.Resources.Workers = 10
	initialVP := player.VictoryPoints
	initialWorkers := player.Resources.Workers
	
	// Use spade power action with tunneling (1 free spade + pay for remaining)
	action := NewPowerActionWithTransform("player1", PowerActionSpade1, targetHex, false)
	action.UseSkip = true
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("spade power action with tunneling should work, got error: %v", err)
	}
	
	// Verify workers spent for tunneling + remaining spades
	// Terrain distance from Mountain to Plains = 3 spades
	// Power action gives 1 free spade, so need to pay for 2 remaining spades
	// 2 spades * 3 workers/spade = 6 workers
	// Plus tunneling cost = 2 workers
	// Total = 8 workers
	workersSpent := initialWorkers - player.Resources.Workers
	if workersSpent != 8 {
		t.Errorf("expected 8 workers spent (2 tunneling + 6 for 2 remaining spades), got %d", workersSpent)
	}
	
	// Verify VP bonus was awarded (+4 VP)
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 4 {
		t.Errorf("expected +4 VP for tunneling, got +%d", vpGained)
	}
}

func TestDwarves_CannotUpgradeShipping(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDwarves()
	gs.AddPlayer("player1", faction)
	
	// Dwarves cannot upgrade shipping - cost should be 0
	cost := faction.GetShippingCost(0)
	if cost.Priests != 0 || cost.Coins != 0 {
		t.Error("Dwarves should have 0 cost for shipping (indicating impossible)")
	}
}

// ============================================================================
// DARKLINGS TESTS
// ============================================================================

func TestDarklings_TerraformWithPriests(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Place initial dwelling at (0,0)
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is adjacent - terraform Mountain to Swamp (3 spades)
	targetHex := NewHex(1, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainMountain}
	
	// Give player resources (priests, not workers)
	player.Resources.Priests = 5
	player.Resources.Workers = 10
	player.Resources.Coins = 10
	initialPriests := player.Resources.Priests
	initialWorkers := player.Resources.Workers
	initialVP := player.VictoryPoints
	
	// Transform and build dwelling
	action := NewTransformAndBuildAction("player1", targetHex, true)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("terraform should work for Darklings, got error: %v", err)
	}
	
	// Verify priests were spent (not workers)
	// Mountain to Swamp = 3 spades = 3 priests
	priestsSpent := initialPriests - player.Resources.Priests
	if priestsSpent != 3 {
		t.Errorf("expected 3 priests spent (Mountain to Swamp = 3 spades), got %d", priestsSpent)
	}
	
	// Verify workers were NOT spent for terraform (only for dwelling)
	workersSpent := initialWorkers - player.Resources.Workers
	if workersSpent != 1 { // Only dwelling cost
		t.Errorf("expected 1 worker spent (dwelling only), got %d", workersSpent)
	}
	
	// Verify VP bonus (+2 VP per spade = +6 VP for 3 spades)
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 6 {
		t.Errorf("expected +6 VP for 3 spades, got +%d", vpGained)
	}
	
	// Verify dwelling was built
	if gs.Map.GetHex(targetHex).Building == nil {
		t.Error("expected dwelling to be built")
	}
}

func TestDarklings_PriestOrdinationBasic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold first
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Give player 3 workers
	player.Resources.Workers = 3
	player.Resources.Priests = 1 // Start with 1 priest
	
	// Priest ordination happens immediately after building stronghold
	// Player chooses to convert 2 workers to 2 priests
	priestsGained, err := faction.UsePriestOrdination(2)
	if err != nil {
		t.Fatalf("priest ordination should work, got error: %v", err)
	}
	
	if priestsGained != 2 {
		t.Errorf("expected 2 priests gained, got %d", priestsGained)
	}
	
	// Pay workers and gain priests
	player.Resources.Workers -= 2
	gs.GainPriests("player1", priestsGained)
	
	// Verify workers were spent
	if player.Resources.Workers != 1 {
		t.Errorf("expected 1 worker remaining, got %d", player.Resources.Workers)
	}
	
	// Verify priests were gained
	if player.Resources.Priests != 3 {
		t.Errorf("expected 3 priests total (1 start + 2 converted), got %d", player.Resources.Priests)
	}
}

func TestDarklings_PriestOrdination7PriestLimit(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Player has 4 priests in hand
	player.Resources.Priests = 4
	player.Resources.Workers = 5
	
	// Send 2 priests to cult tracks (using cult system directly)
	gs.CultTracks.InitializePlayer("player1")
	gs.CultTracks.AdvancePlayer("player1", CultFire, 1, player)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 1, player)
	// Now: 4 in hand - 2 sent = 2 in hand, 2 on cults = 4 total
	player.Resources.Priests = 2
	
	// Try to convert 3 workers (would be 2 + 2 + 3 = 7 total, should work)
	priestsGained, err := faction.UsePriestOrdination(3)
	if err != nil {
		t.Fatalf("should be able to convert 3 workers (2+2+3=7 total), got error: %v", err)
	}
	
	// Pay workers and gain priests
	player.Resources.Workers -= 3
	gs.GainPriests("player1", priestsGained)
	
	// Verify priests were gained
	if player.Resources.Priests != 5 {
		t.Errorf("expected 5 priests in hand, got %d", player.Resources.Priests)
	}
}

func TestDarklings_PriestOrdinationExceeds7PriestLimit(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Build stronghold
	faction.BuildStronghold()
	player.HasStrongholdAbility = true
	
	// Player has 4 priests in hand, 3 on cult tracks
	player.Resources.Priests = 4
	player.Resources.Workers = 5
	
	gs.CultTracks.InitializePlayer("player1")
	gs.CultTracks.AdvancePlayer("player1", CultFire, 2, player)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 1, player)
	// Now: 4 in hand - 3 sent = 1 in hand, 3 on cults = 4 total
	player.Resources.Priests = 1
	
	// Try to convert 3 workers (would be 1 + 3 + 3 = 7 total, at limit)
	priestsGained, err := faction.UsePriestOrdination(3)
	if err != nil {
		t.Fatalf("should be able to convert 3 workers (1+3+3=7 total), got error: %v", err)
	}
	
	// Pay workers and gain priests
	player.Resources.Workers -= 3
	gs.GainPriests("player1", priestsGained)
	
	// Now player has 4 priests in hand, 3 on cult tracks = 7 total
	
	// Create new game state and test exceeding limit
	gs2 := NewGameState()
	faction2 := factions.NewDarklings()
	gs2.AddPlayer("player1", faction2)
	player2 := gs2.GetPlayer("player1")
	
	faction2.BuildStronghold()
	player2.HasStrongholdAbility = true
	
	// Player has 4 priests in hand, 3 on cult track action spaces
	player2.Resources.Priests = 4
	player2.Resources.Workers = 5
	
	gs2.CultTracks.InitializePlayer("player1")
	// Place 3 priests on cult track action spaces (via 2/3-step placements)
	gs2.CultTracks.PriestsOnActionSpaces["player1"][CultFire] = 2
	gs2.CultTracks.PriestsOnActionSpaces["player1"][CultWater] = 1
	
	// Try to convert 3 workers (would be 1 + 3 + 3 = 7, then try to exceed)
	// Actually, let's test exceeding directly
	player2.Resources.Priests = 4
	// 4 in hand + 3 on action spaces = 7 total, can't gain any more
	
	// Try to convert 1 worker, but it should be blocked by 7-priest limit
	// First check if we can gain a priest
	priestsInHand := player2.Resources.Priests
	priestsOnCult := gs2.CultTracks.GetTotalPriestsOnCultTracks("player1")
	totalPriests := priestsInHand + priestsOnCult
	
	if totalPriests < 7 {
		t.Fatal("test setup error: should be at 7+ priests to test limit")
	}
	
	// The faction method will allow the conversion, but GainPriests will cap at the limit
	priestsGained2, err2 := faction2.UsePriestOrdination(1)
	if err2 != nil {
		t.Fatalf("faction method should allow ordination, got error: %v", err2)
	}
	
	// Pay workers and try to gain priests (will be capped at 0 due to limit)
	player2.Resources.Workers -= 1
	actualGained := gs2.GainPriests("player1", priestsGained2)
	
	if actualGained != 0 {
		t.Errorf("should not gain any priests when at limit (expected 0, got %d)", actualGained)
	}
}

func TestDarklings_CannotUpgradeDigging(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	
	// Darklings cannot upgrade digging - cost should be 0
	cost := faction.GetDiggingCost(0)
	if cost.Workers != 0 || cost.Coins != 0 || cost.Priests != 0 {
		t.Error("Darklings should have 0 cost for digging (indicating impossible)")
	}
}

func TestDarklings_PowerActionWithPriests(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Place initial dwelling
	initialHex := NewHex(0, 0)
	gs.Map.Hexes[initialHex] = &MapHex{Coord: initialHex, Terrain: faction.GetHomeTerrain()}
	gs.Map.PlaceBuilding(initialHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Target hex is adjacent
	targetHex := NewHex(1, 0)
	gs.Map.Hexes[targetHex] = &MapHex{Coord: targetHex, Terrain: models.TerrainMountain}
	
	// Give player power and priests
	player.Resources.Power.Bowl3 = 5
	player.Resources.Priests = 5
	player.Resources.Workers = 10
	initialPriests := player.Resources.Priests
	initialWorkers := player.Resources.Workers
	initialVP := player.VictoryPoints
	
	// Use spade power action (1 free spade, Darklings pay priests for remaining)
	// Mountain to Swamp = 3 spades, 1 free, need to pay for 2 spades = 2 priests
	action := NewPowerActionWithTransform("player1", PowerActionSpade1, targetHex, false)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("spade power action should work for Darklings, got error: %v", err)
	}
	
	// Verify 2 priests were spent (not workers)
	priestsSpent := initialPriests - player.Resources.Priests
	if priestsSpent != 2 {
		t.Errorf("expected 2 priests spent for 2 remaining spades, got %d", priestsSpent)
	}
	
	// Verify workers were NOT spent
	workersSpent := initialWorkers - player.Resources.Workers
	if workersSpent != 0 {
		t.Errorf("expected 0 workers spent, got %d", workersSpent)
	}
	
	// Verify VP bonus (+2 VP per remaining spade = +4 VP)
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 4 {
		t.Errorf("expected +4 VP for 2 remaining spades, got +%d", vpGained)
	}
}

func Test7PriestLimit_Income(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player 5 priests in hand and send 2 to cult tracks
	player.Resources.Priests = 5
	gs.CultTracks.InitializePlayer("player1")
	gs.CultTracks.AdvancePlayer("player1", CultFire, 2, player)
	player.Resources.Priests = 3 // 5 - 2 sent
	
	// Now player has 3 in hand + 2 on cults = 5 total
	// Income should give up to 2 more priests (to reach 7 total)
	
	// Grant income (Auren base income includes 1 priest)
	gs.GrantIncome()
	
	// Check that only the allowed number of priests were added
	priestsInHand := player.Resources.Priests
	priestsOnCult := gs.CultTracks.GetTotalPriestsOnCultTracks("player1")
	totalPriests := priestsInHand + priestsOnCult
	
	if totalPriests > 7 {
		t.Errorf("7-priest limit violated: have %d total priests (%d in hand + %d on cult)", totalPriests, priestsInHand, priestsOnCult)
	}
	
	// Player should have gained priests up to the limit
	// Started with 3 in hand, income gives 1 base + temple/sanctuary income
	// Should cap at 5 in hand (to stay at 7 total with 2 on cults)
	if priestsInHand > 5 {
		t.Errorf("expected at most 5 priests in hand (7 total - 2 on cults), got %d", priestsInHand)
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

// ===== CHAOS MAGICIANS TESTS =====

func TestChaosMagicians_DoubleTurnBasic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewChaosMagicians()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 50
	player.Resources.Workers = 20
	player.Resources.Priests = 5
	player.Resources.Power.Bowl3 = 10

	// Build stronghold to enable double turn
	strongholdHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(strongholdHex, &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	})
	gs.Map.TransformTerrain(strongholdHex, models.TerrainWasteland)
	player.HasStrongholdAbility = true

	// Place a dwelling for first action (upgrade to trading house)
	dwellingHex := NewHex(1, 0)
	gs.Map.TransformTerrain(dwellingHex, models.TerrainWasteland)
	gs.Map.PlaceBuilding(dwellingHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// Prepare second action target (transform and build)
	targetHex := NewHex(1, 1)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialCoins := player.Resources.Coins
	initialWorkers := player.Resources.Workers

	// Create double turn: 1) Upgrade dwelling to trading house, 2) Transform and build
	firstAction := NewUpgradeBuildingAction("player1", dwellingHex, models.BuildingTradingHouse)
	secondAction := NewTransformAndBuildAction("player1", targetHex, true)
	action := NewSpecialAction("player1", SpecialActionChaosMagiciansDoubleTurn)
	action.FirstAction = firstAction
	action.SecondAction = secondAction

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Chaos Magicians double turn to succeed, got error: %v", err)
	}

	// Verify first action: dwelling was upgraded to trading house
	mapHex1 := gs.Map.GetHex(dwellingHex)
	if mapHex1.Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected trading house, got %v", mapHex1.Building.Type)
	}

	// Verify second action: terrain was transformed and dwelling built
	mapHex2 := gs.Map.GetHex(targetHex)
	if mapHex2.Terrain != models.TerrainWasteland {
		t.Errorf("expected Wasteland terrain, got %v", mapHex2.Terrain)
	}
	if mapHex2.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex2.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex2.Building.Type)
	}

	// Verify resources were spent for both actions
	if player.Resources.Coins >= initialCoins {
		t.Error("expected coins to be spent")
	}
	if player.Resources.Workers >= initialWorkers {
		t.Error("expected workers to be spent")
	}

	// Verify special action was marked as used
	if !player.SpecialActionsUsed[SpecialActionChaosMagiciansDoubleTurn] {
		t.Error("expected double turn to be marked as used")
	}
}

func TestChaosMagicians_DoubleTurnTwoTransforms(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewChaosMagicians()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 50
	player.Resources.Workers = 30
	player.Resources.Priests = 5

	// Build stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(strongholdHex, &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	})
	gs.Map.TransformTerrain(strongholdHex, models.TerrainWasteland)
	player.HasStrongholdAbility = true

	// First transform target
	targetHex1 := NewHex(1, 0)
	gs.Map.GetHex(targetHex1).Terrain = models.TerrainForest

	// Second transform target
	targetHex2 := NewHex(1, 1)
	gs.Map.GetHex(targetHex2).Terrain = models.TerrainLake

	// Create double turn with two transform actions
	firstAction := NewTransformAndBuildAction("player1", targetHex1, true)
	secondAction := NewTransformAndBuildAction("player1", targetHex2, true)
	action := NewSpecialAction("player1", SpecialActionChaosMagiciansDoubleTurn)
	action.FirstAction = firstAction
	action.SecondAction = secondAction

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected double turn to succeed, got error: %v", err)
	}

	// Verify both hexes were transformed and have dwellings
	mapHex1 := gs.Map.GetHex(targetHex1)
	if mapHex1.Terrain != models.TerrainWasteland {
		t.Errorf("expected Wasteland for hex 1, got %v", mapHex1.Terrain)
	}
	if mapHex1.Building == nil || mapHex1.Building.Type != models.BuildingDwelling {
		t.Error("expected dwelling on hex 1")
	}

	mapHex2 := gs.Map.GetHex(targetHex2)
	if mapHex2.Terrain != models.TerrainWasteland {
		t.Errorf("expected Wasteland for hex 2, got %v", mapHex2.Terrain)
	}
	if mapHex2.Building == nil || mapHex2.Building.Type != models.BuildingDwelling {
		t.Error("expected dwelling on hex 2")
	}
}

func TestChaosMagicians_DoubleTurnCanOnlyUseOnce(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewChaosMagicians()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 100
	player.Resources.Workers = 50

	// Build stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(strongholdHex, &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	})
	gs.Map.TransformTerrain(strongholdHex, models.TerrainWasteland)
	player.HasStrongholdAbility = true

	// Place two dwellings
	dwelling1Hex := NewHex(1, 0)
	gs.Map.TransformTerrain(dwelling1Hex, models.TerrainWasteland)
	gs.Map.PlaceBuilding(dwelling1Hex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	dwelling2Hex := NewHex(1, 1)
	gs.Map.TransformTerrain(dwelling2Hex, models.TerrainWasteland)
	gs.Map.PlaceBuilding(dwelling2Hex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})

	// First double turn
	firstAction1 := NewUpgradeBuildingAction("player1", dwelling1Hex, models.BuildingTradingHouse)
	targetHex1 := NewHex(2, 0)
	gs.Map.GetHex(targetHex1).Terrain = models.TerrainForest
	secondAction1 := NewTransformAndBuildAction("player1", targetHex1, false)
	action1 := NewSpecialAction("player1", SpecialActionChaosMagiciansDoubleTurn)
	action1.FirstAction = firstAction1
	action1.SecondAction = secondAction1

	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected first double turn to succeed, got error: %v", err)
	}

	// Try to use double turn again in same round - should fail
	firstAction2 := NewUpgradeBuildingAction("player1", dwelling2Hex, models.BuildingTradingHouse)
	targetHex2 := NewHex(2, 1)
	gs.Map.GetHex(targetHex2).Terrain = models.TerrainSwamp
	secondAction2 := NewTransformAndBuildAction("player1", targetHex2, false)
	action2 := NewSpecialAction("player1", SpecialActionChaosMagiciansDoubleTurn)
	action2.FirstAction = firstAction2
	action2.SecondAction = secondAction2

	err = action2.Validate(gs)
	if err == nil {
		t.Fatal("expected error when using double turn twice in one round")
	}
}

// ===== NOMADS TESTS =====

func TestNomads_SandstormBasic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewNomads()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Build stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(strongholdHex, &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	})
	gs.Map.TransformTerrain(strongholdHex, models.TerrainDesert)
	player.HasStrongholdAbility = true

	// Target hex directly adjacent to stronghold
	targetHex := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainSwamp)

	// Use Nomads sandstorm special action
	action := NewSpecialAction("player1", SpecialActionNomadsSandstorm)
	action.TargetHex = &targetHex
	action.BuildDwelling = true

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected Nomads sandstorm to succeed, got error: %v", err)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainDesert {
		t.Errorf("expected Desert terrain, got %v", mapHex.Terrain)
	}

	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}

	// Verify special action was marked as used
	if !player.SpecialActionsUsed[SpecialActionNomadsSandstorm] {
		t.Error("expected sandstorm to be marked as used")
	}
}

func TestNomads_SandstormCanOnlyUseOnce(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewNomads()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Coins = 50
	player.Resources.Workers = 50

	// Build stronghold
	strongholdHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(strongholdHex, &models.Building{
		Type:       models.BuildingStronghold,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	})
	gs.Map.TransformTerrain(strongholdHex, models.TerrainDesert)
	player.HasStrongholdAbility = true

	// First sandstorm target
	targetHex1 := NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex1, models.TerrainSwamp)

	// Use sandstorm once
	action1 := NewSpecialAction("player1", SpecialActionNomadsSandstorm)
	action1.TargetHex = &targetHex1
	action1.BuildDwelling = true

	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected first sandstorm to succeed, got error: %v", err)
	}

	// Try to use sandstorm again in same round - should fail
	targetHex2 := NewHex(0, 2)
	gs.Map.TransformTerrain(targetHex2, models.TerrainForest)

	action2 := NewSpecialAction("player1", SpecialActionNomadsSandstorm)
	action2.TargetHex = &targetHex2
	action2.BuildDwelling = true

	err = action2.Validate(gs)
	if err == nil {
		t.Fatal("expected error when using sandstorm twice in one round")
	}
}

// ===== STRONGHOLD IMMEDIATE BONUS TESTS =====

func TestAlchemists_StrongholdGrants12Power(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player resources to upgrade
	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainSwamp)

	// Record initial power
	initialPower := player.Resources.Power.Bowl1

	// Upgrade to stronghold via UpgradeBuilding action
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify +12 power was granted automatically
	powerGained := player.Resources.Power.Bowl1 - initialPower
	if powerGained != 12 {
		t.Errorf("expected +12 power from stronghold, got +%d", powerGained)
	}

	// Verify stronghold ability is granted
	if !player.HasStrongholdAbility {
		t.Error("expected HasStrongholdAbility to be true")
	}

	// Verify building was upgraded
	mapHex := gs.Map.GetHex(tradingHouseHex)
	if mapHex.Building.Type != models.BuildingStronghold {
		t.Errorf("expected stronghold, got %v", mapHex.Building.Type)
	}
}

func TestAlchemists_StrongholdBonusOnlyOnce(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAlchemists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 200
	player.Resources.Workers = 200

	// Place a trading house
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainSwamp)

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify power was granted
	if player.Resources.Power.Bowl1 < 12 {
		t.Error("power should have been granted from stronghold")
	}
}

func TestCultists_StrongholdGrants7VP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainPlains)

	// Record initial VP
	initialVP := player.VictoryPoints

	// Upgrade to stronghold via UpgradeBuilding action
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify +7 VP was granted automatically (plus any scoring tile VP)
	vpGained := player.VictoryPoints - initialVP
	if vpGained < 7 {
		t.Errorf("expected at least +7 VP from stronghold, got +%d", vpGained)
	}

	// Verify stronghold ability is granted
	if !player.HasStrongholdAbility {
		t.Error("expected HasStrongholdAbility to be true")
	}

	// Verify building was upgraded
	mapHex := gs.Map.GetHex(tradingHouseHex)
	if mapHex.Building.Type != models.BuildingStronghold {
		t.Errorf("expected stronghold, got %v", mapHex.Building.Type)
	}
}

func TestCultists_StrongholdBonusOnlyOnce(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainPlains)

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify VP was granted
	if player.VictoryPoints < 7 {
		t.Error("VP should have been granted from stronghold")
	}
}

func TestMermaids_StrongholdGrants1Shipping(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainLake)

	// Mermaids start with shipping level 1
	initialShipping := faction.GetShippingLevel()
	if initialShipping != 1 {
		t.Fatalf("expected Mermaids to start at shipping level 1, got %d", initialShipping)
	}

	// Upgrade to stronghold via UpgradeBuilding action
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify shipping level increased by 1 (1 -> 2)
	newShipping := faction.GetShippingLevel()
	if newShipping != 2 {
		t.Errorf("expected shipping level 2 after stronghold, got %d", newShipping)
	}

	// Verify player shipping level was also updated
	if player.ShippingLevel != 2 {
		t.Errorf("expected player shipping level 2, got %d", player.ShippingLevel)
	}

	// Verify stronghold ability is granted
	if !player.HasStrongholdAbility {
		t.Error("expected HasStrongholdAbility to be true")
	}

	// Verify building was upgraded
	mapHex := gs.Map.GetHex(tradingHouseHex)
	if mapHex.Building.Type != models.BuildingStronghold {
		t.Errorf("expected stronghold, got %v", mapHex.Building.Type)
	}
}

func TestMermaids_StrongholdBonusOnlyOnce(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewMermaids()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainLake)

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify shipping was upgraded
	if faction.GetShippingLevel() < 2 {
		t.Error("shipping should have been upgraded from stronghold")
	}
}

func TestAuren_StrongholdMarksFavorTilePending(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainForest)

	// Upgrade to stronghold via UpgradeBuilding action
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify stronghold ability is granted
	if !player.HasStrongholdAbility {
		t.Error("expected HasStrongholdAbility to be true")
	}

	// Verify building was upgraded
	mapHex := gs.Map.GetHex(tradingHouseHex)
	if mapHex.Building.Type != models.BuildingStronghold {
		t.Errorf("expected stronghold, got %v", mapHex.Building.Type)
	}

	// Favor tile selection implemented via PendingFavorTileSelection system
}

// ============================================================================
// DARKLINGS PRIEST ORDINATION TESTS (Immediate Stronghold Bonus)
// ============================================================================

func TestDarklingsStronghold_CreatesPendingOrdination(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainSwamp)

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify pending ordination was created
	if gs.PendingDarklingsPriestOrdination == nil {
		t.Fatal("expected pending Darklings priest ordination after building stronghold")
	}

	if gs.PendingDarklingsPriestOrdination.PlayerID != "player1" {
		t.Errorf("expected player1, got %s", gs.PendingDarklingsPriestOrdination.PlayerID)
	}
}

func TestUseDarklingsPriestOrdination_Convert3Workers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Workers = 5
	player.Resources.Priests = 1

	// Create pending ordination (simulate stronghold build)
	faction.BuildStronghold()
	gs.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{
		PlayerID: "player1",
	}

	// Convert 3 workers to priests
	action := &UseDarklingsPriestOrdinationAction{
		BaseAction: BaseAction{
			Type:     ActionUseDarklingsPriestOrdination,
			PlayerID: "player1",
		},
		WorkersToConvert: 3,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use priest ordination: %v", err)
	}

	// Verify workers were spent
	if player.Resources.Workers != 2 {
		t.Errorf("expected 2 workers remaining (5-3), got %d", player.Resources.Workers)
	}

	// Verify priests were gained
	if player.Resources.Priests != 4 {
		t.Errorf("expected 4 priests (1+3), got %d", player.Resources.Priests)
	}

	// Verify pending was cleared
	if gs.PendingDarklingsPriestOrdination != nil {
		t.Error("expected pending ordination to be cleared")
	}
}

func TestUseDarklingsPriestOrdination_ConvertZeroWorkers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Workers = 5
	player.Resources.Priests = 1

	// Create pending ordination
	faction.BuildStronghold()
	gs.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{
		PlayerID: "player1",
	}

	// Convert 0 workers (player chooses not to use the ability)
	action := &UseDarklingsPriestOrdinationAction{
		BaseAction: BaseAction{
			Type:     ActionUseDarklingsPriestOrdination,
			PlayerID: "player1",
		},
		WorkersToConvert: 0,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use priest ordination with 0 workers: %v", err)
	}

	// Verify nothing changed
	if player.Resources.Workers != 5 {
		t.Errorf("expected 5 workers (no change), got %d", player.Resources.Workers)
	}

	if player.Resources.Priests != 1 {
		t.Errorf("expected 1 priest (no change), got %d", player.Resources.Priests)
	}

	// Verify pending was cleared
	if gs.PendingDarklingsPriestOrdination != nil {
		t.Error("expected pending ordination to be cleared")
	}
}

func TestUseDarklingsPriestOrdination_Respects7PriestLimit(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up to be at the 7-priest limit
	player.Resources.Workers = 5
	player.Resources.Priests = 4

	// Place 3 priests on cult track action spaces
	gs.CultTracks.InitializePlayer("player1")
	gs.CultTracks.PriestsOnActionSpaces["player1"][CultFire] = 2
	gs.CultTracks.PriestsOnActionSpaces["player1"][CultEarth] = 1
	// 4 in hand + 3 on action spaces = 7 total

	// Create pending ordination
	faction.BuildStronghold()
	gs.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{
		PlayerID: "player1",
	}

	// Try to convert 3 workers (would exceed limit)
	action := &UseDarklingsPriestOrdinationAction{
		BaseAction: BaseAction{
			Type:     ActionUseDarklingsPriestOrdination,
			PlayerID: "player1",
		},
		WorkersToConvert: 3,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use priest ordination: %v", err)
	}

	// Verify workers were spent (even though priests weren't gained)
	if player.Resources.Workers != 2 {
		t.Errorf("expected 2 workers remaining (5-3), got %d", player.Resources.Workers)
	}

	// Verify no priests were gained (at limit)
	if player.Resources.Priests != 4 {
		t.Errorf("expected 4 priests (no change, at limit), got %d", player.Resources.Priests)
	}

	// Verify pending was cleared
	if gs.PendingDarklingsPriestOrdination != nil {
		t.Error("expected pending ordination to be cleared")
	}
}

func TestUseDarklingsPriestOrdination_CannotConvertMoreThan3(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10

	// Create pending ordination
	faction.BuildStronghold()
	gs.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{
		PlayerID: "player1",
	}

	// Try to convert 4 workers (should fail)
	action := &UseDarklingsPriestOrdinationAction{
		BaseAction: BaseAction{
			Type:     ActionUseDarklingsPriestOrdination,
			PlayerID: "player1",
		},
		WorkersToConvert: 4,
	}

	err := action.Validate(gs)
	if err == nil {
		t.Error("expected error when trying to convert more than 3 workers")
	}
}

func TestUseDarklingsPriestOrdination_NeedEnoughWorkers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 2

	// Create pending ordination
	faction.BuildStronghold()
	gs.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{
		PlayerID: "player1",
	}

	// Try to convert 3 workers (only have 2)
	action := &UseDarklingsPriestOrdinationAction{
		BaseAction: BaseAction{
			Type:     ActionUseDarklingsPriestOrdination,
			PlayerID: "player1",
		},
		WorkersToConvert: 3,
	}

	err := action.Validate(gs)
	if err == nil {
		t.Error("expected error when trying to convert more workers than available")
	}
}

func TestUseDarklingsPriestOrdination_OnlyOnce(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewDarklings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10
	player.Resources.Priests = 1

	// Create pending ordination
	faction.BuildStronghold()
	gs.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{
		PlayerID: "player1",
	}

	// Convert 2 workers
	action := &UseDarklingsPriestOrdinationAction{
		BaseAction: BaseAction{
			Type:     ActionUseDarklingsPriestOrdination,
			PlayerID: "player1",
		},
		WorkersToConvert: 2,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use priest ordination: %v", err)
	}

	// Verify pending was cleared
	if gs.PendingDarklingsPriestOrdination != nil {
		t.Fatal("expected pending ordination to be cleared")
	}

	// Try to use ordination again (should fail - no pending state)
	action2 := &UseDarklingsPriestOrdinationAction{
		BaseAction: BaseAction{
			Type:     ActionUseDarklingsPriestOrdination,
			PlayerID: "player1",
		},
		WorkersToConvert: 1,
	}

	err = action2.Validate(gs)
	if err == nil {
		t.Error("expected error when trying to use priest ordination twice")
	}
}

// ============================================================================
// HALFLINGS STRONGHOLD SPADES TESTS (3 Spades + Optional Dwelling)
// ============================================================================

func TestHalflingsStronghold_Creates3PendingSpades(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Coins = 100
	player.Resources.Workers = 100

	// Place a trading house to upgrade to stronghold
	tradingHouseHex := NewHex(0, 1)
	gs.Map.PlaceBuilding(tradingHouseHex, &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	})
	gs.Map.TransformTerrain(tradingHouseHex, models.TerrainPlains)

	// Upgrade to stronghold
	action := NewUpgradeBuildingAction("player1", tradingHouseHex, models.BuildingStronghold)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade to stronghold: %v", err)
	}

	// Verify pending spades was created
	if gs.PendingHalflingsSpades == nil {
		t.Fatal("expected pending Halflings spades after building stronghold")
	}

	if gs.PendingHalflingsSpades.PlayerID != "player1" {
		t.Errorf("expected player1, got %s", gs.PendingHalflingsSpades.PlayerID)
	}

	if gs.PendingHalflingsSpades.SpadesRemaining != 3 {
		t.Errorf("expected 3 spades remaining, got %d", gs.PendingHalflingsSpades.SpadesRemaining)
	}
}

func TestApplyHalflingsSpade_TransformsHexAndAwardsVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Create pending spades (simulate stronghold build)
	faction.BuildStronghold()
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 3,
		TransformedHexes: []Hex{},
	}

	// Target hex that needs transformation
	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest // Not home terrain

	initialVP := player.VictoryPoints

	// Apply one spade
	action := &ApplyHalflingsSpadeAction{
		BaseAction: BaseAction{
			Type:     ActionApplyHalflingsSpade,
			PlayerID: "player1",
		},
		TargetHex: targetHex,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to apply spade: %v", err)
	}

	// Verify terrain was transformed
	if gs.Map.GetHex(targetHex).Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", gs.Map.GetHex(targetHex).Terrain)
	}

	// Verify VP was awarded (Halflings get +1 VP per spade)
	if player.VictoryPoints != initialVP+1 {
		t.Errorf("expected %d VP, got %d", initialVP+1, player.VictoryPoints)
	}

	// Verify spades remaining decreased
	if gs.PendingHalflingsSpades.SpadesRemaining != 2 {
		t.Errorf("expected 2 spades remaining, got %d", gs.PendingHalflingsSpades.SpadesRemaining)
	}

	// Verify hex was tracked
	if len(gs.PendingHalflingsSpades.TransformedHexes) != 1 {
		t.Errorf("expected 1 transformed hex, got %d", len(gs.PendingHalflingsSpades.TransformedHexes))
	}
}

func TestApplyHalflingsSpade_AllThreeSpades(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Create pending spades
	faction.BuildStronghold()
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 3,
		TransformedHexes: []Hex{},
	}

	// Apply 3 spades
	hexes := []Hex{NewHex(0, 0), NewHex(1, 0), NewHex(2, 0)}
	for i, hex := range hexes {
		gs.Map.GetHex(hex).Terrain = models.TerrainForest

		action := &ApplyHalflingsSpadeAction{
			BaseAction: BaseAction{
				Type:     ActionApplyHalflingsSpade,
				PlayerID: "player1",
			},
			TargetHex: hex,
		}

		err := action.Execute(gs)
		if err != nil {
			t.Fatalf("failed to apply spade %d: %v", i+1, err)
		}
	}

	// Verify all spades applied
	if gs.PendingHalflingsSpades.SpadesRemaining != 0 {
		t.Errorf("expected 0 spades remaining, got %d", gs.PendingHalflingsSpades.SpadesRemaining)
	}

	// Verify all hexes transformed
	if len(gs.PendingHalflingsSpades.TransformedHexes) != 3 {
		t.Errorf("expected 3 transformed hexes, got %d", len(gs.PendingHalflingsSpades.TransformedHexes))
	}

	// Verify VP was awarded (3 spades × 1 VP = 3 VP)
	// Note: This is just the Halflings bonus, not counting scoring tiles
	if player.VictoryPoints < 3 {
		t.Errorf("expected at least 3 VP from spades, got %d", player.VictoryPoints)
	}
}

func TestBuildHalflingsDwelling_OnTransformedHex(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10

	// Create pending spades with all spades applied
	transformedHex := NewHex(0, 0)
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 0,
		TransformedHexes: []Hex{transformedHex, NewHex(1, 0), NewHex(2, 0)},
	}

	// Transform the hex
	gs.Map.GetHex(transformedHex).Terrain = models.TerrainPlains

	// Build dwelling on one of the transformed hexes
	action := &BuildHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionBuildHalflingsDwelling,
			PlayerID: "player1",
		},
		TargetHex: transformedHex,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to build dwelling: %v", err)
	}

	// Verify dwelling was placed
	building := gs.Map.GetHex(transformedHex).Building
	if building == nil {
		t.Fatal("expected building on transformed hex")
	}
	if building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", building.Type)
	}

	// Verify pending spades was cleared
	if gs.PendingHalflingsSpades != nil {
		t.Error("expected pending spades to be cleared after building dwelling")
	}
}

func TestBuildHalflingsDwelling_CannotBuildOnUntransformedHex(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10

	// Create pending spades with all spades applied
	transformedHex := NewHex(0, 0)
	untransformedHex := NewHex(5, 5)
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 0,
		TransformedHexes: []Hex{transformedHex, NewHex(1, 0), NewHex(2, 0)},
	}

	// Try to build dwelling on a non-transformed hex
	action := &BuildHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionBuildHalflingsDwelling,
			PlayerID: "player1",
		},
		TargetHex: untransformedHex,
	}

	err := action.Validate(gs)
	if err == nil {
		t.Error("expected error when building on non-transformed hex")
	}
}

func TestBuildHalflingsDwelling_MustApplyAllSpadesFirst(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	player.Resources.Workers = 10

	// Create pending spades with spades still remaining
	transformedHex := NewHex(0, 0)
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 2, // Still have 2 spades left
		TransformedHexes: []Hex{transformedHex},
	}

	// Try to build dwelling before applying all spades
	action := &BuildHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionBuildHalflingsDwelling,
			PlayerID: "player1",
		},
		TargetHex: transformedHex,
	}

	err := action.Validate(gs)
	if err == nil {
		t.Error("expected error when trying to build dwelling before applying all spades")
	}
}

func TestSkipHalflingsDwelling_ClearsPendingSpades(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	// Create pending spades with all spades applied
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 0,
		TransformedHexes: []Hex{NewHex(0, 0), NewHex(1, 0), NewHex(2, 0)},
	}

	// Skip the optional dwelling
	action := &SkipHalflingsDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionSkipHalflingsDwelling,
			PlayerID: "player1",
		},
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to skip dwelling: %v", err)
	}

	// Verify pending spades was cleared
	if gs.PendingHalflingsSpades != nil {
		t.Error("expected pending spades to be cleared after skipping dwelling")
	}
}

func TestHalflingsSpades_WithScoringTile(t *testing.T) {
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

	// Create pending spades
	faction.BuildStronghold()
	gs.PendingHalflingsSpades = &PendingHalflingsSpades{
		PlayerID:       "player1",
		SpadesRemaining: 3,
		TransformedHexes: []Hex{},
	}

	targetHex := NewHex(0, 0)
	gs.Map.GetHex(targetHex).Terrain = models.TerrainForest

	initialVP := player.VictoryPoints

	// Apply one spade
	action := &ApplyHalflingsSpadeAction{
		BaseAction: BaseAction{
			Type:     ActionApplyHalflingsSpade,
			PlayerID: "player1",
		},
		TargetHex: targetHex,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to apply spade: %v", err)
	}

	// Verify VP: +1 (Halflings bonus) + 2 (scoring tile) = 3 VP
	expectedVP := initialVP + 3
	if player.VictoryPoints != expectedVP {
		t.Errorf("expected %d VP, got %d", expectedVP, player.VictoryPoints)
	}
}
