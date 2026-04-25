package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// ActionType represents the type of action a player can take
type ActionType int

const (
	// ActionTransformAndBuild represents transforming terrain and optionally building a dwelling
	ActionTransformAndBuild ActionType = iota
	ActionUpgradeBuilding
	ActionAdvanceShipping
	ActionAdvanceDigging
	ActionAdvanceChashTrack
	ActionSendPriestToCult
	ActionPowerAction
	ActionSpecialAction
	ActionPass
	ActionSetupDwelling                 // Place initial dwelling during setup (no cost, no adjacency)
	ActionUseCultSpade                  // Use a spade from cult track reward (cleanup phase)
	ActionAcceptPowerLeech              // Accept a power leech offer
	ActionDeclinePowerLeech             // Decline a power leech offer
	ActionSelectFavorTile               // Select a favor tile after Temple/Sanctuary/Auren Stronghold
	ActionApplyHalflingsSpade           // Apply one of 3 stronghold spades (Halflings only)
	ActionBuildHalflingsDwelling        // Build dwelling on transformed hex (Halflings optional)
	ActionSkipHalflingsDwelling         // Skip optional dwelling (Halflings)
	ActionBuildWispsStrongholdDwelling  // Build the free Wisps stronghold lake dwelling
	ActionUseGoblinsTreasure            // Spend one Goblins treasure token for a reward
	ActionSelectGoblinsCultTrack        // Resolve Goblins treasure cult-step reward
	ActionUseDarklingsPriestOrdination  // Convert 0-3 workers to priests (Darklings stronghold, one-time)
	ActionSelectCultistsCultTrack       // Select cult track for power leech bonus (Cultists only)
	ActionSelectDjinniStartingCultTrack // Select Djinni starting cult track during setup
	ActionSelectTreasurersDeposit       // Select how many newly gained resources Treasurers bank
	ActionSelectArchivistsBonusCard     // Select Archivists' second bonus card after passing with stronghold
	ActionSelectFaction                 // Select faction at start of game
	ActionAuctionNominateFaction        // Nominate faction in regular/fast auction setup modes
	ActionAuctionPlaceBid               // Place bid in regular auction mode
	ActionFastAuctionSubmitBids         // Submit sealed bid vector in fast auction mode
	ActionSetupBonusCard                // Select bonus card during setup
	ActionSelectTownTile                // Select town tile from pending town formation
	ActionSelectTownCultTop             // Resolve key-limited town cult top choice
	ActionDiscardPendingSpade           // Discard one pending free spade from a follow-up chain
	ActionConversion                    // Free conversion action (does not consume main action)
	ActionBurnPower                     // Free burn action (does not consume main action)
	ActionEngineersBridge               // Engineers SH special action: build bridge for workers
	ActionSetPlayerOptions              // Update player UX/automation options
	ActionConfirmTurn                   // Confirm the current turn before the next player may act
	ActionUndoTurn                      // Undo the current turn back to the last snapshot
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

// GetType returns the action type
func (a *BaseAction) GetType() ActionType {
	return a.Type
}

// GetPlayerID returns the player ID
func (a *BaseAction) GetPlayerID() string {
	return a.PlayerID
}

// TransformAndBuildAction represents terraforming a hex and optionally building a dwelling
// Per rulebook: "First, you may change the type of one Terrain space. Then, if you have
// changed its type to your Home terrain, you may immediately build a Dwelling on that space."
type TransformAndBuildAction struct {
	BaseAction
	TargetHex     board.Hex
	TargetTerrain models.TerrainType // Optional: target terrain type (if not home terrain)
	BuildDwelling bool               // Whether to build a dwelling after transforming
	UseSkip       bool               // Fakirs carpet flight / Dwarves tunneling - skip adjacency for one space
}

// NewTransformAndBuildAction creates a new transform and build action
func NewTransformAndBuildAction(playerID string, targetHex board.Hex, buildDwelling bool, targetTerrain models.TerrainType) *TransformAndBuildAction {
	return &TransformAndBuildAction{
		BaseAction: BaseAction{
			Type:     ActionTransformAndBuild,
			PlayerID: playerID,
		},
		TargetHex:     targetHex,
		TargetTerrain: targetTerrain,
		BuildDwelling: buildDwelling,
		UseSkip:       false,
	}
}

// NewTransformAndBuildActionWithSkip creates a transform action with carpet flight/tunneling
func NewTransformAndBuildActionWithSkip(playerID string, targetHex board.Hex, buildDwelling bool, targetTerrain models.TerrainType) *TransformAndBuildAction {
	return &TransformAndBuildAction{
		BaseAction: BaseAction{
			Type:     ActionTransformAndBuild,
			PlayerID: playerID,
		},
		TargetHex:     targetHex,
		TargetTerrain: targetTerrain,
		BuildDwelling: buildDwelling,
		UseSkip:       true,
	}
}

func (a *TransformAndBuildAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
	}

	// Check if hex exists and is empty (no building)
	mapHex, err := gs.ValidateHex(a.TargetHex)
	if err != nil {
		return err
	}
	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building: %v", a.TargetHex)
	}

	if gs.PendingSpades != nil && gs.PendingSpades[a.PlayerID] > 0 {
		targetTerrain := resolveActionTargetTerrain(player, mapHex.Terrain, a.TargetTerrain)
		requiredSpades, err := fireIceTerraformDistance(player, mapHex.Terrain, targetTerrain)
		if err != nil {
			return err
		}
		if player.Faction.GetType() == models.FactionGiants {
			requiredSpades = 2
		}
		if requiredSpades <= 0 {
			return fmt.Errorf("pending spade follow-up must transform terrain")
		}
		if a.BuildDwelling {
			if allowed, ok := gs.PendingSpadeBuildAllowed[a.PlayerID]; ok && !allowed {
				return fmt.Errorf("cannot build dwelling on remaining pending spade follow-up")
			}
		}
		if sourceHex, ok := gs.PendingWispsTradingPostSpade[a.PlayerID]; ok {
			if a.BuildDwelling {
				return fmt.Errorf("wisps trading post spade cannot build a dwelling")
			}
			if requiredSpades != 1 {
				return fmt.Errorf("wisps trading post spade must transform exactly 1 directly adjacent terrain")
			}
			isDirectNeighbor := false
			for _, neighbor := range gs.Map.GetDirectNeighbors(sourceHex) {
				if neighbor == a.TargetHex {
					isDirectNeighbor = true
					break
				}
			}
			if !isDirectNeighbor {
				return fmt.Errorf("wisps trading post spade must target terrain directly adjacent to the new trading post")
			}
		}
	}

	if err := a.validateAdjacency(gs, player); err != nil {
		return err
	}

	totalWorkersNeeded, totalPriestsNeeded, totalPowerNeeded, totalCoinsNeeded, err := a.calculateCosts(gs, player, mapHex)
	if err != nil {
		return err
	}

	combinedCost := factions.Cost{
		Workers: totalWorkersNeeded,
		Priests: totalPriestsNeeded,
		Coins:   totalCoinsNeeded,
		Power:   totalPowerNeeded,
	}
	if totalPowerNeeded == 0 && gs.canAffordWithReplayAutoConversions(player, combinedCost) {
		return nil
	}

	// Check total workers needed (terraform + dwelling)
	if player.Resources.Workers < totalWorkersNeeded {
		return fmt.Errorf("not enough workers: need %d, have %d", totalWorkersNeeded, player.Resources.Workers)
	}

	// Check total priests needed (Darklings terraform cost)
	if player.Resources.Priests < totalPriestsNeeded {
		return fmt.Errorf("not enough priests for terraform: need %d, have %d", totalPriestsNeeded, player.Resources.Priests)
	}
	if totalPowerNeeded > 0 && !player.Resources.Power.CanSpend(totalPowerNeeded) {
		if player.Faction.GetType() != models.FactionTheEnlightened {
			return fmt.Errorf("not enough power for terraform: need %d, have %d", totalPowerNeeded, player.Resources.Power.Bowl3)
		}
		requiredBurn := totalPowerNeeded - player.Resources.Power.Bowl3
		if requiredBurn <= 0 || !player.Resources.Power.CanBurn(requiredBurn) {
			return fmt.Errorf("not enough power for terraform: need %d, have %d", totalPowerNeeded, player.Resources.Power.Bowl3)
		}
	}
	if player.Resources.Coins < totalCoinsNeeded {
		return fmt.Errorf("not enough coins: need %d, have %d", totalCoinsNeeded, player.Resources.Coins)
	}

	return nil
}

func (a *TransformAndBuildAction) validateAdjacency(gs *GameState, player *Player) error {
	// Check adjacency - required for both transforming and building
	isAdjacent := gs.IsAdjacentToPlayerBuilding(a.TargetHex, a.PlayerID)

	// Auto-detect UseSkip for Dwarves and Fakirs if hex is not adjacent
	if !isAdjacent && !a.UseSkip {
		factionType := player.Faction.GetType()
		if factionType == models.FactionDwarves || factionType == models.FactionFakirs {
			// Automatically enable skip ability for these factions when hex is not adjacent
			a.UseSkip = true
		}
	}

	// If using skip (Fakirs/Dwarves), check if player can skip and if range is valid
	if a.UseSkip {
		if err := ValidateSkipAbility(gs, player, a.TargetHex); err != nil {
			return err
		}
	} else {
		// Normal adjacency required if not using skip
		if !isAdjacent {
			return fmt.Errorf("hex is not adjacent to player's buildings")
		}
	}
	return nil
}

func (a *TransformAndBuildAction) calculateCosts(gs *GameState, player *Player, mapHex *board.MapHex) (int, int, int, int, error) {
	// Check if terrain needs transformation to target terrain (default: home terrain)
	targetTerrain := resolveActionTargetTerrain(player, mapHex.Terrain, a.TargetTerrain)
	if isSelkies(player) && mapHex.Terrain == models.TerrainRiver && a.BuildDwelling {
		targetTerrain = models.TerrainRiver
	}
	needsTransform := mapHex.Terrain != targetTerrain

	totalWorkersNeeded := 0
	totalPriestsNeeded := 0
	totalPowerNeeded := 0
	totalCoinsNeeded := 0

	if needsTransform {
		// Calculate terraform cost
		if isRiverwalkers(player) {
			return 0, 0, 0, 0, fmt.Errorf("riverwalkers cannot transform terrain")
		}
		if targetTerrain == models.TerrainVolcano {
			switch {
			case isDragonlords(player):
				if player.Resources.Power.TotalPower() < volcanoTransformCost(gs, player, mapHex.Terrain) {
					return 0, 0, 0, 0, fmt.Errorf("not enough power tokens for volcano transform")
				}
			case isAcolytes(player):
				if _, ok := gs.acolytesCultPaymentTrack(player, acolytesCultTransformCost(gs, player, mapHex.Terrain)); !ok {
					return 0, 0, 0, 0, fmt.Errorf("not enough cult steps for volcano transform")
				}
			case isFirewalkers(player):
				if firewalkersAvailableVP(player) < firewalkersVPTransformCost(gs, player, mapHex.Terrain) {
					return 0, 0, 0, 0, fmt.Errorf("not enough available victory points for lava transform")
				}
			default:
				return 0, 0, 0, 0, fmt.Errorf("only volcano factions may transform to volcano")
			}
		} else {
			distance, err := fireIceTerraformDistance(player, mapHex.Terrain, targetTerrain)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			requiredSpades := distance
			if player.Faction.GetType() == models.FactionGiants {
				// Giants always require exactly 2 spades for a terrain transform.
				requiredSpades = 2
			}
			requiredSpades = adjustRequiredSpadesForArchitects(gs, player, a.TargetHex, requiredSpades, a.BuildDwelling)

			// Check for free spades from power actions (ACT5/ACT6) or cult rewards
			freeSpades := 0
			if !isProspectors(player) {
				if gs.PendingSpades != nil && gs.PendingSpades[a.PlayerID] > 0 {
					freeSpades += gs.PendingSpades[a.PlayerID]
				}
				if gs.PendingCultRewardSpades != nil && gs.PendingCultRewardSpades[a.PlayerID] > 0 {
					freeSpades += gs.PendingCultRewardSpades[a.PlayerID]
				}
			}
			if player.Faction.GetType() == models.FactionGiants {
				// Giants cannot use a single free spade; only a full pair is usable.
				if freeSpades >= 2 {
					freeSpades = 2
				} else {
					freeSpades = 0
				}
			} else if freeSpades > requiredSpades {
				freeSpades = requiredSpades // Only use what we need
			}

			remainingSpades := requiredSpades - freeSpades

			// Darklings pay priests for terraform (1 priest per spade)
			if remainingSpades > 0 {
				if player.Faction.GetType() == models.FactionDarklings {
					totalPriestsNeeded = remainingSpades
				} else if isProspectors(player) {
					totalCoinsNeeded += remainingSpades * getProspectorsGoldenSpadeCost(player)
				} else if player.Faction.GetType() == models.FactionTheEnlightened {
					totalPowerNeeded = player.Faction.GetTerraformCost(remainingSpades)
				} else if isIceFactionType(player.Faction.GetType()) {
					cost := iceTerraformCost(player, remainingSpades)
					totalWorkersNeeded += cost.Workers
					totalPriestsNeeded += cost.Priests
					totalPowerNeeded += cost.Power
					totalCoinsNeeded += cost.Coins
				} else {
					// Other factions pay workers
					totalWorkersNeeded = player.Faction.GetTerraformCost(remainingSpades)
				}
			}
		}
	}

	// Add tunneling cost to total if using skip (Dwarves)
	if a.UseSkip {
		if player.Faction.GetType() == models.FactionDwarves {
			workerCost := 2
			if player.HasStrongholdAbility {
				workerCost = 1
			}
			totalWorkersNeeded += workerCost
		}
	}

	// If building a dwelling, check requirements
	if a.BuildDwelling {
		if err := a.validateDwelling(gs, player, mapHex, needsTransform, targetTerrain); err != nil {
			return 0, 0, 0, 0, err
		}
		dwellingCost := getDwellingBuildCost(gs, player, a.TargetHex)
		totalWorkersNeeded += dwellingCost.Workers
		totalCoinsNeeded += dwellingCost.Coins
	}

	return totalWorkersNeeded, totalPriestsNeeded, totalPowerNeeded, totalCoinsNeeded, nil
}

func (a *TransformAndBuildAction) validateDwelling(gs *GameState, player *Player, mapHex *board.MapHex, needsTransform bool, targetTerrain models.TerrainType) error {
	// Check building limit (max 8 dwellings)
	if err := gs.CheckBuildingLimit(a.PlayerID, models.BuildingDwelling); err != nil {
		return err
	}

	if isSelkies(player) && mapHex.Terrain == models.TerrainRiver && !needsTransform {
		if !canSelkiesBuildRiverDwelling(gs, player, a.TargetHex) {
			return fmt.Errorf("selkies need two non-adjacent ice buildings adjacent to build on river")
		}
		return nil
	}

	if isRiverwalkers(player) && isStandardLandTerrain(mapHex.Terrain) && !needsTransform {
		if !riverwalkersCanSettleTerrain(player, mapHex.Terrain) {
			return fmt.Errorf("riverwalkers have not unlocked %s terrain", mapHex.Terrain)
		}
		if !isAdjacentToRiver(gs, a.TargetHex) {
			return fmt.Errorf("riverwalkers dwellings must be adjacent to river terrain")
		}
		return nil
	}

	// After transformation (if any), hex must be player's home terrain
	if needsTransform {
		// Will be target terrain after transform
		if targetTerrain != effectiveHomeTerrain(player) {
			return fmt.Errorf("cannot build dwelling: target terrain %v is not home terrain", targetTerrain)
		}
	} else if mapHex.Terrain != effectiveHomeTerrain(player) {
		return fmt.Errorf("cannot build dwelling: hex is not home terrain")
	}

	// Check if player can afford dwelling (coins and priests)
	dwellingCost := getDwellingBuildCost(gs, player, a.TargetHex)
	if !gs.canAffordWithReplayAutoConversions(player, dwellingCost) {
		return fmt.Errorf("not enough resources for dwelling: need %v, have %v", dwellingCost, player.Resources)
	}
	return nil
}

func (a *TransformAndBuildAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	mapHex := gs.Map.GetHex(a.TargetHex)

	totalWorkersNeeded, totalPriestsNeeded, totalPowerNeeded, totalCoinsNeeded, err := a.calculateCosts(gs, player, mapHex)
	if err != nil {
		return err
	}
	if totalPowerNeeded == 0 {
		combinedCost := factions.Cost{
			Workers: totalWorkersNeeded,
			Priests: totalPriestsNeeded,
			Coins:   totalCoinsNeeded,
			Power:   totalPowerNeeded,
		}
		if err := gs.prepareReplayAutoConversions(player, combinedCost); err != nil {
			return err
		}
	}

	// Step 0: Handle skip costs (Fakirs carpet flight / Dwarves tunneling)
	if a.UseSkip {
		if gs.SkipAbilityUsedThisAction == nil {
			gs.SkipAbilityUsedThisAction = make(map[string][]board.Hex)
		}

		alreadyPaid := false
		usedHexes := gs.SkipAbilityUsedThisAction[player.ID]
		// De-duplicate skip payment only while turn advancement is suppressed
		// (i.e. within a single compound/synthetic action execution).
		if gs.SuppressTurnAdvance {
			for _, h := range usedHexes {
				if h == a.TargetHex {
					alreadyPaid = true
					break
				}
			}
		}

		if !alreadyPaid {
			// Not used yet for this hex, pay cost and mark as used
			PaySkipCost(player)
			gs.SkipAbilityUsedThisAction[player.ID] = append(usedHexes, a.TargetHex)
		}
	}

	// Step 1: Transform terrain to target terrain if needed
	if err := a.handleTransform(gs, player, mapHex); err != nil {
		return err
	}

	// Step 2: Build dwelling if requested
	if a.BuildDwelling {
		if err := a.handleBuildDwelling(gs, player); err != nil {
			return err
		}
	}

	// Advance turn
	gs.NextTurn()

	return nil
}

func (a *TransformAndBuildAction) handleTransform(gs *GameState, player *Player, mapHex *board.MapHex) error {
	targetTerrain := resolveActionTargetTerrain(player, mapHex.Terrain, a.TargetTerrain)
	if isSelkies(player) && mapHex.Terrain == models.TerrainRiver && a.BuildDwelling {
		targetTerrain = models.TerrainRiver
	}
	needsTransform := mapHex.Terrain != targetTerrain
	if !needsTransform {
		return nil
	}

	if targetTerrain == models.TerrainVolcano {
		switch {
		case isDragonlords(player):
			if err := gs.removePowerTokens(a.PlayerID, volcanoTransformCost(gs, player, mapHex.Terrain)); err != nil {
				return err
			}
		case isAcolytes(player):
			if err := gs.spendAcolytesCultSteps(a.PlayerID, acolytesCultTransformCost(gs, player, mapHex.Terrain)); err != nil {
				return err
			}
		case isFirewalkers(player):
			cost := firewalkersVPTransformCost(gs, player, mapHex.Terrain)
			player.VictoryPoints -= cost
		default:
			return fmt.Errorf("only volcano factions may transform to volcano")
		}
		return gs.Map.TransformTerrain(a.TargetHex, targetTerrain)
	}

	distance, err := fireIceTerraformDistance(player, mapHex.Terrain, targetTerrain)
	if err != nil {
		return err
	}
	requiredSpades := distance
	if player.Faction.GetType() == models.FactionGiants {
		// Giants always require exactly 2 spades for a terrain transform.
		requiredSpades = 2
	}
	requiredSpades = adjustRequiredSpadesForArchitects(gs, player, a.TargetHex, requiredSpades, a.BuildDwelling)

	// Check for free spades from BON1 (count for VP when used)
	vpEligibleFreeSpades := 0
	// Check for cult reward spades (don't count for VP)
	cultRewardSpades := 0
	if player.Faction.GetType() == models.FactionGiants {
		pending := 0
		if !isProspectors(player) && gs.PendingSpades != nil {
			pending = gs.PendingSpades[a.PlayerID]
		}
		cultPending := 0
		if !isProspectors(player) && gs.PendingCultRewardSpades != nil {
			cultPending = gs.PendingCultRewardSpades[a.PlayerID]
		}
		if pending+cultPending >= 2 {
			remainingToConsume := 2
			if pending > 0 {
				vpEligibleFreeSpades = pending
				if vpEligibleFreeSpades > remainingToConsume {
					vpEligibleFreeSpades = remainingToConsume
				}
				gs.PendingSpades[a.PlayerID] -= vpEligibleFreeSpades
				if gs.PendingSpades[a.PlayerID] == 0 {
					delete(gs.PendingSpades, a.PlayerID)
					delete(gs.PendingSpadeBuildAllowed, a.PlayerID)
					gs.clearPendingWispsTradingPostSpade(a.PlayerID)
				}
				remainingToConsume -= vpEligibleFreeSpades
			}
			if remainingToConsume > 0 && cultPending > 0 {
				cultRewardSpades = cultPending
				if cultRewardSpades > remainingToConsume {
					cultRewardSpades = remainingToConsume
				}
				gs.PendingCultRewardSpades[a.PlayerID] -= cultRewardSpades
				if gs.PendingCultRewardSpades[a.PlayerID] == 0 {
					delete(gs.PendingCultRewardSpades, a.PlayerID)
				}
			}
		}
	} else {
		if !isProspectors(player) && gs.PendingSpades != nil && gs.PendingSpades[a.PlayerID] > 0 {
			vpEligibleFreeSpades = gs.PendingSpades[a.PlayerID]
			if vpEligibleFreeSpades > requiredSpades {
				vpEligibleFreeSpades = requiredSpades // Only use what we need
			}
			// Consume VP-eligible free spades
			gs.PendingSpades[a.PlayerID] -= vpEligibleFreeSpades
			if gs.PendingSpades[a.PlayerID] == 0 {
				delete(gs.PendingSpades, a.PlayerID)
				delete(gs.PendingSpadeBuildAllowed, a.PlayerID)
				gs.clearPendingWispsTradingPostSpade(a.PlayerID)
			}
		}

		remainingRequired := requiredSpades - vpEligibleFreeSpades
		if !isProspectors(player) && remainingRequired > 0 && gs.PendingCultRewardSpades != nil && gs.PendingCultRewardSpades[a.PlayerID] > 0 {
			cultRewardSpades = gs.PendingCultRewardSpades[a.PlayerID]
			if cultRewardSpades > remainingRequired {
				cultRewardSpades = remainingRequired // Only use what we need
			}
			// Consume cult reward spades
			gs.PendingCultRewardSpades[a.PlayerID] -= cultRewardSpades
			if gs.PendingCultRewardSpades[a.PlayerID] == 0 {
				delete(gs.PendingCultRewardSpades, a.PlayerID)
			}
		}
	}

	totalFreeSpades := vpEligibleFreeSpades + cultRewardSpades
	remainingSpades := requiredSpades - totalFreeSpades

	// Pay for remaining spades only
	if remainingSpades > 0 {
		// Darklings pay priests for terraform (instead of workers)
		if player.Faction.GetType() == models.FactionDarklings {
			priestCost := remainingSpades
			player.Resources.Priests -= priestCost

			// Award Darklings VP bonus (+2 VP per remaining spade, not free spades)
			vpBonus := remainingSpades * 2
			player.VictoryPoints += vpBonus
		} else if isProspectors(player) {
			player.Resources.Coins -= remainingSpades * getProspectorsGoldenSpadeCost(player)
			player.Resources.GainPower(remainingSpades)
			player.VictoryPoints += remainingSpades
		} else if player.Faction.GetType() == models.FactionTheEnlightened {
			powerCost := player.Faction.GetTerraformCost(remainingSpades)
			requiredBurn := powerCost - player.Resources.Power.Bowl3
			if requiredBurn > 0 {
				if err := player.Resources.Power.BurnPower(requiredBurn); err != nil {
					return fmt.Errorf("failed to auto-burn power for terraform: %w", err)
				}
			}
			if err := player.Resources.Power.SpendPower(powerCost); err != nil {
				return fmt.Errorf("failed to spend power for terraform: %w", err)
			}
		} else if isIceFactionType(player.Faction.GetType()) {
			if err := gs.spendWithReplayAutoConversions(player, iceTerraformCost(player, remainingSpades)); err != nil {
				return fmt.Errorf("failed to pay for ice terraform: %w", err)
			}
		} else {
			// Other factions pay workers
			totalWorkers := player.Faction.GetTerraformCost(remainingSpades)
			player.Resources.Workers -= totalWorkers
		}
	}

	// Transform terrain to target terrain
	if err := gs.Map.TransformTerrain(a.TargetHex, targetTerrain); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Award VP for paid spades + VP-eligible free spades (BON1)
	// Cult reward spades don't award VP
	vpEligibleDistance := remainingSpades + vpEligibleFreeSpades
	if vpEligibleDistance > 0 {
		spadesForVP := vpEligibleDistance
		if player.Faction.GetType() == models.FactionGiants {
			spadesForVP = 2
		}

		// Prospectors' Golden Spades do not count for round spade scoring.
		if !isProspectors(player) {
			for i := 0; i < spadesForVP; i++ {
				gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
			}
		}

		// Award faction-specific spade bonuses (Halflings VP, Alchemists power)
		AwardFactionSpadeBonuses(player, spadesForVP)
	}

	// Award faction-specific spade bonuses for cult reward spades too
	// Cult reward spades don't count for VP, but Alchemists still get power for them
	if cultRewardSpades > 0 {
		spadesUsed := cultRewardSpades
		if player.Faction.GetType() == models.FactionGiants {
			spadesUsed = 2
		}
		AwardFactionSpadeBonuses(player, spadesUsed)
	}
	return nil
}

func adjustRequiredSpadesForArchitects(gs *GameState, player *Player, targetHex board.Hex, requiredSpades int, buildDwelling bool) int {
	if gs == nil || !buildDwelling || requiredSpades <= 0 || !isArchitects(player) {
		return requiredSpades
	}
	bridgeReduction := gs.Map.CountPlayerBridgesIncidentToHex(targetHex, player.ID)
	if bridgeReduction <= 0 {
		return requiredSpades
	}
	if bridgeReduction >= requiredSpades {
		return 0
	}
	return requiredSpades - bridgeReduction
}

func (a *TransformAndBuildAction) handleBuildDwelling(gs *GameState, player *Player) error {
	// Pay for dwelling
	dwellingCost := getDwellingBuildCost(gs, player, a.TargetHex)
	if err := gs.spendWithReplayAutoConversions(player, dwellingCost); err != nil {
		return fmt.Errorf("failed to pay for dwelling: %w", err)
	}

	// Place dwelling and handle all VP bonuses
	if err := gs.BuildDwelling(a.PlayerID, a.TargetHex); err != nil {
		return err
	}
	return nil
}

// UpgradeBuildingAction represents upgrading a building
type UpgradeBuildingAction struct {
	BaseAction
	TargetHex       board.Hex
	NewBuildingType models.BuildingType
}

// NewUpgradeBuildingAction creates a new upgrade building action
func NewUpgradeBuildingAction(playerID string, targetHex board.Hex, newType models.BuildingType) *UpgradeBuildingAction {
	return &UpgradeBuildingAction{
		BaseAction: BaseAction{
			Type:     ActionUpgradeBuilding,
			PlayerID: playerID,
		},
		TargetHex:       targetHex,
		NewBuildingType: newType,
	}
}

// Validate checks if the upgrade action is valid
func (a *UpgradeBuildingAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
	}

	// Check if hex has player's building
	mapHex, err := gs.ValidateHex(a.TargetHex)
	if err != nil {
		return err
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
	if isSelkies(player) && mapHex.Terrain == models.TerrainRiver {
		return fmt.Errorf("selkies river dwellings cannot be upgraded")
	}

	// Check building limits
	if err := gs.CheckBuildingLimit(a.PlayerID, a.NewBuildingType); err != nil {
		return err
	}

	// Get upgrade cost (may be reduced if adjacent to opponent)
	cost := getUpgradeCost(gs, player, mapHex, a.NewBuildingType)

	if !gs.canAffordWithReplayAutoConversions(player, cost) {
		return fmt.Errorf("cannot afford upgrade to %v", a.NewBuildingType)
	}

	return nil
}

// Execute performs the upgrade action
func (a *UpgradeBuildingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	mapHex := gs.Map.GetHex(a.TargetHex)

	// Get upgrade cost (may be reduced if adjacent to opponent)
	cost := getUpgradeCost(gs, player, mapHex, a.NewBuildingType)

	// Pay for upgrade
	if err := gs.spendWithReplayAutoConversions(player, cost); err != nil {
		return fmt.Errorf("failed to pay for upgrade: %w", err)
	}

	// Upgrade building
	mapHex.Building = &models.Building{
		Type:       a.NewBuildingType,
		Faction:    player.Faction.GetType(),
		PlayerID:   a.PlayerID,
		PowerValue: getStructurePowerValue(player, a.NewBuildingType),
	}

	// Handle special rewards based on upgrade type
	a.handleUpgradeRewards(gs, player)

	// Trigger power leech when upgrading (adjacent players leech based by their adjacent buildings)
	gs.TriggerPowerLeech(a.TargetHex, a.PlayerID)

	// Check for town formation after upgrading
	// For Temple/Sanctuary: defer town check until after favor tile is selected
	// (favor tiles can reduce town power requirement from 7 to 6)
	if a.NewBuildingType != models.BuildingTemple && a.NewBuildingType != models.BuildingSanctuary {
		gs.CheckForTownFormation(a.PlayerID, a.TargetHex)
	}
	gs.updateAtlanteansStrongholdTown(a.PlayerID)

	// Advance turn (unless pending actions exist, checked by NextTurn)
	gs.NextTurn()

	return nil
}

func (a *UpgradeBuildingAction) handleUpgradeRewards(gs *GameState, player *Player) {
	switch a.NewBuildingType {
	case models.BuildingTradingHouse:
		// Award VP from Water+1 favor tile (+3 VP when upgrading Dwelling→Trading House)
		playerTiles := gs.FavorTiles.GetPlayerTiles(a.PlayerID)
		if HasFavorTile(playerTiles, FavorWater1) {
			player.VictoryPoints += 3
		}

		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionTradingHouse)
		if player.Faction.GetType() == models.FactionWisps {
			if gs.PendingSpades == nil {
				gs.PendingSpades = make(map[string]int)
			}
			if gs.PendingSpadeBuildAllowed == nil {
				gs.PendingSpadeBuildAllowed = make(map[string]bool)
			}
			if gs.PendingWispsTradingPostSpade == nil {
				gs.PendingWispsTradingPostSpade = make(map[string]board.Hex)
			}
			gs.PendingSpades[a.PlayerID] = 1
			gs.PendingSpadeBuildAllowed[a.PlayerID] = false
			gs.PendingWispsTradingPostSpade[a.PlayerID] = a.TargetHex
		}
	case models.BuildingTemple, models.BuildingSanctuary:
		// Player must select a Favor tile
		// Chaos Magicians get 2 tiles instead of 1 (special passive ability)
		count := 1
		if player.Faction.GetType() == models.FactionChaosMagicians {
			count = 2
		}

		// Create pending favor tile selection
		gs.PendingFavorTileSelection = &PendingFavorTileSelection{
			PlayerID:      a.PlayerID,
			Count:         count,
			SelectedTiles: []FavorTileType{},
		}

		// Award VP from scoring tile
		if a.NewBuildingType == models.BuildingTemple {
			// SCORE5 (Temple+Priest): 4 VP per temple + 2 coins per priest at end of round
			gs.AwardActionVP(a.PlayerID, ScoringActionTemple)
			if player.Faction.GetType() == models.FactionGoblins {
				player.GoblinTreasureTokens++
			}
		} else if a.NewBuildingType == models.BuildingSanctuary {
			gs.AwardActionVP(a.PlayerID, ScoringActionStronghold)
			if player.Faction.GetType() == models.FactionGoblins {
				player.GoblinTreasureTokens++
			}
		}
	case models.BuildingStronghold:
		// Grant stronghold special ability
		player.HasStrongholdAbility = true

		// Call faction-specific BuildStronghold() methods to grant immediate bonuses
		a.handleStrongholdBonuses(gs, player)

		// Award VP from scoring tile
		gs.AwardActionVP(a.PlayerID, ScoringActionStronghold)
	}
}

func (a *UpgradeBuildingAction) handleStrongholdBonuses(gs *GameState, player *Player) {
	switch player.Faction.GetType() {
	case models.FactionAlchemists:
		// Alchemists gain 12 power immediately when building stronghold
		if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
			powerBonus := alchemists.BuildStronghold()
			player.Resources.GainPower(powerBonus)
		}
	case models.FactionCultists:
		// Cultists get +7 VP immediately when building stronghold
		if cultists, ok := player.Faction.(*factions.Cultists); ok {
			vpBonus := cultists.BuildStronghold()
			player.VictoryPoints += vpBonus
		}
	case models.FactionEngineers:
		// Engineers: Mark stronghold as built so GetVPPerBridgeOnPass() returns 3 VP/bridge
		// Engineers get 3 VP per bridge if they have built their stronghold
		if engineers, ok := player.Faction.(*factions.Engineers); ok {
			engineers.BuildStronghold()
		}
	case models.FactionAuren:
		// Auren gets an immediate favor tile when building stronghold
		if auren, ok := player.Faction.(*factions.Auren); ok {
			auren.BuildStronghold()
			// Create pending favor tile selection (1 tile for Auren)
			gs.PendingFavorTileSelection = &PendingFavorTileSelection{
				PlayerID:      a.PlayerID,
				Count:         1,
				SelectedTiles: []FavorTileType{},
			}
		}
	case models.FactionMermaids:
		// Mermaids get +1 shipping level immediately when building stronghold (no cost, but awards VP)
		if mermaids, ok := player.Faction.(*factions.Mermaids); ok {
			mermaids.BuildStronghold()
			// Advance shipping and award VP based on upgrade number (not level)
			// Mermaids start at level 1, so upgrades award: 2/3/4/5 VP for 1st/2nd/3rd/4th upgrade
			currentLevel := mermaids.GetShippingLevel()
			newLevel := currentLevel + 1
			if newLevel <= mermaids.GetMaxShippingLevel() {
				mermaids.SetShippingLevel(newLevel)
				player.ShippingLevel = newLevel

				// Award VP: Mermaids' upgrade number = newLevel - 1
				// So level 2 = 1st upgrade = 2 VP, level 3 = 2nd upgrade = 3 VP, etc.
				vpBonus := newLevel - 1 + 1 // Simplifies to: newLevel
				player.VictoryPoints += vpBonus
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
				PlayerID:         a.PlayerID,
				SpadesRemaining:  3,
				TransformedHexes: []board.Hex{},
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
	case models.FactionDynionGeifr:
		gs.GainPriests(a.PlayerID, 2)
	case models.FactionConspirators:
		gs.PendingFavorTileSelection = &PendingFavorTileSelection{
			PlayerID:      a.PlayerID,
			Count:         1,
			SelectedTiles: []FavorTileType{},
		}
	case models.FactionWisps:
		player.VictoryPoints += 7
		if gs.hasAvailableWispsStrongholdLake() {
			gs.PendingWispsStrongholdDwelling = &PendingWispsStrongholdDwelling{
				PlayerID: a.PlayerID,
			}
		}
	case models.FactionChildrenOfTheWyrm:
		player.Resources.Power.Bowl1 += gs.childrenRemovedPowerTokenCount(a.PlayerID)
	case models.FactionDragonlords:
		player.Resources.Power.Bowl1 += len(gs.Players)
	case models.FactionSnowShamans:
		gs.grantSnowShamansStrongholdDwellings(a.PlayerID)
	case models.FactionProspectors:
		gs.grantPendingPostActionSpecialAction(a.PlayerID, SpecialActionProspectorsGainCoins)
	case models.FactionTimeTravelers:
		gs.grantPendingPostActionSpecialAction(a.PlayerID, SpecialActionTimeTravelersPowerShift)
	case models.FactionArchitects:
		gs.grantPendingPostActionSpecialAction(a.PlayerID, SpecialActionArchitectsMoveBridge)

	case models.FactionGiants:
		// Giants: Mark stronghold as built so ACTG special action becomes available
		if giants, ok := player.Faction.(*factions.Giants); ok {
			giants.BuildStronghold()
		}
	case models.FactionDwarves:
		// Dwarves: Mark stronghold as built so tunneling cost reduces from 2W to 1W
		if dwarves, ok := player.Faction.(*factions.Dwarves); ok {
			dwarves.BuildStronghold()
		}
	case models.FactionFakirs:
		// Fakirs: Mark stronghold as built so carpet flight cost reduces from 1P to 0P
		if fakirs, ok := player.Faction.(*factions.Fakirs); ok {
			fakirs.BuildStronghold()
		}
	default:
		// All other factions just mark stronghold as built (no immediate bonus)
		// This includes: Witches, Swarmlings, Chaos Magicians, Nomads
	}
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

func getDwellingBuildCost(gs *GameState, player *Player, targetHex board.Hex) factions.Cost {
	if player == nil || player.Faction == nil {
		return factions.Cost{}
	}
	baseCost := player.Faction.GetDwellingCost()
	if isSelkies(player) {
		if mapHex := gs.Map.GetHex(targetHex); mapHex != nil && mapHex.Terrain == models.TerrainRiver {
			baseCost.Workers++
		}
	}
	if player.Faction.GetType() != models.FactionChildrenOfTheWyrm {
		return baseCost
	}
	if hasAdjacentOpponent(gs, player, targetHex) {
		baseCost.Coins = 1
	} else {
		baseCost.Coins = 2
	}
	return baseCost
}

func isProspectors(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionProspectors
}

func getProspectorsGoldenSpadeCost(player *Player) int {
	if player != nil && player.HasStrongholdAbility {
		return 3
	}
	return 4
}

// getUpgradeCost calculates the upgrade cost, applying discount if adjacent to opponent
func getUpgradeCost(gs *GameState, player *Player, mapHex *board.MapHex, newBuildingType models.BuildingType) factions.Cost {
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

	if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
		adjacentOpponent := hasAdjacentOpponent(gs, player, mapHex.Coord)
		switch newBuildingType {
		case models.BuildingTradingHouse, models.BuildingTemple, models.BuildingSanctuary, models.BuildingStronghold:
			if adjacentOpponent {
				baseCost.Coins /= 2
			}
		}
		return baseCost
	}

	// Apply discount for Trading House if adjacent to opponent
	if newBuildingType == models.BuildingTradingHouse && hasAdjacentOpponent(gs, player, mapHex.Coord) {
		// Reduce coin cost by half (6 -> 3 for most factions)
		baseCost.Coins /= 2
	}

	return baseCost
}

// hasAdjacentOpponent checks if there's an opponent building adjacent to the hex
func hasAdjacentOpponent(gs *GameState, player *Player, hex board.Hex) bool {
	if player == nil {
		return false
	}
	for _, mapHex := range gs.Map.Hexes {
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID != player.ID && gs.areHexesDirectlyAdjacentForPlayer(player.ID, hex, mapHex.Coord) {
			return true
		}
	}
	return false
}

// AdvanceShippingAction represents advancing shipping level
type AdvanceShippingAction struct {
	BaseAction
}

// NewAdvanceShippingAction creates a new advance shipping action
func NewAdvanceShippingAction(playerID string) *AdvanceShippingAction {
	return &AdvanceShippingAction{
		BaseAction: BaseAction{
			Type:     ActionAdvanceShipping,
			PlayerID: playerID,
		},
	}
}

// Validate checks if the shipping advancement is valid
func (a *AdvanceShippingAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
	}

	// Check if player can advance shipping (some factions like Dwarves/Fakirs cannot)
	// Faction.CanUpgradeShipping() is checked by the faction implementations
	if player.Faction.GetType() == models.FactionRiverwalkers {
		return fmt.Errorf("riverwalkers cannot advance shipping")
	}
	if player.Faction.GetType() == models.FactionSnowShamans {
		return fmt.Errorf("snow shamans advance shipping only when passing")
	}

	// Check if already at max level
	if player.ShippingLevel >= 5 {
		return fmt.Errorf("shipping already at max level")
	}

	// Check if player can afford shipping upgrade
	cost := player.Faction.GetShippingCost(player.ShippingLevel)
	if !gs.canAffordWithReplayAutoConversions(player, cost) {
		return fmt.Errorf("cannot afford shipping upgrade")
	}

	return nil
}

// Execute performs the shipping advancement
func (a *AdvanceShippingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	cost := player.Faction.GetShippingCost(player.ShippingLevel)

	// Pay for upgrade
	if err := gs.spendWithReplayAutoConversions(player, cost); err != nil {
		return fmt.Errorf("failed to pay for shipping: %w", err)
	}

	// Advance shipping and award VP
	if err := gs.AdvanceShippingLevel(a.PlayerID); err != nil {
		return err
	}

	gs.NextTurn()
	return nil
}

// AdvanceDiggingAction represents advancing digging level
type AdvanceDiggingAction struct {
	BaseAction
}

// NewAdvanceDiggingAction creates a new advance digging action
func NewAdvanceDiggingAction(playerID string) *AdvanceDiggingAction {
	return &AdvanceDiggingAction{
		BaseAction: BaseAction{
			Type:     ActionAdvanceDigging,
			PlayerID: playerID,
		},
	}
}

// Validate checks if the digging advancement is valid
func (a *AdvanceDiggingAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
	}

	factionType := player.Faction.GetType()
	if factionType == models.FactionSnowShamans {
		return fmt.Errorf("snow shamans advance digging only when passing")
	}
	maxLevel, err := maxDiggingLevelForFaction(factionType)
	if err != nil {
		return err
	}

	// Check if already at faction's max level
	if player.DiggingLevel >= maxLevel {
		return fmt.Errorf("digging already at max level (%d) for %s", maxLevel, factionType)
	}

	// Check if player can afford digging upgrade
	cost := player.Faction.GetDiggingCost(player.DiggingLevel)
	if !gs.canAffordWithReplayAutoConversions(player, cost) {
		return fmt.Errorf("cannot afford digging upgrade")
	}

	return nil
}

// Execute performs the digging advancement
func (a *AdvanceDiggingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	cost := player.Faction.GetDiggingCost(player.DiggingLevel)

	// Pay for upgrade
	if err := gs.spendWithReplayAutoConversions(player, cost); err != nil {
		return fmt.Errorf("failed to pay for digging: %w", err)
	}

	// Advance digging and award VP (includes scoring tile bonus if applicable)
	if err := gs.AdvanceDiggingLevel(a.PlayerID); err != nil {
		return err
	}

	gs.NextTurn()
	return nil
}

// PassAction represents passing for the round
type PassAction struct {
	BaseAction
	BonusCard *BonusCardType // Bonus card selection (required)
}

// NewPassAction creates a new pass action
func NewPassAction(playerID string, bonusCard *BonusCardType) *PassAction {
	return &PassAction{
		BaseAction: BaseAction{
			Type:     ActionPass,
			PlayerID: playerID,
		},
		BonusCard: bonusCard,
	}
}

func isArchivists(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionArchivists
}

func isDjinni(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionDjinni
}

// Validate checks if the pass action is valid
func (a *PassAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
	}

	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Validate bonus card selection (not required in final round 6)
	if a.BonusCard == nil && gs.Round < 6 {
		return fmt.Errorf("bonus card selection is required when passing")
	}

	if a.BonusCard != nil && !gs.BonusCards.IsAvailable(*a.BonusCard) {
		// When passing, a player returns their current bonus card and may select it again.
		if old, hasOld := gs.BonusCards.GetPlayerCard(a.PlayerID); hasOld && old == *a.BonusCard {
			return nil
		}
		availableCards := []BonusCardType{}
		for cardType := range gs.BonusCards.Available {
			availableCards = append(availableCards, cardType)
		}
		return fmt.Errorf("bonus card %v is not available. Available: %v", *a.BonusCard, availableCards)
	}
	if isArchivists(player) && player.HasStrongholdAbility && a.BonusCard != nil {
		for _, oldCard := range gs.BonusCards.GetPlayerCards(a.PlayerID) {
			if oldCard == *a.BonusCard {
				return fmt.Errorf("archivists cannot take a bonus card they just returned")
			}
		}
	}

	return nil
}

// Execute performs the pass action
func (a *PassAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	prePassSnapshot := gs.CloneForUndo()
	player := gs.GetPlayer(a.PlayerID)
	player.HasPassed = true

	// Record pass order (determines turn order for next round)
	gs.PassOrder = append(gs.PassOrder, a.PlayerID)

	// Award VP from OLD bonus card(s) being returned.
	for _, oldCard := range gs.BonusCards.GetPlayerCards(a.PlayerID) {
		player.VictoryPoints += GetBonusCardPassVP(oldCard, gs, a.PlayerID)
	}
	if isDjinni(player) && player.HasStrongholdAbility {
		player.VictoryPoints += gs.CultTracks.GetTotalPriestsOnCultTracks(a.PlayerID)
	}

	// Take bonus card and get coins from it (unless it's the final round)
	if a.BonusCard != nil {
		if isArchivists(player) && player.HasStrongholdAbility {
			returnedCards := gs.BonusCards.ReturnAllBonusCards(a.PlayerID)
			coins, err := gs.BonusCards.TakeBonusCard(a.PlayerID, *a.BonusCard)
			if err != nil {
				return fmt.Errorf("failed to take archivists first bonus card: %w", err)
			}
			player.Resources.Coins += coins
			player.Resources.Power.GainPower(coins * 2)
			gs.PendingArchivistsBonusSelection = &PendingArchivistsBonusSelection{
				PlayerID:      a.PlayerID,
				ReturnedCards: returnedCards,
				UndoSnapshot:  prePassSnapshot,
			}
			return nil
		}
		coins, err := gs.BonusCards.TakeBonusCard(a.PlayerID, *a.BonusCard)
		if err != nil {
			return fmt.Errorf("failed to take bonus card: %w", err)
		}
		player.Resources.Coins += coins
		if isArchivists(player) {
			player.Resources.Power.GainPower(coins * 2)
		}
	}

	gs.ApplyAutoConvertOnPass(a.PlayerID)
	applyPostPassBonuses(gs, player)

	return advanceAfterCompletedPass(gs)
}

func applyPostPassBonuses(gs *GameState, player *Player) {
	if gs == nil || player == nil {
		return
	}

	// Award VP from Air+1 favor tile (VP based on Trading House count).
	playerTiles := gs.FavorTiles.GetPlayerTiles(player.ID)
	if HasFavorTile(playerTiles, FavorAir1) {
		tradingHouseCount := 0
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil &&
				mapHex.Building.PlayerID == player.ID &&
				mapHex.Building.Type == models.BuildingTradingHouse {
				tradingHouseCount++
			}
		}

		player.VictoryPoints += GetAir1PassVP(playerTiles, tradingHouseCount)
	}

	// Award VP for Engineers stronghold ability (3 VP per bridge connecting two structures when passing).
	if player.Faction != nil && player.Faction.GetType() == models.FactionEngineers && player.HasStrongholdAbility {
		bridgeCount := gs.Map.CountBridgesConnectingPlayerStructures(player.ID)
		player.VictoryPoints += bridgeCount * 3
	}

	if isIceMaidens(player) && player.HasStrongholdAbility {
		templeCount := 0
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil && mapHex.Building.PlayerID == player.ID && mapHex.Building.Type == models.BuildingTemple {
				templeCount++
			}
		}
		player.VictoryPoints += templeCount * 3
	}

	if isFirewalkers(player) && player.HasStrongholdAbility {
		player.VictoryPoints += gs.countDirectBuildingGroups(player.ID)
	}

	if isSnowShamans(player) {
		if maxDigging, err := maxDiggingLevelForFaction(player.Faction.GetType()); err == nil && player.DiggingLevel < maxDigging {
			player.DiggingLevel++
			gs.updateFactionDiggingLevel(player)
		} else if player.ShippingLevel < 5 {
			player.ShippingLevel++
		}
	}
}

func (gs *GameState) countDirectBuildingGroups(playerID string) int {
	hexes := gs.getPlayerBuildingHexes(playerID)
	visited := make(map[board.Hex]bool, len(hexes))
	groups := 0
	for _, start := range hexes {
		if visited[start] {
			continue
		}
		groups++
		queue := []board.Hex{start}
		visited[start] = true
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, neighbor := range gs.Map.GetDirectNeighbors(current) {
				if visited[neighbor] {
					continue
				}
				mapHex := gs.Map.GetHex(neighbor)
				if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
					continue
				}
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return groups
}

func advanceAfterCompletedPass(gs *GameState) error {
	if gs == nil {
		return nil
	}
	isReplaySimulation := gs.ReplayMode != nil && gs.ReplayMode["__replay__"]
	if roundComplete := gs.NextTurn(); roundComplete && !isReplaySimulation {
		if gs.HasLateRoundPendingDecisions() {
			return nil
		}
		advanceAfterRoundComplete(gs)
	}
	return nil
}

func (gs *GameState) HasLateRoundPendingDecisions() bool {
	if gs == nil {
		return false
	}
	return gs.HasPendingLeechOffers() ||
		gs.PendingCultistsCultSelection != nil ||
		gs.PendingGoblinsCultSteps != nil ||
		gs.PendingTreasurersDeposit != nil ||
		gs.PendingArchivistsBonusSelection != nil
}

func advanceAfterRoundComplete(gs *GameState) {
	if gs == nil {
		return
	}
	justCompletedRound := gs.Round
	if gs.ExecuteCleanupPhase() {
		gs.StartNewRound()
		gs.AwardCultRewardsForRound(justCompletedRound)
		if _, count := gs.GetPendingCultRewardSpadePlayer(); count == 0 {
			gs.GrantIncome()
			if gs.PendingTreasurersDeposit == nil {
				gs.StartActionPhase()
			}
		}
	}
}

// SendPriestToCultAction represents sending a priest to a cult track
type SendPriestToCultAction struct {
	BaseAction
	Track         CultTrack
	SpacesToClimb int // Number of spaces to advance (1-3), always costs 1 priest
}

// GetType returns the action type
func (a *SendPriestToCultAction) GetType() ActionType {
	return ActionSendPriestToCult
}

// Validate checks if the send priest action is valid
func (a *SendPriestToCultAction) Validate(gs *GameState) error {
	player, err := gs.ValidatePlayer(a.PlayerID)
	if err != nil {
		return err
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

// Execute performs the send priest action
func (a *SendPriestToCultAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)

	// Remove 1 priest from player's supply (cost is always 1 priest, regardless of spaces)
	player.Resources.Priests--

	// Advance on cult track (with bonus power at milestones)
	// Note: It's valid to sacrifice a priest even if you can't advance (no refund)
	// Position 10 requires a key (checked in gs.AdvanceCultTrack)
	gs.AdvanceCultTrack(a.PlayerID, a.Track, a.SpacesToClimb)
	if isAcolytes(player) && player.HasStrongholdAbility {
		gs.AdvanceCultTrack(a.PlayerID, a.Track, 1)
	}

	// Track priest placement on cult track action spaces
	// In Terra Mystica, each track has 4 action spaces: one 3-step and three 2-step
	// If placed on action space (2 or 3 steps), priest stays permanently and counts for SCORE5
	// If sacrificed (1 step), priest is removed and doesn't count toward limit or SCORE5
	if a.SpacesToClimb >= 2 {
		// Priest is placed on an action space (2-step or 3-step)
		gs.CultTracks.PriestsOnActionSpaces[a.PlayerID][a.Track]++

		// Record priest on the specific track spot (3 or 2) for UI display
		if gs.CultTracks.PriestsOnTrack == nil {
			gs.CultTracks.PriestsOnTrack = make(map[CultTrack]map[int][]string)
		}
		if gs.CultTracks.PriestsOnTrack[a.Track] == nil {
			gs.CultTracks.PriestsOnTrack[a.Track] = make(map[int][]string)
		}

		// Append player to the list for this spot value
		// Note: We don't strictly enforce the limit here (e.g. max 4 spots for "2")
		// because that should be handled by Validate() checking available spots.
		// But for now, we just record it.
		gs.CultTracks.PriestsOnTrack[a.Track][a.SpacesToClimb] = append(gs.CultTracks.PriestsOnTrack[a.Track][a.SpacesToClimb], a.PlayerID)

		// Record priest sent for scoring tile #5 (Temple + Priest: 2 coins per priest on action space)
		if gs.ScoringTiles != nil {
			gs.ScoringTiles.RecordPriestSent(a.PlayerID)
		}
	}
	// If SpacesToClimb == 1, priest is sacrificed (no tracking needed, no SCORE5 reward)

	gs.NextTurn()
	return nil
}

// ValidateSkipAbility checks if a player can use their faction's skip ability (Carpet Flight/Tunneling)
// Returns error if not valid, or nil if valid.
// Also validates if the player has enough resources (but does not spend them).
func ValidateSkipAbility(gs *GameState, player *Player, targetHex board.Hex) error {
	// Check if already used this action
	if gs.SkipAbilityUsedThisAction != nil {
		usedHexes := gs.SkipAbilityUsedThisAction[player.ID]
		for _, h := range usedHexes {
			if h == targetHex {
				// Already paid for this hex, so valid (skip resource checks)
				return nil
			}
		}
	}

	// Determine if the player's faction has a skip ability and validate its use.
	// This function also checks for resource costs but does not deduct them.
	switch f := player.Faction.(type) {
	case *factions.Fakirs:
		// Calculate skip range for Fakirs
		skipRange := f.GetFlightRange()
		if !gs.Map.IsWithinSkipRange(targetHex, player.ID, skipRange) {
			return fmt.Errorf("target hex is not within carpet flight range %d", skipRange)
		}
		// Check if player has priest to pay
		if player.Resources.Priests < 1 {
			return fmt.Errorf("not enough priests for carpet flight: need 1, have %d", player.Resources.Priests)
		}
	case *factions.Dwarves:
		// Dwarves can tunnel 1 space
		if !gs.Map.IsWithinSkipRange(targetHex, player.ID, 1) {
			return fmt.Errorf("target hex is not within tunneling range 1")
		}
		// Check if player has workers to pay
		workerCost := 2
		if player.HasStrongholdAbility {
			workerCost = 1
		}
		if player.Resources.Workers < workerCost {
			return fmt.Errorf("not enough workers for tunneling: need %d, have %d", workerCost, player.Resources.Workers)
		}
	default:
		return fmt.Errorf("only Fakirs and Dwarves can use skip ability")
	}
	return nil
}

// PaySkipCost deducts the cost for using skip ability and awards VP
func PaySkipCost(player *Player) {
	if player.Faction.GetType() == models.FactionFakirs {
		// Pay priest for carpet flight
		player.Resources.Priests--
		// Award VP bonus
		player.VictoryPoints += 4
	} else if player.Faction.GetType() == models.FactionDwarves {
		// Pay workers for tunneling
		workerCost := 2
		if player.HasStrongholdAbility {
			workerCost = 1
		}
		player.Resources.Workers -= workerCost
		// Award VP bonus
		player.VictoryPoints += 4
	}
}
