package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// BuildWispsStrongholdDwellingAction places the Wisps' immediate free dwelling on a lake.
type BuildWispsStrongholdDwellingAction struct {
	BaseAction
	TargetHex board.Hex
}

func NewBuildWispsStrongholdDwellingAction(playerID string, targetHex board.Hex) *BuildWispsStrongholdDwellingAction {
	return &BuildWispsStrongholdDwellingAction{
		BaseAction: BaseAction{
			Type:     ActionBuildWispsStrongholdDwelling,
			PlayerID: playerID,
		},
		TargetHex: targetHex,
	}
}

func (a *BuildWispsStrongholdDwellingAction) GetType() ActionType {
	return ActionBuildWispsStrongholdDwelling
}

func (a *BuildWispsStrongholdDwellingAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	if player.Faction == nil || player.Faction.GetType() != models.FactionWisps {
		return fmt.Errorf("only Wisps can use this action")
	}
	if gs.PendingWispsStrongholdDwelling == nil || gs.PendingWispsStrongholdDwelling.PlayerID != a.PlayerID {
		return fmt.Errorf("no pending wisps stronghold dwelling for player %s", a.PlayerID)
	}

	mapHex, err := gs.ValidateHex(a.TargetHex)
	if err != nil {
		return err
	}
	if mapHex.Building != nil {
		return fmt.Errorf("hex already has a building")
	}
	if mapHex.Terrain != models.TerrainLake {
		return fmt.Errorf("wisps stronghold dwelling must be built on an unoccupied lakes space")
	}
	return nil
}

func (a *BuildWispsStrongholdDwellingAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	if err := gs.BuildDwelling(a.PlayerID, a.TargetHex); err != nil {
		return err
	}
	gs.PendingWispsStrongholdDwelling = nil
	return nil
}
