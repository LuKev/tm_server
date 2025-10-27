package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
)

// AcceptPowerLeechAction represents accepting a power leech offer
type AcceptPowerLeechAction struct {
	BaseAction
	OfferIndex int // Index of the offer in PendingLeechOffers
}

func NewAcceptPowerLeechAction(playerID string, offerIndex int) *AcceptPowerLeechAction {
	return &AcceptPowerLeechAction{
		BaseAction: BaseAction{
			Type:     ActionAcceptPowerLeech,
			PlayerID: playerID,
		},
		OfferIndex: offerIndex,
	}
}

func (a *AcceptPowerLeechAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}

	offers := gs.PendingLeechOffers[a.PlayerID]
	if len(offers) == 0 {
		return fmt.Errorf("no pending leech offers")
	}

	if a.OfferIndex < 0 || a.OfferIndex >= len(offers) {
		return fmt.Errorf("invalid offer index: %d", a.OfferIndex)
	}

	return nil
}

func (a *AcceptPowerLeechAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	offers := gs.PendingLeechOffers[a.PlayerID]
	offer := offers[a.OfferIndex]

	// Accept the power leech
	vpCost := player.Resources.AcceptPowerLeech(offer)
	player.VictoryPoints -= vpCost

	// Track for Cultists ability (if the building player is Cultists)
	if bonus, exists := gs.PendingCultistsLeech[offer.FromPlayerID]; exists {
		bonus.AcceptedCount++
	}

	// Remove the offer
	gs.PendingLeechOffers[a.PlayerID] = append(offers[:a.OfferIndex], offers[a.OfferIndex+1:]...)

	// Check if all offers for this building are resolved
	gs.ResolveCultistsLeechBonus(offer.FromPlayerID)

	return nil
}

// DeclinePowerLeechAction represents declining a power leech offer
type DeclinePowerLeechAction struct {
	BaseAction
	OfferIndex int // Index of the offer in PendingLeechOffers
}

func NewDeclinePowerLeechAction(playerID string, offerIndex int) *DeclinePowerLeechAction {
	return &DeclinePowerLeechAction{
		BaseAction: BaseAction{
			Type:     ActionDeclinePowerLeech,
			PlayerID: playerID,
		},
		OfferIndex: offerIndex,
	}
}

func (a *DeclinePowerLeechAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}

	offers := gs.PendingLeechOffers[a.PlayerID]
	if len(offers) == 0 {
		return fmt.Errorf("no pending leech offers")
	}

	if a.OfferIndex < 0 || a.OfferIndex >= len(offers) {
		return fmt.Errorf("invalid offer index: %d", a.OfferIndex)
	}

	return nil
}

func (a *DeclinePowerLeechAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	offers := gs.PendingLeechOffers[a.PlayerID]
	offer := offers[a.OfferIndex]

	// Decline the power leech (no effect on resources)
	player.Resources.DeclinePowerLeech(offer)

	// Track for Cultists ability (if the building player is Cultists)
	if bonus, exists := gs.PendingCultistsLeech[offer.FromPlayerID]; exists {
		bonus.DeclinedCount++
	}

	// Remove the offer
	gs.PendingLeechOffers[a.PlayerID] = append(offers[:a.OfferIndex], offers[a.OfferIndex+1:]...)

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

	cultists, ok := player.Faction.(*factions.Cultists)
	if !ok {
		delete(gs.PendingCultistsLeech, cultistsPlayerID)
		return
	}

	if bonus.AcceptedCount > 0 {
		// At least one opponent accepted power - advance 1 space on a cult track
		cultAdvance := cultists.GetCultAdvanceFromPowerLeech()
		// Note: Cult track selection when multiple opponents accept is a future enhancement
		// Currently the bonus is acknowledged but manual selection is needed
		_ = cultAdvance
	} else {
		// All opponents declined - gain 1 power
		powerBonus := cultists.GetPowerIfAllRefuse()
		player.Resources.Power.Bowl1 += powerBonus
	}

	// Clean up
	delete(gs.PendingCultistsLeech, cultistsPlayerID)
}
