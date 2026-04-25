// Package websocket handles websocket connections and messaging.
package websocket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/lobby"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	conn *websocket.Conn
	send chan []byte
	id   string

	deps ServerDeps

	seatsByGame map[string]string
}

type inboundMsg struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type createGamePayload struct {
	Name              string                     `json:"name"`
	MaxPlayers        int                        `json:"maxPlayers"`
	Creator           string                     `json:"creator"`
	MapID             string                     `json:"mapId,omitempty"`
	EnableFanFactions bool                       `json:"enableFanFactions,omitempty"`
	FireIceScoring    string                     `json:"fireIceScoring,omitempty"`
	CustomMap         *board.CustomMapDefinition `json:"customMap,omitempty"`
}

type joinGamePayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type leaveGamePayload struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type startGamePayload struct {
	GameID             string `json:"gameID"`
	RandomizeTurnOrder *bool  `json:"randomizeTurnOrder,omitempty"`
	SetupMode          string `json:"setupMode,omitempty"`
	TurnTimerEnabled   *bool  `json:"turnTimerEnabled,omitempty"`
	TurnTimerSeconds   *int   `json:"turnTimerSeconds,omitempty"`
	TurnTimerIncrement *int   `json:"turnTimerIncrementSeconds,omitempty"`
}

type lobbyStateMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type hexParam struct {
	Q int `json:"q"`
	R int `json:"r"`
}

type performActionPayload struct {
	Type             string          `json:"type"`
	GameID           string          `json:"gameID,omitempty"`
	GameId           string          `json:"gameId,omitempty"`
	ActionID         string          `json:"actionId,omitempty"`
	ExpectedRevision *int            `json:"expectedRevision,omitempty"`
	PlayerID         string          `json:"playerID,omitempty"`
	PlayerId         string          `json:"playerId,omitempty"`
	Params           json.RawMessage `json:"params,omitempty"`

	// Legacy fields supported for backward compatibility.
	Faction string    `json:"faction,omitempty"`
	Hex     *hexParam `json:"hex,omitempty"`
}

type nestedActionPayload struct {
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params,omitempty"`
}

type testApplyFixtureSettingsPayload struct {
	GameID          string   `json:"gameID"`
	ScoringTiles    []string `json:"scoringTiles"`
	BonusCards      []string `json:"bonusCards"`
	TurnOrderPolicy string   `json:"turnOrderPolicy,omitempty"`
}

type testReplayActionPayload struct {
	PlayerID string          `json:"playerId"`
	Type     string          `json:"type"`
	Params   json.RawMessage `json:"params,omitempty"`
}

type testConversionPayload struct {
	ConversionType string `json:"conversionType"`
	Amount         int    `json:"amount"`
}

type testApplyConversionPayload struct {
	GameID         string `json:"gameID"`
	PlayerID       string `json:"playerId"`
	ConversionType string `json:"conversionType"`
	Amount         int    `json:"amount"`
}

type testReplayActionsToIndexPayload struct {
	GameID       string                    `json:"gameID"`
	EndExclusive int                       `json:"endExclusive"`
	Actions      []testReplayActionPayload `json:"actions"`
}

func (c *Client) sendLobbyState() {
	games := c.deps.Lobby.ListGames()
	out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
	c.send <- out
}

func (c *Client) broadcastLobbyState() {
	games := c.deps.Lobby.ListGames()
	out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
	c.hub.BroadcastMessage(out)
}

func (c *Client) sendAvailableMaps() {
	out, _ := json.Marshal(lobbyStateMsg{Type: "available_maps", Payload: board.AvailableMaps()})
	c.send <- out
}

func (c *Client) bindSeat(gameID, playerID string) {
	if c.seatsByGame == nil {
		c.seatsByGame = make(map[string]string)
	}
	c.seatsByGame[gameID] = playerID
}

func (c *Client) seatForGame(gameID string) string {
	if c.seatsByGame == nil {
		return ""
	}
	return c.seatsByGame[gameID]
}

func (c *Client) unbindSeat(gameID string) {
	if c.seatsByGame == nil {
		return
	}
	delete(c.seatsByGame, gameID)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.ReplaceAll(message, newline, space))

		var env inboundMsg
		if err := json.Unmarshal(message, &env); err != nil {
			log.Printf("Received non-JSON message from %s: %s", c.id, string(message))
			continue
		}

		c.handleInboundMessage(env)
	}
}

func (c *Client) handleInboundMessage(env inboundMsg) {
	switch env.Type {
	case "list_games":
		c.sendLobbyState()
		c.sendAvailableMaps()

	case "get_game_state":
		var p struct {
			GameID   string `json:"gameID"`
			PlayerID string `json:"playerID,omitempty"`
		}
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			log.Printf("error parsing get_game_state payload: %v", err)
			return
		}
		if c.seatForGame(p.GameID) == "" {
			if p.PlayerID != "" {
				if meta, ok := c.deps.Lobby.GetGame(p.GameID); ok {
					for _, playerID := range meta.Players {
						if playerID == p.PlayerID {
							c.bindSeat(p.GameID, p.PlayerID)
							break
						}
					}
				}
			}
			if c.seatForGame(p.GameID) == "" {
				meta, ok := c.deps.Lobby.GetGame(p.GameID)
				if !ok || !meta.Started {
					c.sendError("not_in_game")
					return
				}
			}
		}
		c.hub.JoinGame(c, p.GameID)
		gameState := c.deps.Games.SerializeGameState(p.GameID)
		if gameState != nil {
			gameStateMsg, _ := json.Marshal(map[string]any{
				"type":    "game_state_update",
				"payload": gameState,
			})
			c.send <- gameStateMsg
		}

	case "start_game":
		c.handleStartGame(env.Payload)

	case "create_game":
		c.handleCreateGame(env.Payload)

	case "join_game":
		c.handleJoinGame(env.Payload)

	case "leave_game":
		c.handleLeaveGame(env.Payload)

	case "perform_action":
		c.handlePerformAction(env.Payload)
	case "test_apply_conversion":
		c.handleTestApplyConversion(env.Payload)
	case "test_apply_fixture_settings":
		c.handleTestApplyFixtureSettings(env.Payload)
	case "test_replay_actions_to_index":
		c.handleTestReplayActionsToIndex(env.Payload)

	default:
		log.Printf("Unknown message type: %s", env.Type)
	}
}

func (c *Client) handleTestApplyFixtureSettings(payload json.RawMessage) {
	if os.Getenv("TM_ENABLE_TEST_COMMANDS") != "1" {
		c.sendActionRejected("", "forbidden", "test commands are disabled")
		return
	}

	var p testApplyFixtureSettingsPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		c.sendActionRejected("", "invalid_payload", "invalid fixture settings payload")
		return
	}
	p.GameID = strings.TrimSpace(p.GameID)
	if p.GameID == "" {
		c.sendActionRejected("", "missing_game_id", "missing game id")
		return
	}

	seatID := c.seatForGame(p.GameID)
	if seatID == "" {
		c.sendActionRejected("", "unauthorized", "you are not seated in this game")
		return
	}

	scoringTiles, err := scoringTilesFromCodesForFixture(p.ScoringTiles)
	if err != nil {
		c.sendActionRejected("", "invalid_scoring_tiles", err.Error())
		return
	}
	bonusCards, err := bonusCardsFromCodesForFixture(p.BonusCards)
	if err != nil {
		c.sendActionRejected("", "invalid_bonus_cards", err.Error())
		return
	}
	turnOrderPolicy, err := turnOrderPolicyFromFixturePayload(p.TurnOrderPolicy)
	if err != nil {
		c.sendActionRejected("", "invalid_turn_order_policy", err.Error())
		return
	}

	newRevision, err := c.deps.Games.ApplyFixtureSettings(p.GameID, scoringTiles, bonusCards, turnOrderPolicy)
	if err != nil {
		c.sendActionRejected("", "apply_failed", err.Error())
		return
	}

	gameState := c.deps.Games.SerializeGameState(p.GameID)
	if gameState == nil {
		c.sendActionRejected("", "apply_failed", "failed to serialize game state")
		return
	}
	stateMsg, _ := json.Marshal(map[string]any{
		"type":    "game_state_update",
		"payload": gameState,
	})
	c.hub.BroadcastToGame(p.GameID, stateMsg)

	ack, _ := json.Marshal(map[string]any{
		"type": "test_command_applied",
		"payload": map[string]any{
			"gameID":      p.GameID,
			"newRevision": newRevision,
		},
	})
	c.send <- ack
}

func (c *Client) handleTestApplyConversion(payload json.RawMessage) {
	if os.Getenv("TM_ENABLE_TEST_COMMANDS") != "1" {
		c.sendActionRejected("", "forbidden", "test commands are disabled")
		return
	}

	var p testApplyConversionPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		c.sendActionRejected("", "invalid_payload", "invalid conversion payload")
		return
	}

	gameID := strings.TrimSpace(p.GameID)
	if gameID == "" {
		c.sendActionRejected("", "missing_game_id", "missing game id")
		return
	}

	seatID := c.seatForGame(gameID)
	if seatID == "" {
		c.sendActionRejected("", "unauthorized", "you are not seated in this game")
		return
	}

	playerID := strings.TrimSpace(p.PlayerID)
	if playerID == "" {
		playerID = seatID
	}
	if playerID != seatID {
		// Keep test command usage scoped to the sender's seat to avoid
		// unintended cross-game mutations during automation.
		c.sendActionRejected("", "unauthorized", "player is not your seat")
		return
	}

	if playerID == "" {
		c.sendActionRejected("", "missing_player_id", "missing player id")
		return
	}

	amount := p.Amount
	if amount <= 0 {
		c.sendActionRejected("", "invalid_amount", "amount must be positive")
		return
	}

	conversionType := game.ConversionType(strings.TrimSpace(p.ConversionType))
	if conversionType == "" {
		c.sendActionRejected("", "invalid_conversion_type", "missing conversionType")
		return
	}

	if _, err := c.deps.Games.ApplyConversionWithoutTurnCheck(gameID, playerID, conversionType, amount); err != nil {
		c.sendActionRejected("", "conversion_failed", err.Error())
		return
	}

	gameState := c.deps.Games.SerializeGameState(gameID)
	if gameState == nil {
		c.sendActionRejected("", "test_command_failed", "failed to serialize game state")
		return
	}

	stateMsg, _ := json.Marshal(map[string]any{
		"type":    "game_state_update",
		"payload": gameState,
	})
	c.hub.BroadcastToGame(gameID, stateMsg)

	ack, _ := json.Marshal(map[string]any{
		"type": "test_command_applied",
		"payload": map[string]any{
			"gameID":         gameID,
			"playerId":       playerID,
			"conversionType": conversionType,
			"amount":         amount,
		},
	})
	c.send <- ack
}

func (c *Client) handleTestReplayActionsToIndex(payload json.RawMessage) {
	if os.Getenv("TM_ENABLE_TEST_COMMANDS") != "1" {
		c.sendActionRejected("", "forbidden", "test commands are disabled")
		return
	}

	var p testReplayActionsToIndexPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		c.sendActionRejected("", "invalid_payload", "invalid replay payload")
		return
	}
	p.GameID = strings.TrimSpace(p.GameID)
	if p.GameID == "" {
		c.sendActionRejected("", "missing_game_id", "missing game id")
		return
	}
	if p.EndExclusive < 0 {
		c.sendActionRejected("", "invalid_index", "endExclusive must be >= 0")
		return
	}
	if p.EndExclusive > len(p.Actions) {
		c.sendActionRejected("", "invalid_index", "endExclusive exceeds action list length")
		return
	}

	seatID := c.seatForGame(p.GameID)
	if seatID == "" {
		c.sendActionRejected("", "unauthorized", "you are not seated in this game")
		return
	}

	for i := 0; i < p.EndExclusive; i++ {
		replay := p.Actions[i]
		playerID := strings.TrimSpace(replay.PlayerID)
		if playerID == "" {
			c.sendActionRejected("", "invalid_action", fmt.Sprintf("action %d missing playerId", i))
			return
		}

		if strings.TrimSpace(replay.Type) == "replay_conversion" {
			if err := c.handleTestReplayConversionPayload(p.GameID, playerID, replay.Params); err != nil {
				c.sendActionRejected("", "replay_failed", fmt.Sprintf("action %d execute failed: %v", i, err))
				return
			}
			continue
		}

		req := performActionPayload{
			Type:     strings.TrimSpace(replay.Type),
			GameID:   p.GameID,
			ActionID: fmt.Sprintf("test-replay-%d", i),
			Params:   replay.Params,
		}
		action, err := buildActionFromPayload(req, playerID)
		if err != nil {
			c.sendActionRejected("", "invalid_action", fmt.Sprintf("action %d parse failed: %v", i, err))
			return
		}

		_, err = c.deps.Games.ExecuteActionWithMeta(p.GameID, action, game.ActionMeta{
			ActionID:         req.ActionID,
			ExpectedRevision: -1,
			SeatID:           playerID,
		})
		if err != nil {
			c.sendActionRejected("", "replay_failed", fmt.Sprintf("action %d execute failed: %v", i, err))
			return
		}
	}

	gameState := c.deps.Games.SerializeGameState(p.GameID)
	if gameState == nil {
		c.sendActionRejected("", "replay_failed", "failed to serialize game state")
		return
	}

	stateMsg, _ := json.Marshal(map[string]any{
		"type":    "game_state_update",
		"payload": gameState,
	})
	c.hub.BroadcastToGame(p.GameID, stateMsg)

	ack, _ := json.Marshal(map[string]any{
		"type": "test_command_applied",
		"payload": map[string]any{
			"gameID":       p.GameID,
			"endExclusive": p.EndExclusive,
		},
	})
	c.send <- ack
}

func (c *Client) handleTestReplayConversionPayload(gameID, playerID string, payload json.RawMessage) error {
	var p testConversionPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("invalid conversion payload: %w", err)
	}

	conversionType := game.ConversionType(strings.TrimSpace(p.ConversionType))
	if conversionType == "" {
		return fmt.Errorf("missing conversionType")
	}

	amount := p.Amount
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	if _, err := c.deps.Games.ApplyConversionWithoutTurnCheck(gameID, strings.TrimSpace(playerID), conversionType, amount); err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	return nil
}

func (c *Client) handleStartGame(payload json.RawMessage) {
	var p startGamePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("error parsing start_game payload: %v", err)
		return
	}

	meta, ok := c.deps.Lobby.GetGame(p.GameID)
	if !ok {
		c.sendError("game_not_found")
		return
	}

	if len(meta.Players) < meta.MaxPlayers {
		errorMsg, _ := json.Marshal(map[string]any{
			"type": "error",
			"payload": map[string]any{
				"error":       "game_not_full",
				"playerCount": len(meta.Players),
				"maxPlayers":  meta.MaxPlayers,
			},
		})
		c.send <- errorMsg
		return
	}

	startSeat := c.seatForGame(p.GameID)
	if startSeat == "" {
		c.sendError("not_in_game")
		return
	}

	isMember := false
	for _, playerID := range meta.Players {
		if playerID == startSeat {
			isMember = true
			break
		}
	}
	if !isMember {
		c.sendError("not_in_game")
		return
	}
	if strings.TrimSpace(meta.Host) != "" && startSeat != strings.TrimSpace(meta.Host) {
		c.sendActionRejected("", "host_only", "only the host can start this game")
		return
	}

	randomize := true
	if p.RandomizeTurnOrder != nil {
		randomize = *p.RandomizeTurnOrder
	}
	setupMode := game.SetupModeSnellman
	switch strings.ToLower(strings.TrimSpace(p.SetupMode)) {
	case "", "snellman":
		setupMode = game.SetupModeSnellman
	case "auction":
		setupMode = game.SetupModeAuction
	case "fast_auction", "fast-auction", "fast auction":
		setupMode = game.SetupModeFastAuction
	default:
		c.sendActionRejected("", "invalid_setup_mode", fmt.Sprintf("unsupported setup mode: %s", p.SetupMode))
		return
	}

	var turnTimer *game.TurnTimerConfig
	if p.TurnTimerEnabled != nil && *p.TurnTimerEnabled {
		initialSeconds := 25 * 60
		if p.TurnTimerSeconds != nil {
			initialSeconds = *p.TurnTimerSeconds
		}
		incrementSeconds := 0
		if p.TurnTimerIncrement != nil {
			incrementSeconds = *p.TurnTimerIncrement
		}
		if initialSeconds <= 0 {
			c.sendActionRejected("", "invalid_turn_timer", "turn timer must start above 0 seconds")
			return
		}
		if incrementSeconds < 0 {
			c.sendActionRejected("", "invalid_turn_timer", "turn timer increment cannot be negative")
			return
		}
		turnTimer = &game.TurnTimerConfig{
			InitialTimeMs: int64(initialSeconds) * 1000,
			IncrementMs:   int64(incrementSeconds) * 1000,
		}
	}

	err := c.deps.Games.CreateGameWithOptions(p.GameID, meta.Players, game.CreateGameOptions{
		RandomizeTurnOrder: randomize,
		SetupMode:          setupMode,
		TurnTimer:          turnTimer,
		MapID:              board.NormalizeMapID(meta.MapID),
		EnableFanFactions:  meta.EnableFanFactions,
		FireIceScoring:     game.FireIceFinalScoringSetting(strings.TrimSpace(meta.FireIceScoring)),
		CustomMap:          board.CloneCustomMapDefinition(meta.CustomMap),
	})
	if err != nil && !strings.Contains(err.Error(), "game already exists") {
		log.Printf("error creating game: %v", err)
		c.sendError("create_game_failed")
		return
	}
	if err := c.deps.Lobby.StartGame(p.GameID); err != nil {
		log.Printf("error marking game started: %v", err)
		c.sendError("create_game_failed")
		return
	}

	for _, playerID := range meta.Players {
		if playerID == c.seatForGame(p.GameID) {
			c.hub.JoinGame(c, p.GameID)
		}
	}

	gameState := c.deps.Games.SerializeGameState(p.GameID)
	if gameState != nil {
		gameStateMsg, _ := json.Marshal(map[string]any{
			"type":    "game_state_update",
			"payload": gameState,
		})
		c.hub.BroadcastToGame(p.GameID, gameStateMsg)
	}

	c.broadcastLobbyState()
}

func (c *Client) handleCreateGame(payload json.RawMessage) {
	var p createGamePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("create_game payload error: %v", err)
		return
	}
	if p.MaxPlayers <= 0 {
		p.MaxPlayers = 5
	}
	fireIceScoring := strings.ToLower(strings.TrimSpace(p.FireIceScoring))
	switch fireIceScoring {
	case "", string(game.FireIceFinalScoringOff):
		fireIceScoring = string(game.FireIceFinalScoringOff)
	case string(game.FireIceFinalScoringOn), string(game.FireIceFinalScoringRandom):
	default:
		c.sendActionRejected("", "invalid_fire_ice_scoring", fmt.Sprintf("unsupported Fire & Ice scoring option: %s", p.FireIceScoring))
		return
	}

	meta, err := c.deps.Lobby.CreateGame(
		p.Name,
		p.MaxPlayers,
		p.Creator,
		p.MapID,
		p.CustomMap,
		p.EnableFanFactions,
		fireIceScoring,
	)
	if err != nil {
		c.sendLobbyError(err)
		return
	}
	if p.Creator != "" {
		c.bindSeat(meta.ID, p.Creator)
		c.hub.JoinGame(c, meta.ID)
		createdMsg, _ := json.Marshal(map[string]any{
			"type":    "game_created",
			"payload": map[string]string{"gameId": meta.ID, "playerId": p.Creator},
		})
		c.send <- createdMsg
	}
	c.broadcastLobbyState()
}

func (c *Client) handleJoinGame(payload json.RawMessage) {
	var p joinGamePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("join_game payload error: %v", err)
		return
	}
	if err := c.deps.Lobby.JoinGame(p.ID, p.Name); err != nil {
		c.sendLobbyError(err)
		return
	}

	c.bindSeat(p.ID, p.Name)
	c.hub.JoinGame(c, p.ID)

	successMsg, _ := json.Marshal(map[string]any{
		"type":    "game_joined",
		"payload": map[string]string{"gameId": p.ID, "playerId": p.Name},
	})
	c.send <- successMsg

	c.broadcastLobbyState()
}

func (c *Client) handleLeaveGame(payload json.RawMessage) {
	var p leaveGamePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("leave_game payload error: %v", err)
		return
	}

	playerID := strings.TrimSpace(p.Name)
	if playerID == "" {
		playerID = c.seatForGame(p.ID)
	}
	if playerID == "" {
		c.sendLobbyError(lobby.ErrPlayerNotInGame)
		return
	}

	if err := c.deps.Lobby.LeaveGame(p.ID, playerID); err != nil {
		c.sendLobbyError(err)
		return
	}

	c.unbindSeat(p.ID)
	c.hub.LeaveGame(c, p.ID)

	leftMsg, _ := json.Marshal(map[string]any{
		"type": "game_left",
		"payload": map[string]string{
			"gameId":   p.ID,
			"playerId": playerID,
		},
	})
	c.send <- leftMsg

	c.broadcastLobbyState()
}

func (c *Client) handlePerformAction(payload json.RawMessage) {
	var req performActionPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Printf("perform_action payload error: %v", err)
		c.sendActionRejected("", "invalid_action_payload", "invalid action payload")
		return
	}

	gameID := req.GameID
	if gameID == "" {
		gameID = req.GameId
	}
	if gameID == "" {
		c.sendActionRejected(req.ActionID, "missing_game_id", "missing game id")
		return
	}

	seatID := c.seatForGame(gameID)
	if seatID == "" {
		c.sendActionRejected(req.ActionID, "unauthorized", "you are not seated in this game")
		return
	}

	action, err := buildActionFromPayload(req, seatID)
	if err != nil {
		c.sendActionRejected(req.ActionID, "invalid_action", err.Error())
		return
	}

	expectedRevision := -1
	if req.ExpectedRevision != nil {
		expectedRevision = *req.ExpectedRevision
	}

	result, err := c.deps.Games.ExecuteActionWithMeta(gameID, action, game.ActionMeta{
		ActionID:         req.ActionID,
		ExpectedRevision: expectedRevision,
		SeatID:           seatID,
	})
	if err != nil {
		if mismatch, ok := err.(*game.RevisionMismatchError); ok {
			c.sendActionRejected(req.ActionID, "revision_mismatch", mismatch.Error(), map[string]any{
				"expectedRevision": mismatch.Expected,
				"currentRevision":  mismatch.Current,
			})
			return
		}
		c.sendActionRejected(req.ActionID, "action_rejected", err.Error())
		return
	}

	acceptedMsg, _ := json.Marshal(map[string]any{
		"type": "action_accepted",
		"payload": map[string]any{
			"actionId":    req.ActionID,
			"newRevision": result.Revision,
			"duplicate":   result.Duplicate,
		},
	})
	c.send <- acceptedMsg

	gameState := c.deps.Games.SerializeGameState(gameID)
	if gameState == nil {
		return
	}
	stateMsg, _ := json.Marshal(map[string]any{
		"type":    "game_state_update",
		"payload": gameState,
	})
	c.hub.BroadcastToGame(gameID, stateMsg)

	if pendingDecision, ok := gameState["pendingDecision"]; ok && pendingDecision != nil {
		decisionMsg, _ := json.Marshal(map[string]any{
			"type":    "decision_required",
			"payload": pendingDecision,
		})
		c.hub.BroadcastToGame(gameID, decisionMsg)
	}
}

func buildActionFromPayload(req performActionPayload, seatID string) (game.Action, error) {
	params := map[string]json.RawMessage{}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid params payload: %w", err)
		}
	}

	getParam := func(keys ...string) (json.RawMessage, bool) {
		for _, key := range keys {
			if raw, ok := params[key]; ok {
				return raw, true
			}
		}
		return nil, false
	}

	parseHexParam := func(keys ...string) (board.Hex, error) {
		raw, ok := getParam(keys...)
		if ok {
			var hp hexParam
			if err := json.Unmarshal(raw, &hp); err != nil {
				return board.Hex{}, fmt.Errorf("invalid hex parameter: %w", err)
			}
			return board.NewHex(hp.Q, hp.R), nil
		}
		if req.Hex != nil {
			return board.NewHex(req.Hex.Q, req.Hex.R), nil
		}
		return board.Hex{}, fmt.Errorf("missing hex parameter")
	}

	parseIntParam := func(keys ...string) (int, error) {
		raw, ok := getParam(keys...)
		if !ok {
			return 0, fmt.Errorf("missing integer parameter")
		}
		var val int
		if err := json.Unmarshal(raw, &val); err == nil {
			return val, nil
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			parsed, err := strconv.Atoi(s)
			if err != nil {
				return 0, fmt.Errorf("invalid integer parameter: %s", s)
			}
			return parsed, nil
		}
		return 0, fmt.Errorf("invalid integer parameter")
	}

	parseOptionalIntParam := func(defaultValue int, keys ...string) (int, error) {
		raw, ok := getParam(keys...)
		if !ok {
			return defaultValue, nil
		}
		var val int
		if err := json.Unmarshal(raw, &val); err == nil {
			return val, nil
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			parsed, err := strconv.Atoi(s)
			if err != nil {
				return 0, fmt.Errorf("invalid integer parameter: %s", s)
			}
			return parsed, nil
		}
		return 0, fmt.Errorf("invalid integer parameter")
	}

	parseBoolParam := func(defaultValue bool, keys ...string) (bool, error) {
		raw, ok := getParam(keys...)
		if !ok {
			return defaultValue, nil
		}
		var val bool
		if err := json.Unmarshal(raw, &val); err != nil {
			return false, fmt.Errorf("invalid bool parameter")
		}
		return val, nil
	}

	parseOptionalBoolParam := func(keys ...string) (*bool, error) {
		raw, ok := getParam(keys...)
		if !ok {
			return nil, nil
		}
		var val bool
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, fmt.Errorf("invalid bool parameter")
		}
		return &val, nil
	}

	parseStringParam := func(keys ...string) (string, error) {
		raw, ok := getParam(keys...)
		if !ok {
			return "", fmt.Errorf("missing string parameter")
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", fmt.Errorf("invalid string parameter")
		}
		return s, nil
	}

	parseBridgeEndpoints := func() (board.Hex, board.Hex, error) {
		hex1, err := parseHexParam("bridgeHex1", "fromHex")
		if err != nil {
			return board.Hex{}, board.Hex{}, err
		}
		hex2, err := parseHexParam("bridgeHex2", "toHex")
		if err != nil {
			return board.Hex{}, board.Hex{}, err
		}
		return hex1, hex2, nil
	}

	switch req.Type {
	case "auction_nominate":
		factionName, err := parseStringParam("faction")
		if err != nil {
			return nil, err
		}
		return game.NewAuctionNominateFactionAction(seatID, models.FactionTypeFromString(factionName)), nil

	case "auction_bid":
		factionName, err := parseStringParam("faction")
		if err != nil {
			return nil, err
		}
		vpReduction, err := parseIntParam("vpReduction")
		if err != nil {
			return nil, err
		}
		return game.NewAuctionPlaceBidAction(seatID, models.FactionTypeFromString(factionName), vpReduction), nil

	case "fast_auction_submit_bids":
		rawBids, ok := getParam("bids")
		if !ok {
			return nil, fmt.Errorf("missing fast auction bids")
		}
		var bidByFaction map[string]int
		if err := json.Unmarshal(rawBids, &bidByFaction); err != nil {
			return nil, fmt.Errorf("invalid fast auction bids payload: %w", err)
		}
		converted := make(map[models.FactionType]int, len(bidByFaction))
		for factionName, bid := range bidByFaction {
			faction := models.FactionTypeFromString(strings.TrimSpace(factionName))
			if faction == models.FactionUnknown {
				return nil, fmt.Errorf("invalid fast auction faction: %s", factionName)
			}
			converted[faction] = bid
		}
		return game.NewFastAuctionSubmitBidsAction(seatID, converted), nil

	case "select_faction":
		factionName := req.Faction
		if factionName == "" {
			if v, err := parseStringParam("faction"); err == nil {
				factionName = v
			}
		}
		if factionName == "" {
			return nil, fmt.Errorf("missing faction")
		}
		return &game.SelectFactionAction{
			PlayerID:    seatID,
			FactionType: models.FactionTypeFromString(factionName),
		}, nil

	case "setup_dwelling":
		hex, err := parseHexParam("hex")
		if err != nil {
			return nil, err
		}
		return game.NewSetupDwellingAction(seatID, hex), nil

	case "setup_bonus_card":
		bonusCard, err := parseBonusCardType(getParam)
		if err != nil {
			return nil, err
		}
		return &game.SetupBonusCardAction{
			BaseAction: game.BaseAction{Type: game.ActionSetupBonusCard, PlayerID: seatID},
			BonusCard:  bonusCard,
		}, nil

	case "transform_build":
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		buildDwelling, err := parseBoolParam(false, "buildDwelling")
		if err != nil {
			return nil, err
		}
		useSkip, err := parseBoolParam(false, "useSkip")
		if err != nil {
			return nil, err
		}
		targetTerrain := models.TerrainTypeUnknown
		if raw, ok := getParam("targetTerrain"); ok {
			terrain, err := parseTerrainTypeRaw(raw)
			if err != nil {
				return nil, err
			}
			targetTerrain = terrain
		}
		if useSkip {
			a := game.NewTransformAndBuildActionWithSkip(seatID, hex, buildDwelling, targetTerrain)
			return a, nil
		}
		return game.NewTransformAndBuildAction(seatID, hex, buildDwelling, targetTerrain), nil

	case "upgrade_building":
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		newType, err := parseBuildingType(getParam)
		if err != nil {
			return nil, err
		}
		return game.NewUpgradeBuildingAction(seatID, hex, newType), nil

	case "advance_shipping":
		return game.NewAdvanceShippingAction(seatID), nil

	case "advance_digging":
		return game.NewAdvanceDiggingAction(seatID), nil

	case "advance_chash_track":
		return game.NewAdvanceChashTrackAction(seatID), nil

	case "send_priest":
		track, err := parseCultTrack(getParam)
		if err != nil {
			return nil, err
		}
		spaces, err := parseIntParam("spaces", "spacesToClimb")
		if err != nil {
			return nil, err
		}
		return &game.SendPriestToCultAction{
			BaseAction:    game.BaseAction{Type: game.ActionSendPriestToCult, PlayerID: seatID},
			Track:         track,
			SpacesToClimb: spaces,
		}, nil

	case "power_action_claim":
		actionType, err := parsePowerActionType(getParam)
		if err != nil {
			return nil, err
		}
		if actionType == game.PowerActionBridge {
			hex1, hex2, err := parseBridgeEndpoints()
			if err != nil {
				return nil, err
			}
			a := game.NewPowerActionWithBridge(seatID, hex1, hex2)
			useCoins, err := parseBoolParam(false, "useCoins")
			if err != nil {
				return nil, err
			}
			a.UseCoins = useCoins
			return a, nil
		}
		if actionType == game.PowerActionSpade1 || actionType == game.PowerActionSpade2 {
			hex, err := parseHexParam("hex", "targetHex")
			if err != nil {
				return nil, err
			}
			buildDwelling, err := parseBoolParam(false, "buildDwelling")
			if err != nil {
				return nil, err
			}
			a := game.NewPowerActionWithTransform(seatID, actionType, hex, buildDwelling)
			useSkip, err := parseBoolParam(false, "useSkip")
			if err != nil {
				return nil, err
			}
			a.UseSkip = useSkip
			useCoins, err := parseBoolParam(false, "useCoins")
			if err != nil {
				return nil, err
			}
			a.UseCoins = useCoins
			return a, nil
		}
		a := game.NewPowerAction(seatID, actionType)
		useCoins, err := parseBoolParam(false, "useCoins")
		if err != nil {
			return nil, err
		}
		a.UseCoins = useCoins
		return a, nil

	case "power_bridge_place":
		hex1, hex2, err := parseBridgeEndpoints()
		if err != nil {
			return nil, err
		}
		a := game.NewPowerActionWithBridge(seatID, hex1, hex2)
		useCoins, err := parseBoolParam(false, "useCoins")
		if err != nil {
			return nil, err
		}
		a.UseCoins = useCoins
		return a, nil

	case "engineers_bridge":
		hex1, hex2, err := parseBridgeEndpoints()
		if err != nil {
			return nil, err
		}
		return game.NewEngineersBridgeAction(seatID, hex1, hex2), nil

	case "wisps_stronghold_dwelling":
		targetHex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		return game.NewBuildWispsStrongholdDwellingAction(seatID, targetHex), nil

	case "goblins_treasure":
		rawReward, ok := getParam("rewardType")
		if !ok {
			return nil, fmt.Errorf("missing rewardType")
		}
		var rewardType string
		if err := json.Unmarshal(rawReward, &rewardType); err != nil {
			return nil, fmt.Errorf("invalid rewardType: %w", err)
		}
		return game.NewUseGoblinsTreasureAction(seatID, game.GoblinsTreasureRewardType(rewardType)), nil

	case "select_goblins_cult_track":
		track, err := parseCultTrackFromParams(getParam)()
		if err != nil {
			return nil, err
		}
		return game.NewSelectGoblinsCultTrackAction(seatID, track), nil

	case "special_action_use":
		specialType, err := parseSpecialActionType(getParam)
		if err != nil {
			return nil, err
		}
		return buildSpecialAction(
			seatID,
			specialType,
			parseHexParam,
			parseBoolParam,
			parseIntParam,
			parseBridgeEndpoints,
			parseCultTrackFromParams(getParam),
			getParam,
		)

	case "pass":
		bonusCard, err := parseOptionalBonusCardType(getParam)
		if err != nil {
			return nil, err
		}
		return game.NewPassAction(seatID, bonusCard), nil

	case "confirm_turn":
		return game.NewConfirmTurnAction(seatID), nil

	case "undo_turn":
		return game.NewUndoTurnAction(seatID), nil

	case "accept_leech":
		offerIndex, err := parseIntParam("offerIndex")
		if err != nil {
			return nil, err
		}
		amount, err := parseOptionalIntParam(0, "amount", "powerAmount")
		if err != nil {
			return nil, err
		}
		return game.NewAcceptPowerLeechAmountAction(seatID, offerIndex, amount), nil

	case "decline_leech":
		offerIndex, err := parseIntParam("offerIndex")
		if err != nil {
			return nil, err
		}
		return game.NewDeclinePowerLeechAction(seatID, offerIndex), nil

	case "select_favor_tile":
		tile, err := parseFavorTileType(getParam)
		if err != nil {
			return nil, err
		}
		return &game.SelectFavorTileAction{
			BaseAction: game.BaseAction{Type: game.ActionSelectFavorTile, PlayerID: seatID},
			TileType:   tile,
		}, nil

	case "select_town_tile":
		tileType, err := parseTownTileType(getParam)
		if err != nil {
			return nil, err
		}
		var anchorHex *board.Hex
		if hex, err := parseHexParam("anchorHex", "townHex"); err == nil {
			anchorHex = &hex
		}
		return &game.SelectTownTileAction{
			BaseAction: game.BaseAction{Type: game.ActionSelectTownTile, PlayerID: seatID},
			TileType:   tileType,
			AnchorHex:  anchorHex,
		}, nil

	case "select_town_cult_top":
		tracks, err := parseCultTrackList(getParam)
		if err != nil {
			return nil, err
		}
		return &game.SelectTownCultTopAction{
			BaseAction: game.BaseAction{Type: game.ActionSelectTownCultTop, PlayerID: seatID},
			Tracks:     tracks,
		}, nil

	case "use_cult_spade":
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		targetTerrain := models.TerrainTypeUnknown
		if raw, ok := getParam("targetTerrain"); ok {
			targetTerrain, err = parseTerrainTypeRaw(raw)
			if err != nil {
				return nil, err
			}
		}
		return game.NewUseCultSpadeActionWithTerrain(seatID, hex, targetTerrain), nil

	case "select_cultists_track":
		track, err := parseCultTrack(getParam)
		if err != nil {
			return nil, err
		}
		return game.NewSelectCultistsCultTrackAction(seatID, track), nil

	case "select_djinni_start_cult_track":
		track, err := parseCultTrack(getParam)
		if err != nil {
			return nil, err
		}
		return game.NewSelectDjinniStartingCultTrackAction(seatID, track), nil

	case "select_treasurers_deposit":
		coins, err := parseOptionalIntParam(0, "coinsToTreasury")
		if err != nil {
			return nil, err
		}
		workers, err := parseOptionalIntParam(0, "workersToTreasury")
		if err != nil {
			return nil, err
		}
		priests, err := parseOptionalIntParam(0, "priestsToTreasury")
		if err != nil {
			return nil, err
		}
		return game.NewSelectTreasurersDepositAction(seatID, coins, workers, priests), nil

	case "select_archivists_bonus_card":
		bonusCard, err := parseBonusCardType(getParam)
		if err != nil {
			return nil, err
		}
		return game.NewSelectArchivistsBonusCardAction(seatID, bonusCard), nil

	case "halflings_apply_spade":
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		rawTerrain, ok := getParam("targetTerrain")
		if !ok {
			return nil, fmt.Errorf("missing targetTerrain")
		}
		targetTerrain, err := parseTerrainTypeRaw(rawTerrain)
		if err != nil {
			return nil, err
		}
		return &game.ApplyHalflingsSpadeAction{
			BaseAction:    game.BaseAction{Type: game.ActionApplyHalflingsSpade, PlayerID: seatID},
			TargetHex:     hex,
			TargetTerrain: targetTerrain,
		}, nil

	case "halflings_build_dwelling":
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		return &game.BuildHalflingsDwellingAction{
			BaseAction: game.BaseAction{Type: game.ActionBuildHalflingsDwelling, PlayerID: seatID},
			TargetHex:  hex,
		}, nil

	case "halflings_skip_dwelling":
		return &game.SkipHalflingsDwellingAction{BaseAction: game.BaseAction{Type: game.ActionSkipHalflingsDwelling, PlayerID: seatID}}, nil

	case "darklings_ordination":
		workers, err := parseIntParam("workersToConvert")
		if err != nil {
			return nil, err
		}
		return &game.UseDarklingsPriestOrdinationAction{
			BaseAction:       game.BaseAction{Type: game.ActionUseDarklingsPriestOrdination, PlayerID: seatID},
			WorkersToConvert: workers,
		}, nil

	case "discard_pending_spade":
		count := 1
		if raw, ok := getParam("count"); ok {
			v, err := parseIntRaw(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid count: %w", err)
			}
			count = v
		}
		return game.NewDiscardPendingSpadeAction(seatID, count), nil

	case "conversion":
		conversionType, err := parseStringParam("conversionType")
		if err != nil {
			return nil, err
		}
		amount, err := parseIntParam("amount")
		if err != nil {
			return nil, err
		}
		return &game.ConversionAction{
			BaseAction:     game.BaseAction{Type: game.ActionConversion, PlayerID: seatID},
			ConversionType: game.ConversionType(conversionType),
			Amount:         amount,
		}, nil

	case "burn_power":
		amount, err := parseIntParam("amount")
		if err != nil {
			return nil, err
		}
		return &game.BurnPowerAction{
			BaseAction: game.BaseAction{Type: game.ActionBurnPower, PlayerID: seatID},
			Amount:     amount,
		}, nil

	case "set_player_options":
		var autoLeechMode *game.LeechAutoMode
		if raw, ok := getParam("autoLeechMode"); ok {
			s, err := parseStringRaw(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid autoLeechMode: %w", err)
			}
			mode := game.LeechAutoMode(strings.TrimSpace(s))
			autoLeechMode = &mode
		}
		confirmActions, err := parseOptionalBoolParam("confirmActions")
		if err != nil {
			return nil, err
		}
		autoConvertOnPass, err := parseOptionalBoolParam("autoConvertOnPass")
		if err != nil {
			return nil, err
		}
		showIncomePreview, err := parseOptionalBoolParam("showIncomePreview")
		if err != nil {
			return nil, err
		}
		return game.NewSetPlayerOptionsAction(seatID, autoLeechMode, autoConvertOnPass, confirmActions, showIncomePreview), nil

	default:
		return nil, fmt.Errorf("unknown action type: %s", req.Type)
	}
}

func parsePowerActionType(getParam func(...string) (json.RawMessage, bool)) (game.PowerActionType, error) {
	if raw, ok := getParam("actionType", "powerActionType"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return game.PowerActionType(v), nil
		}
		if s, err := parseStringRaw(raw); err == nil {
			actionType := game.PowerActionTypeFromString(s)
			if actionType != game.PowerActionUnknown {
				return actionType, nil
			}
		}
	}
	return game.PowerActionUnknown, fmt.Errorf("missing or invalid power action type")
}

func parseSpecialActionType(getParam func(...string) (json.RawMessage, bool)) (game.SpecialActionType, error) {
	if raw, ok := getParam("specialActionType", "actionType"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return game.SpecialActionType(v), nil
		}
	}
	return 0, fmt.Errorf("missing or invalid special action type")
}

func parseBuildingType(getParam func(...string) (json.RawMessage, bool)) (models.BuildingType, error) {
	if raw, ok := getParam("newBuildingType", "buildingType", "to"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return models.BuildingType(v), nil
		}
		if s, err := parseStringRaw(raw); err == nil {
			s = strings.ToLower(strings.TrimSpace(s))
			switch s {
			case "dwelling":
				return models.BuildingDwelling, nil
			case "tradinghouse", "trading_house", "trading house", "tp":
				return models.BuildingTradingHouse, nil
			case "temple", "te":
				return models.BuildingTemple, nil
			case "sanctuary", "sa":
				return models.BuildingSanctuary, nil
			case "stronghold", "sh":
				return models.BuildingStronghold, nil
			}
		}
	}
	return models.BuildingTypeUnknown, fmt.Errorf("missing or invalid building type")
}

func parseCultTrack(getParam func(...string) (json.RawMessage, bool)) (game.CultTrack, error) {
	if raw, ok := getParam("track", "cultTrack"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return game.CultTrack(v), nil
		}
		if s, err := parseStringRaw(raw); err == nil {
			track := game.CultTrackFromString(strings.Title(strings.ToLower(strings.TrimSpace(s))))
			if track != game.CultUnknown {
				return track, nil
			}
		}
	}
	return game.CultUnknown, fmt.Errorf("missing or invalid cult track")
}

func parseCultTrackList(getParam func(...string) (json.RawMessage, bool)) ([]game.CultTrack, error) {
	raw, ok := getParam("tracks", "cultTracks")
	if !ok {
		return nil, fmt.Errorf("missing cult track list")
	}

	var ints []int
	if err := json.Unmarshal(raw, &ints); err == nil {
		out := make([]game.CultTrack, 0, len(ints))
		for _, v := range ints {
			out = append(out, game.CultTrack(v))
		}
		return out, nil
	}

	var stringsList []string
	if err := json.Unmarshal(raw, &stringsList); err == nil {
		out := make([]game.CultTrack, 0, len(stringsList))
		for _, s := range stringsList {
			track := game.CultTrackFromString(strings.Title(strings.ToLower(strings.TrimSpace(s))))
			if track == game.CultUnknown {
				return nil, fmt.Errorf("invalid cult track value: %s", s)
			}
			out = append(out, track)
		}
		return out, nil
	}

	return nil, fmt.Errorf("invalid cult track list")
}

func parseCultTrackFromParams(getParam func(...string) (json.RawMessage, bool)) func() (game.CultTrack, error) {
	return func() (game.CultTrack, error) {
		if raw, ok := getParam("track", "cultTrack"); ok {
			if v, err := parseIntRaw(raw); err == nil {
				return game.CultTrack(v), nil
			}
			if s, err := parseStringRaw(raw); err == nil {
				track := game.CultTrackFromString(strings.Title(strings.ToLower(strings.TrimSpace(s))))
				if track != game.CultUnknown {
					return track, nil
				}
			}
		}
		return game.CultUnknown, fmt.Errorf("missing or invalid cult track")
	}
}

func buildSpecialAction(
	seatID string,
	specialType game.SpecialActionType,
	parseHexParam func(...string) (board.Hex, error),
	parseBoolParam func(bool, ...string) (bool, error),
	parseIntParam func(...string) (int, error),
	parseBridgeEndpoints func() (board.Hex, board.Hex, error),
	parseCultTrack func() (game.CultTrack, error),
	getParam func(...string) (json.RawMessage, bool),
) (game.Action, error) {
	parseTransformHexAndBuild := func() (board.Hex, bool, error) {
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return board.Hex{}, false, err
		}
		build, err := parseBoolParam(false, "buildDwelling")
		if err != nil {
			return board.Hex{}, false, err
		}
		return hex, build, nil
	}

	switch specialType {
	case game.SpecialActionAurenCultAdvance, game.SpecialActionWater2CultAdvance, game.SpecialActionBonusCardCultAdvance:
		track, err := parseCultTrack()
		if err != nil {
			return nil, err
		}
		switch specialType {
		case game.SpecialActionAurenCultAdvance:
			return game.NewAurenCultAdvanceAction(seatID, track), nil
		case game.SpecialActionWater2CultAdvance:
			return game.NewWater2CultAdvanceAction(seatID, track), nil
		default:
			return game.NewBonusCardCultAction(seatID, track), nil
		}

	case game.SpecialActionWitchesRide:
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		return game.NewWitchesRideAction(seatID, hex), nil

	case game.SpecialActionSwarmlingsUpgrade:
		hex, err := parseHexParam("hex", "upgradeHex", "targetHex")
		if err != nil {
			return nil, err
		}
		return game.NewSwarmlingsUpgradeAction(seatID, hex), nil

	case game.SpecialActionGiantsTransform:
		hex, build, err := parseTransformHexAndBuild()
		if err != nil {
			return nil, err
		}
		return game.NewGiantsTransformAction(seatID, hex, build), nil

	case game.SpecialActionNomadsSandstorm:
		hex, build, err := parseTransformHexAndBuild()
		if err != nil {
			return nil, err
		}
		return game.NewNomadsSandstormAction(seatID, hex, build), nil

	case game.SpecialActionBonusCardSpade:
		var (
			hex   board.Hex
			build bool
			err   error
		)
		if _, ok := getParam("hex", "targetHex"); ok {
			hex, build, err = parseTransformHexAndBuild()
			if err != nil {
				return nil, err
			}
		}
		targetTerrain := models.TerrainTypeUnknown
		if raw, ok := getParam("targetTerrain"); ok {
			terrain, err := parseTerrainTypeRaw(raw)
			if err != nil {
				return nil, err
			}
			targetTerrain = terrain
		}
		return game.NewBonusCardSpadeAction(seatID, hex, build, targetTerrain), nil

	case game.SpecialActionMermaidsRiverTown:
		riverHex, err := parseHexParam("hex", "targetHex", "riverHex")
		if err != nil {
			return nil, err
		}
		return game.NewMermaidsRiverTownAction(seatID, riverHex), nil

	case game.SpecialActionEnlightenedGainPower:
		return game.NewEnlightenedGainPowerAction(seatID), nil

	case game.SpecialActionConspiratorsSwapFavor:
		returnTile, err := parseFavorTileType(func(keys ...string) (json.RawMessage, bool) {
			more := append([]string{"returnTile"}, keys...)
			return getParam(more...)
		})
		if err != nil {
			return nil, fmt.Errorf("missing or invalid return favor tile: %w", err)
		}
		newTile, err := parseFavorTileType(func(keys ...string) (json.RawMessage, bool) {
			more := append([]string{"newTile"}, keys...)
			return getParam(more...)
		})
		if err != nil {
			return nil, fmt.Errorf("missing or invalid new favor tile: %w", err)
		}
		return game.NewConspiratorsSwapFavorAction(seatID, returnTile, newTile), nil

	case game.SpecialActionChildrenPlacePowerTokens:
		rawHexes, ok := getParam("targetHexes")
		if !ok {
			return nil, fmt.Errorf("missing targetHexes payload")
		}
		var coords []struct {
			Q int `json:"q"`
			R int `json:"r"`
		}
		if err := json.Unmarshal(rawHexes, &coords); err != nil {
			return nil, fmt.Errorf("invalid targetHexes payload: %w", err)
		}
		targetHexes := make([]board.Hex, 0, len(coords))
		for _, coord := range coords {
			targetHexes = append(targetHexes, board.NewHex(coord.Q, coord.R))
		}
		confirmSpendBowl3, err := parseBoolParam(false, "confirmSpendBowl3")
		if err != nil {
			return nil, err
		}
		return game.NewChildrenPlacePowerTokensAction(seatID, targetHexes, confirmSpendBowl3), nil

	case game.SpecialActionProspectorsGainCoins:
		return game.NewProspectorsGainCoinsAction(seatID), nil

	case game.SpecialActionTimeTravelersPowerShift:
		amount, err := parseIntParam("amount")
		if err != nil {
			return nil, fmt.Errorf("missing or invalid amount for time travelers action: %w", err)
		}
		return game.NewTimeTravelersPowerShiftAction(seatID, amount), nil

	case game.SpecialActionDjinniSwapCults:
		firstTrack, err := parseCultTrack()
		if err != nil {
			return nil, fmt.Errorf("missing or invalid first cult track: %w", err)
		}
		secondTrack, err := parseCultTrackFromParams(func(keys ...string) (json.RawMessage, bool) {
			more := append([]string{"secondTrack"}, keys...)
			return getParam(more...)
		})()
		if err != nil {
			return nil, fmt.Errorf("missing or invalid second cult track: %w", err)
		}
		return game.NewDjinniSwapCultsAction(seatID, firstTrack, secondTrack), nil

	case game.SpecialActionArchitectsMoveBridge:
		oldHex1, oldHex2, err := parseBridgeEndpoints()
		if err != nil {
			return nil, fmt.Errorf("missing or invalid bridge to move: %w", err)
		}
		newHex1, err := parseHexParam("targetHex", "newBridgeHex1")
		if err != nil {
			return nil, fmt.Errorf("missing or invalid new bridge endpoint: %w", err)
		}
		newHex2, err := parseHexParam("upgradeHex", "newBridgeHex2")
		if err != nil {
			return nil, fmt.Errorf("missing or invalid new bridge endpoint: %w", err)
		}
		return game.NewArchitectsMoveBridgeAction(seatID, oldHex1, oldHex2, newHex1, newHex2), nil

	case game.SpecialActionShapeshiftersShiftTerrain:
		rawTerrain, ok := getParam("targetTerrain")
		if !ok {
			return nil, fmt.Errorf("missing targetTerrain")
		}
		targetTerrain, err := parseTerrainTypeRaw(rawTerrain)
		if err != nil {
			return nil, fmt.Errorf("invalid targetTerrain: %w", err)
		}
		return game.NewShapeshiftersShiftTerrainAction(seatID, targetTerrain), nil

	case game.SpecialActionSelkiesStronghold:
		targetHex, err := parseHexParam("targetHex")
		if err != nil {
			return nil, fmt.Errorf("missing or invalid targetHex for selkies stronghold: %w", err)
		}
		buildDwelling, err := parseBoolParam(false, "buildDwelling")
		if err != nil {
			return nil, fmt.Errorf("invalid buildDwelling for selkies stronghold: %w", err)
		}
		targetTerrain := models.TerrainTypeUnknown
		if rawTerrain, ok := getParam("targetTerrain"); ok {
			targetTerrain, err = parseTerrainTypeRaw(rawTerrain)
			if err != nil {
				return nil, fmt.Errorf("invalid targetTerrain: %w", err)
			}
		}
		return game.NewSelkiesStrongholdAction(seatID, targetHex, buildDwelling, targetTerrain), nil

	case game.SpecialActionChaosMagiciansDoubleTurn:
		rawFirst, ok := getParam("firstAction")
		if !ok {
			return nil, fmt.Errorf("missing firstAction payload for chaos magicians double turn")
		}
		rawSecond, ok := getParam("secondAction")
		if !ok {
			return nil, fmt.Errorf("missing secondAction payload for chaos magicians double turn")
		}

		var firstSpec nestedActionPayload
		if err := json.Unmarshal(rawFirst, &firstSpec); err != nil {
			return nil, fmt.Errorf("invalid firstAction payload: %w", err)
		}
		var secondSpec nestedActionPayload
		if err := json.Unmarshal(rawSecond, &secondSpec); err != nil {
			return nil, fmt.Errorf("invalid secondAction payload: %w", err)
		}
		if firstSpec.Type == "" || secondSpec.Type == "" {
			return nil, fmt.Errorf("both firstAction and secondAction must specify type")
		}

		firstAction, err := buildActionFromPayload(performActionPayload{
			Type:   firstSpec.Type,
			Params: firstSpec.Params,
		}, seatID)
		if err != nil {
			return nil, fmt.Errorf("invalid firstAction payload: %w", err)
		}

		secondAction, err := buildActionFromPayload(performActionPayload{
			Type:   secondSpec.Type,
			Params: secondSpec.Params,
		}, seatID)
		if err != nil {
			return nil, fmt.Errorf("invalid secondAction payload: %w", err)
		}

		return game.NewChaosMagiciansDoubleTurnAction(seatID, firstAction, secondAction), nil

	default:
		return nil, fmt.Errorf("unsupported special action type: %d", specialType)
	}
}

func parseBonusCardType(getParam func(...string) (json.RawMessage, bool)) (game.BonusCardType, error) {
	if raw, ok := getParam("bonusCard", "bonusCardType"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return game.BonusCardType(v), nil
		}
		if s, err := parseStringRaw(raw); err == nil {
			card := game.BonusCardTypeFromString(s)
			if card != game.BonusCardUnknown {
				return card, nil
			}
		}
	}
	return game.BonusCardUnknown, fmt.Errorf("missing or invalid bonus card")
}

func parseOptionalBonusCardType(getParam func(...string) (json.RawMessage, bool)) (*game.BonusCardType, error) {
	raw, ok := getParam("bonusCard", "bonusCardType")
	if !ok {
		return nil, nil
	}
	if string(raw) == "null" {
		return nil, nil
	}
	card, err := parseBonusCardType(getParam)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func parseFavorTileType(getParam func(...string) (json.RawMessage, bool)) (game.FavorTileType, error) {
	if raw, ok := getParam("tileType", "favorTile", "favorTileType"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return game.FavorTileType(v), nil
		}
		if s, err := parseStringRaw(raw); err == nil {
			tile := game.FavorTileTypeFromString(s)
			if tile != game.FavorTileUnknown {
				return tile, nil
			}
		}
	}
	return game.FavorTileUnknown, fmt.Errorf("missing or invalid favor tile")
}

func parseTownTileType(getParam func(...string) (json.RawMessage, bool)) (models.TownTileType, error) {
	if raw, ok := getParam("tileType", "townTile", "townTileType"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return models.TownTileType(v), nil
		}
		if s, err := parseStringRaw(raw); err == nil {
			tile := models.TownTileTypeFromString(s)
			if tile != models.TownTileUnknown {
				return tile, nil
			}
		}
	}
	return models.TownTileUnknown, fmt.Errorf("missing or invalid town tile")
}

func parseTerrainTypeRaw(raw json.RawMessage) (models.TerrainType, error) {
	if v, err := parseIntRaw(raw); err == nil {
		return models.TerrainType(v), nil
	}
	if s, err := parseStringRaw(raw); err == nil {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "plains":
			return models.TerrainPlains, nil
		case "swamp":
			return models.TerrainSwamp, nil
		case "lake":
			return models.TerrainLake, nil
		case "forest":
			return models.TerrainForest, nil
		case "mountain":
			return models.TerrainMountain, nil
		case "wasteland":
			return models.TerrainWasteland, nil
		case "desert":
			return models.TerrainDesert, nil
		case "river":
			return models.TerrainRiver, nil
		}
	}
	return models.TerrainTypeUnknown, fmt.Errorf("invalid terrain type")
}

func parseIntRaw(raw json.RawMessage) (int, error) {
	var v int
	if err := json.Unmarshal(raw, &v); err == nil {
		return v, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		parsed, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	}
	return 0, fmt.Errorf("not an integer")
}

func parseStringRaw(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}

func scoringTilesFromCodesForFixture(codes []string) ([]game.ScoringTile, error) {
	typeByCode := map[string]game.ScoringTileType{
		"SCORE1": game.ScoringSpades,
		"SCORE2": game.ScoringTown,
		"SCORE3": game.ScoringDwellingWater,
		"SCORE4": game.ScoringStrongholdFire,
		"SCORE5": game.ScoringDwellingFire,
		"SCORE6": game.ScoringTradingHouseWater,
		"SCORE7": game.ScoringStrongholdAir,
		"SCORE8": game.ScoringTradingHouseAir,
		"SCORE9": game.ScoringTemplePriest,
	}

	allTiles := game.GetAllScoringTiles()
	tileByType := make(map[game.ScoringTileType]game.ScoringTile, len(allTiles))
	for _, tile := range allTiles {
		tileByType[tile.Type] = tile
	}

	out := make([]game.ScoringTile, 0, len(codes))
	for _, rawCode := range codes {
		code := strings.ToUpper(strings.TrimSpace(rawCode))
		if code == "" {
			continue
		}
		tileType, ok := typeByCode[code]
		if !ok {
			return nil, fmt.Errorf("unknown scoring tile code: %s", rawCode)
		}
		tile, ok := tileByType[tileType]
		if !ok {
			return nil, fmt.Errorf("missing scoring tile type in registry: %v", tileType)
		}
		out = append(out, tile)
	}
	if len(out) != 6 {
		return nil, fmt.Errorf("expected 6 scoring tiles, got %d", len(out))
	}
	return out, nil
}

func turnOrderPolicyFromFixturePayload(raw string) (game.TurnOrderPolicy, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return game.TurnOrderPolicyPassOrder, nil
	}
	policy := game.TurnOrderPolicy(value)
	switch policy {
	case game.TurnOrderPolicyPassOrder, game.TurnOrderPolicyCyclicFromFirstPasser:
		return policy, nil
	default:
		return "", fmt.Errorf("unknown turn order policy: %s", value)
	}
}

func bonusCardsFromCodesForFixture(codes []string) ([]game.BonusCardType, error) {
	out := make([]game.BonusCardType, 0, len(codes))
	for _, rawCode := range codes {
		code := strings.ToUpper(strings.TrimSpace(rawCode))
		if code == "" {
			continue
		}
		card := notation.ParseBonusCardCode(code)
		if card == game.BonusCardUnknown {
			return nil, fmt.Errorf("unknown bonus card code: %s", rawCode)
		}
		out = append(out, card)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("missing bonus cards")
	}
	return out, nil
}

func (c *Client) sendError(code string) {
	msg, _ := json.Marshal(map[string]any{
		"type":    "error",
		"payload": code,
	})
	c.send <- msg
}

func (c *Client) sendLobbyError(err error) {
	payload := map[string]any{
		"error": "lobby_error",
	}
	switch {
	case errors.Is(err, lobby.ErrGameNotFound):
		payload["error"] = "game_not_found"
	case errors.Is(err, lobby.ErrGameAlreadyStarted):
		payload["error"] = "game_started"
	case errors.Is(err, lobby.ErrGameFull):
		payload["error"] = "game_full"
	case errors.Is(err, lobby.ErrAlreadyInOpenGame):
		payload["error"] = "already_in_game"
		if parts := strings.Split(err.Error(), ": "); len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			payload["gameId"] = strings.TrimSpace(parts[1])
		}
	case errors.Is(err, lobby.ErrPlayerNotInGame):
		payload["error"] = "not_in_game"
	case errors.Is(err, lobby.ErrInvalidMap):
		payload["error"] = "invalid_map"
	default:
		payload["error"] = "join_failed"
	}
	msg, _ := json.Marshal(map[string]any{
		"type":    "error",
		"payload": payload,
	})
	c.send <- msg
}

func (c *Client) sendActionRejected(actionID, code, message string, extras ...map[string]any) {
	payload := map[string]any{
		"actionId": actionID,
		"error":    code,
		"message":  message,
	}
	if len(extras) > 0 {
		for k, v := range extras[0] {
			payload[k] = v
		}
	}
	msg, _ := json.Marshal(map[string]any{
		"type":    "action_rejected",
		"payload": payload,
	})
	c.send <- msg
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if err := c.handleWriteMessage(message, ok); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.handlePing(); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleWriteMessage(message []byte, ok bool) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	if !ok {
		_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
		return fmt.Errorf("channel closed")
	}

	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	if _, err := w.Write(message); err != nil {
		return err
	}

	return w.Close()
}

func (c *Client) handlePing() error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}
