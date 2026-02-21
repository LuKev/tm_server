package game

import "fmt"

// DiscardPendingSpadeAction discards one pending free spade from a follow-up chain.
type DiscardPendingSpadeAction struct {
	BaseAction
	Count int
}

// NewDiscardPendingSpadeAction creates a pending-spade discard action.
func NewDiscardPendingSpadeAction(playerID string, count int) *DiscardPendingSpadeAction {
	if count <= 0 {
		count = 1
	}
	return &DiscardPendingSpadeAction{
		BaseAction: BaseAction{
			Type:     ActionDiscardPendingSpade,
			PlayerID: playerID,
		},
		Count: count,
	}
}

// Validate checks whether discarding pending spades is legal.
func (a *DiscardPendingSpadeAction) Validate(gs *GameState) error {
	if a.Count <= 0 {
		return fmt.Errorf("discard count must be at least 1")
	}
	pending := 0
	if gs.PendingSpades != nil {
		pending = gs.PendingSpades[a.PlayerID]
	}
	if pending <= 0 {
		return fmt.Errorf("player has no pending spades to discard")
	}
	if a.Count > pending {
		return fmt.Errorf("cannot discard %d pending spades; only %d available", a.Count, pending)
	}
	return nil
}

// Execute discards pending spades and advances turn when the follow-up is resolved.
func (a *DiscardPendingSpadeAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	gs.PendingSpades[a.PlayerID] -= a.Count
	if gs.PendingSpades[a.PlayerID] <= 0 {
		delete(gs.PendingSpades, a.PlayerID)
		delete(gs.PendingSpadeBuildAllowed, a.PlayerID)
	}

	gs.NextTurn()
	return nil
}
