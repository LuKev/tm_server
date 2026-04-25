package notation

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestLogPowerAction_BridgeUsesMapAwareCoordinates(t *testing.T) {
	gs := game.NewGameState()
	fjords, err := board.NewTerraMysticaMapForID(board.MapFjords)
	if err != nil {
		t.Fatalf("NewTerraMysticaMapForID(MapFjords) failed: %v", err)
	}
	gs.Map = fjords
	if err := gs.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.GetPlayer("p1").Resources.Power = game.NewPowerSystem(0, 0, 3)

	action := &LogPowerAction{PlayerID: "p1", ActionCode: "ACT1-H6-I6"}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogPowerAction.Execute failed: %v", err)
	}

	hex1, err := ConvertLogCoordToAxialForMap(board.MapFjords, "H6")
	if err != nil {
		t.Fatalf("ConvertLogCoordToAxialForMap(H6) failed: %v", err)
	}
	hex2, err := ConvertLogCoordToAxialForMap(board.MapFjords, "I6")
	if err != nil {
		t.Fatalf("ConvertLogCoordToAxialForMap(I6) failed: %v", err)
	}
	if !gs.Map.HasBridge(hex1, hex2) {
		t.Fatalf("expected bridge between %v and %v", hex1, hex2)
	}
}

func TestLogPowerAction_ReplayAutoBurnsMissingPower(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.ReplayMode = map[string]bool{"__replay__": true}
	player := gs.GetPlayer("p1")
	player.Resources.Power = game.NewPowerSystem(0, 6, 1)

	action := &LogPowerAction{PlayerID: "p1", ActionCode: "ACT3"}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogPowerAction.Execute failed: %v", err)
	}

	if got := player.Resources.Power.Bowl1; got != 4 {
		t.Fatalf("bowl1 = %d, want 4 after auto-burn and spend", got)
	}
	if got := player.Resources.Power.Bowl2; got != 0 {
		t.Fatalf("bowl2 = %d, want 0 after auto-burn", got)
	}
	if got := player.Resources.Power.Bowl3; got != 0 {
		t.Fatalf("bowl3 = %d, want 0 after spend", got)
	}
	if got := player.Resources.Workers; got != 5 {
		t.Fatalf("workers = %d, want 5 after gaining 2 workers", got)
	}
}

func TestLogPowerAction_NonReplayRequiresExplicitBurn(t *testing.T) {
	gs := game.NewGameState()
	if err := gs.AddPlayer("p1", factions.NewWitches()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	player := gs.GetPlayer("p1")
	player.Resources.Power = game.NewPowerSystem(0, 6, 1)

	action := &LogPowerAction{PlayerID: "p1", ActionCode: "ACT3"}
	if err := action.Execute(gs); err == nil {
		t.Fatal("expected ACT3 without replay auto-burn to fail")
	}
}

func TestLogRiverBuildAction_ResolvesSelkiesRiverHexFromBoardState(t *testing.T) {
	gs, err := game.NewGameStateWithMap(board.MapFjords)
	if err != nil {
		t.Fatalf("NewGameStateWithMap(MapFjords) failed: %v", err)
	}
	if err := gs.AddPlayer("selkies", factions.NewSelkies()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"selkies"}

	player := gs.GetPlayer("selkies")
	player.Resources.Workers = 3
	player.Resources.Coins = 5

	landHex, ok := board.HexForDisplayCoordinate(board.MapFjords, "B5")
	if !ok {
		t.Fatal("expected B5 coordinate to exist on Fjords")
	}
	defaultHex, err := ConvertRiverCoordToAxialForMap(board.MapFjords, "R~B5")
	if err != nil {
		t.Fatalf("ConvertRiverCoordToAxialForMap(R~B5) failed: %v", err)
	}

	riverCandidates := riverNeighborsForTest(gs.Map, landHex)
	if len(riverCandidates) != 2 {
		t.Fatalf("expected 2 river candidates for R~B5, got %d", len(riverCandidates))
	}

	defaultSupport := landNeighborsForTest(gs.Map, defaultHex)
	defaultSupportSet := make(map[board.Hex]bool, len(defaultSupport))
	for _, hex := range defaultSupport {
		defaultSupportSet[hex] = true
	}

	var (
		expectedHex   board.Hex
		supportA      board.Hex
		supportB      board.Hex
		foundSolution bool
	)
	for _, candidate := range riverCandidates {
		if candidate == defaultHex {
			continue
		}
		pairs := nonAdjacentLandNeighborPairsForTest(gs.Map, candidate)
		for _, pair := range pairs {
			if defaultSupportSet[pair[0]] && defaultSupportSet[pair[1]] {
				continue
			}
			expectedHex = candidate
			supportA = pair[0]
			supportB = pair[1]
			foundSolution = true
			break
		}
		if foundSolution {
			break
		}
	}
	if !foundSolution {
		t.Fatal("failed to find a Selkies support pair that uniquely selects the non-default river hex")
	}

	for _, hex := range []board.Hex{supportA, supportB} {
		if err := gs.Map.TransformTerrain(hex, models.TerrainIce); err != nil {
			t.Fatalf("TransformTerrain(%v) failed: %v", hex, err)
		}
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			PlayerID:   "selkies",
			Faction:    models.FactionSelkies,
			PowerValue: 1,
		})
	}

	action := &LogRiverBuildAction{PlayerID: "selkies", CoordToken: "R~B5"}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogRiverBuildAction.Execute failed: %v", err)
	}

	if building := gs.Map.GetHex(expectedHex).Building; building == nil || building.PlayerID != "selkies" {
		t.Fatalf("expected Selkies dwelling at %v, got %+v", expectedHex, building)
	}
	if expectedHex != defaultHex {
		if building := gs.Map.GetHex(defaultHex).Building; building != nil {
			t.Fatalf("default river hex %v should remain empty, got %+v", defaultHex, building)
		}
	}
}

func TestLogRiverBuildAction_PrefersDefaultRiverHexWhenStillLegal(t *testing.T) {
	gs, err := game.NewGameStateWithMap(board.MapFjords)
	if err != nil {
		t.Fatalf("NewGameStateWithMap(MapFjords) failed: %v", err)
	}
	if err := gs.AddPlayer("selkies", factions.NewSelkies()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"selkies"}

	player := gs.GetPlayer("selkies")
	player.Resources.Workers = 3
	player.Resources.Coins = 5

	landHex, ok := board.HexForDisplayCoordinate(board.MapFjords, "B2")
	if !ok {
		t.Fatal("expected B2 coordinate to exist on Fjords")
	}
	defaultHex, err := ConvertRiverCoordToAxialForMap(board.MapFjords, "R~B2")
	if err != nil {
		t.Fatalf("ConvertRiverCoordToAxialForMap(R~B2) failed: %v", err)
	}

	riverCandidates := riverNeighborsForTest(gs.Map, landHex)
	if len(riverCandidates) != 2 {
		t.Fatalf("expected 2 river candidates for R~B2, got %d", len(riverCandidates))
	}

	defaultPairs := nonAdjacentLandNeighborPairsForTest(gs.Map, defaultHex)
	if len(defaultPairs) == 0 {
		t.Fatal("expected a non-adjacent support pair for the default river hex")
	}
	var otherHex board.Hex
	for _, candidate := range riverCandidates {
		if candidate != defaultHex {
			otherHex = candidate
			break
		}
	}
	otherPairs := nonAdjacentLandNeighborPairsForTest(gs.Map, otherHex)
	if len(otherPairs) == 0 {
		t.Fatal("expected a non-adjacent support pair for the alternate river hex")
	}

	supportHexes := map[board.Hex]bool{
		defaultPairs[0][0]: true,
		defaultPairs[0][1]: true,
		otherPairs[0][0]:   true,
		otherPairs[0][1]:   true,
	}
	for hex := range supportHexes {
		if err := gs.Map.TransformTerrain(hex, models.TerrainIce); err != nil {
			t.Fatalf("TransformTerrain(%v) failed: %v", hex, err)
		}
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			PlayerID:   "selkies",
			Faction:    models.FactionSelkies,
			PowerValue: 1,
		})
	}

	action := &LogRiverBuildAction{PlayerID: "selkies", CoordToken: "R~B2"}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogRiverBuildAction.Execute failed: %v", err)
	}

	if building := gs.Map.GetHex(defaultHex).Building; building == nil || building.PlayerID != "selkies" {
		t.Fatalf("expected Selkies dwelling at default hex %v, got %+v", defaultHex, building)
	}
	if building := gs.Map.GetHex(otherHex).Building; building != nil {
		t.Fatalf("alternate river hex %v should remain empty, got %+v", otherHex, building)
	}
}

func TestLogRiverBuildAction_UsesConfiguredReplayHex(t *testing.T) {
	gs, err := game.NewGameStateWithMap(board.MapFjords)
	if err != nil {
		t.Fatalf("NewGameStateWithMap(MapFjords) failed: %v", err)
	}
	if err := gs.AddPlayer("selkies", factions.NewSelkies()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"selkies"}

	player := gs.GetPlayer("selkies")
	player.Resources.Workers = 3
	player.Resources.Coins = 5

	landHex, ok := board.HexForDisplayCoordinate(board.MapFjords, "B2")
	if !ok {
		t.Fatal("expected B2 coordinate to exist on Fjords")
	}
	defaultHex, err := ConvertRiverCoordToAxialForMap(board.MapFjords, "R~B2")
	if err != nil {
		t.Fatalf("ConvertRiverCoordToAxialForMap(R~B2) failed: %v", err)
	}

	riverCandidates := riverNeighborsForTest(gs.Map, landHex)
	if len(riverCandidates) != 2 {
		t.Fatalf("expected 2 river candidates for R~B2, got %d", len(riverCandidates))
	}

	var configuredHex board.Hex
	for _, candidate := range riverCandidates {
		if candidate != defaultHex {
			configuredHex = candidate
			break
		}
	}

	defaultPairs := nonAdjacentLandNeighborPairsForTest(gs.Map, defaultHex)
	if len(defaultPairs) == 0 {
		t.Fatal("expected a non-adjacent support pair for the default river hex")
	}
	configuredPairs := nonAdjacentLandNeighborPairsForTest(gs.Map, configuredHex)
	if len(configuredPairs) == 0 {
		t.Fatal("expected a non-adjacent support pair for the configured river hex")
	}

	allSupport := make(map[board.Hex]bool)
	for _, pair := range defaultPairs[:1] {
		allSupport[pair[0]] = true
		allSupport[pair[1]] = true
	}
	for _, pair := range configuredPairs[:1] {
		allSupport[pair[0]] = true
		allSupport[pair[1]] = true
	}
	for hex := range allSupport {
		if err := gs.Map.TransformTerrain(hex, models.TerrainIce); err != nil {
			t.Fatalf("TransformTerrain(%v) failed: %v", hex, err)
		}
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			PlayerID:   "selkies",
			Faction:    models.FactionSelkies,
			PowerValue: 1,
		})
	}

	gs.ReplayRiverBuildHexes = map[string][]board.Hex{
		"selkies": {configuredHex},
	}
	gs.ReplayRiverBuildHexIndex = map[string]int{"selkies": 0}

	action := &LogRiverBuildAction{PlayerID: "selkies", CoordToken: "R~B2"}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogRiverBuildAction.Execute failed: %v", err)
	}

	if building := gs.Map.GetHex(configuredHex).Building; building == nil || building.PlayerID != "selkies" {
		t.Fatalf("expected configured Selkies dwelling at %v, got %+v", configuredHex, building)
	}
	if building := gs.Map.GetHex(defaultHex).Building; building != nil {
		t.Fatalf("default river hex %v should remain empty when config overrides it, got %+v", defaultHex, building)
	}
	if got := gs.ReplayRiverBuildHexIndex["selkies"]; got != 1 {
		t.Fatalf("configured replay river-build index = %d, want 1 after use", got)
	}
}

func TestLogRiverBuildAction_UsesRowCountRiverCandidateWhenLandAnchorFails(t *testing.T) {
	gs, err := game.NewGameStateWithMap(board.MapFjords)
	if err != nil {
		t.Fatalf("NewGameStateWithMap(MapFjords) failed: %v", err)
	}
	if err := gs.AddPlayer("selkies", factions.NewSelkies()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"selkies"}

	player := gs.GetPlayer("selkies")
	player.Resources.Workers = 3
	player.Resources.Coins = 5

	defaultHex, err := ConvertRiverCoordToAxialForMap(board.MapFjords, "R~B5")
	if err != nil {
		t.Fatalf("ConvertRiverCoordToAxialForMap(R~B5) failed: %v", err)
	}
	rowCountHex, err := convertRiverCoordToAxialByRowCountForMap(board.MapFjords, "R~B5")
	if err != nil {
		t.Fatalf("convertRiverCoordToAxialByRowCountForMap(R~B5) failed: %v", err)
	}
	if rowCountHex == defaultHex {
		t.Fatal("expected row-count candidate to differ from the land-anchor candidate")
	}

	for _, display := range []string{"A10", "C7"} {
		hex, ok := board.HexForDisplayCoordinate(board.MapFjords, display)
		if !ok {
			t.Fatalf("expected %s coordinate to exist on Fjords", display)
		}
		if err := gs.Map.TransformTerrain(hex, models.TerrainIce); err != nil {
			t.Fatalf("TransformTerrain(%v) failed: %v", hex, err)
		}
		gs.Map.PlaceBuilding(hex, &models.Building{
			Type:       models.BuildingDwelling,
			PlayerID:   "selkies",
			Faction:    models.FactionSelkies,
			PowerValue: 1,
		})
	}

	action := &LogRiverBuildAction{PlayerID: "selkies", CoordToken: "R~B5"}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogRiverBuildAction.Execute failed: %v", err)
	}

	if building := gs.Map.GetHex(rowCountHex).Building; building == nil || building.PlayerID != "selkies" {
		t.Fatalf("expected Selkies dwelling at row-count river hex %v, got %+v", rowCountHex, building)
	}
	if building := gs.Map.GetHex(defaultHex).Building; building != nil {
		t.Fatalf("land-anchor river hex %v should remain empty, got %+v", defaultHex, building)
	}
}

func TestLogRiverBuildAction_ConfiguredReplayHexMustBeRiver(t *testing.T) {
	gs, err := game.NewGameStateWithMap(board.MapFjords)
	if err != nil {
		t.Fatalf("NewGameStateWithMap(MapFjords) failed: %v", err)
	}
	if err := gs.AddPlayer("selkies", factions.NewSelkies()); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	gs.Phase = game.PhaseAction
	gs.TurnOrder = []string{"selkies"}

	player := gs.GetPlayer("selkies")
	player.Resources.Workers = 3
	player.Resources.Coins = 5

	landHex, ok := board.HexForDisplayCoordinate(board.MapFjords, "B5")
	if !ok {
		t.Fatal("expected B5 coordinate to exist on Fjords")
	}
	gs.ReplayRiverBuildHexes = map[string][]board.Hex{
		"selkies": {landHex},
	}
	gs.ReplayRiverBuildHexIndex = map[string]int{"selkies": 0}

	action := &LogRiverBuildAction{PlayerID: "selkies", CoordToken: "R~C4"}
	if err := action.Execute(gs); err == nil {
		t.Fatal("expected configured non-river replay hex to fail")
	}
}

func riverNeighborsForTest(m *board.TerraMysticaMap, hex board.Hex) []board.Hex {
	neighbors := make([]board.Hex, 0, 6)
	for _, neighbor := range hex.Neighbors() {
		mapHex := m.GetHex(neighbor)
		if mapHex == nil || mapHex.Terrain != models.TerrainRiver {
			continue
		}
		neighbors = append(neighbors, neighbor)
	}
	return neighbors
}

func landNeighborsForTest(m *board.TerraMysticaMap, hex board.Hex) []board.Hex {
	neighbors := make([]board.Hex, 0, 6)
	for _, neighbor := range hex.Neighbors() {
		mapHex := m.GetHex(neighbor)
		if mapHex == nil || mapHex.Terrain == models.TerrainRiver {
			continue
		}
		neighbors = append(neighbors, neighbor)
	}
	return neighbors
}

func nonAdjacentLandNeighborPairsForTest(m *board.TerraMysticaMap, hex board.Hex) [][2]board.Hex {
	neighbors := landNeighborsForTest(m, hex)
	pairs := make([][2]board.Hex, 0, len(neighbors))
	for i := 0; i < len(neighbors); i++ {
		for j := i + 1; j < len(neighbors); j++ {
			if neighbors[i].IsDirectlyAdjacent(neighbors[j]) {
				continue
			}
			pairs = append(pairs, [2]board.Hex{neighbors[i], neighbors[j]})
		}
	}
	return pairs
}
