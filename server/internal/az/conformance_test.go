package az

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// User-confirmed Mermaid rulings: river opportunities are current-board
// choices, not reservations; overlaps are resolved by the chosen river anchor.
func TestMermaidCurrentBoardTownChoices(t *testing.T) {
	newFixture := func() *GamePosition {
		state := forcedActionPosition(t, 9601, models.FactionMermaids, models.FactionWitches).StateClone()
		setupMermaidTownChoices(state)
		p, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	first, second := board.NewHex(3, 2), board.NewHex(2, 3)
	choice := func(hex board.Hex) SearchAction {
		return SearchAction{Kind: game.ActionSelectTownTile, TownTile: models.TownTile7Points, Hexes: []board.Hex{hex}}
	}
	for _, selected := range []board.Hex{first, second} {
		p := newFixture()
		if !hasAction(p.LegalActions(), choice(first)) || !hasAction(p.LegalActions(), choice(second)) {
			t.Fatal("both overlapping river anchors must be offered")
		}
		workers := p.state.GetPlayer("p0").Resources.Workers
		if err := p.Apply(choice(selected)); err != nil {
			t.Fatal(err)
		}
		if !p.state.Map.GetHex(selected).HasTownTile || p.state.GetPlayer("p0").Resources.Workers != workers+2 {
			t.Fatal("chosen river must receive the town and its workers immediately")
		}
		for _, action := range p.LegalActions() {
			if action.Kind == game.ActionSelectTownTile {
				t.Fatal("overlapping opportunity survived town claim")
			}
		}
		if p.state.PendingFreeActionsPlayerID != "" {
			t.Fatal("pre-main town consumed the main action")
		}
	}
	for _, forbidden := range []string{"merged", "passed", "leech", "opponent"} {
		p := newFixture()
		state := p.StateClone()
		switch forbidden {
		case "merged":
			// A new connection to an already founded town invalidates the
			// previously offered opportunity, even though its old buildings
			// themselves have never been marked PartOfTown.
			h := board.NewHex(0, 2)
			state.Map.Hexes[h] = &board.MapHex{Coord: h, Terrain: models.TerrainLake, PartOfTown: true, Building: &models.Building{Type: models.BuildingDwelling, PowerValue: 1, PlayerID: "p0", Faction: models.FactionMermaids}}
		case "passed":
			state.GetPlayer("p0").HasPassed = true
		case "leech":
			state.PendingLeechOffers["p0"] = []*game.PowerLeechOffer{{FromPlayerID: "p1", Amount: 2}}
		case "opponent":
			state.CurrentPlayerIndex = 1
		}
		// Deliberately retain the earlier pending record: claim-time validation
		// must reject it rather than trusting this stale opportunity.
		a := &game.SelectTownTileAction{BaseAction: game.BaseAction{Type: game.ActionSelectTownTile, PlayerID: "p0"}, TileType: models.TownTile7Points, AnchorHex: &first}
		before := state.CloneForUndo()
		if err := game.ApplyActionToState(state, a, game.StateActionOptions{}); err == nil {
			t.Fatalf("accepted Mermaid town while %s", forbidden)
		}
		if state.GetPlayer("p0").TownsFormed != before.GetPlayer("p0").TownsFormed {
			t.Fatal("rejected claim formed a town")
		}
	}
	t.Run("canonical cache refresh", func(t *testing.T) {
		p := newFixture()
		stale, fresh := p.StateClone(), p.StateClone()
		stale.PendingTownFormations["p0"] = []*game.PendingTownFormation{{PlayerID: "p0", Hexes: []board.Hex{board.NewHex(1, 2)}, CanBeDelayed: true}}
		fresh.PendingTownFormations = nil
		a, err := NewPosition(stale)
		if err != nil {
			t.Fatal(err)
		}
		b, err := NewPosition(fresh)
		if err != nil {
			t.Fatal(err)
		}
		if a.CanonicalHash() != b.CanonicalHash() {
			t.Fatal("stale delayed opportunity cache changed canonical state")
		}
		if len(stale.PendingTownFormations["p0"][0].Hexes) != 1 {
			t.Fatal("NewPosition mutated input")
		}
	})
}

func TestSimultaneousTownRewardsCanBeChosenInEitherOrder(t *testing.T) {
	for _, row := range []int{0, 4} {
		state := forcedActionPosition(t, 9602, models.FactionWitches, models.FactionMermaids).StateClone()
		for _, hex := range state.Map.Hexes {
			if hex.Building != nil && hex.Building.PlayerID == "p0" {
				hex.Building = nil
			}
		}
		for _, r := range []int{0, 4} {
			kinds := []models.BuildingType{models.BuildingStronghold, models.BuildingTradingHouse, models.BuildingDwelling, models.BuildingDwelling}
			if r == 4 {
				kinds[0] = models.BuildingSanctuary
			}
			for q, kind := range kinds {
				h := board.NewHex(q, r)
				state.Map.Hexes[h] = &board.MapHex{Coord: h, Terrain: models.TerrainForest, Building: &models.Building{Type: kind, PowerValue: game.GetPowerValue(kind), PlayerID: "p0", Faction: models.FactionWitches}}
			}
		}
		state.CheckAllTownFormations("p0")
		p, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		anchor := board.NewHex(0, row)
		choice := SearchAction{Kind: game.ActionSelectTownTile, TownTile: models.TownTile5Points, Hexes: []board.Hex{anchor}}
		if !hasAction(p.LegalActions(), choice) {
			t.Fatalf("missing reward order starting at row%d", row)
		}
		if err := p.Apply(choice); err != nil {
			t.Fatal(err)
		}
		if !p.state.Map.GetHex(anchor).PartOfTown || p.state.Map.GetHex(board.NewHex(0, 4-row)).PartOfTown || len(p.state.PendingTownFormations["p0"]) != 1 {
			t.Fatal("selected reward must apply to the chosen town only")
		}
	}
}

// Source-derived boundary fixtures, not complete externally adjudicated games.
// Rulebook Transform/Build and Halflings appendix; FAQ 2.1: distribute only
// after reaching home, all transformations precede at most one dwelling,
// leftovers on another space cannot be supplemented with paid spades.
func TestHalflingsStrongholdPublicRules(t *testing.T) {
	first, second, third := board.NewHex(0, 0), board.NewHex(1, 0), board.NewHex(2, 0)
	fixture := func() *GamePosition {
		state := forcedActionPosition(t, 9510, models.FactionHalflings, models.FactionMermaids).StateClone()
		for _, hex := range state.Map.Hexes {
			hex.Building = nil
		}
		for i, hex := range []board.Hex{board.NewHex(0, 1), board.NewHex(2, 1)} {
			state.Map.GetHex(hex).Terrain = models.TerrainPlains
			state.Map.GetHex(hex).Building = &models.Building{Type: models.BuildingStronghold, PlayerID: "p0", Faction: models.FactionHalflings, PowerValue: 3}
			if i == 1 {
				state.Map.GetHex(hex).Building.Type = models.BuildingDwelling
				state.Map.GetHex(hex).Building.PowerValue = 1
			}
		}
		for _, hex := range []board.Hex{first, second, third} {
			state.Map.GetHex(hex).Terrain = models.TerrainSwamp
		}
		state.PendingHalflingsSpades = &game.PendingHalflingsSpades{PlayerID: "p0", SpadesRemaining: 3}
		for i, tile := range state.ScoringTiles.Tiles {
			if tile.Type == game.ScoringSpades {
				state.ScoringTiles.Tiles[0], state.ScoringTiles.Tiles[i] = tile, state.ScoringTiles.Tiles[0]
			}
		}
		for _, tile := range game.GetAllScoringTiles() {
			if tile.Type == game.ScoringSpades {
				state.ScoringTiles.Tiles[0] = tile
			}
		}
		state.Round = 1
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		return position
	}
	apply := func(position *GamePosition, hex board.Hex, terrain models.TerrainType) {
		t.Helper()
		action := SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{hex}, Terrain: terrain}
		found := false
		for _, legal := range position.LegalActions() {
			if legal.Key() == action.Key() {
				found = true
			}
		}
		if !found {
			t.Fatalf("legal reward missing: %s", action.Key())
		}
		if err := position.Apply(action); err != nil {
			t.Fatal(err)
		}
	}
	reject := func(position *GamePosition, action SearchAction) {
		t.Helper()
		before, _ := position.CanonicalJSON()
		if err := position.Apply(action); err == nil {
			t.Fatalf("accepted %s", action.Key())
		}
		after, _ := position.CanonicalJSON()
		if !bytes.Equal(before, after) {
			t.Fatal("rejected action changed state")
		}
	}
	t.Run("three home spaces and one dwelling", func(t *testing.T) {
		p := fixture()
		vp, workers := p.state.GetPlayer("p0").VictoryPoints, p.state.GetPlayer("p0").Resources.Workers
		for _, hex := range []board.Hex{first, second, third} {
			apply(p, hex, models.TerrainPlains)
		}
		if p.state.GetPlayer("p0").VictoryPoints != vp+9 || p.state.GetPlayer("p0").Resources.Workers != workers {
			t.Fatal("three free spades must give 3 faction + 6 tile VP without worker cost")
		}
		if err := p.Apply(SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{second}}); err != nil {
			t.Fatal(err)
		}
		if p.state.PendingHalflingsSpades != nil || p.state.PendingFreeActionsPlayerID != "p0" {
			t.Fatal("reward must close into the owner's end-turn window")
		}
		reject(p, SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{first}})
	})
	t.Run("reachability direction and per-space completion", func(t *testing.T) {
		p := fixture()
		reject(p, SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{board.NewHex(8, 0)}, Terrain: models.TerrainPlains})
		apply(p, first, models.TerrainLake)
		reject(p, SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{second}, Terrain: models.TerrainPlains})
		reject(p, SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{first}, Terrain: models.TerrainForest})
		reject(p, SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{first}})
		if err := p.Apply(SearchAction{Kind: game.ActionSkipHalflingsDwelling}); err != nil {
			t.Fatal(err)
		}
		if p.state.PendingHalflingsSpades != nil {
			t.Fatal("unused spades must be discardable")
		}
	})
	t.Run("build forfeits leftovers", func(t *testing.T) {
		p := fixture()
		apply(p, first, models.TerrainPlains)
		if err := p.Apply(SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{first}}); err != nil {
			t.Fatal(err)
		}
		if p.state.PendingHalflingsSpades != nil {
			t.Fatal("building must end all transforming")
		}
	})
	t.Run("dwelling supply", func(t *testing.T) {
		p := fixture()
		state := p.StateClone()
		count := 1 // The fixture already owns one dwelling.
		for _, hex := range state.Map.Hexes {
			if count == 8 {
				break
			}
			if hex.Building == nil && hex.Terrain != models.TerrainRiver && hex.Coord != first {
				hex.Terrain = models.TerrainPlains
				hex.Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p0", Faction: models.FactionHalflings, PowerValue: 1}
				count++
			}
		}
		state.PendingHalflingsSpades.TransformedHexes = []board.Hex{first}
		state.Map.GetHex(first).Terrain = models.TerrainPlains
		var err error
		p, err = NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		reject(p, SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{first}})
	})
	t.Run("leftovers cannot buy topup", func(t *testing.T) {
		p := fixture()
		p.state.Map.GetHex(third).Terrain = models.TerrainLake
		apply(p, first, models.TerrainPlains)
		apply(p, second, models.TerrainPlains)
		workers, vp := p.state.GetPlayer("p0").Resources.Workers, p.state.GetPlayer("p0").VictoryPoints
		for _, legal := range p.LegalActions() {
			if legal.Kind == game.ActionApplyHalflingsSpade && len(legal.Hexes) == 1 && legal.Hexes[0] == third && legal.Terrain == models.TerrainPlains {
				t.Fatal("leftover spade exposed illegal paid completion")
			}
		}
		reject(p, SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{third}, Terrain: models.TerrainPlains})
		apply(p, third, models.TerrainSwamp)
		if p.state.GetPlayer("p0").Resources.Workers != workers || p.state.GetPlayer("p0").VictoryPoints != vp+3 {
			t.Fatal("leftover free intermediate spade must cost no workers and score one spade")
		}
	})
}

// This is INTERNAL invariant/replay coverage, not an independent rules oracle or
// an externally adjudicated game corpus. No resources or strongholds are granted:
// every state is reached from normal setup through the public action interface.
func TestAllBaseFactionsNaturalGameInvariants(t *testing.T) {
	pairs := [][2]models.FactionType{
		{models.FactionChaosMagicians, models.FactionEngineers},
		{models.FactionGiants, models.FactionDwarves},
		{models.FactionFakirs, models.FactionAuren},
		{models.FactionNomads, models.FactionWitches},
		{models.FactionHalflings, models.FactionMermaids},
		{models.FactionCultists, models.FactionDarklings},
		{models.FactionAlchemists, models.FactionSwarmlings},
	}
	for index, pair := range pairs {
		for replicate := 0; replicate < 2; replicate++ {
			seed := int64(9100 + index*10 + replicate)
			t.Run(fmt.Sprintf("%s_%s_seed%d", pair[0], pair[1], seed), func(t *testing.T) {
				position, err := NewBaseGame(seed, pair[0], pair[1])
				if err != nil {
					t.Fatal(err)
				}
				replayed, err := NewBaseGame(seed, pair[0], pair[1])
				if err != nil {
					t.Fatal(err)
				}
				rng := rand.New(rand.NewSource(seed))
				rounds := make(map[int]bool)
				var trace []string
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("seed %d action trace: %v", seed, trace)
					}
				})
				const tripwire = 1000
				for ply := 0; !position.IsTerminal(); ply++ {
					if ply == tripwire {
						t.Fatalf("seed %d hit tripwire, round=%d phase=%v", seed, position.state.Round, position.state.Phase)
					}
					rounds[position.state.Round] = true
					assertNaturalResourceBounds(t, position, ply)
					before, err := position.CanonicalJSON()
					if err != nil {
						t.Fatal(err)
					}
					beforeHash := position.CanonicalHash()
					// Both malformed serialization and an impossible recognized action
					// must leave canonical state and its cached hash untouched.
					for _, invalid := range []SearchAction{{Kind: game.ActionType(-1)}, {Kind: game.ActionBurnPower, Amount: 1000000}} {
						if err := position.Apply(invalid); err == nil {
							t.Fatalf("ply %d accepted impossible action %s", ply, invalid.Key())
						}
						after, err := position.CanonicalJSON()
						if err != nil || !bytes.Equal(before, after) || position.CanonicalHash() != beforeHash {
							t.Fatalf("ply %d failed action mutated state/hash", ply)
						}
					}
					legal := position.LegalActions()
					if len(legal) == 0 {
						t.Fatalf("seed %d ply %d stuck: round=%d phase=%v actor=%s", seed, ply, position.state.Round, position.state.Phase, position.DecisionPlayer())
					}
					choice := legal[rng.Intn(len(legal))]
					trace = append(trace, choice.Key())
					clone := position.Clone().(*GamePosition)
					if err := clone.Apply(choice); err != nil {
						t.Fatalf("ply %d clone action failed: %v", ply, err)
					}
					afterClone, _ := position.CanonicalJSON()
					if !bytes.Equal(before, afterClone) || position.CanonicalHash() != beforeHash {
						t.Fatalf("ply %d clone mutation escaped into parent", ply)
					}
					if err := position.Apply(choice); err != nil {
						t.Fatalf("ply %d legal action failed: %v", ply, err)
					}
					if err := replayed.Apply(choice); err != nil {
						t.Fatalf("ply %d replay failed: %v", ply, err)
					}
					if position.CanonicalHash() != replayed.CanonicalHash() || position.CanonicalHash() != clone.CanonicalHash() {
						t.Fatalf("seed %d replay/clone diverged at ply %d action %s", seed, ply, choice.Key())
					}
				}
				assertNaturalResourceBounds(t, position, tripwire)
				for round := 1; round <= 6; round++ {
					if !rounds[round] {
						t.Fatalf("never visited round %d", round)
					}
				}
				if !replayed.IsTerminal() || position.Outcome("p0") != -position.Outcome("p1") {
					t.Fatal("terminal/outcome disagreement")
				}
				for _, id := range []PlayerID{"p0", "p1"} {
					vp, ok := position.FinalVP(id)
					replayVP, replayOK := replayed.FinalVP(id)
					if !ok || !replayOK || vp != replayVP {
						t.Fatalf("final VP mismatch for %s", id)
					}
				}
			})
		}
	}
}

func assertNaturalResourceBounds(t *testing.T, position *GamePosition, ply int) {
	t.Helper()
	for id, player := range position.state.Players {
		// Physical base-game supplies and home terrain are independent of the
		// action implementation that placed or upgraded these structures.
		counts := make(map[models.BuildingType]int)
		for hex, space := range position.state.Map.Hexes {
			if space.Building == nil || space.Building.PlayerID != id {
				continue
			}
			if space.Terrain != player.Faction.GetHomeTerrain() || space.Building.Faction != player.Faction.GetType() {
				t.Fatalf("ply %d invalid building terrain/faction for %s at %v: terrain=%v home=%v building faction=%v owner faction=%v", ply, id, hex, space.Terrain, player.Faction.GetHomeTerrain(), space.Building.Faction, player.Faction.GetType())
			}
			counts[space.Building.Type]++
		}
		for kind, limit := range map[models.BuildingType]int{
			models.BuildingDwelling: 8, models.BuildingTradingHouse: 4,
			models.BuildingTemple: 3, models.BuildingStronghold: 1, models.BuildingSanctuary: 1,
		} {
			if counts[kind] > limit {
				t.Fatalf("ply %d exceeds %v supply for %s: %d > %d", ply, kind, id, counts[kind], limit)
			}
		}
		if player.BridgesBuilt < 0 || player.BridgesBuilt > 3 {
			t.Fatalf("ply %d invalid bridge supply for %s: %d", ply, id, player.BridgesBuilt)
		}
		r := player.Resources
		if r == nil || r.Power == nil {
			t.Fatalf("ply %d missing resources for %s", ply, id)
		}
		availableKeys := player.Keys
		// The engine represents immediate keys from mandatory, not-yet-chosen
		// towns as credit. Cult advancement can temporarily consume this credit.
		for _, town := range position.state.PendingTownFormations[id] {
			if town != nil && !town.CanBeDelayed {
				availableKeys++
			}
		}
		if r.Coins < 0 || r.Workers < 0 || r.Priests < 0 || player.VictoryPoints < 0 || availableKeys < 0 {
			t.Fatalf("ply %d negative resources/VP/keys for %s: %+v VP=%d keys=%d", ply, id, r, player.VictoryPoints, player.Keys)
		}
		if r.Power.Bowl1 < 0 || r.Power.Bowl2 < 0 || r.Power.Bowl3 < 0 || r.Power.Bowl1+r.Power.Bowl2+r.Power.Bowl3 > 12 {
			t.Fatalf("ply %d invalid power bowls for %s: %+v", ply, id, r.Power)
		}
		if position.state.GetTotalOwnedPriests(id) > 7 {
			t.Fatalf("ply %d exceeds seven priest pieces for %s", ply, id)
		}
	}
}

func TestLeechPrecedesRewardsInPublicActionSet(t *testing.T) {
	for _, reward := range []string{"favor", "town"} {
		for _, passed := range []bool{false, true} {
			state := forcedActionPosition(t, 9301, models.FactionWitches, models.FactionEngineers).StateClone()
			state.GetPlayer("p1").HasPassed = passed
			state.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{{FromPlayerID: "p0", Amount: 2}}
			if reward == "favor" {
				state.PendingFavorTileSelection = &game.PendingFavorTileSelection{PlayerID: "p0", Count: 1}
			} else {
				state.PendingTownFormations["p0"] = []*game.PendingTownFormation{{PlayerID: "p0"}}
			}
			position, err := NewPosition(state)
			if err != nil {
				t.Fatal(err)
			}
			if got := position.DecisionPlayer(); got != "p1" {
				t.Fatalf("%s passed=%v: owner=%s", reward, passed, got)
			}
			actions := position.LegalActions()
			if len(actions) != 2 {
				t.Fatalf("expected accept/decline, got %v", actions)
			}
			for _, action := range actions {
				if action.Kind != game.ActionAcceptPowerLeech && action.Kind != game.ActionDeclinePowerLeech {
					t.Fatalf("builder action before leech: %s", action.Key())
				}
			}
			if err := position.Apply(SearchAction{Kind: game.ActionDeclinePowerLeech}); err != nil {
				t.Fatal(err)
			}
			if got := position.DecisionPlayer(); got != "p0" {
				t.Fatalf("builder reward not restored: %s", got)
			}
		}
	}
}

func TestSimultaneousTownRewardChoicesInPublicActionSet(t *testing.T) {
	for _, faction := range []models.FactionType{models.FactionDarklings, models.FactionChaosMagicians} {
		state := forcedActionPosition(t, 9302, faction, models.FactionEngineers).StateClone()
		anchor := ownedBuildingHex(t, state, "p0")
		state.PendingTownFormations["p0"] = []*game.PendingTownFormation{{PlayerID: "p0", Hexes: []board.Hex{anchor}}}
		otherKind := game.ActionSelectFavorTile
		if faction == models.FactionDarklings {
			state.PendingDarklingsPriestOrdination = &game.PendingDarklingsPriestOrdination{PlayerID: "p0"}
			otherKind = game.ActionUseDarklingsPriestOrdination
		} else {
			state.PendingFavorTileSelection = &game.PendingFavorTileSelection{PlayerID: "p0", Count: 2}
		}
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		foundTown, foundOther := false, false
		for _, action := range position.LegalActions() {
			foundTown = foundTown || action.Kind == game.ActionSelectTownTile
			foundOther = foundOther || action.Kind == otherKind
		}
		if !foundTown || !foundOther {
			t.Fatalf("%s simultaneous town/reward choices missing: town=%v other=%v", faction, foundTown, foundOther)
		}
	}
}

func TestBaseSpecialActionsCannotTerraformRiver(t *testing.T) {
	for _, faction := range []models.FactionType{models.FactionGiants, models.FactionNomads} {
		state := forcedActionPosition(t, 9451, faction, models.FactionEngineers).StateClone()
		player := state.GetPlayer("p0")
		player.HasStrongholdAbility = true
		target, origin := board.NewHex(0, 0), board.NewHex(0, 1)
		state.Map.GetHex(target).Terrain = models.TerrainRiver
		state.Map.GetHex(target).Building = nil
		state.Map.GetHex(origin).Building = &models.Building{Type: models.BuildingDwelling, PlayerID: "p0", Faction: faction, PowerValue: 1}
		kind := game.SpecialActionGiantsTransform
		if faction == models.FactionNomads {
			kind = game.SpecialActionNomadsSandstorm
		}
		position, err := NewPosition(state)
		if err != nil {
			t.Fatal(err)
		}
		invalid := SearchAction{Kind: game.ActionSpecialAction, Special: kind, Hexes: []board.Hex{target}}
		if hasAction(position.LegalActions(), invalid) {
			t.Fatalf("%s river transformation is listed as legal", faction)
		}
		before := position.CanonicalHash()
		if err := position.Apply(invalid); err == nil {
			t.Fatalf("%s river transformation applied", faction)
		}
		if position.CanonicalHash() != before {
			t.Fatal("rejected river special changed position")
		}
	}
}

func TestCultSpecialNoEffectAllowedInPublicActions(t *testing.T) {
	state := forcedActionPosition(t, 9452, models.FactionAuren, models.FactionEngineers).StateClone()
	player := state.GetPlayer("p0")
	player.HasStrongholdAbility = true
	player.Keys = 0
	for _, track := range cultTracks {
		player.CultPositions[track] = 9
		state.CultTracks.PlayerPositions["p0"][track] = 9
	}
	position, err := NewPosition(state)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, action := range position.LegalActions() {
		if action.Kind == game.ActionSpecialAction && action.Special == game.SpecialActionAurenCultAdvance {
			count++
			clone := position.Clone()
			if err := clone.Apply(action); err != nil {
				t.Fatalf("listed capped cult action rejected: %v", err)
			}
		}
	}
	if count != 4 {
		t.Fatalf("capped Auren cult choices = %d, want 4", count)
	}
}
