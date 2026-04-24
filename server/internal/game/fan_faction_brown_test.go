package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestProspectorsCultRewardSpadesBecomePriests(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewProspectors()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Priests = 0

	gs.grantCultReward("p1", player, CultRewardSpade, 2)

	if got := player.Resources.Priests; got != 2 {
		t.Fatalf("priests = %d, want 2", got)
	}
	if got := gs.PendingCultRewardSpades["p1"]; got != 0 {
		t.Fatalf("pending cult reward spades = %d, want 0", got)
	}
}

func TestProspectorsPowerActionSpadesBecomePriests(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewProspectors()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}
	gs.Round = 1
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{Type: ScoringSpades, ActionType: ScoringActionSpades, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
		},
		PriestsSent: map[string]int{},
	}

	player := gs.GetPlayer("p1")
	player.Resources.Power = NewPowerSystem(0, 8, 0)
	player.Resources.Priests = 0
	player.VictoryPoints = 0

	if err := NewPowerAction("p1", PowerActionSpade1).Execute(gs); err != nil {
		t.Fatalf("power action spade1 failed: %v", err)
	}
	if got := player.Resources.Priests; got != 1 {
		t.Fatalf("priests after spade1 = %d, want 1", got)
	}
	if got := player.VictoryPoints; got != 2 {
		t.Fatalf("victory points after spade1 = %d, want 2", got)
	}

	player.Resources.Power = NewPowerSystem(0, 12, 0)
	if err := NewPowerAction("p1", PowerActionSpade2).Execute(gs); err != nil {
		t.Fatalf("power action spade2 failed: %v", err)
	}
	if got := player.Resources.Priests; got != 3 {
		t.Fatalf("priests after spade2 = %d, want 3", got)
	}
	if got := player.VictoryPoints; got != 6 {
		t.Fatalf("victory points after spade2 = %d, want 6", got)
	}
}

func TestProspectorsTransformUsesGoldenSpades(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewProspectors()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.TurnOrder = []string{"p1"}
	gs.Round = 1
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{Type: ScoringSpades, ActionType: ScoringActionSpades, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
		},
		PriestsSent: map[string]int{},
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Power = NewPowerSystem(2, 0, 0)
	player.VictoryPoints = 0

	sourceHex := board.NewHex(0, 0)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(sourceHex, player.Faction.GetHomeTerrain())
	gs.Map.PlaceBuilding(sourceHex, testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling))

	var targetTerrain models.TerrainType
	found := false
	for _, terrain := range []models.TerrainType{
		models.TerrainSwamp,
		models.TerrainLake,
		models.TerrainForest,
		models.TerrainMountain,
		models.TerrainWasteland,
		models.TerrainDesert,
	} {
		if gs.Map.GetTerrainDistance(terrain, player.Faction.GetHomeTerrain()) == 2 {
			targetTerrain = terrain
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("failed to find a distance-2 terrain for Prospectors")
	}
	gs.Map.TransformTerrain(targetHex, targetTerrain)

	if err := NewTransformAndBuildAction("p1", targetHex, false, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("TransformAndBuildAction.Execute failed: %v", err)
	}

	if got := player.Resources.Coins; got != 12 {
		t.Fatalf("coins = %d, want 12", got)
	}
	if got := player.VictoryPoints; got != 2 {
		t.Fatalf("victory points = %d, want 2", got)
	}
	if player.Resources.Power.Bowl1 != 0 || player.Resources.Power.Bowl2 != 2 || player.Resources.Power.Bowl3 != 0 {
		t.Fatalf("unexpected power state after golden spades: %d/%d/%d", player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
	}
}

func TestProspectorsStrongholdDiscountAndAction(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewProspectors()); err != nil {
		t.Fatalf("AddPlayer p1 failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer p2 failed: %v", err)
	}
	if err := gs.AddPlayer("p3", factions.NewGiants()); err != nil {
		t.Fatalf("AddPlayer p3 failed: %v", err)
	}
	gs.TurnOrder = []string{"p1", "p2", "p3"}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Coins = 20
	player.Resources.Workers = 10
	player.Resources.Power = NewPowerSystem(1, 0, 0)

	sourceHex := board.NewHex(0, 0)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(sourceHex, player.Faction.GetHomeTerrain())
	gs.Map.PlaceBuilding(sourceHex, testBuilding("p1", player.Faction.GetType(), models.BuildingDwelling))

	var targetTerrain models.TerrainType
	found := false
	for _, terrain := range []models.TerrainType{
		models.TerrainSwamp,
		models.TerrainLake,
		models.TerrainForest,
		models.TerrainMountain,
		models.TerrainWasteland,
		models.TerrainDesert,
	} {
		if gs.Map.GetTerrainDistance(terrain, player.Faction.GetHomeTerrain()) == 1 {
			targetTerrain = terrain
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("failed to find a distance-1 terrain for Prospectors")
	}
	gs.Map.TransformTerrain(targetHex, targetTerrain)

	if err := NewTransformAndBuildAction("p1", targetHex, false, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("TransformAndBuildAction.Execute failed: %v", err)
	}
	if got := player.Resources.Coins; got != 17 {
		t.Fatalf("coins after discounted golden spade = %d, want 17", got)
	}

	gs.Map.PlaceBuilding(board.NewHex(3, 0), testBuilding("p2", models.FactionWitches, models.BuildingTradingHouse))
	gs.Map.PlaceBuilding(board.NewHex(4, 0), testBuilding("p3", models.FactionGiants, models.BuildingTradingHouse))
	gs.Map.PlaceBuilding(board.NewHex(5, 0), testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse))

	player.Resources.Coins = 0
	if err := NewProspectorsGainCoinsAction("p1").Execute(gs); err != nil {
		t.Fatalf("Prospectors stronghold action failed: %v", err)
	}
	if got := player.Resources.Coins; got != 2 {
		t.Fatalf("coins after Prospectors stronghold action = %d, want 2", got)
	}
}

func TestProspectorsBonusCardSpadeBecomesPriestAndStillScoresSpades(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewProspectors()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Round = 1
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{Type: ScoringSpades, ActionType: ScoringActionSpades, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
		},
		PriestsSent: map[string]int{},
	}
	gs.BonusCards.PlayerCards["p1"] = BonusCardSpade

	player := gs.GetPlayer("p1")
	player.Resources.Priests = 0
	player.VictoryPoints = 0

	if err := NewBonusCardSpadeAction("p1", board.NewHex(0, 0), false, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("bonus card spade failed: %v", err)
	}
	if got := player.Resources.Priests; got != 1 {
		t.Fatalf("priests after bonus card spade = %d, want 1", got)
	}
	if got := player.VictoryPoints; got != 2 {
		t.Fatalf("victory points after bonus card spade = %d, want 2", got)
	}
}

func TestTimeTravelersScorePreviousAndNextRounds(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTimeTravelers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	gs.Round = 1
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{Type: ScoringDwellingWater, ActionType: ScoringActionDwelling, ActionVP: 2},
			{Type: ScoringTradingHouseWater, ActionType: ScoringActionTradingHouse, ActionVP: 3},
			{Type: ScoringTemplePriest, ActionType: ScoringActionTemple, ActionVP: 4},
			{Type: ScoringStrongholdFire, ActionType: ScoringActionStronghold, ActionVP: 5},
			{Type: ScoringSpades, ActionType: ScoringActionSpades, ActionVP: 2},
			{Type: ScoringTradingHouseAir, ActionType: ScoringActionTradingHouse, ActionVP: 3},
		},
		PriestsSent: map[string]int{},
	}

	player := gs.GetPlayer("p1")
	player.VictoryPoints = 0

	gs.AwardActionVP("p1", ScoringActionTradingHouse)
	if got := player.VictoryPoints; got != 6 {
		t.Fatalf("victory points after time-travel trading house scoring = %d, want 6", got)
	}

	gs.AwardActionVP("p1", ScoringActionDwelling)
	if got := player.VictoryPoints; got != 6 {
		t.Fatalf("victory points should ignore current-round dwelling tile, got %d", got)
	}
}

func TestTimeTravelersStrongholdActionMovesPowerFromBowlOneToThree(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTimeTravelers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Power = NewPowerSystem(4, 0, 0)

	if err := NewTimeTravelersPowerShiftAction("p1", 3).Execute(gs); err != nil {
		t.Fatalf("Time Travelers stronghold action failed: %v", err)
	}
	if player.Resources.Power.Bowl1 != 1 || player.Resources.Power.Bowl2 != 0 || player.Resources.Power.Bowl3 != 3 {
		t.Fatalf("unexpected time traveler power state after stronghold action: %d/%d/%d", player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
	}
}

func TestTimeTravelersCannotAdvanceDigging(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTimeTravelers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	err := NewAdvanceDiggingAction("p1").Validate(gs)
	if err == nil || err.Error() != "time travelers cannot advance digging level" {
		t.Fatalf("Validate error = %v, want time travelers digging rejection", err)
	}
}

func TestProspectorsCanUseStrongholdActionOnBuildTurn(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewProspectors()); err != nil {
		t.Fatalf("AddPlayer p1 failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer p2 failed: %v", err)
	}
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = PhaseAction

	player := gs.GetPlayer("p1")
	player.Options.ConfirmActions = false
	player.Resources.Coins = 30
	player.Resources.Workers = 20

	thHex := board.NewHex(0, 0)
	gs.Map.TransformTerrain(thHex, player.Faction.GetHomeTerrain())
	gs.Map.PlaceBuilding(thHex, testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse))
	gs.Map.PlaceBuilding(board.NewHex(3, 0), testBuilding("p2", models.FactionWitches, models.BuildingTradingHouse))

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", NewUpgradeBuildingAction("p1", thHex, models.BuildingStronghold), ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("upgrade to stronghold failed: %v", err)
	}
	if got := gs.PendingFreeActionsPlayerID; got != "p1" {
		t.Fatalf("pending free-actions player = %q, want p1", got)
	}
	if !gs.HasPendingTurnConfirmation() {
		t.Fatal("expected pending turn confirmation after Prospectors stronghold build")
	}

	player.Resources.Coins = 0
	if _, err := mgr.ExecuteActionWithMeta("g1", NewProspectorsGainCoinsAction("p1"), ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("same-turn Prospectors stronghold action failed: %v", err)
	}
	if got := player.Resources.Coins; got != 1 {
		t.Fatalf("coins after same-turn Prospectors stronghold action = %d, want 1", got)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "p2" {
		t.Fatalf("current player after same-turn Prospectors action = %v, want p2", current)
	}
}

func TestTimeTravelersCanUseStrongholdActionOnBuildTurn(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTimeTravelers()); err != nil {
		t.Fatalf("AddPlayer p1 failed: %v", err)
	}
	if err := gs.AddPlayer("p2", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer p2 failed: %v", err)
	}
	gs.TurnOrder = []string{"p1", "p2"}
	gs.CurrentPlayerIndex = 0
	gs.Phase = PhaseAction

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 30
	player.Resources.Workers = 20
	player.Resources.Power = NewPowerSystem(4, 0, 0)

	thHex := board.NewHex(0, 0)
	gs.Map.TransformTerrain(thHex, player.Faction.GetHomeTerrain())
	gs.Map.PlaceBuilding(thHex, testBuilding("p1", player.Faction.GetType(), models.BuildingTradingHouse))

	mgr := NewManager()
	mgr.CreateGameWithState("g1", gs)

	if _, err := mgr.ExecuteActionWithMeta("g1", NewUpgradeBuildingAction("p1", thHex, models.BuildingStronghold), ActionMeta{ExpectedRevision: 0}); err != nil {
		t.Fatalf("upgrade to stronghold failed: %v", err)
	}
	if got := gs.PendingFreeActionsPlayerID; got != "p1" {
		t.Fatalf("pending free-actions player = %q, want p1", got)
	}
	if !gs.HasPendingTurnConfirmation() {
		t.Fatal("expected pending turn confirmation after Time Travelers stronghold build")
	}

	if _, err := mgr.ExecuteActionWithMeta("g1", NewTimeTravelersPowerShiftAction("p1", 3), ActionMeta{ExpectedRevision: 1}); err != nil {
		t.Fatalf("same-turn Time Travelers stronghold action failed: %v", err)
	}
	if player.Resources.Power.Bowl1 != 1 || player.Resources.Power.Bowl2 != 0 || player.Resources.Power.Bowl3 != 3 {
		t.Fatalf("unexpected time traveler power state after same-turn stronghold action: %d/%d/%d", player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3)
	}
	if current := gs.GetCurrentPlayer(); current == nil || current.ID != "p2" {
		t.Fatalf("current player after same-turn Time Travelers action = %v, want p2", current)
	}
}
