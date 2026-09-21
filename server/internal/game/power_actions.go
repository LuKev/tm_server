package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// PowerActionType represents the different power actions available on the game board
type PowerActionType int

const (
	PowerActionBridge  PowerActionType = iota // 3 Power: Build a bridge
	PowerActionPriest                         // 3 Power: Gain 1 priest
	PowerActionWorkers                        // 4 Power: Gain 2 workers
	PowerActionCoins                          // 4 Power: Gain 7 coins
	PowerActionSpade1                         // 4 Power: 1 free spade for transform
	PowerActionSpade2                         // 6 Power: 2 free spades for transform
	PowerActionUnknown PowerActionType = -1
)

// PowerActionTypeFromString converts a string to a PowerActionType
// PowerActionTypeFromString converts a string to a PowerActionType
func PowerActionTypeFromString(s string) PowerActionType {
	switch s {
	case "Bridge":
		return PowerActionBridge
	case "Priest":
		return PowerActionPriest
	case "2 Workers":
		return PowerActionWorkers
	case "7 Coins":
		return PowerActionCoins
	case "Spade":
		return PowerActionSpade1
	case "2 Spades":
		return PowerActionSpade2
	default:
		return PowerActionUnknown
	}
}

// PowerActionState tracks which power actions have been used this round
type PowerActionState struct {
	UsedActions map[PowerActionType]bool // Tracks which actions have been taken this round
}

// NewPowerActionState creates a new power action state
func NewPowerActionState() *PowerActionState {
	return &PowerActionState{
		UsedActions: make(map[PowerActionType]bool),
	}
}

// ResetForNewRound resets all power actions for a new round
func (pas *PowerActionState) ResetForNewRound() {
	pas.UsedActions = make(map[PowerActionType]bool)
}

// IsAvailable checks if a power action is still available this round
func (pas *PowerActionState) IsAvailable(actionType PowerActionType) bool {
	return !pas.UsedActions[actionType]
}

// MarkUsed marks a power action as used for this round
func (pas *PowerActionState) MarkUsed(actionType PowerActionType) {
	pas.UsedActions[actionType] = true
}

// GetPowerCost returns the power cost for a given power action
func GetPowerCost(actionType PowerActionType) int {
	switch actionType {
	case PowerActionBridge, PowerActionPriest:
		return 3
	case PowerActionWorkers, PowerActionCoins, PowerActionSpade1:
		return 4
	case PowerActionSpade2:
		return 6
	default:
		return 0
	}
}

// PowerAction represents taking a power action from the game board
type PowerAction struct {
	BaseAction
	ActionType    PowerActionType
	UseCoins      bool
	DeclineReward bool // Pay/occupy the shared action without taking its reward.
	// For spade actions, these fields specify the transform details
	TargetHex           *board.Hex          // Optional: for spade actions
	TargetTerrain       *models.TerrainType // Nil preserves the legacy home-terrain request.
	BuildDwelling       bool                // Optional: for spade actions
	UseSkip             bool                // Optional: for spade actions (Fakirs/Dwarves skip)
	SecondTargetHex     *board.Hex
	SecondTargetTerrain *models.TerrainType
	SecondUseSkip       bool
	// For bridge action, these fields specify the bridge endpoints
	BridgeHex1 *board.Hex // Optional: for bridge action
	BridgeHex2 *board.Hex // Optional: for bridge action
}

// NewPowerAction creates a new power action
func NewPowerAction(playerID string, actionType PowerActionType) *PowerAction {
	return &PowerAction{
		BaseAction: BaseAction{
			Type:     ActionPowerAction,
			PlayerID: playerID,
		},
		ActionType: actionType,
	}
}

// NewPowerActionWithTransform requests one destination (legacy home terrain).
// Set SecondTargetHex/SecondTargetTerrain for an atomic ACT6 split. Unused spades
// are forfeited; this action never opens a live spade-followup decision.
func NewPowerActionWithTransform(playerID string, actionType PowerActionType, targetHex board.Hex, buildDwelling bool) *PowerAction {
	return &PowerAction{
		BaseAction: BaseAction{
			Type:     ActionPowerAction,
			PlayerID: playerID,
		},
		ActionType:    actionType,
		TargetHex:     &targetHex,
		BuildDwelling: buildDwelling,
	}
}

// NewPowerActionWithBridge creates a power action for building a bridge
func NewPowerActionWithBridge(playerID string, hex1, hex2 board.Hex) *PowerAction {
	return &PowerAction{
		BaseAction: BaseAction{
			Type:     ActionPowerAction,
			PlayerID: playerID,
		},
		ActionType: PowerActionBridge,
		BridgeHex1: &hex1,
		BridgeHex2: &hex2,
	}
}

// Validate checks if the power action is valid
func (a *PowerAction) Validate(gs *GameState) error {
	if a.ActionType < PowerActionBridge || a.ActionType > PowerActionSpade2 {
		return fmt.Errorf("unknown power action %d", a.ActionType)
	}
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if this power action is still available
	if !gs.PowerActions.IsAvailable(a.ActionType) && !(player.Faction != nil && player.Faction.GetType() == models.FactionYetis && player.HasStrongholdAbility) {
		return fmt.Errorf("power action %v has already been taken this round", a.ActionType)
	}

	powerCost := getPowerActionCostForPlayer(player, a.ActionType)
	if a.shouldPayWithCoins(player) {
		if !isChashUsingCoinPowerActions(player) {
			return fmt.Errorf("only Chash Dallah with stronghold may pay coins for power actions")
		}
		if player.Resources.Coins < powerCost {
			return fmt.Errorf("not enough coins for power action: need %d, have %d", powerCost, player.Resources.Coins)
		}
	} else {
		requiredBurn := a.requiredAutoBurn(player)
		canBurn := player.Resources.Power.CanBurn(requiredBurn)
		if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
			canBurn = player.Resources.Power.CanBurnChildren(requiredBurn)
		}
		if requiredBurn > 0 && !canBurn {
			return fmt.Errorf("not enough power for action: need %d in Bowl III, have %d and cannot auto-burn %d more from Bowl II", powerCost, player.Resources.Power.Bowl3, requiredBurn)
		}
	}

	if isRiverwalkers(player) && (a.ActionType == PowerActionSpade1 || a.ActionType == PowerActionSpade2) {
		return fmt.Errorf("riverwalkers cannot take spade power actions")
	}
	if a.DeclineReward {
		if a.TargetHex != nil || a.TargetTerrain != nil || a.SecondTargetHex != nil || a.SecondTargetTerrain != nil || a.BridgeHex1 != nil || a.BridgeHex2 != nil || a.BuildDwelling || a.UseSkip || a.SecondUseSkip {
			return fmt.Errorf("declined reward cannot include placement parameters")
		}
		return nil
	}
	// Validate spade actions
	if a.ActionType == PowerActionSpade1 || a.ActionType == PowerActionSpade2 {
		if !isProspectors(player) && !factionConvertsSpadeRewards(player) {
			if err := a.validateSpadeTransform(gs, player, powerCost); err != nil {
				return err
			}
		}
	}

	// Validate bridge action
	if a.ActionType == PowerActionBridge {
		if err := a.validateBridgeAction(gs, player); err != nil {
			return err
		}
	}
	if a.ActionType == PowerActionPriest && gs.RemainingPriestCapacity(a.PlayerID) < 1 && (!isRiverwalkers(player) || !gs.riverwalkersHasAffordableUnlockOption(player)) {
		return fmt.Errorf("cannot take priest power action at the 7-priest limit")
	}
	if isProspectors(player) && (a.ActionType == PowerActionSpade1 || a.ActionType == PowerActionSpade2) {
		requiredPriests := 1
		if a.ActionType == PowerActionSpade2 {
			requiredPriests = 2
		}
		if gs.RemainingPriestCapacity(a.PlayerID) < requiredPriests {
			return fmt.Errorf("not enough priest capacity for prospectors spade action")
		}
	}

	return nil
}

// validateSpadeTransform checks the transform plan only. Bonus-card spades
// reuse this helper with zero power charge, without claiming a shared action.
func (a *PowerAction) validateSpadeTransform(gs *GameState, player *Player, powerCost int) error {
	if a.TargetHex == nil {
		if a.BuildDwelling || a.UseSkip || a.TargetTerrain != nil || a.SecondTargetHex != nil || a.SecondTargetTerrain != nil || a.SecondUseSkip {
			return fmt.Errorf("unused spade action cannot include transform parameters")
		}
		return nil // Taking a shared action and forfeiting its reward is legal.
	}

	// Validate the transform would be legal
	// This is similar to TransformAndBuild validation
	mapHex := gs.Map.GetHex(*a.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.TargetHex)
	}
	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}
	targetTerrain := a.targetTerrain(player)
	distance, err := TerraformSpadeCount(player, mapHex.Terrain, targetTerrain, 0)
	if err != nil {
		return err
	}
	if distance <= 0 {
		return fmt.Errorf("spade power action must transform terrain")
	}
	if player.Faction.GetType() == models.FactionGiants && targetTerrain != effectiveHomeTerrain(player) {
		return fmt.Errorf("giants may only transform into home terrain")
	}
	if a.BuildDwelling && targetTerrain != effectiveHomeTerrain(player) {
		return fmt.Errorf("dwelling requires home terrain")
	}
	freeSpades := 1
	if a.ActionType == PowerActionSpade2 {
		freeSpades = 2
	}
	requiredSpades, err := a.requiredSpadesForTransform(gs, player)
	if err != nil {
		return err
	}
	if requiredSpades > freeSpades {
		homeDistance, err := TerraformSpadeCount(player, mapHex.Terrain, effectiveHomeTerrain(player), 0)
		if err != nil {
			return err
		}
		remainingDistance, err := TerraformSpadeCount(player, targetTerrain, effectiveHomeTerrain(player), 0)
		if err != nil {
			return err
		}
		if player.Faction.GetType() != models.FactionGiants && homeDistance != distance+remainingDistance {
			return fmt.Errorf("paid extra spades must follow the shortest route toward home terrain")
		}
	}

	// Check adjacency (or skip range for Fakirs/Dwarves)
	isAdjacent := gs.IsAdjacentToPlayerBuilding(*a.TargetHex, a.PlayerID)
	if isAdjacent && a.UseSkip {
		return fmt.Errorf("cannot tunnel or fly to an already adjacent space")
	}
	if !isAdjacent && !a.UseSkip {
		factionType := player.Faction.GetType()
		if factionType == models.FactionDwarves || factionType == models.FactionFakirs {
			// Match transform/build behavior: auto-enable skip when needed.
			a.UseSkip = true
		}
	}
	if a.UseSkip {
		if err := ValidateSkipAbility(gs, player, *a.TargetHex); err != nil {
			return err
		}
	} else {
		if !isAdjacent {
			return fmt.Errorf("hex is not adjacent to player's buildings")
		}
	}
	if a.SecondTargetHex != nil {
		if a.ActionType != PowerActionSpade2 || requiredSpades != 1 || targetTerrain != effectiveHomeTerrain(player) {
			return fmt.Errorf("second space requires one free spade completing first space to home")
		}
		if *a.SecondTargetHex == *a.TargetHex || a.SecondTargetTerrain == nil {
			return fmt.Errorf("second space requires a distinct hex and explicit terrain")
		}
		second := gs.Map.GetHex(*a.SecondTargetHex)
		if second == nil || second.Building != nil {
			return fmt.Errorf("second space must be an empty hex")
		}
		distance, err := TerraformSpadeCount(player, second.Terrain, *a.SecondTargetTerrain, 0)
		if err != nil || distance != 1 {
			return fmt.Errorf("second space must use exactly one free spade")
		}
		secondAdjacent := gs.IsAdjacentToPlayerBuilding(*a.SecondTargetHex, a.PlayerID)
		if secondAdjacent && a.SecondUseSkip {
			return fmt.Errorf("cannot tunnel or fly to an already adjacent second space")
		}
		if !secondAdjacent {
			a.SecondUseSkip = true
		}
		if a.SecondUseSkip {
			if a.UseSkip {
				return fmt.Errorf("cannot tunnel or fly twice in one action")
			}
			if err := ValidateSkipAbility(gs, player, *a.SecondTargetHex); err != nil {
				return err
			}
		}
	} else if a.SecondTargetTerrain != nil || a.SecondUseSkip {
		return fmt.Errorf("second transform parameters require a second hex")
	}
	// Validate all costs before spending power or marking the shared action used.
	copyPlayer := *player
	copyResources := *player.Resources
	copyPower := *player.Resources.Power
	copyResources.Power = &copyPower
	copyPlayer.Resources = &copyResources
	if powerCost > 0 && a.shouldPayWithCoins(player) {
		copyResources.Coins -= powerCost
	} else if powerCost > 0 {
		burn := a.requiredAutoBurn(player)
		if player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
			_ = copyPower.BurnPowerChildren(burn)
		} else {
			_ = copyPower.BurnPower(burn)
		}
		if err := copyPower.SpendPower(powerCost); err != nil {
			return err
		}
	}
	if a.UseSkip {
		PaySkipCost(&copyPlayer)
	}
	if a.SecondUseSkip {
		PaySkipCost(&copyPlayer)
	}
	if err := a.paySpadeCosts(&copyPlayer, a.calculateRemainingSpades(requiredSpades, freeSpades)); err != nil {
		return err
	}
	if a.BuildDwelling {
		if err := gs.CheckBuildingLimit(a.PlayerID, models.BuildingDwelling); err != nil {
			return err
		}
		if err := copyResources.Spend(getDwellingBuildCost(gs, player, *a.TargetHex)); err != nil {
			return err
		}
	}
	return nil
}

func (a *PowerAction) targetTerrain(player *Player) models.TerrainType {
	if a.TargetTerrain != nil {
		return *a.TargetTerrain
	}
	return effectiveHomeTerrain(player)
}

func (a *PowerAction) validateBridgeAction(gs *GameState, player *Player) error {
	// Check if player has bridges remaining (max 3)
	if player.BridgesBuilt >= 3 {
		return fmt.Errorf("player has already built 3 bridges (maximum)")
	}

	if a.BridgeHex1 == nil || a.BridgeHex2 == nil {
		return fmt.Errorf("bridge power action requires two endpoint coordinates")
	}
	if err := gs.Map.ValidateBridgePlacement(*a.BridgeHex1, *a.BridgeHex2); err != nil {
		return err
	}
	first := gs.Map.GetHex(*a.BridgeHex1)
	second := gs.Map.GetHex(*a.BridgeHex2)
	firstOwned := first.Building != nil && first.Building.PlayerID == a.PlayerID
	secondOwned := second.Building != nil && second.Building.PlayerID == a.PlayerID
	if !firstOwned && !secondOwned {
		return fmt.Errorf("bridge must connect to at least one of your structures")
	}
	return nil
}

// Execute performs the power action
func (a *PowerAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	powerCost := getPowerActionCostForPlayer(player, a.ActionType)
	if a.shouldPayWithCoins(player) {
		player.Resources.Coins -= powerCost
	} else {
		if requiredBurn := a.requiredAutoBurn(player); requiredBurn > 0 {
			var err error
			if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
				err = player.Resources.Power.BurnPowerChildren(requiredBurn)
			} else {
				err = player.Resources.Power.BurnPower(requiredBurn)
			}
			if err != nil {
				return fmt.Errorf("failed to auto-burn power for action: %w", err)
			}
		}

		// Move power from Bowl III to Bowl I
		if err := player.Resources.Power.SpendPower(powerCost); err != nil {
			return err
		}
	}

	// Mark action as used
	gs.PowerActions.MarkUsed(a.ActionType)
	if a.DeclineReward {
		gs.NextTurn()
		return nil
	}

	// Execute the specific action
	switch a.ActionType {
	case PowerActionBridge:
		// Place the bridge on the map if coordinates provided
		if a.BridgeHex1 != nil && a.BridgeHex2 != nil {
			if err := gs.Map.BuildBridge(*a.BridgeHex1, *a.BridgeHex2, a.PlayerID); err != nil {
				return fmt.Errorf("failed to build bridge: %w", err)
			}

			// Check for town formation after building bridge
			// The bridge might connect buildings into a town
			// Check from both endpoints - CheckForTownFormation handles appending to PendingTownFormations
			gs.CheckForTownFormation(a.PlayerID, *a.BridgeHex1)
			gs.CheckForTownFormation(a.PlayerID, *a.BridgeHex2)
			gs.updateAtlanteansStrongholdTown(a.PlayerID)
		}

		player.BridgesBuilt++

	case PowerActionPriest:
		gs.GainPriestsForReason(a.PlayerID, 1, "power_action")

	case PowerActionWorkers:
		player.Resources.Workers += 2

	case PowerActionCoins:
		player.Resources.Coins += 7

	case PowerActionSpade1, PowerActionSpade2:
		spades := 1
		if a.ActionType == PowerActionSpade2 {
			spades = 2
		}
		if factionConvertsSpadeRewards(player) {
			gs.convertFactionSpadeReward(a.PlayerID, spades, true)
			break
		}
		if isProspectors(player) {
			priests := 1
			if a.ActionType == PowerActionSpade2 {
				priests = 2
			}
			for i := 0; i < spades; i++ {
				gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
			}
			gs.GainPriests(a.PlayerID, priests)
			break
		}

		// The spade action gives free spades for a transform
		// The actual transform happens as part of this action
		if a.TargetHex == nil {
			break
		}

		freeSpadesFromAction := 1
		if a.ActionType == PowerActionSpade2 {
			freeSpadesFromAction = 2
		}

		// Execute the transform with free spades
		err := a.executeTransformWithFreeSpades(gs, player, freeSpadesFromAction)
		if err != nil {
			return fmt.Errorf("failed to execute transform: %w", err)
		}

		if a.SecondTargetHex != nil {
			if a.SecondUseSkip {
				PaySkipCost(player)
			}
			if err := gs.Map.TransformTerrain(*a.SecondTargetHex, *a.SecondTargetTerrain); err != nil {
				return err
			}
			a.awardSpadeBonuses(gs, player, 1)
		}
		// Both destinations were validated against pre-action buildings. Build
		// only after all transformations, so reachability and reactions cannot
		// be changed halfway through the shared action.
		if a.BuildDwelling {
			if err := a.buildDwelling(gs, player); err != nil {
				return err
			}
		}
	}

	gs.NextTurn()
	return nil
}

func (a *PowerAction) requiredAutoBurn(player *Player) int {
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		return 0
	}
	if a.shouldPayWithCoins(player) {
		return 0
	}

	powerCost := getPowerActionCostForPlayer(player, a.ActionType)
	if player.Resources.Power.Bowl3 >= powerCost {
		return 0
	}

	if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
		return (powerCost - player.Resources.Power.Bowl3 + 1) / 2
	}

	return powerCost - player.Resources.Power.Bowl3
}

func isChashUsingCoinPowerActions(player *Player) bool {
	return player != nil &&
		player.Faction != nil &&
		player.Faction.GetType() == models.FactionChashDallah &&
		player.HasStrongholdAbility
}

func (a *PowerAction) shouldPayWithCoins(player *Player) bool {
	if !isChashUsingCoinPowerActions(player) {
		return false
	}
	if a.UseCoins {
		return true
	}
	powerCost := getPowerActionCostForPlayer(player, a.ActionType)
	return a.requiredAutoBurnWithoutCoins(player) > 0 && player.Resources.Coins >= powerCost
}

func (a *PowerAction) requiredAutoBurnWithoutCoins(player *Player) int {
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		return 0
	}
	powerCost := getPowerActionCostForPlayer(player, a.ActionType)
	if player.Resources.Power.Bowl3 >= powerCost {
		return 0
	}
	return powerCost - player.Resources.Power.Bowl3
}

func (a *PowerAction) requiredSpadesForTransform(gs *GameState, player *Player) (int, error) {
	if a.TargetHex == nil {
		return 0, fmt.Errorf("spade action requires target hex")
	}

	mapHex := gs.Map.GetHex(*a.TargetHex)
	if mapHex == nil {
		return 0, fmt.Errorf("hex does not exist: %v", *a.TargetHex)
	}

	targetTerrain := a.targetTerrain(player)
	distance, err := TerraformSpadeCount(player, mapHex.Terrain, targetTerrain, 0)
	if err != nil {
		return 0, err
	}
	if distance == 0 {
		return 0, fmt.Errorf("hex is already home terrain")
	}
	if player.Faction.GetType() == models.FactionGiants {
		return 2, nil
	}
	return distance, nil
}

// executeTransformWithFreeSpades handles the transform part of spade power actions
func (a *PowerAction) executeTransformWithFreeSpades(gs *GameState, player *Player, freeSpades int) error {
	mapHex := gs.Map.GetHex(*a.TargetHex)

	// Handle skip costs (Fakirs carpet flight / Dwarves tunneling)
	if a.UseSkip {
		PaySkipCost(player)
	}

	// Calculate spades needed
	currentTerrain := mapHex.Terrain
	targetTerrain := a.targetTerrain(player)
	distance, err := TerraformSpadeCount(player, currentTerrain, targetTerrain, 0)
	if err != nil {
		return err
	}
	requiredSpades := distance
	if player.Faction.GetType() == models.FactionGiants {
		requiredSpades = 2
	}

	if distance == 0 {
		return fmt.Errorf("hex is already home terrain")
	}

	remainingSpades := a.calculateRemainingSpades(requiredSpades, freeSpades)

	// Pay for remaining spades
	if err := a.paySpadeCosts(player, remainingSpades); err != nil {
		return err
	}

	// Transform the terrain
	if err := gs.Map.TransformTerrain(*a.TargetHex, targetTerrain); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Award VP from scoring tile for ALL spades used (both free and paid)
	a.awardSpadeBonuses(gs, player, requiredSpades)

	return nil
}

func (a *PowerAction) calculateRemainingSpades(distance, freeSpades int) int {
	spadesNeeded := distance
	spadesFromFreeAction := freeSpades
	if spadesFromFreeAction > spadesNeeded {
		spadesFromFreeAction = spadesNeeded
	}
	return spadesNeeded - spadesFromFreeAction
}

func (a *PowerAction) paySpadeCosts(player *Player, remainingSpades int) error {
	if remainingSpades > 0 {
		// Darklings pay priests (instead of workers)
		if player.Faction.GetType() == models.FactionDarklings {
			priestsNeeded := remainingSpades
			if player.Resources.Priests < priestsNeeded {
				return fmt.Errorf("not enough priests: need %d, have %d", priestsNeeded, player.Resources.Priests)
			}
			player.Resources.Priests -= priestsNeeded

			// Award Darklings VP bonus (+2 VP per remaining spade)
			vpBonus := remainingSpades * 2
			player.VictoryPoints += vpBonus
		} else if player.Faction.GetType() == models.FactionTheEnlightened {
			powerNeeded := player.Faction.GetTerraformCost(remainingSpades)
			if !player.Resources.Power.CanSpend(powerNeeded) {
				return fmt.Errorf("not enough power: need %d, have %d", powerNeeded, player.Resources.Power.Bowl3)
			}
			if err := player.Resources.Power.SpendPower(powerNeeded); err != nil {
				return err
			}
		} else {
			// Other factions pay workers
			workersNeeded := player.Faction.GetTerraformCost(remainingSpades)
			if player.Faction.GetType() == models.FactionGiants {
				workersNeeded = remainingSpades * (player.Faction.GetTerraformCost(2) / 2)
			}
			if player.Resources.Workers < workersNeeded {
				return fmt.Errorf("not enough workers: need %d, have %d", workersNeeded, player.Resources.Workers)
			}
			player.Resources.Workers -= workersNeeded
		}
	}
	return nil
}

func (a *PowerAction) awardSpadeBonuses(gs *GameState, player *Player, totalSpades int) {
	// Power action spades (ACT5/ACT6) count for scoring, unlike cult reward spades
	for i := 0; i < totalSpades; i++ {
		gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
	}
	// Darklings' priest bonus is paid only in paySpadeCosts, never for free
	// spades; their round-tile scoring still counts every spade used.
	AwardFactionSpadeBonuses(player, totalSpades)
}

// Fixed version of executeTransformWithFreeSpades with correct helper usage
func (a *PowerAction) buildDwelling(gs *GameState, player *Player) error {
	// Check building limit
	if err := gs.CheckBuildingLimit(a.PlayerID, models.BuildingDwelling); err != nil {
		return err
	}

	// Pay for dwelling
	dwellingCost := getDwellingBuildCost(gs, player, *a.TargetHex)
	if err := player.Resources.Spend(dwellingCost); err != nil {
		return fmt.Errorf("failed to pay for dwelling: %w", err)
	}

	// Place dwelling and handle all VP bonuses
	if err := gs.BuildDwelling(a.PlayerID, *a.TargetHex); err != nil {
		return err
	}
	return nil
}
