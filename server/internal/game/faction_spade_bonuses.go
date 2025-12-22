package game

import (
	"github.com/lukev/tm_server/internal/models"
)

// AwardFactionSpadeBonuses awards faction-specific bonuses for using spades.
// This includes:
// - Halflings: +1 VP per spade
// - Alchemists: +2 power per spade (after building stronghold)
func AwardFactionSpadeBonuses(player *Player, spadesUsed int) {
	// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
	// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
	if player.Faction.GetType() == models.FactionHalflings {
		vpBonus := 1 * spadesUsed
		player.VictoryPoints += vpBonus
	}

	// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
	// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
	if player.Faction.GetType() == models.FactionAlchemists && player.HasStrongholdAbility {
		powerBonus := 2 * spadesUsed
		if powerBonus > 0 {
			player.Resources.GainPower(powerBonus)
		}
	}
}
