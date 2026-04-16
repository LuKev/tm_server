package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestDynionGeifrAddPlayerStartsWithFire2FavorTile(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDynionGeifr()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	if !gs.FavorTiles.HasTileType("p1", FavorFire2) {
		t.Fatalf("expected Dynion Geifr to start with Fire +2 favor tile")
	}
	if got := gs.FavorTiles.Available[FavorFire2]; got != 2 {
		t.Fatalf("Fire +2 remaining = %d, want 2", got)
	}
	if got := player.CultPositions[CultFire]; got != 2 {
		t.Fatalf("fire cult = %d, want 2", got)
	}
	if got := gs.CultTracks.GetPosition("p1", CultFire); got != 2 {
		t.Fatalf("cult track fire = %d, want 2", got)
	}
}

func TestDynionGeifrSelectFactionStartsWithFire2FavorTile(t *testing.T) {
	gs := NewGameState()
	gs.EnableFanFactions = true
	gs.Phase = PhaseFactionSelection
	gs.TurnOrder = []string{"p1"}
	if err := gs.AddPlayer("p1", nil); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	action := &SelectFactionAction{
		PlayerID:    "p1",
		FactionType: models.FactionDynionGeifr,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("SelectFactionAction.Execute failed: %v", err)
	}

	if !gs.FavorTiles.HasTileType("p1", FavorFire2) {
		t.Fatalf("expected Dynion Geifr to gain Fire +2 during faction selection")
	}
	if got := gs.CultTracks.GetPosition("p1", CultFire); got != 2 {
		t.Fatalf("fire cult = %d, want 2", got)
	}
}

func TestDynionGeifrPriestConversionGivesWorkersAndCoins(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDynionGeifr()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Priests = 2
	player.Resources.Workers = 0
	player.Resources.Coins = 0

	action := &ConversionAction{
		BaseAction:     BaseAction{Type: ActionConversion, PlayerID: "p1"},
		ConversionType: ConversionPriestToWorker,
		Amount:         1,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("ConversionAction.Execute failed: %v", err)
	}

	if got := player.Resources.Priests; got != 1 {
		t.Fatalf("priests = %d, want 1", got)
	}
	if got := player.Resources.Workers; got != 2 {
		t.Fatalf("workers = %d, want 2", got)
	}
	if got := player.Resources.Coins; got != 2 {
		t.Fatalf("coins = %d, want 2", got)
	}
}

func TestDynionGeifrStructuresHavePowerTwoAndTownNeedsOnlyThreeWithStronghold(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDynionGeifr()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	hex1 := board.NewHex(0, 0)
	hex2 := board.NewHex(0, 1)
	hex3 := board.NewHex(1, 0)
	for _, hex := range []board.Hex{hex1, hex2, hex3} {
		gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
	}

	if err := gs.BuildDwelling("p1", hex1); err != nil {
		t.Fatalf("BuildDwelling hex1 failed: %v", err)
	}
	if err := gs.BuildDwelling("p1", hex2); err != nil {
		t.Fatalf("BuildDwelling hex2 failed: %v", err)
	}
	if err := gs.BuildDwelling("p1", hex3); err != nil {
		t.Fatalf("BuildDwelling hex3 failed: %v", err)
	}

	if err := NewUpgradeBuildingAction("p1", hex2, models.BuildingTradingHouse).Execute(gs); err != nil {
		t.Fatalf("upgrade hex2 to trading house failed: %v", err)
	}
	if err := NewUpgradeBuildingAction("p1", hex2, models.BuildingStronghold).Execute(gs); err != nil {
		t.Fatalf("upgrade hex2 to stronghold failed: %v", err)
	}
	if err := NewUpgradeBuildingAction("p1", hex3, models.BuildingTradingHouse).Execute(gs); err != nil {
		t.Fatalf("upgrade hex3 to trading house failed: %v", err)
	}

	if got := gs.Map.GetHex(hex1).Building.PowerValue; got != 2 {
		t.Fatalf("dwelling power = %d, want 2", got)
	}
	if got := gs.Map.GetHex(hex2).Building.PowerValue; got != 2 {
		t.Fatalf("stronghold power = %d, want 2", got)
	}
	if got := gs.Map.GetHex(hex3).Building.PowerValue; got != 2 {
		t.Fatalf("trading house power = %d, want 2", got)
	}
	if !gs.CanFormTown("p1", []board.Hex{hex1, hex2, hex3}) {
		t.Fatalf("expected Dynion Geifr to form a town with 3 structures including the stronghold")
	}
}

func TestDynionGeifrFourDwellingsCanFormTown(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDynionGeifr()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	hexes := []board.Hex{
		board.NewHex(0, 0),
		board.NewHex(1, 0),
		board.NewHex(2, 0),
		board.NewHex(3, 0),
	}

	for _, hex := range hexes {
		gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    player.Faction.GetType(),
			PlayerID:   "p1",
			PowerValue: 2,
		})
	}

	if !gs.CanFormTown("p1", hexes) {
		t.Fatalf("expected Dynion Geifr to form a town with four dwellings")
	}
}

func TestDynionGeifrStrongholdGrantsTwoPriestsImmediately(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewDynionGeifr()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Priests = 0

	action := &UpgradeBuildingAction{
		BaseAction:      BaseAction{Type: ActionUpgradeBuilding, PlayerID: "p1"},
		NewBuildingType: models.BuildingStronghold,
	}
	action.handleStrongholdBonuses(gs, player)

	if got := player.Resources.Priests; got != 2 {
		t.Fatalf("priests after Dynion Geifr stronghold bonus = %d, want 2", got)
	}
}

func TestConspiratorsFavorTileGivesTwoCoins(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewConspirators()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 0
	gs.PendingFavorTileSelection = &PendingFavorTileSelection{
		PlayerID:      "p1",
		Count:         1,
		SelectedTiles: []FavorTileType{},
	}

	action := &SelectFavorTileAction{
		BaseAction: BaseAction{Type: ActionSelectFavorTile, PlayerID: "p1"},
		TileType:   FavorAir3,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("SelectFavorTileAction.Execute failed: %v", err)
	}

	if got := player.Resources.Coins; got != 2 {
		t.Fatalf("coins = %d, want 2", got)
	}
	if got := gs.CultTracks.GetPosition("p1", CultAir); got != 3 {
		t.Fatalf("air cult = %d, want 3", got)
	}
}

func TestConspiratorsStrongholdCreatesFavorSelection(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewConspirators()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	action := &UpgradeBuildingAction{
		BaseAction:      BaseAction{Type: ActionUpgradeBuilding, PlayerID: "p1"},
		NewBuildingType: models.BuildingStronghold,
	}
	action.handleStrongholdBonuses(gs, player)

	if gs.PendingFavorTileSelection == nil {
		t.Fatalf("expected pending favor tile selection after Conspirators stronghold")
	}
	if got := gs.PendingFavorTileSelection.PlayerID; got != "p1" {
		t.Fatalf("pending favor tile player = %q, want p1", got)
	}
	if got := gs.PendingFavorTileSelection.Count; got != 1 {
		t.Fatalf("pending favor tile count = %d, want 1", got)
	}
}

func TestConspiratorsSwapFavorReturnsCultKeyAndFreesTopSpace(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewConspirators()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Resources.Coins = 0
	player.Keys = 0

	if err := gs.FavorTiles.TakeFavorTile("p1", FavorFire3); err != nil {
		t.Fatalf("TakeFavorTile failed: %v", err)
	}
	gs.CultTracks.PlayerPositions["p1"][CultFire] = 10
	player.CultPositions[CultFire] = 10
	gs.CultTracks.Position10Occupied[CultFire] = "p1"

	action := NewConspiratorsSwapFavorAction("p1", FavorFire3, FavorWater3)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("Conspirators swap favor failed: %v", err)
	}

	if got := gs.CultTracks.GetPosition("p1", CultFire); got != 7 {
		t.Fatalf("fire cult = %d, want 7", got)
	}
	if got := gs.CultTracks.GetPosition("p1", CultWater); got != 3 {
		t.Fatalf("water cult = %d, want 3", got)
	}
	if _, occupied := gs.CultTracks.Position10Occupied[CultFire]; occupied {
		t.Fatalf("expected fire cult position 10 to be freed")
	}
	if got := player.Keys; got != 1 {
		t.Fatalf("keys = %d, want 1", got)
	}
	if got := player.Resources.Coins; got != 2 {
		t.Fatalf("coins = %d, want 2", got)
	}
	if !gs.FavorTiles.HasTileType("p1", FavorWater3) {
		t.Fatalf("expected player to gain Water +3")
	}
	if gs.FavorTiles.HasTileType("p1", FavorFire3) {
		t.Fatalf("expected player to return Fire +3")
	}
	if got := gs.FavorTiles.Available[FavorFire3]; got != 1 {
		t.Fatalf("Fire +3 remaining = %d, want 1", got)
	}
	if got := gs.FavorTiles.Available[FavorWater3]; got != 0 {
		t.Fatalf("Water +3 remaining = %d, want 0", got)
	}
}

func TestConspiratorsSwapFavorCanRegainPowerOnSameCultTrack(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", factions.NewConspirators()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	player := gs.GetPlayer("p1")
	player.HasStrongholdAbility = true
	player.Keys = 0
	player.Resources.Coins = 0
	player.Resources.Power = NewPowerSystem(3, 0, 0)

	if err := gs.FavorTiles.TakeFavorTile("p1", FavorWater1); err != nil {
		t.Fatalf("TakeFavorTile failed: %v", err)
	}
	gs.CultTracks.PlayerPositions["p1"][CultWater] = 10
	player.CultPositions[CultWater] = 10
	gs.CultTracks.Position10Occupied[CultWater] = "p1"

	action := NewConspiratorsSwapFavorAction("p1", FavorWater1, FavorWater2)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("Conspirators swap favor failed: %v", err)
	}

	if got := gs.CultTracks.GetPosition("p1", CultWater); got != 10 {
		t.Fatalf("water cult = %d, want 10", got)
	}
	if got := player.Resources.Power.Bowl1; got != 0 {
		t.Fatalf("bowl I = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl2; got != 3 {
		t.Fatalf("bowl II = %d, want 3", got)
	}
	if got := player.Resources.Power.Bowl3; got != 0 {
		t.Fatalf("bowl III = %d, want 0", got)
	}
	if got := player.Keys; got != 0 {
		t.Fatalf("keys = %d, want 0 after refunding and re-spending the cult-top key", got)
	}
	if got := player.Resources.Coins; got != 2 {
		t.Fatalf("coins = %d, want 2", got)
	}
	if occupier := gs.CultTracks.Position10Occupied[CultWater]; occupier != "p1" {
		t.Fatalf("water cult position 10 occupier = %q, want p1", occupier)
	}
}
