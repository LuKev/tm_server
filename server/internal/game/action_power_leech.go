package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// AcceptPowerLeechAction represents accepting a power leech offer
type AcceptPowerLeechAction struct {
	BaseAction
	OfferIndex int // Index of the offer in PendingLeechOffers
	Amount     int // Optional partial amount to accept; <=0 accepts the full offer.
}

// NewAcceptPowerLeechAction creates a new accept power leech action
func NewAcceptPowerLeechAction(playerID string, offerIndex int) *AcceptPowerLeechAction {
	return NewAcceptPowerLeechAmountAction(playerID, offerIndex, 0)
}

// NewAcceptPowerLeechAmountAction creates a new accept power leech action with an optional partial amount.
func NewAcceptPowerLeechAmountAction(playerID string, offerIndex int, amount int) *AcceptPowerLeechAction {
	return &AcceptPowerLeechAction{
		BaseAction: BaseAction{
			Type:     ActionAcceptPowerLeech,
			PlayerID: playerID,
		},
		OfferIndex: offerIndex,
		Amount:     amount,
	}
}

// Validate checks if the action is valid
func (a *AcceptPowerLeechAction) Validate(gs *GameState) error {
	return validatePowerLeechOffer(gs, a.PlayerID, a.OfferIndex)
}

// Execute performs the action
func (a *AcceptPowerLeechAction) Execute(gs *GameState) error {
	return executePowerLeechOfferAmount(gs, a.PlayerID, a.OfferIndex, true, a.Amount)
}

// DeclinePowerLeechAction represents declining a power leech offer
type DeclinePowerLeechAction struct {
	BaseAction
	OfferIndex int // Index of the offer in PendingLeechOffers
}

// NewDeclinePowerLeechAction creates a new decline power leech action
func NewDeclinePowerLeechAction(playerID string, offerIndex int) *DeclinePowerLeechAction {
	return &DeclinePowerLeechAction{
		BaseAction: BaseAction{
			Type:     ActionDeclinePowerLeech,
			PlayerID: playerID,
		},
		OfferIndex: offerIndex,
	}
}

// Validate checks if the action is valid
func (a *DeclinePowerLeechAction) Validate(gs *GameState) error {
	return validatePowerLeechOffer(gs, a.PlayerID, a.OfferIndex)
}

// Execute performs the action
func (a *DeclinePowerLeechAction) Execute(gs *GameState) error {
	return executePowerLeechOffer(gs, a.PlayerID, a.OfferIndex, false)
}

func validatePowerLeechOffer(gs *GameState, playerID string, offerIndex int) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}

	offers := gs.PendingLeechOffers[playerID]
	if len(offers) == 0 {
		return fmt.Errorf("no pending leech offers")
	}

	if offerIndex < 0 || offerIndex >= len(offers) {
		return fmt.Errorf("invalid offer index: %d", offerIndex)
	}

	return nil
}

func executePowerLeechOffer(gs *GameState, playerID string, offerIndex int, accepted bool) error {
	return executePowerLeechOfferAmount(gs, playerID, offerIndex, accepted, 0)
}

func executePowerLeechOfferAmount(gs *GameState, playerID string, offerIndex int, accepted bool, amount int) error {
	if err := validatePowerLeechOffer(gs, playerID, offerIndex); err != nil {
		return err
	}

	wasBlocking := gs.HasBlockingPendingLeechOffers()
	player := gs.GetPlayer(playerID)
	offers := gs.PendingLeechOffers[playerID]
	offer := offers[offerIndex]
	var remainder *PowerLeechOffer
	if accepted && amount > 0 && offer != nil {
		if amount > offer.Amount {
			return fmt.Errorf("cannot accept %d power from offer of %d", amount, offer.Amount)
		}
		if amount < offer.Amount {
			partial := *offer
			partial.Amount = amount
			partial.VPCost = maxInt(0, amount-1)
			remaining := *offer
			remaining.Amount -= amount
			remaining.VPCost = maxInt(0, remaining.Amount-1)
			offers[offerIndex] = &partial
			offer = offers[offerIndex]
			remainder = &remaining
		}
	}

	// Cultists leech bonus depends on whether the leeching player could actually gain
	// any power from this offer at the time they respond. Snellman logs include
	// "Decline N" rows even when the player has 0 capacity (no tokens in bowl 1/2),
	// and those forced "declines" should not trigger Cultists' bonus.
	potentialGain := 0
	if offer != nil && player != nil && player.Resources != nil && player.Resources.Power != nil {
		clone := player.Resources.Power.Clone()
		potentialGain = clone.GainPower(offer.Amount)
	}

	if accepted {
		vpCost := player.Resources.AcceptPowerLeech(offer)
		if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm && vpCost > 0 {
			vpCost--
		}
		player.VictoryPoints -= vpCost
	} else {
		player.Resources.DeclinePowerLeech(offer)
	}

	// Track for Cultists ability (if the building player is Cultists)
	if offer != nil {
		if bonus, exists := gs.PendingCultistsLeech[offer.EventID]; exists && bonus != nil && bonus.PlayerID == offer.FromPlayerID {
			bonus.ResolvedCount++
			if potentialGain > 0 {
				if accepted {
					bonus.AcceptedCount++
				} else {
					bonus.DeclinedCount++
				}
			}
		}
	}

	// Remove the accepted/declined offer, preserving any unaccepted remainder.
	updatedOffers := append(offers[:offerIndex], offers[offerIndex+1:]...)
	if remainder != nil {
		updatedOffers = append(updatedOffers[:offerIndex], append([]*PowerLeechOffer{remainder}, updatedOffers[offerIndex:]...)...)
	}
	gs.PendingLeechOffers[playerID] = updatedOffers

	// Check if all offers for this building are resolved
	if offer != nil {
		gs.ResolveCultistsLeechBonus(offer.EventID)
	}

	// Continue turn flow after the full leech queue resolves.
	if gs.AllPlayersPassed() && !gs.HasLateRoundPendingDecisions() {
		advanceAfterRoundComplete(gs)
		return nil
	}
	if wasBlocking && !gs.HasBlockingPendingLeechOffers() && gs.PendingCultistsCultSelection == nil {
		gs.NextTurn()
	}

	return nil
}

// ResolveCultistsLeechBonus checks if all leech offers for a Cultists player are resolved
// and applies the appropriate bonus (cult advance or power)
func (gs *GameState) ResolveCultistsLeechBonus(eventID int) {
	bonus, exists := gs.PendingCultistsLeech[eventID]
	if !exists {
		return
	}

	// Check if all offers have been resolved
	if bonus.ResolvedCount < bonus.OffersCreated {
		// Not all offers resolved yet
		return
	}

	// All offers resolved - apply Cultists bonus
	cultistsPlayerID := bonus.PlayerID
	player := gs.GetPlayer(cultistsPlayerID)
	if player == nil {
		delete(gs.PendingCultistsLeech, eventID)
		return
	}

	if player.Faction.GetType() != models.FactionCultists {
		delete(gs.PendingCultistsLeech, eventID)
		return
	}

	// If nobody could actually gain any power from the leeches, Snellman still logs
	// "Decline N" rows but Cultists do not receive a bonus.
	if bonus.AcceptedCount == 0 && bonus.DeclinedCount == 0 {
		delete(gs.PendingCultistsLeech, eventID)
		return
	}

	if bonus.AcceptedCount > 0 {
		// At least one opponent accepted power - Cultists must choose a cult track to advance
		// Cultists advance 1 space on cult track (if at least one opponent takes power)
		// Create pending cult track selection
		gs.PendingCultistsCultSelection = &PendingCultistsCultSelection{
			PlayerID: cultistsPlayerID,
		}
	} else {
		// All opponents declined - gain 1 power
		// Cultists gain 1 power if all opponents refuse power
		powerBonus := 1
		player.Resources.GainPower(powerBonus)
	}

	// Clean up
	delete(gs.PendingCultistsLeech, eventID)
}
