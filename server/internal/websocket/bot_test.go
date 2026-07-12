package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/lobby"
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

func TestBotCanAct_WaitsForHumanTurnConfirmation(t *testing.T) {
	gs := game.NewGameState()
	gs.PendingTurnConfirmationPlayerID = "human"
	gs.PendingTurnConfirmationSnapshot = game.NewGameState()

	if botCanAct(gs, "model") {
		t.Fatal("bot should wait while the human owns the turn confirmation window")
	}
	if !botCanAct(gs, "human") {
		t.Fatal("pending player should remain eligible to resolve its confirmation window")
	}

	gs.ClearPendingTurnConfirmation()
	if !botCanAct(gs, "model") {
		t.Fatal("bot should act after the confirmation window is cleared")
	}
}

func TestHandleCreateAndStartModelGame_StartsPlayableGame(t *testing.T) {
	games := game.NewManager()
	lobbies := lobby.NewManager()
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:         hub,
		send:        make(chan []byte, 16),
		deps:        ServerDeps{Games: games, Lobby: lobbies},
		seatsByGame: make(map[string]string),
	}
	hub.register <- client
	defer func() {
		hub.unregister <- client
	}()

	payload, err := json.Marshal(createGamePayload{
		Name:       "Human vs Model",
		Creator:    "human",
		MapID:      string(board.MapBase),
		MaxPlayers: 2,
		ModelOpponent: &modelOpponentPayload{
			Enabled:      true,
			HumanFaction: int(models.FactionNomads),
			BotFaction:   int(models.FactionWitches),
			Simulations:  1,
			MaxDepth:     20,
		},
	})
	if err != nil {
		t.Fatalf("Marshal payload failed: %v", err)
	}

	client.handleCreateAndStartModelGame(payload)

	var started bool
	deadline := time.After(2 * time.Second)
	for !started {
		select {
		case raw := <-client.send:
			var msg struct {
				Type    string            `json:"type"`
				Payload map[string]string `json:"payload"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			if msg.Type == "model_game_started" {
				started = true
				if msg.Payload["gameId"] != "1" {
					t.Fatalf("gameId = %q, want 1", msg.Payload["gameId"])
				}
				if msg.Payload["playerId"] != "human" {
					t.Fatalf("playerId = %q, want human", msg.Payload["playerId"])
				}
			}
		case <-deadline:
			t.Fatal("timed out waiting for model_game_started")
		}
	}

	meta, ok := lobbies.GetGame("1")
	if !ok {
		t.Fatal("lobby game missing")
	}
	if !meta.Started {
		t.Fatal("lobby game was not marked started")
	}
	if len(meta.Players) != 2 || meta.Players[0] != "human" || meta.Players[1] != modelBotPlayerID("1") {
		t.Fatalf("players = %#v, want human and model", meta.Players)
	}

	gs, ok := games.GetGame("1")
	if !ok || gs == nil {
		t.Fatal("game missing")
	}
	if gs.Phase != game.PhaseSetup {
		t.Fatalf("phase = %v, want setup", gs.Phase)
	}
	if got := gs.GetPlayer("human").Faction.GetType(); got != models.FactionNomads {
		t.Fatalf("human faction = %v, want Nomads", got)
	}
	if got := gs.GetPlayer(modelBotPlayerID("1")).Faction.GetType(); got != models.FactionWitches {
		t.Fatalf("model faction = %v, want Witches", got)
	}
}
