package actions_test

import (
	"testing"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestLegalActionsAreExecutableOnScenario(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	if len(legal) == 0 {
		t.Fatal("expected legal actions")
	}
	for _, option := range legal {
		if _, err := actions.ApplyToClone(position.State, option.Action); err != nil {
			t.Fatalf("legal action %s did not apply: %v", option.ID, err)
		}
	}
}

func TestLegalActionsExcludeMainTurnTransformOnly(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	hasTransformBuild := false
	for _, option := range legal {
		if option.Type == "transform" {
			t.Fatalf("main-turn transform-only action should be pruned from AZ surface: %s", option.ID)
		}
		if option.Type == "transform_build" {
			hasTransformBuild = true
		}
	}
	if !hasTransformBuild {
		t.Fatal("expected transform/build actions to remain legal")
	}
}

func TestLegalActionsIncludeExecutablePass(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	for _, option := range legal {
		if option.Type != "pass" {
			continue
		}
		if _, err := actions.ApplyToClone(position.State, option.Action); err != nil {
			t.Fatalf("pass action %s did not apply: %v", option.ID, err)
		}
		return
	}
	t.Fatal("expected at least one legal pass action")
}

func TestLegalActionsExcludeDominatedNonPowerConversions(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	for _, option := range legal {
		if option.Type == "burn" {
			t.Fatalf("burn action should be pruned from AZ surface: %s", option.ID)
		}
		conversion, ok := option.Action.(*game.ConversionAction)
		if !ok {
			continue
		}
		switch conversion.ConversionType {
		case game.ConversionPriestToWorker, game.ConversionWorkerToCoin, game.ConversionAlchCoinToVP:
			t.Fatalf("dominated conversion action should be pruned from AZ surface: %s", option.ID)
		}
	}
}

func TestLegalActionsKeepPowerConversionWhenItCreatesLeechCapacity(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0
	player := gs.GetPlayer("p1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	player.Resources.Power = game.NewPowerSystem(0, 0, 6)

	for _, option := range actions.LegalActions(gs) {
		conversion, ok := option.Action.(*game.ConversionAction)
		if ok && conversion.ConversionType == game.ConversionPowerToCoin {
			return
		}
	}
	t.Fatal("expected power-to-coin conversion to remain when it creates leech capacity")
}

func TestLegalActionsAllowPowerConversionBeforeAffordableTradingPostUpgrade(t *testing.T) {
	gs := game.NewGameState()
	faction := factions.NewWitches()
	if err := gs.AddPlayer("p1", faction); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0

	player := gs.GetPlayer("p1")
	player.Resources.Coins = 10
	player.Resources.Workers = 4
	player.Resources.Power = game.NewPowerSystem(0, 0, 6)

	hex := board.NewHex(0, 0)
	if err := gs.Map.TransformTerrain(hex, faction.GetHomeTerrain()); err != nil {
		t.Fatalf("TransformTerrain failed: %v", err)
	}
	gs.Map.PlaceBuilding(hex, &models.Building{
		Type:       models.BuildingDwelling,
		PlayerID:   "p1",
		Faction:    faction.GetType(),
		PowerValue: 1,
	})

	var conversion actions.Option
	for _, option := range actions.LegalActions(gs) {
		if action, ok := option.Action.(*game.ConversionAction); ok && action.ConversionType == game.ConversionPowerToCoin && action.Amount == 1 {
			conversion = option
			break
		}
	}
	if conversion.Action == nil {
		t.Fatal("expected power-to-coin conversion to remain before affordable TP upgrade")
	}

	afterConversion, err := actions.ApplyToClone(gs, conversion.Action)
	if err != nil {
		t.Fatalf("ApplyToClone conversion failed: %v", err)
	}
	player = afterConversion.GetPlayer("p1")
	if got := player.Resources.Power.Bowl1; got != 1 {
		t.Fatalf("bowl I after conversion = %d, want 1", got)
	}
	if got := player.Resources.Power.Bowl3; got != 5 {
		t.Fatalf("bowl III after conversion = %d, want 5", got)
	}

	for _, option := range actions.LegalActions(afterConversion) {
		if action, ok := option.Action.(*game.UpgradeBuildingAction); ok && action.NewBuildingType == models.BuildingTradingHouse {
			if _, err := actions.ApplyToClone(afterConversion, option.Action); err != nil {
				t.Fatalf("TP upgrade after power conversion did not apply: %v", err)
			}
			return
		}
	}
	t.Fatal("expected affordable TP upgrade to remain legal after leech-capacity power conversion")
}

func TestLegalActionsExcludePowerConversionWhenItCannotCreateLeechCapacity(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0
	player := gs.GetPlayer("p1")
	player.Resources.Coins = 10
	player.Resources.Workers = 10
	player.Resources.Power = game.NewPowerSystem(0, 6, 0)

	for _, option := range actions.LegalActions(gs) {
		conversion, ok := option.Action.(*game.ConversionAction)
		if ok && conversion.ConversionType == game.ConversionPowerToCoin {
			t.Fatalf("power-to-coin should be pruned without Bowl III power to move into Bowl I: %s", option.ID)
		}
	}
}

func TestLegalActionsExcludeAlchemistsCoinToVPConversion(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewAlchemists()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0
	player := gs.GetPlayer("p1")
	player.Resources.Coins = 8

	for _, option := range actions.LegalActions(gs) {
		if conversion, ok := option.Action.(*game.ConversionAction); ok && conversion.ConversionType == game.ConversionAlchCoinToVP {
			t.Fatalf("Alchemists coin-to-VP conversion should be pruned from AZ surface: %s", option.ID)
		}
	}
}

func TestLegalActionsKeepEnlightenedCoinToPowerConversion(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewTheEnlightened()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0
	player := gs.GetPlayer("p1")
	player.Resources.Coins = 2

	for _, option := range actions.LegalActions(gs) {
		if option.Type == "conversion" {
			if conversion, ok := option.Action.(*game.ConversionAction); ok && conversion.ConversionType == game.ConversionCoinToPower {
				return
			}
		}
	}
	t.Fatal("expected Enlightened coin-to-power conversion to remain on AZ surface")
}

func TestLegalActionsExcludeMainTurnActionsForPassedPlayer(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	current := position.State.GetCurrentPlayer()
	if current == nil {
		t.Fatal("expected current player")
	}
	current.HasPassed = true
	legal := actions.LegalActions(position.State)
	for _, option := range legal {
		if option.PlayerID == current.ID {
			t.Fatalf("passed current player should not receive main-turn action: %s", option.ID)
		}
	}
}

func TestLegalActionsExcludeRepeatedMermaidsRiverTownConnect(t *testing.T) {
	gs := game.NewGameState()
	faction := factions.NewMermaids()
	if err := gs.AddPlayer("p1", faction); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"p1"}
	gs.CurrentPlayerIndex = 0

	river := board.NewHex(0, 0)
	hexes := []board.Hex{
		board.NewHex(1, 0),
		board.NewHex(2, 0),
		board.NewHex(-1, 0),
		board.NewHex(-2, 0),
	}
	for _, hex := range append([]board.Hex{river}, hexes...) {
		if gs.Map.GetHex(hex) == nil {
			gs.Map.Hexes[hex] = &board.MapHex{Coord: hex}
		}
	}
	gs.Map.GetHex(river).Terrain = models.TerrainRiver
	gs.Map.RiverHexes[river] = true
	for _, hex := range hexes {
		gs.Map.GetHex(hex).Terrain = faction.GetHomeTerrain()
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "p1",
			PowerValue: 2,
		})
	}

	if err := game.NewMermaidsRiverTownAction("p1", river).Execute(gs); err != nil {
		t.Fatalf("first river town connect failed: %v", err)
	}
	for _, option := range actions.LegalActions(gs) {
		if option.Type == "special_mermaids_town" {
			t.Fatalf("repeated Mermaids river town connect should not be legal: %s", option.ID)
		}
	}
}
