package game

import (
	"fmt"
	"slices"

	"github.com/lukev/tm_server/internal/models"
)

// Standard Auction System for Terra Mystica
// Players bid on factions by reducing their starting VP (from 40)
// Turn order is determined by the order factions were nominated

// AuctionState tracks the current state of the auction
type AuctionState struct {
	Active              bool                          // Whether auction is currently active
	Mode                SetupMode                     // Setup mode (auction or fast_auction)
	NominationOrder     []models.FactionType          // Order factions were nominated (determines turn order)
	CurrentBids         map[models.FactionType]int    // Current VP bid for each faction (40 - bid = starting VP)
	FactionHolders      map[models.FactionType]string // Which player currently holds each faction
	PlayerHasFaction    map[string]bool               // Which players currently hold a faction
	SeatOrder           []string                      // Player IDs in seat order
	CurrentBidderIndex  int                           // Index into SeatOrder for current bidder
	NominationPhase     bool                          // True during nomination, false during bidding
	NominationsComplete int                           // Number of factions nominated so far
	FastBids            map[string]map[models.FactionType]int
	FastSubmitted       map[string]bool
}

// NewAuctionState creates a new auction state
func NewAuctionState(seatOrder []string) *AuctionState {
	return NewAuctionStateWithMode(seatOrder, SetupModeAuction)
}

// NewAuctionStateWithMode creates a new auction state for auction or fast_auction setup modes.
func NewAuctionStateWithMode(seatOrder []string, mode SetupMode) *AuctionState {
	return &AuctionState{
		Active:              true,
		Mode:                mode,
		NominationOrder:     []models.FactionType{},
		CurrentBids:         make(map[models.FactionType]int),
		FactionHolders:      make(map[models.FactionType]string),
		PlayerHasFaction:    make(map[string]bool),
		SeatOrder:           seatOrder,
		CurrentBidderIndex:  0,
		NominationPhase:     true,
		NominationsComplete: 0,
		FastBids:            make(map[string]map[models.FactionType]int),
		FastSubmitted:       make(map[string]bool),
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
	if as.Mode != SetupModeAuction {
		return fmt.Errorf("place bid is only available in regular auction mode")
	}

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
	as.advanceToNextRegularBidder()

	// Check if auction is complete (all players have factions)
	if as.isAuctionComplete() {
		as.Active = false
	}
}

// advanceToNextRegularBidder moves to the next player who doesn't currently hold a faction.
func (as *AuctionState) advanceToNextRegularBidder() {
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

// SubmitFastBids submits one sealed bid vector for the active player in fast auction mode.
func (as *AuctionState) SubmitFastBids(playerID string, bids map[models.FactionType]int) error {
	if !as.Active {
		return fmt.Errorf("auction is not active")
	}
	if as.Mode != SetupModeFastAuction {
		return fmt.Errorf("fast bid submission is only available in fast auction mode")
	}
	if as.NominationPhase {
		return fmt.Errorf("still in nomination phase")
	}
	if as.SeatOrder[as.CurrentBidderIndex] != playerID {
		return fmt.Errorf("not your turn to submit fast auction bids")
	}
	if as.FastSubmitted[playerID] {
		return fmt.Errorf("player already submitted fast auction bids")
	}
	if len(as.NominationOrder) == 0 {
		return fmt.Errorf("no nominated factions available for fast auction bids")
	}
	if len(bids) != len(as.NominationOrder) {
		return fmt.Errorf("must submit bids for exactly %d nominated factions", len(as.NominationOrder))
	}

	playerBids := make(map[models.FactionType]int, len(as.NominationOrder))
	for _, faction := range as.NominationOrder {
		value, ok := bids[faction]
		if !ok {
			return fmt.Errorf("missing fast auction bid for faction %s", faction.String())
		}
		if value < 0 || value > 40 {
			return fmt.Errorf("VP reduction for faction %s must be between 0 and 40", faction.String())
		}
		playerBids[faction] = value
	}

	as.FastBids[playerID] = playerBids
	as.FastSubmitted[playerID] = true
	as.advanceToNextFastBidder()

	if as.allFastBidsSubmitted() {
		if err := as.resolveFastAuctionAssignments(); err != nil {
			return err
		}
		as.Active = false
	}

	return nil
}

func (as *AuctionState) advanceToNextFastBidder() {
	startIndex := as.CurrentBidderIndex

	for {
		as.CurrentBidderIndex = (as.CurrentBidderIndex + 1) % len(as.SeatOrder)
		if as.CurrentBidderIndex == startIndex {
			return
		}
		candidate := as.SeatOrder[as.CurrentBidderIndex]
		if !as.FastSubmitted[candidate] {
			return
		}
	}
}

func (as *AuctionState) allFastBidsSubmitted() bool {
	for _, playerID := range as.SeatOrder {
		if !as.FastSubmitted[playerID] {
			return false
		}
	}
	return true
}

func (as *AuctionState) resolveFastAuctionAssignments() error {
	if len(as.NominationOrder) != len(as.SeatOrder) {
		return fmt.Errorf("cannot resolve fast auction: nominations (%d) must match players (%d)", len(as.NominationOrder), len(as.SeatOrder))
	}

	assignment := make([]models.FactionType, len(as.SeatOrder))
	available := slices.Clone(as.NominationOrder)

	bestFound := false
	bestScore := -1
	bestSeatBids := make([]int, 0, len(as.SeatOrder))
	bestFactionIdx := make([]int, 0, len(as.SeatOrder))
	bestAssignment := make([]models.FactionType, 0, len(as.SeatOrder))

	var search func(playerIndex int)
	search = func(playerIndex int) {
		if playerIndex == len(as.SeatOrder) {
			score := 0
			seatBids := make([]int, len(as.SeatOrder))
			factionIdx := make([]int, len(as.SeatOrder))
			for i, playerID := range as.SeatOrder {
				faction := assignment[i]
				bid := as.FastBids[playerID][faction]
				seatBids[i] = bid
				score += bid
				factionIdx[i] = slices.Index(as.NominationOrder, faction)
			}

			if !bestFound || score > bestScore || (score == bestScore && isLexicographicallyBetter(seatBids, bestSeatBids)) || (score == bestScore && slices.Equal(seatBids, bestSeatBids) && isLexicographicallySmaller(factionIdx, bestFactionIdx)) {
				bestFound = true
				bestScore = score
				bestSeatBids = slices.Clone(seatBids)
				bestFactionIdx = slices.Clone(factionIdx)
				bestAssignment = slices.Clone(assignment)
			}
			return
		}

		for i, faction := range available {
			assignment[playerIndex] = faction

			nextAvailable := make([]models.FactionType, 0, len(available)-1)
			nextAvailable = append(nextAvailable, available[:i]...)
			nextAvailable = append(nextAvailable, available[i+1:]...)
			prevAvailable := available
			available = nextAvailable
			search(playerIndex + 1)
			available = prevAvailable
		}
	}

	search(0)
	if !bestFound {
		return fmt.Errorf("unable to resolve fast auction assignments")
	}

	for i, playerID := range as.SeatOrder {
		faction := bestAssignment[i]
		bid := as.FastBids[playerID][faction]
		as.CurrentBids[faction] = bid
		as.FactionHolders[faction] = playerID
		as.PlayerHasFaction[playerID] = true
	}

	return nil
}

func isLexicographicallyBetter(left, right []int) bool {
	for i := 0; i < len(left) && i < len(right); i++ {
		if left[i] == right[i] {
			continue
		}
		return left[i] > right[i]
	}
	return len(left) > len(right)
}

func isLexicographicallySmaller(left, right []int) bool {
	for i := 0; i < len(left) && i < len(right); i++ {
		if left[i] == right[i] {
			continue
		}
		return left[i] < right[i]
	}
	return len(left) < len(right)
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
	SetupMode   SetupMode
	UseAuction  bool     // Whether to use standard auction
	PlayerCount int      // Number of players (2-5)
	SeatOrder   []string // Player IDs in seat order (for auction)
}

// ValidateSetupOptions validates game setup options
func ValidateSetupOptions(opts GameSetupOptions) error {
	if opts.PlayerCount < 2 || opts.PlayerCount > 5 {
		return fmt.Errorf("player count must be between 2 and 5")
	}

	useAuction := opts.UseAuction || opts.SetupMode == SetupModeAuction || opts.SetupMode == SetupModeFastAuction
	if useAuction {
		if len(opts.SeatOrder) != opts.PlayerCount {
			return fmt.Errorf("seat order must match player count")
		}
	}

	return nil
}
