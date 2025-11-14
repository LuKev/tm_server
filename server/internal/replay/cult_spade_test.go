package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// TestParseTransformOnly_WithCultSpade tests that transform-only actions
// use UseCultSpadeAction when player has pending cult spades
func TestParseTransformOnly_WithCultSpade(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("Cultists", &factions.Cultists{})

	// Grant cult reward spade to Cultists
	gs.PendingSpades = make(map[string]int)
	gs.PendingSpades["Cultists"] = 1

	// Place a dwelling for adjacency
	hex1 := game.Hex{Q: 0, R: 0}
	gs.Map.GetHex(hex1).Building = &models.Building{
		Type:     models.BuildingDwelling,
		Faction:  models.FactionCultists,
		PlayerID: "Cultists",
	}

	entry := &LogEntry{
		Faction: models.FactionCultists,
		Action:  "transform G2 to yellow",
	}

	// Parse the compound action
	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	if len(compound.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(compound.Components))
	}

	// Should contain a MainActionComponent wrapping UseCultSpadeAction
	mainComp, ok := compound.Components[0].(*MainActionComponent)
	if !ok {
		t.Fatalf("expected MainActionComponent, got %T", compound.Components[0])
	}

	// Verify it's UseCultSpadeAction
	if mainComp.Action.GetType() != game.ActionUseCultSpade {
		t.Errorf("expected ActionUseCultSpade, got %v", mainComp.Action.GetType())
	}

	// Verify it's the correct action type
	cultSpadeAction, ok := mainComp.Action.(*game.UseCultSpadeAction)
	if !ok {
		t.Fatalf("expected *UseCultSpadeAction, got %T", mainComp.Action)
	}

	// Verify target hex is correct (G2)
	expectedHex, _ := ConvertLogCoordToAxial("G2")
	if cultSpadeAction.TargetHex != expectedHex {
		t.Errorf("expected hex %v, got %v", expectedHex, cultSpadeAction.TargetHex)
	}

	t.Logf("✓ Transform with pending cult spade correctly parsed as UseCultSpadeAction")
}

// TestParseTransformOnly_WithoutCultSpade tests that transform-only actions
// use TransformTerrainComponent when player has NO pending cult spades
func TestParseTransformOnly_WithoutCultSpade(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("Darklings", &factions.Darklings{})

	// NO cult spades
	gs.PendingSpades = make(map[string]int)

	entry := &LogEntry{
		Faction: models.FactionDarklings,
		Action:  "dig 1. transform H4 to green",
	}

	// Parse the compound action
	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have 1 component: TransformTerrainComponent
	// Note: "dig 1" is just notation and doesn't create a GrantSpadesComponent anymore
	if len(compound.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(compound.Components))
	}

	// Component should be TransformTerrainComponent
	transformComp, ok := compound.Components[0].(*TransformTerrainComponent)
	if !ok {
		t.Fatalf("expected component to be *TransformTerrainComponent, got %T", compound.Components[0])
	}

	// Verify target hex is correct (H4)
	expectedHex, _ := ConvertLogCoordToAxial("H4")
	if transformComp.TargetHex != expectedHex {
		t.Errorf("expected hex %v, got %v", expectedHex, transformComp.TargetHex)
	}

	// Verify target terrain is green (Forest)
	expectedTerrain, _ := ParseTerrainColor("green")
	if transformComp.TargetTerrain != expectedTerrain {
		t.Errorf("expected terrain %v, got %v", expectedTerrain, transformComp.TargetTerrain)
	}

	t.Logf("✓ Transform without cult spade correctly parsed as TransformTerrainComponent")
}

// TestCultSpadeUsage_FullFlow tests the complete flow:
// 1. Player reaches cult milestone and receives spade
// 2. Transform action uses the cult spade
// 3. Spade is deducted from pending count
func TestCultSpadeUsage_FullFlow(t *testing.T) {
	gs := game.NewGameState()
	cultists := &factions.Cultists{}
	gs.AddPlayer("Cultists", cultists)

	player := gs.GetPlayer("Cultists")
	player.Resources.Workers = 0 // No workers!

	// Grant cult reward spade
	gs.PendingSpades = make(map[string]int)
	gs.PendingSpades["Cultists"] = 1

	// Place a dwelling for adjacency (at hex with Q=0, R=0)
	hex1 := game.Hex{Q: 0, R: 0}
	gs.Map.GetHex(hex1).Terrain = cultists.GetHomeTerrain()
	gs.Map.GetHex(hex1).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionCultists,
		PlayerID:   "Cultists",
		PowerValue: 1,
	}

	// Transform at adjacent hex (Q=1, R=0)
	// Set it to Swamp (1 spade away from Plains/Cultists home)
	hex2 := game.Hex{Q: 1, R: 0}
	gs.Map.GetHex(hex2).Terrain = models.TerrainSwamp

	// Create UseCultSpadeAction directly since we know player has pending spades
	action := game.NewUseCultSpadeAction("Cultists", hex2)

	// Execute the action
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("action.Execute() error = %v", err)
	}

	// Verify spade was used
	if gs.PendingSpades["Cultists"] != 0 {
		t.Errorf("expected 0 pending spades, got %d", gs.PendingSpades["Cultists"])
	}

	// Verify terrain was transformed BY 1 spade (Swamp -> Plains)
	// Since Swamp is 1 spade away from Plains (home), cult spade should transform all the way
	if gs.Map.GetHex(hex2).Terrain != cultists.GetHomeTerrain() {
		t.Errorf("terrain should be Plains (home), got %v", gs.Map.GetHex(hex2).Terrain)
	}

	// Verify no workers were spent (still 0)
	if player.Resources.Workers != 0 {
		t.Errorf("workers should still be 0, got %d", player.Resources.Workers)
	}

	t.Logf("✓ Cult spade used successfully: transform completed with 0 workers")
}
