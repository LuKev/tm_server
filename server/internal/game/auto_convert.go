package game

// ApplyAutoConvertOnPass performs quality-of-life conversions that are guaranteed
// not to reduce next-round effective income value when a player passes (rounds 1-5).
func (gs *GameState) ApplyAutoConvertOnPass(playerID string) {
	if gs == nil || gs.Round < 1 || gs.Round > 5 {
		return
	}
	player := gs.GetPlayer(playerID)
	if player == nil || !player.Options.AutoConvertOnPass || player.Resources == nil || player.Resources.Power == nil {
		return
	}
	preview, ok := gs.GetNextRoundIncomePreview(playerID)
	if !ok {
		return
	}

	// 1) Priest overflow protection:
	// If next-round priest income would overflow the 7-priest cap, convert the overflow
	// from priests in hand to workers now.
	priestsOnCult := 0
	if gs.CultTracks != nil {
		priestsOnCult = gs.CultTracks.GetTotalPriestsOnCultTracks(playerID)
	}
	maxPriestsGainable := 7 - (player.Resources.Priests + priestsOnCult)
	if maxPriestsGainable < 0 {
		maxPriestsGainable = 0
	}
	overflowPriests := preview.Priests - maxPriestsGainable
	if overflowPriests > 0 {
		if overflowPriests > player.Resources.Priests {
			overflowPriests = player.Resources.Priests
		}
		player.Resources.Priests -= overflowPriests
		player.Resources.Workers += overflowPriests
	}

	// 2) Free power-to-coin protection:
	// Convert spendable power to coins only when the same next-round power income would
	// still leave bowls I+II empty (all power active), preserving future flexibility.
	maxSafePowerToCoin := 0
	for spend := 1; spend <= player.Resources.Power.Bowl3; spend++ {
		candidate := player.Resources.Power.Clone()
		if err := candidate.SpendPower(spend); err != nil {
			break
		}
		candidate.GainPower(preview.Power)
		if candidate.Bowl1 == 0 && candidate.Bowl2 == 0 {
			maxSafePowerToCoin = spend
		}
	}
	if maxSafePowerToCoin > 0 {
		if err := player.Resources.Power.SpendPower(maxSafePowerToCoin); err == nil {
			player.Resources.Coins += maxSafePowerToCoin
		}
	}
}
