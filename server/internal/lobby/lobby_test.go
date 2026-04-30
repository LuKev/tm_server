package lobby

import (
	"errors"
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

func TestManager_CreateJoinLeave_OpenSeatRestriction(t *testing.T) {
	manager := NewManager()

	first, err := manager.CreateGame("First", 3, "host", "", nil, false, false, "off")
	if err != nil {
		t.Fatalf("create first game: %v", err)
	}
	if len(first.Players) != 1 || first.Players[0] != "host" {
		t.Fatalf("expected creator to be auto-seated, got %+v", first.Players)
	}

	if _, err := manager.CreateGame("Second", 3, "host", "", nil, false, false, "off"); !errors.Is(err, ErrAlreadyInOpenGame) {
		t.Fatalf("expected ErrAlreadyInOpenGame for duplicate create, got %v", err)
	}

	second, err := manager.CreateGame("Second", 3, "other-host", "", nil, false, false, "off")
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

	meta, err := manager.CreateGame("Table", 2, "host", "", nil, false, false, "off")
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if err := manager.JoinGame(meta.ID, "guest"); err != nil {
		t.Fatalf("join game: %v", err)
	}
	if err := manager.StartGame(meta.ID); err != nil {
		t.Fatalf("start game: %v", err)
	}

	listedGames := manager.ListGames()
	if len(listedGames) != 1 {
		t.Fatalf("expected started game to remain visible in lobby list, got %d entries", len(listedGames))
	}
	if !listedGames[0].Started {
		t.Fatalf("expected listed game to be marked started")
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

	meta, err := manager.CreateGame("Archipelago Table", 3, "host", string(board.MapArchipelago), nil, false, false, "off")
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if meta.MapID != string(board.MapArchipelago) {
		t.Fatalf("expected map id %q, got %q", board.MapArchipelago, meta.MapID)
	}
}

func TestManager_CreateGame_InvalidMapRejected(t *testing.T) {
	manager := NewManager()

	if _, err := manager.CreateGame("Bad Table", 3, "host", "unknown-map", nil, false, false, "off"); !errors.Is(err, ErrInvalidMap) {
		t.Fatalf("expected ErrInvalidMap, got %v", err)
	}
}

func TestManager_CreateGame_StoresCustomMap(t *testing.T) {
	manager := NewManager()

	custom := &board.CustomMapDefinition{
		Name:            "Tiny",
		RowCount:        2,
		FirstRowColumns: 3,
		FirstRowLonger:  true,
		Rows: [][]models.TerrainType{
			{models.TerrainPlains, models.TerrainRiver, models.TerrainForest},
			{models.TerrainLake, models.TerrainDesert},
		},
	}

	meta, err := manager.CreateGame("Custom Table", 3, "host", string(board.MapCustom), custom, false, false, "off")
	if err != nil {
		t.Fatalf("create custom game: %v", err)
	}
	if meta.MapID != string(board.MapCustom) {
		t.Fatalf("expected custom map id, got %q", meta.MapID)
	}
	if meta.CustomMap == nil {
		t.Fatalf("expected custom map to be stored")
	}
	if meta.CustomMap.Name != "Tiny" {
		t.Fatalf("expected custom name Tiny, got %q", meta.CustomMap.Name)
	}
}

func TestManager_CreateGame_StoresEnableFanFactions(t *testing.T) {
	manager := NewManager()

	meta, err := manager.CreateGame("Fan Table", 3, "host", "", nil, true, false, "off")
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if !meta.EnableFanFactions {
		t.Fatalf("expected enableFanFactions to be true")
	}
}

func TestManager_CreateGame_StoresEnableFireIceFactions(t *testing.T) {
	manager := NewManager()

	meta, err := manager.CreateGame("FireIce Table", 3, "host", "", nil, false, true, "off")
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if !meta.EnableFireIceFactions {
		t.Fatalf("expected enableFireIceFactions to be true")
	}
}

func TestManager_CreateGame_StoresFireIceScoring(t *testing.T) {
	manager := NewManager()

	meta, err := manager.CreateGame("FireIce Table", 3, "host", "", nil, false, false, "random")
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if meta.FireIceScoring != "random" {
		t.Fatalf("expected fireIceScoring=random, got %q", meta.FireIceScoring)
	}
}
