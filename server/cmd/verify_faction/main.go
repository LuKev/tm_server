package main

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	serverAddr = "localhost:8080"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type GameState struct {
	ID          string             `json:"id"`
	Phase       int                `json:"phase"`
	CurrentTurn int                `json:"currentTurn"`
	Players     map[string]*Player `json:"players"`
	Order       []string           `json:"order"`
}

type Player struct {
	ID      string `json:"id"`
	Faction string `json:"faction"`
}

func connect(name string) *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/ws", RawQuery: "name=" + name}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	return c
}

func send(c *websocket.Conn, msgType string, payload interface{}) {
	p, _ := json.Marshal(payload)
	msg := Message{Type: msgType, Payload: p}
	if err := c.WriteJSON(msg); err != nil {
		log.Fatalf("write: %v", err)
	}
}

func read(c *websocket.Conn) Message {
	var msg Message
	if err := c.ReadJSON(&msg); err != nil {
		log.Fatalf("read: %v", err)
	}
	log.Printf("Read message: %s", msg.Type)
	return msg
}

func main() {
	log.Println("Starting verification...")

	// 1. Alice connects
	alice := connect("Alice")
	defer alice.Close()
	log.Println("Alice connected")

	// Read initial lobby update (optional, might block if not sent immediately)
	// read(alice)

	// 2. Alice creates game
	send(alice, "create_game", map[string]interface{}{
		"name":       "VerifyGame",
		"maxPlayers": 2,
		"creator":    "Alice",
	})
	log.Println("Alice created game")

	// Read game created message
	var gameID string
	var msg Message
	for {
		msg = read(alice)
		log.Printf("Received message: %s", msg.Type)
		if msg.Type == "game_created" {
			var payload map[string]string
			json.Unmarshal(msg.Payload, &payload)
			gameID = payload["gameId"]
			break
		}
	}
	log.Printf("Game ID: %s", gameID)

	// Read lobby update
	read(alice)

	// 3. Bob connects
	bob := connect("Bob")
	defer bob.Close()
	log.Println("Bob connected")
	// read(bob) // lobby update (might block)

	// 4. Bob joins game
	send(bob, "join_game", map[string]string{"gameID": gameID})
	log.Println("Bob joined game")

	// Read updates (Bob joined)
	log.Println("Reading Alice update...")
	read(alice) // game_joined or lobby update?
	log.Println("Reading Bob update...")
	read(bob) // game_joined

	// 5. Alice starts game
	send(alice, "start_game", map[string]string{"gameID": gameID})
	log.Println("Alice started game")

	// Read game_started and initial state
	// Alice receives: game_started, game_state_update
	// Bob receives: game_started, game_state_update

	// Helper to wait for game state with specific phase
	waitForPhase := func(c *websocket.Conn, phase int) GameState {
		for {
			msg := read(c)
			if msg.Type == "game_state_update" {
				var gs GameState
				json.Unmarshal(msg.Payload, &gs)
				if gs.Phase == phase {
					return gs
				}
			}
		}
	}

	// Verify PhaseFactionSelection (1)
	log.Println("Waiting for faction selection phase...")
	gsAlice := waitForPhase(alice, 1)
	log.Println("Alice in faction selection")

	// 6. Alice selects Nomads
	// Check whose turn it is
	firstPlayer := gsAlice.Order[gsAlice.CurrentTurn]
	log.Printf("First player: %s", firstPlayer)

	var activeConn *websocket.Conn
	var activeName string
	if firstPlayer == "Alice" {
		activeConn = alice
		activeName = "Alice"
	} else {
		activeConn = bob
		activeName = "Bob"
	}

	log.Printf("%s selecting Nomads...", activeName)
	send(activeConn, "perform_action", map[string]interface{}{
		"type":     "select_faction",
		"playerID": activeName,
		"faction":  "Nomads",
		"gameID":   gameID,
	})

	// Verify update
	// Both receive update
	// Wait for next turn
	time.Sleep(100 * time.Millisecond)

	// 7. Second player selects Witches
	var secondConn *websocket.Conn
	var secondName string
	if activeName == "Alice" {
		secondConn = bob
		secondName = "Bob"
	} else {
		secondConn = alice
		secondName = "Alice"
	}

	log.Printf("%s selecting Witches...", secondName)
	send(secondConn, "perform_action", map[string]interface{}{
		"type":     "select_faction",
		"playerID": secondName,
		"faction":  "Witches",
		"gameID":   gameID,
	})

	// 8. Verify transition to Setup phase (0)
	log.Println("Waiting for setup phase...")
	gsFinal := waitForPhase(alice, 0)
	log.Println("Game transitioned to Setup phase!")

	// Verify factions
	p1 := gsFinal.Players["Alice"]
	p2 := gsFinal.Players["Bob"]
	log.Printf("Alice faction: %s", p1.Faction)
	log.Printf("Bob faction: %s", p2.Faction)

	if p1.Faction == "" || p2.Faction == "" {
		log.Fatal("Factions not assigned correctly")
	}

	log.Println("Verification SUCCESS!")
}
