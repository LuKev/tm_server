package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// SelectRiverwalkersPriestChoiceAction resolves the Riverwalkers choice made
// immediately when a priest would be gained.
type SelectRiverwalkersPriestChoiceAction struct {
	BaseAction
	TakePriest bool
	Terrain    models.TerrainType
}

func NewSelectRiverwalkersPriestChoiceAction(playerID string, takePriest bool, terrain models.TerrainType) *SelectRiverwalkersPriestChoiceAction {
	return &SelectRiverwalkersPriestChoiceAction{
		BaseAction: BaseAction{Type: ActionSelectRiverwalkersPriestChoice, PlayerID: playerID},
		TakePriest: takePriest,
		Terrain:    terrain,
	}
}

func (a *SelectRiverwalkersPriestChoiceAction) GetType() ActionType {
	return ActionSelectRiverwalkersPriestChoice
}

func (a *SelectRiverwalkersPriestChoiceAction) Validate(gs *GameState) error {
	if gs.PendingRiverwalkersPriestChoice == nil {
		return fmt.Errorf("no pending riverwalkers priest choice")
	}
	if gs.PendingRiverwalkersPriestChoice.PlayerID != a.PlayerID {
		return fmt.Errorf("riverwalkers priest choice required from player %s", gs.PendingRiverwalkersPriestChoice.PlayerID)
	}
	player := gs.GetPlayer(a.PlayerID)
	if !isRiverwalkers(player) {
		return fmt.Errorf("riverwalkers priest choice is only available to Riverwalkers")
	}
	if a.TakePriest {
		if gs.RemainingPriestCapacity(a.PlayerID) < 1 {
			return fmt.Errorf("not enough priest capacity")
		}
		return nil
	}
	if !isStandardLandTerrain(a.Terrain) {
		return fmt.Errorf("riverwalkers can only unlock standard land terrain")
	}
	if player.UnlockedTerrains != nil && player.UnlockedTerrains[a.Terrain] {
		return fmt.Errorf("riverwalkers have already unlocked %s terrain", a.Terrain)
	}
	cost := gs.riverwalkersUnlockCost(player, a.Terrain)
	if player.Resources.Coins < cost {
		return fmt.Errorf("not enough coins to unlock terrain: need %d, have %d", cost, player.Resources.Coins)
	}
	return nil
}

func (a *SelectRiverwalkersPriestChoiceAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	player := gs.GetPlayer(a.PlayerID)
	if a.TakePriest {
		if gained := gs.gainPriestsToResources(a.PlayerID, 1); gained != 1 {
			return fmt.Errorf("failed to take priest into resources")
		}
		gs.consumeRiverwalkersPriestChoice(player)
		return nil
	}

	cost := gs.riverwalkersUnlockCost(player, a.Terrain)
	player.Resources.Coins -= cost
	if player.UnlockedTerrains == nil {
		player.UnlockedTerrains = make(map[models.TerrainType]bool)
	}
	player.UnlockedTerrains[a.Terrain] = true
	gs.consumeRiverwalkersPriestChoice(player)
	return nil
}
