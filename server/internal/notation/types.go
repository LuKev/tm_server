package notation

import (
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

// LogItem represents an item in the game log, which can be a player action or metadata.
type LogItem interface {
	isLogItem()
}

// ActionItem wraps a standard game action.
type ActionItem struct {
	Action game.Action
}

func (i ActionItem) isLogItem() {}

// GameSettingsItem represents the game configuration header.
type GameSettingsItem struct {
	Settings map[string]string
}

func (i GameSettingsItem) isLogItem() {}

// RoundStartItem marks the start of a new round.
type RoundStartItem struct {
	Round     int
	TurnOrder []string
}

// LogAcceptLeechAction is a log-only representation of accepting leech
type LogAcceptLeechAction struct {
	PlayerID    string
	PowerAmount int
	VPCost      int
}

func (a *LogAcceptLeechAction) GetType() game.ActionType          { return game.ActionAcceptPowerLeech }
func (a *LogAcceptLeechAction) GetPlayerID() string               { return a.PlayerID }
func (a *LogAcceptLeechAction) Validate(gs *game.GameState) error { return nil }
func (a *LogAcceptLeechAction) Execute(gs *game.GameState) error  { return nil }

// LogPowerAction is a log-only representation of a power action
type LogPowerAction struct {
	PlayerID   string
	ActionCode string // e.g. "ACT1", "ACT6"
}

func (a *LogPowerAction) GetType() game.ActionType          { return game.ActionPowerAction }
func (a *LogPowerAction) GetPlayerID() string               { return a.PlayerID }
func (a *LogPowerAction) Validate(gs *game.GameState) error { return nil }
func (a *LogPowerAction) Execute(gs *game.GameState) error  { return nil }

// LogBurnAction is a log-only representation of burning power
type LogBurnAction struct {
	PlayerID string
	Amount   int
}

func (a *LogBurnAction) GetType() game.ActionType          { return game.ActionSpecialAction } // Placeholder
func (a *LogBurnAction) GetPlayerID() string               { return a.PlayerID }
func (a *LogBurnAction) Validate(gs *game.GameState) error { return nil }
func (a *LogBurnAction) Execute(gs *game.GameState) error  { return nil }

// LogFavorTileAction is a log-only representation of taking a favor tile
type LogFavorTileAction struct {
	PlayerID string
	Tile     string // e.g. "FAV-F1"
}

func (a *LogFavorTileAction) GetType() game.ActionType          { return game.ActionSelectFavorTile }
func (a *LogFavorTileAction) GetPlayerID() string               { return a.PlayerID }
func (a *LogFavorTileAction) Validate(gs *game.GameState) error { return nil }
func (a *LogFavorTileAction) Execute(gs *game.GameState) error  { return nil }

// LogSpecialAction is a log-only representation of special faction actions (e.g. Witches Ride)
type LogSpecialAction struct {
	PlayerID   string
	ActionCode string // e.g. "ACT-SH-D-C4"
}

func (a *LogSpecialAction) GetType() game.ActionType          { return game.ActionSpecialAction }
func (a *LogSpecialAction) GetPlayerID() string               { return a.PlayerID }
func (a *LogSpecialAction) Validate(gs *game.GameState) error { return nil }
func (a *LogSpecialAction) Execute(gs *game.GameState) error  { return nil }

// LogConversionAction is a log-only representation of a conversion action
type LogConversionAction struct {
	PlayerID string
	Cost     map[models.ResourceType]int
	Reward   map[models.ResourceType]int
}

func (a *LogConversionAction) GetType() game.ActionType          { return game.ActionSpecialAction } // Placeholder
func (a *LogConversionAction) GetPlayerID() string               { return a.PlayerID }
func (a *LogConversionAction) Validate(gs *game.GameState) error { return nil }
func (a *LogConversionAction) Execute(gs *game.GameState) error  { return nil }

// LogCompoundAction is a log-only representation of multiple actions chained together
type LogCompoundAction struct {
	Actions []game.Action
}

func (a *LogCompoundAction) GetType() game.ActionType { return game.ActionSpecialAction } // Placeholder
func (a *LogCompoundAction) GetPlayerID() string {
	if len(a.Actions) > 0 {
		return a.Actions[0].GetPlayerID()
	}
	return ""
}
func (a *LogCompoundAction) Validate(gs *game.GameState) error { return nil }
func (a *LogCompoundAction) Execute(gs *game.GameState) error  { return nil }

// LogTownAction is a log-only representation of founding a town
type LogTownAction struct {
	PlayerID string
	VP       int
}

func (a *LogTownAction) GetType() game.ActionType          { return game.ActionSpecialAction } // Placeholder
func (a *LogTownAction) GetPlayerID() string               { return a.PlayerID }
func (a *LogTownAction) Validate(gs *game.GameState) error { return nil }
func (a *LogTownAction) Execute(gs *game.GameState) error  { return nil }

func (i RoundStartItem) isLogItem() {}
