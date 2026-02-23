package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestSelectFaction_SetsStartingCultPositions(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", nil); err != nil {
		t.Fatalf("add p1: %v", err)
	}
	if err := gs.AddPlayer("p2", nil); err != nil {
		t.Fatalf("add p2: %v", err)
	}
	gs.Phase = PhaseFactionSelection
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 0

	action := &SelectFactionAction{
		PlayerID:    "p1",
		FactionType: models.FactionWitches,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("execute select faction: %v", err)
	}

	player := gs.GetPlayer("p1")
	if player == nil {
		t.Fatalf("missing player p1")
	}
	if got := player.CultPositions[CultFire]; got != 0 {
		t.Fatalf("fire cult mismatch: got %d want 0", got)
	}
	if got := player.CultPositions[CultWater]; got != 0 {
		t.Fatalf("water cult mismatch: got %d want 0", got)
	}
	if got := player.CultPositions[CultEarth]; got != 0 {
		t.Fatalf("earth cult mismatch: got %d want 0", got)
	}
	if got := player.CultPositions[CultAir]; got != 2 {
		t.Fatalf("air cult mismatch: got %d want 2", got)
	}

	if got := gs.CultTracks.GetPosition("p1", CultAir); got != 2 {
		t.Fatalf("cult track state air mismatch: got %d want 2", got)
	}
}
