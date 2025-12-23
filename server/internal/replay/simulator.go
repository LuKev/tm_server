package replay

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
)

// GameSimulator manages the execution of a game replay
type GameSimulator struct {
	mu           sync.RWMutex
	InitialState *game.GameState
	CurrentState *game.GameState
	Actions      []notation.LogItem
	CurrentIndex int               // Index of the *next* action to execute
	History      []*game.GameState // Snapshots for undo
}

// NewGameSimulator creates a new simulator
func NewGameSimulator(initialState *game.GameState, actions []notation.LogItem) *GameSimulator {
	// Deep copy initial state for current state
	// For now, we assume initialState is fresh and we can just use it as base
	// But we need a deep copy mechanism if we want to reset.
	// Let's assume game.NewGameState() gives us a fresh empty state,
	// and we might need to re-apply setup if initialState had setup.

	// Better: Store the initial state as a snapshot.
	// But game.GameState might not be easily deep-copyable without a helper.
	// We'll implement a simple copy or serialization/deserialization later if needed.

	sim := &GameSimulator{
		InitialState: initialState, // Warning: Reference
		CurrentState: initialState, // Warning: Reference
		Actions:      actions,
		CurrentIndex: 0,
		History:      make([]*game.GameState, 0),
	}

	// Save initial state to history
	// sim.History = append(sim.History, deepCopy(initialState))

	return sim
}

// StepForward executes the next action
func (s *GameSimulator) StepForward() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.CurrentIndex >= len(s.Actions) {
		return fmt.Errorf("no more actions")
	}

	item := s.Actions[s.CurrentIndex]

	// Check for missing Round 0 bonus cards before Round 1 starts
	if rs, ok := item.(notation.RoundStartItem); ok && rs.Round == 1 {
		// Ensure TurnOrder is populated if empty (might happen if players added via map iteration)
		if len(s.CurrentState.TurnOrder) == 0 {
			for p := range s.CurrentState.Players {
				s.CurrentState.TurnOrder = append(s.CurrentState.TurnOrder, p)
			}
		}

		// Check if all players have a bonus card
		missingPlayers := make([]string, 0)
		for _, p := range s.CurrentState.TurnOrder {
			if !s.CurrentState.BonusCards.PlayerHasCard[p] {
				missingPlayers = append(missingPlayers, p)
			}
		}
		if len(missingPlayers) > 0 {
			return &game.MissingInfoError{
				Type:    "initial_bonus_card",
				Players: missingPlayers,
			}
		}
	}

	switch v := item.(type) {
	case notation.ActionItem:
		if v.Action != nil {
			// Check for missing bonus card in PassAction
			if pass, ok := v.Action.(*game.PassAction); ok {
				// Round 6 pass does not need bonus card
				if s.CurrentState.Round < 6 && pass.BonusCard == nil {
					return &game.MissingInfoError{
						Type:    "pass_bonus_card",
						Players: []string{pass.PlayerID},
					}
				}
			}

			fmt.Printf("Executing action %d: %T\n", s.CurrentIndex, v.Action)
			// Execute the action against the current state
			if err := v.Action.Execute(s.CurrentState); err != nil {
				return fmt.Errorf("action execution failed at index %d: %w", s.CurrentIndex, err)
			}
		}
	case notation.RoundStartItem:
		// Start a new round
		s.CurrentState.StartNewRound()

		// Force sync round number and turn order from log
		s.CurrentState.Round = v.Round
		if len(v.TurnOrder) > 0 {
			s.CurrentState.TurnOrder = v.TurnOrder
		}

		// Grant income for all rounds (including Round 1, as per BGA log)
		s.CurrentState.GrantIncome()

		// Transition to action phase immediately (income is automatic)
		s.CurrentState.StartActionPhase()
	case notation.GameSettingsItem:
		// Initialize players from settings
		for k, v := range v.Settings {
			if strings.HasPrefix(k, "Player:") {
				// k is "Player:Name", v is "Faction"
				// We use the faction name as the player ID in the simulator for simplicity,
				// matching how actions are parsed (Action.GetPlayerID() returns faction name).
				factionName := v
				// Create player if not exists
				if _, exists := s.CurrentState.Players[factionName]; !exists {
					factionType := models.FactionTypeFromString(factionName)
					faction := factions.NewFaction(factionType)
					if err := s.CurrentState.AddPlayer(factionName, faction); err != nil {
						// Log error but continue? Or return error?
						// Since we are in a loop in StepForward, we can't easily return error without changing signature
						// But StepForward returns error.
						return fmt.Errorf("failed to add player %s: %v", factionName, err)
					}
				}
			} else if k == "BonusCards" {
				// Parse bonus cards
				cards := strings.Split(v, ",")
				availableCards := make([]game.BonusCardType, 0)
				for _, cardCode := range cards {
					// Parse "BON1 (Desc)" -> "BON1"
					parts := strings.Split(cardCode, " ")
					code := parts[0]
					cardType := notation.ParseBonusCardCode(code)
					if cardType != game.BonusCardUnknown {
						availableCards = append(availableCards, cardType)
					}
				}
				s.CurrentState.BonusCards.SetAvailableBonusCards(availableCards)
			} else if k == "ScoringTiles" {
				// Parse scoring tiles
				tiles := strings.Split(v, ",")
				s.CurrentState.ScoringTiles = game.NewScoringTileState()
				for i, tileCode := range tiles {
					// Parse "SCORE1 (Desc)" -> "SCORE1"
					parts := strings.Split(tileCode, " ")
					code := parts[0]
					tile, err := parseScoringTile(code)
					if err != nil {
						fmt.Printf("Warning: failed to parse scoring tile %s: %v\n", code, err)
						continue
					}
					// Ensure we don't add more than 6
					if i < 6 {
						s.CurrentState.ScoringTiles.Tiles = append(s.CurrentState.ScoringTiles.Tiles, tile)
					}
				}
			}
		}
	}

	s.CurrentIndex++
	return nil
}

// GetState returns the current game state
func (s *GameSimulator) GetState() *game.GameState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentState
}
