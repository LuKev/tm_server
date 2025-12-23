package game

import (
	"fmt"
	"strings"
)

// MissingInfoError is returned when the simulator encounters missing information
type MissingInfoError struct {
	Type    string   // "initial_bonus_card" or "pass_bonus_card"
	Players []string // List of players involved
}

func (e *MissingInfoError) Error() string {
	return fmt.Sprintf("missing info: %s for players [%s]", e.Type, strings.Join(e.Players, ","))
}
