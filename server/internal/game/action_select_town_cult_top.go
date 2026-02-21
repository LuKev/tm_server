package game

import "fmt"

// SelectTownCultTopAction resolves a key-limited town cult-top choice.
type SelectTownCultTopAction struct {
	BaseAction
	Tracks []CultTrack
}

// GetType returns the action type.
func (a *SelectTownCultTopAction) GetType() ActionType {
	return ActionSelectTownCultTop
}

// Validate checks whether the selection is legal.
func (a *SelectTownCultTopAction) Validate(gs *GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	if gs.PendingTownCultTopChoice == nil {
		return fmt.Errorf("no pending town cult-top choice")
	}
	if gs.PendingTownCultTopChoice.PlayerID != a.PlayerID {
		return fmt.Errorf("pending town cult-top choice is for player %s, not %s", gs.PendingTownCultTopChoice.PlayerID, a.PlayerID)
	}

	seen := make(map[CultTrack]bool)
	candidate := make(map[CultTrack]bool)
	for _, track := range gs.PendingTownCultTopChoice.CandidateTracks {
		candidate[track] = true
	}

	for _, track := range a.Tracks {
		if seen[track] {
			return fmt.Errorf("duplicate cult track selection: %v", track)
		}
		seen[track] = true
		if !candidate[track] {
			return fmt.Errorf("track %v is not a valid top-choice candidate", track)
		}
	}

	if len(a.Tracks) != gs.PendingTownCultTopChoice.MaxSelections {
		return fmt.Errorf("must choose exactly %d track(s)", gs.PendingTownCultTopChoice.MaxSelections)
	}

	return nil
}

// Execute applies the deferred town cult bonus using the selected top tracks.
func (a *SelectTownCultTopAction) Execute(gs *GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	pending := gs.PendingTownCultTopChoice
	topChoices := make(map[CultTrack]bool)
	for _, track := range a.Tracks {
		topChoices[track] = true
	}

	gs.CultTracks.ApplyTownCultBonusWithTopChoice(a.PlayerID, pending.AdvanceAmount, player, gs, topChoices)
	gs.PendingTownCultTopChoice = nil
	gs.NextTurn()
	return nil
}
