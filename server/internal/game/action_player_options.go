package game

import "fmt"

// SetPlayerOptionsAction updates per-player UI/automation options.
// This action is intentionally non-turn-bound and may be sent whenever legal seat authority exists.
type SetPlayerOptionsAction struct {
	BaseAction
	AutoLeechMode     *LeechAutoMode
	AutoConvertOnPass *bool
	ConfirmActions    *bool
	ShowIncomePreview *bool
}

func NewSetPlayerOptionsAction(
	playerID string,
	autoLeechMode *LeechAutoMode,
	autoConvertOnPass *bool,
	confirmActions *bool,
	showIncomePreview *bool,
) *SetPlayerOptionsAction {
	return &SetPlayerOptionsAction{
		BaseAction: BaseAction{
			Type:     ActionSetPlayerOptions,
			PlayerID: playerID,
		},
		AutoLeechMode:     autoLeechMode,
		AutoConvertOnPass: autoConvertOnPass,
		ConfirmActions:    confirmActions,
		ShowIncomePreview: showIncomePreview,
	}
}

func (a *SetPlayerOptionsAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if a.AutoLeechMode != nil && !a.AutoLeechMode.IsValid() {
		return fmt.Errorf("invalid auto leech mode: %s", string(*a.AutoLeechMode))
	}
	return nil
}

func (a *SetPlayerOptionsAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	if a.AutoLeechMode != nil {
		player.Options.AutoLeechMode = *a.AutoLeechMode
	}
	if a.AutoConvertOnPass != nil {
		player.Options.AutoConvertOnPass = *a.AutoConvertOnPass
	}
	if a.ConfirmActions != nil {
		player.Options.ConfirmActions = *a.ConfirmActions
	}
	if a.ShowIncomePreview != nil {
		player.Options.ShowIncomePreview = *a.ShowIncomePreview
	}
	return nil
}
