package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
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
	templeHex := board.NewHex(0, 0)
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
	templeHex := board.NewHex(0, 0)
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
		templeHex := board.NewHex(i, 0)
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
		templeHex := board.NewHex(i, 0)
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

	// Engineers: 1st temple gives +1 priest, 2nd temple gives +5 power (no priest)
	priestsGained := player.Resources.Priests - initialPriests
	if priestsGained != 1 {
		t.Errorf("expected +1 priest from Engineers' 1st temple (2nd temple gives power), got %d", priestsGained)
	}

	// Check power gain (2nd temple should give +5 power)
	// Check power gain (2nd temple should give +5 power)
	// Note: GainPower cycles power, it doesn't increase total tokens
	// Engineers start with 3/9/0. Gain 5:
	// 3 from Bowl1 -> Bowl2 (Bowl1=0, Bowl2=12)
	// 2 from Bowl2 -> Bowl3 (Bowl2=10, Bowl3=2)
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected 2 power in Bowl 3 after gaining 5 power, got %d", player.Resources.Power.Bowl3)
	}
	if player.Resources.Power.Bowl1 != 0 {
		t.Errorf("expected 0 power in Bowl 1, got %d", player.Resources.Power.Bowl1)
	}
}
