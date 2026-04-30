package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestIceFactionSetupSelectsStartingTerrainAndPlacesIce(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("ice", factions.NewIceMaidens()); err != nil {
		t.Fatalf("add ice: %v", err)
	}
	gs.TurnOrder = []string{"ice"}
	gs.InitializeSetupSequence()

	hex := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(hex, models.TerrainForest); err != nil {
		t.Fatalf("set terrain: %v", err)
	}

	if err := NewSetupDwellingAction("ice", hex).Execute(gs); err != nil {
		t.Fatalf("setup dwelling: %v", err)
	}

	player := gs.GetPlayer("ice")
	if !player.HasStartingTerrain || player.StartingTerrain != models.TerrainForest {
		t.Fatalf("starting terrain = %v / %v, want Forest", player.HasStartingTerrain, player.StartingTerrain)
	}
	if got := gs.Map.GetHex(hex).Terrain; got != models.TerrainIce {
		t.Fatalf("terrain after setup = %v, want Ice", got)
	}
}

func TestIceFactionTransformUsesWorkerExchangeRate(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("ice", factions.NewIceMaidens()); err != nil {
		t.Fatalf("add ice: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"ice"}
	gs.PendingSpades = map[string]int{"ice": 1}

	player := gs.GetPlayer("ice")
	player.HasStartingTerrain = true
	player.StartingTerrain = models.TerrainMountain
	player.DiggingLevel = 2
	gs.updateFactionDiggingLevel(player)
	player.Resources.Coins = 5
	player.Resources.Workers = 1
	player.Resources.Priests = 1

	source := board.NewHex(0, 0)
	gs.Map.Hexes[source] = &board.MapHex{Coord: source, Terrain: models.TerrainIce}
	gs.Map.PlaceBuilding(source, &models.Building{Type: models.BuildingDwelling, PlayerID: "ice", Faction: models.FactionIceMaidens, PowerValue: 1})

	target := board.NewHex(1, 0)
	if err := gs.Map.TransformTerrain(target, models.TerrainDesert); err != nil {
		t.Fatalf("set target: %v", err)
	}

	if err := NewTransformAndBuildAction("ice", target, false, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("ice transform: %v", err)
	}

	if got := gs.Map.GetHex(target).Terrain; got != models.TerrainIce {
		t.Fatalf("terrain = %v, want Ice", got)
	}
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers after ice terraform = %d, want 0", got)
	}
	if got := player.Resources.Coins; got != 5 {
		t.Fatalf("coins after ice terraform = %d, want 5", got)
	}
	if got := player.Resources.Priests; got != 1 {
		t.Fatalf("priests after ice terraform = %d, want 1", got)
	}
	if pending := gs.PendingSpades["ice"]; pending != 0 {
		t.Fatalf("pending spades after ice terraform = %d, want 0", pending)
	}
}

func TestIceFactionTransformFromSelectedStartingTerrainStillCostsOneSpade(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("ice", factions.NewIceMaidens()); err != nil {
		t.Fatalf("add ice: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"ice"}

	player := gs.GetPlayer("ice")
	player.HasStartingTerrain = true
	player.StartingTerrain = models.TerrainMountain
	player.DiggingLevel = 2
	gs.updateFactionDiggingLevel(player)
	player.Resources.Workers = 1

	source := board.NewHex(0, 0)
	gs.Map.Hexes[source] = &board.MapHex{Coord: source, Terrain: models.TerrainIce}
	gs.Map.PlaceBuilding(source, &models.Building{Type: models.BuildingDwelling, PlayerID: "ice", Faction: models.FactionIceMaidens, PowerValue: 1})

	target := board.NewHex(1, 0)
	if err := gs.Map.TransformTerrain(target, models.TerrainMountain); err != nil {
		t.Fatalf("set target: %v", err)
	}

	if err := NewTransformAndBuildAction("ice", target, false, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("ice transform from starting terrain: %v", err)
	}

	if got := gs.Map.GetHex(target).Terrain; got != models.TerrainIce {
		t.Fatalf("terrain = %v, want Ice", got)
	}
	if got := player.Resources.Workers; got != 0 {
		t.Fatalf("workers after starting-terrain ice terraform = %d, want 0", got)
	}
}

func TestDragonlordsTransformRemovesPowerTokenAndCreatesVolcano(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("dragon", factions.NewDragonlords()); err != nil {
		t.Fatalf("add dragon: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"dragon"}

	target := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(target, models.TerrainForest); err != nil {
		t.Fatalf("set target: %v", err)
	}
	before := gs.GetPlayer("dragon").Resources.Power.TotalPower()

	if err := NewTransformAndBuildAction("dragon", target, false, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("dragon transform: %v", err)
	}

	if got := gs.Map.GetHex(target).Terrain; got != models.TerrainVolcano {
		t.Fatalf("terrain = %v, want Volcano", got)
	}
	if got := gs.GetPlayer("dragon").Resources.Power.TotalPower(); got != before-1 {
		t.Fatalf("total power tokens = %d, want %d", got, before-1)
	}
}

func TestAcolytesTransformSpendsCultSteps(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add acolytes: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"acolytes"}

	target := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(target, models.TerrainForest); err != nil {
		t.Fatalf("set target: %v", err)
	}

	if err := NewTransformAndBuildAction("acolytes", target, false, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("acolytes transform: %v", err)
	}

	player := gs.GetPlayer("acolytes")
	if got := gs.Map.GetHex(target).Terrain; got != models.TerrainVolcano {
		t.Fatalf("terrain = %v, want Volcano", got)
	}
	if player.CultPositions[CultFire] != 0 && player.CultPositions[CultWater] != 0 && player.CultPositions[CultEarth] != 0 && player.CultPositions[CultAir] != 0 {
		t.Fatalf("expected one cult track to pay down to 0, got %+v", player.CultPositions)
	}
}

func TestAcolytesTransformUsesSelectedCultTrack(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add acolytes: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"acolytes"}

	player := gs.GetPlayer("acolytes")
	player.CultPositions[CultFire] = 6
	player.CultPositions[CultWater] = 3
	player.CultPositions[CultEarth] = 5
	player.CultPositions[CultAir] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultFire] = 6
	gs.CultTracks.PlayerPositions["acolytes"][CultWater] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultEarth] = 5
	gs.CultTracks.PlayerPositions["acolytes"][CultAir] = 3

	target := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(target, models.TerrainForest); err != nil {
		t.Fatalf("set target: %v", err)
	}

	track := CultEarth
	action := NewTransformAndBuildAction("acolytes", target, false, models.TerrainTypeUnknown)
	action.AcolytesCultTrack = &track
	if err := action.Execute(gs); err != nil {
		t.Fatalf("acolytes selected-track transform: %v", err)
	}

	if got := gs.Map.GetHex(target).Terrain; got != models.TerrainVolcano {
		t.Fatalf("terrain = %v, want Volcano", got)
	}
	if got := player.CultPositions[CultEarth]; got != 2 {
		t.Fatalf("earth cult = %d, want 2", got)
	}
	if got := player.CultPositions[CultFire]; got != 6 {
		t.Fatalf("fire cult = %d, want 6", got)
	}
}

func TestAcolytesTransformRejectsInsufficientSelectedCultTrack(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add acolytes: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"acolytes"}

	player := gs.GetPlayer("acolytes")
	player.CultPositions[CultFire] = 6
	player.CultPositions[CultWater] = 2
	player.CultPositions[CultEarth] = 5
	player.CultPositions[CultAir] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultFire] = 6
	gs.CultTracks.PlayerPositions["acolytes"][CultWater] = 2
	gs.CultTracks.PlayerPositions["acolytes"][CultEarth] = 5
	gs.CultTracks.PlayerPositions["acolytes"][CultAir] = 3

	target := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(target, models.TerrainForest); err != nil {
		t.Fatalf("set target: %v", err)
	}

	track := CultWater
	action := NewTransformAndBuildAction("acolytes", target, false, models.TerrainTypeUnknown)
	action.AcolytesCultTrack = &track
	if err := action.Execute(gs); err == nil {
		t.Fatalf("expected insufficient selected cult track to fail")
	}
	if got := gs.Map.GetHex(target).Terrain; got != models.TerrainForest {
		t.Fatalf("terrain = %v, want Forest after rejected transform", got)
	}
	if got := player.CultPositions[CultWater]; got != 2 {
		t.Fatalf("water cult = %d, want 2 after rejected transform", got)
	}
}

func TestAcolytesReplayCultSpendPreservesCurrentRoundCultRewardTrack(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add acolytes: %v", err)
	}
	gs.ReplayMode = map[string]bool{"__replay__": true}
	gs.Round = 1
	gs.ScoringTiles.Tiles = []ScoringTile{
		{Type: ScoringStrongholdFire, CultTrack: CultFire, CultThreshold: 2, CultRewardType: CultRewardWorker, CultRewardAmount: 1},
	}

	player := gs.GetPlayer("acolytes")
	player.CultPositions[CultFire] = 6
	player.CultPositions[CultWater] = 3
	player.CultPositions[CultEarth] = 5
	player.CultPositions[CultAir] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultFire] = 6
	gs.CultTracks.PlayerPositions["acolytes"][CultWater] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultEarth] = 5
	gs.CultTracks.PlayerPositions["acolytes"][CultAir] = 3

	if err := gs.spendAcolytesCultSteps("acolytes", 3); err != nil {
		t.Fatalf("spend cult steps: %v", err)
	}
	if got := player.CultPositions[CultFire]; got != 6 {
		t.Fatalf("fire cult = %d, want 6", got)
	}
	if got := player.CultPositions[CultWater]; got != 0 {
		t.Fatalf("water cult = %d, want 0", got)
	}
}

func TestAcolytesReplayCultSpendUsesConfiguredQueue(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add acolytes: %v", err)
	}
	gs.ReplayMode = map[string]bool{"__replay__": true}
	gs.ReplayAcolytesCultTracks = map[string][]CultTrack{
		"acolytes": {CultWater, CultFire},
	}
	gs.ReplayAcolytesCultTrackIndex = map[string]int{"acolytes": 0}

	player := gs.GetPlayer("acolytes")
	player.CultPositions[CultFire] = 6
	player.CultPositions[CultWater] = 3
	player.CultPositions[CultEarth] = 5
	player.CultPositions[CultAir] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultFire] = 6
	gs.CultTracks.PlayerPositions["acolytes"][CultWater] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultEarth] = 5
	gs.CultTracks.PlayerPositions["acolytes"][CultAir] = 3

	if err := gs.spendAcolytesCultSteps("acolytes", 3); err != nil {
		t.Fatalf("first spend cult steps: %v", err)
	}
	if got := player.CultPositions[CultWater]; got != 0 {
		t.Fatalf("water cult = %d, want 0 after configured payment", got)
	}
	if got := gs.ReplayAcolytesCultTrackIndex["acolytes"]; got != 1 {
		t.Fatalf("replay queue index = %d, want 1 after first spend", got)
	}

	if err := gs.spendAcolytesCultSteps("acolytes", 3); err != nil {
		t.Fatalf("second spend cult steps: %v", err)
	}
	if got := player.CultPositions[CultFire]; got != 3 {
		t.Fatalf("fire cult = %d, want 3 after second configured payment", got)
	}
	if got := gs.ReplayAcolytesCultTrackIndex["acolytes"]; got != 2 {
		t.Fatalf("replay queue index = %d, want 2 after second spend", got)
	}
}

func TestAcolytesCultSpendClearsMilestoneClaimsForReclimb(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add acolytes: %v", err)
	}

	player := gs.GetPlayer("acolytes")
	player.CultPositions[CultWater] = 5
	player.Resources.Power = NewPowerSystem(0, 7, 0)
	gs.CultTracks.PlayerPositions["acolytes"][CultWater] = 5
	gs.CultTracks.BonusPositionsClaimed["acolytes"][CultWater][5] = true

	if err := gs.spendAcolytesCultSteps("acolytes", 3); err != nil {
		t.Fatalf("spend cult steps: %v", err)
	}
	if got := player.CultPositions[CultWater]; got != 2 {
		t.Fatalf("water cult after spend = %d, want 2", got)
	}
	if gs.CultTracks.BonusPositionsClaimed["acolytes"][CultWater][5] {
		t.Fatalf("expected water level 5 bonus claim to be cleared after dropping below it")
	}

	beforeBowl3 := player.Resources.Power.Bowl3
	if _, err := gs.AdvanceCultTrack("acolytes", CultWater, 3); err != nil {
		t.Fatalf("re-advance cult track: %v", err)
	}
	if got := player.CultPositions[CultWater]; got != 5 {
		t.Fatalf("water cult after re-advance = %d, want 5", got)
	}
	if got := player.Resources.Power.Bowl3; got != beforeBowl3+3 {
		t.Fatalf("bowl 3 power after re-advance = %d, want %d", got, beforeBowl3+3)
	}
}

func TestAcolytesNormalCultSpendUsesHighestTrack(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("acolytes", factions.NewAcolytes()); err != nil {
		t.Fatalf("add acolytes: %v", err)
	}

	player := gs.GetPlayer("acolytes")
	player.CultPositions[CultFire] = 6
	player.CultPositions[CultWater] = 3
	player.CultPositions[CultEarth] = 5
	player.CultPositions[CultAir] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultFire] = 6
	gs.CultTracks.PlayerPositions["acolytes"][CultWater] = 3
	gs.CultTracks.PlayerPositions["acolytes"][CultEarth] = 5
	gs.CultTracks.PlayerPositions["acolytes"][CultAir] = 3

	if err := gs.spendAcolytesCultSteps("acolytes", 3); err != nil {
		t.Fatalf("spend cult steps: %v", err)
	}
	if got := player.CultPositions[CultFire]; got != 3 {
		t.Fatalf("fire cult = %d, want 3", got)
	}
}

func TestSelkiesCanBuildRiverDwellingBetweenIceBuildings(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("selkies", factions.NewSelkies()); err != nil {
		t.Fatalf("add selkies: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"selkies"}
	player := gs.GetPlayer("selkies")
	player.Resources.Workers = 3
	player.Resources.Coins = 5

	river := board.NewHex(0, 0)
	iceA := board.NewHex(1, 0)
	iceB := board.NewHex(-1, 0)
	gs.Map.Hexes[river] = &board.MapHex{Coord: river, Terrain: models.TerrainRiver}
	gs.Map.Hexes[iceA] = &board.MapHex{Coord: iceA, Terrain: models.TerrainIce}
	gs.Map.Hexes[iceB] = &board.MapHex{Coord: iceB, Terrain: models.TerrainIce}
	gs.Map.RiverHexes[river] = true
	gs.Map.PlaceBuilding(iceA, &models.Building{Type: models.BuildingDwelling, PlayerID: "selkies", Faction: models.FactionSelkies, PowerValue: 1})
	gs.Map.PlaceBuilding(iceB, &models.Building{Type: models.BuildingDwelling, PlayerID: "selkies", Faction: models.FactionSelkies, PowerValue: 1})

	if err := NewTransformAndBuildAction("selkies", river, true, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("selkies river dwelling: %v", err)
	}

	if building := gs.Map.GetHex(river).Building; building == nil || building.PlayerID != "selkies" {
		t.Fatalf("expected river dwelling, got %+v", building)
	}
	if got := player.Resources.Workers; got != 1 {
		t.Fatalf("workers after river dwelling = %d, want 1", got)
	}
	if got := player.VictoryPoints; got != 22 {
		t.Fatalf("victory points after river dwelling = %d, want 22", got)
	}
}

func TestRiverwalkersCannotGainOrUseSpades(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("river", factions.NewRiverwalkers()); err != nil {
		t.Fatalf("add riverwalkers: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"river"}
	player := gs.GetPlayer("river")
	player.Resources.Power = NewPowerSystem(0, 0, 6)

	if err := NewPowerAction("river", PowerActionSpade1).Validate(gs); err == nil {
		t.Fatalf("expected riverwalkers spade power action to be invalid")
	}

	gs.grantCultReward("river", player, CultRewardSpade, 2)
	if got := gs.PendingCultRewardSpades["river"]; got != 0 {
		t.Fatalf("expected no pending cult spades for riverwalkers, got %d", got)
	}

	gs.BonusCards.PlayerCards["river"] = BonusCardSpade
	if err := NewBonusCardSpadeAction("river", board.NewHex(0, 0), false, models.TerrainTypeUnknown).Validate(gs); err == nil {
		t.Fatalf("expected riverwalkers bonus-card spade action to be invalid")
	}
}

func TestDragonlordsAndRiverwalkersCannotAdvanceDigging(t *testing.T) {
	for _, tt := range []struct {
		name    string
		faction factions.Faction
	}{
		{name: "dragonlords", faction: factions.NewDragonlords()},
		{name: "riverwalkers", faction: factions.NewRiverwalkers()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGameState()
			if err := gs.AddPlayer("player", tt.faction); err != nil {
				t.Fatalf("add player: %v", err)
			}
			player := gs.GetPlayer("player")
			player.Resources.Coins = 100
			player.Resources.Workers = 100
			player.Resources.Priests = 5

			if err := NewAdvanceDiggingAction("player").Execute(gs); err == nil {
				t.Fatalf("%s should not be able to advance digging", tt.name)
			}
			if got := player.DiggingLevel; got != 0 {
				t.Fatalf("digging level = %d, want 0", got)
			}
		})
	}
}

func TestRiverwalkersShippingIsFixedAtOne(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("river", factions.NewRiverwalkers()); err != nil {
		t.Fatalf("add riverwalkers: %v", err)
	}
	player := gs.GetPlayer("river")
	player.Resources.Coins = 100
	player.Resources.Priests = 5

	if got := player.ShippingLevel; got != 1 {
		t.Fatalf("starting shipping = %d, want 1", got)
	}
	if err := NewAdvanceShippingAction("river").Execute(gs); err == nil {
		t.Fatalf("expected paid shipping advancement to fail")
	}
	if err := gs.AdvanceShippingLevel("river"); err == nil {
		t.Fatalf("expected direct shipping advancement to fail")
	}
	gs.ApplyTownTileBenefits("river", models.TownTile4Points)

	if got := player.ShippingLevel; got != 1 {
		t.Fatalf("shipping after attempted upgrades = %d, want 1", got)
	}
	if got := player.VictoryPoints; got != 24 {
		t.Fatalf("VP after shipping town tile = %d, want 24", got)
	}
}

func TestRiverwalkersUnlockTerrainBeforeBuilding(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("river", factions.NewRiverwalkers()); err != nil {
		t.Fatalf("add riverwalkers: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"river"}
	player := gs.GetPlayer("river")
	player.Resources.Workers = 3
	player.Resources.Coins = 10
	player.Resources.Priests = 0
	player.UnlockedTerrains = map[models.TerrainType]bool{models.TerrainForest: true}

	start := board.NewHex(0, 0)
	target := board.NewHex(1, 0)
	river := board.NewHex(1, -1)
	gs.Map.Hexes[start] = &board.MapHex{Coord: start, Terrain: models.TerrainForest}
	gs.Map.Hexes[target] = &board.MapHex{Coord: target, Terrain: models.TerrainMountain}
	gs.Map.Hexes[river] = &board.MapHex{Coord: river, Terrain: models.TerrainRiver}
	gs.Map.RiverHexes[river] = true
	gs.Map.PlaceBuilding(start, &models.Building{Type: models.BuildingDwelling, PlayerID: "river", Faction: models.FactionRiverwalkers, PowerValue: 1})

	build := NewTransformAndBuildAction("river", target, true, models.TerrainTypeUnknown)
	if err := build.Validate(gs); err == nil {
		t.Fatalf("expected riverwalkers build to require unlocked terrain")
	}

	gs.GainPriestsForReason("river", 1, "power_action")
	unlock := NewSelectRiverwalkersPriestChoiceAction("river", false, models.TerrainMountain)
	if err := unlock.Execute(gs); err != nil {
		t.Fatalf("unlock mountain: %v", err)
	}
	if !player.UnlockedTerrains[models.TerrainMountain] {
		t.Fatalf("expected mountain terrain to be unlocked")
	}
	if player.Resources.Priests != 0 || player.Resources.Coins != 9 {
		t.Fatalf("unlock cost = %d priests/%d coins, want 0 priests/9 coins", player.Resources.Priests, player.Resources.Coins)
	}

	if err := build.Execute(gs); err != nil {
		t.Fatalf("build on unlocked mountain: %v", err)
	}
	if building := gs.Map.GetHex(target).Building; building == nil || building.PlayerID != "river" {
		t.Fatalf("expected riverwalker dwelling on target, got %+v", building)
	}
}

func TestRiverwalkersPriestGainQueuesChoice(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("river", factions.NewRiverwalkers()); err != nil {
		t.Fatalf("add riverwalkers: %v", err)
	}
	player := gs.GetPlayer("river")
	player.Resources.Coins = 5
	player.Resources.Priests = 0

	if gained := gs.GainPriestsForReason("river", 1, "income"); gained != 0 {
		t.Fatalf("immediate priests gained = %d, want 0", gained)
	}
	if player.Resources.Priests != 0 {
		t.Fatalf("priests in resources = %d, want 0 before choice", player.Resources.Priests)
	}
	if gs.PendingRiverwalkersPriestChoice == nil {
		t.Fatalf("expected pending riverwalkers priest choice")
	}
	if got := gs.PendingRiverwalkersPriestChoice.PriestsRemaining; got != 1 {
		t.Fatalf("pending priests = %d, want 1", got)
	}
	if err := NewSelectRiverwalkersPriestChoiceAction("river", true, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("take priest: %v", err)
	}
	if player.Resources.Priests != 1 {
		t.Fatalf("priests in resources = %d, want 1 after choice", player.Resources.Priests)
	}
	if gs.PendingRiverwalkersPriestChoice != nil {
		t.Fatalf("expected pending choice to clear")
	}
}

func TestRiverwalkersPriestChoiceStartsActionPhaseAfterIncome(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("river", factions.NewRiverwalkers()); err != nil {
		t.Fatalf("add riverwalkers: %v", err)
	}
	gs.Phase = PhaseIncome
	gs.TurnOrder = []string{"river"}
	player := gs.GetPlayer("river")
	player.Resources.Coins = 5
	player.Resources.Priests = 0

	gs.GainPriestsForReason("river", 1, "income")
	if err := NewSelectRiverwalkersPriestChoiceAction("river", true, models.TerrainTypeUnknown).Execute(gs); err != nil {
		t.Fatalf("take priest: %v", err)
	}
	if gs.Phase != PhaseAction {
		t.Fatalf("phase = %v, want action", gs.Phase)
	}
}

func TestRiverwalkersUnlockCostIgnoresIceAndVolcanoStartingTerrainButCountsShapeshiftersHome(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("river", factions.NewRiverwalkers()); err != nil {
		t.Fatalf("add riverwalkers: %v", err)
	}
	if err := gs.AddPlayer("ice", factions.NewIceMaidens()); err != nil {
		t.Fatalf("add ice: %v", err)
	}
	if err := gs.AddPlayer("dragon", factions.NewDragonlords()); err != nil {
		t.Fatalf("add dragonlords: %v", err)
	}
	ice := gs.GetPlayer("ice")
	ice.StartingTerrain = models.TerrainMountain
	ice.HasStartingTerrain = true
	dragon := gs.GetPlayer("dragon")
	dragon.StartingTerrain = models.TerrainDesert
	dragon.HasStartingTerrain = true
	shapeFaction := factions.NewShapeshifters()
	gs.Players["shape"] = &Player{
		ID:                 "shape",
		Faction:            shapeFaction,
		Resources:          NewResourcePool(shapeFaction.GetStartingResources()),
		StartingTerrain:    models.TerrainWasteland,
		HasStartingTerrain: true,
	}
	river := gs.GetPlayer("river")

	if got := gs.riverwalkersUnlockCost(river, models.TerrainMountain); got != 1 {
		t.Fatalf("ice starting terrain unlock cost = %d, want 1", got)
	}
	if got := gs.riverwalkersUnlockCost(river, models.TerrainDesert); got != 1 {
		t.Fatalf("volcano starting terrain unlock cost = %d, want 1", got)
	}
	if got := gs.riverwalkersUnlockCost(river, models.TerrainWasteland); got != 2 {
		t.Fatalf("shapeshifters home terrain unlock cost = %d, want 2", got)
	}
}

func TestVolcanoTransformCostIgnoresIceAndVolcanoStartingTerrainButCountsShapeshiftersHome(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("dragon", factions.NewDragonlords()); err != nil {
		t.Fatalf("add dragonlords: %v", err)
	}
	if err := gs.AddPlayer("ice", factions.NewIceMaidens()); err != nil {
		t.Fatalf("add ice: %v", err)
	}
	if err := gs.AddPlayer("river", factions.NewRiverwalkers()); err != nil {
		t.Fatalf("add riverwalkers: %v", err)
	}
	ice := gs.GetPlayer("ice")
	ice.StartingTerrain = models.TerrainMountain
	ice.HasStartingTerrain = true
	river := gs.GetPlayer("river")
	river.StartingTerrain = models.TerrainLake
	river.HasStartingTerrain = true
	dragon := gs.GetPlayer("dragon")
	dragon.StartingTerrain = models.TerrainDesert
	dragon.HasStartingTerrain = true
	shapeFaction := factions.NewShapeshifters()
	gs.Players["shape"] = &Player{
		ID:                 "shape",
		Faction:            shapeFaction,
		Resources:          NewResourcePool(shapeFaction.GetStartingResources()),
		StartingTerrain:    models.TerrainWasteland,
		HasStartingTerrain: true,
	}

	if got := volcanoTransformCost(gs, dragon, models.TerrainMountain); got != 1 {
		t.Fatalf("dragonlords ice starting terrain cost = %d, want 1", got)
	}
	if got := volcanoTransformCost(gs, dragon, models.TerrainLake); got != 1 {
		t.Fatalf("dragonlords riverwalkers starting terrain cost = %d, want 1", got)
	}
	if got := acolytesCultTransformCost(gs, dragon, models.TerrainMountain); got != 3 {
		t.Fatalf("acolytes ice starting terrain cost = %d, want 3", got)
	}
	if got := firewalkersVPTransformCost(gs, dragon, models.TerrainMountain); got != 4 {
		t.Fatalf("firewalkers ice starting terrain cost = %d, want 4", got)
	}
	if got := volcanoTransformCost(gs, dragon, models.TerrainDesert); got != 1 {
		t.Fatalf("dragonlords volcano starting terrain cost = %d, want 1", got)
	}
	if got := volcanoTransformCost(gs, dragon, models.TerrainWasteland); got != 2 {
		t.Fatalf("dragonlords shapeshifters home terrain cost = %d, want 2", got)
	}
	if got := acolytesCultTransformCost(gs, dragon, models.TerrainWasteland); got != 4 {
		t.Fatalf("acolytes shapeshifters home terrain cost = %d, want 4", got)
	}
	if got := firewalkersVPTransformCost(gs, dragon, models.TerrainWasteland); got != 6 {
		t.Fatalf("firewalkers shapeshifters home terrain cost = %d, want 6", got)
	}
}

func TestShapeshiftersLeechAcceptedAddsPowerTokenForVP(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("shape", factions.NewShapeshifters()); err != nil {
		t.Fatalf("add shapeshifters: %v", err)
	}
	if err := gs.AddPlayer("neighbor", factions.NewAuren()); err != nil {
		t.Fatalf("add neighbor: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"shape", "neighbor"}
	shape := gs.GetPlayer("shape")
	shape.VictoryPoints = 20
	beforeBowl3 := shape.Resources.Power.Bowl3
	neighbor := gs.GetPlayer("neighbor")
	neighbor.Resources.Power = NewPowerSystem(2, 0, 0)

	sourceHex := board.NewHex(0, 0)
	neighborHex := board.NewHex(1, 0)
	gs.Map.PlaceBuilding(sourceHex, &models.Building{Type: models.BuildingDwelling, PlayerID: "shape", Faction: models.FactionShapeshifters, PowerValue: 1})
	gs.Map.PlaceBuilding(neighborHex, &models.Building{Type: models.BuildingDwelling, PlayerID: "neighbor", Faction: models.FactionAuren, PowerValue: 1})

	gs.TriggerPowerLeech(sourceHex, "shape")
	if err := NewAcceptPowerLeechAction("neighbor", 0).Execute(gs); err != nil {
		t.Fatalf("accept leech: %v", err)
	}

	if got := shape.VictoryPoints; got != 19 {
		t.Fatalf("shapeshifters VP = %d, want 19", got)
	}
	if got := shape.Resources.Power.Bowl3; got != beforeBowl3+1 {
		t.Fatalf("shapeshifters bowl III = %d, want %d", got, beforeBowl3+1)
	}
}

func TestShapeshiftersBGAReplayWaitsForExplicitLeechBonusRow(t *testing.T) {
	gs := NewGameState()
	gs.ReplayMode = map[string]bool{"__replay__": true, "__bga__": true}
	if err := gs.AddPlayer("shape", factions.NewShapeshifters()); err != nil {
		t.Fatalf("add shapeshifters: %v", err)
	}
	if err := gs.AddPlayer("neighbor", factions.NewAuren()); err != nil {
		t.Fatalf("add neighbor: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"shape", "neighbor"}
	shape := gs.GetPlayer("shape")
	shape.VictoryPoints = 20
	beforeBowl3 := shape.Resources.Power.Bowl3
	neighbor := gs.GetPlayer("neighbor")
	neighbor.Resources.Power = NewPowerSystem(2, 0, 0)

	sourceHex := board.NewHex(0, 0)
	neighborHex := board.NewHex(1, 0)
	gs.Map.PlaceBuilding(sourceHex, &models.Building{Type: models.BuildingDwelling, PlayerID: "shape", Faction: models.FactionShapeshifters, PowerValue: 1})
	gs.Map.PlaceBuilding(neighborHex, &models.Building{Type: models.BuildingDwelling, PlayerID: "neighbor", Faction: models.FactionAuren, PowerValue: 1})

	gs.TriggerPowerLeech(sourceHex, "shape")
	if err := NewAcceptPowerLeechAction("neighbor", 0).Execute(gs); err != nil {
		t.Fatalf("accept leech: %v", err)
	}

	if got := shape.VictoryPoints; got != 20 {
		t.Fatalf("shapeshifters VP = %d, want 20 before explicit BGA row", got)
	}
	if got := shape.Resources.Power.Bowl3; got != beforeBowl3 {
		t.Fatalf("shapeshifters bowl III = %d, want %d before explicit BGA row", got, beforeBowl3)
	}
}

func TestShapeshiftersAllLeechDeclinedGainsPower(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("shape", factions.NewShapeshifters()); err != nil {
		t.Fatalf("add shapeshifters: %v", err)
	}
	if err := gs.AddPlayer("neighbor", factions.NewAuren()); err != nil {
		t.Fatalf("add neighbor: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"shape", "neighbor"}
	shape := gs.GetPlayer("shape")
	shape.Resources.Power = NewPowerSystem(1, 0, 0)
	neighbor := gs.GetPlayer("neighbor")
	neighbor.Resources.Power = NewPowerSystem(2, 0, 0)

	sourceHex := board.NewHex(0, 0)
	neighborHex := board.NewHex(1, 0)
	gs.Map.PlaceBuilding(sourceHex, &models.Building{Type: models.BuildingDwelling, PlayerID: "shape", Faction: models.FactionShapeshifters, PowerValue: 1})
	gs.Map.PlaceBuilding(neighborHex, &models.Building{Type: models.BuildingDwelling, PlayerID: "neighbor", Faction: models.FactionAuren, PowerValue: 1})

	gs.TriggerPowerLeech(sourceHex, "shape")
	if err := NewDeclinePowerLeechAction("neighbor", 0).Execute(gs); err != nil {
		t.Fatalf("decline leech: %v", err)
	}

	if shape.Resources.Power.Bowl1 != 0 || shape.Resources.Power.Bowl2 != 1 || shape.Resources.Power.Bowl3 != 0 {
		t.Fatalf("shapeshifters power after declined leech = %d/%d/%d, want 0/1/0", shape.Resources.Power.Bowl1, shape.Resources.Power.Bowl2, shape.Resources.Power.Bowl3)
	}
}

func TestShapeshiftersStrongholdCanShiftHomeTerrain(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("shape", factions.NewShapeshifters()); err != nil {
		t.Fatalf("add shapeshifters: %v", err)
	}
	gs.Phase = PhaseAction
	gs.TurnOrder = []string{"shape"}
	player := gs.GetPlayer("shape")
	player.HasStrongholdAbility = true
	player.HasStartingTerrain = true
	player.StartingTerrain = models.TerrainPlains
	player.Resources.Power = NewPowerSystem(0, 0, 5)

	action := NewShapeshiftersShiftTerrainAction("shape", models.TerrainMountain)
	if err := action.Execute(gs); err != nil {
		t.Fatalf("shift home terrain: %v", err)
	}
	if got := player.StartingTerrain; got != models.TerrainMountain {
		t.Fatalf("home terrain = %v, want Mountain", got)
	}
	if got := player.Resources.Power.Bowl3; got != 0 {
		t.Fatalf("bowl III after shift = %d, want 0", got)
	}
}

func TestSnowShamansPassUsesQueuedShippingUpgrade(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("snow", factions.NewSnowShamans()); err != nil {
		t.Fatalf("add snow shamans: %v", err)
	}
	gs.Phase = PhaseAction
	gs.Round = 6
	gs.TurnOrder = []string{"snow"}

	player := gs.GetPlayer("snow")
	if err := gs.QueueSnowShamansPassUpgrade("snow", SnowShamansPassUpgradeShipping); err != nil {
		t.Fatalf("queue pass upgrade: %v", err)
	}
	if err := NewPassAction("snow", nil).Execute(gs); err != nil {
		t.Fatalf("pass: %v", err)
	}

	if got := player.ShippingLevel; got != 1 {
		t.Fatalf("shipping level = %d, want 1", got)
	}
	if got := player.DiggingLevel; got != 0 {
		t.Fatalf("digging level = %d, want 0", got)
	}
}

func TestSnowShamansPassUsesActionSelectedShippingUpgrade(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("snow", factions.NewSnowShamans()); err != nil {
		t.Fatalf("add snow shamans: %v", err)
	}
	gs.Phase = PhaseAction
	gs.Round = 6
	gs.TurnOrder = []string{"snow"}

	player := gs.GetPlayer("snow")
	upgrade := SnowShamansPassUpgradeShipping
	action := NewPassAction("snow", nil)
	action.SnowShamansUpgrade = &upgrade
	if err := action.Execute(gs); err != nil {
		t.Fatalf("pass: %v", err)
	}

	if got := player.ShippingLevel; got != 1 {
		t.Fatalf("shipping level = %d, want 1", got)
	}
	if got := player.DiggingLevel; got != 0 {
		t.Fatalf("digging level = %d, want 0", got)
	}
}
