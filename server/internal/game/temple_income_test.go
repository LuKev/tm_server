package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// TestTempleIncome_WithPriestsOnCultTracks tests that temples provide +1 priest income
// even when player has priests on cult track action spaces (Bug #34 regression test)
// The bug was that GetTotalPriestsOnCultTracks() was summing cult track positions instead
// of counting priests on action spaces, incorrectly blocking temple income.
func TestTempleIncome_WithPriestsOnCultTracks(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Build a temple on the map
	templeHex := NewHex(0, 0)
	gs.Map.GetHex(templeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(templeHex).Building = &models.Building{
		Type:       models.BuildingTemple,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}

	// Player has priests on cult track action spaces (placed via 2/3-step advancement)
	// This should NOT block temple income
	gs.CultTracks.PriestsOnActionSpaces["player1"][CultFire] = 2
	gs.CultTracks.PriestsOnActionSpaces["player1"][CultEarth] = 2

	// Grant income
	initialPriests := player.Resources.Priests
	gs.GrantIncome()

	// Should still get +1 priest from temple (not blocked by priests on action spaces)
	priestsGained := player.Resources.Priests - initialPriests
	if priestsGained != 1 {
		t.Errorf("expected +1 priest from temple income (not blocked by 4 priests on action spaces), got %d", priestsGained)
	}
}

// TestTempleIncome tests that temples provide +1 priest income
func TestTempleIncome(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Build a temple on the map
	templeHex := NewHex(0, 0)
	gs.Map.GetHex(templeHex).Terrain = faction.GetHomeTerrain()
	gs.Map.GetHex(templeHex).Building = &models.Building{
		Type:       models.BuildingTemple,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 2,
	}

	// Grant income
	initialPriests := player.Resources.Priests
	gs.GrantIncome()

	// Should get +1 priest from temple
	priestsGained := player.Resources.Priests - initialPriests
	if priestsGained != 1 {
		t.Errorf("expected +1 priest from temple income, got %d", priestsGained)
	}
}

// TestMultipleTemplesIncome tests that multiple temples provide priests
func TestMultipleTemplesIncome(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewCultists()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Build 2 temples on the map
	for i := 0; i < 2; i++ {
		templeHex := NewHex(i, 0)
		gs.Map.GetHex(templeHex).Terrain = faction.GetHomeTerrain()
		gs.Map.GetHex(templeHex).Building = &models.Building{
			Type:       models.BuildingTemple,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		}
	}

	// Grant income
	initialPriests := player.Resources.Priests
	gs.GrantIncome()

	// Should get +2 priests from 2 temples
	priestsGained := player.Resources.Priests - initialPriests
	if priestsGained != 2 {
		t.Errorf("expected +2 priests from 2 temples, got %d", priestsGained)
	}
}

// TestEngineersTempleIncome tests Engineers' special temple income
// (1st and 3rd temples: +1 priest, 2nd temple: +5 power instead)
func TestEngineersTempleIncome(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Build 2 temples
	for i := 0; i < 2; i++ {
		templeHex := NewHex(i, 0)
		gs.Map.GetHex(templeHex).Terrain = faction.GetHomeTerrain()
		gs.Map.GetHex(templeHex).Building = &models.Building{
			Type:       models.BuildingTemple,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		}
	}

	// Grant income
	initialPriests := player.Resources.Priests
	initialPower := player.Resources.Power.Bowl1 + player.Resources.Power.Bowl2 + player.Resources.Power.Bowl3
	gs.GrantIncome()

	// Engineers: 1st temple gives +1 priest, 2nd temple gives +5 power (no priest)
	priestsGained := player.Resources.Priests - initialPriests
	if priestsGained != 1 {
		t.Errorf("expected +1 priest from Engineers' 1st temple (2nd temple gives power), got %d", priestsGained)
	}

	// Check power gain (2nd temple should give +5 power)
	finalPower := player.Resources.Power.Bowl1 + player.Resources.Power.Bowl2 + player.Resources.Power.Bowl3
	powerGained := finalPower - initialPower
	if powerGained != 5 {
		t.Errorf("expected +5 power from Engineers' 2nd temple, got %d", powerGained)
	}
}
