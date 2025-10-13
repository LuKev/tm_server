package game

import (
	"testing"
	"github.com/lukev/tm_server/internal/game/factions"
)

func TestTurnOrder_PassOrderDeterminesNextRound(t *testing.T) {
	gs := NewGameState()
	
	// Add 3 players
	faction1 := factions.NewHalflings()
	faction2 := factions.NewCultists()
	faction3 := factions.NewNomads()
	
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	gs.AddPlayer("player3", faction3)
	
	// Set initial turn order
	gs.TurnOrder = []string{"player1", "player2", "player3"}
	gs.CurrentPlayerIndex = 0
	
	// Players pass in order: player2, player3, player1
	pass2 := NewPassAction("player2", "")
	err := pass2.Execute(gs)
	if err != nil {
		t.Fatalf("player2 pass failed: %v", err)
	}
	
	pass3 := NewPassAction("player3", "")
	err = pass3.Execute(gs)
	if err != nil {
		t.Fatalf("player3 pass failed: %v", err)
	}
	
	pass1 := NewPassAction("player1", "")
	err = pass1.Execute(gs)
	if err != nil {
		t.Fatalf("player1 pass failed: %v", err)
	}
	
	// Verify pass order was recorded
	if len(gs.PassOrder) != 3 {
		t.Fatalf("expected 3 players in pass order, got %d", len(gs.PassOrder))
	}
	if gs.PassOrder[0] != "player2" {
		t.Errorf("expected player2 to pass first, got %s", gs.PassOrder[0])
	}
	if gs.PassOrder[1] != "player3" {
		t.Errorf("expected player3 to pass second, got %s", gs.PassOrder[1])
	}
	if gs.PassOrder[2] != "player1" {
		t.Errorf("expected player1 to pass third, got %s", gs.PassOrder[2])
	}
	
	// Start new round
	gs.StartNewRound()
	
	// Verify turn order matches pass order
	if len(gs.TurnOrder) != 3 {
		t.Fatalf("expected 3 players in turn order, got %d", len(gs.TurnOrder))
	}
	if gs.TurnOrder[0] != "player2" {
		t.Errorf("expected player2 to go first, got %s", gs.TurnOrder[0])
	}
	if gs.TurnOrder[1] != "player3" {
		t.Errorf("expected player3 to go second, got %s", gs.TurnOrder[1])
	}
	if gs.TurnOrder[2] != "player1" {
		t.Errorf("expected player1 to go third, got %s", gs.TurnOrder[2])
	}
	
	// Verify pass order was reset
	if len(gs.PassOrder) != 0 {
		t.Errorf("expected pass order to be reset, got %d entries", len(gs.PassOrder))
	}
	
	// Verify all players' HasPassed was reset
	for _, playerID := range gs.TurnOrder {
		player := gs.GetPlayer(playerID)
		if player.HasPassed {
			t.Errorf("expected player %s HasPassed to be reset", playerID)
		}
	}
}

func TestTurnOrder_GetCurrentPlayer(t *testing.T) {
	gs := NewGameState()
	
	faction1 := factions.NewHalflings()
	faction2 := factions.NewCultists()
	
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	gs.TurnOrder = []string{"player1", "player2"}
	gs.CurrentPlayerIndex = 0
	
	// Current player should be player1
	currentPlayer := gs.GetCurrentPlayer()
	if currentPlayer == nil {
		t.Fatal("expected current player to be non-nil")
	}
	if currentPlayer.ID != "player1" {
		t.Errorf("expected current player to be player1, got %s", currentPlayer.ID)
	}
	
	// Advance to next turn
	gs.NextTurn()
	
	// Current player should be player2
	currentPlayer = gs.GetCurrentPlayer()
	if currentPlayer == nil {
		t.Fatal("expected current player to be non-nil")
	}
	if currentPlayer.ID != "player2" {
		t.Errorf("expected current player to be player2, got %s", currentPlayer.ID)
	}
}

func TestTurnOrder_SkipsPassedPlayers(t *testing.T) {
	gs := NewGameState()
	
	faction1 := factions.NewHalflings()
	faction2 := factions.NewCultists()
	faction3 := factions.NewNomads()
	
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	gs.AddPlayer("player3", faction3)
	
	gs.TurnOrder = []string{"player1", "player2", "player3"}
	gs.CurrentPlayerIndex = 0
	
	// Player2 passes
	pass2 := NewPassAction("player2", "")
	err := pass2.Execute(gs)
	if err != nil {
		t.Fatalf("player2 pass failed: %v", err)
	}
	
	// Advance turn from player1
	gs.NextTurn()
	
	// Should skip player2 and go to player3
	currentPlayer := gs.GetCurrentPlayer()
	if currentPlayer == nil {
		t.Fatal("expected current player to be non-nil")
	}
	if currentPlayer.ID != "player3" {
		t.Errorf("expected current player to be player3 (skipping passed player2), got %s", currentPlayer.ID)
	}
}

func TestTurnOrder_AllPlayersPassed(t *testing.T) {
	gs := NewGameState()
	
	faction1 := factions.NewHalflings()
	faction2 := factions.NewCultists()
	
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)
	
	gs.TurnOrder = []string{"player1", "player2"}
	
	// Initially, not all players have passed
	if gs.AllPlayersPassed() {
		t.Error("expected AllPlayersPassed to be false initially")
	}
	
	// Player1 passes
	pass1 := NewPassAction("player1", "")
	pass1.Execute(gs)
	
	// Still not all passed
	if gs.AllPlayersPassed() {
		t.Error("expected AllPlayersPassed to be false with 1 player passed")
	}
	
	// Player2 passes
	pass2 := NewPassAction("player2", "")
	pass2.Execute(gs)
	
	// Now all have passed
	if !gs.AllPlayersPassed() {
		t.Error("expected AllPlayersPassed to be true with all players passed")
	}
}
