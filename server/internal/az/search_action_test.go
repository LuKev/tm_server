package az

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

func TestSearchActionSchemaCoversEveryBaseBranch(t *testing.T) {
	h1, h2 := board.Hex{Q: 1, R: 2}, board.Hex{Q: 3, R: 4}
	actions := []SearchAction{
		{Kind: game.ActionTransformAndBuild, Hexes: []board.Hex{h1}, Terrain: models.TerrainForest, Build: true, UseSkip: true},
		{Kind: game.ActionUpgradeBuilding, Hexes: []board.Hex{h1}, Building: models.BuildingTemple},
		{Kind: game.ActionAdvanceShipping}, {Kind: game.ActionAdvanceDigging},
		{Kind: game.ActionSendPriestToCult, Track: game.CultAir, Amount: 2},
		{Kind: game.ActionPowerAction, Power: game.PowerActionWorkers},
		{Kind: game.ActionPowerAction, Power: game.PowerActionSpade1, Hexes: []board.Hex{h1}, Build: true, UseSkip: true},
		{Kind: game.ActionPowerAction, Power: game.PowerActionBridge, Hexes: []board.Hex{h1, h2}},
		{Kind: game.ActionPass, Card: game.BonusCardUnknown},
		{Kind: game.ActionPass, Card: game.BonusCardPriest},
		{Kind: game.ActionSetupDwelling, Hexes: []board.Hex{h1}},
		{Kind: game.ActionUseCultSpade, Hexes: []board.Hex{h1}, Terrain: models.TerrainSwamp},
		{Kind: game.ActionAcceptPowerLeech, Amount: 1, Amount2: 2},
		{Kind: game.ActionDeclinePowerLeech, Amount: 1},
		{Kind: game.ActionSelectFavorTile, FavorTile: game.FavorAir1},
		{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{h1}, Terrain: models.TerrainPlains},
		{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{h1}},
		{Kind: game.ActionSkipHalflingsDwelling},
		{Kind: game.ActionUseDarklingsPriestOrdination, Amount: 2},
		{Kind: game.ActionSelectCultistsCultTrack, Track: game.CultWater},
		{Kind: game.ActionSelectFaction, Faction: models.FactionNomads},
		{Kind: game.ActionSetupBonusCard, Card: game.BonusCardSpade},
		{Kind: game.ActionSelectTownTile, TownTile: models.TownTile7Points, Hexes: []board.Hex{h1}},
		{Kind: game.ActionSelectTownCultTop, Tracks: []game.CultTrack{game.CultFire, game.CultEarth}},
		{Kind: game.ActionDiscardPendingSpade, Amount: 2},
		{Kind: game.ActionConversion, Conversion: game.ConversionWorkerToCoin, Amount: 3},
		{Kind: game.ActionBurnPower, Amount: 2},
		{Kind: game.ActionEngineersBridge, Hexes: []board.Hex{h1, h2}},
	}
	specials := []SearchAction{
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionAurenCultAdvance, Track: game.CultFire},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionWitchesRide, Hexes: []board.Hex{h1}},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionSwarmlingsUpgrade, Hexes: []board.Hex{h1}},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionChaosMagiciansDoubleTurn},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionGiantsTransform, Hexes: []board.Hex{h1}, Build: true},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionNomadsSandstorm, Hexes: []board.Hex{h1}, Build: true},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionWater2CultAdvance, Track: game.CultWater},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionBonusCardSpade, Hexes: []board.Hex{h1}, Terrain: models.TerrainLake, Build: true, UseSkip: true},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionBonusCardCultAdvance, Track: game.CultEarth},
		{Kind: game.ActionSpecialAction, Special: game.SpecialActionMermaidsRiverTown, Hexes: []board.Hex{h1}},
	}
	actions = append(actions, specials...)

	kinds := make(map[game.ActionType]bool)
	specialKinds := make(map[game.SpecialActionType]bool)
	for _, action := range actions {
		engineAction, err := action.GameAction("actor")
		if err != nil {
			t.Fatalf("construct %s: %v", action.Key(), err)
		}
		roundTrip, err := SearchActionFromGameAction(engineAction)
		if err != nil {
			t.Fatalf("round trip %s: %v", action.Key(), err)
		}
		if roundTrip.Key() != action.Key() {
			t.Fatalf("round trip mismatch\ngot=%s\nwant=%s", roundTrip.Key(), action.Key())
		}
		kinds[action.Kind] = true
		if action.Kind == game.ActionSpecialAction {
			specialKinds[action.Special] = true
		}
	}

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
		if !kinds[kind] {
			t.Fatalf("missing action kind %v", kind)
		}
	}
	expectedSpecials := []game.SpecialActionType{
		game.SpecialActionAurenCultAdvance, game.SpecialActionWitchesRide,
		game.SpecialActionSwarmlingsUpgrade, game.SpecialActionChaosMagiciansDoubleTurn,
		game.SpecialActionGiantsTransform, game.SpecialActionNomadsSandstorm,
		game.SpecialActionWater2CultAdvance, game.SpecialActionBonusCardSpade,
		game.SpecialActionBonusCardCultAdvance, game.SpecialActionMermaidsRiverTown,
	}
	for _, special := range expectedSpecials {
		if !specialKinds[special] {
			t.Fatalf("missing special action subtype %v", special)
		}
	}
}
