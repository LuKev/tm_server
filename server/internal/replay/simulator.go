package replay

import (
	"fmt"
	"strconv"
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
	incomePending bool             // RoundStart seen but income not yet granted
	incomeGranted bool             // Income granted for current round, but action phase may not have started yet
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
		incomePending: false,
		incomeGranted: false,
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
			// Income-phase handling:
			// - LogPreIncomeAction: executes before income is granted.
			// - LogPostIncomeAction: executes after income is granted but before action phase begins.
			// - Any other action: triggers transition into action phase.
			if s.incomePending {
				switch v.Action.(type) {
				case *notation.LogPreIncomeAction:
					// Keep waiting; income is granted when we hit the first non-pre-income action.
				case *notation.LogPostIncomeAction:
					s.CurrentState.GrantIncome()
					s.incomePending = false
					s.incomeGranted = true
				default:
					s.CurrentState.GrantIncome()
					s.incomePending = false
					s.incomeGranted = true
					// Any unused cult reward spades must be spent during income; they are not available
					// during the action phase.
					if s.CurrentState.PendingCultRewardSpades != nil && len(s.CurrentState.PendingCultRewardSpades) > 0 {
						s.CurrentState.PendingCultRewardSpades = make(map[string]int)
					}
					s.CurrentState.StartActionPhase()
				}
			} else if s.incomeGranted && s.CurrentState.Phase == game.PhaseIncome {
				// Income was granted, but we haven't entered action phase yet (post-income actions).
				if _, ok := v.Action.(*notation.LogPostIncomeAction); !ok {
					// Leaving income phase now.
					if s.CurrentState.PendingCultRewardSpades != nil && len(s.CurrentState.PendingCultRewardSpades) > 0 {
						s.CurrentState.PendingCultRewardSpades = make(map[string]int)
					}
					s.CurrentState.StartActionPhase()
				}
			}
			// Check for missing bonus card in PassAction
			if pass, ok := v.Action.(*game.PassAction); ok {
				// Round 6 pass does not need bonus card
				if s.CurrentState.Round < 6 && pass.BonusCard == nil {
					// Scan ahead for ALL missing pass bonus cards
					allMissing := s.scanAllMissingPassBonusCards()
					return &game.MissingInfoError{
						Type:             "pass_bonus_card",
						Players:          []string{pass.PlayerID},
						Round:            s.CurrentState.Round,
						AllMissingPasses: allMissing,
					}
				}
			}

			// Execute the action against the current state
			if err := v.Action.Execute(s.CurrentState); err != nil {
				if shouldIgnoreMissingDeclineLeechAtFullPower(v.Action, s.CurrentState, err) {
					s.CurrentIndex++
					return nil
				}
				return fmt.Errorf("action execution failed at index %d (%T %#v): %w", s.CurrentIndex, v.Action, v.Action, err)
			}
		}
	case notation.RoundStartItem:
		// If the previous round has ended (all players passed), execute cleanup now.
		// Snellman/BGA logs can contain late reactions after the final PASS of the round
		// (leeches, Cultists +TRACK bonuses, etc.). Triggering cleanup immediately on
		// "all players passed" can run scoring/bonus-card coin placement too early.
		if s.CurrentState.Phase == game.PhaseAction && s.CurrentState.AllPlayersPassed() {
			s.CurrentState.ExecuteCleanupPhase()
		}

		// Start a new round
		s.CurrentState.StartNewRound()

		// Force sync round number and turn order from log
		s.CurrentState.Round = v.Round
		if len(v.TurnOrder) > 0 {
			s.CurrentState.TurnOrder = v.TurnOrder
		}

		// Apply the cult-track reward associated with the *previous* round's scoring tile.
		// Snellman logs these as "cult_income_for_faction" at the start of the next round.
		if v.Round > 1 {
			s.CurrentState.AwardCultRewardsForRound(v.Round - 1)
		}

		// Delay income until we reach the first non-pre-income action for this round.
		// This allows replay to match Snellman interludes between income blocks.
		s.incomePending = true
		s.incomeGranted = false
	case notation.GameSettingsItem:
		// Initialize players from settings
		settings := v.Settings
		for k, settingValue := range settings {
			if strings.HasPrefix(k, "Player:") {
				// k is "Player:Name", settingValue is "Faction"
				// We use the faction name as the player ID in the simulator for simplicity,
				// matching how actions are parsed (Action.GetPlayerID() returns faction name).
				factionName := settingValue
				// Create player if not exists
				if _, exists := s.CurrentState.Players[factionName]; !exists {
					factionType := models.FactionTypeFromString(factionName)
					faction := factions.NewFaction(factionType)
					if err := s.CurrentState.AddPlayer(factionName, faction); err != nil {
						return fmt.Errorf("failed to add player %s: %w", factionName, err)
					}
				} else {
				}

				// Set starting VPs if specified (always update, even if player exists)
				if vpStr, ok := settings["StartingVP:"+factionName]; ok {
					if vp, err := strconv.Atoi(vpStr); err == nil {
						s.CurrentState.Players[factionName].VictoryPoints = vp
					}
				}
			} else if k == "BonusCards" {
				// Parse bonus cards
				cards := strings.Split(settingValue, ",")
				availableCards := make([]game.BonusCardType, 0)
				for _, cardCode := range cards {
					cardCode = strings.TrimSpace(cardCode)
					if cardCode == "" {
						continue
					}
					// Parse "BON1 (Desc)" -> "BON1"
					parts := strings.Fields(cardCode)
					code := parts[0]
					cardType := notation.ParseBonusCardCode(code)
					if cardType != game.BonusCardUnknown {
						availableCards = append(availableCards, cardType)
					}
				}
				s.CurrentState.BonusCards.SetAvailableBonusCards(availableCards)
			} else if k == "ScoringTiles" {
				// Parse scoring tiles
				tiles := strings.Split(settingValue, ",")
				s.CurrentState.ScoringTiles = game.NewScoringTileState()
				for i, tileCode := range tiles {
					// Parse "SCORE1 (Desc)" -> "SCORE1"
					tileCode = strings.TrimSpace(tileCode)
					if tileCode == "" {
						continue
					}
					parts := strings.Fields(tileCode)
					code := parts[0]
					tile, err := parseScoringTile(code)
					if err != nil {
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

func shouldIgnoreMissingDeclineLeechAtFullPower(action game.Action, gs *game.GameState, err error) bool {
	if err == nil || action == nil || gs == nil {
		return false
	}
	if !strings.Contains(err.Error(), "no pending leech offers") {
		return false
	}

	// Replay logs may contain delayed DL rows after the underlying offer has
	// already been resolved; treat log-decline as idempotent.
	if _, ok := action.(*notation.LogDeclineLeechAction); ok {
		return true
	}

	if _, ok := action.(*game.DeclinePowerLeechAction); !ok {
		return false
	}
	player := gs.GetPlayer(action.GetPlayerID())
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		return false
	}
	// Replay-only tolerance: Snellman can emit delayed decline rows after leech has
	// already been resolved. Ignore only when player cannot gain any power anyway.
	return player.Resources.Power.Bowl1 == 0 && player.Resources.Power.Bowl2 == 0
}

// JumpTo fast-forwards the simulator to the target index
// Note: This only supports moving forward. For backward jumps, the simulator must be reset first.
func (s *GameSimulator) JumpTo(targetIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if targetIndex < s.CurrentIndex {
		return fmt.Errorf("cannot jump backwards (current: %d, target: %d)", s.CurrentIndex, targetIndex)
	}

	for s.CurrentIndex < targetIndex {
		// We need to call StepForward, but it locks the mutex.
		// We should unlock here or refactor StepForward to have an internal unlocked version.
		// Since StepForward is public and locks, we can't call it while holding lock.
		s.mu.Unlock()
		err := s.StepForward()
		s.mu.Lock()
		if err != nil {
			return err
		}
	}

	// If we jumped to the end of the log, ensure final cleanup/endgame scoring runs.
	// Some logs end immediately after the last PASS, with no following RoundStartItem
	// to trigger cleanup. Snellman logs can also contain dropped players in round 6,
	// which means not every faction will emit an explicit PASS.
	if targetIndex >= len(s.Actions) && s.CurrentState.Phase == game.PhaseAction {
		if s.CurrentState.AllPlayersPassed() || s.CurrentState.Round >= 6 {
			s.CurrentState.ExecuteCleanupPhase()
		}
	}
	return nil
}

// GetState returns the current game state
func (s *GameSimulator) GetState() *game.GameState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentState
}

// scanAllMissingPassBonusCards scans all remaining actions for missing pass bonus cards
func (s *GameSimulator) scanAllMissingPassBonusCards() map[int][]string {
	result := make(map[int][]string)
	currentRound := s.CurrentState.Round

	// Scan from current index to end of actions
	for i := s.CurrentIndex; i < len(s.Actions); i++ {
		item := s.Actions[i]

		// Track round changes
		if rs, ok := item.(notation.RoundStartItem); ok {
			currentRound = rs.Round
			continue
		}

		// Check for PassAction with nil BonusCard
		if actionItem, ok := item.(notation.ActionItem); ok {
			if pass, ok := actionItem.Action.(*game.PassAction); ok {
				// Round 6 pass does not need bonus card
				if currentRound < 6 && pass.BonusCard == nil {
					result[currentRound] = append(result[currentRound], pass.PlayerID)
				}
			}
		}
	}

	return result
}
