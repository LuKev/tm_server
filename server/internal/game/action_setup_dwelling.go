package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// SetupDwellingAction represents placing an initial dwelling during game setup
// Setup dwellings are placed without cost and without adjacency requirements
type SetupDwellingAction struct {
	BaseAction
	Hex board.Hex
}

// NewSetupDwellingAction creates a new setup dwelling action
func NewSetupDwellingAction(playerID string, hex board.Hex) *SetupDwellingAction {
	return &SetupDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionSetupDwelling,
			PlayerID: playerID,
		},
		Hex: hex,
	}
}

// Validate checks if the setup dwelling placement is valid
func (a *SetupDwellingAction) Validate(gs *GameState) error {
	// Must be in setup phase
	if gs.Phase != PhaseSetup {
		return fmt.Errorf("can only place setup dwellings during setup phase")
	}

	// Replay imports can begin directly in setup without going through faction
	// selection in-engine. Initialize the strict setup sequence lazily so both
	// replay and live multiplayer share one ordering/validation path.
	if gs.SetupSubphase == SetupSubphaseNone && len(gs.SetupDwellingOrder) == 0 && len(gs.TurnOrder) > 0 && allPlayersHaveFactions(gs) {
		gs.InitializeSetupSequence()
	}
	if gs.SetupSubphase != SetupSubphaseDwellings && gs.SetupSubphase != SetupSubphaseNone {
		return fmt.Errorf("setup dwellings are only allowed during setup phase")
	}

	// Strict setup-order validation is used for live multiplayer once the sequence
	// is initialized. Replay imports may execute setup dwellings before turn order
	// is known; in that compatibility mode we only validate placement legality.
	if gs.SetupSubphase == SetupSubphaseDwellings {
		expectedPlayer := gs.currentSetupDwellingPlayerID()
		if expectedPlayer == "" {
			return fmt.Errorf("no setup dwelling placement expected")
		}
		if expectedPlayer != a.PlayerID {
			return fmt.Errorf("not your setup dwelling turn")
		}
	}

	player, exists := gs.Players[a.PlayerID]
	if !exists {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	if player.Faction == nil {
		return fmt.Errorf("player has no faction selected")
	}

	// Check if hex exists
	hexData := gs.Map.GetHex(a.Hex)
	if hexData == nil {
		return fmt.Errorf("hex does not exist: %v", a.Hex)
	}

	// Check if hex already has a building
	if hexData.Building != nil {
		return fmt.Errorf("hex already has a building")
	}

	factionType := player.Faction.GetType()
	if isIceFactionType(factionType) || isVolcanoFactionType(factionType) {
		if !isStandardLandTerrain(hexData.Terrain) {
			return fmt.Errorf("fire and ice setup dwellings must start on a standard land terrain")
		}
		if player.HasStartingTerrain && hexData.Terrain != player.StartingTerrain {
			return fmt.Errorf("setup dwelling must be placed on selected starting terrain %s", player.StartingTerrain)
		}
		return nil
	}

	if factionType == models.FactionShapeshifters || factionType == models.FactionRiverwalkers {
		if !isStandardLandTerrain(hexData.Terrain) {
			return fmt.Errorf("variable faction setup dwellings must be placed on standard land terrain")
		}
		if factionType == models.FactionRiverwalkers && !isAdjacentToRiver(gs, a.Hex) {
			return fmt.Errorf("riverwalkers setup dwellings must be adjacent to river terrain")
		}
		if player.HasStartingTerrain && hexData.Terrain != player.StartingTerrain {
			return fmt.Errorf("setup dwelling must be placed on selected home terrain %s", player.StartingTerrain)
		}
		return nil
	}

	// Cannot be river
	if hexData.Terrain == models.TerrainRiver {
		return fmt.Errorf("cannot build on river (unless faction has water building ability)")
	}

	// Must be player's home terrain
	if hexData.Terrain != player.Faction.GetHomeTerrain() {
		return fmt.Errorf("setup dwellings must be placed on home terrain")
	}

	return nil
}

// Execute places the dwelling on the map during setup
func (a *SetupDwellingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player, exists := gs.Players[a.PlayerID]
	if !exists {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	buildingType := models.BuildingDwelling
	if player.Faction.GetType() == models.FactionAtlanteans {
		if gs.SetupPlacedDwellings[a.PlayerID] > 0 {
			return fmt.Errorf("atlanteans only place their starting stronghold once during setup")
		}
		buildingType = models.BuildingStronghold
	}

	// Place the starting structure on the map
	hexData := gs.Map.GetHex(a.Hex)
	factionType := player.Faction.GetType()
	if (isIceFactionType(factionType) || isVolcanoFactionType(factionType)) && hexData.Terrain != player.Faction.GetHomeTerrain() {
		if !player.HasStartingTerrain {
			player.StartingTerrain = hexData.Terrain
			player.HasStartingTerrain = true
		}
		hexData.Terrain = player.Faction.GetHomeTerrain()
	} else if factionType == models.FactionShapeshifters || factionType == models.FactionRiverwalkers {
		if !player.HasStartingTerrain {
			player.StartingTerrain = hexData.Terrain
			player.HasStartingTerrain = true
			if factionType == models.FactionRiverwalkers {
				player.UnlockedTerrains = map[models.TerrainType]bool{hexData.Terrain: true}
			}
		}
	}
	hexData.Building = &models.Building{
		Type:       buildingType,
		PlayerID:   a.PlayerID,
		Faction:    player.Faction.GetType(),
		PowerValue: getStructurePowerValue(player, buildingType),
	}

	if buildingType == models.BuildingStronghold && player.Faction.GetType() == models.FactionAtlanteans {
		player.AtlanteansTownHexes = []board.Hex{a.Hex}
		player.AtlanteansTownRewards = make(map[int]bool)
		gs.PendingTownFormations[a.PlayerID] = append(gs.PendingTownFormations[a.PlayerID], &PendingTownFormation{
			PlayerID: a.PlayerID,
			Hexes:    []board.Hex{a.Hex},
		})
	}

	if gs.SetupPlacedDwellings == nil {
		gs.SetupPlacedDwellings = make(map[string]int)
	}
	gs.SetupPlacedDwellings[a.PlayerID]++

	if gs.SetupSubphase == SetupSubphaseDwellings && len(gs.SetupDwellingOrder) > 0 {
		gs.AdvanceSetupAfterDwelling()
	}

	return nil
}
