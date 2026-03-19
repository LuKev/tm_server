package game

import (
	"fmt"
	"strings"
)

// HasPendingTurnConfirmation reports whether the current game state is waiting
// for the acting player to either confirm or undo their most recent turn.
func (gs *GameState) HasPendingTurnConfirmation() bool {
	if gs == nil {
		return false
	}
	return strings.TrimSpace(gs.PendingTurnConfirmationPlayerID) != "" && gs.PendingTurnConfirmationSnapshot != nil
}

// BeginPendingTurnConfirmation marks the current move as requiring explicit
// confirmation before the next player may act.
func (gs *GameState) BeginPendingTurnConfirmation(playerID string, snapshot *GameState) {
	if gs == nil || snapshot == nil {
		return
	}
	if strings.TrimSpace(playerID) == "" {
		return
	}
	if gs.HasPendingTurnConfirmation() {
		return
	}

	gs.PendingTurnConfirmationPlayerID = strings.TrimSpace(playerID)
	gs.PendingTurnConfirmationSnapshot = snapshot
}

// ClearPendingTurnConfirmation clears the current confirmation window.
func (gs *GameState) ClearPendingTurnConfirmation() {
	if gs == nil {
		return
	}
	gs.PendingTurnConfirmationPlayerID = ""
	gs.PendingTurnConfirmationSnapshot = nil
}

// ConfirmPendingTurn commits the move and closes the confirmation window.
func (gs *GameState) ConfirmPendingTurn(playerID string) error {
	if gs == nil || !gs.HasPendingTurnConfirmation() {
		return fmt.Errorf("no pending turn confirmation")
	}

	pendingPlayerID := strings.TrimSpace(gs.PendingTurnConfirmationPlayerID)
	if strings.TrimSpace(playerID) != pendingPlayerID {
		return fmt.Errorf("turn confirmation pending for player %s", pendingPlayerID)
	}

	gs.PendingFreeActionsPlayerID = ""
	gs.ClearPendingTurnConfirmation()
	return nil
}

// UndoPendingTurn restores the game state to the snapshot captured before the
// last committed turn.
func (gs *GameState) UndoPendingTurn(playerID string) error {
	if gs == nil || !gs.HasPendingTurnConfirmation() {
		return fmt.Errorf("no pending turn confirmation")
	}

	pendingPlayerID := strings.TrimSpace(gs.PendingTurnConfirmationPlayerID)
	if strings.TrimSpace(playerID) != pendingPlayerID {
		return fmt.Errorf("turn confirmation pending for player %s", pendingPlayerID)
	}

	snapshot := gs.PendingTurnConfirmationSnapshot
	if snapshot == nil {
		return fmt.Errorf("no pending turn snapshot")
	}

	restored := snapshot.CloneForUndo()
	*gs = *restored
	gs.ClearPendingTurnConfirmation()
	return nil
}

// ConfirmTurnAction commits the current turn and closes the confirmation window.
type ConfirmTurnAction struct {
	BaseAction
}

// NewConfirmTurnAction creates a turn-confirmation action.
func NewConfirmTurnAction(playerID string) *ConfirmTurnAction {
	return &ConfirmTurnAction{
		BaseAction: BaseAction{
			Type:     ActionConfirmTurn,
			PlayerID: playerID,
		},
	}
}

// GetType returns the action type.
func (a *ConfirmTurnAction) GetType() ActionType {
	return ActionConfirmTurn
}

// Validate checks whether the confirm action is legal.
func (a *ConfirmTurnAction) Validate(gs *GameState) error {
	if !gs.HasPendingTurnConfirmation() {
		return fmt.Errorf("no pending turn confirmation")
	}
	if strings.TrimSpace(gs.PendingTurnConfirmationPlayerID) != strings.TrimSpace(a.PlayerID) {
		return fmt.Errorf("turn confirmation pending for player %s", gs.PendingTurnConfirmationPlayerID)
	}
	return nil
}

// Execute commits the turn.
func (a *ConfirmTurnAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	return gs.ConfirmPendingTurn(a.PlayerID)
}

// UndoTurnAction restores the game state to the pre-turn snapshot.
type UndoTurnAction struct {
	BaseAction
}

// NewUndoTurnAction creates a turn-undo action.
func NewUndoTurnAction(playerID string) *UndoTurnAction {
	return &UndoTurnAction{
		BaseAction: BaseAction{
			Type:     ActionUndoTurn,
			PlayerID: playerID,
		},
	}
}

// GetType returns the action type.
func (a *UndoTurnAction) GetType() ActionType {
	return ActionUndoTurn
}

// Validate checks whether the undo action is legal.
func (a *UndoTurnAction) Validate(gs *GameState) error {
	if !gs.HasPendingTurnConfirmation() {
		return fmt.Errorf("no pending turn confirmation")
	}
	if strings.TrimSpace(gs.PendingTurnConfirmationPlayerID) != strings.TrimSpace(a.PlayerID) {
		return fmt.Errorf("turn confirmation pending for player %s", gs.PendingTurnConfirmationPlayerID)
	}
	return nil
}

// Execute restores the pre-turn snapshot.
func (a *UndoTurnAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	return gs.UndoPendingTurn(a.PlayerID)
}
