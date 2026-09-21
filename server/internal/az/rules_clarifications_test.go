package az

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// Working interpretation selected by the user, still tentative: simultaneous
// Halflings town rewards and stronghold spades may resolve in either order.
// These are pending-decision boundary fixtures, not town-geometry fixtures.
func TestTentativeHalflingsTownAndSpadesEitherOrder(t *testing.T) {
	for _, order := range []string{"town-spade-build", "spade-town-build", "spade-build-town"} {
		t.Run(order, func(t *testing.T) {
			state := forcedActionPosition(t, 9511, models.FactionHalflings, models.FactionWitches).StateClone()
			anchor := ownedBuildingHex(t, state, "p0")
			var target board.Hex
			found := false
			for _, hex := range sortedHexes(state) {
				space := state.Map.GetHex(hex)
				if space.Building == nil && space.Terrain != models.TerrainRiver && state.IsAdjacentToPlayerBuilding(hex, "p0") {
					target, found = hex, true
					break
				}
			}
			if !found {
				t.Fatal("fixture needs reachable empty land")
			}
			state.Map.GetHex(target).Terrain = models.TerrainSwamp
			state.GetPlayer("p0").Resources.Workers = 0
			if order == "spade-build-town" {
				state.GetPlayer("p0").Resources.Workers = 1
			}
			state.PendingTownFormations["p0"] = []*game.PendingTownFormation{{PlayerID: "p0", Hexes: []board.Hex{anchor}}}
			state.PendingHalflingsSpades = &game.PendingHalflingsSpades{PlayerID: "p0", SpadesRemaining: 3}
			position, err := NewPosition(state)
			if err != nil {
				t.Fatal(err)
			}
			town := SearchAction{Kind: game.ActionSelectTownTile, TownTile: models.TownTile7Points, Hexes: []board.Hex{anchor}}
			spade := SearchAction{Kind: game.ActionApplyHalflingsSpade, Terrain: models.TerrainPlains, Hexes: []board.Hex{target}}
			build := SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{target}}
			if !hasAction(position.LegalActions(), town) || !hasAction(position.LegalActions(), spade) {
				t.Fatal("simultaneous reward choices missing from legal action set")
			}
			sequence := []SearchAction{town, spade, build}
			if order == "spade-town-build" {
				sequence = []SearchAction{spade, town, build}
			}
			if order == "spade-build-town" {
				sequence = []SearchAction{spade, build, town}
			}
			for _, action := range sequence {
				if !hasAction(position.LegalActions(), action) {
					t.Fatalf("missing choice %s", action.Key())
				}
				if err := position.Apply(action); err != nil {
					t.Fatal(err)
				}
				// A dwelling may offer opponent leech before the builder can
				// resume either reward resolution or the after-action window.
				for position.state.GetNextBlockingLeechResponder() != "" {
					if err := position.Apply(SearchAction{Kind: game.ActionDeclinePowerLeech}); err != nil {
						t.Fatal(err)
					}
				}
			}
			if position.state.Map.GetHex(target).Building == nil {
				t.Fatal("dwelling not built")
			}
			if position.state.PendingHalflingsSpades != nil {
				t.Fatal("unused spades not forfeited")
			}
			if !hasAction(position.LegalActions(), SearchAction{Kind: game.ActionFinishTurn}) {
				t.Fatal("missing post-action boundary")
			}
		})
	}
}

func TestHalflingsDwellingCannotRefoundPendingTown(t *testing.T) {
	state := forcedActionPosition(t, 9512, models.FactionHalflings, models.FactionWitches).StateClone()
	for _, space := range state.Map.Hexes {
		space.Building = nil
	}
	types := []models.BuildingType{models.BuildingStronghold, models.BuildingTradingHouse, models.BuildingDwelling, models.BuildingDwelling}
	powers := []int{3, 2, 1, 1}
	for i, kind := range types {
		hex := board.NewHex(i, 0)
		space := state.Map.GetHex(hex)
		if space == nil {
			t.Fatal("fixture hex missing")
		}
		space.Terrain = models.TerrainPlains
		space.Building = &models.Building{Type: kind, Faction: models.FactionHalflings, PlayerID: "p0", PowerValue: powers[i]}
	}
	anchor, target := board.NewHex(0, 0), board.NewHex(4, 0)
	state.Map.GetHex(target).Terrain = models.TerrainSwamp
	state.CheckForTownFormation("p0", anchor)
	if len(state.PendingTownFormations["p0"]) != 1 {
		t.Fatal("fixture must found exactly one town")
	}
	state.PendingHalflingsSpades = &game.PendingHalflingsSpades{PlayerID: "p0", SpadesRemaining: 3}
	position, err := NewPosition(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []SearchAction{
		{Kind: game.ActionApplyHalflingsSpade, Terrain: models.TerrainPlains, Hexes: []board.Hex{target}},
		{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{target}},
	} {
		if err := position.Apply(action); err != nil {
			t.Fatal(err)
		}
	}
	if len(position.state.PendingTownFormations["p0"]) != 1 {
		t.Fatal("growing a founded but unrewarded town created another reward/key")
	}
	if err := position.Apply(SearchAction{Kind: game.ActionSelectTownTile, TownTile: models.TownTile7Points, Hexes: []board.Hex{anchor}}); err != nil {
		t.Fatal(err)
	}
	if len(position.state.PendingTownFormations["p0"]) != 0 {
		t.Fatal("extra town reward remains")
	}
}
