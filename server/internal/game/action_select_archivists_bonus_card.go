package game

import "fmt"

// SelectArchivistsBonusCardAction resolves the Archivists stronghold pass follow-up.
type SelectArchivistsBonusCardAction struct {
	BaseAction
	BonusCard BonusCardType
}

func NewSelectArchivistsBonusCardAction(playerID string, bonusCard BonusCardType) *SelectArchivistsBonusCardAction {
	return &SelectArchivistsBonusCardAction{
		BaseAction: BaseAction{Type: ActionSelectArchivistsBonusCard, PlayerID: playerID},
		BonusCard:  bonusCard,
	}
}

func (a *SelectArchivistsBonusCardAction) GetType() ActionType {
	return ActionSelectArchivistsBonusCard
}

func (a *SelectArchivistsBonusCardAction) Validate(gs *GameState) error {
	if gs.PendingArchivistsBonusSelection == nil {
		return fmt.Errorf("no pending archivists bonus card selection")
	}
	if gs.PendingArchivistsBonusSelection.PlayerID != a.PlayerID {
		return fmt.Errorf("archivists bonus card selection required from player %s", gs.PendingArchivistsBonusSelection.PlayerID)
	}
	if !gs.BonusCards.IsAvailable(a.BonusCard) {
		return fmt.Errorf("bonus card %v is not available", a.BonusCard)
	}
	for _, returnedCard := range gs.PendingArchivistsBonusSelection.ReturnedCards {
		if returnedCard == a.BonusCard {
			return fmt.Errorf("archivists cannot take a bonus card they just returned")
		}
	}
	return nil
}

func (a *SelectArchivistsBonusCardAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	pending := gs.PendingArchivistsBonusSelection
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	coins, err := gs.BonusCards.TakeAdditionalBonusCard(a.PlayerID, a.BonusCard)
	if err != nil {
		return fmt.Errorf("failed to take archivists second bonus card: %w", err)
	}
	player.Resources.Coins += coins
	player.Resources.Power.GainPower(coins * 2)
	gs.PendingArchivistsBonusSelection = nil
	gs.ApplyAutoConvertOnPass(a.PlayerID)
	if err := advanceAfterCompletedPass(gs); err != nil {
		return err
	}
	gs.PendingFreeActionsPlayerID = a.PlayerID
	if pending != nil && pending.UndoSnapshot != nil {
		gs.BeginPendingTurnConfirmation(a.PlayerID, pending.UndoSnapshot)
	}
	return nil
}
