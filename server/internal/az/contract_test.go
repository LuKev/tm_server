package az

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestOracleCorpusCoversBaseFactionsRoundsAndPendingDecisions(t *testing.T) {
	pairs := [][2]models.FactionType{
		{models.FactionNomads, models.FactionWitches},
		{models.FactionFakirs, models.FactionAuren},
		{models.FactionChaosMagicians, models.FactionEngineers},
		{models.FactionGiants, models.FactionDwarves},
		{models.FactionSwarmlings, models.FactionWitches},
		{models.FactionMermaids, models.FactionAuren},
		{models.FactionWitches, models.FactionEngineers},
		{models.FactionAuren, models.FactionNomads},
		{models.FactionHalflings, models.FactionWitches},
		{models.FactionCultists, models.FactionEngineers},
		{models.FactionAlchemists, models.FactionWitches},
		{models.FactionDarklings, models.FactionNomads},
		{models.FactionEngineers, models.FactionWitches},
		{models.FactionDwarves, models.FactionAuren},
	}
	for i, pair := range pairs {
		position := forcedActionPosition(t, int64(100+i), pair[0], pair[1])
		assertActionSetsEqual(t, position.LegalActions(), oracleLegalActions(position))
		assertLegalActionRoundTrips(t, position)
	}

	for round := 1; round <= 6; round++ {
		position := forcedActionPosition(t, int64(200+round), models.FactionWitches, models.FactionEngineers)
		position.state.Round = round
		assertActionSetsEqual(t, position.LegalActions(), oracleLegalActions(position))
		assertLegalActionRoundTrips(t, position)
	}

	selection := factionSelectionPosition(t)
	assertActionSetsEqual(t, selection.LegalActions(), oracleLegalActions(selection))
	assertLegalActionRoundTrips(t, selection)

	base := forcedActionPosition(t, 301, models.FactionWitches, models.FactionEngineers)
	states := map[string]*game.GameState{}

	favor := base.StateClone()
	favor.PendingFavorTileSelection = &game.PendingFavorTileSelection{PlayerID: "p0", Count: 1}
	states["favor"] = favor

	darklings := forcedActionPosition(t, 302, models.FactionDarklings, models.FactionNomads).StateClone()
	darklings.PendingDarklingsPriestOrdination = &game.PendingDarklingsPriestOrdination{PlayerID: "p0"}
	states["darklings"] = darklings

	cultists := forcedActionPosition(t, 303, models.FactionCultists, models.FactionEngineers).StateClone()
	cultists.PendingCultistsCultSelection = &game.PendingCultistsCultSelection{PlayerID: "p0"}
	states["cultists"] = cultists

	halflings := forcedActionPosition(t, 304, models.FactionHalflings, models.FactionWitches).StateClone()
	halflings.PendingHalflingsSpades = &game.PendingHalflingsSpades{PlayerID: "p0", SpadesRemaining: 1}
	states["halflings"] = halflings

	spade := base.StateClone()
	spade.PendingSpades["p0"] = 2
	spade.PendingSpadeBuildAllowed["p0"] = true
	states["action_spade"] = spade

	cultSpade := base.StateClone()
	cultSpade.PendingCultRewardSpades = map[string]int{"p0": 1}
	states["cult_spade"] = cultSpade

	town := base.StateClone()
	anchor := ownedBuildingHex(t, town, "p0")
	town.PendingTownFormations["p0"] = []*game.PendingTownFormation{{PlayerID: "p0", Hexes: []board.Hex{anchor}}}
	states["town"] = town

	townCult := base.StateClone()
	townCult.PendingTownCultTopChoice = &game.PendingTownCultTopChoice{
		PlayerID: "p0", AdvanceAmount: 2,
		CandidateTracks: []game.CultTrack{game.CultFire, game.CultWater, game.CultAir},
		MaxSelections:   2,
	}
	states["town_cult"] = townCult

	townCultZero := base.StateClone()
	townCultZero.PendingTownCultTopChoice = &game.PendingTownCultTopChoice{
		PlayerID: "p0", AdvanceAmount: 2,
		CandidateTracks: []game.CultTrack{game.CultFire}, MaxSelections: 0,
	}
	states["town_cult_zero_keys"] = townCultZero

	leech := base.StateClone()
	leech.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{Amount: 3, VPCost: 2, FromPlayerID: "p0", EventID: 1}}
	states["leech"] = leech

	for name, state := range states {
		t.Run(name, func(t *testing.T) {
			position, err := NewPosition(state)
			if err != nil {
				t.Fatal(err)
			}
			assertActionSetsEqual(t, position.LegalActions(), oracleLegalActions(position))
			assertLegalActionRoundTrips(t, position)
			if len(position.LegalActions()) == 0 {
				t.Fatal("pending state has no legal resolution")
			}
		})
	}
}

func TestPrunedCandidatesMatchOracleAtResourceAndAbilityBoundaries(t *testing.T) {
	type boundaryCase struct {
		faction models.FactionType
		mutate  func(*game.GameState)
		check   func([]SearchAction) bool
		want    bool
	}
	powerBridge := func(actions []SearchAction) bool {
		for _, action := range actions {
			if action.Kind == game.ActionPowerAction && action.Power == game.PowerActionBridge {
				return true
			}
		}
		return false
	}
	witchesRide := func(actions []SearchAction) bool {
		for _, action := range actions {
			if action.Kind == game.ActionSpecialAction && action.Special == game.SpecialActionWitchesRide {
				return true
			}
		}
		return false
	}
	skipAction := func(actions []SearchAction) bool {
		for _, action := range actions {
			if action.UseSkip {
				return true
			}
		}
		return false
	}
	bonusSpadeSkip := func(actions []SearchAction) bool {
		for _, action := range actions {
			if action.Kind == game.ActionSpecialAction && action.Special == game.SpecialActionBonusCardSpade && action.UseSkip {
				return true
			}
		}
		return false
	}
	configureBridge := func(state *game.GameState) {
		first := board.NewHex(0, 0)
		river1 := board.NewHex(0, -1)
		river2 := board.NewHex(1, -1)
		second := board.NewHex(1, -2)
		player := state.GetPlayer("p0")
		state.Map.Hexes[first] = &board.MapHex{Coord: first, Terrain: player.Faction.GetHomeTerrain(), Building: &models.Building{
			Type: models.BuildingDwelling, Faction: player.Faction.GetType(), PlayerID: "p0", PowerValue: 1,
		}}
		state.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
		state.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
		state.Map.Hexes[second] = &board.MapHex{Coord: second, Terrain: models.TerrainSwamp}
		state.Map.RiverHexes[river1] = true
		state.Map.RiverHexes[river2] = true
	}
	cases := map[string]boundaryCase{
		"power_exact_bowl3": {
			faction: models.FactionWitches,
			mutate: func(state *game.GameState) {
				configureBridge(state)
				power := state.GetPlayer("p0").Resources.Power
				power.Bowl2, power.Bowl3 = 0, 3
			},
			check: powerBridge, want: true,
		},
		"power_exact_auto_burn": {
			faction: models.FactionWitches,
			mutate: func(state *game.GameState) {
				configureBridge(state)
				power := state.GetPlayer("p0").Resources.Power
				power.Bowl2, power.Bowl3 = 4, 1
			},
			check: powerBridge, want: true,
		},
		"power_one_short_of_auto_burn": {
			faction: models.FactionWitches,
			mutate: func(state *game.GameState) {
				configureBridge(state)
				power := state.GetPlayer("p0").Resources.Power
				power.Bowl2, power.Bowl3 = 3, 1
			},
			check: powerBridge, want: false,
		},
		"power_used": {
			faction: models.FactionWitches,
			mutate: func(state *game.GameState) {
				configureBridge(state)
				state.PowerActions.UsedActions[game.PowerActionBridge] = true
			},
			check: powerBridge, want: false,
		},
		"bridge_limit": {
			faction: models.FactionWitches,
			mutate: func(state *game.GameState) {
				configureBridge(state)
				state.GetPlayer("p0").BridgesBuilt = 3
			},
			check: powerBridge, want: false,
		},
		"stronghold_special_available": {
			faction: models.FactionWitches,
			mutate: func(state *game.GameState) {
				state.GetPlayer("p0").HasStrongholdAbility = true
			},
			check: witchesRide, want: true,
		},
		"stronghold_special_used": {
			faction: models.FactionWitches,
			mutate: func(state *game.GameState) {
				player := state.GetPlayer("p0")
				player.HasStrongholdAbility = true
				player.SpecialActionsUsed[game.SpecialActionWitchesRide] = true
			},
			check: witchesRide, want: false,
		},
		"fakirs_skip_affordable": {
			faction: models.FactionFakirs,
			mutate: func(state *game.GameState) {
				state.GetPlayer("p0").Resources.Priests = 1
			},
			check: skipAction, want: true,
		},
		"fakirs_skip_unaffordable": {
			faction: models.FactionFakirs,
			mutate: func(state *game.GameState) {
				state.GetPlayer("p0").Resources.Priests = 0
			},
			check: skipAction, want: false,
		},
		"dwarves_tunnel_exact_workers": {
			faction: models.FactionDwarves,
			mutate: func(state *game.GameState) {
				state.GetPlayer("p0").Resources.Workers = 2
			},
			check: skipAction, want: true,
		},
		"fakirs_bonus_spade_skip_affordable": {
			faction: models.FactionFakirs,
			mutate: func(state *game.GameState) {
				state.GetPlayer("p0").Resources.Priests = 1
				state.BonusCards.PlayerCards["p0"] = game.BonusCardSpade
				delete(state.BonusCards.Available, game.BonusCardSpade)
			},
			check: bonusSpadeSkip, want: true,
		},
		"fakirs_bonus_spade_skip_unaffordable": {
			faction: models.FactionFakirs,
			mutate: func(state *game.GameState) {
				state.GetPlayer("p0").Resources.Priests = 0
				state.BonusCards.PlayerCards["p0"] = game.BonusCardSpade
				delete(state.BonusCards.Available, game.BonusCardSpade)
			},
			check: bonusSpadeSkip, want: false,
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			opponent := models.FactionEngineers
			if testCase.faction == models.FactionEngineers || testCase.faction == models.FactionDwarves {
				opponent = models.FactionWitches
			}
			state := forcedActionPosition(t, 1200+int64(len(name)), testCase.faction, opponent).StateClone()
			testCase.mutate(state)
			position, err := NewPosition(state)
			if err != nil {
				t.Fatal(err)
			}
			actions := position.LegalActions()
			assertActionSetsEqual(t, actions, oracleLegalActions(position))
			if got := testCase.check(actions); got != testCase.want {
				t.Fatalf("boundary action presence = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestTownSelectionCanonicalizesAnchorAndPrioritizesMandatoryFormation(t *testing.T) {
	state := forcedActionPosition(t, 320, models.FactionWitches, models.FactionEngineers).StateClone()
	delayedAnchor := ownedBuildingHex(t, state, "p0")
	mandatoryAnchor := placeOwnedBuilding(t, state, "p0")
	state.PendingTownFormations["p0"] = []*game.PendingTownFormation{
		{PlayerID: "p0", Hexes: []board.Hex{delayedAnchor}, CanBeDelayed: true},
		{PlayerID: "p0", Hexes: []board.Hex{mandatoryAnchor}, CanBeDelayed: false},
	}
	position, err := NewPosition(state)
	if err != nil {
		t.Fatal(err)
	}
	actions := position.LegalActions()
	if len(actions) == 0 {
		t.Fatal("mandatory town formation has no tile choices")
	}
	for _, action := range actions {
		if action.Kind != game.ActionSelectTownTile || len(action.Hexes) != 1 || action.Hexes[0] != mandatoryAnchor {
			t.Fatalf("non-canonical town choice %s", action.Key())
		}
	}
	if err := position.Apply(actions[0]); err != nil {
		t.Fatal(err)
	}
	remaining := position.state.PendingTownFormations["p0"]
	if len(remaining) != 1 || !remaining[0].CanBeDelayed {
		t.Fatal("town selection did not consume mandatory formation first")
	}
}

func TestPositionSequencesLeechAndDelayedMermaidTownToOneOwner(t *testing.T) {
	t.Run("leech_before_actor_conversions", func(t *testing.T) {
		state := forcedActionPosition(t, 340, models.FactionWitches, models.FactionEngineers).StateClone()
		state.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{Amount: 2, VPCost: 1, FromPlayerID: "p0", EventID: 1}}
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		if position.DecisionPlayer() != "p1" {
			t.Fatalf("leech responder is not the sole decision owner: %q", position.DecisionPlayer())
		}
		for _, action := range position.LegalActions() {
			if action.Kind != game.ActionAcceptPowerLeech && action.Kind != game.ActionDeclinePowerLeech {
				t.Fatalf("leech state emitted another owner's action %s", action.Key())
			}
		}
		conversion := &game.ConversionAction{
			BaseAction:     game.BaseAction{Type: game.ActionConversion, PlayerID: "p0"},
			ConversionType: game.ConversionPowerToCoin, Amount: 1,
		}
		if err := game.ApplyActionToState(state, conversion, game.StateActionOptions{}); err == nil {
			t.Fatal("acting player converted while opponent leech response was pending")
		}
	})

	t.Run("delayed_mermaid_town_only_on_own_turn", func(t *testing.T) {
		state := forcedActionPosition(t, 341, models.FactionMermaids, models.FactionEngineers).StateClone()
		anchor := ownedBuildingHex(t, state, "p0")
		state.PendingTownFormations["p0"] = []*game.PendingTownFormation{{
			PlayerID: "p0", Hexes: []board.Hex{anchor}, CanBeDelayed: true,
		}}
		state.CurrentPlayerIndex = 1
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		if position.DecisionPlayer() != "p1" {
			t.Fatalf("opponent is not sole decision owner: %q", position.DecisionPlayer())
		}
		town := &game.SelectTownTileAction{
			BaseAction: game.BaseAction{Type: game.ActionSelectTownTile, PlayerID: "p0"},
			TileType:   models.TownTile5Points, AnchorHex: &anchor,
		}
		if err := game.ApplyActionToState(state, town, game.StateActionOptions{}); err == nil {
			t.Fatal("Mermaids founded a delayed town during the opponent's turn")
		}
		state.CurrentPlayerIndex = 0
		position, err = NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, action := range position.LegalActions() {
			found = found || action.Kind == game.ActionSelectTownTile
		}
		if !found {
			t.Fatal("delayed Mermaid town was not available on its owner's turn")
		}
	})
}

func TestChaosDoubleTurnEnumeratesSecondActionEnabledByFirst(t *testing.T) {
	base := forcedActionPosition(t, 350, models.FactionChaosMagicians, models.FactionEngineers).StateClone()
	player := base.GetPlayer("p0")
	player.HasStrongholdAbility = true
	player.Resources.Coins = 6
	player.Resources.Workers = 0
	player.Resources.Priests = 0
	player.Resources.Power.Bowl1 = 0
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 4
	for action := game.PowerActionBridge; action <= game.PowerActionCoins; action++ {
		base.PowerActions.UsedActions[action] = action != game.PowerActionWorkers
	}
	position, err := NewPosition(base)
	if err != nil {
		t.Fatal(err)
	}
	activate := SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionChaosMagiciansDoubleTurn}
	if !hasAction(position.LegalActions(), activate) {
		t.Fatal("missing Chaos double-turn activation")
	}
	if err := position.Apply(activate); err != nil {
		t.Fatal(err)
	}
	if err := position.Apply(SearchAction{Kind: game.ActionPowerAction, Power: game.PowerActionWorkers}); err != nil {
		t.Fatal(err)
	}
	second := SearchAction{Kind: game.ActionUpgradeBuilding, Hexes: []board.Hex{ownedBuildingHex(t, base, "p0")}, Building: models.BuildingTradingHouse}
	if !hasAction(position.LegalActions(), second) {
		t.Fatalf("first Chaos action did not enable second action %s", second.Key())
	}
	if err := position.Apply(second); err != nil {
		t.Fatal(err)
	}
	if position.DecisionPlayer() != "p1" {
		t.Fatalf("turn did not advance after second Chaos action: %q", position.DecisionPlayer())
	}
}

func TestChaosDoubleTurnExposesInterveningPendingDecision(t *testing.T) {
	base := forcedActionPosition(t, 351, models.FactionChaosMagicians, models.FactionEngineers).StateClone()
	player := base.GetPlayer("p0")
	player.HasStrongholdAbility = true
	buildingHex := ownedBuildingHex(t, base, "p0")
	base.Map.GetHex(buildingHex).Building.Type = models.BuildingTradingHouse
	position, err := NewPosition(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := position.Apply(SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionChaosMagiciansDoubleTurn}); err != nil {
		t.Fatal(err)
	}
	if err := position.Apply(SearchAction{Kind: game.ActionUpgradeBuilding, Hexes: []board.Hex{buildingHex}, Building: models.BuildingTemple}); err != nil {
		t.Fatal(err)
	}
	if position.state.PendingFavorTileSelection == nil {
		t.Fatal("temple upgrade did not expose favor-tile decision")
	}
	for _, action := range position.LegalActions() {
		if action.Kind != game.ActionSelectFavorTile {
			t.Fatalf("pending favor choice leaked non-resolution action %s", action.Key())
		}
	}
	if err := position.Apply(position.LegalActions()[0]); err != nil {
		t.Fatal(err)
	}
	if position.DecisionPlayer() != "p0" || position.state.PendingChaosMagiciansDoubleTurn == nil || position.state.PendingChaosMagiciansDoubleTurn.ActionsRemaining != 1 {
		t.Fatal("Chaos player did not regain the second action after resolving favor choice")
	}
}

func TestChaosDoubleTurnCanEndByPassing(t *testing.T) {
	for _, afterFirstAction := range []bool{false, true} {
		t.Run(map[bool]string{false: "pass_first", true: "pass_second"}[afterFirstAction], func(t *testing.T) {
			base := forcedActionPosition(t, 352, models.FactionChaosMagicians, models.FactionEngineers).StateClone()
			player := base.GetPlayer("p0")
			player.HasStrongholdAbility = true
			position, err := NewPosition(base)
			if err != nil {
				t.Fatal(err)
			}
			if err := position.Apply(SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionChaosMagiciansDoubleTurn}); err != nil {
				t.Fatal(err)
			}
			if afterFirstAction {
				if err := position.Apply(SearchAction{Kind: game.ActionPowerAction, Power: game.PowerActionWorkers}); err != nil {
					t.Fatal(err)
				}
			}
			var pass SearchAction
			for _, action := range position.LegalActions() {
				if action.Kind == game.ActionPass {
					pass = action
					break
				}
			}
			if pass.Kind != game.ActionPass {
				t.Fatal("pass was not legal during Chaos double turn")
			}
			if err := position.Apply(pass); err != nil {
				t.Fatal(err)
			}
			if position.state.PendingChaosMagiciansDoubleTurn != nil {
				t.Fatal("passing did not finish Chaos double turn")
			}
			if !position.state.GetPlayer("p0").HasPassed {
				t.Fatal("passing did not mark Chaos player passed")
			}
			if position.DecisionPlayer() != "p1" {
				t.Fatalf("turn did not advance after pass: %q", position.DecisionPlayer())
			}
		})
	}
}

func hasAction(actions []SearchAction, want SearchAction) bool {
	for _, action := range actions {
		if action.Key() == want.Key() {
			return true
		}
	}
	return false
}

func TestCanonicalHashCoversSearchRelevantPendingState(t *testing.T) {
	base := forcedActionPosition(t, 401, models.FactionWitches, models.FactionEngineers)
	baseline := base.CanonicalHash()
	mutations := map[string]func(*game.GameState){
		"round":     func(gs *game.GameState) { gs.Round++ },
		"resources": func(gs *game.GameState) { gs.GetPlayer("p0").Resources.Coins++ },
		"map": func(gs *game.GameState) {
			for _, hex := range gs.Map.Hexes {
				hex.PartOfTown = !hex.PartOfTown
				break
			}
		},
		"pending_spade":      func(gs *game.GameState) { gs.PendingSpades["p0"] = 1 },
		"pending_cult_spade": func(gs *game.GameState) { gs.PendingCultRewardSpades = map[string]int{"p0": 1} },
		"pending_favor": func(gs *game.GameState) {
			gs.PendingFavorTileSelection = &game.PendingFavorTileSelection{PlayerID: "p0", Count: 1}
		},
		"pending_leech": func(gs *game.GameState) {
			gs.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{Amount: 2, FromPlayerID: "p0", EventID: 1}}
		},
		"pending_town_cult": func(gs *game.GameState) {
			gs.PendingTownCultTopChoice = &game.PendingTownCultTopChoice{PlayerID: "p0", AdvanceAmount: 2, CandidateTracks: []game.CultTrack{game.CultFire}, MaxSelections: 1}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			state := base.StateClone()
			mutate(state)
			position, err := NewPosition(state)
			if err != nil {
				t.Fatal(err)
			}
			if position.CanonicalHash() == baseline {
				t.Fatal("search-relevant mutation did not change hash")
			}
		})
	}
}

func TestFactionSelectionRejectsSameHomeColor(t *testing.T) {
	position := factionSelectionPosition(t)
	if err := position.Apply(SearchAction{Kind: game.ActionSelectFaction, Faction: models.FactionWitches}); err != nil {
		t.Fatal(err)
	}
	paired := SearchAction{Kind: game.ActionSelectFaction, Faction: models.FactionAuren}
	if hasAction(position.LegalActions(), paired) {
		t.Fatal("same-home-color faction was emitted")
	}
	if err := position.Apply(paired); err == nil {
		t.Fatal("same-home-color faction selection applied")
	}
}

func TestLegalCorpusContainsEveryBaseSpecialBranch(t *testing.T) {
	seenKinds := make(map[game.ActionType]bool)
	seenSpecials := make(map[game.SpecialActionType]bool)
	record := func(position *GamePosition) {
		t.Helper()
		actions := position.LegalActions()
		assertActionSetsEqual(t, actions, oracleLegalActions(position))
		for _, action := range actions {
			clone := position.Clone()
			if err := clone.Apply(action); err != nil {
				t.Fatalf("legal coverage action did not apply %s: %v", action.Key(), err)
			}
			seenKinds[action.Kind] = true
			if action.Kind == game.ActionSpecialAction {
				seenSpecials[action.Special] = true
			}
		}
	}
	for i, faction := range baseFactions {
		opponent := models.FactionEngineers
		if faction == models.FactionEngineers || faction == models.FactionDwarves {
			opponent = models.FactionWitches
		}
		state := forcedActionPosition(t, 430+int64(i), faction, opponent).StateClone()
		if faction == models.FactionEngineers {
			hex1 := board.NewHex(0, 0)
			river1 := board.NewHex(0, -1)
			river2 := board.NewHex(1, -1)
			hex2 := board.NewHex(1, -2)
			state.Map.Hexes[hex1] = &board.MapHex{Coord: hex1, Terrain: models.TerrainMountain, Building: &models.Building{
				Type: models.BuildingDwelling, Faction: models.FactionEngineers, PlayerID: "p0", PowerValue: 1,
			}}
			state.Map.Hexes[river1] = &board.MapHex{Coord: river1, Terrain: models.TerrainRiver}
			state.Map.Hexes[river2] = &board.MapHex{Coord: river2, Terrain: models.TerrainRiver}
			state.Map.Hexes[hex2] = &board.MapHex{Coord: hex2, Terrain: models.TerrainMountain}
			state.Map.RiverHexes[river1] = true
			state.Map.RiverHexes[river2] = true
		}
		if faction == models.FactionMermaids {
			river := board.NewHex(0, 0)
			state.Map.Hexes[river] = &board.MapHex{Coord: river, Terrain: models.TerrainRiver}
			state.Map.RiverHexes[river] = true
			for _, h := range []board.Hex{board.NewHex(1, 0), board.NewHex(2, 0), board.NewHex(-1, 0), board.NewHex(-2, 0)} {
				state.Map.Hexes[h] = &board.MapHex{Coord: h, Terrain: models.TerrainLake, Building: &models.Building{
					Type: models.BuildingTradingHouse, Faction: models.FactionMermaids, PlayerID: "p0", PowerValue: 2,
				}}
			}
		}
		if faction == models.FactionNomads {
			placed := false
			for _, source := range sortedHexes(state) {
				sourceHex := state.Map.Hexes[source]
				if sourceHex == nil || sourceHex.Terrain == models.TerrainRiver {
					continue
				}
				for _, target := range source.Neighbors() {
					targetHex := state.Map.Hexes[target]
					if targetHex == nil || targetHex.Terrain == models.TerrainRiver {
						continue
					}
					sourceHex.Terrain = models.TerrainDesert
					sourceHex.Building = &models.Building{Type: models.BuildingStronghold, Faction: faction, PlayerID: "p0", PowerValue: 3}
					targetHex.Terrain = models.TerrainSwamp
					targetHex.Building = nil
					placed = true
					break
				}
				if placed {
					break
				}
			}
			if !placed {
				t.Fatal("could not construct Nomads sandstorm branch")
			}
		}
		state.GetPlayer("p0").HasStrongholdAbility = true
		state.FavorTiles.PlayerTiles["p0"] = append(state.FavorTiles.PlayerTiles["p0"], game.FavorWater2)
		state.BonusCards.PlayerCards["p0"] = game.BonusCardCultAdvance
		delete(state.BonusCards.Available, game.BonusCardCultAdvance)
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		record(position)
	}

	spadeState := forcedActionPosition(t, 449, models.FactionWitches, models.FactionEngineers).StateClone()
	spadeState.BonusCards.PlayerCards["p0"] = game.BonusCardSpade
	delete(spadeState.BonusCards.Available, game.BonusCardSpade)
	spadePosition, err := NewPosition(spadeState)
	if err != nil {
		t.Fatal(err)
	}
	record(spadePosition)

	setup, err := NewBaseGame(450, models.FactionWitches, models.FactionEngineers)
	if err != nil {
		t.Fatal(err)
	}
	record(setup)
	setupBonusState := setup.StateClone()
	setupBonusState.SetupSubphase = game.SetupSubphaseBonusCards
	setupBonusState.SetupDwellingOrder = nil
	setupBonusState.SetupBonusOrder = []string{"p0", "p1"}
	setupBonusState.SetupBonusIndex = 0
	setupBonus, err := NewPosition(setupBonusState)
	if err != nil {
		t.Fatal(err)
	}
	record(setupBonus)
	record(factionSelectionPosition(t))

	base := forcedActionPosition(t, 451, models.FactionWitches, models.FactionEngineers).StateClone()
	recordPending := func(state *game.GameState) {
		t.Helper()
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		record(position)
	}
	favor := base.CloneForUndo()
	favor.PendingFavorTileSelection = &game.PendingFavorTileSelection{PlayerID: "p0", Count: 1}
	recordPending(favor)
	leech := base.CloneForUndo()
	leech.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{Amount: 2, VPCost: 1, FromPlayerID: "p0", EventID: 1}}
	recordPending(leech)
	town := base.CloneForUndo()
	anchor := ownedBuildingHex(t, town, "p0")
	town.PendingTownFormations["p0"] = []*game.PendingTownFormation{{PlayerID: "p0", Hexes: []board.Hex{anchor}}}
	recordPending(town)
	townCult := base.CloneForUndo()
	townCult.PendingTownCultTopChoice = &game.PendingTownCultTopChoice{
		PlayerID: "p0", AdvanceAmount: 1,
		CandidateTracks: []game.CultTrack{game.CultFire, game.CultWater}, MaxSelections: 1,
	}
	recordPending(townCult)
	spade := base.CloneForUndo()
	spade.PendingSpades["p0"] = 1
	spade.PendingSpadeBuildAllowed["p0"] = true
	recordPending(spade)
	cultSpade := base.CloneForUndo()
	cultSpade.PendingCultRewardSpades = map[string]int{"p0": 1}
	recordPending(cultSpade)
	halflings := forcedActionPosition(t, 452, models.FactionHalflings, models.FactionWitches).StateClone()
	halflings.PendingHalflingsSpades = &game.PendingHalflingsSpades{PlayerID: "p0", SpadesRemaining: 1}
	recordPending(halflings)
	halflingsBuild := halflings.CloneForUndo()
	var transformed board.Hex
	for h, mapHex := range halflingsBuild.Map.Hexes {
		if mapHex != nil && mapHex.Building == nil && mapHex.Terrain != models.TerrainRiver {
			transformed = h
			mapHex.Terrain = models.TerrainPlains
			break
		}
	}
	halflingsBuild.PendingHalflingsSpades = &game.PendingHalflingsSpades{
		PlayerID: "p0", SpadesRemaining: 0, TransformedHexes: []board.Hex{transformed},
	}
	recordPending(halflingsBuild)
	darklings := forcedActionPosition(t, 453, models.FactionDarklings, models.FactionWitches).StateClone()
	darklings.PendingDarklingsPriestOrdination = &game.PendingDarklingsPriestOrdination{PlayerID: "p0"}
	recordPending(darklings)
	cultists := forcedActionPosition(t, 454, models.FactionCultists, models.FactionEngineers).StateClone()
	cultists.PendingCultistsCultSelection = &game.PendingCultistsCultSelection{PlayerID: "p0"}
	recordPending(cultists)

	expectedKinds := []game.ActionType{
		game.ActionTransformAndBuild, game.ActionUpgradeBuilding, game.ActionAdvanceShipping,
		game.ActionAdvanceDigging, game.ActionSendPriestToCult, game.ActionPowerAction,
		game.ActionSpecialAction, game.ActionPass, game.ActionSetupDwelling,
		game.ActionUseCultSpade, game.ActionAcceptPowerLeech, game.ActionDeclinePowerLeech,
		game.ActionSelectFavorTile, game.ActionApplyHalflingsSpade,
		game.ActionBuildHalflingsDwelling, game.ActionSkipHalflingsDwelling,
		game.ActionUseDarklingsPriestOrdination, game.ActionSelectCultistsCultTrack,
		game.ActionSelectFaction, game.ActionSetupBonusCard, game.ActionSelectTownTile,
		game.ActionSelectTownCultTop, game.ActionDiscardPendingSpade,
		game.ActionConversion, game.ActionBurnPower, game.ActionEngineersBridge,
	}
	for _, kind := range expectedKinds {
		if !seenKinds[kind] {
			t.Fatalf("legal corpus did not contain action kind %v", kind)
		}
	}
	for _, special := range []game.SpecialActionType{
		game.SpecialActionAurenCultAdvance, game.SpecialActionWitchesRide,
		game.SpecialActionSwarmlingsUpgrade, game.SpecialActionChaosMagiciansDoubleTurn,
		game.SpecialActionGiantsTransform, game.SpecialActionNomadsSandstorm,
		game.SpecialActionWater2CultAdvance, game.SpecialActionBonusCardSpade,
		game.SpecialActionBonusCardCultAdvance, game.SpecialActionMermaidsRiverTown,
	} {
		if !seenSpecials[special] {
			t.Fatalf("legal corpus did not contain special subtype %v", special)
		}
	}
}

func TestPositionApplyIsTransactionalOnError(t *testing.T) {
	position := forcedActionPosition(t, 501, models.FactionWitches, models.FactionEngineers)
	before := position.CanonicalHash()
	err := position.Apply(SearchAction{Kind: game.ActionTransformAndBuild, Hexes: []board.Hex{{Q: 999, R: 999}}, Terrain: models.TerrainForest})
	if err == nil {
		t.Fatal("expected invalid action to fail")
	}
	if got := position.CanonicalHash(); got != before {
		t.Fatalf("failed apply mutated position: got %s want %s", got, before)
	}
}

func TestLegalActionsHaveUniquePostValidationSemantics(t *testing.T) {
	for _, faction := range []models.FactionType{models.FactionFakirs, models.FactionDwarves} {
		position := forcedActionPosition(t, 550+int64(faction), faction, models.FactionWitches)
		seen := make(map[string]string)
		successors := make(map[Hash128]string)
		for _, searchAction := range position.LegalActions() {
			engineAction, err := searchAction.GameAction("p0")
			if err != nil {
				t.Fatal(err)
			}
			state := position.StateClone()
			if err := game.ApplyActionToState(state, engineAction, game.StateActionOptions{}); err != nil {
				t.Fatalf("emitted action failed: %s: %v", searchAction.Key(), err)
			}
			canonical, err := SearchActionFromGameAction(engineAction)
			if err != nil {
				t.Fatal(err)
			}
			if canonical.Key() != searchAction.Key() {
				t.Fatalf("emitted action was not canonical: got %s want %s", searchAction.Key(), canonical.Key())
			}
			if previous, exists := seen[canonical.Key()]; exists {
				t.Fatalf("actions %s and %s share post-validation semantics", previous, searchAction.Key())
			}
			seen[canonical.Key()] = searchAction.Key()
			next := position.Clone()
			if err := next.Apply(searchAction); err != nil {
				t.Fatal(err)
			}
			hash := next.CanonicalHash()
			if previous, exists := successors[hash]; exists {
				t.Fatalf("actions %s and %s produce the same successor %s", previous, searchAction.Key(), hash)
			}
			successors[hash] = searchAction.Key()
		}
	}
}

func TestV0ProfileBoundary(t *testing.T) {
	base := forcedActionPosition(t, 601, models.FactionWitches, models.FactionEngineers).StateClone()
	tests := map[string]func(*game.GameState){
		"setup":         func(gs *game.GameState) { gs.SetupMode = game.SetupModeAuction },
		"turn_order":    func(gs *game.GameState) { gs.TurnOrderPolicy = game.TurnOrderPolicyCyclicFromFirstPasser },
		"final_scoring": func(gs *game.GameState) { gs.FireIceFinalScoringSetting = game.FireIceFinalScoringOn },
		"scratch_state": func(gs *game.GameState) { gs.SuppressTurnAdvance = true },
		"replay_queue":  func(gs *game.GameState) { gs.ReplayRiverBuildHexes = map[string][]board.Hex{"p0": {{Q: 1, R: 1}}} },
		"fan_pending": func(gs *game.GameState) {
			gs.PendingGoblinsCultSteps = &game.PendingGoblinsCultSteps{PlayerID: "p0", StepsRemaining: 1}
		},
		"nil_faction": func(gs *game.GameState) { gs.GetPlayer("p0").Faction = nil },
		"same_home":   func(gs *game.GameState) { gs.GetPlayer("p1").Faction = factions.NewAuren() },
		"expansion_town_tile": func(gs *game.GameState) {
			gs.TownTiles.Available[models.TownTile11Points] = 1
		},
		"expansion_bonus_card": func(gs *game.GameState) {
			gs.BonusCards.Available[game.BonusCardShippingVP] = 0
		},
		"fan_player_state": func(gs *game.GameState) {
			gs.GetPlayer("p0").HasStartingTerrain = true
			gs.GetPlayer("p0").StartingTerrain = models.TerrainMountain
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			state := base.CloneForUndo()
			mutate(state)
			if _, err := NewPosition(state); err == nil {
				t.Fatal("expected unsupported profile to be rejected")
			}
		})
	}

	alternate, err := game.NewGameStateWithMap(board.MapLakes)
	if err != nil {
		t.Fatal(err)
	}
	if err := alternate.AddPlayer("p0", factions.NewWitches()); err != nil {
		t.Fatal(err)
	}
	if err := alternate.AddPlayer("p1", factions.NewEngineers()); err != nil {
		t.Fatal(err)
	}
	alternate.TurnOrder = []string{"p0", "p1"}
	if _, err := NewPosition(alternate); err == nil {
		t.Fatal("expected alternate map to be rejected")
	}

	fan := game.NewGameState()
	if err := fan.AddPlayer("p0", factions.NewFaction(models.FactionArchitects)); err != nil {
		t.Fatal(err)
	}
	if err := fan.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatal(err)
	}
	fan.TurnOrder = []string{"p0", "p1"}
	if _, err := NewPosition(fan); err == nil {
		t.Fatal("expected fan faction to be rejected")
	}
}

func assertLegalActionRoundTrips(t *testing.T, position *GamePosition) {
	t.Helper()
	before, err := position.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	playerID := string(position.DecisionPlayer())
	for _, action := range position.LegalActions() {
		engineAction, err := action.GameAction(playerID)
		if err != nil {
			t.Fatalf("construct engine action %s: %v", action.Key(), err)
		}
		roundTrip, err := SearchActionFromGameAction(engineAction)
		if err != nil {
			t.Fatalf("round-trip engine action %s: %v", action.Key(), err)
		}
		if roundTrip.Key() != action.Key() {
			t.Fatalf("engine action round trip mismatch\ngot=%s\nwant=%s", roundTrip.Key(), action.Key())
		}
	}
	after, err := position.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("legal enumeration mutated its source state")
	}
}

func FuzzCloneApplyDoesNotMutateParent(f *testing.F) {
	f.Add(int64(1), uint8(0))
	f.Add(int64(99), uint8(7))
	f.Fuzz(func(t *testing.T, seed int64, choice uint8) {
		position, err := NewBaseGame(seed, models.FactionWitches, models.FactionEngineers)
		if err != nil {
			t.Fatal(err)
		}
		actions := position.LegalActions()
		if len(actions) == 0 {
			t.Skip()
		}
		before := position.CanonicalHash()
		clone := position.Clone()
		if err := clone.Apply(actions[int(choice)%len(actions)]); err != nil {
			t.Fatal(err)
		}
		if position.CanonicalHash() != before {
			t.Fatal("applying to clone mutated parent")
		}
	})
}

func forcedActionPosition(t *testing.T, seed int64, first, second models.FactionType) *GamePosition {
	t.Helper()
	position, err := NewBaseGame(seed, first, second)
	if err != nil {
		t.Fatal(err)
	}
	state := position.StateClone()
	state.Phase = game.PhaseAction
	state.SetupSubphase = game.SetupSubphaseComplete
	state.Round = 3
	state.CurrentPlayerIndex = 0
	state.SetupDwellingOrder = nil
	state.SetupBonusOrder = nil
	for _, player := range state.Players {
		player.Resources.Coins = 30
		player.Resources.Workers = 12
		player.Resources.Priests = 5
		player.Resources.Power.Bowl1 = 2
		player.Resources.Power.Bowl2 = 8
		player.Resources.Power.Bowl3 = 10
		player.HasPassed = false
		player.HasStrongholdAbility = false
	}
	placeOwnedBuilding(t, state, "p0")
	placeOwnedBuilding(t, state, "p1")
	position, err = NewPosition(state)
	if err != nil {
		t.Fatal(err)
	}
	return position
}

func placeOwnedBuilding(t *testing.T, state *game.GameState, playerID string) board.Hex {
	t.Helper()
	player := state.GetPlayer(playerID)
	for _, h := range sortedHexes(state) {
		mapHex := state.Map.Hexes[h]
		if mapHex.Terrain == models.TerrainRiver || mapHex.Building != nil {
			continue
		}
		mapHex.Terrain = player.Faction.GetHomeTerrain()
		mapHex.Building = &models.Building{Type: models.BuildingDwelling, PlayerID: playerID, Faction: player.Faction.GetType(), PowerValue: 1}
		return h
	}
	t.Fatal("no land hex for building")
	return board.Hex{}
}

func ownedBuildingHex(t *testing.T, state *game.GameState, playerID string) board.Hex {
	t.Helper()
	for h, mapHex := range state.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			return h
		}
	}
	t.Fatal("owned building not found")
	return board.Hex{}
}

func factionSelectionPosition(t *testing.T) *GamePosition {
	t.Helper()
	state := game.NewGameState()
	state.TownTiles = game.NewBaseTownTileState()
	state.BonusCards.SetAvailableBonusCards([]game.BonusCardType{
		game.BonusCardPriest, game.BonusCardShipping, game.BonusCardDwellingVP,
		game.BonusCardWorkerPower, game.BonusCardSpade,
	})
	if err := state.AddPlayer("p0", nil); err != nil {
		t.Fatal(err)
	}
	if err := state.AddPlayer("p1", nil); err != nil {
		t.Fatal(err)
	}
	state.Phase = game.PhaseFactionSelection
	state.TurnOrder = []string{"p0", "p1"}
	position, err := NewPosition(state)
	if err != nil {
		t.Fatal(err)
	}
	return position
}
