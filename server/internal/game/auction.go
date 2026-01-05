package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// Standard Auction System for Terra Mystica
// Players bid on factions by reducing their starting VP (from 40)
// Turn order is determined by the order factions were nominated

// AuctionState tracks the current state of the auction
type AuctionState struct {
	Active              bool                          // Whether auction is currently active
	NominationOrder     []models.FactionType          // Order factions were nominated (determines turn order)
	CurrentBids         map[models.FactionType]int    // Current VP bid for each faction (40 - bid = starting VP)
	FactionHolders      map[models.FactionType]string // Which player currently holds each faction
	PlayerHasFaction    map[string]bool               // Which players currently hold a faction
	SeatOrder           []string                      // Player IDs in seat order
	CurrentBidderIndex  int                           // Index into SeatOrder for current bidder
	NominationPhase     bool                          // True during nomination, false during bidding
	NominationsComplete int                           // Number of factions nominated so far
}

// NewAuctionState creates a new auction state
func NewAuctionState(seatOrder []string) *AuctionState {
	return &AuctionState{
		Active:              true,
		NominationOrder:     []models.FactionType{},
		CurrentBids:         make(map[models.FactionType]int),
		FactionHolders:      make(map[models.FactionType]string),
		PlayerHasFaction:    make(map[string]bool),
		SeatOrder:           seatOrder,
		CurrentBidderIndex:  0,
		NominationPhase:     true,
		NominationsComplete: 0,
	}
}

// NominateFaction nominates a faction for auction (during nomination phase)
func (as *AuctionState) NominateFaction(playerID string, faction models.FactionType) error {
	if !as.Active {
		return fmt.Errorf("auction is not active")
	}

	if !as.NominationPhase {
		return fmt.Errorf("nomination phase is over")
	}

	// Verify it's this player's turn to nominate
	if as.SeatOrder[as.NominationsComplete] != playerID {
		return fmt.Errorf("not your turn to nominate")
	}

	// Check if faction already nominated
	for _, f := range as.NominationOrder {
		if f == faction {
			return fmt.Errorf("faction already nominated")
		}
	}

	// Check if a faction of the same color has already been nominated
	factionColor := faction.GetFactionColor()
	for _, f := range as.NominationOrder {
		if f.GetFactionColor() == factionColor {
			return fmt.Errorf("a faction of this color has already been nominated")
		}
	}

	// Add to nomination order
	as.NominationOrder = append(as.NominationOrder, faction)
	as.CurrentBids[faction] = 0 // Starting bid is 0 (40 VP)
	as.NominationsComplete++

	// Check if all players have nominated
	if as.NominationsComplete == len(as.SeatOrder) {
		as.NominationPhase = false
		as.CurrentBidderIndex = 0 // Start bidding from first player
	}

	return nil
}

// PlaceBid places a bid on a faction (during bidding phase)
func (as *AuctionState) PlaceBid(playerID string, faction models.FactionType, vpReduction int) error {
	if err := as.validateBid(playerID, faction, vpReduction); err != nil {
		return err
	}

	as.executeBid(playerID, faction, vpReduction)
	return nil
}

func (as *AuctionState) validateBid(playerID string, faction models.FactionType, vpReduction int) error {
	if !as.Active {
		return fmt.Errorf("auction is not active")
	}

	if as.NominationPhase {
		return fmt.Errorf("still in nomination phase")
	}

	// Check if player already has a faction (skip them)
	if as.PlayerHasFaction[playerID] {
		return fmt.Errorf("you already have a faction")
	}

	// Verify it's this player's turn
	if as.SeatOrder[as.CurrentBidderIndex] != playerID {
		return fmt.Errorf("not your turn to bid")
	}

	// Check if faction was nominated
	currentBid, exists := as.CurrentBids[faction]
	if !exists {
		return fmt.Errorf("faction not in auction")
	}

	// Validate bid
	if vpReduction < 0 || vpReduction > 40 {
		return fmt.Errorf("VP reduction must be between 0 and 40")
	}

	// If faction is held by another player, must bid at least 1 more VP
	if holder, held := as.FactionHolders[faction]; held && holder != "" {
		if vpReduction <= currentBid {
			return fmt.Errorf("must reduce VP by at least 1 more than current bid (%d)", currentBid)
		}
	}
	return nil
}

func (as *AuctionState) executeBid(playerID string, faction models.FactionType, vpReduction int) {
	// Remove player's previous faction if they had one
	for f, holder := range as.FactionHolders {
		if holder == playerID {
			delete(as.FactionHolders, f)
			as.PlayerHasFaction[playerID] = false
		}
	}

	// Remove previous holder of this faction (if being overbid)
	if previousHolder, exists := as.FactionHolders[faction]; exists && previousHolder != "" {
		as.PlayerHasFaction[previousHolder] = false
	}

	// Place the bid
	as.CurrentBids[faction] = vpReduction
	as.FactionHolders[faction] = playerID
	as.PlayerHasFaction[playerID] = true

	// Move to next bidder (skip players who have factions)
	as.advanceToNextBidder()

	// Check if auction is complete (all players have factions)
	if as.isAuctionComplete() {
		as.Active = false
	}
}

// advanceToNextBidder moves to the next player who doesn't have a faction
func (as *AuctionState) advanceToNextBidder() {
	startIndex := as.CurrentBidderIndex

	for {
		as.CurrentBidderIndex = (as.CurrentBidderIndex + 1) % len(as.SeatOrder)

		// If we've gone full circle, auction might be complete
		if as.CurrentBidderIndex == startIndex {
			break
		}

		// Found a player without a faction
		playerID := as.SeatOrder[as.CurrentBidderIndex]
		if !as.PlayerHasFaction[playerID] {
			break
		}
	}
}

// isAuctionComplete checks if all players have factions
func (as *AuctionState) isAuctionComplete() bool {
	for _, playerID := range as.SeatOrder {
		if !as.PlayerHasFaction[playerID] {
			return false
		}
	}
	return true
}

// GetCurrentBidder returns the player ID of the current bidder
func (as *AuctionState) GetCurrentBidder() string {
	if as.NominationPhase {
		if as.NominationsComplete < len(as.SeatOrder) {
			return as.SeatOrder[as.NominationsComplete]
		}
		return ""
	}

	if as.CurrentBidderIndex < len(as.SeatOrder) {
		return as.SeatOrder[as.CurrentBidderIndex]
	}
	return ""
}

// GetStartingVP returns the starting VP for a faction based on the final bid
func (as *AuctionState) GetStartingVP(faction models.FactionType) int {
	bid, exists := as.CurrentBids[faction]
	if !exists {
		return 40 // Default if not in auction
	}
	return 40 - bid
}

// GetTurnOrder returns the turn order based on nomination order
func (as *AuctionState) GetTurnOrder() []string {
	turnOrder := make([]string, 0, len(as.NominationOrder))

	for _, faction := range as.NominationOrder {
		if playerID, exists := as.FactionHolders[faction]; exists {
			turnOrder = append(turnOrder, playerID)
		}
	}

	return turnOrder
}

// GetPlayerFaction returns the faction a player won in the auction
func (as *AuctionState) GetPlayerFaction(playerID string) (models.FactionType, bool) {
	for faction, holder := range as.FactionHolders {
		if holder == playerID {
			return faction, true
		}
	}
	return 0, false
}

// GetAuctionSummary returns a summary of the auction results
func (as *AuctionState) GetAuctionSummary() map[string]AuctionResult {
	results := make(map[string]AuctionResult)

	for _, playerID := range as.SeatOrder {
		if faction, ok := as.GetPlayerFaction(playerID); ok {
			results[playerID] = AuctionResult{
				PlayerID:   playerID,
				Faction:    faction,
				StartingVP: as.GetStartingVP(faction),
				VPBid:      as.CurrentBids[faction],
			}
		}
	}

	return results
}

// AuctionResult represents the result of an auction for a player
type AuctionResult struct {
	PlayerID   string
	Faction    models.FactionType
	StartingVP int
	VPBid      int
}

// GameSetupOptions contains options for game setup
type GameSetupOptions struct {
	UseAuction  bool     // Whether to use standard auction
	PlayerCount int      // Number of players (2-5)
	SeatOrder   []string // Player IDs in seat order (for auction)
}

// ValidateSetupOptions validates game setup options
func ValidateSetupOptions(opts GameSetupOptions) error {
	if opts.PlayerCount < 2 || opts.PlayerCount > 5 {
		return fmt.Errorf("player count must be between 2 and 5")
	}

	if opts.UseAuction {
		if len(opts.SeatOrder) != opts.PlayerCount {
			return fmt.Errorf("seat order must match player count")
		}
	}

	return nil
}
