package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestArchivistsSelectFactionAddsExtraBonusCard(t *testing.T) {
	gs := NewGameState()
	gs.EnableFanFactions = true
	gs.Phase = PhaseFactionSelection
	gs.TurnOrder = []string{"p1"}
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardPriest,
		BonusCardShipping,
		BonusCardDwellingVP,
		BonusCardWorkerPower,
	})
	if err := gs.AddPlayer("p1", nil); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	action := &SelectFactionAction{
		PlayerID:    "p1",
		FactionType: models.FactionArchivists,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("SelectFactionAction.Execute failed: %v", err)
	}

	if got := len(gs.BonusCards.Available); got != 5 {
		t.Fatalf("available bonus cards = %d, want 5", got)
	}
}

func TestArchivistsSkipCultRewards(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewArchivists()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Workers = 0
	player.Resources.Coins = 0
	gs.CultTracks.PlayerPositions["p1"][CultFire] = 6
	player.CultPositions[CultFire] = 6

	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{Type: ScoringDwellingWater, CultTrack: CultFire, CultThreshold: 2, CultRewardType: CultRewardWorker, CultRewardAmount: 1},
			{Type: ScoringTemplePriest, CultRewardAmount: 2},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
			{Type: ScoringDwellingWater},
		},
		PriestsSent: map[string]int{"p1": 2},
	}

	gs.AwardCultRewardsForRound(1)
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers after regular cult rewards = %d, want 0", got)
	}

	gs.AwardCultRewardsForRound(2)
	if got := player.Resources.Coins; got != 0 {
		t.Fatalf("coins after temple-priest cult rewards = %d, want 0", got)
	}
}

func TestArchivistsPassGainsPowerFromBonusCardCoins(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewArchivists()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}
	gs.SuppressTurnAdvance = true
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardSpade})
	gs.BonusCards.Available[BonusCardSpade] = 2

	player := gs.GetPlayer("p1")
	startCoins := player.Resources.Coins
	startBowl1 := player.Resources.Power.Bowl1
	startBowl2 := player.Resources.Power.Bowl2

	action := NewPassAction("p1", func() *BonusCardType { card := BonusCardSpade; return &card }())
	if err := action.Execute(gs); err != nil {
		t.Fatalf("PassAction.Execute failed: %v", err)
	}

	if got := player.Resources.Coins; got != startCoins+2 {
		t.Fatalf("coins = %d, want %d", got, startCoins+2)
	}
	if got := player.Resources.Power.Bowl1; got != startBowl1-4 {
		t.Fatalf("bowl I = %d, want %d", got, startBowl1-4)
	}
	if got := player.Resources.Power.Bowl2; got != startBowl2+4 {
		t.Fatalf("bowl II = %d, want %d", got, startBowl2+4)
	}
}

func TestArchivistsStrongholdPassTakesTwoBonusCards(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewArchivists()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}
	gs.SuppressTurnAdvance = true
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{
		BonusCardSpade,
		BonusCardCultAdvance,
		BonusCardShipping,
	})

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	gs.BonusCards.PlayerCards["p1"] = BonusCard6Coins
	gs.BonusCards.PlayerHasCard["p1"] = false
	gs.BonusCards.Available[BonusCardSpade] = 2
	gs.BonusCards.Available[BonusCardCultAdvance] = 1
	gs.BonusCards.Available[BonusCardShipping] = 0

	firstCard := BonusCardSpade
	if err := NewPassAction("p1", &firstCard).Execute(gs); err != nil {
		t.Fatalf("PassAction.Execute failed: %v", err)
	}

	if gs.PendingArchivistsBonusSelection == nil {
		t.Fatalf("expected pending Archivists second-card selection")
	}
	if got := gs.BonusCards.GetPlayerCards("p1"); len(got) != 1 || got[0] != BonusCardSpade {
		t.Fatalf("held cards after first Archivists pick = %v, want [Spade]", got)
	}

	if err := NewSelectArchivistsBonusCardAction("p1", BonusCard6Coins).Validate(gs); err == nil {
		t.Fatalf("expected returned card to be invalid for Archivists second pick")
	}

	if err := NewSelectArchivistsBonusCardAction("p1", BonusCardCultAdvance).Execute(gs); err != nil {
		t.Fatalf("SelectArchivistsBonusCardAction.Execute failed: %v", err)
	}

	if gs.PendingArchivistsBonusSelection != nil {
		t.Fatalf("expected Archivists pending selection to be cleared")
	}
	if got := gs.BonusCards.GetPlayerCards("p1"); len(got) != 2 {
		t.Fatalf("held cards after second Archivists pick = %v, want 2 cards", got)
	}
	if got := player.Resources.Coins; got != 18 {
		t.Fatalf("coins after Archivists two-card pass = %d, want 18", got)
	}
	if got := player.Resources.Power.Bowl1; got != 0 {
		t.Fatalf("bowl I after Archivists two-card pass = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl2; got != 11 {
		t.Fatalf("bowl II after Archivists two-card pass = %d, want 11", got)
	}
	if got := player.Resources.Power.Bowl3; got != 1 {
		t.Fatalf("bowl III after Archivists two-card pass = %d, want 1", got)
	}
	if got := gs.PendingFreeActionsPlayerID; got != "p1" {
		t.Fatalf("pending free actions player = %q, want p1", got)
	}
	if !gs.HasPendingTurnConfirmation() {
		t.Fatalf("expected Archivists second bonus-card choice to open turn confirmation")
	}
	if got := gs.PendingTurnConfirmationPlayerID; got != "p1" {
		t.Fatalf("pending turn confirmation player = %q, want p1", got)
	}
	if err := NewUndoTurnAction("p1").Execute(gs); err != nil {
		t.Fatalf("UndoTurnAction.Execute failed: %v", err)
	}
	if gs.GetPlayer("p1").HasPassed {
		t.Fatalf("expected undo to revert Archivists pass")
	}
}

func TestArchivistsExtraBonusCardEnablesSpecialAction(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewArchivists()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10

	origin := board.NewHex(0, 0)
	target := board.NewHex(0, 1)
	gs.Map.GetHex(origin).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(origin).Building = testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling)

	gs.BonusCards.PlayerCards["p1"] = BonusCard6Coins
	gs.BonusCards.PlayerExtraCards["p1"] = []BonusCardType{BonusCardSpade}
	gs.BonusCards.PlayerHasCard["p1"] = true

	if err := NewBonusCardSpadeAction("p1", target, false, models.TerrainTypeUnknown).Validate(gs); err != nil {
		t.Fatalf("bonus card spade from Archivists extra card should validate, got: %v", err)
	}
}

func TestDjinniStartWithThreeLamps(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDjinni()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	if got := gs.GetPlayer("p1").DjinniLampTokens; got != 3 {
		t.Fatalf("Djinni lamp tokens = %d, want 3", got)
	}
	if gs.PendingDjinniStartingCultChoice == nil || gs.PendingDjinniStartingCultChoice.PlayerID != "p1" {
		t.Fatalf("expected pending Djinni starting cult choice for p1")
	}
}

func TestDjinniStartingCultChoiceAdvancesSelectedTrack(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDjinni()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	if err := NewSelectDjinniStartingCultTrackAction("p1", CultWater).Execute(gs); err != nil {
		t.Fatalf("SelectDjinniStartingCultTrackAction.Execute failed: %v", err)
	}

	if got := gs.CultTracks.GetPosition("p1", CultWater); got != 2 {
		t.Fatalf("water cult = %d, want 2", got)
	}
	if gs.PendingDjinniStartingCultChoice != nil {
		t.Fatalf("expected Djinni starting cult choice to clear")
	}
}

func TestDjinniSwapCultsUsesLamp(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDjinni()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	gs.CultTracks.PlayerPositions["p1"][CultFire] = 4
	gs.CultTracks.PlayerPositions["p1"][CultWater] = 7
	player.CultPositions[CultFire] = 4
	player.CultPositions[CultWater] = 7

	if err := NewDjinniSwapCultsAction("p1", CultFire, CultWater).Execute(gs); err != nil {
		t.Fatalf("Djinni swap failed: %v", err)
	}

	if got := gs.CultTracks.GetPosition("p1", CultFire); got != 7 {
		t.Fatalf("fire cult = %d, want 7", got)
	}
	if got := gs.CultTracks.GetPosition("p1", CultWater); got != 4 {
		t.Fatalf("water cult = %d, want 4", got)
	}
	if got := player.DjinniLampTokens; got != 2 {
		t.Fatalf("Djinni lamp tokens = %d, want 2", got)
	}
	if player.SpecialActionsUsed[SpecialActionDjinniSwapCults] {
		t.Fatalf("Djinni lamp action should not be tracked as once-per-round")
	}
}

func TestDjinniSwapCultsRespectsOccupiedTen(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDjinni()); err != nil {
		t.Fatalf("AddPlayer p1 failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewGiants()); err != nil {
		t.Fatalf("AddPlayer p2 failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	gs.CultTracks.PlayerPositions["p1"][CultFire] = 10
	gs.CultTracks.PlayerPositions["p1"][CultWater] = 2
	player.CultPositions[CultFire] = 10
	player.CultPositions[CultWater] = 2
	gs.CultTracks.Position10Occupied[CultFire] = "p1"
	gs.CultTracks.Position10Occupied[CultWater] = "p2"

	if err := NewDjinniSwapCultsAction("p1", CultFire, CultWater).Validate(gs); err == nil {
		t.Fatalf("expected Djinni swap into occupied position 10 to fail")
	}
}

func TestDjinniStrongholdPassScoresPriestsOnCultBoard(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDjinni()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}
	gs.SuppressTurnAdvance = true
	gs.BonusCards.SetAvailableBonusCards([]BonusCardType{BonusCardPriest})

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.VictoryPoints = 20
	gs.CultTracks.PriestsOnActionSpaces["p1"][CultFire] = 2
	gs.CultTracks.PriestsOnActionSpaces["p1"][CultAir] = 1

	card := BonusCardPriest
	if err := NewPassAction("p1", &card).Execute(gs); err != nil {
		t.Fatalf("PassAction.Execute failed: %v", err)
	}

	if got := player.VictoryPoints; got != 23 {
		t.Fatalf("victory points after Djinni pass = %d, want 23", got)
	}
}
