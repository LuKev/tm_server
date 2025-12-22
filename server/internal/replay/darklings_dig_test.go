package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// TestDarklingsDig_BothBonuses tests that Darklings get BOTH their faction bonus (+2 VP per spade)
// AND the scoring tile VP (2 VP per spade in Round 2 with SCORE1)
func TestDarklingsDig_BothBonuses(t *testing.T) {
	gs := game.NewGameState()
	darklings := factions.NewDarklings()
	gs.AddPlayer("Darklings", darklings)

	player := gs.GetPlayer("Darklings")
	player.Resources.Coins = 10
	player.Resources.Priests = 5
	player.Resources.Workers = 5

	// Set up Round 2 scoring tile (spade scoring: 2 VP per spade)
	scoringTiles := game.GetAllScoringTiles()
	for _, tile := range scoringTiles {
		if tile.Type == game.ScoringSpades {
			gs.ScoringTiles = game.NewScoringTileState()
			gs.ScoringTiles.Tiles = append(gs.ScoringTiles.Tiles, tile, tile) // Round 1 and 2
			break
		}
	}
	gs.Round = 2

	// Place a dwelling for adjacency
	hex1 := board.Hex{Q: 0, R: 0}
	gs.Map.GetHex(hex1).Terrain = darklings.GetHomeTerrain()
	gs.Map.GetHex(hex1).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionDarklings,
		PlayerID:   "Darklings",
		PowerValue: 1,
	}

	// Target hex for build (adjacent to dwelling)
	// Set terrain to be 1 distance from Swamp (Darklings' home terrain)
	hex2 := board.Hex{Q: 1, R: 0}
	// Terrain wheel: Plains -> Swamp -> Lake -> Forest -> Mountain -> Wasteland -> Desert -> Plains
	// Swamp is 1 distance from Plains and Lake
	gs.Map.GetHex(hex2).Terrain = models.TerrainPlains // 1 distance from Swamp

	initialVP := player.VictoryPoints
	initialPriests := player.Resources.Priests

	// For Darklings, "dig 1" is just notation - create a normal TransformAndBuildAction
	action := game.NewTransformAndBuildAction("Darklings", hex2, true, models.TerrainTypeUnknown)

	// For Darklings, no pending spades should be granted
	if gs.PendingSpades != nil && gs.PendingSpades["Darklings"] > 0 {
		t.Errorf("Expected no pending spades for Darklings, got %d", gs.PendingSpades["Darklings"])
	}

	// Execute the action
	// This should:
	// 1. Pay 1 priest for 1 spade
	// 2. Award +2 VP from Darklings faction bonus
	// 3. Award +2 VP from scoring tile
	// 4. Award +2 VP for building dwelling (from buildDwelling function)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("action.Execute() error = %v", err)
	}

	// Verify priest was spent
	priestsSpent := initialPriests - player.Resources.Priests
	if priestsSpent != 1 {
		t.Errorf("Expected to spend 1 priest, spent %d", priestsSpent)
	}

	// Verify VP gains
	vpGained := player.VictoryPoints - initialVP

	// Expected VP breakdown:
	// +2 VP from Darklings faction bonus (GetTerraformVPBonus returns +2 per spade)
	// +2 VP from scoring tile (Round 2 SCORE1: 2 VP per spade)
	// +0 VP from dwelling itself (dwelling VP comes from favor tiles which we haven't set up)
	// = +4 VP total
	expectedVP := 4

	if vpGained != expectedVP {
		t.Errorf("Expected %d VP total, got %d VP", expectedVP, vpGained)
		t.Logf("Breakdown should be: +2 (Darklings bonus) + 2 (scoring tile) = 4 VP")
	}

	// Verify terrain was transformed
	if gs.Map.GetHex(hex2).Terrain != darklings.GetHomeTerrain() {
		t.Errorf("Terrain was not transformed to home terrain")
	}

	// Verify building was placed
	if gs.Map.GetHex(hex2).Building == nil {
		t.Errorf("Building was not placed")
	}

	t.Logf("✓ Darklings dig action awarded correct VP:")
	t.Logf("  - Spent: 1 priest")
	t.Logf("  - Gained: %d VP total", vpGained)
	t.Logf("  - Breakdown: +2 (Darklings faction bonus) + 2 (scoring tile) = 4 VP")
}

// TestDarklingsDig_NoScoringTile tests that Darklings still get their faction bonus
// even when there's no spade scoring tile active
func TestDarklingsDig_NoScoringTile(t *testing.T) {
	gs := game.NewGameState()
	darklings := factions.NewDarklings()
	gs.AddPlayer("Darklings", darklings)

	player := gs.GetPlayer("Darklings")
	player.Resources.Coins = 10
	player.Resources.Priests = 5
	player.Resources.Workers = 5

	// Set up Round 3 with a different scoring tile (not spades)
	scoringTiles := game.GetAllScoringTiles()
	for _, tile := range scoringTiles {
		if tile.Type == game.ScoringDwellingWater {
			gs.ScoringTiles = game.NewScoringTileState()
			gs.ScoringTiles.Tiles = append(gs.ScoringTiles.Tiles, tile, tile, tile)
			break
		}
	}
	gs.Round = 3

	// Place a dwelling for adjacency
	hex1 := board.Hex{Q: 0, R: 0}
	gs.Map.GetHex(hex1).Terrain = darklings.GetHomeTerrain()
	gs.Map.GetHex(hex1).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionDarklings,
		PlayerID:   "Darklings",
		PowerValue: 1,
	}

	initialVP := player.VictoryPoints

	// Target hex for build (adjacent to dwelling)
	hex2 := board.Hex{Q: 1, R: 0}
	gs.Map.GetHex(hex2).Terrain = models.TerrainPlains // 1 distance from Swamp

	// For Darklings, "dig 1" is just notation - create a normal TransformAndBuildAction
	action := game.NewTransformAndBuildAction("Darklings", hex2, true, models.TerrainTypeUnknown)

	// Execute the action
	if err := action.Execute(gs); err != nil {
		t.Fatalf("action.Execute() error = %v", err)
	}

	// Verify VP gains
	vpGained := player.VictoryPoints - initialVP

	// Expected VP breakdown:
	// +2 VP from Darklings faction bonus (for transforming with priests)
	// +0 VP from spade scoring (not the current scoring tile)
	// +2 VP from dwelling scoring tile (ScoringDwellingWater: 2 VP per dwelling built)
	// = +4 VP total
	expectedVP := 4

	if vpGained != expectedVP {
		t.Errorf("Expected %d VP total, got %d VP", expectedVP, vpGained)
		t.Logf("Breakdown should be: +2 (Darklings bonus) + 2 (dwelling scoring) = 4 VP")
	}

	t.Logf("✓ Darklings without spade scoring awarded correct VP:")
	t.Logf("  - Gained: %d VP total", vpGained)
	t.Logf("  - Breakdown: +2 (Darklings faction bonus) + 2 (dwelling scoring tile) = 4 VP")
}
