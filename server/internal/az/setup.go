package az

import (
	"fmt"
	"math/rand"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// NewBaseGame creates a deterministic base-map, base-rules 1v1 setup with the
// factions supplied by the caller. Faction choice is deliberately excluded from
// the assigned-faction competition profile.
func NewBaseGame(seed int64, first, second models.FactionType) (*GamePosition, error) {
	return newBaseGame(seed, first, second, "p0", "p1")
}

// NewBaseGameWithPlayerIDs is the deterministic setup seam used by protocol
// and representation tests to prove that internal IDs do not leak into
// player-relative records.
func NewBaseGameWithPlayerIDs(seed int64, first, second models.FactionType, firstID, secondID string) (*GamePosition, error) {
	return newBaseGame(seed, first, second, firstID, secondID)
}

func newBaseGame(seed int64, first, second models.FactionType, firstID, secondID string) (*GamePosition, error) {
	if !isBaseFaction(first) || !isBaseFaction(second) {
		return nil, fmt.Errorf("v0 requires base factions, got %s and %s", first, second)
	}
	gs := game.NewGameState()
	// Pin the competition contract rather than inheriting future lobby defaults.
	gs.SetupMode = game.SetupModeSnellman
	gs.TurnOrderPolicy = game.TurnOrderPolicyPassOrder
	gs.FireIceFinalScoringSetting = game.FireIceFinalScoringOff
	gs.FireIceFinalScoringTile = game.FireIceFinalScoringTileNone
	gs.TownTiles = game.NewBaseTownTileState()
	rng := rand.New(rand.NewSource(seed))

	if err := gs.ScoringTiles.InitializeBaseGameWithRand(rng); err != nil {
		return nil, err
	}
	if cards := gs.BonusCards.SelectRandomBaseBonusCardsWithRand(2, rng); len(cards) != 5 {
		return nil, fmt.Errorf("expected five bonus cards, got %d", len(cards))
	}

	if err := gs.AddPlayer(firstID, factions.NewFaction(first)); err != nil {
		return nil, err
	}
	if err := gs.AddPlayer(secondID, factions.NewFaction(second)); err != nil {
		return nil, err
	}
	gs.TurnOrder = []string{firstID, secondID}
	gs.CurrentPlayerIndex = 0
	gs.InitializeSetupSequence()
	return NewPosition(gs)
}

func isBaseFaction(faction models.FactionType) bool {
	for _, candidate := range baseFactions {
		if candidate == faction {
			return true
		}
	}
	return false
}
