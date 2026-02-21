package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
)

func TestSetupFlow_StrictDwellingOrderAndBonusSelection(t *testing.T) {
	gs := NewGameState()

	if err := gs.AddPlayer("chaos", factions.NewChaosMagicians()); err != nil {
		t.Fatalf("failed adding chaos: %v", err)
	}
	if err := gs.AddPlayer("nomads", factions.NewNomads()); err != nil {
		t.Fatalf("failed adding nomads: %v", err)
	}
	if err := gs.AddPlayer("witches", factions.NewWitches()); err != nil {
		t.Fatalf("failed adding witches: %v", err)
	}

	gs.TurnOrder = []string{"chaos", "nomads", "witches"}
	gs.InitializeSetupSequence()

	expectedDwellingOrder := []string{"nomads", "witches", "witches", "nomads", "nomads", "chaos"}
	if len(gs.SetupDwellingOrder) != len(expectedDwellingOrder) {
		t.Fatalf("unexpected dwelling order length: got %d want %d", len(gs.SetupDwellingOrder), len(expectedDwellingOrder))
	}
	for i := range expectedDwellingOrder {
		if gs.SetupDwellingOrder[i] != expectedDwellingOrder[i] {
			t.Fatalf("unexpected setup dwelling order at %d: got %s want %s", i, gs.SetupDwellingOrder[i], expectedDwellingOrder[i])
		}
	}

	placements := []struct {
		player string
		hex    board.Hex
	}{
		{player: "nomads", hex: board.NewHex(0, 1)},
		{player: "witches", hex: board.NewHex(0, 0)},
		{player: "witches", hex: board.NewHex(1, 0)},
		{player: "nomads", hex: board.NewHex(1, 1)},
		{player: "nomads", hex: board.NewHex(2, 1)},
		{player: "chaos", hex: board.NewHex(3, 1)},
	}

	for _, placement := range placements {
		player := gs.GetPlayer(placement.player)
		gs.Map.TransformTerrain(placement.hex, player.Faction.GetHomeTerrain())
	}

	// Wrong player should be rejected.
	if err := NewSetupDwellingAction("chaos", placements[0].hex).Validate(gs); err == nil {
		t.Fatalf("expected wrong-player setup dwelling to fail")
	}

	for i, placement := range placements {
		action := NewSetupDwellingAction(placement.player, placement.hex)
		if err := action.Execute(gs); err != nil {
			t.Fatalf("placement %d failed: %v", i, err)
		}
	}

	if gs.SetupSubphase != SetupSubphaseBonusCards {
		t.Fatalf("expected bonus-card setup subphase, got %s", gs.SetupSubphase)
	}

	expectedBonusOrder := []string{"witches", "nomads", "chaos"}
	for i := range expectedBonusOrder {
		if gs.SetupBonusOrder[i] != expectedBonusOrder[i] {
			t.Fatalf("unexpected setup bonus order at %d: got %s want %s", i, gs.SetupBonusOrder[i], expectedBonusOrder[i])
		}
	}

	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardPriest,
		BonusCardShipping,
		BonusCardDwellingVP,
		BonusCardWorkerPower,
		BonusCardSpade,
		BonusCardTradingHouseVP,
	})

	bonusPicks := []struct {
		player string
		card   BonusCardType
	}{
		{player: "witches", card: BonusCardPriest},
		{player: "nomads", card: BonusCardShipping},
		{player: "chaos", card: BonusCardDwellingVP},
	}

	for _, pick := range bonusPicks {
		action := &SetupBonusCardAction{
			BaseAction: BaseAction{Type: ActionSetupBonusCard, PlayerID: pick.player},
			BonusCard:  pick.card,
		}
		if err := action.Execute(gs); err != nil {
			t.Fatalf("bonus pick failed for %s: %v", pick.player, err)
		}
	}

	if gs.Phase != PhaseAction {
		t.Fatalf("expected action phase after setup, got %v", gs.Phase)
	}
	if gs.SetupSubphase != SetupSubphaseComplete {
		t.Fatalf("expected complete setup subphase, got %s", gs.SetupSubphase)
	}
	if gs.Round != 1 {
		t.Fatalf("expected round to remain 1 after setup, got %d", gs.Round)
	}

	leftoverCards := []BonusCardType{BonusCardWorkerPower, BonusCardSpade, BonusCardTradingHouseVP}
	for _, card := range leftoverCards {
		if gs.BonusCards.Available[card] != 1 {
			t.Fatalf("expected leftover setup bonus card %v to have 1 coin, got %d", card, gs.BonusCards.Available[card])
		}
	}
}

func TestSetupDwelling_LazySetupInitializationForReplayCompatibility(t *testing.T) {
	gs := NewGameState()
	gs.Phase = PhaseSetup
	gs.SetupSubphase = SetupSubphaseNone
	gs.TurnOrder = []string{"engineers", "witches"}

	if err := gs.AddPlayer("engineers", factions.NewEngineers()); err != nil {
		t.Fatalf("failed adding engineers: %v", err)
	}
	if err := gs.AddPlayer("witches", factions.NewWitches()); err != nil {
		t.Fatalf("failed adding witches: %v", err)
	}

	firstHex := board.NewHex(0, 1)
	secondHex := board.NewHex(0, 0)
	gs.Map.TransformTerrain(firstHex, gs.GetPlayer("engineers").Faction.GetHomeTerrain())
	gs.Map.TransformTerrain(secondHex, gs.GetPlayer("witches").Faction.GetHomeTerrain())

	first := NewSetupDwellingAction("engineers", firstHex)
	if err := first.Execute(gs); err != nil {
		t.Fatalf("first setup dwelling failed: %v", err)
	}

	if gs.SetupSubphase != SetupSubphaseDwellings {
		t.Fatalf("expected dwelling subphase after lazy initialization, got %s", gs.SetupSubphase)
	}
	if len(gs.SetupDwellingOrder) == 0 {
		t.Fatalf("expected setup dwelling order to be initialized")
	}
	if gs.SetupPlacedDwellings["engineers"] != 1 {
		t.Fatalf("expected engineers to have one placed setup dwelling, got %d", gs.SetupPlacedDwellings["engineers"])
	}

	second := NewSetupDwellingAction("witches", secondHex)
	if err := second.Validate(gs); err != nil {
		t.Fatalf("second setup dwelling should be valid after initialization: %v", err)
	}
}

func TestSetupDwelling_ReplayCompatibilityWithoutTurnOrder(t *testing.T) {
	gs := NewGameState()
	gs.Phase = PhaseSetup
	gs.SetupSubphase = SetupSubphaseNone

	if err := gs.AddPlayer("engineers", factions.NewEngineers()); err != nil {
		t.Fatalf("failed adding engineers: %v", err)
	}
	if err := gs.AddPlayer("witches", factions.NewWitches()); err != nil {
		t.Fatalf("failed adding witches: %v", err)
	}

	hex := board.NewHex(0, 1)
	gs.Map.TransformTerrain(hex, gs.GetPlayer("engineers").Faction.GetHomeTerrain())

	action := NewSetupDwellingAction("engineers", hex)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("setup dwelling should succeed in replay compatibility mode: %v", err)
	}

	if gs.SetupPlacedDwellings["engineers"] != 1 {
		t.Fatalf("expected one setup dwelling recorded for engineers, got %d", gs.SetupPlacedDwellings["engineers"])
	}
	if gs.SetupSubphase != SetupSubphaseNone {
		t.Fatalf("expected setup subphase to remain none in replay compatibility mode, got %s", gs.SetupSubphase)
	}
}
