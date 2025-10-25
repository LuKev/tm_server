package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestGetPowerValue(t *testing.T) {
	tests := []struct {
		building models.BuildingType
		expected int
	}{
		{models.BuildingDwelling, 1},
		{models.BuildingTradingHouse, 2},
		{models.BuildingTemple, 2},
		{models.BuildingSanctuary, 3},
		{models.BuildingStronghold, 3},
	}

	for _, tt := range tests {
		t.Run(string(tt.building), func(t *testing.T) {
			result := GetPowerValue(tt.building)
			if result != tt.expected {
				t.Errorf("GetPowerValue(%v) = %d, want %d", tt.building, result, tt.expected)
			}
		})
	}
}

func TestCalculateAdjacencyBonus(t *testing.T) {
	m := NewTerraMysticaMap()
	faction1 := models.FactionNomads  // Desert (Yellow)
	faction2 := models.FactionGiants  // Wasteland (Gray)

	// Place building at (0,0)
	h := NewHex(0, 0)

	// Place opponent buildings adjacent
	neighbors := m.GetDirectNeighbors(h)
	if len(neighbors) < 2 {
		t.Skip("not enough neighbors for test")
	}

	// Place 2 opponent buildings
	m.PlaceBuilding(neighbors[0], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2,
		PowerValue: 1,
	})
	m.PlaceBuilding(neighbors[1], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2,
		PowerValue: 1,
	})

	// Calculate adjacency bonus
	bonus := m.CalculateAdjacencyBonus(h, faction1)
	if bonus != 2 {
		t.Errorf("expected adjacency bonus of 2, got %d", bonus)
	}

	// Place own faction building
	if len(neighbors) > 2 {
		m.PlaceBuilding(neighbors[2], &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction1,
			PowerValue: 1,
		})

		// Bonus should still be 2 (own buildings don't count)
		bonus = m.CalculateAdjacencyBonus(h, faction1)
		if bonus != 2 {
			t.Errorf("expected adjacency bonus of 2 (own buildings don't count), got %d", bonus)
		}
	}
}

func TestGetPowerLeechTargets(t *testing.T) {
	m := NewTerraMysticaMap()
	faction1 := models.FactionNomads   // Desert (Yellow)
	faction2 := models.FactionWitches  // Forest (Green)
	faction3 := models.FactionGiants   // Wasteland (Gray)

	h := NewHex(0, 0)
	neighbors := m.GetDirectNeighbors(h)
	if len(neighbors) < 3 {
		t.Skip("not enough neighbors for test")
	}

	// Place buildings from 2 different opponent factions
	m.PlaceBuilding(neighbors[0], &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction2,
		PowerValue: 1,
	})
	m.PlaceBuilding(neighbors[1], &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    faction3,
		PowerValue: 2,
	})

	// Place a temple (power value 2) at h
	powerValue := 2
	targets := m.GetPowerLeechTargets(h, faction1, powerValue)

	// Both opponents should be able to leech 2 power
	if len(targets) != 2 {
		t.Errorf("expected 2 leech targets, got %d", len(targets))
	}

	if targets[faction2] != 2 {
		t.Errorf("expected faction2 to leech 2 power, got %d", targets[faction2])
	}

	if targets[faction3] != 2 {
		t.Errorf("expected faction3 to leech 2 power, got %d", targets[faction3])
	}

	// Own faction should not be in targets
	if _, ok := targets[faction1]; ok {
		t.Errorf("own faction should not be able to leech power")
	}
}

func TestTownFormation_5PointsTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Use helper to set up 4 connected buildings with power = 7
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	// Check that town can be formed
	connected := gs.CheckForTownFormation("player1", hexes[0])
	if connected == nil {
		t.Fatal("expected town to be formable")
	}
	
	// Form town with 5 points tile
	initialVP := player.VictoryPoints
	initialCoins := player.Resources.Coins
	initialKeys := player.Keys
	
	err := gs.FormTown("player1", connected, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +5 VP, +6 coins, +1 key
	if player.VictoryPoints != initialVP+5 {
		t.Errorf("expected %d VP, got %d", initialVP+5, player.VictoryPoints)
	}
	if player.Resources.Coins != initialCoins+6 {
		t.Errorf("expected %d coins, got %d", initialCoins+6, player.Resources.Coins)
	}
	if player.Keys != initialKeys+1 {
		t.Errorf("expected %d keys, got %d", initialKeys+1, player.Keys)
	}
	if player.TownsFormed != 1 {
		t.Errorf("expected 1 town formed, got %d", player.TownsFormed)
	}
	
	// Verify buildings are marked as part of town
	for _, h := range connected {
		if !gs.Map.GetHex(h).PartOfTown {
			t.Errorf("building at %v should be marked as part of town", h)
		}
	}
	
	// Verify tile is no longer available
	if gs.TownTiles.Available[TownTile5Points] != 1 {
		t.Errorf("expected 1 copy of 5 points tile remaining, got %d", gs.TownTiles.Available[TownTile5Points])
	}
}

func TestTownFormation_6PointsTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up 4 connected buildings
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	// Form town with 6 points tile
	initialVP := player.VictoryPoints
	// Set up power for gaining (need power in Bowl1 to gain)
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0
	initialKeys := player.Keys
	
	err := gs.FormTown("player1", hexes, TownTile6Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +6 VP, +8 power, +1 key
	if player.VictoryPoints != initialVP+6 {
		t.Errorf("expected %d VP, got %d", initialVP+6, player.VictoryPoints)
	}
	if player.Resources.Power.Bowl2 != 8 {
		t.Errorf("expected 8 power in bowl2, got %d", player.Resources.Power.Bowl2)
	}
	if player.Keys != initialKeys+1 {
		t.Errorf("expected %d keys, got %d", initialKeys+1, player.Keys)
	}
}

func TestTownFormation_7PointsTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	initialVP := player.VictoryPoints
	initialWorkers := player.Resources.Workers
	initialKeys := player.Keys
	
	err := gs.FormTown("player1", hexes, TownTile7Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +7 VP, +2 workers, +1 key
	if player.VictoryPoints != initialVP+7 {
		t.Errorf("expected %d VP, got %d", initialVP+7, player.VictoryPoints)
	}
	if player.Resources.Workers != initialWorkers+2 {
		t.Errorf("expected %d workers, got %d", initialWorkers+2, player.Resources.Workers)
	}
	if player.Keys != initialKeys+1 {
		t.Errorf("expected %d keys, got %d", initialKeys+1, player.Keys)
	}
}

func TestTownFormation_8PointsTile_CultAdvancement(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up power for cult advancement
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	initialVP := player.VictoryPoints
	initialKeys := player.Keys
	
	err := gs.FormTown("player1", hexes, TownTile8Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +8 VP, +1 key, +1 on all cult tracks
	if player.VictoryPoints != initialVP+8 {
		t.Errorf("expected %d VP, got %d", initialVP+8, player.VictoryPoints)
	}
	if player.Keys != initialKeys+1 {
		t.Errorf("expected %d keys, got %d", initialKeys+1, player.Keys)
	}
	
	// Verify cult advancement (+1 on all tracks)
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		pos := gs.CultTracks.GetPosition("player1", track)
		if pos != 1 {
			t.Errorf("expected position 1 on %v, got %d", track, pos)
		}
	}
}

func TestTownFormation_9PointsTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	initialVP := player.VictoryPoints
	initialPriests := player.Resources.Priests
	initialKeys := player.Keys
	
	err := gs.FormTown("player1", hexes, TownTile9Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +9 VP, +1 priest, +1 key
	if player.VictoryPoints != initialVP+9 {
		t.Errorf("expected %d VP, got %d", initialVP+9, player.VictoryPoints)
	}
	if player.Resources.Priests != initialPriests+1 {
		t.Errorf("expected %d priests, got %d", initialPriests+1, player.Resources.Priests)
	}
	if player.Keys != initialKeys+1 {
		t.Errorf("expected %d keys, got %d", initialKeys+1, player.Keys)
	}
}

func TestTownFormation_11PointsTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	initialVP := player.VictoryPoints
	initialKeys := player.Keys
	
	err := gs.FormTown("player1", hexes, TownTile11Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +11 VP, +1 key
	if player.VictoryPoints != initialVP+11 {
		t.Errorf("expected %d VP, got %d", initialVP+11, player.VictoryPoints)
	}
	if player.Keys != initialKeys+1 {
		t.Errorf("expected %d keys, got %d", initialKeys+1, player.Keys)
	}
	
	// Verify only 1 copy exists
	if gs.TownTiles.Available[TownTile11Points] != 0 {
		t.Errorf("expected 0 copies of 11 points tile remaining, got %d", gs.TownTiles.Available[TownTile11Points])
	}
}

func TestTownFormation_2PointsTile_CultAdvancement(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up power for cult advancement
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	initialVP := player.VictoryPoints
	initialKeys := player.Keys
	
	err := gs.FormTown("player1", hexes, TownTile2Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +2 VP, +2 keys, +2 on all cult tracks
	if player.VictoryPoints != initialVP+2 {
		t.Errorf("expected %d VP, got %d", initialVP+2, player.VictoryPoints)
	}
	if player.Keys != initialKeys+2 {
		t.Errorf("expected %d keys, got %d", initialKeys+2, player.Keys)
	}
	
	// Verify cult advancement (+2 on all tracks)
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		pos := gs.CultTracks.GetPosition("player1", track)
		if pos != 2 {
			t.Errorf("expected position 2 on %v, got %d", track, pos)
		}
	}
	
	// Verify only 1 copy exists
	if gs.TownTiles.Available[TownTile2Points] != 0 {
		t.Errorf("expected 0 copies of 2 points tile remaining, got %d", gs.TownTiles.Available[TownTile2Points])
	}
}

func TestTownFormation_WithSanctuary_3Buildings(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up 3 connected buildings with sanctuary (power = 3+2+2 = 7)
	// Use hexes from row 0: (0,0), (1,0), (2,0)
	hexes := []Hex{
		NewHex(0, 0), // Plains
		NewHex(1, 0), // Mountain
		NewHex(2, 0), // Forest
	}
	
	// Sanctuary + 2 trading houses
	gs.Map.PlaceBuilding(hexes[0], &models.Building{
		Type:       models.BuildingSanctuary,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 3,
	})
	gs.Map.GetHex(hexes[0]).Terrain = faction.GetHomeTerrain()
	
	for _, h := range hexes[1:] {
		gs.Map.PlaceBuilding(h, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		})
		gs.Map.GetHex(h).Terrain = faction.GetHomeTerrain()
	}
	
	// Check that town can be formed with only 3 buildings (sanctuary allows this)
	connected := gs.CheckForTownFormation("player1", hexes[0])
	if connected == nil {
		t.Fatal("expected town to be formable with sanctuary + 3 buildings")
	}
	
	if len(connected) != 3 {
		t.Errorf("expected 3 buildings in town, got %d", len(connected))
	}
	
	// Form town
	err := gs.FormTown("player1", connected, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	if player.TownsFormed != 1 {
		t.Errorf("expected 1 town formed, got %d", player.TownsFormed)
	}
}

func TestTownFormation_Fire2FavorTile_PowerRequirement6(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player Fire 2 favor tile
	// Manually add to player's tiles (simulating they selected it)
	gs.FavorTiles.PlayerTiles["player1"] = []FavorTileType{FavorFire2}
	
	// Use helper to set up 4 connected buildings with power = 6 (not normally enough)
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 6)
	
	// Check that town can be formed with power = 6 (Fire 2 reduces requirement)
	connected := gs.CheckForTownFormation("player1", hexes[0])
	if connected == nil {
		t.Fatal("expected town to be formable with Fire 2 favor tile (power = 6)")
	}
	
	// Form town
	err := gs.FormTown("player1", connected, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	if player.TownsFormed != 1 {
		t.Errorf("expected 1 town formed, got %d", player.TownsFormed)
	}
}

func TestTownFormation_MultipleTownsInSameTurn(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player Fire 2 favor tile to make it easier
	// Manually add to player's tiles (simulating they selected it)
	gs.FavorTiles.PlayerTiles["player1"] = []FavorTileType{FavorFire2}
	
	// Set up two separate clusters of buildings
	// Cluster 1: row 0, hexes 0-3 (Plains, Mountain, Forest, Lake)
	cluster1 := setupConnectedBuildings(gs, "player1", faction, 4, 6)
	
	// Cluster 2: row 0, hexes 9-12 (Forest, Lake, Wasteland, Swamp) - far from cluster 1
	cluster2 := []Hex{
		NewHex(9, 0),  // Forest
		NewHex(10, 0), // Lake
		NewHex(11, 0), // Wasteland
		NewHex(12, 0), // Swamp
	}
	
	// Build cluster 2 (power = 6): 2 dwellings + 2 trading houses
	for _, h := range cluster2[:2] {
		gs.Map.PlaceBuilding(h, &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 1,
		})
		gs.Map.GetHex(h).Terrain = faction.GetHomeTerrain()
	}
	for _, h := range cluster2[2:] {
		gs.Map.PlaceBuilding(h, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		})
		gs.Map.GetHex(h).Terrain = faction.GetHomeTerrain()
	}
	
	// Form first town
	connected1 := gs.CheckForTownFormation("player1", cluster1[0])
	if connected1 == nil {
		t.Fatal("expected first town to be formable")
	}
	
	err := gs.FormTown("player1", connected1, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form first town: %v", err)
	}
	
	if player.TownsFormed != 1 {
		t.Errorf("expected 1 town formed, got %d", player.TownsFormed)
	}
	
	// Form second town
	connected2 := gs.CheckForTownFormation("player1", cluster2[0])
	if connected2 == nil {
		t.Fatal("expected second town to be formable")
	}
	
	err = gs.FormTown("player1", connected2, TownTile6Points, nil)
	if err != nil {
		t.Fatalf("failed to form second town: %v", err)
	}
	
	if player.TownsFormed != 2 {
		t.Errorf("expected 2 towns formed, got %d", player.TownsFormed)
	}
	
	// Verify both clusters are marked as towns
	for _, h := range connected1 {
		if !gs.Map.GetHex(h).PartOfTown {
			t.Errorf("cluster 1 building at %v should be marked as part of town", h)
		}
	}
	for _, h := range connected2 {
		if !gs.Map.GetHex(h).PartOfTown {
			t.Errorf("cluster 2 building at %v should be marked as part of town", h)
		}
	}
}

func TestTownFormation_CannotReformSameTown(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	// Form town
	err := gs.FormTown("player1", hexes, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Try to check for town formation again - should return nil
	connected := gs.CheckForTownFormation("player1", hexes[0])
	if connected != nil {
		t.Error("should not be able to form town from buildings already in a town")
	}
}

func TestTownFormation_WithBridges(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up buildings that would be connected via bridge
	// Use hexes that are naturally adjacent: (0,0), (1,0), (2,0), (3,0)
	// These are already adjacent, so the bridge test is really testing that
	// bridges don't break the connection logic
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	// Build a bridge between two of them (even though they're already adjacent)
	// This tests that bridges are properly handled in the connection logic
	gs.Map.BuildBridge(hexes[1], hexes[2])
	
	// Check that town can still be formed with the bridge present
	connected := gs.CheckForTownFormation("player1", hexes[0])
	if connected == nil {
		t.Fatalf("expected town to be formable with buildings (bridge present)")
	}
	
	if len(connected) != 4 {
		t.Errorf("expected 4 buildings in town, got %d", len(connected))
	}
	
	// Form town
	err := gs.FormTown("player1", connected, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	if player.TownsFormed != 1 {
		t.Errorf("expected 1 town formed, got %d", player.TownsFormed)
	}
}

func TestTownFormation_WitchesBonus(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewWitches()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	initialVP := player.VictoryPoints
	
	err := gs.FormTown("player1", hexes, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify benefits: +5 VP from tile + 5 VP from Witches faction bonus = +10 VP total
	if player.VictoryPoints != initialVP+10 {
		t.Errorf("expected %d VP (5 from tile + 5 from Witches), got %d", initialVP+10, player.VictoryPoints)
	}
}

func TestTownFormation_SwarmlingsBonus(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewSwarmlings()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	initialWorkers := player.Resources.Workers
	
	err := gs.FormTown("player1", hexes, TownTile5Points, nil)
	if err != nil {
		t.Fatalf("failed to form town: %v", err)
	}
	
	// Verify Swarmlings get +3 workers per town
	if player.Resources.Workers != initialWorkers+3 {
		t.Errorf("expected %d workers (+3 from Swarmlings), got %d", initialWorkers+3, player.Resources.Workers)
	}
}

func TestSelectTownTile(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Set up buildings that can form a town
	hexes := setupConnectedBuildings(gs, "player1", faction, 4, 7)
	
	// Create a pending town formation
	gs.PendingTownFormations["player1"] = &PendingTownFormation{
		PlayerID: "player1",
		Hexes:    hexes,
	}
	
	// Verify pending town exists
	if gs.PendingTownFormations["player1"] == nil {
		t.Fatal("expected pending town formation")
	}
	
	// Select town tile
	err := gs.SelectTownTile("player1", TownTile7Points)
	if err != nil {
		t.Fatalf("failed to select town tile: %v", err)
	}
	
	// Verify town was formed
	if player.TownsFormed != 1 {
		t.Errorf("expected 1 town formed, got %d", player.TownsFormed)
	}
	
	// Verify pending town was removed
	if gs.PendingTownFormations["player1"] != nil {
		t.Error("expected pending town formation to be removed")
	}
	
	// Verify tile was taken
	if gs.TownTiles.Available[TownTile7Points] != 1 {
		t.Errorf("expected 1 copy of 7 points tile remaining, got %d", gs.TownTiles.Available[TownTile7Points])
	}
}

func TestSelectTownTile_NoPendingTown(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	
	// Try to select town tile without pending town
	err := gs.SelectTownTile("player1", TownTile5Points)
	if err == nil {
		t.Error("expected error when no pending town formation")
	}
}

func TestBuildActionCreatesPendingTown(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")
	
	// Give player resources
	player.Resources.Coins = 100
	player.Resources.Workers = 100
	
	// Build 3 trading houses first (manually)
	hexes := []Hex{
		NewHex(0, 0),
		NewHex(1, 0),
		NewHex(2, 0),
	}
	
	for _, h := range hexes {
		gs.Map.PlaceBuilding(h, &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   "player1",
			PowerValue: 2,
		})
		gs.Map.GetHex(h).Terrain = faction.GetHomeTerrain()
	}
	
	// Build 4th trading house which should trigger town formation
	action := NewUpgradeBuildingAction("player1", NewHex(3, 0), models.BuildingTradingHouse)
	
	// First need to place a dwelling there
	gs.Map.PlaceBuilding(NewHex(3, 0), &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    faction.GetType(),
		PlayerID:   "player1",
		PowerValue: 1,
	})
	gs.Map.GetHex(NewHex(3, 0)).Terrain = faction.GetHomeTerrain()
	
	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("failed to upgrade building: %v", err)
	}
	
	// Verify pending town formation was created
	if gs.PendingTownFormations["player1"] == nil {
		t.Fatal("expected pending town formation after building 4th trading house")
	}
	
	if len(gs.PendingTownFormations["player1"].Hexes) != 4 {
		t.Errorf("expected 4 hexes in pending town, got %d", len(gs.PendingTownFormations["player1"].Hexes))
	}
}

// Helper function to set up connected buildings for testing
// Uses valid hexes from row 0 of the base map: (0,0), (1,0), (2,0), (3,0) are all adjacent
func setupConnectedBuildings(gs *GameState, playerID string, faction factions.Faction, count int, totalPower int) []Hex {
	// Use hexes from row 0 which are adjacent: (0,0), (1,0), (2,0), (3,0)
	hexes := []Hex{
		NewHex(0, 0), // Plains
		NewHex(1, 0), // Mountain
		NewHex(2, 0), // Forest
		NewHex(3, 0), // Lake
	}
	hexes = hexes[:count]
	
	// Distribute power across buildings to reach totalPower
	// Use trading houses (power 2) and dwellings (power 1)
	tradingHouses := totalPower / 2
	if tradingHouses > count {
		tradingHouses = count
	}
	dwellings := count - tradingHouses
	
	// Adjust if we don't have enough power
	for tradingHouses*2+dwellings < totalPower && dwellings > 0 {
		dwellings--
		tradingHouses++
	}
	
	// Place trading houses
	for i := 0; i < tradingHouses; i++ {
		gs.Map.PlaceBuilding(hexes[i], &models.Building{
			Type:       models.BuildingTradingHouse,
			Faction:    faction.GetType(),
			PlayerID:   playerID,
			PowerValue: 2,
		})
		gs.Map.GetHex(hexes[i]).Terrain = faction.GetHomeTerrain()
	}
	
	// Place dwellings
	for i := tradingHouses; i < count; i++ {
		gs.Map.PlaceBuilding(hexes[i], &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    faction.GetType(),
			PlayerID:   playerID,
			PowerValue: 1,
		})
		gs.Map.GetHex(hexes[i]).Terrain = faction.GetHomeTerrain()
	}
	
	return hexes
}
