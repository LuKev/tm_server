package az

import (
	"sort"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

var baseFactions = []models.FactionType{
	models.FactionNomads, models.FactionFakirs, models.FactionChaosMagicians,
	models.FactionGiants, models.FactionSwarmlings, models.FactionMermaids,
	models.FactionWitches, models.FactionAuren, models.FactionHalflings,
	models.FactionCultists, models.FactionAlchemists, models.FactionDarklings,
	models.FactionEngineers, models.FactionDwarves,
}

var landTerrains = []models.TerrainType{
	models.TerrainPlains, models.TerrainSwamp, models.TerrainLake,
	models.TerrainForest, models.TerrainMountain, models.TerrainWasteland,
	models.TerrainDesert,
}

var cultTracks = []game.CultTrack{game.CultFire, game.CultWater, game.CultEarth, game.CultAir}

func legalActions(gs *game.GameState) []SearchAction {
	ids := game.DecisionPlayerIDs(gs)
	if len(ids) != 1 {
		return nil
	}
	playerID := ids[0]
	return validatedActions(gs, playerID, productionCandidates(gs, playerID))
}

func validatedActions(gs *game.GameState, playerID string, candidates []SearchAction) []SearchAction {
	unique := make(map[string]SearchAction, len(candidates))
	for _, candidate := range candidates {
		canonical, ok := canonicalAppliedAction(gs, playerID, candidate)
		if ok {
			unique[canonical.Key()] = canonical
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

// canonicalAppliedAction returns the authoritative action after validation and
// execution resolve engine defaults such as skip auto-detection. Persisting
// that form prevents two policy keys from naming the same rules decision.
func canonicalAppliedAction(gs *game.GameState, playerID string, candidate SearchAction) (SearchAction, bool) {
	action, err := candidate.GameAction(playerID)
	if err != nil {
		return SearchAction{}, false
	}
	clone := gs.CloneForUndo()
	if game.ApplyActionToState(clone, action, game.StateActionOptions{}) != nil {
		return SearchAction{}, false
	}
	canonical, err := SearchActionFromGameAction(action)
	return canonical, err == nil
}

func productionCandidates(gs *game.GameState, playerID string) []SearchAction {
	if gs.PendingTownCultTopChoice != nil {
		return townCultTopCandidates(gs.PendingTownCultTopChoice)
	}
	if player := gs.GetPendingTownSelectionPlayer(); player != "" {
		return townCandidates(gs, player)
	}
	if gs.PendingDarklingsPriestOrdination != nil {
		return amountCandidates(game.ActionUseDarklingsPriestOrdination, 0, 3)
	}
	if gs.PendingFavorTileSelection != nil {
		return favorCandidates()
	}
	if gs.HasPendingLeechOffers() && gs.GetNextBlockingLeechResponder() != "" {
		return leechCandidates(gs, playerID)
	}
	if gs.PendingCultistsCultSelection != nil {
		return trackCandidates(game.ActionSelectCultistsCultTrack)
	}
	if gs.PendingHalflingsSpades != nil {
		return halflingsCandidates(gs)
	}
	if required, count := gs.GetPendingSpadeFollowupPlayer(); required != "" && count > 0 {
		return pendingSpadeCandidates(gs, false, count)
	}
	if required, count := gs.GetPendingCultRewardSpadePlayer(); required != "" && count > 0 {
		return pendingSpadeCandidates(gs, true, count)
	}

	switch gs.Phase {
	case game.PhaseFactionSelection:
		out := make([]SearchAction, 0, len(baseFactions))
		for _, faction := range baseFactions {
			out = append(out, SearchAction{Kind: game.ActionSelectFaction, Faction: faction})
		}
		return out
	case game.PhaseSetup:
		if gs.SetupSubphase == game.SetupSubphaseBonusCards {
			return bonusCardCandidates(game.ActionSetupBonusCard, false)
		}
		out := make([]SearchAction, 0, len(gs.Map.Hexes))
		for _, h := range sortedHexes(gs) {
			if mapHex := gs.Map.GetHex(h); mapHex != nil && mapHex.Building == nil && mapHex.Terrain != models.TerrainRiver {
				out = append(out, SearchAction{Kind: game.ActionSetupDwelling, Hexes: []board.Hex{h}})
			}
		}
		return out
	case game.PhaseAction:
		return normalCandidates(gs, playerID)
	default:
		return nil
	}
}

func normalCandidates(gs *game.GameState, playerID string) []SearchAction {
	hexes := sortedHexes(gs)
	out := make([]SearchAction, 0, 256)
	player := gs.GetPlayer(playerID)
	canSkip := player != nil && player.Faction != nil && (player.Faction.GetType() == models.FactionFakirs || player.Faction.GetType() == models.FactionDwarves)
	for _, h := range hexes {
		mapHex := gs.Map.GetHex(h)
		if mapHex == nil {
			continue
		}
		normallyReachable := gs.IsAdjacentToPlayerBuilding(h, playerID)
		skipReachable := false
		if !normallyReachable && canSkip {
			skipReachable = game.ValidateSkipAbility(gs, player, h) == nil
		}
		if mapHex.Building == nil && mapHex.Terrain != models.TerrainRiver && (normallyReachable || skipReachable) {
			useSkip := !normallyReachable
			for _, terrain := range landTerrains {
				for _, build := range []bool{false, true} {
					if terrain == mapHex.Terrain && !build {
						continue
					}
					out = append(out, SearchAction{Kind: game.ActionTransformAndBuild, Hexes: []board.Hex{h}, Terrain: terrain, Build: build, UseSkip: useSkip})
				}
			}
			for _, power := range []game.PowerActionType{game.PowerActionSpade1, game.PowerActionSpade2} {
				if mapHex.Terrain == player.Faction.GetHomeTerrain() || !canUseBasePowerAction(gs, player, power) {
					continue
				}
				for _, build := range []bool{false, true} {
					out = append(out, SearchAction{Kind: game.ActionPowerAction, Power: power, Hexes: []board.Hex{h}, Build: build, UseSkip: useSkip})
				}
			}
		}
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			for _, building := range []models.BuildingType{models.BuildingTradingHouse, models.BuildingTemple, models.BuildingSanctuary, models.BuildingStronghold} {
				out = append(out, SearchAction{Kind: game.ActionUpgradeBuilding, Hexes: []board.Hex{h}, Building: building})
			}
		}
	}
	powerBridge := canUseBasePowerAction(gs, player, game.PowerActionBridge) && player.BridgesBuilt < 3
	engineersBridge := player.Faction.GetType() == models.FactionEngineers && player.BridgesBuilt < 3
	if powerBridge || engineersBridge {
		for i, h1 := range hexes {
			for j := i + 1; j < len(hexes); j++ {
				h2 := hexes[j]
				if h1.Distance(h2) != 2 || gs.Map.ValidateBridgePlacement(h1, h2) != nil {
					continue
				}
				if powerBridge && bridgeTouchesPlayerStructure(gs, playerID, h1, h2) {
					out = append(out, SearchAction{Kind: game.ActionPowerAction, Power: game.PowerActionBridge, Hexes: []board.Hex{h1, h2}})
				}
				if engineersBridge && bridgeTouchesPlayerStructure(gs, playerID, h1, h2) {
					out = append(out, SearchAction{Kind: game.ActionEngineersBridge, Hexes: []board.Hex{h1, h2}})
				}
			}
		}
	}
	out = append(out, commonNonSpatialCandidates(gs, playerID)...)
	out = append(out, baseSpecialCandidates(gs, playerID)...)
	out = append(out, townCandidates(gs, playerID)...)
	return out
}

func canUseBasePowerAction(gs *game.GameState, player *game.Player, action game.PowerActionType) bool {
	if gs == nil || gs.PowerActions == nil || player == nil || player.Resources == nil || player.Resources.Power == nil {
		return false
	}
	if !gs.PowerActions.IsAvailable(action) {
		return false
	}
	cost := game.GetPowerCost(action)
	missing := cost - player.Resources.Power.Bowl3
	return missing <= 0 || player.Resources.Power.CanBurn(missing)
}

func bridgeTouchesPlayerStructure(gs *game.GameState, playerID string, first, second board.Hex) bool {
	for _, target := range []board.Hex{first, second} {
		mapHex := gs.Map.GetHex(target)
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			return true
		}
	}
	return false
}

func commonNonSpatialCandidates(gs *game.GameState, playerID string) []SearchAction {
	out := []SearchAction{
		{Kind: game.ActionAdvanceShipping},
		{Kind: game.ActionAdvanceDigging},
	}
	for _, track := range cultTracks {
		for spaces := 1; spaces <= 3; spaces++ {
			out = append(out, SearchAction{Kind: game.ActionSendPriestToCult, Track: track, Amount: spaces})
		}
	}
	for power := game.PowerActionPriest; power <= game.PowerActionCoins; power++ {
		out = append(out, SearchAction{Kind: game.ActionPowerAction, Power: power})
	}
	out = append(out, conversionCandidates(gs, playerID)...)
	out = append(out, bonusCardCandidates(game.ActionPass, gs.Round >= 6)...)
	return out
}

func conversionCandidates(gs *game.GameState, playerID string) []SearchAction {
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
	out := make([]SearchAction, 0, 32)
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

func baseSpecialCandidates(gs *game.GameState, playerID string) []SearchAction {
	hexes := sortedHexes(gs)
	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil {
		return nil
	}
	out := make([]SearchAction, 0, len(hexes)*2)
	if player.Faction.GetType() == models.FactionChaosMagicians && availableStrongholdSpecial(player, game.SpecialActionChaosMagiciansDoubleTurn) {
		out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionChaosMagiciansDoubleTurn})
	}
	waterCult := hasFavorTile(gs, playerID, game.FavorWater2) && !player.SpecialActionsUsed[game.SpecialActionWater2CultAdvance]
	bonusCult := hasBonusCard(gs, playerID, game.BonusCardCultAdvance) && !player.SpecialActionsUsed[game.SpecialActionBonusCardCultAdvance]
	aurenCult := player.Faction.GetType() == models.FactionAuren && availableStrongholdSpecial(player, game.SpecialActionAurenCultAdvance)
	for _, track := range cultTracks {
		if waterCult {
			out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionWater2CultAdvance, Track: track})
		}
		if bonusCult {
			out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionBonusCardCultAdvance, Track: track})
		}
		if aurenCult {
			out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionAurenCultAdvance, Track: track})
		}
	}
	spadeCard := hasBonusCard(gs, playerID, game.BonusCardSpade) && !player.SpecialActionsUsed[game.SpecialActionBonusCardSpade]
	for _, h := range hexes {
		mapHex := gs.Map.GetHex(h)
		if mapHex == nil {
			continue
		}
		switch player.Faction.GetType() {
		case models.FactionWitches:
			if availableStrongholdSpecial(player, game.SpecialActionWitchesRide) && mapHex.Building == nil && mapHex.Terrain == models.TerrainForest {
				out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionWitchesRide, Hexes: []board.Hex{h}})
			}
		case models.FactionSwarmlings:
			if availableStrongholdSpecial(player, game.SpecialActionSwarmlingsUpgrade) && mapHex.Building != nil && mapHex.Building.PlayerID == playerID && mapHex.Building.Type == models.BuildingDwelling {
				out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionSwarmlingsUpgrade, Hexes: []board.Hex{h}})
			}
		case models.FactionMermaids:
			if mapHex.Terrain == models.TerrainRiver {
				out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionMermaidsRiverTown, Hexes: []board.Hex{h}})
			}
		case models.FactionGiants:
			if availableStrongholdSpecial(player, game.SpecialActionGiantsTransform) && mapHex.Building == nil && mapHex.Terrain != player.Faction.GetHomeTerrain() && gs.IsAdjacentToPlayerBuilding(h, playerID) {
				for _, build := range []bool{false, true} {
					out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionGiantsTransform, Hexes: []board.Hex{h}, Build: build})
				}
			}
		case models.FactionNomads:
			if availableStrongholdSpecial(player, game.SpecialActionNomadsSandstorm) && mapHex.Building == nil && mapHex.Terrain != player.Faction.GetHomeTerrain() && directlyAdjacentToPlayer(gs, h, playerID) {
				for _, build := range []bool{false, true} {
					out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionNomadsSandstorm, Hexes: []board.Hex{h}, Build: build})
				}
			}
		}
		if spadeCard && mapHex.Building == nil && mapHex.Terrain != models.TerrainRiver {
			useSkip := false
			if player.Faction.GetType() == models.FactionFakirs || player.Faction.GetType() == models.FactionDwarves {
				useSkip = !gs.IsAdjacentToPlayerBuilding(h, playerID)
				if useSkip && game.ValidateSkipAbility(gs, player, h) != nil {
					continue
				}
			} else if !gs.IsAdjacentToPlayerBuilding(h, playerID) {
				continue
			}
			for _, terrain := range landTerrains {
				if terrain == mapHex.Terrain {
					continue
				}
				for _, build := range []bool{false, true} {
					out = append(out, SearchAction{Kind: game.ActionSpecialAction, Special: game.SpecialActionBonusCardSpade, Hexes: []board.Hex{h}, Terrain: terrain, Build: build, UseSkip: useSkip})
				}
			}
		}
	}
	return out
}

func availableStrongholdSpecial(player *game.Player, action game.SpecialActionType) bool {
	return player != nil && player.HasStrongholdAbility && !player.SpecialActionsUsed[action]
}

func hasBonusCard(gs *game.GameState, playerID string, wanted game.BonusCardType) bool {
	for _, card := range gs.BonusCards.GetPlayerCards(playerID) {
		if card == wanted {
			return true
		}
	}
	return false
}

func hasFavorTile(gs *game.GameState, playerID string, wanted game.FavorTileType) bool {
	for _, tile := range gs.FavorTiles.GetPlayerTiles(playerID) {
		if tile == wanted {
			return true
		}
	}
	return false
}

func directlyAdjacentToPlayer(gs *game.GameState, target board.Hex, playerID string) bool {
	for _, neighbor := range target.Neighbors() {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			return true
		}
	}
	return false
}

func bonusCardCandidates(kind game.ActionType, includeNoCard bool) []SearchAction {
	out := make([]SearchAction, 0, 11)
	if includeNoCard {
		out = append(out, SearchAction{Kind: kind, Card: game.BonusCardUnknown})
	}
	for card := game.BonusCardPriest; card <= game.BonusCardStrongholdSanctuary; card++ {
		out = append(out, SearchAction{Kind: kind, Card: card})
	}
	return out
}

func favorCandidates() []SearchAction {
	out := make([]SearchAction, 0, 12)
	for tile := game.FavorFire3; tile <= game.FavorAir1; tile++ {
		out = append(out, SearchAction{Kind: game.ActionSelectFavorTile, FavorTile: tile})
	}
	return out
}

func trackCandidates(kind game.ActionType) []SearchAction {
	out := make([]SearchAction, 0, 4)
	for _, track := range cultTracks {
		out = append(out, SearchAction{Kind: kind, Track: track})
	}
	return out
}

func amountCandidates(kind game.ActionType, minAmount, maxAmount int) []SearchAction {
	out := make([]SearchAction, 0, maxAmount-minAmount+1)
	for amount := minAmount; amount <= maxAmount; amount++ {
		out = append(out, SearchAction{Kind: kind, Amount: amount})
	}
	return out
}

func leechCandidates(gs *game.GameState, playerID string) []SearchAction {
	offers := gs.PendingLeechOffers[playerID]
	out := make([]SearchAction, 0, len(offers)*4)
	for index, offer := range offers {
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

func townCandidates(gs *game.GameState, playerID string) []SearchAction {
	pending := gs.PendingTownFormations[playerID]
	selected := nextTownFormation(pending)
	if selected == nil {
		return nil
	}
	anchor, ok := canonicalTownAnchor(selected)
	if !ok {
		return nil
	}
	tiles := gs.TownTiles.GetAvailableTiles()
	sort.Slice(tiles, func(i, j int) bool { return tiles[i] < tiles[j] })
	out := make([]SearchAction, 0, len(tiles))
	for _, tile := range tiles {
		out = append(out, SearchAction{Kind: game.ActionSelectTownTile, TownTile: tile, Hexes: []board.Hex{anchor}})
	}
	return out
}

func nextTownFormation(pending []*game.PendingTownFormation) *game.PendingTownFormation {
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

func canonicalTownAnchor(pending *game.PendingTownFormation) (board.Hex, bool) {
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

func townCultTopCandidates(pending *game.PendingTownCultTopChoice) []SearchAction {
	if pending == nil || pending.MaxSelections < 0 {
		return nil
	}
	tracks := append([]game.CultTrack(nil), pending.CandidateTracks...)
	sort.Slice(tracks, func(i, j int) bool { return tracks[i] < tracks[j] })
	var out []SearchAction
	var visit func(start int, selected []game.CultTrack)
	visit = func(start int, selected []game.CultTrack) {
		if len(selected) == pending.MaxSelections {
			out = append(out, SearchAction{Kind: game.ActionSelectTownCultTop, Tracks: append([]game.CultTrack(nil), selected...)})
			return
		}
		for i := start; i < len(tracks); i++ {
			visit(i+1, append(selected, tracks[i]))
		}
	}
	visit(0, nil)
	return out
}

func halflingsCandidates(gs *game.GameState) []SearchAction {
	out := []SearchAction{{Kind: game.ActionSkipHalflingsDwelling}}
	for _, h := range sortedHexes(gs) {
		out = append(out, SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{h}})
		for _, terrain := range landTerrains {
			out = append(out, SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{h}, Terrain: terrain})
		}
	}
	return out
}

func pendingSpadeCandidates(gs *game.GameState, cult bool, count int) []SearchAction {
	out := amountCandidates(game.ActionDiscardPendingSpade, 1, count)
	for _, h := range sortedHexes(gs) {
		for _, terrain := range landTerrains {
			if cult {
				out = append(out, SearchAction{Kind: game.ActionUseCultSpade, Hexes: []board.Hex{h}, Terrain: terrain})
			} else {
				for _, build := range []bool{false, true} {
					out = append(out, SearchAction{Kind: game.ActionTransformAndBuild, Hexes: []board.Hex{h}, Terrain: terrain, Build: build})
				}
			}
		}
	}
	return out
}

func sortedHexes(gs *game.GameState) []board.Hex {
	out := make([]board.Hex, 0, len(gs.Map.Hexes))
	for h := range gs.Map.Hexes {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].R != out[j].R {
			return out[i].R < out[j].R
		}
		return out[i].Q < out[j].Q
	})
	return out
}
