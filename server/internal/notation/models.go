package notation

import (
	"fmt"
)

// ActionType represents the type of action in the concise notation
type ActionType string

const (
	ActionBuild               ActionType = "BUILD"         // C4
	ActionUpgrade             ActionType = "UPGRADE"       // TP-C4
	ActionDigBuild            ActionType = "DIG"           // D-C4
	ActionTransform           ActionType = "TRANSFORM"     // D-C4-T (Transform only)
	ActionBonusSpade          ActionType = "BONUS_SPADE"   // ACTS-C4
	ActionPower               ActionType = "POWER"         // ACT4
	ActionSpecial             ActionType = "SPECIAL"       // ACTW
	ActionSendPriest          ActionType = "PRIEST"        // ->F
	ActionConvert             ActionType = "CONVERT"       // C3PW:1W
	ActionBurn                ActionType = "BURN"          // B3
	ActionPass                ActionType = "PASS"          // Pass-BON1
	ActionAdvance             ActionType = "ADVANCE"       // +SHIP
	ActionLeech               ActionType = "LEECH"         // L or DL
	ActionCultReaction        ActionType = "CULT_REACTION" // CULT-F
	ActionDarklingsOrdination ActionType = "ORDINATION"    // ORD-3
)

// GameAction represents a single action in the concise notation
type GameAction struct {
	Faction  string
	Type     ActionType
	Params   map[string]string
	Original string // The original string representation
}

func (a *GameAction) String() string {
	if a.Original != "" {
		return a.Original
	}

	switch a.Type {
	case ActionBuild:
		return a.Params["coord"]
	case ActionUpgrade:
		return fmt.Sprintf("%s-%s", a.Params["building"], a.Params["coord"])
	case ActionDigBuild:
		s := fmt.Sprintf("%s-%s", a.Params["spades"], a.Params["coord"])
		if a.Params["transform_only"] == "true" {
			s += "-T"
		}
		return s
	case ActionTransform:
		return fmt.Sprintf("%s-%s-T", a.Params["spades"], a.Params["coord"])
	case ActionPower:
		s := a.Params["code"]
		if args, ok := a.Params["args"]; ok && args != "" {
			s += "-" + args
		}
		return s
	case ActionPass:
		return fmt.Sprintf("Pass-%s", a.Params["bonus"])
	case ActionLeech:
		if a.Params["decline"] == "true" {
			return "DL"
		}
		return "L"
	case ActionCultReaction:
		return fmt.Sprintf("CULT-%s", a.Params["track"])
	case ActionSendPriest:
		return fmt.Sprintf("->%s", a.Params["target"])
	case ActionBurn:
		return fmt.Sprintf("B%s", a.Params["amount"])
	case ActionConvert:
		return fmt.Sprintf("C%s:%s", a.Params["in"], a.Params["out"])
	case ActionSpecial:
		return a.Params["code"]
	case ActionBonusSpade:
		// Usually handled as ACTS-C4 or similar
		// If code is present, use it
		if code, ok := a.Params["code"]; ok {
			return code
		}
		return "ACTS"
	case ActionDarklingsOrdination:
		return fmt.Sprintf("ORD-%s", a.Params["amount"])
	case ActionAdvance:
		return fmt.Sprintf("+%s", a.Params["track"])
	}

	return fmt.Sprintf("%s: %s", a.Faction, a.Type)
}

// Log represents the full game log
type Log struct {
	MapName      string
	ScoringTiles []string
	BonusCards   []string
	Options      []string
	Rounds       []*RoundLog
}

// RoundLog represents a single round of actions
type RoundLog struct {
	RoundNumber int
	TurnOrder   []string
	Actions     []*GameAction
}

// Helper to create a new action
func NewGameAction(faction string, actionType ActionType, params map[string]string) *GameAction {
	if params == nil {
		params = make(map[string]string)
	}
	return &GameAction{
		Faction: faction,
		Type:    actionType,
		Params:  params,
	}
}
