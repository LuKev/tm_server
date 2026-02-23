package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestAuctionSetupFlow_RegularAuctionTransitionsToSetup(t *testing.T) {
	mgr := NewManager()
	if err := mgr.CreateGameWithOptions("auction_regular", []string{"p1", "p2", "p3"}, CreateGameOptions{
		RandomizeTurnOrder: false,
		SetupMode:          SetupModeAuction,
	}); err != nil {
		t.Fatalf("create regular auction game: %v", err)
	}

	gs, ok := mgr.GetGame("auction_regular")
	if !ok {
		t.Fatalf("expected game to exist")
	}
	if gs.SetupMode != SetupModeAuction {
		t.Fatalf("expected setup mode auction, got %s", gs.SetupMode)
	}

	actions := []Action{
		NewAuctionNominateFactionAction("p1", models.FactionNomads),
		NewAuctionNominateFactionAction("p2", models.FactionWitches),
		NewAuctionNominateFactionAction("p3", models.FactionEngineers),
		NewAuctionPlaceBidAction("p1", models.FactionNomads, 0),
		NewAuctionPlaceBidAction("p2", models.FactionWitches, 1),
		NewAuctionPlaceBidAction("p3", models.FactionEngineers, 2),
	}
	for _, action := range actions {
		if err := mgr.ExecuteAction("auction_regular", action); err != nil {
			t.Fatalf("execute %T: %v", action, err)
		}
	}

	if gs.Phase != PhaseSetup {
		t.Fatalf("expected setup phase after auction resolution, got %v", gs.Phase)
	}
	if gs.SetupSubphase != SetupSubphaseDwellings {
		t.Fatalf("expected dwelling subphase, got %s", gs.SetupSubphase)
	}

	if got := gs.GetPlayer("p1").Faction.GetType(); got != models.FactionNomads {
		t.Fatalf("p1 faction mismatch: got %s", got)
	}
	if got := gs.GetPlayer("p2").Faction.GetType(); got != models.FactionWitches {
		t.Fatalf("p2 faction mismatch: got %s", got)
	}
	if got := gs.GetPlayer("p3").Faction.GetType(); got != models.FactionEngineers {
		t.Fatalf("p3 faction mismatch: got %s", got)
	}

	if got := gs.GetPlayer("p1").VictoryPoints; got != 40 {
		t.Fatalf("p1 starting VP mismatch: got %d want 40", got)
	}
	if got := gs.GetPlayer("p2").VictoryPoints; got != 39 {
		t.Fatalf("p2 starting VP mismatch: got %d want 39", got)
	}
	if got := gs.GetPlayer("p3").VictoryPoints; got != 38 {
		t.Fatalf("p3 starting VP mismatch: got %d want 38", got)
	}

	wantTurnOrder := []string{"p1", "p2", "p3"}
	for i := range wantTurnOrder {
		if gs.TurnOrder[i] != wantTurnOrder[i] {
			t.Fatalf("turn order mismatch at %d: got %s want %s", i, gs.TurnOrder[i], wantTurnOrder[i])
		}
	}
}

func TestAuctionSetupFlow_FastAuctionTransitionsToSetup(t *testing.T) {
	mgr := NewManager()
	if err := mgr.CreateGameWithOptions("auction_fast", []string{"p1", "p2", "p3"}, CreateGameOptions{
		RandomizeTurnOrder: false,
		SetupMode:          SetupModeFastAuction,
	}); err != nil {
		t.Fatalf("create fast auction game: %v", err)
	}

	gs, ok := mgr.GetGame("auction_fast")
	if !ok {
		t.Fatalf("expected game to exist")
	}
	if gs.SetupMode != SetupModeFastAuction {
		t.Fatalf("expected setup mode fast_auction, got %s", gs.SetupMode)
	}

	setupActions := []Action{
		NewAuctionNominateFactionAction("p1", models.FactionNomads),
		NewAuctionNominateFactionAction("p2", models.FactionWitches),
		NewAuctionNominateFactionAction("p3", models.FactionEngineers),
		NewFastAuctionSubmitBidsAction("p1", map[models.FactionType]int{
			models.FactionNomads:    2,
			models.FactionWitches:   4,
			models.FactionEngineers: 1,
		}),
		NewFastAuctionSubmitBidsAction("p2", map[models.FactionType]int{
			models.FactionNomads:    0,
			models.FactionWitches:   3,
			models.FactionEngineers: 5,
		}),
		NewFastAuctionSubmitBidsAction("p3", map[models.FactionType]int{
			models.FactionNomads:    1,
			models.FactionWitches:   2,
			models.FactionEngineers: 4,
		}),
	}
	for _, action := range setupActions {
		if err := mgr.ExecuteAction("auction_fast", action); err != nil {
			t.Fatalf("execute %T: %v", action, err)
		}
	}

	if gs.Phase != PhaseSetup {
		t.Fatalf("expected setup phase after fast auction resolution, got %v", gs.Phase)
	}
	if gs.SetupSubphase != SetupSubphaseDwellings {
		t.Fatalf("expected dwelling subphase, got %s", gs.SetupSubphase)
	}

	if got := gs.GetPlayer("p1").Faction.GetType(); got != models.FactionWitches {
		t.Fatalf("p1 faction mismatch: got %s", got)
	}
	if got := gs.GetPlayer("p2").Faction.GetType(); got != models.FactionEngineers {
		t.Fatalf("p2 faction mismatch: got %s", got)
	}
	if got := gs.GetPlayer("p3").Faction.GetType(); got != models.FactionNomads {
		t.Fatalf("p3 faction mismatch: got %s", got)
	}

	if got := gs.GetPlayer("p1").VictoryPoints; got != 36 {
		t.Fatalf("p1 starting VP mismatch: got %d want 36", got)
	}
	if got := gs.GetPlayer("p2").VictoryPoints; got != 35 {
		t.Fatalf("p2 starting VP mismatch: got %d want 35", got)
	}
	if got := gs.GetPlayer("p3").VictoryPoints; got != 39 {
		t.Fatalf("p3 starting VP mismatch: got %d want 39", got)
	}

	wantTurnOrder := []string{"p3", "p1", "p2"}
	for i := range wantTurnOrder {
		if gs.TurnOrder[i] != wantTurnOrder[i] {
			t.Fatalf("turn order mismatch at %d: got %s want %s", i, gs.TurnOrder[i], wantTurnOrder[i])
		}
	}
}
