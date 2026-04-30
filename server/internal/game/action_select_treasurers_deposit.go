package game

import "fmt"

// SelectTreasurersDepositAction resolves a pending Treasurers banking choice.
type SelectTreasurersDepositAction struct {
	BaseAction
	CoinsToTreasury   int
	WorkersToTreasury int
	PriestsToTreasury int
}

func NewSelectTreasurersDepositAction(playerID string, coins, workers, priests int) *SelectTreasurersDepositAction {
	return &SelectTreasurersDepositAction{
		BaseAction:        BaseAction{Type: ActionSelectTreasurersDeposit, PlayerID: playerID},
		CoinsToTreasury:   coins,
		WorkersToTreasury: workers,
		PriestsToTreasury: priests,
	}
}

func (a *SelectTreasurersDepositAction) GetType() ActionType {
	return ActionSelectTreasurersDeposit
}

func (a *SelectTreasurersDepositAction) Validate(gs *GameState) error {
	if gs.PendingTreasurersDeposit == nil {
		return fmt.Errorf("no pending treasurers deposit")
	}
	if gs.PendingTreasurersDeposit.PlayerID != a.PlayerID {
		return fmt.Errorf("treasurers deposit required from player %s", gs.PendingTreasurersDeposit.PlayerID)
	}
	if a.CoinsToTreasury < 0 || a.WorkersToTreasury < 0 || a.PriestsToTreasury < 0 {
		return fmt.Errorf("treasury deposit amounts cannot be negative")
	}
	if a.CoinsToTreasury > gs.PendingTreasurersDeposit.AvailableCoins {
		return fmt.Errorf("cannot treasury %d coins, only %d available", a.CoinsToTreasury, gs.PendingTreasurersDeposit.AvailableCoins)
	}
	if a.WorkersToTreasury > gs.PendingTreasurersDeposit.AvailableWorkers {
		return fmt.Errorf("cannot treasury %d workers, only %d available", a.WorkersToTreasury, gs.PendingTreasurersDeposit.AvailableWorkers)
	}
	if a.PriestsToTreasury > gs.PendingTreasurersDeposit.AvailablePriests {
		return fmt.Errorf("cannot treasury %d priests, only %d available", a.PriestsToTreasury, gs.PendingTreasurersDeposit.AvailablePriests)
	}
	player := gs.GetPlayer(a.PlayerID)
	if !isTreasurers(player) {
		return fmt.Errorf("only Treasurers can use treasury deposits")
	}
	if player.Resources.Coins < a.CoinsToTreasury || player.Resources.Workers < a.WorkersToTreasury || player.Resources.Priests < a.PriestsToTreasury {
		return fmt.Errorf("not enough on-board resources to move to treasury")
	}
	return nil
}

func (a *SelectTreasurersDepositAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	player := gs.GetPlayer(a.PlayerID)
	player.Resources.Coins -= a.CoinsToTreasury
	player.Resources.Workers -= a.WorkersToTreasury
	player.Resources.Priests -= a.PriestsToTreasury
	player.TreasuryCoins += a.CoinsToTreasury
	player.TreasuryWorkers += a.WorkersToTreasury
	player.TreasuryPriests += a.PriestsToTreasury

	gs.advanceTreasurersDepositQueue()
	if !gs.HasPendingIncomeDecisions() && gs.Phase == PhaseIncome {
		gs.StartActionPhase()
	}
	return nil
}
