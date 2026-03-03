package game

import "fmt"

// SetupBonusCardAction represents selecting a setup bonus card before round 1 starts.
type SetupBonusCardAction struct {
	BaseAction
	BonusCard BonusCardType
}

// GetType returns the action type.
func (a *SetupBonusCardAction) GetType() ActionType {
	return ActionSetupBonusCard
}

// Validate checks if setup bonus card selection is valid.
func (a *SetupBonusCardAction) Validate(gs *GameState) error {
	if gs.Phase != PhaseSetup {
		return fmt.Errorf("setup bonus cards can only be selected during setup phase")
	}
	if gs.SetupSubphase != SetupSubphaseBonusCards {
		return fmt.Errorf("not in setup bonus card selection subphase")
	}

	expectedPlayer := gs.currentSetupBonusPlayerID()
	if expectedPlayer == "" {
		return fmt.Errorf("no setup bonus card selection expected")
	}
	if expectedPlayer != a.PlayerID {
		return fmt.Errorf("not your setup bonus selection turn")
	}

	if !gs.BonusCards.IsAvailable(a.BonusCard) {
		return fmt.Errorf("bonus card %v is not available", a.BonusCard)
	}

	return nil
}

// Execute performs setup bonus card selection.
func (a *SetupBonusCardAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	if _, err := gs.BonusCards.TakeBonusCard(a.PlayerID, a.BonusCard); err != nil {
		return fmt.Errorf("failed to take setup bonus card: %w", err)
	}
	// Bonus card coins are awarded during income/grant rounds, not during setup selection.

	gs.AdvanceSetupAfterBonusSelection()
	return nil
}
