package az

import (
	"sort"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// oracleLegalActions deliberately does not call any production candidate
// generator. Only SearchAction.GameAction and the authoritative transition are
// shared, so a production pruning omission cannot certify itself.
func oracleLegalActions(position *GamePosition) []SearchAction {
	if position == nil || position.state == nil || position.IsTerminal() {
		return nil
	}
	ids := game.DecisionPlayerIDs(position.state)
	if len(ids) != 1 {
		return nil
	}
	playerID := ids[0]
	candidates := oracleCandidateUniverse(position.state, playerID)
	unique := make(map[string]SearchAction, len(candidates))
	for _, candidate := range candidates {
		action, err := candidate.GameAction(playerID)
		if err != nil {
			continue
		}
		clone := position.state.CloneForUndo()
		if game.ApplyActionToState(clone, action, game.StateActionOptions{}) == nil {
			canonical, err := SearchActionFromGameAction(action)
			if err == nil {
				unique[canonical.Key()] = canonical
			}
		}
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]SearchAction, 0, len(keys))
	for _, key := range keys {
		out = append(out, unique[key])
	}
	return out
}

func oracleCandidateUniverse(gs *game.GameState, playerID string) []SearchAction {
	baseFactionTypes := []models.FactionType{
		models.FactionNomads, models.FactionFakirs, models.FactionChaosMagicians,
		models.FactionGiants, models.FactionSwarmlings, models.FactionMermaids,
		models.FactionWitches, models.FactionAuren, models.FactionHalflings,
		models.FactionCultists, models.FactionAlchemists, models.FactionDarklings,
		models.FactionEngineers, models.FactionDwarves,
	}
	terrains := []models.TerrainType{
		models.TerrainPlains, models.TerrainSwamp, models.TerrainLake,
		models.TerrainForest, models.TerrainMountain, models.TerrainWasteland,
		models.TerrainDesert,
	}
	tracks := []game.CultTrack{game.CultFire, game.CultWater, game.CultEarth, game.CultAir}
	hexes := make([]board.Hex, 0, len(gs.Map.Hexes))
	for h := range gs.Map.Hexes {
		hexes = append(hexes, h)
	}
	sort.Slice(hexes, func(i, j int) bool {
		if hexes[i].R != hexes[j].R {
			return hexes[i].R < hexes[j].R
		}
		return hexes[i].Q < hexes[j].Q
	})

	out := make([]SearchAction, 0, 14000)
	for _, faction := range baseFactionTypes {
		out = append(out, SearchAction{Kind: game.ActionSelectFaction, Faction: faction})
	}
	for _, h := range hexes {
		out = append(out,
			SearchAction{Kind: game.ActionSetupDwelling, Hexes: []board.Hex{h}},
			SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{h}},
		)
		for _, terrain := range terrains {
			for _, build := range []bool{false, true} {
				for _, skip := range []bool{false, true} {
					out = append(out,
						SearchAction{Kind: game.ActionTransformAndBuild, Hexes: []board.Hex{h}, Terrain: terrain, Build: build, UseSkip: skip},
						SearchAction{Kind: game.ActionPowerAction, Power: game.PowerActionSpade1, Hexes: []board.Hex{h}, Build: build, UseSkip: skip},
						SearchAction{Kind: game.ActionPowerAction, Power: game.PowerActionSpade2, Hexes: []board.Hex{h}, Build: build, UseSkip: skip},
						SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionBonusCardSpade, Hexes: []board.Hex{h}, Terrain: terrain, Build: build, UseSkip: skip},
					)
				}
				out = append(out,
					SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionGiantsTransform, Hexes: []board.Hex{h}, Build: build},
					SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionNomadsSandstorm, Hexes: []board.Hex{h}, Build: build},
				)
			}
			out = append(out,
				SearchAction{Kind: game.ActionUseCultSpade, Hexes: []board.Hex{h}, Terrain: terrain},
				SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{h}, Terrain: terrain},
			)
		}
		for building := models.BuildingDwelling; building <= models.BuildingStronghold; building++ {
			out = append(out, SearchAction{Kind: game.ActionUpgradeBuilding, Hexes: []board.Hex{h}, Building: building})
		}
		out = append(out,
			SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionWitchesRide, Hexes: []board.Hex{h}},
			SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionSwarmlingsUpgrade, Hexes: []board.Hex{h}},
			SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionMermaidsRiverTown, Hexes: []board.Hex{h}},
		)
	}
	for i, h1 := range hexes {
		for j := i + 1; j < len(hexes); j++ {
			h2 := hexes[j]
			out = append(out,
				SearchAction{Kind: game.ActionPowerAction, Power: game.PowerActionBridge, Hexes: []board.Hex{h1, h2}},
				SearchAction{Kind: game.ActionEngineersBridge, Hexes: []board.Hex{h1, h2}},
			)
		}
	}

	out = append(out,
		SearchAction{Kind: game.ActionAdvanceShipping},
		SearchAction{Kind: game.ActionAdvanceDigging},
		SearchAction{Kind: game.ActionSkipHalflingsDwelling},
		SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionChaosMagiciansDoubleTurn},
	)
	for _, track := range tracks {
		for spaces := 1; spaces <= 3; spaces++ {
			out = append(out, SearchAction{Kind: game.ActionSendPriestToCult, Track: track, Amount: spaces})
		}
		out = append(out,
			SearchAction{Kind: game.ActionSelectCultistsCultTrack, Track: track},
			SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionAurenCultAdvance, Track: track},
			SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionWater2CultAdvance, Track: track},
			SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionBonusCardCultAdvance, Track: track},
		)
	}
	for power := game.PowerActionPriest; power <= game.PowerActionCoins; power++ {
		out = append(out, SearchAction{Kind: game.ActionPowerAction, Power: power})
	}
	for card := game.BonusCardPriest; card <= game.BonusCardStrongholdSanctuary; card++ {
		out = append(out,
			SearchAction{Kind: game.ActionPass, Card: card},
			SearchAction{Kind: game.ActionSetupBonusCard, Card: card},
		)
	}
	out = append(out, SearchAction{Kind: game.ActionPass, Card: game.BonusCardUnknown})
	for favor := game.FavorFire3; favor <= game.FavorAir1; favor++ {
		out = append(out, SearchAction{Kind: game.ActionSelectFavorTile, FavorTile: favor})
	}
	for amount := 0; amount <= 3; amount++ {
		out = append(out, SearchAction{Kind: game.ActionUseDarklingsPriestOrdination, Amount: amount})
	}
	for count := 1; count <= oraclePendingSpadeBound(gs, playerID); count++ {
		out = append(out, SearchAction{Kind: game.ActionDiscardPendingSpade, Amount: count})
	}
	out = append(out, oracleConversionCandidates(gs, playerID)...)
	out = append(out, oracleLeechCandidates(gs, playerID)...)
	out = append(out, oracleTownCandidates(gs, playerID)...)
	out = append(out, oracleTownCultCandidates(gs)...)
	return out
}

func oracleConversionCandidates(gs *game.GameState, playerID string) []SearchAction {
	player := gs.GetPlayer(playerID)
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		return nil
	}
	type domain struct {
		kind game.ConversionType
		max  int
		step int
	}
	domains := []domain{
		{game.ConversionPowerToCoin, player.Resources.Power.Bowl3, 1},
		{game.ConversionPowerToWorker, player.Resources.Power.Bowl3 / 3, 1},
		{game.ConversionPowerToPriest, player.Resources.Power.Bowl3 / 5, 1},
		{game.ConversionPriestToWorker, player.Resources.Priests, 1},
		{game.ConversionWorkerToCoin, player.Resources.Workers, 1},
		{game.ConversionAlchVPToCoin, player.VictoryPoints, 1},
		{game.ConversionAlchCoinToVP, player.Resources.Coins, 2},
	}
	var out []SearchAction
	for amount := 1; amount <= player.Resources.Power.Bowl2/2; amount++ {
		out = append(out, SearchAction{Kind: game.ActionBurnPower, Amount: amount})
	}
	for _, domain := range domains {
		for amount := domain.step; amount <= domain.max; amount += domain.step {
			out = append(out, SearchAction{Kind: game.ActionConversion, Conversion: domain.kind, Amount: amount})
		}
	}
	return out
}

func oracleLeechCandidates(gs *game.GameState, playerID string) []SearchAction {
	var out []SearchAction
	for index, offer := range gs.PendingLeechOffers[playerID] {
		out = append(out,
			SearchAction{Kind: game.ActionAcceptPowerLeech, Amount: index},
			SearchAction{Kind: game.ActionDeclinePowerLeech, Amount: index},
		)
		if offer != nil {
			for amount := 1; amount < offer.Amount; amount++ {
				out = append(out, SearchAction{Kind: game.ActionAcceptPowerLeech, Amount: index, Amount2: amount})
			}
		}
	}
	return out
}

func oracleTownCandidates(gs *game.GameState, playerID string) []SearchAction {
	pending := gs.PendingTownFormations[playerID]
	selected := oracleNextTownFormation(pending)
	if selected == nil {
		return nil
	}
	anchor, ok := oracleCanonicalTownAnchor(selected)
	if !ok {
		return nil
	}
	var out []SearchAction
	for tile := models.TownTile5Points; tile <= models.TownTile2Points; tile++ {
		out = append(out, SearchAction{Kind: game.ActionSelectTownTile, TownTile: tile, Hexes: []board.Hex{anchor}})
	}
	return out
}

func oracleNextTownFormation(pending []*game.PendingTownFormation) *game.PendingTownFormation {
	for _, formation := range pending {
		if formation != nil && !formation.CanBeDelayed {
			return formation
		}
	}
	for _, formation := range pending {
		if formation != nil {
			return formation
		}
	}
	return nil
}

func oracleCanonicalTownAnchor(pending *game.PendingTownFormation) (board.Hex, bool) {
	if pending == nil {
		return board.Hex{}, false
	}
	if pending.SkippedRiverHex != nil {
		return *pending.SkippedRiverHex, true
	}
	if len(pending.Hexes) == 0 {
		return board.Hex{}, false
	}
	hexes := append([]board.Hex(nil), pending.Hexes...)
	sort.Slice(hexes, func(i, j int) bool {
		if hexes[i].R != hexes[j].R {
			return hexes[i].R < hexes[j].R
		}
		return hexes[i].Q < hexes[j].Q
	})
	return hexes[0], true
}

func oracleTownCultCandidates(gs *game.GameState) []SearchAction {
	pending := gs.PendingTownCultTopChoice
	if pending == nil || pending.MaxSelections < 0 {
		return nil
	}
	tracks := append([]game.CultTrack(nil), pending.CandidateTracks...)
	sort.Slice(tracks, func(i, j int) bool { return tracks[i] < tracks[j] })
	var out []SearchAction
	var choose func(int, []game.CultTrack)
	choose = func(start int, selected []game.CultTrack) {
		if len(selected) == pending.MaxSelections {
			out = append(out, SearchAction{Kind: game.ActionSelectTownCultTop, Tracks: append([]game.CultTrack(nil), selected...)})
			return
		}
		for i := start; i < len(tracks); i++ {
			choose(i+1, append(selected, tracks[i]))
		}
	}
	choose(0, nil)
	return out
}

func oraclePendingSpadeBound(gs *game.GameState, playerID string) int {
	bound := gs.PendingSpades[playerID]
	if gs.PendingCultRewardSpades[playerID] > bound {
		bound = gs.PendingCultRewardSpades[playerID]
	}
	if bound < 3 {
		bound = 3
	}
	return bound
}
