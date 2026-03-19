package game

import (
	"slices"
	"strings"
	"time"
)

func NewTurnTimerState(playerIDs []string, config TurnTimerConfig) *TurnTimerState {
	if config.InitialTimeMs <= 0 {
		return nil
	}

	timers := &TurnTimerState{
		Config: TurnTimerConfig{
			InitialTimeMs: config.InitialTimeMs,
			IncrementMs:   maxInt64(0, config.IncrementMs),
		},
		Players: make(map[string]*PlayerTurnTimer, len(playerIDs)),
	}
	for _, playerID := range playerIDs {
		timers.Players[playerID] = &PlayerTurnTimer{RemainingMs: config.InitialTimeMs}
	}
	return timers
}

func (tt *TurnTimerState) ChargeActivePlayers(now time.Time) []string {
	if tt == nil {
		return nil
	}
	nowMs := now.UnixMilli()
	active := make([]string, 0, len(tt.Players))
	for playerID, timer := range tt.Players {
		if timer == nil || timer.ActiveSinceMs <= 0 {
			continue
		}
		elapsed := maxInt64(0, nowMs-timer.ActiveSinceMs)
		timer.RemainingMs -= elapsed
		timer.ActiveSinceMs = nowMs
		active = append(active, playerID)
	}
	slices.Sort(active)
	return active
}

func (tt *TurnTimerState) SyncActivePlayers(activePlayerIDs []string, now time.Time) {
	if tt == nil {
		return
	}

	previouslyActive := tt.ChargeActivePlayers(now)
	previouslyActiveSet := make(map[string]bool, len(previouslyActive))
	for _, playerID := range previouslyActive {
		previouslyActiveSet[playerID] = true
	}

	nowMs := now.UnixMilli()
	nextActiveSet := make(map[string]bool, len(activePlayerIDs))
	for _, playerID := range activePlayerIDs {
		playerID = strings.TrimSpace(playerID)
		if playerID == "" {
			continue
		}
		timer := tt.Players[playerID]
		if timer == nil {
			continue
		}
		nextActiveSet[playerID] = true
	}

	for playerID, timer := range tt.Players {
		if timer == nil {
			continue
		}
		if previouslyActiveSet[playerID] && !nextActiveSet[playerID] {
			timer.RemainingMs += tt.Config.IncrementMs
			timer.ActiveSinceMs = 0
			continue
		}
		if nextActiveSet[playerID] {
			timer.ActiveSinceMs = nowMs
			continue
		}
		timer.ActiveSinceMs = 0
	}
}

func activeDecisionPlayerIDs(gs *GameState) []string {
	if gs == nil {
		return nil
	}

	if gs.Phase == PhaseFactionSelection && gs.AuctionState != nil && gs.AuctionState.Active {
		if gs.AuctionState.NominationPhase {
			if playerID := strings.TrimSpace(gs.AuctionState.GetCurrentBidder()); playerID != "" {
				return []string{playerID}
			}
			return nil
		}
		if gs.SetupMode == SetupModeFastAuction {
			return gs.AuctionState.GetPendingFastSubmitters()
		}
		if playerID := strings.TrimSpace(gs.AuctionState.GetCurrentBidder()); playerID != "" {
			return []string{playerID}
		}
		return nil
	}

	if gs.Phase == PhaseSetup && gs.SetupSubphase == SetupSubphaseBonusCards {
		if playerID := strings.TrimSpace(gs.currentSetupBonusPlayerID()); playerID != "" {
			return []string{playerID}
		}
	}

	if gs.PendingTownCultTopChoice != nil {
		return []string{gs.PendingTownCultTopChoice.PlayerID}
	}
	if townPlayer := strings.TrimSpace(gs.GetPendingTownSelectionPlayer()); townPlayer != "" {
		return []string{townPlayer}
	}
	if gs.PendingDarklingsPriestOrdination != nil {
		return []string{gs.PendingDarklingsPriestOrdination.PlayerID}
	}
	if gs.HasPendingLeechOffers() {
		if playerID := strings.TrimSpace(gs.GetNextBlockingLeechResponder()); playerID != "" {
			return []string{playerID}
		}
	}
	if gs.PendingCultistsCultSelection != nil {
		return []string{gs.PendingCultistsCultSelection.PlayerID}
	}
	if gs.PendingFavorTileSelection != nil {
		return []string{gs.PendingFavorTileSelection.PlayerID}
	}
	if gs.PendingHalflingsSpades != nil {
		return []string{gs.PendingHalflingsSpades.PlayerID}
	}
	if playerID, _ := gs.GetPendingSpadeFollowupPlayer(); strings.TrimSpace(playerID) != "" {
		return []string{playerID}
	}
	if playerID, _ := gs.GetPendingCultRewardSpadePlayer(); strings.TrimSpace(playerID) != "" {
		return []string{playerID}
	}
	if playerID := strings.TrimSpace(gs.PendingFreeActionsPlayerID); playerID != "" {
		return []string{playerID}
	}
	if playerID := strings.TrimSpace(gs.PendingTurnConfirmationPlayerID); playerID != "" {
		return []string{playerID}
	}

	switch gs.Phase {
	case PhaseFactionSelection, PhaseSetup, PhaseAction:
		if current := gs.GetCurrentPlayer(); current != nil && strings.TrimSpace(current.ID) != "" {
			return []string{current.ID}
		}
	}

	return nil
}

func serializeTurnTimer(tt *TurnTimerState, now time.Time) interface{} {
	if tt == nil {
		return nil
	}

	nowMs := now.UnixMilli()
	activePlayerIDs := make([]string, 0, len(tt.Players))
	players := make(map[string]interface{}, len(tt.Players))
	for playerID, timer := range tt.Players {
		if timer == nil {
			continue
		}
		isActive := timer.ActiveSinceMs > 0
		remainingMs := timer.RemainingMs
		if isActive {
			remainingMs -= maxInt64(0, nowMs-timer.ActiveSinceMs)
			activePlayerIDs = append(activePlayerIDs, playerID)
		}
		players[playerID] = map[string]interface{}{
			"remainingMs": remainingMs,
			"isActive":    isActive,
		}
	}
	slices.Sort(activePlayerIDs)

	return map[string]interface{}{
		"initialTimeMs":   tt.Config.InitialTimeMs,
		"incrementMs":     tt.Config.IncrementMs,
		"serverNowMs":     nowMs,
		"activePlayerIds": activePlayerIDs,
		"players":         players,
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
