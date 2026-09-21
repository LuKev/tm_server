package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// User-confirmed rule: a leftover ACT6 spade on the second space cannot be
// topped up with paid spades. Dwelling choice is tested separately.
func TestPowerSpadeSecondSpaceRestrictions(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewHalflings())
	p := gs.GetPlayer("p1")
	p.Resources.Power.Bowl3 = 6
	p.Resources.Workers, p.Resources.Coins = 20, 20
	start, first, second := board.NewHex(0, 1), board.NewHex(1, 0), board.NewHex(1, 1)
	gs.Map.GetHex(start).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionHalflings, PowerValue: 1}
	gs.Map.GetHex(first).Terrain = models.TerrainSwamp
	gs.Map.GetHex(second).Terrain = models.TerrainSwamp
	a := NewPowerActionWithTransform("p1", PowerActionSpade2, first, true)
	a.SecondTargetHex = &second
	terrain := models.TerrainPlains
	a.SecondTargetTerrain = &terrain
	if err := a.Validate(gs); err != nil {
		t.Fatal(err)
	}
	gs.Map.GetHex(second).Terrain = models.TerrainMountain
	if err := a.Validate(gs); err == nil {
		t.Error("second ACT6 space illegally allows paid extra spades")
	}
	// The same spare spade remains legal if it stops at intermediate terrain.
	terrain = models.TerrainForest
	if err := a.Execute(gs); err != nil {
		t.Fatal(err)
	}
	if gs.Map.GetHex(second).Terrain != models.TerrainForest {
		t.Fatal("leftover free spade failed to stop at intermediate terrain")
	}
}

func TestAlchemistsCannotSpendSpadePowerBeforeOptionalDwelling(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewAlchemists())
	p := gs.GetPlayer("p1")
	p.HasStrongholdAbility = true
	p.Resources.Power.Bowl1, p.Resources.Power.Bowl2, p.Resources.Power.Bowl3 = 0, 8, 4
	p.Resources.Workers, p.Resources.Coins = 7, 1
	start, target := board.NewHex(0, 1), board.NewHex(1, 0)
	gs.Map.GetHex(start).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionAlchemists, PowerValue: 1}
	gs.Map.GetHex(target).Terrain = models.TerrainMountain
	a := NewPowerActionWithTransform("p1", PowerActionSpade1, target, true)
	// Three used spades would produce six power, two ending in Bowl III.
	// Those prospective conversion coins cannot pay this action's dwelling.
	if err := a.Execute(gs); err == nil {
		t.Fatal("Alchemists financed dwelling with power not yet gained")
	}
	if p.Resources.Coins != 1 || p.Resources.Workers != 7 || p.Resources.Power.Bowl3 != 4 || gs.Map.GetHex(target).Terrain != models.TerrainMountain || !gs.PowerActions.IsAvailable(PowerActionSpade1) {
		t.Fatal("rejected combined action mutated state")
	}
	a.BuildDwelling = false
	if err := a.Execute(gs); err != nil {
		t.Fatal(err)
	}
	if p.Resources.Power.Bowl3 != 2 || gs.Map.GetHex(target).Building != nil {
		t.Fatal("transform-only must finish before gained power becomes convertible")
	}
}

func TestPowerSpadeExplicitTerrainAndCosts(t *testing.T) {
	// The printed terrain wheel has Plains-Swamp-Lake-Forest on one side.
	// A Halfling may stop Forest->Lake for one free spade, or pay two
	// workers exchanges to continue to Plains, but may not build on Lake.
	for _, tc := range []struct {
		name        string
		target      models.TerrainType
		power       PowerActionType
		workers     int
		build       bool
		wantWorkers int
		wantError   bool
	}{
		{"one free intermediate", models.TerrainLake, PowerActionSpade1, 0, false, 0, false},
		{"two free intermediate", models.TerrainSwamp, PowerActionSpade2, 0, false, 0, false},
		{"optional paid completion", models.TerrainPlains, PowerActionSpade1, 6, false, 0, false},
		{"paid intermediate toward home", models.TerrainSwamp, PowerActionSpade1, 3, false, 0, false},
		{"no building on intermediate", models.TerrainLake, PowerActionSpade1, 10, true, 10, true},
		{"unaffordable completion is atomic", models.TerrainPlains, PowerActionSpade1, 0, false, 0, true},
		{"cannot buy away from home", models.TerrainWasteland, PowerActionSpade1, 10, false, 10, true},
		{"cannot make river", models.TerrainRiver, PowerActionSpade1, 10, false, 10, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := NewGameState()
			gs.AddPlayer("p1", factions.NewHalflings())
			p := gs.GetPlayer("p1")
			p.Resources.Power.Bowl1, p.Resources.Power.Bowl2, p.Resources.Power.Bowl3 = 0, 0, 6
			p.Resources.Workers, p.Resources.Coins = tc.workers, 20
			start, target := board.NewHex(0, 1), board.NewHex(1, 0)
			gs.Map.GetHex(start).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionHalflings, PowerValue: 1}
			gs.Map.GetHex(target).Terrain = models.TerrainForest
			a := NewPowerActionWithTransform("p1", tc.power, target, tc.build)
			a.TargetTerrain = &tc.target
			err := a.Execute(gs)
			if (err != nil) != tc.wantError {
				t.Fatalf("Execute error = %v, want error %v", err, tc.wantError)
			}
			if p.Resources.Workers != tc.wantWorkers {
				t.Fatalf("workers = %d, want %d", p.Resources.Workers, tc.wantWorkers)
			}
			if tc.wantError {
				if p.Resources.Power.Bowl3 != 6 || !gs.PowerActions.IsAvailable(tc.power) || gs.Map.GetHex(target).Terrain != models.TerrainForest {
					t.Fatal("failed action mutated state")
				}
			} else {
				if gs.Map.GetHex(target).Terrain != tc.target {
					t.Fatal("wrong destination terrain")
				}
				if gs.PendingSpades["p1"] != 0 {
					t.Fatal("intermediate terrain must not enable a second space")
				}
			}
		})
	}
}

func TestBonusSpadeUsesCorrectFactionCostsWithoutPowerCharge(t *testing.T) {
	for _, faction := range []models.FactionType{models.FactionGiants, models.FactionDarklings} {
		gs := NewGameState()
		gs.AddPlayer("p1", factions.NewFaction(faction))
		p := gs.GetPlayer("p1")
		p.Resources.Power.Bowl1, p.Resources.Power.Bowl2, p.Resources.Power.Bowl3 = 0, 0, 0
		p.Resources.Workers, p.Resources.Priests = 3, 1
		gs.BonusCards.PlayerCards["p1"] = BonusCardSpade
		gs.PowerActions.MarkUsed(PowerActionSpade1)
		start, target := board.NewHex(0, 1), board.NewHex(1, 0)
		gs.Map.GetHex(start).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: faction, PowerValue: 1}
		gs.Map.GetHex(target).Terrain = models.TerrainForest
		vp := p.VictoryPoints
		if err := NewBonusCardSpadeAction("p1", target, false, p.Faction.GetHomeTerrain()).Execute(gs); err != nil {
			t.Fatal(err)
		}
		if faction == models.FactionGiants && (p.Resources.Workers != 0 || p.Resources.Priests != 1) {
			t.Fatal("Giants must pay three workers for missing bonus spade")
		}
		if faction == models.FactionDarklings && (p.Resources.Workers != 3 || p.Resources.Priests != 0 || p.VictoryPoints != vp+2) {
			t.Fatal("Darklings must pay a priest, not workers, and receive two faction VP")
		}
		if p.Resources.Power.Bowl3 != 0 {
			t.Fatal("bonus spade charged shared-action power")
		}
	}
}

func TestPowerSpadeGiantsCannotChooseIntermediateTerrain(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewGiants())
	p := gs.GetPlayer("p1")
	p.Resources.Power.Bowl3 = 6
	start, target := board.NewHex(0, 1), board.NewHex(1, 0)
	gs.Map.GetHex(start).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionGiants, PowerValue: 1}
	gs.Map.GetHex(target).Terrain = models.TerrainForest
	a := NewPowerActionWithTransform("p1", PowerActionSpade2, target, false)
	terrain := models.TerrainLake
	a.TargetTerrain = &terrain
	if err := a.Validate(gs); err == nil {
		t.Fatal("Giants accepted nonhome destination")
	}
	terrain = models.TerrainWasteland
	if err := a.Execute(gs); err != nil {
		t.Fatal(err)
	}
	if gs.Map.GetHex(target).Terrain != terrain {
		t.Fatal("Giants did not reach home")
	}
}

func TestPowerSpadeGiantsCanBuyOnlyMissingSpade(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewGiants())
	p := gs.GetPlayer("p1")
	p.Resources.Power.Bowl3, p.Resources.Workers = 4, 3
	start, target := board.NewHex(0, 1), board.NewHex(1, 0)
	gs.Map.GetHex(start).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionGiants, PowerValue: 1}
	gs.Map.GetHex(target).Terrain = models.TerrainForest
	if err := NewPowerActionWithTransform("p1", PowerActionSpade1, target, false).Execute(gs); err != nil {
		t.Fatal(err)
	}
	if p.Resources.Workers != 0 || gs.Map.GetHex(target).Terrain != models.TerrainWasteland {
		t.Fatal("one free spade plus one paid spade should cost three workers")
	}
}

func TestPowerSpadeDarklingsRoundAndPriestBonuses(t *testing.T) {
	for _, tc := range []struct {
		power           PowerActionType
		priests, wantVP int
	}{{PowerActionSpade2, 0, 4}, {PowerActionSpade1, 1, 6}} {
		gs := NewGameState()
		gs.AddPlayer("p1", factions.NewDarklings())
		gs.Round = 1
		gs.ScoringTiles.Tiles = []ScoringTile{{Type: ScoringSpades, ActionType: ScoringActionSpades, ActionVP: 2}}
		p := gs.GetPlayer("p1")
		p.Resources.Power.Bowl3, p.Resources.Priests = 6, tc.priests
		start, target := board.NewHex(0, 1), board.NewHex(1, 0)
		gs.Map.GetHex(start).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionDarklings, PowerValue: 1}
		gs.Map.GetHex(target).Terrain = models.TerrainForest // two to Swamp
		vp := p.VictoryPoints
		if err := NewPowerActionWithTransform("p1", tc.power, target, false).Execute(gs); err != nil {
			t.Fatal(err)
		}
		if p.VictoryPoints != vp+tc.wantVP || p.Resources.Priests != 0 {
			t.Fatalf("power%d got VP%d priests%d, expected VP%d priests0", tc.power, p.VictoryPoints, p.Resources.Priests, vp+tc.wantVP)
		}
	}
}

func TestPowerSpadeAtomicSplitUsesPreActionReachability(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewHalflings())
	p := gs.GetPlayer("p1")
	p.Resources.Power.Bowl1, p.Resources.Power.Bowl2, p.Resources.Power.Bowl3 = 0, 0, 6
	p.Resources.Workers, p.Resources.Coins = 20, 20
	// Use an explicit tiny board: second is reachable only AFTER a first dwelling.
	start, first, second := board.NewHex(0, 0), board.NewHex(1, 0), board.NewHex(2, 0)
	gs.Map.Hexes = map[board.Hex]*board.MapHex{
		start: {Coord: start, Terrain: models.TerrainPlains, Building: &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionHalflings, PowerValue: 1}},
		first: {Coord: first, Terrain: models.TerrainSwamp}, second: {Coord: second, Terrain: models.TerrainSwamp},
	}
	a := NewPowerActionWithTransform("p1", PowerActionSpade2, first, true)
	terrain := models.TerrainPlains
	a.SecondTargetHex, a.SecondTargetTerrain = &second, &terrain
	if err := a.Execute(gs); err == nil {
		t.Fatal("new dwelling improperly extends same-action reachability")
	}
	if p.Resources.Power.Bowl3 != 6 || gs.Map.GetHex(first).Terrain != models.TerrainSwamp || gs.Map.GetHex(first).Building != nil {
		t.Fatal("failed split mutated state")
	}
	// Move second into original reach. Both transforms now occur atomically.
	delete(gs.Map.Hexes, second)
	second = board.NewHex(0, 1)
	gs.Map.Hexes[second] = &board.MapHex{Coord: second, Terrain: models.TerrainSwamp}
	a.SecondTargetHex, a.SecondUseSkip = &second, false
	if err := a.Execute(gs); err != nil {
		t.Fatal(err)
	}
	if gs.Map.GetHex(first).Building == nil || gs.Map.GetHex(second).Building != nil || gs.Map.GetHex(second).Terrain != models.TerrainPlains || gs.PendingSpades["p1"] != 0 {
		t.Fatal("split result incorrect")
	}
}

func TestPowerSpadeMayForfeitReward(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewGiants())
	p := gs.GetPlayer("p1")
	p.Resources.Power.Bowl3 = 4
	if err := NewPowerAction("p1", PowerActionSpade1).Execute(gs); err != nil {
		t.Fatal(err)
	}
	if p.Resources.Power.Bowl3 != 0 || gs.PowerActions.IsAvailable(PowerActionSpade1) || gs.PendingSpades["p1"] != 0 {
		t.Fatal("forfeited reward not resolved")
	}
}

func TestPowerActionExplicitDeclineReward(t *testing.T) {
	for kind := PowerActionBridge; kind <= PowerActionSpade2; kind++ {
		gs := NewGameState()
		gs.AddPlayer("p1", factions.NewHalflings())
		p := gs.GetPlayer("p1")
		p.Resources.Power.Bowl1, p.Resources.Power.Bowl2, p.Resources.Power.Bowl3 = 0, 0, 6
		p.Resources.Priests, p.BridgesBuilt = 7, 3
		coins, workers := p.Resources.Coins, p.Resources.Workers
		a := NewPowerAction("p1", kind)
		a.DeclineReward = true
		if err := a.Execute(gs); err != nil {
			t.Fatalf("kind%d: %v", kind, err)
		}
		if p.Resources.Power.Bowl3 != 6-GetPowerCost(kind) || gs.PowerActions.IsAvailable(kind) || p.Resources.Coins != coins || p.Resources.Workers != workers || p.Resources.Priests != 7 || p.BridgesBuilt != 3 {
			t.Fatalf("kind%d: declined action changed reward resources", kind)
		}
	}
}

func TestPowerActionRiverwalkersCannotDeclineForbiddenSpades(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewRiverwalkers())
	gs.GetPlayer("p1").Resources.Power.Bowl3 = 6
	for _, kind := range []PowerActionType{PowerActionSpade1, PowerActionSpade2} {
		a := NewPowerAction("p1", kind)
		a.DeclineReward = true
		if err := a.Execute(gs); err == nil {
			t.Fatal("Riverwalkers took forbidden spade action by declining reward")
		}
	}
}

func TestLegacyPendingSpadeUsesExistingSkipTracker(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("p1", factions.NewDwarves())
	gs.Phase, gs.TurnOrder = PhaseAction, []string{"p1"}
	p := gs.GetPlayer("p1")
	p.Resources.Workers = 20
	start, first, second, adjacent := board.NewHex(0, 0), board.NewHex(0, 2), board.NewHex(2, 0), board.NewHex(1, 0)
	gs.Map.Hexes = map[board.Hex]*board.MapHex{
		start: {Coord: start, Terrain: models.TerrainMountain, Building: &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionDwarves, PowerValue: 1}},
		first: {Coord: first, Terrain: models.TerrainForest}, second: {Coord: second, Terrain: models.TerrainForest}, adjacent: {Coord: adjacent, Terrain: models.TerrainForest},
	}
	gs.PendingSpades["p1"] = 2
	if err := NewTransformAndBuildAction("p1", first, false, models.TerrainMountain).Execute(gs); err != nil {
		t.Fatal(err)
	}
	if len(gs.SkipAbilityUsedThisAction["p1"]) != 1 {
		t.Fatal("skip payment tracker not retained across pending decision")
	}
	if err := NewTransformAndBuildAction("p1", second, false, models.TerrainMountain).Validate(gs); err == nil {
		t.Fatal("legacy second spade allowed a second tunnel")
	}
	if err := NewTransformAndBuildAction("p1", adjacent, false, models.TerrainMountain).Execute(gs); err != nil {
		t.Fatal(err)
	}
	if gs.PendingSpades["p1"] != 0 || p.Resources.Workers != 18 {
		t.Fatal("legacy split did not consume its spades and one tunnel cost")
	}
}

func TestPowerSpadeAtomicSplitTunnelOnce(t *testing.T) {
	for _, remoteFirst := range []bool{false, true} {
		gs := NewGameState()
		gs.AddPlayer("p1", factions.NewDwarves())
		p := gs.GetPlayer("p1")
		p.Resources.Power.Bowl1, p.Resources.Power.Bowl2, p.Resources.Power.Bowl3 = 0, 0, 6
		p.Resources.Workers, p.Resources.Coins = 20, 20
		start, adjacent, remote := board.NewHex(0, 0), board.NewHex(1, 0), board.NewHex(0, 2)
		gs.Map.Hexes = map[board.Hex]*board.MapHex{
			start:    {Coord: start, Terrain: models.TerrainMountain, Building: &models.Building{Type: models.BuildingDwelling, PlayerID: "p1", Faction: models.FactionDwarves, PowerValue: 1}},
			adjacent: {Coord: adjacent, Terrain: models.TerrainForest}, remote: {Coord: remote, Terrain: models.TerrainForest},
		}
		first, second := adjacent, remote
		if remoteFirst {
			first, second = remote, adjacent
		}
		a := NewPowerActionWithTransform("p1", PowerActionSpade2, first, false)
		home := models.TerrainMountain
		a.SecondTargetHex, a.SecondTargetTerrain = &second, &home
		vp := p.VictoryPoints
		if err := a.Execute(gs); err != nil {
			t.Fatalf("remoteFirst=%v: %v", remoteFirst, err)
		}
		if p.Resources.Workers != 18 || p.VictoryPoints != vp+4 {
			t.Fatal("tunnel must be paid and scored exactly once")
		}
	}
}

func setupOwnedPowerBridge(t testing.TB, gs *GameState, playerID string, q int) (board.Hex, board.Hex) {
	t.Helper()
	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil {
		t.Fatal("bridge fixture requires an assigned player faction")
	}
	first := board.NewHex(q, 0)
	river1 := board.NewHex(q, -1)
	river2 := board.NewHex(q+1, -1)
	second := board.NewHex(q+1, -2)
	gs.Map.Hexes[first] = &board.MapHex{Coord: first, Terrain: player.Faction.GetHomeTerrain(), Building: &models.Building{
		Type: models.BuildingDwelling, Faction: player.Faction.GetType(), PlayerID: playerID, PowerValue: 1,
	}}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[second] = &board.MapHex{Coord: second, Terrain: player.Faction.GetHomeTerrain()}
	delete(gs.Map.RiverHexes, first)
	delete(gs.Map.RiverHexes, second)
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	return first, second
}

func TestPowerAction_Bridge(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 5
	initialBowl1 := player.Resources.Power.Bowl1

	first, second := setupOwnedPowerBridge(t, gs, "player1", 10)
	action := NewPowerActionWithBridge("player1", first, second)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected bridge action to succeed, got error: %v", err)
	}

	// Verify power was moved from Bowl3 to Bowl1
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (5-3), got %d", player.Resources.Power.Bowl3)
	}
	if player.Resources.Power.Bowl1 != initialBowl1+3 {
		t.Errorf("expected Bowl1 to have %d power, got %d", initialBowl1+3, player.Resources.Power.Bowl1)
	}

	// Verify bridge count increased
	if player.BridgesBuilt != 1 {
		t.Errorf("expected 1 bridge built, got %d", player.BridgesBuilt)
	}

	// Verify action is marked as used
	if gs.PowerActions.IsAvailable(PowerActionBridge) {
		t.Error("expected bridge action to be marked as used")
	}
}

func TestPowerAction_BridgeCoordinatesRequireOwnedEndpoint(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	if err := gs.AddPlayer("player1", faction); err != nil {
		t.Fatal(err)
	}
	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 10
	if err := NewPowerAction("player1", PowerActionBridge).Validate(gs); err == nil {
		t.Fatal("coordinate-free power bridge was accepted")
	}
	partial := NewPowerAction("player1", PowerActionBridge)
	partialEndpoint := board.NewHex(0, 0)
	partial.BridgeHex1 = &partialEndpoint
	if err := partial.Validate(gs); err == nil {
		t.Fatal("partially specified power bridge was accepted")
	}

	var first, second board.Hex
	found := false
	for h1 := range gs.Map.Hexes {
		for h2 := range gs.Map.Hexes {
			if h1 == h2 || gs.Map.ValidateBridgePlacement(h1, h2) != nil {
				continue
			}
			first, second, found = h1, h2, true
			break
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("base map has no valid bridge placement")
	}

	remote := NewPowerActionWithBridge("player1", first, second)
	if err := remote.Validate(gs); err == nil {
		t.Fatal("power bridge without an owned endpoint was accepted")
	}
	gs.Map.GetHex(first).Building = &models.Building{
		Type: models.BuildingDwelling, Faction: faction.GetType(),
		PlayerID: "player1", PowerValue: 1,
	}
	owned := NewPowerActionWithBridge("player1", first, second)
	if err := owned.Validate(gs); err != nil {
		t.Fatalf("power bridge touching an owned structure was rejected: %v", err)
	}
	if err := owned.Execute(gs); err != nil {
		t.Fatalf("execute owned power bridge: %v", err)
	}
	if !gs.Map.HasBridge(first, second) {
		t.Fatal("owned power bridge was not placed")
	}
}

func TestPowerAction_BridgeLimit(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 20
	player.BridgesBuilt = 3 // Already at limit

	action := NewPowerAction("player1", PowerActionBridge)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when building 4th bridge")
	}
}

func TestPowerAction_Priest(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 5
	initialPriests := player.Resources.Priests

	action := NewPowerAction("player1", PowerActionPriest)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected priest action to succeed, got error: %v", err)
	}

	// Verify power was spent
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (5-3), got %d", player.Resources.Power.Bowl3)
	}

	// Verify priest was gained
	if player.Resources.Priests != initialPriests+1 {
		t.Errorf("expected %d priests, got %d", initialPriests+1, player.Resources.Priests)
	}

	// Verify action is marked as used
	if gs.PowerActions.IsAvailable(PowerActionPriest) {
		t.Error("expected priest action to be marked as used")
	}
}

func TestPowerAction_Workers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	initialWorkers := player.Resources.Workers

	action := NewPowerAction("player1", PowerActionWorkers)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected workers action to succeed, got error: %v", err)
	}

	// Verify power was spent
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (6-4), got %d", player.Resources.Power.Bowl3)
	}

	// Verify 2 workers were gained
	if player.Resources.Workers != initialWorkers+2 {
		t.Errorf("expected %d workers, got %d", initialWorkers+2, player.Resources.Workers)
	}
}

func TestPowerAction_Coins(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	initialCoins := player.Resources.Coins

	action := NewPowerAction("player1", PowerActionCoins)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected coins action to succeed, got error: %v", err)
	}

	// Verify power was spent
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (6-4), got %d", player.Resources.Power.Bowl3)
	}

	// Verify 7 coins were gained
	if player.Resources.Coins != initialCoins+7 {
		t.Errorf("expected %d coins, got %d", initialCoins+7, player.Resources.Coins)
	}
}

func TestPowerAction_Spade1WithTransform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Place player1's initial dwelling at (0, 1)
	initialHex := board.NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Forest (1 spade away from Plains)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	// Use 1 spade power action to transform and build
	action := NewPowerActionWithTransform("player1", PowerActionSpade1, targetHex, true)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade1 action to succeed, got error: %v", err)
	}

	// Verify power was spent (4 power for 1 spade action)
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (6-4), got %d", player.Resources.Power.Bowl3)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}

	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}

	// Verify action is marked as used
	if gs.PowerActions.IsAvailable(PowerActionSpade1) {
		t.Error("expected spade1 action to be marked as used")
	}
}

func TestPowerAction_Spade2WithTransform(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 8
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Place player1's initial dwelling at (0, 1)
	initialHex := board.NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Lake (2 spades away from Plains: Plains -> Swamp -> Lake)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainLake)

	// Use 2 spade power action to transform and build
	action := NewPowerActionWithTransform("player1", PowerActionSpade2, targetHex, true)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade2 action to succeed, got error: %v", err)
	}

	// Verify power was spent (6 power for 2 spade action)
	if player.Resources.Power.Bowl3 != 2 {
		t.Errorf("expected Bowl3 to have 2 power (8-6), got %d", player.Resources.Power.Bowl3)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}

	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestPowerAction_Spade1WithAdditionalWorkers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 6
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Place player1's initial dwelling at (0, 1)
	initialHex := board.NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Lake (2 spades away from Plains: Plains -> Swamp -> Lake)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainLake)

	initialWorkers := player.Resources.Workers

	// Use 1 spade power action - need 1 more spade from workers
	action := NewPowerActionWithTransform("player1", PowerActionSpade1, targetHex, true)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade1 action with workers to succeed, got error: %v", err)
	}

	// Verify 1 spade was paid with workers (for the 2nd spade needed)
	// Halflings have 3 workers per spade at base level
	// Also, building the dwelling costs 1 worker
	dwellingCost := faction.GetDwellingCost()
	workersPerSpade := faction.GetTerraformCost(1)
	expectedWorkers := initialWorkers - workersPerSpade - dwellingCost.Workers
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers remaining, got %d", expectedWorkers, player.Resources.Workers)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}
}

func TestPowerAction_Spade2WithAdditionalWorkers(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 8
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Place player1's initial dwelling at (0, 1)
	initialHex := board.NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Forest (3 spades away from Plains: Plains -> Swamp -> Lake -> Forest)
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	initialWorkers := player.Resources.Workers

	// Use 2 spade power action - need 1 more spade from workers (3 total needed, 2 free)
	action := NewPowerActionWithTransform("player1", PowerActionSpade2, targetHex, true)

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade2 action with workers to succeed, got error: %v", err)
	}

	// Verify 1 spade was paid with workers (for the 3rd spade needed)
	// Halflings have 3 workers per spade at base level
	// Also, building the dwelling costs 1 worker
	dwellingCost := faction.GetDwellingCost()
	workersPerSpade := faction.GetTerraformCost(1)
	expectedWorkers := initialWorkers - workersPerSpade - dwellingCost.Workers
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers remaining, got %d", expectedWorkers, player.Resources.Workers)
	}

	// Verify terrain was transformed
	mapHex := gs.Map.GetHex(targetHex)
	if mapHex.Terrain != models.TerrainPlains {
		t.Errorf("expected terrain to be Plains, got %v", mapHex.Terrain)
	}

	// Verify dwelling was built
	if mapHex.Building == nil {
		t.Fatal("expected dwelling to be built")
	}
	if mapHex.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling, got %v", mapHex.Building.Type)
	}
}

func TestPowerAction_Spade2TwoHexes(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings() // Plains
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 8
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	// Place player1's initial dwelling at (1, 1)
	initialHex := board.NewHex(1, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// First target hex at (1, 0) - adjacent to initial dwelling
	// Set it to Swamp (1 spade away from Plains)
	targetHex1 := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex1, models.TerrainSwamp)

	// Second target hex at (2, 1) - adjacent to initial dwelling
	// Set it to Swamp (1 spade away from Plains)
	targetHex2 := board.NewHex(2, 1)
	gs.Map.TransformTerrain(targetHex2, models.TerrainSwamp)

	// Use 2 spade power action - transform first hex and build dwelling
	action := NewPowerActionWithTransform("player1", PowerActionSpade2, targetHex1, true)
	action.SecondTargetHex = &targetHex2
	secondTerrain := models.TerrainPlains
	action.SecondTargetTerrain = &secondTerrain

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("expected spade2 action to succeed, got error: %v", err)
	}

	// Verify first hex was transformed and has dwelling
	mapHex1 := gs.Map.GetHex(targetHex1)
	if mapHex1.Terrain != models.TerrainPlains {
		t.Errorf("expected first hex terrain to be Plains, got %v", mapHex1.Terrain)
	}
	if mapHex1.Building == nil {
		t.Fatal("expected dwelling to be built on first hex")
	}
	if mapHex1.Building.Type != models.BuildingDwelling {
		t.Errorf("expected dwelling on first hex, got %v", mapHex1.Building.Type)
	}

	mapHex2 := gs.Map.GetHex(targetHex2)
	if mapHex2.Terrain != models.TerrainPlains {
		t.Errorf("expected second hex to be transformed to Plains, got %v", mapHex2.Terrain)
	}
	if mapHex2.Building != nil {
		t.Error("expected no building on second hex follow-up")
	}
	if _, ok := gs.PendingSpades["player1"]; ok {
		t.Fatalf("expected pending spade to be cleared after follow-up transform")
	}
	if _, ok := gs.PendingSpadeBuildAllowed["player1"]; ok {
		t.Fatalf("expected pending spade build policy to be cleared after follow-up transform")
	}
}

func TestPowerAction_OncePerRound(t *testing.T) {
	gs := NewGameState()
	faction1 := factions.NewHalflings()  // Plains
	faction2 := factions.NewSwarmlings() // Lake - different from Halflings
	gs.AddPlayer("player1", faction1)
	gs.AddPlayer("player2", faction2)

	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")

	player1.Resources.Power.Bowl3 = 10
	player2.Resources.Power.Bowl3 = 10

	// Player1 takes bridge action
	first, second := setupOwnedPowerBridge(t, gs, "player1", 10)
	action1 := NewPowerActionWithBridge("player1", first, second)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected player1 bridge action to succeed, got error: %v", err)
	}

	// Player2 tries to take same action - should fail
	action2 := NewPowerAction("player2", PowerActionBridge)
	err = action2.Execute(gs)
	if err == nil {
		t.Fatal("expected error when player2 tries to take already-used bridge action")
	}

	// Player2 can take a different action
	action3 := NewPowerAction("player2", PowerActionPriest)
	err = action3.Execute(gs)
	if err != nil {
		t.Fatalf("expected player2 priest action to succeed, got error: %v", err)
	}
}

func TestPowerAction_ResetBetweenRounds(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 20

	// Take bridge action in round 1
	first1, second1 := setupOwnedPowerBridge(t, gs, "player1", 10)
	action1 := NewPowerActionWithBridge("player1", first1, second1)
	err := action1.Execute(gs)
	if err != nil {
		t.Fatalf("expected bridge action to succeed, got error: %v", err)
	}

	// Verify action is used
	if gs.PowerActions.IsAvailable(PowerActionBridge) {
		t.Error("expected bridge action to be marked as used")
	}

	// Start new round
	gs.StartNewRound()

	// Verify action is available again
	if !gs.PowerActions.IsAvailable(PowerActionBridge) {
		t.Error("expected bridge action to be available after new round")
	}

	// Can take the action again at a second legal placement.
	first2, second2 := setupOwnedPowerBridge(t, gs, "player1", 20)
	action2 := NewPowerActionWithBridge("player1", first2, second2)
	err = action2.Execute(gs)
	if err != nil {
		t.Fatalf("expected bridge action to succeed in round 2, got error: %v", err)
	}

	// Should have 2 bridges now
	if player.BridgesBuilt != 2 {
		t.Errorf("expected 2 bridges built, got %d", player.BridgesBuilt)
	}
}

func TestPowerAction_InsufficientPower(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")
	player.Resources.Power.Bowl3 = 2 // Not enough for any action
	player.Resources.Power.Bowl2 = 0

	action := NewPowerAction("player1", PowerActionBridge)

	err := action.Execute(gs)
	if err == nil {
		t.Fatal("expected error when player has insufficient power")
	}
}

func TestPowerAction_AutoBurnsMissingPowerBeforeClaim(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewHalflings()
	if err := gs.AddPlayer("player1", faction); err != nil {
		t.Fatalf("add player: %v", err)
	}

	player := gs.GetPlayer("player1")
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		t.Fatal("missing player resources")
	}
	player.Resources.Power.Bowl1 = 0
	player.Resources.Power.Bowl2 = 11
	player.Resources.Power.Bowl3 = 1
	player.Resources.Coins = 20
	player.Resources.Workers = 20
	player.Resources.Priests = 5

	initialHex := board.NewHex(0, 1)
	gs.Map.GetHex(initialHex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	}
	targetHex := board.NewHex(1, 0)
	gs.Map.TransformTerrain(targetHex, models.TerrainForest)

	if err := NewPowerActionWithTransform("player1", PowerActionSpade2, targetHex, true).Execute(gs); err != nil {
		t.Fatalf("expected auto-burned ACT6 to succeed, got error: %v", err)
	}

	if player.Resources.Power.Bowl1 != 6 {
		t.Fatalf("bowl1 = %d, want 6 after ACT6", player.Resources.Power.Bowl1)
	}
	if player.Resources.Power.Bowl2 != 1 {
		t.Fatalf("bowl2 = %d, want 1 after auto-burn", player.Resources.Power.Bowl2)
	}
	if player.Resources.Power.Bowl3 != 0 {
		t.Fatalf("bowl3 = %d, want 0 after auto-burn and spend", player.Resources.Power.Bowl3)
	}
	if gs.Map.GetHex(targetHex).Building == nil {
		t.Fatal("expected ACT6 target hex to have a dwelling after execution")
	}
}

// ============================================================================
// BRIDGE GEOMETRY TESTS
// ============================================================================

func TestBridge_ValidGeometry_BaseOrientation(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Test base orientation: delta (1,-2) with midpoints (0,-1) and (1,-1)
	// This is the canonical valid bridge pattern
	hex1 := board.NewHex(0, 0)
	river1 := board.NewHex(0, -1)
	river2 := board.NewHex(1, -1)
	hex2 := board.NewHex(1, -2)

	// Set up map
	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain(), Building: &models.Building{
		Type: models.BuildingDwelling, Faction: faction.GetType(), PlayerID: "player1", PowerValue: 1,
	}}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain(), Building: &models.Building{
		Type: models.BuildingDwelling, Faction: faction.GetType(), PlayerID: "player1", PowerValue: 1,
	}}
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true

	// Build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, hex2)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("valid bridge should succeed: %v", err)
	}

	// Verify bridge exists
	if !gs.Map.HasBridge(hex1, hex2) {
		t.Error("bridge should exist on map")
	}

	// Verify hexes are now considered adjacent
	if !gs.Map.IsDirectlyAdjacent(hex1, hex2) {
		t.Error("hexes should be adjacent via bridge")
	}
}

func TestBridge_ValidGeometry_BidirectionalBridge(t *testing.T) {
	// Bridges are bidirectional - can be built in either direction
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	hex1 := board.NewHex(0, 0)
	river1 := board.NewHex(0, -1)
	river2 := board.NewHex(1, -1)
	hex2 := board.NewHex(1, -2)

	// Set up map
	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain(), Building: &models.Building{
		Type: models.BuildingDwelling, Faction: faction.GetType(), PlayerID: "player1", PowerValue: 1,
	}}
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true

	// Build bridge in reverse direction (hex2 to hex1)
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex2, hex1)
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("bridge should work in both directions: %v", err)
	}

	// Verify bridge exists (should work both ways)
	if !gs.Map.HasBridge(hex1, hex2) {
		t.Error("bridge should exist on map")
	}
	if !gs.Map.HasBridge(hex2, hex1) {
		t.Error("bridge should work in reverse direction too")
	}

	// Verify hexes are adjacent
	if !gs.Map.IsDirectlyAdjacent(hex1, hex2) {
		t.Error("hexes should be adjacent via bridge")
	}
	if !gs.Map.IsDirectlyAdjacent(hex2, hex1) {
		t.Error("adjacency should be bidirectional")
	}
}

func TestBridge_InvalidGeometry_NonRiverMidpoint(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Try to build bridge where one midpoint is NOT a river
	hex1 := board.NewHex(0, 0)
	river1 := board.NewHex(0, -1)
	notRiver := board.NewHex(1, -1) // This should be river but isn't
	hex2 := board.NewHex(1, -2)

	// Set up map
	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[notRiver] = &board.MapHex{Coord: notRiver, Terrain: models.TerrainPlains} // NOT river!
	gs.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain()}
	gs.Map.RiverHexes[river1] = true
	// notRiver is NOT marked as river

	// Try to build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, hex2)
	err := action.Execute(gs)
	if err == nil {
		t.Error("bridge with non-river midpoint should fail")
	}

	// Verify bridge was not created
	if gs.Map.HasBridge(hex1, hex2) {
		t.Error("invalid bridge should not exist on map")
	}
}

func TestBridge_InvalidGeometry_WrongDistance(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Try to build bridge with wrong distance (adjacent hexes = distance 1, not distance 2)
	hex1 := board.NewHex(0, 0)
	hex2 := board.NewHex(1, 0) // Adjacent, but bridges must span distance 2

	// Set up map
	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: faction.GetHomeTerrain()}

	// Try to build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, hex2)
	err := action.Execute(gs)
	if err == nil {
		t.Error("bridge between adjacent hexes should fail")
	}

	// Verify bridge was not created
	if gs.Map.HasBridge(hex1, hex2) {
		t.Error("invalid bridge should not exist on map")
	}
}

func TestBridge_InvalidGeometry_RiverEndpoint(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Try to build bridge where one endpoint is a river (not allowed)
	hex1 := board.NewHex(0, 0)
	riverEndpoint := board.NewHex(1, -2)
	river1 := board.NewHex(0, -1)
	river2 := board.NewHex(1, -1)

	// Set up map
	gs.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: faction.GetHomeTerrain()}
	gs.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
	gs.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
	gs.Map.Hexes[riverEndpoint] = &board.MapHex{Coord: riverEndpoint, Terrain: models.TerrainRiver} // Endpoint is river!
	gs.Map.RiverHexes[river1] = true
	gs.Map.RiverHexes[river2] = true
	gs.Map.RiverHexes[riverEndpoint] = true

	// Try to build bridge
	player.Resources.Power.Bowl3 = 3
	action := NewPowerActionWithBridge("player1", hex1, riverEndpoint)
	err := action.Execute(gs)
	if err == nil {
		t.Error("bridge with river endpoint should fail")
	}

	// Verify bridge was not created
	if gs.Map.HasBridge(hex1, riverEndpoint) {
		t.Error("invalid bridge should not exist on map")
	}
}

func Test7PriestLimit_PowerAction(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Give player enough power and set up at the priest limit
	player.Resources.Power.Bowl3 = 10
	player.Resources.Priests = 1

	// Place 3 priests on cult track action spaces
	gs.CultTracks.InitializePlayer("player1")
	gs.CultTracks.PriestsOnActionSpaces["player1"][CultFire] = 2
	gs.CultTracks.PriestsOnActionSpaces["player1"][CultWater] = 1

	// Now player has 1 in hand + 3 on action spaces = 4 total
	// Power action should work (can gain up to 3 more)
	action := &PowerAction{
		BaseAction: BaseAction{
			Type:     ActionPowerAction,
			PlayerID: "player1",
		},
		ActionType: PowerActionPriest,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("power action should work when under 7-priest limit, got error: %v", err)
	}

	// Verify priest was gained
	if player.Resources.Priests != 2 {
		t.Errorf("expected 2 priests in hand after power action, got %d", player.Resources.Priests)
	}

	// Now test at the limit (7 total) - need fresh game state since power actions are one-time per round
	gs2 := NewGameState()
	faction2 := factions.NewAuren()
	gs2.AddPlayer("player1", faction2)
	player2 := gs2.GetPlayer("player1")

	// Set up at limit
	player2.Resources.Power.Bowl3 = 10
	player2.Resources.Priests = 4
	gs2.CultTracks.InitializePlayer("player1")
	gs2.CultTracks.PriestsOnActionSpaces["player1"][CultFire] = 2
	gs2.CultTracks.PriestsOnActionSpaces["player1"][CultWater] = 1
	// 4 in hand + 3 on action spaces = 7 total

	action2 := &PowerAction{
		BaseAction: BaseAction{
			Type:     ActionPowerAction,
			PlayerID: "player1",
		},
		ActionType: PowerActionPriest,
	}

	// Action should fail at the priest limit
	err = action2.Execute(gs2)
	if err == nil {
		t.Fatalf("power action should fail at the 7-priest limit")
	}

	// Verify no priest was gained and power was not spent
	if player2.Resources.Priests != 4 {
		t.Errorf("expected 4 priests in hand (no change at limit), got %d", player2.Resources.Priests)
	}
	if player2.Resources.Power.Bowl3 != 10 {
		t.Errorf("expected power to remain unspent, got %d", player2.Resources.Power.Bowl3)
	}
}

// Regression test for Bug #7: ACT6 split transform/build
// The bug was that power actions like "ACT6. transform F2 to gray. build D4"
// (where transform and build are on different hexes) were not working correctly.
// The transform hex was being transformed, but the build hex was charged full
// terraform cost instead of using the remaining free spades from ACT6.
func TestPowerActionSpade2_SplitTransformAndBuild(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewEngineers() // Engineers home terrain is Mountain (gray)
	gs.AddPlayer("player1", faction)

	player := gs.GetPlayer("player1")

	// Set up power for ACT6 (2 free spades, costs 6 power)
	player.Resources.Power.Bowl3 = 6

	// Set up two hexes:
	// 1. Transform hex (F2): Forest -> needs to be transformed to Mountain
	// 2. Build hex (D4): Wasteland -> needs to be transformed to Mountain for dwelling
	transformHex := board.NewHex(-1, 5) // F2 in log notation
	buildHex := board.NewHex(4, 3)      // D4 in log notation

	transformMapHex := gs.Map.GetHex(transformHex)
	transformMapHex.Terrain = models.TerrainForest // Will transform to Mountain

	buildMapHex := gs.Map.GetHex(buildHex)
	buildMapHex.Terrain = models.TerrainWasteland // Will transform to Mountain and build

	// Place adjacent dwelling for build hex adjacency
	adjacentHex := board.NewHex(3, 3)
	adjacentMapHex := gs.Map.GetHex(adjacentHex)
	adjacentMapHex.Terrain = models.TerrainMountain
	adjacentMapHex.Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionEngineers,
		PlayerID:   "player1",
		PowerValue: 1,
	}

	// Engineers start with 2 workers
	initialWorkers := player.Resources.Workers

	// Manually transform both hexes (simulating the action converter's fix)
	// This is what the replay validator does for split transform/build
	err := gs.Map.TransformTerrain(transformHex, models.TerrainMountain)
	if err != nil {
		t.Fatalf("failed to transform F2: %v", err)
	}

	err = gs.Map.TransformTerrain(buildHex, models.TerrainMountain)
	if err != nil {
		t.Fatalf("failed to transform D4: %v", err)
	}

	// Mark power action as used
	gs.PowerActions.MarkUsed(PowerActionSpade2)

	// Now build dwelling on D4 (should only cost dwelling resources)
	buildAction := NewTransformAndBuildAction("player1", buildHex, true, models.TerrainTypeUnknown)

	err = buildAction.Execute(gs)
	if err != nil {
		t.Fatalf("expected build to succeed, got error: %v", err)
	}

	// Verify both hexes were transformed
	if transformMapHex.Terrain != models.TerrainMountain {
		t.Errorf("F2 should be Mountain, got %v", transformMapHex.Terrain)
	}

	if buildMapHex.Terrain != models.TerrainMountain {
		t.Errorf("D4 should be Mountain, got %v", buildMapHex.Terrain)
	}

	// Verify dwelling was built on D4
	if buildMapHex.Building == nil || buildMapHex.Building.Type != models.BuildingDwelling {
		t.Error("expected dwelling to be built on D4")
	}

	// Verify workers consumed = only dwelling cost (Engineers dwelling costs 1 worker)
	// NOT transformation cost since ACT6 provided 2 free spades
	expectedWorkers := initialWorkers - 1
	if player.Resources.Workers != expectedWorkers {
		t.Errorf("expected %d workers (spent 1 for dwelling), got %d workers",
			expectedWorkers, player.Resources.Workers)
	}

	// Verify F2 has no building (it was just transformed, not built on)
	if transformMapHex.Building != nil {
		t.Error("F2 should not have a building")
	}
}
