package game

import "github.com/lukev/tm_server/internal/game/factions"

// AwardFactionSpadeBonuses awards faction-specific bonuses for using spades.
// This includes:
// - Halflings: +1 VP per spade
// - Alchemists: +2 power per spade (after building stronghold)
func AwardFactionSpadeBonuses(player *Player, spadesUsed int) {
	// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
	if halflings, ok := player.Faction.(*factions.Halflings); ok {
		vpBonus := halflings.GetVPPerSpade() * spadesUsed
		player.VictoryPoints += vpBonus
	}

	// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
	if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
		powerBonus := alchemists.GetPowerPerSpade() * spadesUsed
		if powerBonus > 0 {
			player.Resources.GainPower(powerBonus)
		}
	}
}
