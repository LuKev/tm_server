package game

// Cleanup Phase (Phase 3)
// Executes at the end of each round (rounds 1-5, not round 6)
// Order of operations:
// 1. Award cult track rewards (based on scoring tile)
// 2. Add coins to leftover bonus cards (players keep their cards across rounds)
// 3. Reset round-specific state
// Note: Bonus cards are NOT returned during cleanup - players keep them across rounds.
// Cards are only returned when players pass and select a new card (handled in TakeBonusCard).

// ExecuteCleanupPhase performs all cleanup tasks at the end of a round
// This should be called after all players have passed
// Returns true if the game should continue, false if it should end
func (gs *GameState) ExecuteCleanupPhase() bool {
	// Round 6 doesn't have a cleanup phase - game ends immediately
	if gs.Round >= 6 {
		gs.Phase = PhaseEnd
		return false
	}
	
	gs.Phase = PhaseCleanup
	
	// 1. Award cult track rewards based on the current round's scoring tile
	gs.AwardCultRewards()
	
	// 2. Add 1 coin to each available (unselected) bonus card
	// NOTE: Players keep their bonus cards across rounds - cards are only returned when
	// players pass and select a new card. Coins accumulate on cards in the Available pool.
	if gs.BonusCards != nil {
		gs.BonusCards.AddCoinsToLeftoverCards()
	}
	
	// 3. Reset round-specific state
	gs.ResetRoundState()
	
	// Game continues to next round
	return true
}

// ResetRoundState resets all round-specific state
// Called at end of round during cleanup
func (gs *GameState) ResetRoundState() {
	// Reset power actions
	if gs.PowerActions != nil {
		gs.PowerActions.ResetForNewRound()
	}
	
	// Reset player round-specific flags
	for _, player := range gs.Players {
		player.HasPassed = false
		// Note: Bonus card special actions are tied to having the card,
		// which is reset when cards are returned
		// Note: Stronghold abilities are tracked per-faction, not on Player
	}
	
	// Clear pass order (will be rebuilt next round)
	gs.PassOrder = []string{}
	
	// Reset pending offers/formations
	gs.PendingLeechOffers = make(map[string][]*PowerLeechOffer)
	gs.PendingTownFormations = make(map[string]*PendingTownFormation)

	// Note: PendingSpades will be cleared at the start of the next income phase
	// This allows cult reward spades to be used for bonus transforms during cleanup
}

// HasPendingSpades checks if any player has pending spades to use
func (gs *GameState) HasPendingSpades() bool {
	if gs.PendingSpades == nil {
		return false
	}
	
	for _, count := range gs.PendingSpades {
		if count > 0 {
			return true
		}
	}
	
	return false
}

// GetNextPlayerWithSpades returns the next player (in pass order) who has pending spades
// Returns empty string if no players have spades
func (gs *GameState) GetNextPlayerWithSpades() string {
	if gs.PendingSpades == nil || len(gs.PassOrder) == 0 {
		return ""
	}
	
	// Check players in pass order
	for _, playerID := range gs.PassOrder {
		if count, ok := gs.PendingSpades[playerID]; ok && count > 0 {
			return playerID
		}
	}
	
	return ""
}

// UseSpadeFromReward uses one pending spade for a player
// Returns true if successful, false if player has no spades
func (gs *GameState) UseSpadeFromReward(playerID string) bool {
	if gs.PendingSpades == nil {
		return false
	}
	
	if count, ok := gs.PendingSpades[playerID]; ok && count > 0 {
		gs.PendingSpades[playerID]--
		if gs.PendingSpades[playerID] == 0 {
			delete(gs.PendingSpades, playerID)
		}
		return true
	}
	
	return false
}

// ClearPendingSpades clears all pending spades (called after all spades are used or forfeited)
func (gs *GameState) ClearPendingSpades() {
	gs.PendingSpades = make(map[string]int)
}
