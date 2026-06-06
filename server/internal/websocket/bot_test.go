package websocket

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

func TestPrepareModelGame_AppliesFixedFactionsAndBotOptions(t *testing.T) {
	games := game.NewManager()
	gameID := "1"
	humanID := "human"
	botID := modelBotPlayerID(gameID)
	if err := games.CreateGameWithOptions(gameID, []string{humanID, botID}, game.CreateGameOptions{
		RandomizeTurnOrder: false,
		SetupMode:          game.SetupModeSnellman,
		MapID:              board.MapBase,
	}); err != nil {
		t.Fatalf("CreateGameWithOptions failed: %v", err)
	}

	client := &Client{deps: ServerDeps{Games: games}}
	if err := client.prepareModelGame(gameID, humanID, BotGameConfig{
		PlayerID: botID,
		Faction:  models.FactionWitches,
	}, models.FactionNomads); err != nil {
		t.Fatalf("prepareModelGame failed: %v", err)
	}

	gs, ok := games.GetGame(gameID)
	if !ok || gs == nil {
		t.Fatalf("game missing after prepare")
	}
	if gs.Phase != game.PhaseSetup {
		t.Fatalf("phase = %v, want setup", gs.Phase)
	}
	if got := gs.GetPlayer(humanID).Faction.GetType(); got != models.FactionNomads {
		t.Fatalf("human faction = %v, want Nomads", got)
	}
	if got := gs.GetPlayer(botID).Faction.GetType(); got != models.FactionWitches {
		t.Fatalf("bot faction = %v, want Witches", got)
	}
	if gs.GetPlayer(botID).Options.ConfirmActions {
		t.Fatalf("bot turn confirmations should be disabled")
	}
}
