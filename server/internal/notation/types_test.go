package notation

import (
	"fmt"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestLogCompoundAction_AllowsAuxiliaryOnlySequence(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}
	// Ensure the auxiliary actions are executable.
	gs.Players[playerID].Resources.Power = game.NewPowerSystem(0, 2, 1)

	compound := &LogCompoundAction{
		Actions: []game.Action{
			&LogBurnAction{PlayerID: playerID, Amount: 1},
			&LogConversionAction{
				PlayerID: playerID,
				Cost: map[models.ResourceType]int{
					models.ResourcePower: 1,
				},
				Reward: map[models.ResourceType]int{
					models.ResourceCoin: 1,
				},
			},
		},
	}

	if err := compound.Execute(gs); err != nil {
		t.Fatalf("compound.Execute(auxiliary-only) error = %v, want nil", err)
	}
}

func TestLogBurnAction_UsesStandardTwoToOneBurnSemantics(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}
	gs.Players[playerID].Resources.Power = game.NewPowerSystem(0, 4, 0)

	action := &LogBurnAction{PlayerID: playerID, Amount: 2}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogBurnAction.Execute() error = %v", err)
	}

	power := gs.Players[playerID].Resources.Power
	if power.Bowl2 != 0 || power.Bowl3 != 2 {
		t.Fatalf("power after burn = %d/%d/%d, want 0/0/2", power.Bowl1, power.Bowl2, power.Bowl3)
	}
}

func TestLogConversionAction_AutoBurnsToFundLoggedPowerSpend(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}
	gs.Players[playerID].Resources.Power = game.NewPowerSystem(0, 2, 4)

	action := &LogConversionAction{
		PlayerID: playerID,
		Cost: map[models.ResourceType]int{
			models.ResourcePower: 5,
		},
		Reward: map[models.ResourceType]int{
			models.ResourcePriest: 1,
		},
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogConversionAction.Execute() error = %v", err)
	}

	player := gs.Players[playerID]
	if got := player.Resources.Power.Bowl1; got != 5 {
		t.Fatalf("bowl I after conversion = %d, want 5", got)
	}
	if got := player.Resources.Power.Bowl2; got != 0 {
		t.Fatalf("bowl II after conversion = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl3; got != 0 {
		t.Fatalf("bowl III after conversion = %d, want 0", got)
	}
	if got := player.Resources.Priests; got != 1 {
		t.Fatalf("priests after conversion = %d, want 1", got)
	}
}

func TestLogDeclineLeechAction_NoPendingOffers_NoError(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}

	action := &LogDeclineLeechAction{PlayerID: playerID}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogDeclineLeechAction.Execute(no pending) error = %v, want nil", err)
	}
}

func TestLogDeclineLeechAction_FromPlayerIDDeclinesMatchingOffer(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewAuren()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	gs.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{
		{Amount: 2, VPCost: 1, FromPlayerID: "Archivists"},
		{Amount: 2, VPCost: 1, FromPlayerID: "Conspirators"},
	}

	action := &LogDeclineLeechAction{PlayerID: "p1", FromPlayerID: "Conspirators"}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogDeclineLeechAction.Execute() error = %v", err)
	}

	offers := gs.PendingLeechOffers["p1"]
	if len(offers) != 1 {
		t.Fatalf("pending offers after decline = %d, want 1", len(offers))
	}
	if got := offers[0].FromPlayerID; got != "Archivists" {
		t.Fatalf("remaining pending offer source = %q, want Archivists", got)
	}
}

func TestLogDeclineLeechAction_FromHexDeclinesMatchingOfferWithSameSourcePlayer(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewAuren()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	firstHex := board.NewHex(1, 5)
	secondHex := board.NewHex(2, 5)
	gs.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{
		{Amount: 3, VPCost: 2, FromPlayerID: "Conspirators", SourceHex: &firstHex},
		{Amount: 2, VPCost: 1, FromPlayerID: "Conspirators", SourceHex: &secondHex},
	}

	action := &LogDeclineLeechAction{PlayerID: "p1", FromPlayerID: "Conspirators", FromHex: &secondHex}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogDeclineLeechAction.Execute() error = %v", err)
	}

	offers := gs.PendingLeechOffers["p1"]
	if len(offers) != 1 {
		t.Fatalf("pending offers after hex decline = %d, want 1", len(offers))
	}
	if offers[0].SourceHex == nil || *offers[0].SourceHex != firstHex {
		t.Fatalf("remaining pending offer source hex = %+v, want %+v", offers[0].SourceHex, firstHex)
	}
}

func TestLogAcceptLeechAction_ExplicitAmountOverridesPendingOffer(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewAuren()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer("p1")
	player.VictoryPoints = 20
	player.Resources.Power = game.NewPowerSystem(0, 3, 6)
	gs.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{
		{Amount: 6, VPCost: 5},
	}

	action := &LogAcceptLeechAction{
		PlayerID:    "p1",
		PowerAmount: 2,
		VPCost:      1,
		Explicit:    true,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogAcceptLeechAction.Execute() error = %v", err)
	}

	if got := player.VictoryPoints; got != 19 {
		t.Fatalf("victory points after capped leech = %d, want 19", got)
	}
	if got := player.Resources.Power.Bowl2; got != 1 {
		t.Fatalf("bowl II after capped leech = %d, want 1", got)
	}
	if got := player.Resources.Power.Bowl3; got != 8 {
		t.Fatalf("bowl III after capped leech = %d, want 8", got)
	}
}

func TestLogAcceptLeechAction_FromPlayerIDFallsBackToSourceWhenAmountIsCapped(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewAuren()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer("p1")
	player.VictoryPoints = 20
	player.Resources.Power = game.NewPowerSystem(0, 3, 6)
	gs.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{
		{Amount: 6, VPCost: 5, FromPlayerID: "Archivists"},
	}

	action := &LogAcceptLeechAction{
		PlayerID:     "p1",
		FromPlayerID: "Archivists",
		PowerAmount:  2,
		VPCost:       1,
		Explicit:     true,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogAcceptLeechAction.Execute() error = %v", err)
	}

	if got := player.VictoryPoints; got != 19 {
		t.Fatalf("victory points after capped source leech = %d, want 19", got)
	}
}

func TestLogAcceptLeechAction_FromHexSelectsMatchingOfferWithSameSourcePlayer(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewAuren()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer("p1")
	player.VictoryPoints = 20
	player.Resources.Power = game.NewPowerSystem(0, 4, 4)

	firstHex := board.NewHex(1, 5)
	secondHex := board.NewHex(2, 5)
	gs.PendingLeechOffers["p1"] = []*game.PowerLeechOffer{
		{Amount: 3, VPCost: 2, FromPlayerID: "Conspirators", SourceHex: &firstHex},
		{Amount: 2, VPCost: 1, FromPlayerID: "Conspirators", SourceHex: &secondHex},
	}

	action := &LogAcceptLeechAction{
		PlayerID:     "p1",
		FromPlayerID: "Conspirators",
		FromHex:      &secondHex,
		PowerAmount:  2,
		VPCost:       1,
		Explicit:     true,
	}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogAcceptLeechAction.Execute() error = %v", err)
	}

	if got := len(gs.PendingLeechOffers["p1"]); got != 1 {
		t.Fatalf("pending offers after hex accept = %d, want 1", got)
	}
	if gs.PendingLeechOffers["p1"][0].SourceHex == nil || *gs.PendingLeechOffers["p1"][0].SourceHex != firstHex {
		t.Fatalf("remaining pending offer source hex = %+v, want %+v", gs.PendingLeechOffers["p1"][0].SourceHex, firstHex)
	}
}

func TestLogCultTrackDecreaseAction_DecrementsCultTrack(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	if err := gs.AddPlayer(playerID, factions.NewCultists()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer(playerID)
	if player == nil {
		t.Fatalf("player %q missing", playerID)
	}
	player.CultPositions[game.CultWater] = 2
	gs.CultTracks.PlayerPositions[playerID][game.CultWater] = 2

	action := &LogCultTrackDecreaseAction{PlayerID: playerID, Track: game.CultWater}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogCultTrackDecreaseAction.Execute() error = %v", err)
	}
	if got := player.CultPositions[game.CultWater]; got != 1 {
		t.Fatalf("player cult water = %d, want 1", got)
	}
	if got := gs.CultTracks.GetPosition(playerID, game.CultWater); got != 1 {
		t.Fatalf("cult track water = %d, want 1", got)
	}
}

func TestLogCompoundAction_CultTownSelectionWithDecreaseToken(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	if err := gs.AddPlayer(playerID, factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer(playerID)
	if player == nil {
		t.Fatalf("player %q missing", playerID)
	}

	// Mirror the S63_G2 shape: two cults near 10, one track intentionally reduced
	// before applying a +1-all town tile.
	player.CultPositions[game.CultFire] = 6
	player.CultPositions[game.CultWater] = 9
	player.CultPositions[game.CultEarth] = 4
	player.CultPositions[game.CultAir] = 9
	gs.CultTracks.PlayerPositions[playerID][game.CultFire] = 6
	gs.CultTracks.PlayerPositions[playerID][game.CultWater] = 9
	gs.CultTracks.PlayerPositions[playerID][game.CultEarth] = 4
	gs.CultTracks.PlayerPositions[playerID][game.CultAir] = 9

	// Allow bonus power gain visibility (+3 when topping exactly one cult).
	player.Resources.Power.Bowl1 = 0
	player.Resources.Power.Bowl2 = 3
	player.Resources.Power.Bowl3 = 3

	// LogTownAction requires a pending town formation entry.
	gs.PendingTownFormations[playerID] = []game.PendingTownFormation{
		{PlayerID: playerID, Hexes: []board.Hex{}},
	}

	action, err := parseActionCode(playerID, "-W.TW8VP")
	if err != nil {
		t.Fatalf("parseActionCode(-W.TW8VP) error = %v", err)
	}
	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parsed action type = %T, want *LogCompoundAction", action)
	}
	if err := compound.Execute(gs); err != nil {
		t.Fatalf("compound.Execute() error = %v", err)
	}

	if got := player.CultPositions[game.CultFire]; got != 7 {
		t.Fatalf("fire cult = %d, want 7", got)
	}
	if got := player.CultPositions[game.CultWater]; got != 9 {
		t.Fatalf("water cult = %d, want 9", got)
	}
	if got := player.CultPositions[game.CultEarth]; got != 5 {
		t.Fatalf("earth cult = %d, want 5", got)
	}
	if got := player.CultPositions[game.CultAir]; got != 10 {
		t.Fatalf("air cult = %d, want 10", got)
	}
	if got := player.Resources.Power.Bowl1; got != 0 {
		t.Fatalf("power bowl1 = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl2; got != 0 {
		t.Fatalf("power bowl2 = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl3; got != 6 {
		t.Fatalf("power bowl3 = %d, want 6", got)
	}
}

func TestLogCompoundAction_MultipleCultTownDecreaseTokens(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	if err := gs.AddPlayer(playerID, factions.NewEngineers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer(playerID)
	if player == nil {
		t.Fatalf("player %q missing", playerID)
	}

	player.CultPositions[game.CultFire] = 0
	player.CultPositions[game.CultWater] = 0
	player.CultPositions[game.CultEarth] = 0
	player.CultPositions[game.CultAir] = 0
	gs.CultTracks.PlayerPositions[playerID][game.CultFire] = 0
	gs.CultTracks.PlayerPositions[playerID][game.CultWater] = 0
	gs.CultTracks.PlayerPositions[playerID][game.CultEarth] = 0
	gs.CultTracks.PlayerPositions[playerID][game.CultAir] = 0

	player.Resources.Power.Bowl1 = 12
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0
	gs.PendingTownFormations[playerID] = []game.PendingTownFormation{
		{PlayerID: playerID, Hexes: []board.Hex{}},
	}

	action, err := parseActionCode(playerID, "-F.-W.-E.TW2VP")
	if err != nil {
		t.Fatalf("parseActionCode(-F.-W.-E.TW2VP) error = %v", err)
	}
	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parsed action type = %T, want *LogCompoundAction", action)
	}
	if err := compound.Execute(gs); err != nil {
		t.Fatalf("compound.Execute() error = %v", err)
	}

	if got := player.CultPositions[game.CultFire]; got != 1 {
		t.Fatalf("fire cult = %d, want 1", got)
	}
	if got := player.CultPositions[game.CultWater]; got != 1 {
		t.Fatalf("water cult = %d, want 1", got)
	}
	if got := player.CultPositions[game.CultEarth]; got != 1 {
		t.Fatalf("earth cult = %d, want 1", got)
	}
	if got := player.CultPositions[game.CultAir]; got != 2 {
		t.Fatalf("air cult = %d, want 2", got)
	}
	// No milestone crossings in this setup; power should remain unchanged.
	if got := player.Resources.Power.Bowl1; got != 12 {
		t.Fatalf("power bowl1 = %d, want 12", got)
	}
	if got := player.Resources.Power.Bowl2; got != 0 {
		t.Fatalf("power bowl2 = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl3; got != 0 {
		t.Fatalf("power bowl3 = %d, want 0", got)
	}
}

func TestLogCompoundAction_CultTownSelectorsLimitTopToSingleTrackWithOneKey(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	if err := gs.AddPlayer(playerID, factions.NewEngineers()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer(playerID)
	if player == nil {
		t.Fatalf("player %q missing", playerID)
	}

	// Start with no keys; TW8 grants exactly one key before cult advancement,
	// so only one track can top to 10.
	player.Keys = 0
	player.CultPositions[game.CultFire] = 9
	player.CultPositions[game.CultWater] = 9
	player.CultPositions[game.CultEarth] = 9
	player.CultPositions[game.CultAir] = 9
	gs.CultTracks.PlayerPositions[playerID][game.CultFire] = 9
	gs.CultTracks.PlayerPositions[playerID][game.CultWater] = 9
	gs.CultTracks.PlayerPositions[playerID][game.CultEarth] = 9
	gs.CultTracks.PlayerPositions[playerID][game.CultAir] = 9

	player.Resources.Power.Bowl1 = 0
	player.Resources.Power.Bowl2 = 3
	player.Resources.Power.Bowl3 = 3

	gs.PendingTownFormations[playerID] = []game.PendingTownFormation{
		{PlayerID: playerID, Hexes: []board.Hex{}},
	}

	action, err := parseActionCode(playerID, "-F.-W.-E.TW8VP")
	if err != nil {
		t.Fatalf("parseActionCode(-F.-W.-E.TW8VP) error = %v", err)
	}
	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parsed action type = %T, want *LogCompoundAction", action)
	}
	if err := compound.Execute(gs); err != nil {
		t.Fatalf("compound.Execute() error = %v", err)
	}

	if got := player.CultPositions[game.CultFire]; got != 9 {
		t.Fatalf("fire cult = %d, want 9", got)
	}
	if got := player.CultPositions[game.CultWater]; got != 9 {
		t.Fatalf("water cult = %d, want 9", got)
	}
	if got := player.CultPositions[game.CultEarth]; got != 9 {
		t.Fatalf("earth cult = %d, want 9", got)
	}
	if got := player.CultPositions[game.CultAir]; got != 10 {
		t.Fatalf("air cult = %d, want 10", got)
	}
	if got := gs.CultTracks.Position10Occupied[game.CultAir]; got != playerID {
		t.Fatalf("air position 10 occupier = %q, want %q", got, playerID)
	}
	if got := player.Keys; got != 1 {
		t.Fatalf("keys = %d, want 1", got)
	}
	if _, occupied := gs.CultTracks.Position10Occupied[game.CultFire]; occupied {
		t.Fatalf("fire position 10 should not be occupied")
	}
	if _, occupied := gs.CultTracks.Position10Occupied[game.CultWater]; occupied {
		t.Fatalf("water position 10 should not be occupied")
	}
	if _, occupied := gs.CultTracks.Position10Occupied[game.CultEarth]; occupied {
		t.Fatalf("earth position 10 should not be occupied")
	}

	// Only one 10-step crossing happened (air), so exactly +3 power is gained.
	if got := player.Resources.Power.Bowl1; got != 0 {
		t.Fatalf("power bowl1 = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl2; got != 0 {
		t.Fatalf("power bowl2 = %d, want 0", got)
	}
	if got := player.Resources.Power.Bowl3; got != 6 {
		t.Fatalf("power bowl3 = %d, want 6", got)
	}
}

func TestLogCompoundAction_CultTownSelectorsRequirePendingTownFormation(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	if err := gs.AddPlayer(playerID, factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer(playerID)
	if player == nil {
		t.Fatalf("player %q missing", playerID)
	}
	player.CultPositions[game.CultWater] = 9
	gs.CultTracks.PlayerPositions[playerID][game.CultWater] = 9

	action, err := parseActionCode(playerID, "-W.TW8VP")
	if err != nil {
		t.Fatalf("parseActionCode(-W.TW8VP) error = %v", err)
	}
	compound, ok := action.(*LogCompoundAction)
	if !ok {
		t.Fatalf("parsed action type = %T, want *LogCompoundAction", action)
	}
	if err := compound.Execute(gs); err == nil {
		t.Fatalf("compound.Execute() expected error when no pending town exists")
	}
}

func TestParseReplayInsufficientResources_WrappedPrefix(t *testing.T) {
	err := fmt.Errorf("failed to pay for dwelling: insufficient resources: need (coins:2, workers:1, priests:0, power:0), have (coins:0, workers:5, priests:0, power:0)")
	got, ok := parseReplayInsufficientResources(err)
	if !ok {
		t.Fatalf("parseReplayInsufficientResources() did not match wrapped message")
	}
	if got.needCoins != 2 || got.needWorkers != 1 || got.needPriests != 0 || got.needPower != 0 {
		t.Fatalf("parsed need = %+v, want coins=2 workers=1 priests=0 power=0", got)
	}
	if got.haveCoins != 0 || got.haveWorkers != 5 || got.havePriests != 0 || got.havePower != 0 {
		t.Fatalf("parsed have = %+v, want coins=0 workers=5 priests=0 power=0", got)
	}
}

func TestIncomeWrapper_TransformOnlyRetriesWithSyntheticSpade(t *testing.T) {
	testCases := []struct {
		name  string
		wrap  func(game.Action) game.Action
	}{
		{
			name: "pre_income",
			wrap: func(a game.Action) game.Action { return &LogPreIncomeAction{Action: a} },
		},
		{
			name: "post_income",
			wrap: func(a game.Action) game.Action { return &LogPostIncomeAction{Action: a} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := game.NewGameState()
			playerID := "cultists"
			if err := gs.AddPlayer(playerID, factions.NewCultists()); err != nil {
				t.Fatalf("AddPlayer failed: %v", err)
			}
			player := gs.GetPlayer(playerID)
			if player == nil {
				t.Fatalf("player %q missing after AddPlayer", playerID)
			}

			target, anchor, ok := findAdjacentNonRiverPair(gs)
			if !ok {
				t.Fatalf("failed to find adjacent non-river hex pair")
			}

			anchorHex := gs.Map.GetHex(anchor)
			if anchorHex == nil {
				t.Fatalf("anchor hex missing")
			}
			anchorHex.Building = &models.Building{
				Type:       models.BuildingDwelling,
				Faction:    player.Faction.GetType(),
				PlayerID:   playerID,
				PowerValue: game.GetPowerValue(models.BuildingDwelling),
			}

			home := player.Faction.GetHomeTerrain()
			oneStepTerrain, ok := findTerrainDistanceOneToHome(gs, home)
			if !ok {
				t.Fatalf("failed to find distance-1 terrain for home %v", home)
			}
			targetHex := gs.Map.GetHex(target)
			if targetHex == nil {
				t.Fatalf("target hex missing")
			}
			targetHex.Terrain = oneStepTerrain

			player.Resources.Workers = 0
			action := game.NewTransformAndBuildAction(playerID, target, false, home)
			wrapped := tc.wrap(action)

			if err := wrapped.Execute(gs); err != nil {
				t.Fatalf("wrapped.Execute() error = %v, want nil", err)
			}
			if got := gs.Map.GetHex(target).Terrain; got != home {
				t.Fatalf("target terrain = %v, want %v", got, home)
			}
			if got := player.Resources.Workers; got != 0 {
				t.Fatalf("workers = %d, want 0", got)
			}
		})
	}
}

func findAdjacentNonRiverPair(gs *game.GameState) (board.Hex, board.Hex, bool) {
	if gs == nil || gs.Map == nil {
		return board.Hex{}, board.Hex{}, false
	}
	for h, hex := range gs.Map.Hexes {
		if hex == nil || hex.Terrain == models.TerrainRiver {
			continue
		}
		for _, n := range h.Neighbors() {
			neighbor := gs.Map.GetHex(n)
			if neighbor == nil || neighbor.Terrain == models.TerrainRiver {
				continue
			}
			return h, n, true
		}
	}
	return board.Hex{}, board.Hex{}, false
}

func findTerrainDistanceOneToHome(gs *game.GameState, home models.TerrainType) (models.TerrainType, bool) {
	candidates := []models.TerrainType{
		models.TerrainDesert,
		models.TerrainPlains,
		models.TerrainSwamp,
		models.TerrainLake,
		models.TerrainForest,
		models.TerrainMountain,
		models.TerrainWasteland,
	}
	for _, t := range candidates {
		if t == home {
			continue
		}
		if gs.Map.GetTerrainDistance(t, home) == 1 {
			return t, true
		}
	}
	return models.TerrainTypeUnknown, false
}
