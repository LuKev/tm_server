package notation

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

func TestBGAParser(t *testing.T) {
	// Inline log content for test
	content := `Game board: Base Game
Mini-expansions: On
deragned selected the faction Halflings on plains to play in position #1
kezilu selected the faction Auren on forest to play in position #2
Redrame selected the faction Chaos Magicians on wasteland to play in position #3
Locky_91 selected the faction Alchemists on swamp to play in position #4
deragned is playing the Halflings Faction (with 32 VP Starting VPs)
Redrame is playing the Auren Faction (with 40 VP Starting VPs)
kezilu is playing the Chaos Magicians Faction (with 40 VP Starting VPs)
Locky_91 is playing the Alchemists Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
deragned places a Dwelling [F5]
Redrame places a Dwelling [F4]
Locky_91 places a Dwelling [E5]
Locky_91 places a Dwelling [B5]
Redrame places a Dwelling [C3]
deragned places a Dwelling [E6]
kezilu places a Dwelling [D4]
~ Action phase ~
`

	parser := NewBGAParser(string(content))
	actions, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	if len(actions) == 0 {
		t.Errorf("Expected actions, got 0")
	}

	// Print first few actions for inspection
	t.Logf("Parsed %d actions", len(actions))
	for i, action := range actions {
		if i >= 20 {
			break
		}
		t.Logf("Action %d: %T %+v", i, action, action)
	}

	// Verify specific known actions
	// Move 54: felipebart places a Dwelling [F3] -> SetupDwellingAction
	foundSetup := false
	for _, item := range actions {
		if actionItem, ok := item.(ActionItem); ok {
			if _, ok := actionItem.Action.(*game.SetupDwellingAction); ok {
				foundSetup = true
				break
			}
		}
	}
	if !foundSetup {
		t.Errorf("Did not find SetupDwellingAction")
	}

	// Check Round 1 TurnOrder
	foundRound1 := false
	for _, item := range actions {
		if rs, ok := item.(RoundStartItem); ok && rs.Round == 1 {
			foundRound1 = true
			t.Logf("Round 1 TurnOrder: %v", rs.TurnOrder)
			// Check for duplicates
			seen := make(map[string]bool)
			for _, p := range rs.TurnOrder {
				if seen[p] {
					t.Errorf("Duplicate player in TurnOrder: %s", p)
				}
				seen[p] = true
			}
			if len(rs.TurnOrder) != 4 {
				t.Errorf("Expected 4 players in TurnOrder, got %d", len(rs.TurnOrder))
			}
		}
	}
	if !foundRound1 {
		t.Errorf("Did not find Round 1 Start")
	}
}

func TestBGAParser_SkipsFireIceSetupTransformRows(t *testing.T) {
	content := `Game board: Revised Base Game
Mini-expansions: On
tanu_schka selected the faction Selkies on forest to play in position #1
tanu_schka is playing the Selkies Faction (with 38 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
tanu_schka transforms a Terrain space forest → ice [E7]
tanu_schka places a Dwelling [E7]
~ Action phase ~
`

	parser := NewBGAParser(content)
	actions, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	foundSetup := false
	for _, item := range actions {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		switch actionItem.Action.(type) {
		case *game.TransformAndBuildAction:
			t.Fatalf("setup transform row was parsed as a transform action: %#v", actionItem.Action)
		case *game.SetupDwellingAction:
			foundSetup = true
		}
	}
	if !foundSetup {
		t.Fatalf("expected setup dwelling action")
	}
}

func TestBGAParser_RiverwalkersUnlockRows(t *testing.T) {
	content := `Game board: Revised Base Game
Mini-expansions: On
octo86 selected the faction Riverwalkers on lakes to play in position #1
octo86 is playing the Riverwalkers Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
octo86 places a Dwelling [E4]
~ Income phase ~
octo86 spends 1 coins to unlock 1 Priests from mountains Terrain cycle (Riverwalkers ability) (Income)
~ Action phase ~
octo86 spends 3 power + 1 coins to unlock 1 Priests from wasteland Terrain cycle (Riverwalkers ability) (Power action)
`

	parser := NewBGAParser(content)
	actions, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	var incomeUnlock *LogRiverwalkersUnlockAction
	var powerUnlock *LogRiverwalkersUnlockAction
	for _, item := range actions {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		if postIncome, ok := actionItem.Action.(*LogPostIncomeAction); ok {
			incomeUnlock, _ = postIncome.Action.(*LogRiverwalkersUnlockAction)
			continue
		}
		if unlock, ok := actionItem.Action.(*LogRiverwalkersUnlockAction); ok {
			powerUnlock = unlock
		}
	}

	if incomeUnlock == nil {
		t.Fatalf("expected income Riverwalkers unlock action")
	}
	if incomeUnlock.Terrain != models.TerrainMountain || incomeUnlock.CoinCost != 1 || incomeUnlock.PowerCost != 0 {
		t.Fatalf("income unlock = %+v, want mountain/1C/0PW", incomeUnlock)
	}
	if powerUnlock == nil {
		t.Fatalf("expected power-action Riverwalkers unlock action")
	}
	if powerUnlock.Terrain != models.TerrainWasteland || powerUnlock.CoinCost != 1 || powerUnlock.PowerCost != 3 {
		t.Fatalf("power unlock = %+v, want wasteland/1C/3PW", powerUnlock)
	}
}

func TestBGAParser_RiverwalkersStrongholdBridgeRows(t *testing.T) {
	content := `Game board: Revised Base Game
Mini-expansions: On
octo86 selected the faction Riverwalkers on lakes to play in position #1
octo86 is playing the Riverwalkers Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
octo86 places a Dwelling [E4]
~ Action phase ~
octo86 build a Bridge (Riverwalkers Stronghold) [E4-G1]
`

	parser := NewBGAParser(content)
	actions, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	for _, item := range actions {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		if bridge, ok := actionItem.Action.(*LogFreeBridgeAction); ok {
			if bridge.PlayerID != "Riverwalkers" {
				t.Fatalf("bridge player = %s, want Riverwalkers", bridge.PlayerID)
			}
			return
		}
	}
	t.Fatalf("expected free bridge action")
}

func TestBGAParser_UsesGameBoardForCoordinates(t *testing.T) {
	content := `Game board: Lakes
Mini-expansions: On
Alice is playing the Auren Faction (with 40 VP Starting VPs)
Bob is playing the Witches Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
Alice places a Dwelling [B1]
Bob places a Dwelling [B2]
~ Income phase ~`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected, ok := board.HexForDisplayCoordinate(board.MapLakes, "B1")
	if !ok {
		t.Fatal("missing Lakes coordinate B1")
	}
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		setup, ok := actionItem.Action.(*game.SetupDwellingAction)
		if ok && setup.PlayerID == "Auren" {
			if setup.Hex != expected {
				t.Fatalf("setup hex = %v, want Lakes B1 %v", setup.Hex, expected)
			}
			return
		}
	}
	t.Fatalf("expected Auren setup dwelling in parsed items: %#v", items)
}

func TestBGAParserSettings(t *testing.T) {
	content := `Game board: Base Game
Round 1 scoring: SCORE2, TOWN >> 5
Round 2 scoring: SCORE9, TE >> 4
Round 3 scoring: SCORE4, SH/SA >> 5
Round 4 scoring: SCORE1, SPADE >> 2
Round 5 scoring: SCORE6, TP >> 3
Round 6 scoring: SCORE7, SH/SA >> 5
Removing tile BON9
Removing tile BON2
Removing tile BON5
deragned is playing the Halflings Faction
~ Every player has chosen a Faction ~
`
	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var settings *GameSettingsItem
	for _, item := range items {
		if s, ok := item.(GameSettingsItem); ok {
			settings = &s
			break
		}
	}

	if settings == nil {
		t.Fatal("GameSettingsItem not found")
	}

	// Check ScoringTiles
	expectedTiles := "SCORE2,SCORE9,SCORE4,SCORE1,SCORE6,SCORE7"
	if settings.Settings["ScoringTiles"] != expectedTiles {
		t.Errorf("Expected ScoringTiles %q, got %q", expectedTiles, settings.Settings["ScoringTiles"])
	}

	// Check BonusCards
	// Removed: BON9 (BON-DW), BON2 (BON-4C), BON5 (BON-WP)
	// All: SPD, 4C, 6C, SHIP, WP, BB, TP, P, DW, SHIP-VP
	// Expected: SPD, 6C, SHIP, BB, TP, P, SHIP-VP
	// (Order depends on allBonusCodes order in parser)
	// allBonusCodes: SPD, 4C, 6C, SHIP, WP, BB, TP, P, DW, SHIP-VP
	// Removed: 4C, WP, DW
	// Remaining: SPD, 6C, SHIP, BB, TP, P, SHIP-VP
	expectedBonus := "BON-SPD,BON-6C,BON-SHIP,BON-BB,BON-TP,BON-P,BON-SHIP-VP"
	if settings.Settings["BonusCards"] != expectedBonus {
		t.Errorf("Expected BonusCards %q, got %q", expectedBonus, settings.Settings["BonusCards"])
	}
}

func TestBGAParserSettings_PreservesFactionStartingTerrain(t *testing.T) {
	content := `Game board: Base Game
Barnawal selected the faction Ice Maidens on mountains to play in position #1
Zoras selected the faction Nomads on desert to play in position #2
mellison is playing the Ice Maidens Faction (with 27 VP Starting VPs)
Zoras is playing the Nomads Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var settings *GameSettingsItem
	for _, item := range items {
		if s, ok := item.(GameSettingsItem); ok {
			settings = &s
			break
		}
	}

	if settings == nil {
		t.Fatal("GameSettingsItem not found")
	}
	if got := settings.Settings["StartingTerrain:Ice Maidens"]; got != "mountains" {
		t.Fatalf("StartingTerrain:Ice Maidens = %q, want %q", got, "mountains")
	}
	if got := settings.Settings["StartingTerrain:Nomads"]; got != "desert" {
		t.Fatalf("StartingTerrain:Nomads = %q, want %q", got, "desert")
	}
}

func TestBGAParser_FirewalkersMarkerToCoinDoesNotCreateResidualPower(t *testing.T) {
	content := "Game board: Base Game\n" +
		"snakeixirr selected the faction Firewalkers on volcano to play in position #1\n" +
		"snakeixirr is playing the Firewalkers Faction (with 39 VP Starting VPs)\n" +
		"~ Every player has chosen a Faction and receives the matching starting resources. ~\n" +
		"snakeixirr moves their VP marker by 2 VP forward to convert to 2 power \u2192 1 coins (Firewalkers Ability)\n"

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var markerToCoin *LogConversionAction
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		if _, ok := actionItem.Action.(*LogCompoundAction); ok {
			t.Fatalf("Firewalkers marker-to-coin should parse as a direct conversion, not a power compound")
		}
		conversion, ok := actionItem.Action.(*LogConversionAction)
		if !ok || conversion.PlayerID != "Firewalkers" {
			continue
		}
		if conversion.Cost[models.ResourceVictoryPoint] == 2 {
			markerToCoin = conversion
			break
		}
	}

	if markerToCoin == nil {
		t.Fatal("Firewalkers marker-to-coin conversion not found")
	}
	if got := markerToCoin.Reward[models.ResourceCoin]; got != 1 {
		t.Fatalf("coin reward = %d, want 1", got)
	}
	if got := markerToCoin.Reward[models.ResourcePower]; got != 0 {
		t.Fatalf("power reward = %d, want 0", got)
	}
	if got := markerToCoin.Cost[models.ResourcePower]; got != 0 {
		t.Fatalf("power cost = %d, want 0", got)
	}
}

func TestBGAParser_TransformToIceUsesIceTargetTerrain(t *testing.T) {
	content := `Game board: Base Game
mellison selected the faction Ice Maidens on mountains to play in position #1
mellison is playing the Ice Maidens Faction (with 27 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
mellison transforms a Terrain space forest → ice for 1 spade(s) [F4]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		action, ok := actionItem.Action.(*game.TransformAndBuildAction)
		if !ok {
			continue
		}
		if action.PlayerID != "Ice Maidens" {
			continue
		}
		if action.TargetTerrain != models.TerrainIce {
			t.Fatalf("target terrain = %v, want %v", action.TargetTerrain, models.TerrainIce)
		}
		return
	}

	t.Fatal("Did not find Ice Maidens transform action")
}

func TestBGAParser_SnowShamansPassUpgradeRowsAreChoices(t *testing.T) {
	content := `Game board: Fjords
bballrace selected the faction Snow Shamans on lakes to play in position #1
bballrace is playing the Snow Shamans Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
bballrace advances on the Exchange Track for free and earns 0 VP (Snow Shamans ability)
bballrace advances on the Shipping Track for free and earns 0 VP (Snow Shamans ability)
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var upgrades []game.SnowShamansPassUpgrade
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		action, ok := actionItem.Action.(*LogSnowShamansPassUpgradeAction)
		if !ok {
			if _, isPaidDigging := actionItem.Action.(*game.AdvanceDiggingAction); isPaidDigging {
				t.Fatalf("Snow Shamans exchange row parsed as paid digging action")
			}
			if _, isPaidShipping := actionItem.Action.(*game.AdvanceShippingAction); isPaidShipping {
				t.Fatalf("Snow Shamans shipping row parsed as paid shipping action")
			}
			continue
		}
		upgrades = append(upgrades, action.Upgrade)
	}

	if len(upgrades) != 2 {
		t.Fatalf("parsed Snow Shamans upgrades = %v, want 2 choices", upgrades)
	}
	if upgrades[0] != game.SnowShamansPassUpgradeDigging || upgrades[1] != game.SnowShamansPassUpgradeShipping {
		t.Fatalf("parsed upgrades = %v, want digging then shipping", upgrades)
	}
}

func TestBGAParser_ShapeshiftersLeechBonusRowsAreExplicitActions(t *testing.T) {
	content := `Game board: Fjords
mellison selected the faction Shapeshifters on forest to play in position #1
mellison is playing the Shapeshifters Faction (with 38 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
mellison pays 1 VP to gain 1 power in Bowl 3 (Shapeshifters ability)
mellison gets 1 power (Shapeshifters Ability)
mellison declines gaining a new power in Bowl 3 (Shapeshifters Ability)
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	var actions []*LogShapeshiftersLeechBonusAction
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		action, ok := actionItem.Action.(*LogShapeshiftersLeechBonusAction)
		if ok {
			actions = append(actions, action)
		}
	}

	if len(actions) != 3 {
		t.Fatalf("parsed Shapeshifters bonus actions = %d, want 3", len(actions))
	}
	if !actions[0].Paid || actions[0].Declined {
		t.Fatalf("first action = %+v, want paid", actions[0])
	}
	if actions[1].Paid || actions[1].Declined {
		t.Fatalf("second action = %+v, want free power", actions[1])
	}
	if !actions[2].Declined || actions[2].Paid {
		t.Fatalf("third action = %+v, want declined", actions[2])
	}
}

func TestBGAParser_ShapeshiftersStrongholdShiftParsesTerrainChange(t *testing.T) {
	content := `Game board: Base Game
Zoras selected the faction Shapeshifters on plains to play in position #1
Zoras is playing the Shapeshifters Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Zoras changes his Home Terrain to mountains for Power (Shapeshifters Stronghold)
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		action, ok := actionItem.Action.(*game.SpecialAction)
		if !ok {
			continue
		}
		if action.PlayerID != "Shapeshifters" {
			continue
		}
		if action.ActionType != game.SpecialActionShapeshiftersShiftTerrain {
			t.Fatalf("action type = %v, want %v", action.ActionType, game.SpecialActionShapeshiftersShiftTerrain)
		}
		if action.TargetTerrain == nil || *action.TargetTerrain != models.TerrainMountain {
			t.Fatalf("target terrain = %+v, want Mountain", action.TargetTerrain)
		}
		return
	}

	t.Fatal("Did not find Shapeshifters stronghold terrain-shift action")
}

func TestBGAParser_FjordsSetupCoordinates(t *testing.T) {
	content := `Game board: Fjords
haligh selected the faction Witches on forest to play in position #1
haligh is playing the Witches Faction (with 24 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
haligh places a Dwelling [F5]
~ Action phase ~
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	expectedHex, ok := board.HexForDisplayCoordinate(board.MapFjords, "F5")
	if !ok {
		t.Fatal("expected Fjords F5 coordinate to exist")
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		action, ok := actionItem.Action.(*game.SetupDwellingAction)
		if !ok {
			continue
		}
		if action.Hex != expectedHex {
			t.Fatalf("setup dwelling hex = %+v, want %+v", action.Hex, expectedHex)
		}
		return
	}

	t.Fatal("Did not find SetupDwellingAction")
}

func TestBGAParser_FjordsRiverDwellingCoordinate(t *testing.T) {
	content := `Game board: Fjords
kezilu selected the faction Selkies on plains to play in position #1
kezilu is playing the Selkies Faction (with 40 VP Starting VPs)
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
kezilu builds a Dwelling for 1 workers 2 coins + 1 workers (Selkies Ability) [R~C4]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok {
			continue
		}
		action, ok := actionItem.Action.(*LogRiverBuildAction)
		if !ok {
			continue
		}
		if action.PlayerID != "Selkies" {
			continue
		}
		if action.CoordToken != "R~C4" {
			t.Fatalf("river dwelling token = %q, want %q", action.CoordToken, "R~C4")
		}
		return
	}

	t.Fatal("Did not find Selkies LogRiverBuildAction")
}

func TestConvertRiverCoordToAxialForMap_FjordsUsesLandDisplayReference(t *testing.T) {
	hex, err := ConvertRiverCoordToAxialForMap(board.MapFjords, "R~C4")
	if err != nil {
		t.Fatalf("ConvertRiverCoordToAxialForMap failed: %v", err)
	}
	if want := board.NewHex(5, 2); hex != want {
		t.Fatalf("river hex = %+v, want %+v", hex, want)
	}
}

func TestBGAParser_ParsesChashIncomeTrackAdvancement(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Chash Dallah Faction
Bob is playing the Halflings Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice advances on the Income Track for 2 workers 2 coins and earns 1 VP
Bob passes
Alice advances on the Income Track for 2 workers 2 coins and earns 2 VP
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	trackActions := 0
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		if _, ok := actionItem.Action.(*game.AdvanceChashTrackAction); ok {
			trackActions++
		}
	}

	if trackActions != 2 {
		t.Fatalf("AdvanceChashTrackAction count = %d, want 2", trackActions)
	}
}

func TestBGAParser_ParsesTreasurersSafeDepositsWithoutSkippingPhaseSemantics(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Treasurers Faction
Bob is playing the Engineers Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
Alice places a Dwelling [D4]
Bob places a Dwelling [E5]
~ Action phase ~
Alice passes
Bob passes
~ Cleanup phase ~
Alice places 2 workers into their Safe (Income)
~ Income phase ~
Alice places 1 workers 1 Priests 3 coins into their Safe (Income)
Alice places 7 coins into their Safe (Power action)
~ Action phase ~
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	deposits := make([]game.Action, 0, 3)
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		switch actionItem.Action.(type) {
		case *game.SelectTreasurersDepositAction, *LogPostIncomeAction:
			deposits = append(deposits, actionItem.Action)
		}
	}

	if len(deposits) != 3 {
		t.Fatalf("deposit action count = %d, want 3", len(deposits))
	}

	if _, ok := deposits[0].(*game.SelectTreasurersDepositAction); !ok {
		t.Fatalf("cleanup-phase deposit type = %T, want *game.SelectTreasurersDepositAction", deposits[0])
	}

	postIncome, ok := deposits[1].(*LogPostIncomeAction)
	if !ok {
		t.Fatalf("income-phase deposit type = %T, want *LogPostIncomeAction", deposits[1])
	}
	inner, ok := postIncome.Action.(*game.SelectTreasurersDepositAction)
	if !ok {
		t.Fatalf("wrapped income-phase deposit type = %T, want *game.SelectTreasurersDepositAction", postIncome.Action)
	}
	if inner.CoinsToTreasury != 3 || inner.WorkersToTreasury != 1 || inner.PriestsToTreasury != 1 {
		t.Fatalf("wrapped income-phase deposit = %+v, want coins=3 workers=1 priests=1", inner)
	}

	actionDeposit, ok := deposits[2].(*game.SelectTreasurersDepositAction)
	if !ok {
		t.Fatalf("action-phase deposit type = %T, want *game.SelectTreasurersDepositAction", deposits[2])
	}
	if actionDeposit.CoinsToTreasury != 7 || actionDeposit.WorkersToTreasury != 0 || actionDeposit.PriestsToTreasury != 0 {
		t.Fatalf("action-phase deposit = %+v, want coins=7", actionDeposit)
	}
}

func TestBGAParser_ParsesTreasurersActionPhaseSafeDepositsFromRealSnippet(t *testing.T) {
	content := `Game board: Base Game
Zaarito is playing the Treasurers Faction
marszej76 is playing the Nomads Faction
philvec is playing the Wisps Faction
zjwlanlan is playing the Dynion Geifr Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Zaarito cancels their move
Zaarito transforms a Terrain space mountains → wasteland for 1 spade(s) (Bonus card action) [F1]
Zaarito builds a Dwelling for 1 workers 2 coins [F1]
Zaarito gets 2 VP (Scoring tile bonus)
Zaarito founds a Town [F1]
Zaarito gets 5 VP (Town bonus)
Zaarito places 6 coins into their Safe (Town bonus)
Zaarito declines doing Conversions
marszej76 builds a Dwelling for 1 workers 2 coins [B4]
marszej76 gets 2 VP (Scoring tile bonus)
marszej76 gets 2 VP (Favor tile bonus)
marszej76 declines doing Conversions
philvec builds a Dwelling for 2 workers 2 coins [G1]
philvec gets 2 VP (Scoring tile bonus)
philvec gets 2 VP (Favor tile bonus)
philvec declines doing Conversions
zjwlanlan builds a Dwelling for 1 workers 2 coins [E4]
zjwlanlan gets 2 VP (Scoring tile bonus)
zjwlanlan gets 2 VP (Favor tile bonus)
zjwlanlan declines doing Conversions
Zaarito gets 1 power via Structures [E4]
Zaarito Power gain via Structures is capped from 3 power to 1 power
marszej76 pays 1 VP and gets 2 power via Structures [E4]
Zaarito places 2 workers into their Safe (Power action)
Zaarito declines doing Conversions
marszej76 builds a Dwelling for 1 workers 2 coins [G4]
marszej76 gets 2 VP (Scoring tile bonus)
marszej76 gets 2 VP (Favor tile bonus)
marszej76 declines doing Conversions
philvec does some Conversions (spent: 0 power 1 Priests 0 workers ; collects: 0 Priests  2 workers 2 coins)
philvec builds a Dwelling for 2 workers 2 coins [D7]
philvec gets 2 VP (Scoring tile bonus)
philvec gets 2 VP (Favor tile bonus)
philvec founds a Town [E9]
philvec gets 9 VP and collects 1 Priests (Town bonus)
philvec declines doing Conversions
zjwlanlan builds a Dwelling for 1 workers 2 coins [H2]
zjwlanlan gets 2 VP (Scoring tile bonus)
zjwlanlan gets 2 VP (Favor tile bonus)
zjwlanlan declines doing Conversions
Zaarito does some Conversions (spent: 1 power 1 Priests 0 workers ; collects: 0 Priests  1 workers 1 coins)
Zaarito places 1 workers 1 coins into their Safe (Conversion)
Zaarito passes and becomes the first player for the next round
Zaarito chooses 1 Bonus card
Zaarito places 1 coins into their Safe (Bonus card)
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	var deposits []*game.SelectTreasurersDepositAction
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		deposit, ok := actionItem.Action.(*game.SelectTreasurersDepositAction)
		if !ok {
			continue
		}
		if deposit.PlayerID != "Treasurers" {
			continue
		}
		deposits = append(deposits, deposit)
	}

	if len(deposits) != 4 {
		t.Fatalf("Treasurers action-phase deposit count = %d, want 4", len(deposits))
	}
	if deposits[0].CoinsToTreasury != 6 || deposits[0].WorkersToTreasury != 0 {
		t.Fatalf("town-bonus deposit = %+v, want 6 coins", deposits[0])
	}
	if deposits[1].CoinsToTreasury != 0 || deposits[1].WorkersToTreasury != 2 {
		t.Fatalf("power-action deposit = %+v, want 2 workers", deposits[1])
	}
	if deposits[2].CoinsToTreasury != 1 || deposits[2].WorkersToTreasury != 1 {
		t.Fatalf("conversion deposit = %+v, want 1 coin + 1 worker", deposits[2])
	}
	if deposits[3].CoinsToTreasury != 1 || deposits[3].WorkersToTreasury != 0 {
		t.Fatalf("bonus-card deposit = %+v, want 1 coin", deposits[3])
	}
}

func TestBGAParser_SynthesizesMissingPowerActionBeforeTreasurersSafeDeposit(t *testing.T) {
	content := `Game board: Base Game
Zaarito is playing the Treasurers Faction
marszej76 is playing the Nomads Faction
philvec is playing the Wisps Faction
zjwlanlan is playing the Dynion Geifr Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Zaarito advances on the Shipping track for 1 Priests 4 coins and gets 2 VP
Zaarito declines doing Conversions
zjwlanlan builds a Dwelling for 1 workers 2 coins [E4]
Zaarito gets 1 power via Structures [E4]
Zaarito Power gain via Structures is capped from 3 power to 1 power
marszej76 pays 1 VP and gets 2 power via Structures [E4]
Zaarito places 2 workers into their Safe (Power action)
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	foundPower := false
	foundDeposit := false
	for i, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		if powerAction, ok := actionItem.Action.(*LogPowerAction); ok && powerAction.PlayerID == "Treasurers" {
			foundPower = true
			if powerAction.ActionCode != "ACT3" {
				t.Fatalf("synthetic power action code = %q, want ACT3", powerAction.ActionCode)
			}
			if i+1 >= len(items) {
				t.Fatalf("expected deposit action after synthetic power action")
			}
			nextItem, ok := items[i+1].(ActionItem)
			if !ok {
				t.Fatalf("item after synthetic power action = %T, want ActionItem", items[i+1])
			}
			deposit, ok := nextItem.Action.(*game.SelectTreasurersDepositAction)
			if !ok {
				t.Fatalf("item after synthetic power action = %T, want *game.SelectTreasurersDepositAction", nextItem.Action)
			}
			if deposit.WorkersToTreasury != 2 || deposit.CoinsToTreasury != 0 || deposit.PriestsToTreasury != 0 {
				t.Fatalf("synthetic power-action deposit = %+v, want 2 workers", deposit)
			}
			foundDeposit = true
			break
		}
	}

	if !foundPower || !foundDeposit {
		t.Fatalf("expected synthetic power action and deposit, foundPower=%t foundDeposit=%t", foundPower, foundDeposit)
	}
}

func TestBGAParser_ParsesFinalScoringBlock(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Witches Faction
Bob is playing the Giants Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice passes
Bob passes
~ Final scoring ~
Alice scores 8 VP (Cult of Fire)
Bob scores 4 VP (Cult of Fire)
Alice scores 18 VP with 15 connected Structures (Area scoring)
Bob scores 6 VP with 14 connected Structures (Area scoring)
Alice converts 2 workers into 2 coins
Alice converts 1 power into 1 coins
Alice scores 3 VP with 10 coins (Resource scoring)
Bob scores 1 VP with 3 coins (Resource scoring)
End of game
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	lastItem, ok := items[len(items)-1].(FinalScoringValidationItem)
	if !ok {
		t.Fatalf("last item type = %T, want FinalScoringValidationItem", items[len(items)-1])
	}

	alice := lastItem.Scores["Witches"]
	if alice == nil {
		t.Fatal("missing Witches final scoring expectation")
	}
	if alice.CultVP != 8 || alice.AreaVP != 18 || alice.ResourceVP != 3 {
		t.Fatalf("Witches final scoring = %+v, want cult=8 area=18 resource=3", *alice)
	}
	if !alice.HasAreaScore || alice.LargestAreaSize != 15 {
		t.Fatalf("Witches area expectation = %+v, want area size 15", *alice)
	}
	if !alice.HasResourceScore || alice.TotalResourceValue != 10 {
		t.Fatalf("Witches resource expectation = %+v, want resource value 10", *alice)
	}
	if !alice.HasExactResourceBreakdown {
		t.Fatalf("Witches exact resource breakdown missing: %+v", *alice)
	}
	if alice.FinalCoinsBeforeResourceScoring != 7 || alice.FinalWorkersBeforeResourceScoring != 2 || alice.FinalPowerCoinsConverted != 1 {
		t.Fatalf("Witches exact resource breakdown = %+v, want coins=7 workers=2 powerCoins=1", *alice)
	}

	bob := lastItem.Scores["Giants"]
	if bob == nil {
		t.Fatal("missing Giants final scoring expectation")
	}
	if bob.CultVP != 4 || bob.AreaVP != 6 || bob.ResourceVP != 1 {
		t.Fatalf("Giants final scoring = %+v, want cult=4 area=6 resource=1", *bob)
	}
	if !bob.HasExactResourceBreakdown {
		t.Fatalf("Giants exact resource breakdown missing: %+v", *bob)
	}
	if bob.FinalCoinsBeforeResourceScoring != 3 || bob.FinalWorkersBeforeResourceScoring != 0 || bob.FinalPriestsBeforeResourceScoring != 0 || bob.FinalPowerCoinsConverted != 0 {
		t.Fatalf("Giants exact resource breakdown = %+v, want only 3 coins on board", *bob)
	}
}

func TestBGAParser_EmitsCleanupPhaseItem(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Witches Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice passes
~ Cleanup phase ~
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	for _, item := range items {
		if _, ok := item.(CleanupPhaseItem); ok {
			return
		}
	}
	t.Fatal("expected CleanupPhaseItem")
}

func TestBGAParser_SkipsDynionGeifrStartingFavorTileDuringSetup(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Dynion Geifr Faction
Bob is playing the Wisps Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
Alice takes a Favor tile
Alice gains 2 on the Cult of Fire track (Favor tile)
Bob places a Dwelling [D5]
~ Action phase ~
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		if favorAction, ok := actionItem.Action.(*LogFavorTileAction); ok && favorAction.PlayerID == "Dynion Geifr" {
			t.Fatalf("unexpected Dynion Geifr setup favor action parsed: %+v", favorAction)
		}
	}
}

func TestBGAParser_ParsesAtlanteansStartingStrongholdSetupPlacement(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Atlanteans Faction
Bob is playing the Engineers Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
Alice places a Faction Stronghold [D5]
Alice founds a Town [D5]
Alice gains 2 on the Cult of Fire track and earns 1 power
Alice gains 2 on the Cult of Air track
Alice gains 2 on the Cult of Earth track
Alice gains 2 on the Cult of Water track and earns 1 power
Alice gets 2 VP and moves 2 spaces forward on each of the 4 Cult tracks
~ Action phase ~
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if len(items) < 3 {
		t.Fatalf("expected setup stronghold and town actions, got %d items", len(items))
	}

	foundSetup := false
	foundTown := false
	foundTownAnchor := false
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		switch action := actionItem.Action.(type) {
		case *game.SetupDwellingAction:
			if action.PlayerID == "Atlanteans" {
				foundSetup = true
			}
		case *LogTownAction:
			if action.PlayerID == "Atlanteans" && action.VP == 2 {
				foundTown = true
				foundTownAnchor = action.AnchorHex != nil && *action.AnchorHex == parseCoord("D5")
			}
		case *LogCompoundAction:
			for _, subaction := range action.Actions {
				if setup, ok := subaction.(*game.SetupDwellingAction); ok && setup.PlayerID == "Atlanteans" {
					foundSetup = true
				}
				if town, ok := subaction.(*LogTownAction); ok && town.PlayerID == "Atlanteans" && town.VP == 2 {
					foundTown = true
					foundTownAnchor = town.AnchorHex != nil && *town.AnchorHex == parseCoord("D5")
				}
			}
		}
	}
	if !foundSetup || !foundTown || !foundTownAnchor {
		t.Fatalf("foundSetup=%t foundTown=%t foundTownAnchor=%t, items=%#v", foundSetup, foundTown, foundTownAnchor, items)
	}
}

func TestBGAParser_ParsesAtlanteansBridgeAbility(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Atlanteans Faction
Bob is playing the Engineers Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice spends 2 workers to build a Bridge (Atlanteans Ability) [C4-D5]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		bridge, ok := actionItem.Action.(*game.EngineersBridgeAction)
		if !ok {
			continue
		}
		if bridge.PlayerID != "Atlanteans" {
			t.Fatalf("bridge parsed for wrong player: %+v", bridge)
		}
		if bridge.BridgeHex1 != parseCoord("C4") || bridge.BridgeHex2 != parseCoord("D5") {
			t.Fatalf("bridge coordinates = %v-%v, want C4-D5", bridge.BridgeHex1, bridge.BridgeHex2)
		}
		return
	}

	t.Fatalf("expected Atlanteans bridge action in parsed items: %#v", items)
}

func TestBGAParser_RemovesCanceledTurnSegmentWhenReplacementFollows(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Conspirators Faction
Bob is playing the Giants Faction
Carol is playing the Wisps Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice does some Conversions (spent: 3 power 0 Priests 0 workers ; collects: 0 Priests  0 workers 3 coins)
Alice upgrades a Dwelling to a Trading house for 2 workers 3 coins [D2]
Bob declines getting Power via Structures [D2]
Alice cancels their move
Alice upgrades a Dwelling to a Trading house for 2 workers 3 coins [C2]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	foundReplacement := false
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}

		if action, ok := actionItem.Action.(*LogConversionAction); ok && action.PlayerID == "Conspirators" {
			t.Fatalf("found canceled Conspirators conversion still present: %+v", action)
		}

		if action, ok := actionItem.Action.(*game.UpgradeBuildingAction); ok && action.PlayerID == "Conspirators" {
			foundReplacement = true
			if got := action.TargetHex; got != parseCoord("C2") {
				t.Fatalf("replacement upgrade target = %v, want %v", got, parseCoord("C2"))
			}
		}
	}
	if !foundReplacement {
		t.Fatal("expected replacement upgrade after cancel")
	}
}

func TestBGAParser_CancelMoveKeepsPreviousTurnWhenNextLineIsFollowUpChoice(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Conspirators Faction
Bob is playing the Giants Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice does some Conversions (spent: 3 power 0 Priests 0 workers ; collects: 0 Priests  0 workers 3 coins)
Alice upgrades a Dwelling to a Trading house for 2 workers 3 coins [H8]
Bob declines getting Power via Structures [H8]
Alice cancels their move
Alice gives back a Favor tile and loses 1 Cult points (Conspirators Stronghold)
Alice takes a Favor tile
Alice gains 1 on the Cult of Air track (Favor tile) and earns 3 power
Alice earns 2 coins (Conspirators ability)
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	foundUpgrade := false
	foundSwap := false
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		if action, ok := actionItem.Action.(*game.UpgradeBuildingAction); ok && action.PlayerID == "Conspirators" {
			foundUpgrade = true
		}
		if _, ok := actionItem.Action.(*LogConspiratorsSwapFavorAction); ok {
			foundSwap = true
		}
	}
	if !foundUpgrade {
		t.Fatal("expected original upgrade to remain when cancel is followed by a swap choice")
	}
	if !foundSwap {
		t.Fatal("expected follow-up Conspirators swap action after cancel")
	}
}

func TestBGAParser_CancelMoveKeepsPreviousTurnWhenNextLineIsPass(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Alchemists Faction
Bob is playing the Giants Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice builds a Dwelling for 1 workers 2 coins [E5]
Alice cancels their move
Alice passes
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	foundBuild := false
	foundPass := false
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		if action, ok := actionItem.Action.(*game.TransformAndBuildAction); ok && action.PlayerID == "Alchemists" {
			foundBuild = true
		}
		if action, ok := actionItem.Action.(*game.PassAction); ok && action.PlayerID == "Alchemists" {
			foundPass = true
		}
	}
	if !foundBuild {
		t.Fatal("expected committed build to remain when cancel is followed by pass")
	}
	if !foundPass {
		t.Fatal("expected pass after cancel")
	}
}

func TestBGAParser_CancelMoveDoesNotRemoveCommittedTurnAfterOtherPlayersAct(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Conspirators Faction
Bob is playing the Giants Faction
Carol is playing the Wisps Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice sacrificed 1 power in Bowl 2 to get 1 power from Bowl 2 to Bowl 3
Alice spends 4 power to collect 7 coins (Power action)
Alice declines doing Conversions
Bob spends 4 power to collect 2 workers (Power action)
Bob declines doing Conversions
Carol builds a Dwelling for 1 workers 2 coins [B2]
Alice gets 1 power via Structures [B2]
Alice cancels their move
Alice builds a Dwelling for 1 workers 2 coins [G3]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	foundCoins := false
	foundBurn := false
	foundReplacementBuild := false
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		switch action := actionItem.Action.(type) {
		case *LogBurnAction:
			if action.PlayerID == "Conspirators" && action.Amount == 1 && action.Moved == 1 {
				foundBurn = true
			}
		case *LogPowerAction:
			if action.PlayerID == "Conspirators" && action.ActionCode == "ACT4" {
				foundCoins = true
			}
		case *game.SetupDwellingAction:
			if action.PlayerID == "Conspirators" && action.Hex == parseCoord("G3") {
				foundReplacementBuild = true
			}
		case *game.TransformAndBuildAction:
			if action.PlayerID == "Conspirators" && action.BuildDwelling && action.TargetHex == parseCoord("G3") {
				foundReplacementBuild = true
			}
		}
	}

	if !foundBurn {
		t.Fatal("expected committed burn action to remain after later cancel")
	}
	if !foundCoins {
		t.Fatal("expected committed ACT4 coin action to remain after later cancel")
	}
	if !foundReplacementBuild {
		t.Fatal("expected replacement build after cancel")
	}
}

func TestBGAParser_CancelMoveKeepsChaosMagiciansDoubleTurnActions(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Chaos Magicians Faction
Bob is playing the Giants Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
~ Action phase ~
Alice takes a double-turn (Chaos Magicians Stronghold)
Alice sacrificed 3 power in Bowl 2 to get 3 power from Bowl 2 to Bowl 3
Alice spends 4 power to collect 7 coins (Power action)
Alice declines doing Conversions
Alice cancels their move
Alice cancels their move
Alice upgrades a Trading house to a Temple for 2 workers 5 coins [C5]
Alice takes a Favor tile
Alice gains 3 on the Cult of Earth track (Favor tile) and earns 3 power
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	foundDoubleTurn := false
	foundBurn := false
	foundCoins := false
	foundUpgrade := false
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		switch action := actionItem.Action.(type) {
		case *LogSpecialAction:
			if action.PlayerID == "Chaos Magicians" && action.ActionCode == "ACT-SH-2X" {
				foundDoubleTurn = true
			}
		case *LogBurnAction:
			if action.PlayerID == "Chaos Magicians" && action.Amount == 3 && action.Moved == 3 {
				foundBurn = true
			}
		case *LogPowerAction:
			if action.PlayerID == "Chaos Magicians" && action.ActionCode == "ACT4" {
				foundCoins = true
			}
		case *game.UpgradeBuildingAction:
			if action.PlayerID == "Chaos Magicians" && action.NewBuildingType == models.BuildingTemple {
				foundUpgrade = true
			}
		}
	}

	if !foundDoubleTurn {
		t.Fatal("expected Chaos Magicians double-turn action to remain after cancel")
	}
	if !foundBurn {
		t.Fatal("expected committed burn action to remain after cancel")
	}
	if !foundCoins {
		t.Fatal("expected committed ACT4 coin action to remain after cancel")
	}
	if !foundUpgrade {
		t.Fatal("expected replacement temple upgrade after cancel")
	}
}

func TestBGAParser_ParsesChildrenStrongholdPowerTokenPlacement(t *testing.T) {
	content := `Game board: Base Game
Alice is playing the Children Of The Wyrm Faction
Bob is playing the Halflings Faction
~ Every player has chosen a Faction and receives the matching starting resources. ~
Alice places a Dwelling [E5]
Alice places a Dwelling [E10]
Bob places a Dwelling [E6]
Bob places a Dwelling [F5]
~ Action phase ~
Alice sacrifices Power to place 2 power on the game board (Children Of The Wyrm Stronghold) [R~D3] + [R~C2]
`

	parser := NewBGAParser(content)
	items, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	found := false
	for _, item := range items {
		actionItem, ok := item.(ActionItem)
		if !ok || actionItem.Action == nil {
			continue
		}
		action, ok := actionItem.Action.(*LogChildrenPlacePowerTokensAction)
		if !ok {
			continue
		}
		if action.PlayerID != "Children Of The Wyrm" {
			continue
		}
		found = true
		if len(action.RiverCoords) != 2 {
			t.Fatalf("children power-token target count = %d, want 2", len(action.RiverCoords))
		}
		if action.RiverCoords[0] != "R~D3" || action.RiverCoords[1] != "R~C2" {
			t.Fatalf("children power-token refs = %v, want [R~D3 R~C2]", action.RiverCoords)
		}
	}
	if !found {
		t.Fatal("expected children power-token placement action")
	}
}
