package game

// IncomePreview is a JSON-friendly next-round resource summary.
type IncomePreview struct {
	Coins   int `json:"coins"`
	Workers int `json:"workers"`
	Priests int `json:"priests"`
	Power   int `json:"power"`
	Spades  int `json:"spades"`
}

// GetNextRoundIncomePreview returns an estimate of resources gained at the next round start
// for rounds 1-5. This includes normal income and round-tile cult rewards.
func (gs *GameState) GetNextRoundIncomePreview(playerID string) (IncomePreview, bool) {
	var preview IncomePreview
	if gs == nil || gs.Round < 1 || gs.Round > 5 {
		return preview, false
	}

	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil {
		return preview, false
	}

	base := calculatePlayerIncome(gs, player)
	preview.Coins += base.Coins
	preview.Workers += base.Workers
	preview.Priests += base.Priests
	preview.Power += base.Power

	cultReward := gs.getRoundCultRewardPreview(playerID, gs.Round)
	preview.Coins += cultReward.Coins
	preview.Workers += cultReward.Workers
	preview.Priests += cultReward.Priests
	preview.Power += cultReward.Power
	preview.Spades += cultReward.Spades

	return preview, true
}

func (gs *GameState) getRoundCultRewardPreview(playerID string, round int) IncomePreview {
	var preview IncomePreview
	if gs == nil || gs.ScoringTiles == nil || gs.CultTracks == nil {
		return preview
	}

	tile := gs.ScoringTiles.GetTileForRound(round)
	if tile == nil {
		return preview
	}

	if tile.Type == ScoringTemplePriest {
		if priestsSent := gs.ScoringTiles.PriestsSent[playerID]; priestsSent > 0 {
			preview.Coins += priestsSent * tile.CultRewardAmount
		}
		return preview
	}

	if tile.CultThreshold <= 0 {
		return preview
	}

	position := gs.CultTracks.GetPosition(playerID, tile.CultTrack)
	rewardCount := position / tile.CultThreshold
	if rewardCount <= 0 {
		return preview
	}

	totalReward := rewardCount * tile.CultRewardAmount
	switch tile.CultRewardType {
	case CultRewardPriest:
		preview.Priests += totalReward
	case CultRewardPower:
		preview.Power += totalReward
	case CultRewardSpade:
		preview.Spades += totalReward
	case CultRewardWorker:
		preview.Workers += totalReward
	case CultRewardCoin:
		preview.Coins += totalReward
	}
	return preview
}
