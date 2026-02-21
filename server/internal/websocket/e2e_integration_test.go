package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/lobby"
	"github.com/lukev/tm_server/internal/models"
)

func TestWebsocketE2E_SetupToActionAndTurnAuthority(t *testing.T) {
	deps, server, gameID, clients, state := setupWebsocketGameToAction(t,
		[]string{"p1", "p2"},
		map[string]string{"p1": "Engineers", "p2": "Auren"},
		false,
	)
	defer server.Close()
	defer closeConnections(clients)

	// Wrong player tries to act out of turn.
	currentPlayerID := currentTurnPlayerID(state)
	other := "p1"
	if currentPlayerID == "p1" {
		other = "p2"
	}
	performActionExpectReject(t, clients[other], gameID, "conversion", map[string]any{
		"conversionType": "worker_to_coin",
		"amount":         1,
	}, asInt(state["revision"]))

	// Current player performs a legal free conversion.
	state = performActionAndReadState(t, clients[currentPlayerID], gameID, "conversion", map[string]any{
		"conversionType": "worker_to_coin",
		"amount":         1,
	}, asInt(state["revision"]))
	if asInt(state["phase"]) != int(game.PhaseAction) {
		t.Fatalf("expected to remain in action phase after conversion")
	}

	// Keep manager/deps referenced so helper return values are fully used.
	if _, ok := deps.Games.GetGame(gameID); !ok {
		t.Fatalf("game disappeared unexpectedly")
	}
}

func TestWebsocketContract_SpadeFollowupAndDiscard(t *testing.T) {
	deps, server, gameID, clients, state := setupWebsocketGameToAction(t,
		[]string{"p1", "p2"},
		map[string]string{"p1": "Halflings", "p2": "Witches"},
		false,
	)
	defer server.Close()
	defer closeConnections(clients)

	gs, ok := deps.Games.GetGame(gameID)
	if !ok {
		t.Fatalf("game %s not found", gameID)
	}

	target1, target2 := configureSpadeFollowupScenario(t, gs, "p1")

	sendJSON(t, clients["p1"], map[string]any{
		"type": "get_game_state",
		"payload": map[string]any{
			"gameID":   gameID,
			"playerID": "p1",
		},
	})
	state = asMap(readUntilType(t, clients["p1"], "game_state_update", 4*time.Second)["payload"])

	state = performActionAndReadState(t, clients["p1"], gameID, "power_action_claim", map[string]any{
		"actionType":    int(game.PowerActionSpade2),
		"targetHex":     map[string]any{"q": target1.Q, "r": target1.R},
		"buildDwelling": true,
	}, asInt(state["revision"]))

	pending := asMap(state["pendingDecision"])
	if asString(pending["type"]) != "spade_followup" {
		t.Fatalf("expected spade_followup pending decision, got %v", pending)
	}
	if asString(pending["playerId"]) != "p1" {
		t.Fatalf("expected p1 to resolve spade follow-up, got %v", pending["playerId"])
	}
	if asInt(pending["spadesRemaining"]) != 1 {
		t.Fatalf("expected one remaining spade follow-up, got %v", pending["spadesRemaining"])
	}
	if canBuild, ok := pending["canBuildDwelling"].(bool); !ok || canBuild {
		t.Fatalf("expected canBuildDwelling=false after first ACT6 build, got %v", pending["canBuildDwelling"])
	}

	performActionExpectReject(t, clients["p1"], gameID, "conversion", map[string]any{
		"conversionType": "worker_to_coin",
		"amount":         1,
	}, asInt(state["revision"]))

	performActionExpectReject(t, clients["p1"], gameID, "transform_build", map[string]any{
		"targetHex":     map[string]any{"q": target2.Q, "r": target2.R},
		"buildDwelling": true,
		"targetTerrain": int(models.TerrainPlains),
	}, asInt(state["revision"]))

	turnBeforeDiscard := currentTurnPlayerID(state)
	state = performActionAndReadState(t, clients["p1"], gameID, "discard_pending_spade", map[string]any{
		"count": 1,
	}, asInt(state["revision"]))

	newPending := asMap(state["pendingDecision"])
	if asString(newPending["type"]) == "spade_followup" {
		t.Fatalf("expected spade_followup to clear after discard, still pending: %v", newPending)
	}
	if currentTurnPlayerID(state) == turnBeforeDiscard {
		t.Fatalf("expected turn to advance after resolving spade follow-up")
	}
}

func TestWebsocketSoak_FivePlayers_ReconnectChurn(t *testing.T) {
	playerIDs := []string{"p1", "p2", "p3", "p4", "p5"}
	factions := map[string]string{
		"p1": "Engineers",
		"p2": "Auren",
		"p3": "Cultists",
		"p4": "Darklings",
		"p5": "Nomads",
	}
	_, server, gameID, clients, state := setupWebsocketGameToAction(t, playerIDs, factions, false)
	defer server.Close()
	defer closeConnections(clients)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	performedPasses := 0
	reconnects := 0

	for i := 0; i < 20; i++ {
		if asInt(state["phase"]) == int(game.PhaseEnd) {
			break
		}
		if asInt(state["phase"]) != int(game.PhaseAction) {
			t.Fatalf("expected action phase during soak loop, got %v", state["phase"])
		}

		currentPlayerID := currentTurnPlayerID(state)
		if currentPlayerID == "" {
			t.Fatalf("missing current player at soak iteration %d", i)
		}

		if i%4 == 0 {
			_ = clients[currentPlayerID].Close()
			reconnected := dialWS(t, wsURL)
			clients[currentPlayerID] = reconnected
			reconnects++
			sendJSON(t, reconnected, map[string]any{
				"type": "get_game_state",
				"payload": map[string]any{
					"gameID":   gameID,
					"playerID": currentPlayerID,
				},
			})
			state = readUntilStateRevisionAtLeast(t, reconnected, asInt(state["revision"]), 4*time.Second)
		}

		players := asMap(state["players"])
		currentPlayer := asMap(players[currentPlayerID])
		if asBool(currentPlayer["hasPassed"]) {
			sendJSON(t, clients[currentPlayerID], map[string]any{
				"type": "get_game_state",
				"payload": map[string]any{
					"gameID":   gameID,
					"playerID": currentPlayerID,
				},
			})
			state = readUntilStateRevisionAtLeast(t, clients[currentPlayerID], asInt(state["revision"]), 4*time.Second)
			if allPlayersPassed(state) {
				break
			}
			continue
		}

		otherPlayerID := ""
		for _, pid := range playerIDs {
			if pid != currentPlayerID {
				otherPlayerID = pid
				break
			}
		}
		if otherPlayerID == "" {
			t.Fatalf("failed to find non-current player during soak")
		}

		performActionExpectReject(t, clients[otherPlayerID], gameID, "conversion", map[string]any{
			"conversionType": "worker_to_coin",
			"amount":         1,
		}, asInt(state["revision"]))

		card := firstAvailableBonusCard(state)
		if card < 0 {
			t.Fatalf("no available bonus card during soak at iteration %d", i)
		}

		state = performActionAndReadState(t, clients[currentPlayerID], gameID, "pass", map[string]any{
			"bonusCard": card,
		}, asInt(state["revision"]))
		performedPasses++
	}

	if reconnects < 2 {
		t.Fatalf("expected reconnect churn during soak test, got reconnects=%d", reconnects)
	}
	if performedPasses < len(playerIDs) {
		t.Fatalf("expected at least %d successful pass actions in soak, got %d", len(playerIDs), performedPasses)
	}
}

func setupWebsocketGameToAction(
	t *testing.T,
	playerIDs []string,
	factions map[string]string,
	randomizeTurnOrder bool,
) (ServerDeps, *httptest.Server, string, map[string]*gws.Conn, map[string]any) {
	t.Helper()

	if len(playerIDs) < 2 {
		t.Fatalf("setup requires at least 2 players")
	}

	hub := NewHub()
	go hub.Run()

	deps := ServerDeps{
		Lobby: lobby.NewManager(),
		Games: game.NewManager(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, deps, w, r)
	}))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clients := make(map[string]*gws.Conn, len(playerIDs))
	for _, playerID := range playerIDs {
		clients[playerID] = dialWS(t, wsURL)
	}

	creatorID := playerIDs[0]
	sendJSON(t, clients[creatorID], map[string]any{
		"type": "create_game",
		"payload": map[string]any{
			"name":       "e2e",
			"maxPlayers": len(playerIDs),
			"creator":    creatorID,
		},
	})
	created := readUntilType(t, clients[creatorID], "game_created", 4*time.Second)
	gameID := asString(asMap(created["payload"])["gameId"])
	if gameID == "" {
		t.Fatalf("missing game id in game_created payload")
	}
	// Drain lobby broadcast emitted as part of create_game before issuing joins.
	_ = readUntilType(t, clients[creatorID], "lobby_state", 4*time.Second)

	for _, playerID := range playerIDs[1:] {
		sendJSON(t, clients[playerID], map[string]any{
			"type": "join_game",
			"payload": map[string]any{
				"id":   gameID,
				"name": playerID,
			},
		})
		_ = readUntilType(t, clients[playerID], "game_joined", 4*time.Second)
	}

	sendJSON(t, clients[creatorID], map[string]any{
		"type": "start_game",
		"payload": map[string]any{
			"gameID":             gameID,
			"randomizeTurnOrder": randomizeTurnOrder,
		},
	})

	state := asMap(readUntilType(t, clients[creatorID], "game_state_update", 4*time.Second)["payload"])
	for phase := asInt(state["phase"]); phase != int(game.PhaseAction); phase = asInt(state["phase"]) {
		switch phase {
		case int(game.PhaseFactionSelection):
			currentPlayerID := currentTurnPlayerID(state)
			faction := factions[currentPlayerID]
			if faction == "" {
				t.Fatalf("missing faction mapping for %s", currentPlayerID)
			}
			state = performActionAndReadState(t, clients[currentPlayerID], gameID, "select_faction", map[string]any{
				"faction": faction,
			}, asInt(state["revision"]))
		case int(game.PhaseSetup):
			if asString(state["setupSubphase"]) == "bonus_cards" {
				playerID := currentSetupBonusPlayerID(state)
				card := firstAvailableBonusCard(state)
				if card < 0 {
					t.Fatalf("no available bonus card found during setup")
				}
				state = performActionAndReadState(t, clients[playerID], gameID, "setup_bonus_card", map[string]any{
					"bonusCard": card,
				}, asInt(state["revision"]))
				continue
			}

			playerID := currentSetupDwellingPlayerID(state)
			q, r := firstSetupDwellingHex(t, state, playerID)
			state = performActionAndReadState(t, clients[playerID], gameID, "setup_dwelling", map[string]any{
				"hex": map[string]any{"q": q, "r": r},
			}, asInt(state["revision"]))
		default:
			t.Fatalf("unexpected phase while progressing setup: %v", state["phase"])
		}
	}

	return deps, server, gameID, clients, state
}

func configureSpadeFollowupScenario(t *testing.T, gs *game.GameState, playerID string) (board.Hex, board.Hex) {
	t.Helper()

	player := gs.GetPlayer(playerID)
	if player == nil {
		t.Fatalf("player not found: %s", playerID)
	}

	player.Resources.Power.Bowl3 = 10
	player.Resources.Workers = 20
	player.Resources.Coins = 20
	player.Resources.Priests = 3
	player.HasPassed = false

	gs.Phase = game.PhaseAction
	gs.SetupSubphase = game.SetupSubphaseComplete
	gs.PowerActions.ResetForNewRound()
	gs.PendingSpades = make(map[string]int)
	gs.PendingSpadeBuildAllowed = make(map[string]bool)
	gs.PendingCultRewardSpades = make(map[string]int)

	for i, pid := range gs.TurnOrder {
		if pid == playerID {
			gs.CurrentPlayerIndex = i
			break
		}
	}

	candidates := make([]board.Hex, 0, 4)
	seen := make(map[board.Hex]bool)
	for hex, mapHex := range gs.Map.Hexes {
		if mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
			continue
		}
		for _, neighbor := range gs.Map.GetDirectNeighbors(hex) {
			if seen[neighbor] {
				continue
			}
			neighborHex := gs.Map.GetHex(neighbor)
			if neighborHex == nil || neighborHex.Building != nil || neighborHex.Terrain == models.TerrainRiver {
				continue
			}
			seen[neighbor] = true
			candidates = append(candidates, neighbor)
		}
	}
	if len(candidates) < 2 {
		t.Fatalf("expected at least two adjacent empty hexes for player %s", playerID)
	}

	target1 := candidates[0]
	target2 := candidates[1]

	if err := gs.Map.TransformTerrain(target1, models.TerrainSwamp); err != nil {
		t.Fatalf("failed to set target1 terrain: %v", err)
	}
	if err := gs.Map.TransformTerrain(target2, models.TerrainSwamp); err != nil {
		t.Fatalf("failed to set target2 terrain: %v", err)
	}

	return target1, target2
}

func dialWS(t *testing.T, wsURL string) *gws.Conn {
	t.Helper()
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	return conn
}

func closeConnections(clients map[string]*gws.Conn) {
	for _, conn := range clients {
		if conn != nil {
			_ = conn.Close()
		}
	}
}

func sendJSON(t *testing.T, conn *gws.Conn, payload map[string]any) {
	t.Helper()
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set write deadline failed: %v", err)
	}
	if err := conn.WriteJSON(payload); err != nil {
		t.Fatalf("write json failed: %v", err)
	}
}

func readUntilType(t *testing.T, conn *gws.Conn, want string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatalf("set read deadline failed: %v", err)
		}
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read json failed while waiting for %s: %v", want, err)
		}
		if asString(msg["type"]) == "action_rejected" && want != "action_rejected" {
			t.Fatalf("unexpected action_rejected while waiting for %s: %v", want, msg["payload"])
		}
		if asString(msg["type"]) == "error" {
			t.Fatalf("unexpected error while waiting for %s: %v", want, msg["payload"])
		}
		if asString(msg["type"]) == want {
			return msg
		}
	}
}

func performActionAndReadState(t *testing.T, conn *gws.Conn, gameID, actionType string, params map[string]any, expectedRevision int) map[string]any {
	t.Helper()
	sendJSON(t, conn, map[string]any{
		"type": "perform_action",
		"payload": map[string]any{
			"type":             actionType,
			"gameID":           gameID,
			"actionId":         fmt.Sprintf("%s-%d", actionType, time.Now().UnixNano()),
			"expectedRevision": expectedRevision,
			"params":           params,
		},
	})

	_ = readUntilType(t, conn, "action_accepted", 4*time.Second)
	targetRevision := expectedRevision + 1
	return readUntilStateRevisionAtLeast(t, conn, targetRevision, 4*time.Second)
}

func performActionExpectReject(t *testing.T, conn *gws.Conn, gameID, actionType string, params map[string]any, expectedRevision int) {
	t.Helper()
	sendJSON(t, conn, map[string]any{
		"type": "perform_action",
		"payload": map[string]any{
			"type":             actionType,
			"gameID":           gameID,
			"actionId":         fmt.Sprintf("reject-%s-%d", actionType, time.Now().UnixNano()),
			"expectedRevision": expectedRevision,
			"params":           params,
		},
	})
	rejected := readUntilType(t, conn, "action_rejected", 4*time.Second)
	payload := asMap(rejected["payload"])
	if asString(payload["error"]) == "" {
		t.Fatalf("expected rejection error code, got: %v", payload)
	}
}

func readUntilStateRevisionAtLeast(t *testing.T, conn *gws.Conn, revision int, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatalf("set read deadline failed: %v", err)
		}
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read json failed while waiting for state revision %d: %v", revision, err)
		}

		msgType := asString(msg["type"])
		if msgType == "action_rejected" {
			t.Fatalf("unexpected action_rejected while waiting for state revision %d: %v", revision, msg["payload"])
		}
		if msgType == "error" {
			t.Fatalf("unexpected error while waiting for state revision %d: %v", revision, msg["payload"])
		}
		if msgType != "game_state_update" {
			continue
		}

		payload := asMap(msg["payload"])
		if asInt(payload["revision"]) >= revision {
			return payload
		}
	}
}

func currentTurnPlayerID(state map[string]any) string {
	turnOrderAny := state["turnOrder"].([]any)
	currentTurn := asInt(state["currentTurn"])
	return asString(turnOrderAny[currentTurn])
}

func currentSetupDwellingPlayerID(state map[string]any) string {
	order := state["setupDwellingOrder"].([]any)
	idx := asInt(state["setupDwellingIndex"])
	return asString(order[idx])
}

func currentSetupBonusPlayerID(state map[string]any) string {
	order := state["setupBonusOrder"].([]any)
	idx := asInt(state["setupBonusIndex"])
	return asString(order[idx])
}

func firstAvailableBonusCard(state map[string]any) int {
	bonusCards := asMap(state["bonusCards"])
	available := asMap(bonusCards["available"])
	best := -1
	for key := range available {
		var cardID int
		if _, err := fmt.Sscanf(key, "%d", &cardID); err != nil {
			continue
		}
		if best == -1 || cardID < best {
			best = cardID
		}
	}
	return best
}

func firstSetupDwellingHex(t *testing.T, state map[string]any, playerID string) (int, int) {
	t.Helper()
	players := asMap(state["players"])
	player := asMap(players[playerID])
	faction := asInt(player["faction"])
	homeTerrain := terrainForFaction(models.FactionType(faction))

	hexes := asMap(asMap(state["map"])["hexes"])
	for _, rawHex := range hexes {
		hex := asMap(rawHex)
		if asInt(hex["terrain"]) != int(homeTerrain) {
			continue
		}
		if _, hasBuilding := hex["building"]; hasBuilding {
			continue
		}
		coord := asMap(hex["coord"])
		return asInt(coord["q"]), asInt(coord["r"])
	}

	t.Fatalf("no setup dwelling hex found for player %s", playerID)
	return 0, 0
}

func terrainForFaction(faction models.FactionType) models.TerrainType {
	switch faction {
	case models.FactionNomads, models.FactionFakirs:
		return models.TerrainDesert
	case models.FactionChaosMagicians, models.FactionGiants:
		return models.TerrainWasteland
	case models.FactionSwarmlings, models.FactionMermaids:
		return models.TerrainLake
	case models.FactionWitches, models.FactionAuren:
		return models.TerrainForest
	case models.FactionHalflings, models.FactionCultists:
		return models.TerrainPlains
	case models.FactionAlchemists, models.FactionDarklings:
		return models.TerrainSwamp
	case models.FactionEngineers, models.FactionDwarves:
		return models.TerrainMountain
	default:
		return models.TerrainTypeUnknown
	}
}

func asMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	// handle json.RawMessage-like values passed through interface
	if raw, ok := v.([]byte); ok {
		out := map[string]any{}
		_ = json.Unmarshal(raw, &out)
		return out
	}
	return map[string]any{}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	default:
		return 0
	}
}

func asBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func allPlayersPassed(state map[string]any) bool {
	players := asMap(state["players"])
	if len(players) == 0 {
		return false
	}
	for _, raw := range players {
		player := asMap(raw)
		if !asBool(player["hasPassed"]) {
			return false
		}
	}
	return true
}
