package game

import (
	"fmt"
	"github.com/lukev/tm_server/internal/models"
)

// PowerActionType represents the different power actions available on the game board
type PowerActionType int

const (
	PowerActionBridge PowerActionType = iota // 3 Power: Build a bridge
	PowerActionPriest                        // 3 Power: Gain 1 priest
	PowerActionWorkers                       // 4 Power: Gain 2 workers
	PowerActionCoins                         // 4 Power: Gain 7 coins
	PowerActionSpade1                        // 4 Power: 1 free spade for transform
	PowerActionSpade2                        // 6 Power: 2 free spades for transform
)

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
	TargetHex      *Hex // Optional: for spade actions
	BuildDwelling  bool // Optional: for spade actions
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
func NewPowerActionWithTransform(playerID string, actionType PowerActionType, targetHex Hex, buildDwelling bool) *PowerAction {
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
		
		// Check adjacency to player's buildings
		if !gs.IsAdjacentToPlayerBuilding(*a.TargetHex, a.PlayerID) {
			return fmt.Errorf("hex is not adjacent to player's buildings")
		}
	}

	// Validate bridge action
	if a.ActionType == PowerActionBridge {
		// Check if player has bridges remaining (max 3)
		if player.BridgesBuilt >= 3 {
			return fmt.Errorf("player has already built 3 bridges (maximum)")
		}
		// TODO: Validate bridge placement location
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
		player.BridgesBuilt++
		// TODO: Actually place the bridge on the map
		
	case PowerActionPriest:
		player.Resources.Priests++
		
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
	
	// Calculate spades needed
	currentTerrain := mapHex.Terrain
	targetTerrain := player.Faction.GetHomeTerrain()
	distance := gs.Map.GetTerrainDistance(currentTerrain, targetTerrain)
	
	if distance == 0 {
		return fmt.Errorf("hex is already home terrain")
	}
	
	// Use free spades first, then pay workers for remaining
	spadesNeeded := distance
	spadesFromFreeAction := freeSpades
	if spadesFromFreeAction > spadesNeeded {
		spadesFromFreeAction = spadesNeeded
	}
	
	remainingSpades := spadesNeeded - spadesFromFreeAction
	
	// Pay for remaining spades with workers
	if remainingSpades > 0 {
		workersNeeded := player.Faction.GetTerraformCost(remainingSpades)
		if player.Resources.Workers < workersNeeded {
			return fmt.Errorf("not enough workers: need %d, have %d", workersNeeded, player.Resources.Workers)
		}
		player.Resources.Workers -= workersNeeded
	}
	
	// Transform the terrain
	gs.Map.TransformTerrain(*a.TargetHex, targetTerrain)
	
	// Build dwelling if requested
	if a.BuildDwelling {
		// Check building limit
		if err := checkBuildingLimit(gs, a.PlayerID, models.BuildingDwelling); err != nil {
			return err
		}
		
		// Pay for dwelling
		dwellingCost := player.Faction.GetDwellingCost()
		if player.Resources.Coins < dwellingCost.Coins {
			return fmt.Errorf("not enough coins for dwelling: need %d, have %d", dwellingCost.Coins, player.Resources.Coins)
		}
		if player.Resources.Workers < dwellingCost.Workers {
			return fmt.Errorf("not enough workers for dwelling: need %d, have %d", dwellingCost.Workers, player.Resources.Workers)
		}
		if player.Resources.Priests < dwellingCost.Priests {
			return fmt.Errorf("not enough priests for dwelling: need %d, have %d", dwellingCost.Priests, player.Resources.Priests)
		}
		
		player.Resources.Coins -= dwellingCost.Coins
		player.Resources.Workers -= dwellingCost.Workers
		player.Resources.Priests -= dwellingCost.Priests
		
		// Place dwelling
		dwelling := &models.Building{
			Type:       models.BuildingDwelling,
			Faction:    player.Faction.GetType(),
			PlayerID:   a.PlayerID,
			PowerValue: 1,
		}
		mapHex.Building = dwelling
		
		// Trigger power leech
		gs.TriggerPowerLeech(*a.TargetHex, a.PlayerID)
	}
	
	return nil
}
