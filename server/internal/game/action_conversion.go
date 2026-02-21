package game

import "fmt"

// ConversionType identifies a legal free conversion action.
type ConversionType string

const (
	ConversionPowerToCoin    ConversionType = "power_to_coin"
	ConversionPowerToWorker  ConversionType = "power_to_worker"
	ConversionPowerToPriest  ConversionType = "power_to_priest"
	ConversionPriestToWorker ConversionType = "priest_to_worker"
	ConversionWorkerToCoin   ConversionType = "worker_to_coin"
	ConversionAlchVPToCoin   ConversionType = "alchemists_vp_to_coin"
	ConversionAlchCoinToVP   ConversionType = "alchemists_coin_to_vp"
)

// ConversionAction represents a free conversion during the acting player's turn.
type ConversionAction struct {
	BaseAction
	ConversionType ConversionType
	Amount         int
}

// GetType returns the action type.
func (a *ConversionAction) GetType() ActionType {
	return ActionConversion
}

// Validate checks if the conversion is valid.
func (a *ConversionAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}
	if a.Amount <= 0 {
		return fmt.Errorf("conversion amount must be positive")
	}

	return nil
}

// Execute performs the conversion without ending the turn.
func (a *ConversionAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	switch a.ConversionType {
	case ConversionPowerToCoin:
		return player.Resources.ConvertPowerToCoins(a.Amount)
	case ConversionPowerToWorker:
		return player.Resources.ConvertPowerToWorkers(a.Amount)
	case ConversionPowerToPriest:
		return player.Resources.ConvertPowerToPriests(a.Amount)
	case ConversionPriestToWorker:
		return player.Resources.ConvertPriestToWorker(a.Amount)
	case ConversionWorkerToCoin:
		return player.Resources.ConvertWorkerToCoin(a.Amount)
	case ConversionAlchVPToCoin:
		return gs.AlchemistsConvertVPToCoins(a.PlayerID, a.Amount)
	case ConversionAlchCoinToVP:
		return gs.AlchemistsConvertCoinsToVP(a.PlayerID, a.Amount)
	default:
		return fmt.Errorf("unsupported conversion type: %s", a.ConversionType)
	}
}

// BurnPowerAction represents burning power (2 from bowl II -> 1 to bowl III).
type BurnPowerAction struct {
	BaseAction
	Amount int
}

// GetType returns the action type.
func (a *BurnPowerAction) GetType() ActionType {
	return ActionBurnPower
}

// Validate checks if burning power is valid.
func (a *BurnPowerAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}
	if a.Amount <= 0 {
		return fmt.Errorf("burn amount must be positive")
	}
	if !player.Resources.Power.CanBurn(a.Amount) {
		return fmt.Errorf("cannot burn %d power", a.Amount)
	}

	return nil
}

// Execute burns power without ending the turn.
func (a *BurnPowerAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	return player.Resources.BurnPower(a.Amount)
}
