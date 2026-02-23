package game

import (
	"errors"
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// SelectFactionAction represents selecting a faction
type SelectFactionAction struct {
	PlayerID    string
	FactionType models.FactionType
}

// GetPlayerID returns the player ID
func (a *SelectFactionAction) GetPlayerID() string {
	return a.PlayerID
}

// GetType returns the action type
func (a *SelectFactionAction) GetType() ActionType {
	return ActionSelectFaction
}

// Validate checks if the action is valid
func (a *SelectFactionAction) Validate(gs *GameState) error {
	if gs.Phase != PhaseFactionSelection {
		return errors.New("not in faction selection phase")
	}
	if gs.SetupMode != SetupModeSnellman {
		return fmt.Errorf("select_faction is disabled for setup mode %s", gs.SetupMode)
	}

	// Check if it's player's turn
	if gs.TurnOrder[gs.CurrentPlayerIndex] != a.PlayerID {
		return errors.New("not your turn")
	}

	// Check if faction is valid
	if !isValidFaction(a.FactionType) {
		return fmt.Errorf("invalid faction type: %s", a.FactionType)
	}

	// Check if faction is already taken
	for _, p := range gs.Players {
		if p.Faction != nil && p.Faction.GetType() == a.FactionType {
			return fmt.Errorf("faction %s is already taken", a.FactionType)
		}
	}

	return nil
}

// Execute performs the action
func (a *SelectFactionAction) Execute(gs *GameState) error {
	if err := assignFactionToPlayer(gs, a.PlayerID, a.FactionType, 20); err != nil {
		return err
	}

	// Move to next player or start setup phase
	if allPlayersHaveFactions(gs) {
		gs.InitializeSetupSequence()
	} else {
		gs.CurrentPlayerIndex++
		if gs.CurrentPlayerIndex >= len(gs.TurnOrder) {
			gs.CurrentPlayerIndex = 0 // Should not happen if logic is correct
		}
	}

	return nil
}

func isValidFaction(f models.FactionType) bool {
	switch f {
	case models.FactionNomads, models.FactionWitches, models.FactionHalflings, models.FactionMermaids, models.FactionGiants, models.FactionChaosMagicians, models.FactionEngineers, models.FactionDarklings, models.FactionAlchemists, models.FactionCultists, models.FactionAuren, models.FactionSwarmlings, models.FactionDwarves, models.FactionFakirs:
		return true
	}
	return false
}

func allPlayersHaveFactions(gs *GameState) bool {
	for _, p := range gs.Players {
		if p.Faction == nil {
			return false
		}
	}
	return true
}

func assignFactionToPlayer(gs *GameState, playerID string, factionType models.FactionType, startingVP int) error {
	faction := factions.NewFaction(factionType)
	player := gs.Players[playerID]
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	player.Faction = faction
	player.Resources = NewResourcePool(faction.GetStartingResources())
	player.VictoryPoints = startingVP

	startingCult := faction.GetStartingCultPositions()
	if player.CultPositions == nil {
		player.CultPositions = make(map[CultTrack]int, 4)
	}
	player.CultPositions[CultFire] = startingCult.Fire
	player.CultPositions[CultWater] = startingCult.Water
	player.CultPositions[CultEarth] = startingCult.Earth
	player.CultPositions[CultAir] = startingCult.Air
	if gs.CultTracks != nil {
		if _, ok := gs.CultTracks.PlayerPositions[playerID]; !ok {
			gs.CultTracks.InitializePlayer(playerID)
		}
		gs.CultTracks.PlayerPositions[playerID][CultFire] = startingCult.Fire
		gs.CultTracks.PlayerPositions[playerID][CultWater] = startingCult.Water
		gs.CultTracks.PlayerPositions[playerID][CultEarth] = startingCult.Earth
		gs.CultTracks.PlayerPositions[playerID][CultAir] = startingCult.Air
	}

	if shippingFaction, ok := faction.(interface{ GetShippingLevel() int }); ok {
		player.ShippingLevel = shippingFaction.GetShippingLevel()
	}

	return nil
}
