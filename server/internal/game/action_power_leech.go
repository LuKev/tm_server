package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// AcceptPowerLeechAction represents accepting a power leech offer
type AcceptPowerLeechAction struct {
	BaseAction
	OfferIndex int // Index of the offer in PendingLeechOffers
}

// NewAcceptPowerLeechAction creates a new accept power leech action
func NewAcceptPowerLeechAction(playerID string, offerIndex int) *AcceptPowerLeechAction {
	return &AcceptPowerLeechAction{
		BaseAction: BaseAction{
			Type:     ActionAcceptPowerLeech,
			PlayerID: playerID,
		},
		OfferIndex: offerIndex,
	}
}

// Validate checks if the action is valid
func (a *AcceptPowerLeechAction) Validate(gs *GameState) error {
	return validatePowerLeechOffer(gs, a.PlayerID, a.OfferIndex)
}

// Execute performs the action
func (a *AcceptPowerLeechAction) Execute(gs *GameState) error {
	return executePowerLeechOffer(gs, a.PlayerID, a.OfferIndex, true)
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
	if err := validatePowerLeechOffer(gs, playerID, offerIndex); err != nil {
		return err
	}

	player := gs.GetPlayer(playerID)
	offers := gs.PendingLeechOffers[playerID]
	offer := offers[offerIndex]

	if accepted {
		vpCost := player.Resources.AcceptPowerLeech(offer)
		player.VictoryPoints -= vpCost
	} else {
		player.Resources.DeclinePowerLeech(offer)
	}

	// Track for Cultists ability (if the building player is Cultists)
	if bonus, exists := gs.PendingCultistsLeech[offer.FromPlayerID]; exists {
		if accepted {
			bonus.AcceptedCount++
		} else {
			bonus.DeclinedCount++
		}
	}

	// Remove the offer
	gs.PendingLeechOffers[playerID] = append(offers[:offerIndex], offers[offerIndex+1:]...)

	// Check if all offers for this building are resolved
	gs.ResolveCultistsLeechBonus(offer.FromPlayerID)

	return nil
}

// ResolveCultistsLeechBonus checks if all leech offers for a Cultists player are resolved
// and applies the appropriate bonus (cult advance or power)
func (gs *GameState) ResolveCultistsLeechBonus(cultistsPlayerID string) {
	bonus, exists := gs.PendingCultistsLeech[cultistsPlayerID]
	if !exists {
		return
	}

	// Check if all offers have been resolved
	totalResolved := bonus.AcceptedCount + bonus.DeclinedCount
	if totalResolved < bonus.OffersCreated {
		// Not all offers resolved yet
		return
	}

	// All offers resolved - apply Cultists bonus
	player := gs.GetPlayer(cultistsPlayerID)
	if player == nil {
		delete(gs.PendingCultistsLeech, cultistsPlayerID)
		return
	}

	if player.Faction.GetType() != models.FactionCultists {
		delete(gs.PendingCultistsLeech, cultistsPlayerID)
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
	delete(gs.PendingCultistsLeech, cultistsPlayerID)
}
