package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestAuction_NominationPhase(t *testing.T) {
	seatOrder := []string{"player1", "player2", "player3"}
	auction := NewAuctionState(seatOrder)

	// Player 1 nominates Nomads
	err := auction.NominateFaction("player1", models.FactionNomads)
	if err != nil {
		t.Fatalf("failed to nominate: %v", err)
	}

	// Verify nomination order
	if len(auction.NominationOrder) != 1 {
		t.Errorf("expected 1 nomination, got %d", len(auction.NominationOrder))
	}
	if auction.NominationOrder[0] != models.FactionNomads {
		t.Errorf("expected Nomads, got %v", auction.NominationOrder[0])
	}

	// Still in nomination phase
	if !auction.NominationPhase {
		t.Error("should still be in nomination phase")
	}

	// Player 2 nominates Witches
	err = auction.NominateFaction("player2", models.FactionWitches)
	if err != nil {
		t.Fatalf("failed to nominate: %v", err)
	}

	// Player 3 nominates Engineers
	err = auction.NominateFaction("player3", models.FactionEngineers)
	if err != nil {
		t.Fatalf("failed to nominate: %v", err)
	}

	// Should now be in bidding phase
	if auction.NominationPhase {
		t.Error("should be in bidding phase")
	}

	// Verify all nominations
	if len(auction.NominationOrder) != 3 {
		t.Errorf("expected 3 nominations, got %d", len(auction.NominationOrder))
	}
}

func TestAuction_CannotNominateSameFactionTwice(t *testing.T) {
	seatOrder := []string{"player1", "player2"}
	auction := NewAuctionState(seatOrder)

	auction.NominateFaction("player1", models.FactionNomads)

	// Player 2 tries to nominate same faction
	err := auction.NominateFaction("player2", models.FactionNomads)
	if err == nil {
		t.Error("should not allow duplicate faction nomination")
	}
}

func TestAuction_BiddingPhase_TakeUnclaimedFaction(t *testing.T) {
	seatOrder := []string{"player1", "player2", "player3"}
	auction := NewAuctionState(seatOrder)

	// Complete nominations
	auction.NominateFaction("player1", models.FactionNomads)
	auction.NominateFaction("player2", models.FactionWitches)
	auction.NominateFaction("player3", models.FactionEngineers)

	// Player 1 takes Nomads at 40 VP (bid 0)
	err := auction.PlaceBid("player1", models.FactionNomads, 0)
	if err != nil {
		t.Fatalf("failed to place bid: %v", err)
	}

	// Verify player has faction
	if !auction.PlayerHasFaction["player1"] {
		t.Error("player1 should have a faction")
	}
	if auction.FactionHolders[models.FactionNomads] != "player1" {
		t.Error("player1 should hold Nomads")
	}
	if auction.GetStartingVP(models.FactionNomads) != 40 {
		t.Errorf("expected 40 VP, got %d", auction.GetStartingVP(models.FactionNomads))
	}
}

func TestAuction_BiddingPhase_Overbid(t *testing.T) {
	seatOrder := []string{"player1", "player2", "player3"}
	auction := NewAuctionState(seatOrder)

	// Complete nominations
	auction.NominateFaction("player1", models.FactionNomads)
	auction.NominateFaction("player2", models.FactionWitches)
	auction.NominateFaction("player3", models.FactionEngineers)

	// Player 1 takes Nomads at 40 VP
	auction.PlaceBid("player1", models.FactionNomads, 0)

	// Player 2 takes Witches at 38 VP (bid 2)
	auction.PlaceBid("player2", models.FactionWitches, 2)

	// Player 3 overbids Nomads at 37 VP (bid 3)
	err := auction.PlaceBid("player3", models.FactionNomads, 3)
	if err != nil {
		t.Fatalf("failed to overbid: %v", err)
	}

	// Player 3 should now hold Nomads
	if auction.FactionHolders[models.FactionNomads] != "player3" {
		t.Error("player3 should hold Nomads after overbid")
	}

	// Player 1 should no longer have a faction
	if auction.PlayerHasFaction["player1"] {
		t.Error("player1 should not have a faction after being overbid")
	}

	// Starting VP should be 37
	if auction.GetStartingVP(models.FactionNomads) != 37 {
		t.Errorf("expected 37 VP, got %d", auction.GetStartingVP(models.FactionNomads))
	}
}

func TestAuction_MustOverbidByAtLeastOne(t *testing.T) {
	seatOrder := []string{"player1", "player2"}
	auction := NewAuctionState(seatOrder)

	auction.NominateFaction("player1", models.FactionNomads)
	auction.NominateFaction("player2", models.FactionWitches)

	// Player 1 bids 2 VP on Nomads
	auction.PlaceBid("player1", models.FactionNomads, 2)

	// Player 2 tries to bid same amount (should fail)
	err := auction.PlaceBid("player2", models.FactionNomads, 2)
	if err == nil {
		t.Error("should not allow equal bid")
	}

	// Player 2 tries to bid less (should fail)
	err = auction.PlaceBid("player2", models.FactionNomads, 1)
	if err == nil {
		t.Error("should not allow lower bid")
	}

	// Player 2 bids 3 VP (should succeed)
	err = auction.PlaceBid("player2", models.FactionNomads, 3)
	if err != nil {
		t.Errorf("should allow higher bid: %v", err)
	}
}

func TestAuction_SkipPlayersWithFactions(t *testing.T) {
	seatOrder := []string{"player1", "player2", "player3"}
	auction := NewAuctionState(seatOrder)

	auction.NominateFaction("player1", models.FactionNomads)
	auction.NominateFaction("player2", models.FactionWitches)
	auction.NominateFaction("player3", models.FactionEngineers)

	// Player 1 takes Nomads
	auction.PlaceBid("player1", models.FactionNomads, 0)

	// Current bidder should skip player 1 and go to player 2
	currentBidder := auction.GetCurrentBidder()
	if currentBidder != "player2" {
		t.Errorf("expected player2, got %s", currentBidder)
	}

	// Player 2 takes Witches
	auction.PlaceBid("player2", models.FactionWitches, 0)

	// Current bidder should skip players 1 and 2, go to player 3
	currentBidder = auction.GetCurrentBidder()
	if currentBidder != "player3" {
		t.Errorf("expected player3, got %s", currentBidder)
	}
}

func TestAuction_AuctionComplete(t *testing.T) {
	seatOrder := []string{"player1", "player2", "player3"}
	auction := NewAuctionState(seatOrder)

	auction.NominateFaction("player1", models.FactionNomads)
	auction.NominateFaction("player2", models.FactionWitches)
	auction.NominateFaction("player3", models.FactionEngineers)

	// All players take factions
	auction.PlaceBid("player1", models.FactionNomads, 0)
	auction.PlaceBid("player2", models.FactionWitches, 1)
	auction.PlaceBid("player3", models.FactionEngineers, 2)

	// Auction should be complete
	if auction.Active {
		t.Error("auction should be complete")
	}
}

func TestAuction_TurnOrderByNominationOrder(t *testing.T) {
	seatOrder := []string{"playerA", "playerB", "playerC"}
	auction := NewAuctionState(seatOrder)

	// Nominations (determines turn order)
	auction.NominateFaction("playerA", models.FactionNomads)    // 1st in turn order
	auction.NominateFaction("playerB", models.FactionWitches)   // 2nd in turn order
	auction.NominateFaction("playerC", models.FactionEngineers) // 3rd in turn order

	// Bidding (playerB wins Nomads, playerC wins Witches, playerA wins Engineers)
	auction.PlaceBid("playerA", models.FactionNomads, 0)
	auction.PlaceBid("playerB", models.FactionNomads, 1) // Overbids playerA
	auction.PlaceBid("playerC", models.FactionWitches, 0)
	auction.PlaceBid("playerA", models.FactionEngineers, 0)

	// Get turn order
	turnOrder := auction.GetTurnOrder()

	// Turn order should be: playerB (Nomads), playerC (Witches), playerA (Engineers)
	// Based on nomination order, not seat order
	if len(turnOrder) != 3 {
		t.Fatalf("expected 3 players in turn order, got %d", len(turnOrder))
	}
	if turnOrder[0] != "playerB" {
		t.Errorf("1st in turn order should be playerB (won Nomads), got %s", turnOrder[0])
	}
	if turnOrder[1] != "playerC" {
		t.Errorf("2nd in turn order should be playerC (won Witches), got %s", turnOrder[1])
	}
	if turnOrder[2] != "playerA" {
		t.Errorf("3rd in turn order should be playerA (won Engineers), got %s", turnOrder[2])
	}
}

func TestAuction_GetAuctionSummary(t *testing.T) {
	seatOrder := []string{"player1", "player2"}
	auction := NewAuctionState(seatOrder)

	auction.NominateFaction("player1", models.FactionNomads)
	auction.NominateFaction("player2", models.FactionWitches)

	auction.PlaceBid("player1", models.FactionNomads, 3)
	auction.PlaceBid("player2", models.FactionWitches, 5)

	summary := auction.GetAuctionSummary()

	// Check player1
	if result, ok := summary["player1"]; ok {
		if result.Faction != models.FactionNomads {
			t.Errorf("player1 should have Nomads, got %v", result.Faction)
		}
		if result.StartingVP != 37 {
			t.Errorf("player1 should start with 37 VP, got %d", result.StartingVP)
		}
		if result.VPBid != 3 {
			t.Errorf("player1 bid should be 3, got %d", result.VPBid)
		}
	} else {
		t.Error("player1 not in summary")
	}

	// Check player2
	if result, ok := summary["player2"]; ok {
		if result.Faction != models.FactionWitches {
			t.Errorf("player2 should have Witches, got %v", result.Faction)
		}
		if result.StartingVP != 35 {
			t.Errorf("player2 should start with 35 VP, got %d", result.StartingVP)
		}
		if result.VPBid != 5 {
			t.Errorf("player2 bid should be 5, got %d", result.VPBid)
		}
	} else {
		t.Error("player2 not in summary")
	}
}

func TestGameSetupOptions_Validation(t *testing.T) {
	// Valid options
	opts := GameSetupOptions{
		UseAuction:  true,
		PlayerCount: 3,
		SeatOrder:   []string{"p1", "p2", "p3"},
	}

	err := ValidateSetupOptions(opts)
	if err != nil {
		t.Errorf("valid options should pass: %v", err)
	}

	// Invalid player count
	opts.PlayerCount = 1
	err = ValidateSetupOptions(opts)
	if err == nil {
		t.Error("should reject player count < 2")
	}

	opts.PlayerCount = 6
	err = ValidateSetupOptions(opts)
	if err == nil {
		t.Error("should reject player count > 5")
	}

	// Mismatched seat order
	opts.PlayerCount = 3
	opts.SeatOrder = []string{"p1", "p2"}
	err = ValidateSetupOptions(opts)
	if err == nil {
		t.Error("should reject mismatched seat order")
	}
}

func TestAuction_CannotNominateSameColor(t *testing.T) {
	// Test that only one faction per color can be nominated
	seatOrder := []string{"Alice", "Bob", "Charlie", "Diana"}
	auction := NewAuctionState(seatOrder)

	// Alice nominates Nomads (Yellow/Desert)
	err := auction.NominateFaction("Alice", models.FactionNomads)
	if err != nil {
		t.Fatalf("Alice failed to nominate Nomads: %v", err)
	}

	// Bob tries to nominate Fakirs (also Yellow/Desert) - should fail
	err = auction.NominateFaction("Bob", models.FactionFakirs)
	if err == nil {
		t.Error("Bob should not be able to nominate Fakirs (same color as Nomads)")
	}

	// Bob nominates Giants (Red/Wasteland) - should succeed
	err = auction.NominateFaction("Bob", models.FactionGiants)
	if err != nil {
		t.Fatalf("Bob failed to nominate Giants: %v", err)
	}

	// Charlie tries to nominate Chaos Magicians (also Red/Wasteland) - should fail
	err = auction.NominateFaction("Charlie", models.FactionChaosMagicians)
	if err == nil {
		t.Error("Charlie should not be able to nominate Chaos Magicians (same color as Giants)")
	}

	// Charlie nominates Swarmlings (Blue/Lake) - should succeed
	err = auction.NominateFaction("Charlie", models.FactionSwarmlings)
	if err != nil {
		t.Fatalf("Charlie failed to nominate Swarmlings: %v", err)
	}

	// Diana nominates Witches (Green/Forest) - should succeed
	err = auction.NominateFaction("Diana", models.FactionWitches)
	if err != nil {
		t.Fatalf("Diana failed to nominate Witches: %v", err)
	}

	// Verify nomination order
	if len(auction.NominationOrder) != 4 {
		t.Errorf("expected 4 nominations, got %d", len(auction.NominationOrder))
	}

	// Verify all different colors
	colors := make(map[models.FactionColor]bool)
	for _, faction := range auction.NominationOrder {
		color := faction.GetFactionColor()
		if colors[color] {
			t.Errorf("duplicate color found: %v", color)
		}
		colors[color] = true
	}
}

func TestAuction_ComplexScenario(t *testing.T) {
	// 4-player game with multiple overbids
	seatOrder := []string{"Alice", "Bob", "Charlie", "Diana"}
	auction := NewAuctionState(seatOrder)

	// Nominations
	auction.NominateFaction("Alice", models.FactionNomads)
	auction.NominateFaction("Bob", models.FactionWitches)
	auction.NominateFaction("Charlie", models.FactionEngineers)
	auction.NominateFaction("Diana", models.FactionGiants)

	// Bidding (in seat order: Alice, Bob, Charlie, Diana)
	auction.PlaceBid("Alice", models.FactionNomads, 0) // Alice: Nomads @ 40
	auction.PlaceBid("Bob", models.FactionNomads, 1)   // Bob: Nomads @ 39 (overbids Alice)
	// Bob now has a faction, skip to Charlie
	auction.PlaceBid("Charlie", models.FactionWitches, 0) // Charlie: Witches @ 40
	// Charlie now has a faction, skip to Diana
	auction.PlaceBid("Diana", models.FactionEngineers, 0) // Diana: Engineers @ 40

	// Check state after Diana's bid
	t.Logf("After Diana's bid - Active: %v, PlayerHasFaction: %+v", auction.Active, auction.PlayerHasFaction)

	// Diana now has a faction, skip Bob/Charlie/Diana, back to Alice
	err := auction.PlaceBid("Alice", models.FactionGiants, 0) // Alice: Giants @ 40
	if err != nil {
		t.Fatalf("Alice failed to bid on Giants: %v", err)
	}

	// Auction complete
	if auction.Active {
		t.Error("auction should be complete")
	}

	// Debug: Check faction holders
	t.Logf("Faction holders: %+v", auction.FactionHolders)
	t.Logf("Nomination order: %+v", auction.NominationOrder)

	// Verify final assignments
	turnOrder := auction.GetTurnOrder()
	if len(turnOrder) != 4 {
		t.Fatalf("expected 4 players in turn order, got %d", len(turnOrder))
	}
	if turnOrder[0] != "Bob" { // Won Nomads (1st nominated)
		t.Errorf("1st should be Bob, got %s", turnOrder[0])
	}
	if turnOrder[1] != "Charlie" { // Won Witches (2nd nominated)
		t.Errorf("2nd should be Charlie, got %s", turnOrder[1])
	}
	if turnOrder[2] != "Diana" { // Won Engineers (3rd nominated)
		t.Errorf("3rd should be Diana, got %s", turnOrder[2])
	}
	if turnOrder[3] != "Alice" { // Won Giants (4th nominated)
		t.Errorf("4th should be Alice, got %s", turnOrder[3])
	}

	// Verify starting VPs
	if auction.GetStartingVP(models.FactionNomads) != 39 {
		t.Errorf("Nomads should start at 39 VP")
	}
	if auction.GetStartingVP(models.FactionWitches) != 40 {
		t.Errorf("Witches should start at 40 VP")
	}
}
