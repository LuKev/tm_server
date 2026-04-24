package game

import "github.com/lukev/tm_server/internal/models"

// ResolveAutoLeechOffers applies per-player auto-leech policy until the next manual decision point.
func (gs *GameState) ResolveAutoLeechOffers() error {
	if gs == nil {
		return nil
	}
	for {
		if !gs.HasPendingLeechOffers() {
			return nil
		}
		playerID := gs.GetNextLeechResponder()
		if playerID == "" {
			return nil
		}
		offers := gs.PendingLeechOffers[playerID]
		if len(offers) == 0 {
			return nil
		}

		handled := false
		for offerIndex, offer := range offers {
			accept, auto := gs.shouldAutoResolveLeechOffer(playerID, offer)
			if !auto {
				continue
			}
			if err := executePowerLeechOffer(gs, playerID, offerIndex, accept); err != nil {
				return err
			}
			handled = true
			break
		}

		if !handled {
			return nil
		}
	}
}

func (gs *GameState) shouldAutoResolveLeechOffer(playerID string, offer *PowerLeechOffer) (accept bool, auto bool) {
	player := gs.GetPlayer(playerID)
	if player == nil || offer == nil {
		return false, false
	}

	mode := player.Options.AutoLeechMode
	if !mode.IsValid() {
		mode = LeechAutoModeOff
	}

	// Exception 1: if source is Cultists or Shapeshifters, require explicit choice.
	if source := gs.GetPlayer(offer.FromPlayerID); source != nil && source.Faction != nil {
		switch source.Faction.GetType() {
		case models.FactionCultists, models.FactionShapeshifters:
			return false, false
		}
	}

	// Exception 2: if player already passed and next-round income fully saturates bowl III anyway,
	// auto-decline all leech to avoid VP loss for no benefit.
	if mode != LeechAutoModeOff && player.HasPassed && gs.willIncomeFullySaturatePower(playerID) {
		return false, true
	}

	switch mode {
	case LeechAutoModeOff:
		return false, false
	case LeechAutoModeAccept1:
		return offer.Amount <= 1, offer.Amount <= 1
	case LeechAutoModeAccept2:
		return offer.Amount <= 2, offer.Amount <= 2
	case LeechAutoModeAccept3:
		return offer.Amount <= 3, offer.Amount <= 3
	case LeechAutoModeAccept4:
		return offer.Amount <= 4, offer.Amount <= 4
	case LeechAutoModeDeclineVP:
		if offer.Amount <= 1 {
			return true, true
		}
		return false, true
	default:
		return false, false
	}
}

func (gs *GameState) willIncomeFullySaturatePower(playerID string) bool {
	player := gs.GetPlayer(playerID)
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		return false
	}
	income, ok := gs.GetNextRoundIncomePreview(playerID)
	if !ok {
		return false
	}
	power := player.Resources.Power.Clone()
	power.GainPower(income.Power)
	return power.Bowl1 == 0 && power.Bowl2 == 0
}
