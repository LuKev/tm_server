package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// UseDarklingsPriestOrdinationAction represents converting workers to priests after building stronghold
// The player chooses how many workers (0-3) to convert
type UseDarklingsPriestOrdinationAction struct {
	BaseAction
	WorkersToConvert int // Number of workers to convert (0-3)
}

func (a *UseDarklingsPriestOrdinationAction) GetType() ActionType {
	return ActionUseDarklingsPriestOrdination
}

func (a *UseDarklingsPriestOrdinationAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Check if player has passed
	if player.HasPassed {
		return fmt.Errorf("player has already passed")
	}

	// Check if there's a pending priest ordination
	if gs.PendingDarklingsPriestOrdination == nil {
		return fmt.Errorf("no pending Darklings priest ordination")
	}

	// Check if this is the correct player
	if gs.PendingDarklingsPriestOrdination.PlayerID != a.PlayerID {
		return fmt.Errorf("pending priest ordination is for player %s, not %s",
			gs.PendingDarklingsPriestOrdination.PlayerID, a.PlayerID)
	}

	// Check if player is Darklings
	if player.Faction.GetType() != models.FactionDarklings {
		return fmt.Errorf("only Darklings can use priest ordination")
	}

	// Check worker count is valid (0-3)
	if a.WorkersToConvert < 0 || a.WorkersToConvert > 3 {
		return fmt.Errorf("can only convert 0-3 workers, got %d", a.WorkersToConvert)
	}

	// Check if player has enough workers
	if player.Resources.Workers < a.WorkersToConvert {
		return fmt.Errorf("not enough workers (have %d, need %d)", 
			player.Resources.Workers, a.WorkersToConvert)
	}

	return nil
}

func (a *UseDarklingsPriestOrdinationAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	darklings, ok := player.Faction.(*factions.Darklings)
	if !ok {
		return fmt.Errorf("player is not Darklings")
	}

	// Use the faction's priest ordination method
	priestsGained, err := darklings.UsePriestOrdination(a.WorkersToConvert)
	if err != nil {
		return fmt.Errorf("failed to use priest ordination: %w", err)
	}

	// Pay the workers
	player.Resources.Workers -= a.WorkersToConvert

	// Gain priests (respects 7-priest limit)
	actualPriestsGained := gs.GainPriests(a.PlayerID, priestsGained)

	// If we couldn't gain all priests due to 7-priest limit, that's okay
	// The workers are still spent (this is the official rule)
	_ = actualPriestsGained

	// Clear pending state
	gs.PendingDarklingsPriestOrdination = nil

	return nil
}
