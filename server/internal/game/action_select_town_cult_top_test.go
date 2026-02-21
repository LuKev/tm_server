package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestTownCultTopChoice_CreatedAndResolved(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	player := gs.GetPlayer("player1")
	if player == nil {
		t.Fatalf("player should exist")
	}

	player.Keys = 0
	player.CultPositions[CultFire] = 9
	player.CultPositions[CultWater] = 9
	player.CultPositions[CultEarth] = 9
	player.CultPositions[CultAir] = 9
	gs.CultTracks.PlayerPositions[player.ID][CultFire] = 9
	gs.CultTracks.PlayerPositions[player.ID][CultWater] = 9
	gs.CultTracks.PlayerPositions[player.ID][CultEarth] = 9
	gs.CultTracks.PlayerPositions[player.ID][CultAir] = 9

	// TW8 gives one key and +1 on all tracks, which requires choosing one top.
	gs.applyTownTileSpecifics(player, models.TownTile8Points, false)
	if gs.PendingTownCultTopChoice == nil {
		t.Fatalf("expected pending town cult-top choice")
	}
	if gs.PendingTownCultTopChoice.MaxSelections != 1 {
		t.Fatalf("expected max selections 1, got %d", gs.PendingTownCultTopChoice.MaxSelections)
	}

	// No top is applied until the pending choice is resolved.
	if got := gs.CultTracks.GetPosition(player.ID, CultFire); got != 9 {
		t.Fatalf("expected fire 9 before choice resolution, got %d", got)
	}

	action := &SelectTownCultTopAction{
		BaseAction: BaseAction{Type: ActionSelectTownCultTop, PlayerID: player.ID},
		Tracks:     []CultTrack{CultWater},
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("expected town cult-top resolution to succeed, got: %v", err)
	}

	if gs.PendingTownCultTopChoice != nil {
		t.Fatalf("expected pending town cult-top choice to be cleared")
	}
	if got := gs.CultTracks.GetPosition(player.ID, CultWater); got != 10 {
		t.Fatalf("expected water to top at 10, got %d", got)
	}
	if got := gs.CultTracks.GetPosition(player.ID, CultFire); got != 9 {
		t.Fatalf("expected fire to remain 9, got %d", got)
	}
}

func TestTownCultTopChoice_ValidateTrackCount(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	player := gs.GetPlayer("player1")
	if player == nil {
		t.Fatalf("player should exist")
	}

	player.Keys = 0
	player.CultPositions[CultFire] = 9
	player.CultPositions[CultWater] = 9
	gs.CultTracks.PlayerPositions[player.ID][CultFire] = 9
	gs.CultTracks.PlayerPositions[player.ID][CultWater] = 9

	gs.applyTownCultBonusWithPotentialTopChoice(player, 1)
	if gs.PendingTownCultTopChoice == nil {
		t.Fatalf("expected pending town cult-top choice")
	}

	action := &SelectTownCultTopAction{
		BaseAction: BaseAction{Type: ActionSelectTownCultTop, PlayerID: player.ID},
		Tracks:     []CultTrack{CultFire}, // must select exactly zero for this setup
	}
	if err := action.Validate(gs); err == nil {
		t.Fatalf("expected validation error for wrong selected track count")
	}
}
