// Package websocket handles websocket connections and messaging.
package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
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
	Name       string `json:"name"`
	MaxPlayers int    `json:"maxPlayers"`
	Creator    string `json:"creator"`
}

type joinGamePayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type startGamePayload struct {
	GameID             string `json:"gameID"`
	RandomizeTurnOrder *bool  `json:"randomizeTurnOrder,omitempty"`
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
		games := c.deps.Lobby.ListGames()
		out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
		c.send <- out

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
				c.sendError("not_in_game")
				return
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

	case "perform_action":
		c.handlePerformAction(env.Payload)

	default:
		log.Printf("Unknown message type: %s", env.Type)
	}
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

	randomize := true
	if p.RandomizeTurnOrder != nil {
		randomize = *p.RandomizeTurnOrder
	}

	err := c.deps.Games.CreateGameWithOptions(p.GameID, meta.Players, game.CreateGameOptions{RandomizeTurnOrder: randomize})
	if err != nil && !strings.Contains(err.Error(), "game already exists") {
		log.Printf("error creating game: %v", err)
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
	meta := c.deps.Lobby.CreateGame(p.Name, p.MaxPlayers)
	if p.Creator != "" {
		_ = c.deps.Lobby.JoinGame(meta.ID, p.Creator)
		c.bindSeat(meta.ID, p.Creator)
		c.hub.JoinGame(c, meta.ID)
		createdMsg, _ := json.Marshal(map[string]any{
			"type":    "game_created",
			"payload": map[string]string{"gameId": meta.ID, "playerId": p.Creator},
		})
		c.send <- createdMsg
	}
	games := c.deps.Lobby.ListGames()
	out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
	c.hub.broadcast <- out
}

func (c *Client) handleJoinGame(payload json.RawMessage) {
	var p joinGamePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("join_game payload error: %v", err)
		return
	}
	ok := c.deps.Lobby.JoinGame(p.ID, p.Name)
	if !ok {
		meta, exists := c.deps.Lobby.GetGame(p.ID)
		if !exists {
			out, _ := json.Marshal(map[string]any{"type": "error", "payload": "join_failed"})
			c.send <- out
			return
		}
		rejoinAllowed := false
		for _, playerID := range meta.Players {
			if playerID == p.Name {
				rejoinAllowed = true
				break
			}
		}
		if !rejoinAllowed {
			out, _ := json.Marshal(map[string]any{"type": "error", "payload": "join_failed"})
			c.send <- out
			return
		}
	}

	c.bindSeat(p.ID, p.Name)
	c.hub.JoinGame(c, p.ID)

	successMsg, _ := json.Marshal(map[string]any{
		"type":    "game_joined",
		"payload": map[string]string{"gameId": p.ID, "playerId": p.Name},
	})
	c.send <- successMsg

	games := c.deps.Lobby.ListGames()
	out, _ := json.Marshal(lobbyStateMsg{Type: "lobby_state", Payload: games})
	c.hub.broadcast <- out
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

	switch req.Type {
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
		bonusCard, err := parseBonusCardType(req, getParam)
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
		newType, err := parseBuildingType(req, getParam)
		if err != nil {
			return nil, err
		}
		return game.NewUpgradeBuildingAction(seatID, hex, newType), nil

	case "advance_shipping":
		return game.NewAdvanceShippingAction(seatID), nil

	case "advance_digging":
		return game.NewAdvanceDiggingAction(seatID), nil

	case "send_priest":
		track, err := parseCultTrack(req, getParam)
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
		actionType, err := parsePowerActionType(req, getParam)
		if err != nil {
			return nil, err
		}
		if actionType == game.PowerActionBridge {
			hex1, err := parseHexParam("bridgeHex1", "fromHex")
			if err != nil {
				return nil, err
			}
			hex2, err := parseHexParam("bridgeHex2", "toHex")
			if err != nil {
				return nil, err
			}
			return game.NewPowerActionWithBridge(seatID, hex1, hex2), nil
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
			return a, nil
		}
		return game.NewPowerAction(seatID, actionType), nil

	case "power_bridge_place":
		hex1, err := parseHexParam("bridgeHex1", "fromHex")
		if err != nil {
			return nil, err
		}
		hex2, err := parseHexParam("bridgeHex2", "toHex")
		if err != nil {
			return nil, err
		}
		return game.NewPowerActionWithBridge(seatID, hex1, hex2), nil

	case "engineers_bridge":
		hex1, err := parseHexParam("bridgeHex1", "fromHex")
		if err != nil {
			return nil, err
		}
		hex2, err := parseHexParam("bridgeHex2", "toHex")
		if err != nil {
			return nil, err
		}
		return game.NewEngineersBridgeAction(seatID, hex1, hex2), nil

	case "special_action_use":
		specialType, err := parseSpecialActionType(req, getParam)
		if err != nil {
			return nil, err
		}
		return buildSpecialAction(seatID, specialType, parseHexParam, parseBoolParam, parseCultTrackFromParams(getParam), getParam)

	case "pass":
		bonusCard, err := parseOptionalBonusCardType(req, getParam)
		if err != nil {
			return nil, err
		}
		return game.NewPassAction(seatID, bonusCard), nil

	case "accept_leech":
		offerIndex, err := parseIntParam("offerIndex")
		if err != nil {
			return nil, err
		}
		return game.NewAcceptPowerLeechAction(seatID, offerIndex), nil

	case "decline_leech":
		offerIndex, err := parseIntParam("offerIndex")
		if err != nil {
			return nil, err
		}
		return game.NewDeclinePowerLeechAction(seatID, offerIndex), nil

	case "select_favor_tile":
		tile, err := parseFavorTileType(req, getParam)
		if err != nil {
			return nil, err
		}
		return &game.SelectFavorTileAction{
			BaseAction: game.BaseAction{Type: game.ActionSelectFavorTile, PlayerID: seatID},
			TileType:   tile,
		}, nil

	case "select_town_tile":
		tileType, err := parseTownTileType(req, getParam)
		if err != nil {
			return nil, err
		}
		return &game.SelectTownTileAction{
			BaseAction: game.BaseAction{Type: game.ActionSelectTownTile, PlayerID: seatID},
			TileType:   tileType,
		}, nil

	case "select_town_cult_top":
		tracks, err := parseCultTrackList(req, getParam)
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
		return game.NewUseCultSpadeAction(seatID, hex), nil

	case "select_cultists_track":
		track, err := parseCultTrack(req, getParam)
		if err != nil {
			return nil, err
		}
		return game.NewSelectCultistsCultTrackAction(seatID, track), nil

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

	default:
		return nil, fmt.Errorf("unknown action type: %s", req.Type)
	}
}

func parsePowerActionType(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (game.PowerActionType, error) {
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

func parseSpecialActionType(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (game.SpecialActionType, error) {
	if raw, ok := getParam("specialActionType", "actionType"); ok {
		if v, err := parseIntRaw(raw); err == nil {
			return game.SpecialActionType(v), nil
		}
	}
	return 0, fmt.Errorf("missing or invalid special action type")
}

func parseBuildingType(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (models.BuildingType, error) {
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

func parseCultTrack(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (game.CultTrack, error) {
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

func parseCultTrackList(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) ([]game.CultTrack, error) {
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
	parseCultTrack func() (game.CultTrack, error),
	getParam func(...string) (json.RawMessage, bool),
) (game.Action, error) {
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
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		build, err := parseBoolParam(false, "buildDwelling")
		if err != nil {
			return nil, err
		}
		return game.NewGiantsTransformAction(seatID, hex, build), nil

	case game.SpecialActionNomadsSandstorm:
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		build, err := parseBoolParam(false, "buildDwelling")
		if err != nil {
			return nil, err
		}
		return game.NewNomadsSandstormAction(seatID, hex, build), nil

	case game.SpecialActionBonusCardSpade:
		hex, err := parseHexParam("hex", "targetHex")
		if err != nil {
			return nil, err
		}
		build, err := parseBoolParam(false, "buildDwelling")
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
		return game.NewBonusCardSpadeAction(seatID, hex, build, targetTerrain), nil

	case game.SpecialActionMermaidsRiverTown:
		riverHex, err := parseHexParam("hex", "targetHex", "riverHex")
		if err != nil {
			return nil, err
		}
		return game.NewMermaidsRiverTownAction(seatID, riverHex), nil

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

func parseBonusCardType(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (game.BonusCardType, error) {
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

func parseOptionalBonusCardType(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (*game.BonusCardType, error) {
	raw, ok := getParam("bonusCard", "bonusCardType")
	if !ok {
		return nil, nil
	}
	if string(raw) == "null" {
		return nil, nil
	}
	card, err := parseBonusCardType(req, getParam)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func parseFavorTileType(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (game.FavorTileType, error) {
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

func parseTownTileType(req performActionPayload, getParam func(...string) (json.RawMessage, bool)) (models.TownTileType, error) {
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

func (c *Client) sendError(code string) {
	msg, _ := json.Marshal(map[string]any{
		"type":    "error",
		"payload": code,
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
