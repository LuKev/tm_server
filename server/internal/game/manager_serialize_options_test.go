package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
)

func TestSerializeStateWithRevision_IncludesPlayerOptions(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", nil); err != nil {
		t.Fatalf("add player: %v", err)
	}

	state := SerializeStateWithRevision(gs, "g1", 0)
	playersRaw, ok := state["players"].(map[string]interface{})
	if !ok {
		t.Fatalf("players missing from serialized state")
	}
	playerRaw, ok := playersRaw["p1"].(map[string]interface{})
	if !ok {
		t.Fatalf("player p1 missing from serialized state")
	}
	optionsRaw, ok := playerRaw["options"].(PlayerOptions)
	if !ok {
		t.Fatalf("player options missing from serialized state")
	}
	if optionsRaw.AutoLeechMode != LeechAutoModeOff {
		t.Fatalf("unexpected auto leech mode: got %q", optionsRaw.AutoLeechMode)
	}
	if optionsRaw.AutoConvertOnPass {
		t.Fatalf("expected auto convert on pass to default false")
	}
	if !optionsRaw.ConfirmActions {
		t.Fatalf("expected confirm actions to default true")
	}
	if optionsRaw.ShowIncomePreview {
		t.Fatalf("expected show income preview to default false")
	}
}

func TestCreateGameWithOptions_UsesSelectedMapAndSerializesMapID(t *testing.T) {
	manager := NewManager()
	if err := manager.CreateGameWithOptions("g1", []string{"p1", "p2"}, CreateGameOptions{
		RandomizeTurnOrder: false,
		SetupMode:          SetupModeSnellman,
		MapID:              board.MapArchipelago,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}

	state := manager.SerializeGameState("g1")
	if got := state["mapId"]; got != string(board.MapArchipelago) {
		t.Fatalf("top-level mapId: got %v, want %q", got, board.MapArchipelago)
	}

	mapRaw, ok := state["map"].(map[string]interface{})
	if !ok {
		t.Fatalf("serialized map missing")
	}
	if got := mapRaw["id"]; got != string(board.MapArchipelago) {
		t.Fatalf("map.id: got %v, want %q", got, board.MapArchipelago)
	}
}
