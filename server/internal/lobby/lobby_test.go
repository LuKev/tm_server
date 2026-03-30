package lobby

import (
	"errors"
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
)

func TestManager_CreateJoinLeave_OpenSeatRestriction(t *testing.T) {
	manager := NewManager()

	first, err := manager.CreateGame("First", 3, "host", "")
	if err != nil {
		t.Fatalf("create first game: %v", err)
	}
	if len(first.Players) != 1 || first.Players[0] != "host" {
		t.Fatalf("expected creator to be auto-seated, got %+v", first.Players)
	}

	if _, err := manager.CreateGame("Second", 3, "host", ""); !errors.Is(err, ErrAlreadyInOpenGame) {
		t.Fatalf("expected ErrAlreadyInOpenGame for duplicate create, got %v", err)
	}

	second, err := manager.CreateGame("Second", 3, "other-host", "")
	if err != nil {
		t.Fatalf("create second game: %v", err)
	}

	if err := manager.JoinGame(second.ID, "host"); !errors.Is(err, ErrAlreadyInOpenGame) {
		t.Fatalf("expected ErrAlreadyInOpenGame for cross-game join, got %v", err)
	}

	if err := manager.LeaveGame(first.ID, "host"); err != nil {
		t.Fatalf("leave first game: %v", err)
	}

	if err := manager.JoinGame(second.ID, "host"); err != nil {
		t.Fatalf("join second after leaving first: %v", err)
	}

	openGames := manager.ListGames()
	if len(openGames) != 1 {
		t.Fatalf("expected one open game after host left first game, got %d", len(openGames))
	}
	if openGames[0].ID != second.ID {
		t.Fatalf("expected remaining open game to be second, got %s", openGames[0].ID)
	}
}

func TestManager_StartGame_RemovesGameFromOpenListAndBlocksLeave(t *testing.T) {
	manager := NewManager()

	meta, err := manager.CreateGame("Table", 2, "host", "")
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if err := manager.JoinGame(meta.ID, "guest"); err != nil {
		t.Fatalf("join game: %v", err)
	}
	if err := manager.StartGame(meta.ID); err != nil {
		t.Fatalf("start game: %v", err)
	}

	if openGames := manager.ListGames(); len(openGames) != 0 {
		t.Fatalf("expected started games to disappear from open list, got %d entries", len(openGames))
	}
	if err := manager.LeaveGame(meta.ID, "host"); !errors.Is(err, ErrGameAlreadyStarted) {
		t.Fatalf("expected ErrGameAlreadyStarted after start, got %v", err)
	}

	stored, ok := manager.GetGame(meta.ID)
	if !ok {
		t.Fatalf("expected started game metadata to remain for reconnects")
	}
	if !stored.Started {
		t.Fatalf("expected started flag to be preserved")
	}
}

func TestManager_CreateGame_StoresSelectedMap(t *testing.T) {
	manager := NewManager()

	meta, err := manager.CreateGame("Archipelago Table", 3, "host", string(board.MapArchipelago))
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if meta.MapID != string(board.MapArchipelago) {
		t.Fatalf("expected map id %q, got %q", board.MapArchipelago, meta.MapID)
	}
}

func TestManager_CreateGame_InvalidMapRejected(t *testing.T) {
	manager := NewManager()

	if _, err := manager.CreateGame("Bad Table", 3, "host", "unknown-map"); !errors.Is(err, ErrInvalidMap) {
		t.Fatalf("expected ErrInvalidMap, got %v", err)
	}
}
