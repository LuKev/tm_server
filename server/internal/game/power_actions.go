package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
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
	ActionType PowerActionType
	// For spade actions, these fields specify the transform details
	TargetHex     *board.Hex // Optional: for spade actions
	BuildDwelling bool       // Optional: for spade actions
	UseSkip       bool       // Optional: for spade actions (Fakirs/Dwarves skip)
	// For bridge action, these fields specify the bridge endpoints
	BridgeHex1 *board.Hex // Optional: for bridge action
	BridgeHex2 *board.Hex // Optional: for bridge action
}

func NewPowerAction(playerID string, actionType PowerActionType) *PowerAction {
	return &PowerAction{
		BaseAction: BaseAction{
			Type:     ActionPowerAction,
			PlayerID: playerID,
		},
		ActionType: actionType,
	}
}

// NewPowerActionWithTransform creates a power action that includes a transform
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

func (a *PowerAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if this power action is still available
	if !gs.PowerActions.IsAvailable(a.ActionType) {
		return fmt.Errorf("power action %v has already been taken this round", a.ActionType)
	}

	// Check if player has enough power in Bowl III
	powerCost := GetPowerCost(a.ActionType)
	if player.Resources.Power.Bowl3 < powerCost {
		return fmt.Errorf("not enough power in Bowl III: need %d, have %d", powerCost, player.Resources.Power.Bowl3)
	}

	// Validate spade actions
	if a.ActionType == PowerActionSpade1 || a.ActionType == PowerActionSpade2 {
		if a.TargetHex == nil {
			return fmt.Errorf("spade power action requires a target hex")
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

		// Check adjacency (or skip range for Fakirs/Dwarves)
		if a.UseSkip {
			// Validate skip ability usage (same as TransformAndBuildAction)
			if fakirs, ok := player.Faction.(*factions.Fakirs); ok {
				if !fakirs.CanCarpetFlight() {
					return fmt.Errorf("Fakirs cannot use carpet flight")
				}
				skipRange := fakirs.GetCarpetFlightRange()
				if !gs.Map.IsWithinSkipRange(*a.TargetHex, a.PlayerID, skipRange) {
					return fmt.Errorf("target hex is not within carpet flight range %d", skipRange)
				}
				if player.Resources.Priests < 1 {
					return fmt.Errorf("not enough priests for carpet flight")
				}
			} else if dwarves, ok := player.Faction.(*factions.Dwarves); ok {
				if !dwarves.CanTunnel() {
					return fmt.Errorf("Dwarves cannot tunnel")
				}
				if !gs.Map.IsWithinSkipRange(*a.TargetHex, a.PlayerID, 1) {
					return fmt.Errorf("target hex is not within tunneling range 1")
				}
				workerCost := dwarves.GetTunnelingCost()
				if player.Resources.Workers < workerCost {
					return fmt.Errorf("not enough workers for tunneling")
				}
			} else {
				return fmt.Errorf("only Fakirs and Dwarves can use skip ability")
			}
		} else {
			if !gs.IsAdjacentToPlayerBuilding(*a.TargetHex, a.PlayerID) {
				return fmt.Errorf("hex is not adjacent to player's buildings")
			}
		}
	}

	// Validate bridge action
	if a.ActionType == PowerActionBridge {
		// Check if player has bridges remaining (max 3)
		if player.BridgesBuilt >= 3 {
			return fmt.Errorf("player has already built 3 bridges (maximum)")
		}

		// If bridge hex coordinates are provided, validate the bridge placement
		if a.BridgeHex1 != nil && a.BridgeHex2 != nil {
			// Check if bridge already exists
			if gs.Map.HasBridge(*a.BridgeHex1, *a.BridgeHex2) {
				return fmt.Errorf("bridge already exists between these hexes")
			}

			// Validate hex coordinates are on the map
			if gs.Map.GetHex(*a.BridgeHex1) == nil {
				return fmt.Errorf("bridge hex1 is not on the map")
			}
			if gs.Map.GetHex(*a.BridgeHex2) == nil {
				return fmt.Errorf("bridge hex2 is not on the map")
			}

			// Note: Full geometry validation happens in BuildBridge during Execute
		}
		// Note: Bridge coordinates are optional for backward compatibility
		// If not provided, the action just increments the counter
	}

	return nil
}

func (a *PowerAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	powerCost := GetPowerCost(a.ActionType)

	// Move power from Bowl III to Bowl I
	player.Resources.Power.Bowl3 -= powerCost
	player.Resources.Power.Bowl1 += powerCost

	// Mark action as used
	gs.PowerActions.MarkUsed(a.ActionType)

	// Execute the specific action
	switch a.ActionType {
	case PowerActionBridge:
		// Place the bridge on the map if coordinates provided
		if a.BridgeHex1 != nil && a.BridgeHex2 != nil {
			if err := gs.Map.BuildBridge(*a.BridgeHex1, *a.BridgeHex2); err != nil {
				return fmt.Errorf("failed to build bridge: %w", err)
			}

			// Check for town formation after building bridge
			// The bridge might connect buildings into a town
			// Check from both endpoints - CheckForTownFormation handles appending to PendingTownFormations
			gs.CheckForTownFormation(a.PlayerID, *a.BridgeHex1)
			gs.CheckForTownFormation(a.PlayerID, *a.BridgeHex2)
		}

		player.BridgesBuilt++

	case PowerActionPriest:
		// Grant priest with 7-priest limit enforcement
		// If at limit, action still succeeds (power is spent) but no priest is gained
		gs.GainPriests(a.PlayerID, 1)

	case PowerActionWorkers:
		player.Resources.Workers += 2

	case PowerActionCoins:
		player.Resources.Coins += 7

	case PowerActionSpade1, PowerActionSpade2:
		// The spade action gives free spades for a transform
		// The actual transform happens as part of this action
		if a.TargetHex == nil {
			return fmt.Errorf("spade action requires target hex")
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
	}

	return nil
}

// executeTransformWithFreeSpades handles the transform part of spade power actions
func (a *PowerAction) executeTransformWithFreeSpades(gs *GameState, player *Player, freeSpades int) error {
	mapHex := gs.Map.GetHex(*a.TargetHex)

	// Handle skip costs (Fakirs carpet flight / Dwarves tunneling)
	if a.UseSkip {
		if player.Faction.GetType() == models.FactionFakirs {
			// Pay priest for carpet flight
			player.Resources.Priests -= 1
			// Award VP bonus
			player.VictoryPoints += 4
		} else if dwarves, ok := player.Faction.(*factions.Dwarves); ok {
			// Pay workers for tunneling
			workerCost := dwarves.GetTunnelingCost()
			player.Resources.Workers -= workerCost
			// Award VP bonus
			player.VictoryPoints += 4
		}
	}

	// Calculate spades needed
	currentTerrain := mapHex.Terrain
	targetTerrain := player.Faction.GetHomeTerrain()
	distance := gs.Map.GetTerrainDistance(currentTerrain, targetTerrain)

	if distance == 0 {
		return fmt.Errorf("hex is already home terrain")
	}

	// Use free spades first, then pay workers/priests for remaining
	spadesNeeded := distance
	spadesFromFreeAction := freeSpades
	if spadesFromFreeAction > spadesNeeded {
		spadesFromFreeAction = spadesNeeded
	}

	remainingSpades := spadesNeeded - spadesFromFreeAction

	// Pay for remaining spades
	if remainingSpades > 0 {
		// Darklings pay priests (instead of workers)
		if darklings, ok := player.Faction.(*factions.Darklings); ok {
			priestsNeeded := darklings.GetTerraformCostInPriests(remainingSpades)
			if player.Resources.Priests < priestsNeeded {
				return fmt.Errorf("not enough priests: need %d, have %d", priestsNeeded, player.Resources.Priests)
			}
			player.Resources.Priests -= priestsNeeded

			// Award Darklings VP bonus (+2 VP per remaining spade)
			vpBonus := remainingSpades * 2
			player.VictoryPoints += vpBonus
		} else {
			// Other factions pay workers
			workersNeeded := player.Faction.GetTerraformCost(remainingSpades)
			if player.Resources.Workers < workersNeeded {
				return fmt.Errorf("not enough workers: need %d, have %d", workersNeeded, player.Resources.Workers)
			}
			player.Resources.Workers -= workersNeeded
		}
	}

	// Transform the terrain
	gs.Map.TransformTerrain(*a.TargetHex, targetTerrain)

	// Award VP from scoring tile for ALL spades used (both free and paid)
	// Power action spades (ACT5/ACT6) count for scoring, unlike cult reward spades
	totalSpades := spadesFromFreeAction + remainingSpades
	if _, isDarklings := player.Faction.(*factions.Darklings); !isDarklings && totalSpades > 0 {
		// Convert worker/priest cost back to spades
		spadesUsed := player.Faction.GetTerraformSpades(totalSpades)
		for i := 0; i < spadesUsed; i++ {
			gs.AwardActionVP(a.PlayerID, ScoringActionSpades)
		}

		// Award faction-specific spade bonuses (Halflings VP, Alchemists power)
		AwardFactionSpadeBonuses(player, spadesUsed)
	}

	// Build dwelling if requested
	if a.BuildDwelling {
		// Check building limit
		if err := gs.CheckBuildingLimit(a.PlayerID, models.BuildingDwelling); err != nil {
			return err
		}

		// Pay for dwelling
		dwellingCost := player.Faction.GetDwellingCost()
		if err := player.Resources.Spend(dwellingCost); err != nil {
			return fmt.Errorf("failed to pay for dwelling: %w", err)
		}

		// Place dwelling and handle all VP bonuses
		if err := gs.BuildDwelling(a.PlayerID, *a.TargetHex); err != nil {
			return err
		}
	}

	return nil
}
