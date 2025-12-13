package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestSnapshotRoundTrip(t *testing.T) {
	// 1. Setup a complex game state
	gs := game.NewGameState()

	// Add players
	witches := factions.NewWitches()
	gs.AddPlayer("player1", witches)
	nomads := factions.NewNomads()
	gs.AddPlayer("player2", nomads)

	// Set some state
	p1 := gs.GetPlayer("player1")
	p1.VictoryPoints = 20
	p1.Resources.Coins = 10
	p1.Resources.Workers = 5
	p1.Resources.Priests = 2
	p1.Resources.Power.Bowl1 = 1
	p1.Resources.Power.Bowl2 = 2
	p1.Resources.Power.Bowl3 = 3
	p1.CultPositions[game.CultFire] = 3
	p1.ShippingLevel = 1

	p2 := gs.GetPlayer("player2")
	p2.VictoryPoints = 15
	p2.Resources.Coins = 5
	p2.Resources.Workers = 2
	p2.Resources.Priests = 1
	p2.CultPositions[game.CultWater] = 2
	p2.DiggingLevel = 1

	// Place buildings
	hex1 := board.NewHex(0, 0)
	gs.Map.GetHex(hex1).Terrain = models.TerrainForest
	gs.Map.GetHex(hex1).Building = &models.Building{
		Type:     models.BuildingDwelling,
		PlayerID: "player1",
		Faction:  models.FactionWitches,
	}
	// p1.Structures[models.BuildingDwelling]++

	hex2 := board.NewHex(1, 0)
	gs.Map.GetHex(hex2).Terrain = models.TerrainDesert
	gs.Map.GetHex(hex2).Building = &models.Building{
		Type:     models.BuildingTradingHouse,
		PlayerID: "player2",
		Faction:  models.FactionNomads,
	}
	// p2.Structures[models.BuildingTradingHouse]++

	// Terraformed hex without building
	hex3 := board.NewHex(2, 0)
	gs.Map.GetHex(hex3).Terrain = models.TerrainSwamp // Witches home is Forest, so Swamp is terraformed? No, Swamp is Alchemists/Darklings.
	// Let's just set it to something non-default if we knew the map.
	// Since we use base map, (2,0) might be something.
	// But let's just set it.
	gs.Map.GetHex(hex3).Terrain = models.TerrainSwamp

	// Set global state
	gs.Round = 2
	gs.Phase = game.PhaseAction
	gs.CurrentPlayerIndex = 0 // player1 starts
	gs.TurnOrder = []string{"player1", "player2"}

	// 2. Generate Snapshot
	snapshot := GenerateSnapshot(gs)
	t.Logf("Generated Snapshot:\n%s", snapshot)

	// 3. Parse Snapshot
	parsedGS, err := ParseSnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to parse snapshot: %v", err)
	}

	// 4. Compare
	if parsedGS.Round != gs.Round {
		t.Errorf("Round mismatch: expected %d, got %d", gs.Round, parsedGS.Round)
	}
	if parsedGS.Phase != gs.Phase {
		t.Errorf("Phase mismatch: expected %v, got %v", gs.Phase, parsedGS.Phase)
	}
	parsedCurrentPID := parsedGS.TurnOrder[parsedGS.CurrentPlayerIndex]
	originalCurrentPID := gs.TurnOrder[gs.CurrentPlayerIndex]
	if parsedGS.Players[parsedCurrentPID].Faction.GetType() != gs.Players[originalCurrentPID].Faction.GetType() {
		t.Errorf("CurrentTurn mismatch: expected %v, got %v", gs.Players[originalCurrentPID].Faction.GetType(), parsedGS.Players[parsedCurrentPID].Faction.GetType())
	}

	// Check Player 1
	// Parser creates players. We need to find the player corresponding to Witches.
	var parsedP1 *game.Player
	for _, p := range parsedGS.Players {
		if p.Faction.GetType() == models.FactionWitches {
			parsedP1 = p
			break
		}
	}
	if parsedP1 == nil {
		t.Logf("Parsed Players: %v", parsedGS.Players)
		for id, p := range parsedGS.Players {
			t.Logf("Player %s: Faction %v", id, p.Faction.GetType())
		}
		t.Fatalf("Parsed state missing Witches player")
	}

	if parsedP1.VictoryPoints != p1.VictoryPoints {
		t.Errorf("P1 VP mismatch: expected %d, got %d", p1.VictoryPoints, parsedP1.VictoryPoints)
	}
	if parsedP1.Resources.Coins != p1.Resources.Coins {
		t.Errorf("P1 Coins mismatch: expected %d, got %d", p1.Resources.Coins, parsedP1.Resources.Coins)
	}

	// Check Map
	parsedHex1 := parsedGS.Map.GetHex(hex1)
	if parsedHex1.Terrain != models.TerrainForest {
		t.Errorf("Hex1 Terrain mismatch: expected Forest, got %v", parsedHex1.Terrain)
	}
	if parsedHex1.Building == nil || parsedHex1.Building.Type != models.BuildingDwelling {
		t.Errorf("Hex1 Building mismatch: expected Dwelling, got %v", parsedHex1.Building)
	}
}
