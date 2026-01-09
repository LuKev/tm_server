package game

import (
	"fmt"
	"strings"
)

// MissingInfoError is returned when the simulator encounters missing information
type MissingInfoError struct {
	Type    string   // "initial_bonus_card" or "pass_bonus_card"
	Players []string // List of players involved (for the first/current issue)
	Round   int      // Round number (for pass_bonus_card)

	// AllMissingPasses contains ALL missing pass bonus cards: Round -> []PlayerID
	// Used when Type is "pass_bonus_card" to allow collecting all at once
	AllMissingPasses map[int][]string
}

func (e *MissingInfoError) Error() string {
	return fmt.Sprintf("missing info: %s for players [%s]", e.Type, strings.Join(e.Players, ","))
}
