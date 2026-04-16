package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
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

func TestCreateGameWithOptions_SerializesFanFactionToggle(t *testing.T) {
	manager := NewManager()
	if err := manager.CreateGameWithOptions("g1", []string{"p1", "p2"}, CreateGameOptions{
		RandomizeTurnOrder: false,
		SetupMode:          SetupModeSnellman,
		EnableFanFactions:  true,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}

	state := manager.SerializeGameState("g1")
	if got := state["enableFanFactions"]; got != true {
		t.Fatalf("enableFanFactions: got %v, want true", got)
	}
}

func TestSerializeStateWithRevision_IncludesMapDisplayCoordinates(t *testing.T) {
	manager := NewManager()
	if err := manager.CreateGameWithOptions("g1", []string{"p1", "p2"}, CreateGameOptions{
		RandomizeTurnOrder: false,
		SetupMode:          SetupModeSnellman,
		MapID:              board.MapLakes,
	}); err != nil {
		t.Fatalf("create game: %v", err)
	}

	state := manager.SerializeGameState("g1")
	mapRaw, ok := state["map"].(map[string]interface{})
	if !ok {
		t.Fatalf("serialized map missing")
	}
	hexesRaw, ok := mapRaw["hexes"].(map[string]interface{})
	if !ok {
		t.Fatalf("serialized hexes missing")
	}

	b1Raw, ok := hexesRaw["-1,1"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected serialized Lakes B1 at -1,1")
	}
	if got := b1Raw["displayCoord"]; got != "B1" {
		t.Fatalf("serialized Lakes B1 display coord: got %v, want %q", got, "B1")
	}

	b2Raw, ok := hexesRaw["0,1"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected serialized Lakes B2 at 0,1")
	}
	if got := b2Raw["displayCoord"]; got != "B2" {
		t.Fatalf("serialized Lakes B2 display coord: got %v, want %q", got, "B2")
	}
}

func TestCreateGameWithOptions_SerializesCustomMapDisplayCoordinates(t *testing.T) {
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

	if err := manager.CreateGameWithOptions("g1", []string{"p1", "p2"}, CreateGameOptions{
		RandomizeTurnOrder: false,
		SetupMode:          SetupModeSnellman,
		MapID:              board.MapCustom,
		CustomMap:          custom,
	}); err != nil {
		t.Fatalf("create custom game: %v", err)
	}

	state := manager.SerializeGameState("g1")
	if got := state["mapId"]; got != string(board.MapCustom) {
		t.Fatalf("top-level mapId: got %v, want %q", got, board.MapCustom)
	}

	mapRaw, ok := state["map"].(map[string]interface{})
	if !ok {
		t.Fatalf("serialized map missing")
	}
	hexesRaw, ok := mapRaw["hexes"].(map[string]interface{})
	if !ok {
		t.Fatalf("serialized hexes missing")
	}

	a1Raw, ok := hexesRaw["0,0"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected custom A1 at 0,0")
	}
	if got := a1Raw["displayCoord"]; got != "A1" {
		t.Fatalf("custom A1 display coord: got %v, want %q", got, "A1")
	}

	b1Raw, ok := hexesRaw["0,1"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected custom B1 at 0,1")
	}
	if got := b1Raw["displayCoord"]; got != "B1" {
		t.Fatalf("custom B1 display coord: got %v, want %q", got, "B1")
	}
}
