package game

import (
	"fmt"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ActionType represents the type of action a player can take
type ActionType int

const (
	ActionTransformAndBuild ActionType = iota
	ActionUpgradeBuilding
	ActionAdvanceShipping
	ActionAdvanceDigging
	ActionSendPriestToCult
	ActionPowerAction
	ActionSpecialAction
	ActionPass
	ActionSetupDwelling        // Place initial dwelling during setup (no cost, no adjacency)
	ActionUseCultSpade         // Use a spade from cult track reward (cleanup phase)
	ActionAcceptPowerLeech     // Accept a power leech offer
	ActionDeclinePowerLeech    // Decline a power leech offer
	ActionSelectFavorTile      // Select a favor tile after Temple/Sanctuary/Auren Stronghold
	ActionApplyHalflingsSpade  // Apply one of 3 stronghold spades (Halflings only)
	ActionBuildHalflingsDwelling // Build dwelling on transformed hex (Halflings optional)
	ActionSkipHalflingsDwelling  // Skip optional dwelling (Halflings)
	ActionUseDarklingsPriestOrdination // Convert 0-3 workers to priests (Darklings stronghold, one-time)
	ActionSelectCultistsCultTrack // Select cult track for power leech bonus (Cultists only)
)

// Action represents a player action
type Action interface {
	GetType() ActionType
	GetPlayerID() string
	Validate(gs *GameState) error
	Execute(gs *GameState) error
}

// BaseAction provides common fields for all actions
type BaseAction struct {
	Type     ActionType
	PlayerID string
}

func (a *BaseAction) GetType() ActionType {
	return a.Type
}

func (a *BaseAction) GetPlayerID() string {
	return a.PlayerID
}

// TransformAndBuildAction represents terraforming a hex and optionally building a dwelling
// Per rulebook: "First, you may change the type of one Terrain space. Then, if you have 
// changed its type to your Home terrain, you may immediately build a Dwelling on that space."
type TransformAndBuildAction struct {
	BaseAction
	TargetHex      Hex
	BuildDwelling  bool // Whether to build a dwelling after transforming
	UseSkip        bool // Fakirs carpet flight / Dwarves tunneling - skip adjacency for one space
}

func NewTransformAndBuildAction(playerID string, targetHex Hex, buildDwelling bool) *TransformAndBuildAction {
	return &TransformAndBuildAction{
		BaseAction: BaseAction{
			Type:     ActionTransformAndBuild,
			PlayerID: playerID,
		},
		TargetHex:      targetHex,
		BuildDwelling:  buildDwelling,
		UseSkip:        false,
	}
}

// NewTransformAndBuildActionWithSkip creates a transform action with carpet flight/tunneling
func NewTransformAndBuildActionWithSkip(playerID string, targetHex Hex, buildDwelling bool) *TransformAndBuildAction {
	return &TransformAndBuildAction{
		BaseAction: BaseAction{
			Type:     ActionTransformAndBuild,
			PlayerID: playerID,
		},
		TargetHex:      targetHex,
		BuildDwelling:  buildDwelling,
		UseSkip:        true,
	}
}

func (a *TransformAndBuildAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if hex exists and is empty (no building)
	mapHex := gs.Map.GetHex(a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}
	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building: %v", a.TargetHex)
	}

	// Check adjacency - required for both transforming and building
	// "Even when transforming a Terrain space without building a Dwelling, the transformed 
	// Terrain space needs to be directly or indirectly adjacent to one of your Structures"
	isAdjacent := gs.IsAdjacentToPlayerBuilding(a.TargetHex, a.PlayerID)
	
	// If using skip (Fakirs/Dwarves), check if player can skip and if range is valid
	if a.UseSkip {
		// Check if player's faction can use skip ability
		canSkip := false
		if fakirs, ok := player.Faction.(*factions.Fakirs); ok {
			canSkip = fakirs.CanCarpetFlight()
			// Check if target is within skip range
			skipRange := fakirs.GetCarpetFlightRange()
			if !gs.Map.IsWithinSkipRange(a.TargetHex, a.PlayerID, skipRange) {
				return fmt.Errorf("target hex is not within carpet flight range %d", skipRange)
			}
			// Check if player has priest to pay
			if player.Resources.Priests < 1 {
				return fmt.Errorf("not enough priests for carpet flight: need 1, have %d", player.Resources.Priests)
			}
		} else if dwarves, ok := player.Faction.(*factions.Dwarves); ok {
			canSkip = dwarves.CanTunnel()
			// Dwarves can tunnel 1 space
			if !gs.Map.IsWithinSkipRange(a.TargetHex, a.PlayerID, 1) {
				return fmt.Errorf("target hex is not within tunneling range 1")
			}
			// Check if player has workers to pay
			workerCost := dwarves.GetTunnelingCost()
			// This cost is in addition to transform/dwelling costs
			// Just verify they have it here, will deduct later
			if player.Resources.Workers < workerCost {
				return fmt.Errorf("not enough workers for tunneling: need %d, have %d", workerCost, player.Resources.Workers)
			}
		} else {
			return fmt.Errorf("only Fakirs and Dwarves can use skip ability")
		}
		
		if !canSkip {
			return fmt.Errorf("player cannot use skip ability")
		}
	} else {
		// Normal adjacency required if not using skip
		if !isAdjacent {
			return fmt.Errorf("hex is not adjacent to player's buildings")
		}
	}

	// Check if terrain needs transformation to home terrain
	needsTransform := mapHex.Terrain != player.Faction.GetHomeTerrain()
	
	totalWorkersNeeded := 0
	totalPriestsNeeded := 0
	
	if needsTransform {
		// Calculate terraform cost
		distance := gs.Map.GetTerrainDistance(mapHex.Terrain, player.Faction.GetHomeTerrain())
		if distance == 0 {
			return fmt.Errorf("terrain distance calculation failed")
		}
		
		// Darklings pay priests for terraform (1 priest per spade)
		if darklings, ok := player.Faction.(*factions.Darklings); ok {
			totalPriestsNeeded = darklings.GetTerraformCostInPriests(distance)
		} else {
			// Other factions pay workers
			// GetTerraformCost returns total workers needed (already accounts for distance)
			totalWorkersNeeded = player.Faction.GetTerraformCost(distance)
		}
	}
	
	// Add tunneling cost to total if using skip (Dwarves)
	if a.UseSkip {
		if dwarves, ok := player.Faction.(*factions.Dwarves); ok {
			totalWorkersNeeded += dwarves.GetTunnelingCost()
		}
	}

	// If building a dwelling, check requirements
	if a.BuildDwelling {
		// Check building limit (max 8 dwellings)
		if err := checkBuildingLimit(gs, a.PlayerID, models.BuildingDwelling); err != nil {
			return err
		}
		
		// After transformation (if any), hex must be player's home terrain
		if needsTransform {
			// Will be home terrain after transform
		} else if mapHex.Terrain != player.Faction.GetHomeTerrain() {
			return fmt.Errorf("cannot build dwelling: hex is not home terrain")
		}
		
		// Check if player can afford dwelling (coins and priests)
		dwellingCost := player.Faction.GetDwellingCost()
		if player.Resources.Coins < dwellingCost.Coins {
			return fmt.Errorf("not enough coins for dwelling: need %d, have %d", dwellingCost.Coins, player.Resources.Coins)
		}
		if player.Resources.Priests < dwellingCost.Priests {
			return fmt.Errorf("not enough priests for dwelling: need %d, have %d", dwellingCost.Priests, player.Resources.Priests)
		}
		
		// Add dwelling workers to total needed (checked separately below)
		totalWorkersNeeded += dwellingCost.Workers
	}
	
	// Check total workers needed (terraform + dwelling)
	if player.Resources.Workers < totalWorkersNeeded {
		return fmt.Errorf("not enough workers: need %d, have %d", totalWorkersNeeded, player.Resources.Workers)
	}
	
	// Check total priests needed (Darklings terraform cost)
	if player.Resources.Priests < totalPriestsNeeded {
		return fmt.Errorf("not enough priests for terraform: need %d, have %d", totalPriestsNeeded, player.Resources.Priests)
	}

	return nil
}

func (a *TransformAndBuildAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	mapHex := gs.Map.GetHex(a.TargetHex)

	// Step 0: Handle skip costs (Fakirs carpet flight / Dwarves tunneling)
	if a.UseSkip {
		if fakirs, ok := player.Faction.(*factions.Fakirs); ok {
			// Pay priest for carpet flight
			player.Resources.Priests -= 1
			// Award VP bonus
			player.VictoryPoints += fakirs.GetCarpetFlightVPBonus()
		} else if dwarves, ok := player.Faction.(*factions.Dwarves); ok {
			// Pay workers for tunneling
			workerCost := dwarves.GetTunnelingCost()
			player.Resources.Workers -= workerCost
			// Award VP bonus
			player.VictoryPoints += dwarves.GetTunnelingVPBonus()
		}
	}

	// Step 1: Transform terrain to home terrain if needed
	needsTransform := mapHex.Terrain != player.Faction.GetHomeTerrain()
	fmt.Printf("DEBUG TransformAndBuild: %s building on %v, terrain=%v, home=%v, needsTransform=%v\n",
		a.PlayerID, a.TargetHex, mapHex.Terrain, player.Faction.GetHomeTerrain(), needsTransform)
	if needsTransform {
		distance := gs.Map.GetTerrainDistance(mapHex.Terrain, player.Faction.GetHomeTerrain())

		// Darklings pay priests for terraform (instead of workers)
		if darklings, ok := player.Faction.(*factions.Darklings); ok {
			priestCost := darklings.GetTerraformCostInPriests(distance)
			fmt.Printf("DEBUG: %s paying %d priests for %d spades of terraform\n", a.PlayerID, priestCost, distance)
			player.Resources.Priests -= priestCost
			
			// Award Darklings VP bonus (+2 VP per spade)
			vpBonus := darklings.GetTerraformVPBonus(distance)
			player.VictoryPoints += vpBonus
		} else {
			// Other factions pay workers
			totalWorkers := player.Faction.GetTerraformCost(distance)
			fmt.Printf("DEBUG: %s paying %d workers for %d spades of terraform (before: %d W, after: %d W)\n",
				a.PlayerID, totalWorkers, distance, player.Resources.Workers, player.Resources.Workers-totalWorkers)
			player.Resources.Workers -= totalWorkers
		}

		// Transform terrain to home terrain
		if err := gs.Map.TransformTerrain(a.TargetHex, player.Faction.GetHomeTerrain()); err != nil {
			return fmt.Errorf("failed to transform terrain: %w", err)
		}
		
		// Award VP from scoring tile (per spade used)
		// Note: Darklings get their own VP bonus above, not from scoring tiles or Halflings
		if _, isDarklings := player.Faction.(*factions.Darklings); !isDarklings {
			spadesUsed := player.Faction.GetTerraformSpades(distance)
			for i := 0; i < spadesUsed; i++ {
				gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
			}
			
			// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
			if halflings, ok := player.Faction.(*factions.Halflings); ok {
				vpBonus := halflings.GetVPPerSpade() * spadesUsed
				player.VictoryPoints += vpBonus
			}
			
			// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
			if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
				powerBonus := alchemists.GetPowerPerSpade() * spadesUsed
				if powerBonus > 0 {
					player.Resources.Power.Bowl1 += powerBonus
				}
			}
		}
	}

	// Step 2: Build dwelling if requested
	if a.BuildDwelling {
		// Pay for dwelling
		dwellingCost := player.Faction.GetDwellingCost()
		if err := player.Resources.Spend(dwellingCost); err != nil {
			return fmt.Errorf("failed to pay for dwelling: %w", err)
		}

		// Place dwelling
		dwelling := &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    player.Faction.GetType(),
			PlayerID:   a.PlayerID,
			PowerValue: 1, // Dwellings provide 1 power to neighbors
		}
		mapHex.Building = dwelling

		// Award VP from Earth+1 favor tile (+2 VP when building Dwelling)
		playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
		if HasFavorTile(playerTiles, FavorEarth1) {
			player.VictoryPoints += 2
		}

		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionDwelling)

		// Trigger power leech for adjacent players
		gs.TriggerPowerLeech(a.TargetHex, a.PlayerID)
		
		// Check for town formation after building dwelling
		// CheckForTownFormation handles creating PendingTownFormation if needed
		gs.CheckForTownFormation(a.PlayerID, a.TargetHex)
	}

	return nil
}

// UpgradeBuildingAction represents upgrading a building
type UpgradeBuildingAction struct {
	BaseAction
	TargetHex      Hex
	NewBuildingType models.BuildingType
}

func NewUpgradeBuildingAction(playerID string, targetHex Hex, newType models.BuildingType) *UpgradeBuildingAction {
	return &UpgradeBuildingAction{
		BaseAction: BaseAction{
			Type:     ActionUpgradeBuilding,
			PlayerID: playerID,
		},
		TargetHex:      targetHex,
		NewBuildingType: newType,
	}
}

func (a *UpgradeBuildingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if hex has player's building
	mapHex := gs.Map.GetHex(a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}
	if mapHex.Building == nil {
		return fmt.Errorf("no building at hex: %v", a.TargetHex)
	}
	if mapHex.Building.PlayerID != a.PlayerID {
		return fmt.Errorf("building does not belong to player")
	}

	// Validate upgrade path
	if !isValidUpgrade(mapHex.Building.Type, a.NewBuildingType) {
		return fmt.Errorf("invalid upgrade: cannot upgrade %v to %v", mapHex.Building.Type, a.NewBuildingType)
	}

	// Check building limits
	if err := checkBuildingLimit(gs, a.PlayerID, a.NewBuildingType); err != nil {
		return err
	}

	// Get upgrade cost (may be reduced if adjacent to opponent)
	cost := getUpgradeCost(gs, player, mapHex, a.NewBuildingType)

	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford upgrade to %v", a.NewBuildingType)
	}

	return nil
}

func (a *UpgradeBuildingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	mapHex := gs.Map.GetHex(a.TargetHex)

	// Get upgrade cost (may be reduced if adjacent to opponent)
	cost := getUpgradeCost(gs, player, mapHex, a.NewBuildingType)

	// Pay for upgrade
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for upgrade: %w", err)
	}

	// Return old building to faction board (reduces income)
	// Buildings are returned to the rightmost position on their track
	// This is handled by the faction board state (not implemented yet)

	// Get new power value
	var newPowerValue int
	switch a.NewBuildingType {
	case models.BuildingTradingHouse:
		newPowerValue = 2
	case models.BuildingTemple:
		newPowerValue = 2
	case models.BuildingSanctuary:
		newPowerValue = 3
	case models.BuildingStronghold:
		newPowerValue = 3
	}

	// Upgrade building
	mapHex.Building = &models.Building{
		Type:       a.NewBuildingType,
		Faction:    player.Faction.GetType(),
		PlayerID:   a.PlayerID,
		PowerValue: newPowerValue,
	}

	// Handle special rewards based on upgrade type
	switch a.NewBuildingType {
	case models.BuildingTradingHouse:
		// Award VP from Water+1 favor tile (+3 VP when upgrading Dwelling→Trading House)
		playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
		if HasFavorTile(playerTiles, FavorWater1) {
			player.VictoryPoints += 3
		}
		
		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionTradingHouse)
		break
	case models.BuildingTemple, models.BuildingSanctuary:
		// Player must select a Favor tile
		// Chaos Magicians get 2 tiles instead of 1 (special passive ability)
		count := 1
		if chaosMagicians, ok := player.Faction.(*factions.ChaosMagicians); ok {
			if a.NewBuildingType == models.BuildingTemple {
				count = chaosMagicians.GetFavorTilesForTemple()
			} else {
				count = chaosMagicians.GetFavorTilesForSanctuary()
			}
		}
		
		// Create pending favor tile selection
		gs.PendingFavorTileSelection = &PendingFavorTileSelection{
			PlayerID:      a.PlayerID,
			Count:         count,
			SelectedTiles: []FavorTileType{},
		}
		
		// Award VP from scoring tile for Sanctuary
		if a.NewBuildingType == models.BuildingSanctuary {
			gs.AwardActionVP(a.PlayerID, ScoringActionStronghold)
		}
		break
	case models.BuildingStronghold:
		// Grant stronghold special ability
		player.HasStrongholdAbility = true

		// Call faction-specific BuildStronghold() methods to grant immediate bonuses
		switch player.Faction.GetType() {
		case models.FactionAlchemists:
			// Alchemists get +12 power immediately when building stronghold
			if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
				powerBonus := alchemists.BuildStronghold()
				if powerBonus > 0 {
					player.Resources.Power.Bowl1 += powerBonus
				}
			}
		case models.FactionCultists:
			// Cultists get +7 VP immediately when building stronghold
			if cultists, ok := player.Faction.(*factions.Cultists); ok {
				vpBonus := cultists.BuildStronghold()
				if vpBonus > 0 {
					player.VictoryPoints += vpBonus
				}
			}
		case models.FactionAuren:
			// Auren gets an immediate favor tile when building stronghold
			if auren, ok := player.Faction.(*factions.Auren); ok {
				shouldGrantFavorTile := auren.BuildStronghold()
				if shouldGrantFavorTile {
					// Create pending favor tile selection (1 tile for Auren)
					gs.PendingFavorTileSelection = &PendingFavorTileSelection{
						PlayerID:      a.PlayerID,
						Count:         1,
						SelectedTiles: []FavorTileType{},
					}
				}
			}
		case models.FactionMermaids:
			// Mermaids get +1 shipping level immediately when building stronghold
			if mermaids, ok := player.Faction.(*factions.Mermaids); ok {
				shouldGrantShipping := mermaids.BuildStronghold()
				if shouldGrantShipping {
					// Increase shipping level by 1
					currentLevel := mermaids.GetShippingLevel()
					newLevel := currentLevel + 1
					if newLevel <= mermaids.GetMaxShippingLevel() {
						mermaids.SetShippingLevel(newLevel)
						player.ShippingLevel = newLevel
						
						// Award VP based on new shipping level
						// Shipping Level 1: 2 VP, Level 2: 3 VP, Level 3: 4 VP, Level 4: 5 VP, Level 5: 6 VP
						vpBonus := newLevel + 1
						player.VictoryPoints += vpBonus
					}
				}
			}
		case models.FactionHalflings:
			// Halflings: Immediately get 3 spades to apply on terrain spaces
			// May build a dwelling on exactly one of these spaces by paying its costs
			if halflings, ok := player.Faction.(*factions.Halflings); ok {
				halflings.BuildStronghold()
				
				// Create pending spades application
				// Player must apply these 3 spades before continuing
				gs.PendingHalflingsSpades = &PendingHalflingsSpades{
					PlayerID:       a.PlayerID,
					SpadesRemaining: 3,
					TransformedHexes: []Hex{},
				}
			}
		case models.FactionDarklings:
			// Darklings: Priest ordination happens IMMEDIATELY after building stronghold
			// Player must choose how many workers (0-3) to convert to priests
			if darklings, ok := player.Faction.(*factions.Darklings); ok {
				darklings.BuildStronghold()
				
				// Create pending priest ordination
				// Player must complete this immediately before continuing
				gs.PendingDarklingsPriestOrdination = &PendingDarklingsPriestOrdination{
					PlayerID: a.PlayerID,
				}
			}
		default:
			// All other factions just mark stronghold as built (no immediate bonus)
			// This includes: Witches, Swarmlings, Chaos Magicians, Giants, Nomads, Engineers, Dwarves, Fakirs
		}

		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionStronghold)
		break
	}

	// Trigger power leech when upgrading (adjacent players leech based by their adjacent buildings)
	gs.TriggerPowerLeech(a.TargetHex, a.PlayerID)
	
	// Check for town formation after upgrading
	// CheckForTownFormation handles creating PendingTownFormation if needed
	gs.CheckForTownFormation(a.PlayerID, a.TargetHex)

	return nil
}

// isValidUpgrade checks if an upgrade path is valid
func isValidUpgrade(from, to models.BuildingType) bool {
	validUpgrades := map[models.BuildingType][]models.BuildingType{
		models.BuildingDwelling: {
			models.BuildingTradingHouse,
		},
		models.BuildingTradingHouse: {
			models.BuildingTemple,
			models.BuildingStronghold,
		},
		models.BuildingTemple: {
			models.BuildingSanctuary,
		},
	}

	allowed, exists := validUpgrades[from]
	if !exists {
		return false
	}

	for _, validTo := range allowed {
		if validTo == to {
			return true
		}
	}
	return false
}

// checkBuildingLimit checks if player has reached the building limit for a type
// Limits: 8 dwellings, 4 trading houses, 3 temples, 1 sanctuary, 1 stronghold
func checkBuildingLimit(gs *GameState, playerID string, buildingType models.BuildingType) error {
	// Count existing buildings of this type
	count := 0
	for _, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID && mapHex.Building.Type == buildingType {
			count++
		}
	}

	// Check limits
	var limit int
	switch buildingType {
	case models.BuildingDwelling:
		limit = 8
	case models.BuildingTradingHouse:
		limit = 4
	case models.BuildingTemple:
		limit = 3
	case models.BuildingSanctuary, models.BuildingStronghold:
		limit = 1
	default:
		return nil
	}

	if count >= limit {
		return fmt.Errorf("building limit reached: cannot have more than %d %v", limit, buildingType)
	}

	return nil
}

// getUpgradeCost calculates the upgrade cost, applying discount if adjacent to opponent
func getUpgradeCost(gs *GameState, player *Player, mapHex *MapHex, newBuildingType models.BuildingType) factions.Cost {
	var baseCost factions.Cost

	switch newBuildingType {
	case models.BuildingTradingHouse:
		baseCost = player.Faction.GetTradingHouseCost()
	case models.BuildingTemple:
		baseCost = player.Faction.GetTempleCost()
	case models.BuildingSanctuary:
		baseCost = player.Faction.GetSanctuaryCost()
	case models.BuildingStronghold:
		baseCost = player.Faction.GetStrongholdCost()
	default:
		return baseCost
	}

	// Apply discount for Trading House if adjacent to opponent
	if newBuildingType == models.BuildingTradingHouse {
		if hasAdjacentOpponent(gs, mapHex.Coord, player.ID) {
			// Reduce coin cost by half (6 -> 3 for most factions)
			baseCost.Coins = baseCost.Coins / 2
		}
	}

	return baseCost
}

// hasAdjacentOpponent checks if there's an opponent building adjacent to the hex
func hasAdjacentOpponent(gs *GameState, hex Hex, playerID string) bool {
	neighbors := hex.Neighbors()
	for _, neighbor := range neighbors {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID != playerID {
			return true
		}
	}
	return false
}

// AdvanceShippingAction represents advancing shipping level
type AdvanceShippingAction struct {
	BaseAction
}

func NewAdvanceShippingAction(playerID string) *AdvanceShippingAction {
	return &AdvanceShippingAction{
		BaseAction: BaseAction{
			Type:     ActionAdvanceShipping,
			PlayerID: playerID,
		},
	}
}

func (a *AdvanceShippingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player can advance shipping (some factions like Dwarves/Fakirs cannot)
	// Faction.CanUpgradeShipping() is checked by the faction implementations

	// Check if already at max level
	if player.ShippingLevel >= 5 {
		return fmt.Errorf("shipping already at max level")
	}

	// Check if player can afford shipping upgrade
	cost := player.Faction.GetShippingCost(player.ShippingLevel)
	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford shipping upgrade")
	}

	return nil
}

func (a *AdvanceShippingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	cost := player.Faction.GetShippingCost(player.ShippingLevel)

	// Pay for upgrade
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for shipping: %w", err)
	}

	// Advance shipping
	player.ShippingLevel++
	
	// Award VP based on new shipping level
	// Shipping Level 1: 2 VP, Level 2: 3 VP, Level 3: 4 VP, Level 4: 5 VP, Level 5: 6 VP
	vpBonus := player.ShippingLevel + 1
	player.VictoryPoints += vpBonus

	return nil
}

// AdvanceDiggingAction represents advancing digging level
type AdvanceDiggingAction struct {
	BaseAction
}

func NewAdvanceDiggingAction(playerID string) *AdvanceDiggingAction {
	return &AdvanceDiggingAction{
		BaseAction: BaseAction{
			Type:     ActionAdvanceDigging,
			PlayerID: playerID,
		},
	}
}

func (a *AdvanceDiggingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if already at max level
	if player.DiggingLevel >= 2 {
		return fmt.Errorf("digging already at max level")
	}

	// Check if player can afford digging upgrade
	cost := player.Faction.GetDiggingCost(player.DiggingLevel)
	if !player.Resources.CanAfford(cost) {
		return fmt.Errorf("cannot afford digging upgrade")
	}

	return nil
}

func (a *AdvanceDiggingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	cost := player.Faction.GetDiggingCost(player.DiggingLevel)

	// Pay for upgrade
	if err := player.Resources.Spend(cost); err != nil {
		return fmt.Errorf("failed to pay for digging: %w", err)
	}

	// Advance digging
	player.DiggingLevel++
	
	// Award VP based on new digging level
	// Digging Level 1: 2 VP, Level 2: 3 VP, Level 3: 4 VP, Level 4: 5 VP, Level 5: 6 VP
	vpBonus := player.DiggingLevel + 1
	player.VictoryPoints += vpBonus

	return nil
}

// PassAction represents passing for the round
type PassAction struct {
	BaseAction
	BonusCard *BonusCardType // Bonus card selection (required)
}

func NewPassAction(playerID string, bonusCard *BonusCardType) *PassAction {
	return &PassAction{
		BaseAction: BaseAction{
			Type:     ActionPass,
			PlayerID: playerID,
		},
		BonusCard: bonusCard,
	}
}

func (a *PassAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Validate bonus card selection
	if a.BonusCard == nil {
		return fmt.Errorf("bonus card selection is required when passing")
	}

	if !gs.BonusCards.IsAvailable(*a.BonusCard) {
		// Debug: show which cards ARE available
		availableCards := []BonusCardType{}
		for cardType := range gs.BonusCards.Available {
			availableCards = append(availableCards, cardType)
		}
		fmt.Printf("DEBUG: Player %s trying to take card %v, but available cards are: %v\n",
			a.PlayerID, *a.BonusCard, availableCards)
		return fmt.Errorf("bonus card %v is not available", *a.BonusCard)
	}

	return nil
}

func (a *PassAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	player.HasPassed = true
	
	// Record pass order (determines turn order for next round)
	gs.PassOrder = append(gs.PassOrder, a.PlayerID)

	// Take bonus card and get coins from it
	coins, err := gs.BonusCards.TakeBonusCard(a.PlayerID, *a.BonusCard)
	if err != nil {
		return fmt.Errorf("failed to take bonus card: %w", err)
	}
	player.Resources.Coins += coins

	// Award VP from Air+1 favor tile (VP based on Trading House count)
	playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
	if HasFavorTile(playerTiles, FavorAir1) {
		// Count trading houses on the map
		tradingHouseCount := 0
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil && 
			   mapHex.Building.PlayerID == a.PlayerID && 
			   mapHex.Building.Type == models.BuildingTradingHouse {
				tradingHouseCount++
			}
		}
		
		vp := GetAir1PassVP(playerTiles, tradingHouseCount)
		player.VictoryPoints += vp
	}

	// Award VP from bonus card (based on buildings/shipping)
	bonusVP := GetBonusCardPassVP(*a.BonusCard, gs, a.PlayerID)
	player.VictoryPoints += bonusVP

	// Award VP for Engineers stronghold ability (3 VP per bridge when passing)
	if player.Faction.GetType() == models.FactionEngineers && player.HasStrongholdAbility {
		engineersFaction, ok := player.Faction.(*factions.Engineers)
		if ok {
			vpPerBridge := engineersFaction.GetVPPerBridgeOnPass()
			bridgeVP := player.BridgesBuilt * vpPerBridge
			player.VictoryPoints += bridgeVP
		}
	}

	return nil
}

// SendPriestToCultAction represents sending a priest to a cult track
type SendPriestToCultAction struct {
	BaseAction
	Track         CultTrack
	SpacesToClimb int // Number of spaces to advance (1-3), always costs 1 priest
}

func (a *SendPriestToCultAction) GetType() ActionType {
	return ActionSendPriestToCult
}

func (a *SendPriestToCultAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has passed
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Check if player has a priest
	if player.Resources.Priests < 1 {
		return fmt.Errorf("not enough priests: need 1, have %d", player.Resources.Priests)
	}

	// Validate spaces to climb
	if a.SpacesToClimb < 1 || a.SpacesToClimb > 3 {
		return fmt.Errorf("invalid spaces to climb: %d (must be 1-3)", a.SpacesToClimb)
	}

	return nil
}

func (a *SendPriestToCultAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)

	// Remove 1 priest from player's supply (cost is always 1 priest, regardless of spaces)
	player.Resources.Priests -= 1

	// Advance on cult track (with bonus power at milestones)
	// Note: It's valid to sacrifice a priest even if you can't advance (no refund)
	// Position 10 requires a key (checked in gs.AdvanceCultTrack)
	gs.AdvanceCultTrack(a.PlayerID, a.Track, a.SpacesToClimb)

	// Record priest sent for scoring tile #5 (Trading House + Priest)
	if gs.ScoringTiles != nil {
		gs.ScoringTiles.RecordPriestSent(a.PlayerID)
	}

	// Note: If UsePriestSlot is true, the priest is permanently placed on the board
	// If false, the priest is returned to supply (already handled by not placing it)

	return nil
}
