package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/models"
)

// ConversionType identifies a legal free conversion action.
type ConversionType string

const (
	ConversionPowerToCoin    ConversionType = "power_to_coin"
	ConversionPowerToWorker  ConversionType = "power_to_worker"
	ConversionPowerToPriest  ConversionType = "power_to_priest"
	ConversionPriestToWorker ConversionType = "priest_to_worker"
	ConversionWorkerToPriest ConversionType = "worker_to_priest"
	ConversionWorkerToCoin   ConversionType = "worker_to_coin"
	ConversionCoinToPower    ConversionType = "coin_to_power"
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
	if a.ConversionType == ConversionCoinToPower && player.Faction.GetType() != models.FactionTheEnlightened {
		return fmt.Errorf("coin to power conversion is only available to The Enlightened")
	}
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}
	if a.Amount <= 0 {
		return fmt.Errorf("conversion amount must be positive")
	}
	if a.ConversionType == ConversionWorkerToPriest {
		return fmt.Errorf("worker to priest conversion is only allowed through Darklings priest ordination")
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
		if player.Faction.GetType() == models.FactionTheEnlightened && player.HasStrongholdAbility {
			if err := player.Resources.Power.SpendPower(a.Amount); err != nil {
				return err
			}
			player.Resources.Coins += a.Amount * 2
			return nil
		}
		return player.Resources.ConvertPowerToCoins(a.Amount)
	case ConversionPowerToWorker:
		if player.Faction.GetType() == models.FactionTheEnlightened && player.HasStrongholdAbility {
			powerNeeded := a.Amount * 3
			if err := player.Resources.Power.SpendPower(powerNeeded); err != nil {
				return err
			}
			player.Resources.Workers += a.Amount * 2
			return nil
		}
		return player.Resources.ConvertPowerToWorkers(a.Amount)
	case ConversionPowerToPriest:
		if player.Faction.GetType() == models.FactionTheEnlightened && player.HasStrongholdAbility {
			powerNeeded := a.Amount * 5
			if err := player.Resources.Power.SpendPower(powerNeeded); err != nil {
				return err
			}
			player.Resources.Priests += a.Amount * 2
			return nil
		}
		return player.Resources.ConvertPowerToPriests(a.Amount)
	case ConversionPriestToWorker:
		if player.Faction.GetType() == models.FactionDynionGeifr {
			if player.Resources.Priests < a.Amount {
				return fmt.Errorf("need %d priests, only have %d", a.Amount, player.Resources.Priests)
			}
			player.Resources.Priests -= a.Amount
			player.Resources.Workers += 2 * a.Amount
			player.Resources.Coins += 2 * a.Amount
			return nil
		}
		return player.Resources.ConvertPriestToWorker(a.Amount)
	case ConversionWorkerToCoin:
		return player.Resources.ConvertWorkerToCoin(a.Amount)
	case ConversionCoinToPower:
		return player.Resources.ConvertCoinToPowerTokens(a.Amount)
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
	if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
		if !player.Resources.Power.CanBurnChildren(a.Amount) {
			return fmt.Errorf("cannot burn %d power", a.Amount)
		}
		return nil
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

	if player.Faction != nil && player.Faction.GetType() == models.FactionChildrenOfTheWyrm {
		return player.Resources.Power.BurnPowerChildren(a.Amount)
	}
	return player.Resources.BurnPower(a.Amount)
}
