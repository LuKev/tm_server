package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestAwardCultRewards_MultipleThresholds(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 2 steps on Fire = 1 worker
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringStrongholdFire,
			CultTrack:        CultFire,
			CultThreshold:    2,
			CultRewardType:   CultRewardWorker,
			CultRewardAmount: 1,
		},
	}
	
	// Advance player to position 8 on Fire
	gs.CultTracks.AdvancePlayer("player1", CultFire, 8, player)
	
	initialWorkers := player.Resources.Workers
	
	// Award cult rewards
	gs.AwardCultRewards()
	
	// Should get 4 workers (8 / 2 = 4)
	workersGained := player.Resources.Workers - initialWorkers
	if workersGained != 4 {
		t.Errorf("expected 4 workers (8/2), got %d", workersGained)
	}
}

func TestAwardCultRewards_Spades(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 4 steps on Water = 1 spade
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringTradingHouseWater,
			CultTrack:        CultWater,
			CultThreshold:    4,
			CultRewardType:   CultRewardSpade,
			CultRewardAmount: 1,
		},
	}
	
	// Advance player to position 8 on Water
	gs.CultTracks.AdvancePlayer("player1", CultWater, 8, player)
	
	// Award cult rewards
	gs.AwardCultRewards()
	
	// Should have 2 pending spades (8 / 4 = 2)
	if gs.PendingSpades == nil || gs.PendingSpades["player1"] != 2 {
		t.Errorf("expected 2 pending spades, got %d", gs.PendingSpades["player1"])
	}
}

func TestExecuteCleanupPhase_Round6(t *testing.T) {
	gs := NewGameState()
	gs.Round = 6
	
	// Execute cleanup for round 6
	shouldContinue := gs.ExecuteCleanupPhase()
	
	// Game should end after round 6
	if shouldContinue {
		t.Error("game should not continue after round 6")
	}
	
	if gs.Phase != PhaseEnd {
		t.Errorf("expected phase to be PhaseEnd, got %v", gs.Phase)
	}
}

func TestExecuteCleanupPhase_Round5(t *testing.T) {
	gs := NewGameState()
	gs.Round = 5
	
	// Initialize scoring tiles (required for cleanup)
	gs.ScoringTiles.InitializeForGame()
	
	// Execute cleanup for round 5
	shouldContinue := gs.ExecuteCleanupPhase()
	
	// Game should continue
	if !shouldContinue {
		t.Error("game should continue after round 5")
	}
	
	if gs.Phase != PhaseCleanup {
		t.Errorf("expected phase to be PhaseCleanup, got %v", gs.Phase)
	}
}

func TestReturnBonusCards(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren() // Forest
	faction2 := factions.NewAlchemists() // Swamp
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	// Give players bonus cards
	gs.BonusCards.PlayerCards["player1"] = BonusCardPriest
	gs.BonusCards.PlayerCards["player2"] = BonusCard6Coins
	gs.BonusCards.PlayerHasCard["player1"] = true
	gs.BonusCards.PlayerHasCard["player2"] = true
	
	// Return bonus cards
	gs.ReturnBonusCards()
	
	// Players should no longer have cards
	if len(gs.BonusCards.PlayerCards) != 0 {
		t.Errorf("expected no player cards, got %d", len(gs.BonusCards.PlayerCards))
	}
	
	if len(gs.BonusCards.PlayerHasCard) != 0 {
		t.Errorf("expected no player has card flags, got %d", len(gs.BonusCards.PlayerHasCard))
	}
	
	// Cards should be back in available pool
	if _, ok := gs.BonusCards.Available[BonusCardPriest]; !ok {
		t.Error("priest card should be in available pool")
	}
	if _, ok := gs.BonusCards.Available[BonusCard6Coins]; !ok {
		t.Error("6 coins card should be in available pool")
	}
}

func TestResetRoundState(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set some round-specific state
	player.HasPassed = true
	gs.PassOrder = []string{"player1"}
	gs.PendingLeechOffers = map[string][]*PowerLeechOffer{
		"player1": {{Amount: 5}},
	}
	
	// Reset round state
	gs.ResetRoundState()
	
	// Check that state was reset
	if player.HasPassed {
		t.Error("HasPassed should be reset to false")
	}
	if len(gs.PassOrder) != 0 {
		t.Error("PassOrder should be cleared")
	}
	if len(gs.PendingLeechOffers) != 0 {
		t.Error("PendingLeechOffers should be cleared")
	}
}

func TestGetNextPlayerWithSpades(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren() // Forest
	faction2 := factions.NewAlchemists() // Swamp
	faction3 := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	gs.AddPlayer("player3", faction3)
	
	// Set pass order
	gs.PassOrder = []string{"player2", "player1", "player3"}
	
	// Give spades to players
	gs.PendingSpades = map[string]int{
		"player1": 1,
		"player3": 2,
	}
	
	// Should return player2 first (even though they have no spades)
	// Actually, should return player1 (first in pass order with spades)
	nextPlayer := gs.GetNextPlayerWithSpades()
	if nextPlayer != "player1" {
		t.Errorf("expected player1, got %s", nextPlayer)
	}
	
	// Use player1's spade
	gs.UseSpadeFromReward("player1")
	
	// Should now return player3
	nextPlayer = gs.GetNextPlayerWithSpades()
	if nextPlayer != "player3" {
		t.Errorf("expected player3, got %s", nextPlayer)
	}
}

func TestUseCultSpadeAction_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Give player a pending spade
	gs.PendingSpades = map[string]int{"player1": 1}
	
	// Place a dwelling to establish territory
	hex1 := NewHex(0, 0)
	gs.Map.GetHex(hex1).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Try to terraform an adjacent hex
	hex2 := NewHex(1, 0)
	action := NewUseCultSpadeAction("player1", hex2)
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use cult spade: %v", err)
	}
	
	// Terrain should be transformed
	if gs.Map.GetHex(hex2).Terrain != faction.GetHomeTerrain() {
		t.Error("terrain should be transformed to home terrain")
	}
	
	// Spade should be used
	if gs.PendingSpades["player1"] != 0 {
		t.Errorf("expected 0 pending spades, got %d", gs.PendingSpades["player1"])
	}
}

func TestUseCultSpadeAction_NotAdjacent(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Give player a pending spade
	gs.PendingSpades = map[string]int{"player1": 1}
	
	// Place a dwelling to establish territory
	hex1 := NewHex(0, 0)
	gs.Map.GetHex(hex1).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	// Try to terraform a non-adjacent hex
	hex2 := NewHex(5, 5)
	action := NewUseCultSpadeAction("player1", hex2)
	
	err := action.Execute(gs)
	if err == nil {
		t.Error("expected error when terraforming non-adjacent hex")
	}
}

func TestUseCultSpadeAction_ScoringTileVP(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: Spades (2 VP per spade)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:       ScoringSpades,
			ActionType: ScoringActionSpades,
			ActionVP:   2,
		},
	}
	
	// Give player a pending spade
	gs.PendingSpades = map[string]int{"player1": 1}
	
	// Place a dwelling to establish territory
	hex1 := NewHex(0, 0)
	gs.Map.GetHex(hex1).Terrain = faction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	
	initialVP := player.VictoryPoints
	
	// Use cult spade
	hex2 := NewHex(1, 0)
	action := NewUseCultSpadeAction("player1", hex2)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to use cult spade: %v", err)
	}
	
	// Should get 2 VP from scoring tile
	vpGained := player.VictoryPoints - initialVP
	if vpGained != 2 {
		t.Errorf("expected 2 VP from scoring tile, got %d", vpGained)
	}
}

// Additional comprehensive tests for all cult reward types

func TestAwardCultRewards_Priests(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren() // Forest
	faction2 := factions.NewAlchemists() // Swamp - different terrain from Auren
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	// Set up scoring tile: 4 steps on Water = 1 priest
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringDwellingWater,
			CultTrack:        CultWater,
			CultThreshold:    4,
			CultRewardType:   CultRewardPriest,
			CultRewardAmount: 1,
		},
	}
	
	// Player 1 at position 8 (should get 2 priests: 8/4 = 2)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 8, player1)
	
	// Player 2 at position 3 (should get 0 priests: 3/4 = 0)
	gs.CultTracks.AdvancePlayer("player2", CultWater, 3, player2)
	
	initialPriests1 := player1.Resources.Priests
	initialPriests2 := player2.Resources.Priests
	
	gs.AwardCultRewards()
	
	if player1.Resources.Priests != initialPriests1+2 {
		t.Errorf("player1: expected 2 priests, got %d", player1.Resources.Priests-initialPriests1)
	}
	
	if player2.Resources.Priests != initialPriests2 {
		t.Errorf("player2: expected 0 priests, got %d", player2.Resources.Priests-initialPriests2)
	}
}

func TestAwardCultRewards_Power(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 4 steps on Fire = 4 power
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringDwellingFire,
			CultTrack:        CultFire,
			CultThreshold:    4,
			CultRewardType:   CultRewardPower,
			CultRewardAmount: 4,
		},
	}
	
	// Set up power bowls
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0
	
	// Position 10 (should get 8 power: 10/4 = 2, 2*4 = 8)
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player)
	
	initialBowl2 := player.Resources.Power.Bowl2
	
	gs.AwardCultRewards()
	
	powerGained := player.Resources.Power.Bowl2 - initialBowl2
	if powerGained != 8 {
		t.Errorf("expected 8 power, got %d", powerGained)
	}
}

func TestAwardCultRewards_Coins(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 1 step on Earth = 1 coin
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringSpades,
			CultTrack:        CultEarth,
			CultThreshold:    1,
			CultRewardType:   CultRewardCoin,
			CultRewardAmount: 1,
		},
	}
	
	// Position 7 (should get 7 coins)
	gs.CultTracks.AdvancePlayer("player1", CultEarth, 7, player)
	
	initialCoins := player.Resources.Coins
	
	gs.AwardCultRewards()
	
	coinsGained := player.Resources.Coins - initialCoins
	if coinsGained != 7 {
		t.Errorf("expected 7 coins, got %d", coinsGained)
	}
}

func TestAwardCultRewards_PriestCoinTile(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren() // Forest
	faction2 := factions.NewAlchemists() // Swamp
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	// Set up scoring tile: Trading House + Priest (2 coins per priest sent)
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringTradingHousePriest,
			CultTrack:        CultFire,
			CultThreshold:    0,
			CultRewardType:   CultRewardCoin,
			CultRewardAmount: 2,
		},
	}
	
	// Record priests sent to cult
	gs.ScoringTiles.RecordPriestSent("player1")
	gs.ScoringTiles.RecordPriestSent("player1")
	gs.ScoringTiles.RecordPriestSent("player1") // 3 priests
	
	gs.ScoringTiles.RecordPriestSent("player2") // 1 priest
	
	initialCoins1 := player1.Resources.Coins
	initialCoins2 := player2.Resources.Coins
	
	gs.AwardCultRewards()
	
	// Player 1: 3 priests * 2 coins = 6 coins
	if player1.Resources.Coins != initialCoins1+6 {
		t.Errorf("player1: expected 6 coins, got %d", player1.Resources.Coins-initialCoins1)
	}
	
	// Player 2: 1 priest * 2 coins = 2 coins
	if player2.Resources.Coins != initialCoins2+2 {
		t.Errorf("player2: expected 2 coins, got %d", player2.Resources.Coins-initialCoins2)
	}
	
	// Priest count should be reset
	if gs.ScoringTiles.GetPriestsSent("player1") != 0 {
		t.Error("player1 priest count should be reset to 0")
	}
	if gs.ScoringTiles.GetPriestsSent("player2") != 0 {
		t.Error("player2 priest count should be reset to 0")
	}
}

func TestAwardCultRewards_Position0(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 2 steps on Fire = 1 worker
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringStrongholdFire,
			CultTrack:        CultFire,
			CultThreshold:    2,
			CultRewardType:   CultRewardWorker,
			CultRewardAmount: 1,
		},
	}
	
	// Player at position 0 (should get nothing)
	initialWorkers := player.Resources.Workers
	
	gs.AwardCultRewards()
	
	if player.Resources.Workers != initialWorkers {
		t.Errorf("expected 0 workers, got %d", player.Resources.Workers-initialWorkers)
	}
}

func TestAwardCultRewards_Position10(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up scoring tile: 2 steps on Air = 1 worker
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringStrongholdAir,
			CultTrack:        CultAir,
			CultThreshold:    2,
			CultRewardType:   CultRewardWorker,
			CultRewardAmount: 1,
		},
	}
	
	// Give player a key to reach position 10
	player.Keys = 1
	
	// Player at position 10 (max: should get 5 workers: 10/2 = 5)
	gs.CultTracks.AdvancePlayer("player1", CultAir, 10, player)
	
	// Verify player actually reached position 10
	position := gs.CultTracks.GetPosition("player1", CultAir)
	if position != 10 {
		t.Fatalf("expected position 10, got %d (player needs a key to reach position 10)", position)
	}
	
	initialWorkers := player.Resources.Workers
	
	gs.AwardCultRewards()
	
	workersGained := player.Resources.Workers - initialWorkers
	if workersGained != 5 {
		t.Errorf("expected 5 workers, got %d", workersGained)
	}
}

func TestBonusCards_AddCoinsToLeftover(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Set up available bonus cards with some coins
	gs.BonusCards.Available[BonusCardPriest] = 2
	gs.BonusCards.Available[BonusCard6Coins] = 0
	gs.BonusCards.Available[BonusCardSpade] = 1
	
	// Player 1 takes the priest card
	gs.BonusCards.PlayerCards["player1"] = BonusCardPriest
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Add coins to leftover cards (not taken by players)
	gs.BonusCards.AddCoinsToLeftoverCards()
	
	// Priest card was taken, so it shouldn't get a coin (it's not in Available anymore)
	// 6 Coins card should now have 1 coin
	if gs.BonusCards.Available[BonusCard6Coins] != 1 {
		t.Errorf("expected 1 coin on 6 coins card, got %d", gs.BonusCards.Available[BonusCard6Coins])
	}
	
	// Spade card should now have 2 coins
	if gs.BonusCards.Available[BonusCardSpade] != 2 {
		t.Errorf("expected 2 coins on spade card, got %d", gs.BonusCards.Available[BonusCardSpade])
	}
}

func TestPowerActions_ResetForNewRound(t *testing.T) {
	gs := NewGameState()
	
	// Mark some power actions as used
	gs.PowerActions.UsedActions[PowerActionBridge] = true
	gs.PowerActions.UsedActions[PowerActionPriest] = true
	
	// Reset round state (which should reset power actions)
	gs.ResetRoundState()
	
	// Power actions should be available again
	if gs.PowerActions.UsedActions[PowerActionBridge] {
		t.Error("bridge power action should be available after reset")
	}
	if gs.PowerActions.UsedActions[PowerActionPriest] {
		t.Error("priest power action should be available after reset")
	}
}

func TestMultiplePlayers_DifferentRewards(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren() // Forest
	faction2 := factions.NewAlchemists() // Swamp  
	faction3 := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	gs.AddPlayer("player3", faction3)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")
	
	// Set up scoring tile: 4 steps on Water = 1 spade
	gs.ScoringTiles.Tiles = []ScoringTile{
		{
			Type:             ScoringTradingHouseWater,
			CultTrack:        CultWater,
			CultThreshold:    4,
			CultRewardType:   CultRewardSpade,
			CultRewardAmount: 1,
		},
	}
	
	// Different positions
	gs.CultTracks.AdvancePlayer("player1", CultWater, 10, player1) // 2 spades
	gs.CultTracks.AdvancePlayer("player2", CultWater, 6, player2)  // 1 spade
	gs.CultTracks.AdvancePlayer("player3", CultWater, 2, player3)  // 0 spades
	
	gs.AwardCultRewards()
	
	if gs.PendingSpades["player1"] != 2 {
		t.Errorf("player1: expected 2 spades, got %d", gs.PendingSpades["player1"])
	}
	if gs.PendingSpades["player2"] != 1 {
		t.Errorf("player2: expected 1 spade, got %d", gs.PendingSpades["player2"])
	}
	if gs.PendingSpades["player3"] != 0 {
		t.Errorf("player3: expected 0 spades, got %d", gs.PendingSpades["player3"])
	}
}

func TestMultipleSpades_Sequential(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Give player 3 pending spades
	gs.PendingSpades = map[string]int{"player1": 3}
	
	// Use spades one by one
	for i := 3; i > 0; i-- {
		if gs.PendingSpades["player1"] != i {
			t.Errorf("expected %d spades, got %d", i, gs.PendingSpades["player1"])
		}
		
		success := gs.UseSpadeFromReward("player1")
		if !success {
			t.Errorf("failed to use spade %d", i)
		}
	}
	
	// Should have no spades left
	if gs.PendingSpades["player1"] != 0 {
		t.Errorf("expected 0 spades, got %d", gs.PendingSpades["player1"])
	}
	
	// Trying to use another should fail
	success := gs.UseSpadeFromReward("player1")
	if success {
		t.Error("should not be able to use spade when none available")
	}
}

func TestFullCleanupFlow(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewAuren() // Forest
	faction2 := factions.NewAlchemists() // Swamp
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	
	gs.Round = 3
	
	// Initialize scoring tiles
	gs.ScoringTiles.InitializeForGame()
	
	// Set up scoring tile for round 3: 2 steps Fire = 1 worker
	gs.ScoringTiles.Tiles[2] = ScoringTile{
		Type:             ScoringStrongholdFire,
		CultTrack:        CultFire,
		CultThreshold:    2,
		CultRewardType:   CultRewardWorker,
		CultRewardAmount: 1,
	}
	
	// Set up cult positions
	gs.CultTracks.AdvancePlayer("player1", CultFire, 6, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 4, player2)
	
	// Set up bonus cards
	gs.BonusCards.Available[BonusCardPriest] = 1
	gs.BonusCards.PlayerCards["player1"] = BonusCard6Coins
	gs.BonusCards.PlayerHasCard["player1"] = true
	
	// Set up round state
	player1.HasPassed = true
	player2.HasPassed = true
	gs.PassOrder = []string{"player1", "player2"}
	
	// Mark power action as used
	gs.PowerActions.UsedActions[PowerActionBridge] = true
	
	initialWorkers1 := player1.Resources.Workers
	initialWorkers2 := player2.Resources.Workers
	
	// Execute full cleanup
	shouldContinue := gs.ExecuteCleanupPhase()
	
	// Verify game continues
	if !shouldContinue {
		t.Error("game should continue after round 3")
	}
	
	// Verify cult rewards awarded (6/2=3 workers, 4/2=2 workers)
	if player1.Resources.Workers != initialWorkers1+3 {
		t.Errorf("player1: expected 3 workers, got %d", player1.Resources.Workers-initialWorkers1)
	}
	if player2.Resources.Workers != initialWorkers2+2 {
		t.Errorf("player2: expected 2 workers, got %d", player2.Resources.Workers-initialWorkers2)
	}
	
	// Verify bonus card coins added
	if gs.BonusCards.Available[BonusCardPriest] != 2 {
		t.Errorf("expected 2 coins on priest card, got %d", gs.BonusCards.Available[BonusCardPriest])
	}
	
	// Verify bonus cards returned
	if len(gs.BonusCards.PlayerCards) != 0 {
		t.Error("player cards should be empty")
	}
	
	// Verify round state reset
	if player1.HasPassed {
		t.Error("player1 HasPassed should be reset")
	}
	if player2.HasPassed {
		t.Error("player2 HasPassed should be reset")
	}
	if len(gs.PassOrder) != 0 {
		t.Error("PassOrder should be cleared")
	}
	
	// Verify power actions reset
	if gs.PowerActions.UsedActions[PowerActionBridge] {
		t.Error("power actions should be reset")
	}
}
