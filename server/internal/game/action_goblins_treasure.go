package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

type GoblinsTreasureRewardType string

const (
	GoblinsTreasureDwellings     GoblinsTreasureRewardType = "dwellings"
	GoblinsTreasureTradingPosts  GoblinsTreasureRewardType = "trading_posts"
	GoblinsTreasureTemples       GoblinsTreasureRewardType = "temples"
	GoblinsTreasureBigStructures GoblinsTreasureRewardType = "big_structures"
)

type UseGoblinsTreasureAction struct {
	BaseAction
	RewardType GoblinsTreasureRewardType
}

func NewUseGoblinsTreasureAction(playerID string, rewardType GoblinsTreasureRewardType) *UseGoblinsTreasureAction {
	return &UseGoblinsTreasureAction{
		BaseAction: BaseAction{
			Type:     ActionUseGoblinsTreasure,
			PlayerID: playerID,
		},
		RewardType: rewardType,
	}
}

func (a *UseGoblinsTreasureAction) GetType() ActionType {
	return ActionUseGoblinsTreasure
}

func (a *UseGoblinsTreasureAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if player.Faction == nil || player.Faction.GetType() != models.FactionGoblins {
		return fmt.Errorf("only Goblins can use treasure actions")
	}
	if player.GoblinTreasureTokens <= 0 {
		return fmt.Errorf("no goblin treasure tokens available")
	}
	switch a.RewardType {
	case GoblinsTreasureDwellings, GoblinsTreasureTradingPosts, GoblinsTreasureTemples, GoblinsTreasureBigStructures:
		return nil
	default:
		return fmt.Errorf("invalid goblins treasure reward type")
	}
}

func (a *UseGoblinsTreasureAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	player.GoblinTreasureTokens--

	dwellingCount := 0
	tradingPostCount := 0
	templeCount := 0
	hasStronghold := false
	hasSanctuary := false
	for _, mapHex := range gs.Map.Hexes {
		if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != a.PlayerID {
			continue
		}
		switch mapHex.Building.Type {
		case models.BuildingDwelling:
			dwellingCount++
		case models.BuildingTradingHouse:
			tradingPostCount++
		case models.BuildingTemple:
			templeCount++
		case models.BuildingStronghold:
			hasStronghold = true
		case models.BuildingSanctuary:
			hasSanctuary = true
		}
	}

	switch a.RewardType {
	case GoblinsTreasureDwellings:
		player.Resources.GainPower(dwellingCount)
	case GoblinsTreasureTradingPosts:
		player.Resources.Coins += tradingPostCount * 2
	case GoblinsTreasureTemples:
		player.Resources.Workers += templeCount
	case GoblinsTreasureBigStructures:
		steps := 0
		if hasStronghold {
			steps++
		}
		if hasSanctuary {
			steps += 2
		}
		if steps > 0 {
			gs.PendingGoblinsCultSteps = &PendingGoblinsCultSteps{
				PlayerID:       a.PlayerID,
				StepsRemaining: steps,
			}
		}
	}

	gs.NextTurn()
	return nil
}
