package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// AuctionNominateFactionAction nominates a faction during auction setup.
type AuctionNominateFactionAction struct {
	BaseAction
	FactionType models.FactionType
}

func NewAuctionNominateFactionAction(playerID string, faction models.FactionType) *AuctionNominateFactionAction {
	return &AuctionNominateFactionAction{
		BaseAction:  BaseAction{Type: ActionAuctionNominateFaction, PlayerID: playerID},
		FactionType: faction,
	}
}

func (a *AuctionNominateFactionAction) Validate(gs *GameState) error {
	if gs.Phase != PhaseFactionSelection {
		return fmt.Errorf("not in faction selection phase")
	}
	if gs.SetupMode != SetupModeAuction && gs.SetupMode != SetupModeFastAuction {
		return fmt.Errorf("auction nomination unavailable for setup mode %s", gs.SetupMode)
	}
	if gs.AuctionState == nil || !gs.AuctionState.Active {
		return fmt.Errorf("auction is not active")
	}
	if !gs.AuctionState.NominationPhase {
		return fmt.Errorf("auction nomination phase is complete")
	}
	if !isValidFaction(a.FactionType) {
		return fmt.Errorf("invalid faction type: %s", a.FactionType)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != a.PlayerID {
		return fmt.Errorf("not your turn")
	}
	return nil
}

func (a *AuctionNominateFactionAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	if err := gs.AuctionState.NominateFaction(a.PlayerID, a.FactionType); err != nil {
		return err
	}
	syncCurrentPlayerToAuction(gs)
	return nil
}

// AuctionPlaceBidAction places one bid in regular auction mode.
type AuctionPlaceBidAction struct {
	BaseAction
	FactionType models.FactionType
	VPReduction int
}

func NewAuctionPlaceBidAction(playerID string, faction models.FactionType, vpReduction int) *AuctionPlaceBidAction {
	return &AuctionPlaceBidAction{
		BaseAction:  BaseAction{Type: ActionAuctionPlaceBid, PlayerID: playerID},
		FactionType: faction,
		VPReduction: vpReduction,
	}
}

func (a *AuctionPlaceBidAction) Validate(gs *GameState) error {
	if gs.Phase != PhaseFactionSelection {
		return fmt.Errorf("not in faction selection phase")
	}
	if gs.SetupMode != SetupModeAuction {
		return fmt.Errorf("regular auction bidding unavailable for setup mode %s", gs.SetupMode)
	}
	if gs.AuctionState == nil || !gs.AuctionState.Active {
		return fmt.Errorf("auction is not active")
	}
	if gs.AuctionState.NominationPhase {
		return fmt.Errorf("auction is still in nomination phase")
	}
	if !isValidFaction(a.FactionType) {
		return fmt.Errorf("invalid faction type: %s", a.FactionType)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != a.PlayerID {
		return fmt.Errorf("not your turn")
	}
	return nil
}

func (a *AuctionPlaceBidAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	if err := gs.AuctionState.PlaceBid(a.PlayerID, a.FactionType, a.VPReduction); err != nil {
		return err
	}
	if gs.AuctionState.Active {
		syncCurrentPlayerToAuction(gs)
		return nil
	}
	return finalizeAuctionAndStartSetup(gs)
}

// FastAuctionSubmitBidsAction submits one sealed bid vector for fast auction.
type FastAuctionSubmitBidsAction struct {
	BaseAction
	Bids map[models.FactionType]int
}

func NewFastAuctionSubmitBidsAction(playerID string, bids map[models.FactionType]int) *FastAuctionSubmitBidsAction {
	return &FastAuctionSubmitBidsAction{
		BaseAction: BaseAction{Type: ActionFastAuctionSubmitBids, PlayerID: playerID},
		Bids:       bids,
	}
}

func (a *FastAuctionSubmitBidsAction) Validate(gs *GameState) error {
	if gs.Phase != PhaseFactionSelection {
		return fmt.Errorf("not in faction selection phase")
	}
	if gs.SetupMode != SetupModeFastAuction {
		return fmt.Errorf("fast auction bidding unavailable for setup mode %s", gs.SetupMode)
	}
	if gs.AuctionState == nil || !gs.AuctionState.Active {
		return fmt.Errorf("auction is not active")
	}
	if gs.AuctionState.NominationPhase {
		return fmt.Errorf("auction is still in nomination phase")
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != a.PlayerID {
		return fmt.Errorf("not your turn")
	}
	if len(a.Bids) == 0 {
		return fmt.Errorf("missing fast auction bid payload")
	}
	return nil
}

func (a *FastAuctionSubmitBidsAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	if err := gs.AuctionState.SubmitFastBids(a.PlayerID, a.Bids); err != nil {
		return err
	}
	if gs.AuctionState.Active {
		syncCurrentPlayerToAuction(gs)
		return nil
	}
	return finalizeAuctionAndStartSetup(gs)
}

func syncCurrentPlayerToAuction(gs *GameState) {
	if gs == nil || gs.AuctionState == nil {
		return
	}
	currentBidder := gs.AuctionState.GetCurrentBidder()
	if currentBidder == "" {
		return
	}
	for i, playerID := range gs.TurnOrder {
		if playerID == currentBidder {
			gs.CurrentPlayerIndex = i
			return
		}
	}
}

func finalizeAuctionAndStartSetup(gs *GameState) error {
	if gs == nil || gs.AuctionState == nil {
		return fmt.Errorf("auction state is not initialized")
	}
	results := gs.AuctionState.GetAuctionSummary()
	if len(results) != len(gs.TurnOrder) {
		return fmt.Errorf("auction incomplete: expected %d winners, got %d", len(gs.TurnOrder), len(results))
	}

	for _, playerID := range gs.TurnOrder {
		result, ok := results[playerID]
		if !ok {
			return fmt.Errorf("missing auction result for player %s", playerID)
		}
		if err := assignFactionToPlayer(gs, playerID, result.Faction, result.StartingVP); err != nil {
			return err
		}
	}

	newTurnOrder := gs.AuctionState.GetTurnOrder()
	if len(newTurnOrder) != len(gs.TurnOrder) {
		return fmt.Errorf("auction turn order mismatch: expected %d, got %d", len(gs.TurnOrder), len(newTurnOrder))
	}
	gs.TurnOrder = newTurnOrder
	gs.CurrentPlayerIndex = 0
	gs.InitializeSetupSequence()
	return nil
}
