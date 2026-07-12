package actions

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// Option is one concrete engine move. ID is stable within the current rules
// version and is the policy-head key used by MCTS, self-play, and training.
type Option struct {
	ID       string            `json:"id"`
	PlayerID string            `json:"playerId"`
	Type     string            `json:"type"`
	Label    string            `json:"label"`
	Meta     map[string]string `json:"meta,omitempty"`
	Params   map[string]any    `json:"params,omitempty"`
	Action   game.Action       `json:"-"`
}

// LegalActions returns legal actions for the current state by over-generating
// candidates and filtering through the live multiplayer engine.
func LegalActions(gs *game.GameState) []Option {
	if gs == nil || gs.Phase == game.PhaseEnd {
		return nil
	}
	candidates := generateCandidates(gs)
	seen := make(map[string]bool, len(candidates))
	legal := make([]Option, 0, len(candidates))
	for _, option := range candidates {
		if option.Action == nil || seen[option.ID] {
			continue
		}
		seen[option.ID] = true
		if IsLegal(gs, option.Action) {
			legal = append(legal, option)
		}
	}
	sort.Slice(legal, func(i, j int) bool {
		return legal[i].ID < legal[j].ID
	})
	return legal
}

// IsLegal validates an action by executing it on a cloned state through the
// same manager path used by live games.
func IsLegal(gs *game.GameState, action game.Action) bool {
	if gs == nil || action == nil {
		return false
	}
	_, err := ApplyToClone(gs, action)
	return err == nil
}

// ApplyToClone executes an action on a clone and returns the resulting state.
func ApplyToClone(gs *game.GameState, action game.Action) (*game.GameState, error) {
	if gs == nil {
		return nil, fmt.Errorf("nil game state")
	}
	if action == nil {
		return nil, fmt.Errorf("nil action")
	}
	clone := gs.CloneForUndo()
	disableConfirmations(clone)
	enableReplayAutoConversions(clone)
	mgr := game.NewManager()
	mgr.CreateGameWithState("__az__", clone)
	if _, err := mgr.ExecuteActionWithMeta("__az__", action, game.ActionMeta{ExpectedRevision: -1}); err != nil {
		return nil, err
	}
	next, ok := mgr.GetGame("__az__")
	if !ok || next == nil {
		return nil, fmt.Errorf("missing cloned state after action")
	}
	return next, nil
}

func disableConfirmations(gs *game.GameState) {
	if gs == nil {
		return
	}
	gs.PendingTurnConfirmationPlayerID = ""
	gs.PendingTurnConfirmationSnapshot = nil
	for _, player := range gs.Players {
		if player == nil {
			continue
		}
		player.Options.ConfirmActions = false
	}
}

func enableReplayAutoConversions(gs *game.GameState) {
	game.EnableAZAutoConversionsForClone(gs)
}

func generateCandidates(gs *game.GameState) []Option {
	var out []Option
	for _, playerID := range sortedPlayerIDs(gs) {
		out = append(out, pendingCandidates(gs, playerID)...)
	}
	if current := gs.GetCurrentPlayer(); current != nil {
		out = append(out, mainTurnCandidates(gs, current.ID)...)
	}
	return out
}

func pendingCandidates(gs *game.GameState, playerID string) []Option {
	var out []Option
	if offers := gs.PendingLeechOffers[playerID]; len(offers) > 0 {
		for i, offer := range offers {
			out = append(out,
				option(playerID, "leech_accept", fmt.Sprintf("Accept %d power leech", offer.Amount), game.NewAcceptPowerLeechAction(playerID, i), "leech_accept", i),
				option(playerID, "leech_decline", "Decline power leech", game.NewDeclinePowerLeechAction(playerID, i), "leech_decline", i),
			)
		}
	}
	if gs.PendingFavorTileSelection != nil && gs.PendingFavorTileSelection.PlayerID == playerID {
		for _, tile := range availableFavorTiles(gs) {
			out = append(out, option(playerID, "favor", fmt.Sprintf("Take favor %d", tile), &game.SelectFavorTileAction{
				BaseAction: game.BaseAction{Type: game.ActionSelectFavorTile, PlayerID: playerID},
				TileType:   tile,
			}, "favor", int(tile)))
		}
	}
	if pendingTowns := gs.PendingTownFormations[playerID]; len(pendingTowns) > 0 {
		anchors := townAnchors(pendingTowns[0])
		for _, tile := range availableTownTiles(gs) {
			for _, anchor := range anchors {
				hex := anchor
				out = append(out, option(playerID, "town", fmt.Sprintf("Take town tile %d", tile), &game.SelectTownTileAction{
					BaseAction: game.BaseAction{Type: game.ActionSelectTownTile, PlayerID: playerID},
					TileType:   tile,
					AnchorHex:  &hex,
				}, "town", int(tile), hex.Q, hex.R))
			}
		}
	}
	if gs.PendingTownCultTopChoice != nil && gs.PendingTownCultTopChoice.PlayerID == playerID {
		for _, tracks := range cultTrackSelections(gs.PendingTownCultTopChoice.CandidateTracks, gs.PendingTownCultTopChoice.MaxSelections) {
			idParts := append([]interface{}{"town_cult_top"}, trackInts(tracks)...)
			out = append(out, option(playerID, "town_cult_top", "Resolve town cult top choice", &game.SelectTownCultTopAction{
				BaseAction: game.BaseAction{Type: game.ActionSelectTownCultTop, PlayerID: playerID},
				Tracks:     tracks,
			}, idParts...))
		}
	}
	if gs.PendingDarklingsPriestOrdination != nil && gs.PendingDarklingsPriestOrdination.PlayerID == playerID {
		for workers := 0; workers <= 3; workers++ {
			out = append(out, option(playerID, "darklings_ordination", fmt.Sprintf("Ordain %d workers", workers), &game.UseDarklingsPriestOrdinationAction{
				BaseAction:       game.BaseAction{Type: game.ActionUseDarklingsPriestOrdination, PlayerID: playerID},
				WorkersToConvert: workers,
			}, "darklings_ordination", workers))
		}
	}
	if gs.PendingTreasurersDeposit != nil && gs.PendingTreasurersDeposit.PlayerID == playerID {
		p := gs.PendingTreasurersDeposit
		for c := 0; c <= p.AvailableCoins; c++ {
			for w := 0; w <= p.AvailableWorkers; w++ {
				for pr := 0; pr <= p.AvailablePriests; pr++ {
					out = append(out, option(playerID, "treasury", fmt.Sprintf("Treasury %dC %dW %dP", c, w, pr), game.NewSelectTreasurersDepositAction(playerID, c, w, pr), "treasury", c, w, pr))
				}
			}
		}
	}
	if gs.PendingRiverwalkersPriestChoice != nil && gs.PendingRiverwalkersPriestChoice.PlayerID == playerID {
		out = append(out, option(playerID, "riverwalkers_priest", "Take Riverwalkers priest", game.NewSelectRiverwalkersPriestChoiceAction(playerID, true, models.TerrainTypeUnknown), "river_priest"))
		for _, terrain := range standardTerrains() {
			out = append(out, option(playerID, "riverwalkers_unlock", fmt.Sprintf("Unlock %s", terrain), game.NewSelectRiverwalkersPriestChoiceAction(playerID, false, terrain), "river_unlock", int(terrain)))
		}
	}
	if gs.PendingCultistsCultSelection != nil && gs.PendingCultistsCultSelection.PlayerID == playerID {
		for _, track := range allCultTracks() {
			out = append(out, option(playerID, "cultists_track", fmt.Sprintf("Cultists choose %d", track), game.NewSelectCultistsCultTrackAction(playerID, track), "cultists_track", int(track)))
		}
	}
	if gs.PendingDjinniStartingCultChoice != nil && gs.PendingDjinniStartingCultChoice.PlayerID == playerID {
		for _, track := range allCultTracks() {
			out = append(out, option(playerID, "djinni_start", fmt.Sprintf("Djinni start %d", track), game.NewSelectDjinniStartingCultTrackAction(playerID, track), "djinni_start", int(track)))
		}
	}
	if gs.PendingGoblinsCultSteps != nil && gs.PendingGoblinsCultSteps.PlayerID == playerID {
		for _, track := range allCultTracks() {
			out = append(out, option(playerID, "goblins_cult", fmt.Sprintf("Goblins choose %d", track), game.NewSelectGoblinsCultTrackAction(playerID, track), "goblins_cult", int(track)))
		}
	}
	if gs.PendingArchivistsBonusSelection != nil && gs.PendingArchivistsBonusSelection.PlayerID == playerID {
		for _, card := range allBonusCards() {
			out = append(out, option(playerID, "archivists_bonus", fmt.Sprintf("Archivists take bonus %d", card), game.NewSelectArchivistsBonusCardAction(playerID, card), "archivists_bonus", int(card)))
		}
	}
	if gs.PendingHalflingsSpades != nil && gs.PendingHalflingsSpades.PlayerID == playerID {
		for _, hex := range sortedHexes(gs) {
			for _, terrain := range buildableTerrains() {
				out = append(out, option(playerID, "halflings_spade", fmt.Sprintf("Halflings spade %s", terrain), &game.ApplyHalflingsSpadeAction{
					BaseAction:    game.BaseAction{Type: game.ActionApplyHalflingsSpade, PlayerID: playerID},
					TargetHex:     hex,
					TargetTerrain: terrain,
				}, "halflings_spade", hex.Q, hex.R, int(terrain)))
			}
			out = append(out, option(playerID, "halflings_dwelling", "Build Halflings dwelling", &game.BuildHalflingsDwellingAction{
				BaseAction: game.BaseAction{Type: game.ActionBuildHalflingsDwelling, PlayerID: playerID},
				TargetHex:  hex,
			}, "halflings_dwelling", hex.Q, hex.R))
		}
		out = append(out, option(playerID, "halflings_skip", "Skip Halflings dwelling", &game.SkipHalflingsDwellingAction{
			BaseAction: game.BaseAction{Type: game.ActionSkipHalflingsDwelling, PlayerID: playerID},
		}, "halflings_skip"))
	}
	if gs.PendingWispsStrongholdDwelling != nil && gs.PendingWispsStrongholdDwelling.PlayerID == playerID {
		for _, hex := range sortedHexes(gs) {
			out = append(out, option(playerID, "wisps_lake_dwelling", "Build Wisps lake dwelling", game.NewBuildWispsStrongholdDwellingAction(playerID, hex), "wisps_lake", hex.Q, hex.R))
		}
	}
	if player, count := gs.GetPendingSpadeFollowupPlayer(); player == playerID && count > 0 {
		out = append(out, transformCandidates(gs, playerID, "pending_spade", true)...)
		out = append(out, option(playerID, "discard_spade", "Discard pending spade", game.NewDiscardPendingSpadeAction(playerID, 1), "discard_spade"))
	}
	if player, count := gs.GetPendingCultRewardSpadePlayer(); player == playerID && count > 0 {
		for _, hex := range sortedHexes(gs) {
			for _, terrain := range buildableTerrains() {
				out = append(out, option(playerID, "cult_spade", "Use cult spade", game.NewUseCultSpadeActionWithTerrain(playerID, hex, terrain), "cult_spade", hex.Q, hex.R, int(terrain)))
			}
		}
		out = append(out, option(playerID, "discard_cult_spade", "Discard cult spade", game.NewDiscardPendingSpadeAction(playerID, 1), "discard_cult_spade"))
	}
	return out
}

func mainTurnCandidates(gs *game.GameState, playerID string) []Option {
	var out []Option
	if player := gs.GetPlayer(playerID); player != nil && player.HasPassed {
		return out
	}
	if gs.Phase == game.PhaseFactionSelection {
		for _, faction := range baseFactions() {
			out = append(out, option(playerID, "select_faction", fmt.Sprintf("Select %s", faction), &game.SelectFactionAction{PlayerID: playerID, FactionType: faction}, "faction", int(faction)))
		}
	}
	if gs.Phase == game.PhaseSetup {
		for _, hex := range sortedHexes(gs) {
			out = append(out, option(playerID, "setup_dwelling", "Place setup dwelling", game.NewSetupDwellingAction(playerID, hex), "setup_dwelling", hex.Q, hex.R))
		}
		for _, card := range allBonusCards() {
			out = append(out, option(playerID, "setup_bonus", fmt.Sprintf("Take setup bonus %d", card), &game.SetupBonusCardAction{
				BaseAction: game.BaseAction{Type: game.ActionSetupBonusCard, PlayerID: playerID},
				BonusCard:  card,
			}, "setup_bonus", int(card)))
		}
	}
	if gs.Phase != game.PhaseAction {
		return out
	}
	out = append(out, transformCandidates(gs, playerID, "transform", false)...)
	out = append(out, upgradeCandidates(gs, playerID)...)
	out = append(out, option(playerID, "advance_shipping", "Advance shipping", game.NewAdvanceShippingAction(playerID), "shipping"))
	out = append(out, option(playerID, "advance_digging", "Advance digging", game.NewAdvanceDiggingAction(playerID), "digging"))
	out = append(out, option(playerID, "advance_chash", "Advance Chash Dallah track", game.NewAdvanceChashTrackAction(playerID), "chash"))
	for _, track := range allCultTracks() {
		for spaces := 1; spaces <= 3; spaces++ {
			out = append(out, option(playerID, "cult_priest", fmt.Sprintf("Send priest %d spaces", spaces), &game.SendPriestToCultAction{
				BaseAction:    game.BaseAction{Type: game.ActionSendPriestToCult, PlayerID: playerID},
				Track:         track,
				SpacesToClimb: spaces,
			}, "cult_priest", int(track), spaces))
		}
	}
	out = append(out, powerActionCandidates(gs, playerID)...)
	out = append(out, specialActionCandidates(gs, playerID)...)
	out = append(out, strategicConversionCandidates(gs, playerID)...)
	for _, card := range availablePassCards(gs, playerID) {
		c := card
		out = append(out, option(playerID, "pass", fmt.Sprintf("Pass for bonus %d", card), game.NewPassAction(playerID, &c), "pass", int(card)))
	}
	if gs.Round >= 6 {
		out = append(out, option(playerID, "pass_final", "Pass final round", game.NewPassAction(playerID, nil), "pass_final"))
	}
	return out
}

func transformCandidates(gs *game.GameState, playerID, kind string, includeTransformOnly bool) []Option {
	var out []Option
	for _, hex := range actionTargetHexes(gs, playerID) {
		for _, terrain := range buildableTerrains() {
			if includeTransformOnly {
				out = append(out, option(playerID, kind, fmt.Sprintf("Transform %d,%d to %s", hex.Q, hex.R, terrain), game.NewTransformAndBuildAction(playerID, hex, false, terrain), kind, hex.Q, hex.R, int(terrain), 0))
			}
			out = append(out, option(playerID, kind+"_build", fmt.Sprintf("Transform/build %d,%d", hex.Q, hex.R), game.NewTransformAndBuildAction(playerID, hex, true, terrain), kind+"_build", hex.Q, hex.R, int(terrain), 1))
		}
	}
	return out
}

func upgradeCandidates(gs *game.GameState, playerID string) []Option {
	var out []Option
	for _, hex := range sortedHexes(gs) {
		mapHex := gs.Map.GetHex(hex)
		if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
			continue
		}
		for _, building := range upgradeDestinations(mapHex.Building.Type) {
			out = append(out, option(playerID, "upgrade", fmt.Sprintf("Upgrade %d,%d to %s", hex.Q, hex.R, building), game.NewUpgradeBuildingAction(playerID, hex, building), "upgrade", hex.Q, hex.R, int(building)))
		}
	}
	return out
}

func upgradeDestinations(from models.BuildingType) []models.BuildingType {
	switch from {
	case models.BuildingDwelling:
		return []models.BuildingType{models.BuildingTradingHouse}
	case models.BuildingTradingHouse:
		return []models.BuildingType{models.BuildingTemple, models.BuildingStronghold}
	case models.BuildingTemple:
		return []models.BuildingType{models.BuildingSanctuary}
	default:
		return nil
	}
}

func powerActionCandidates(gs *game.GameState, playerID string) []Option {
	var out []Option
	for _, powerType := range []game.PowerActionType{game.PowerActionBridge, game.PowerActionPriest, game.PowerActionWorkers, game.PowerActionCoins, game.PowerActionSpade1, game.PowerActionSpade2} {
		out = append(out, option(playerID, "power", fmt.Sprintf("Power action %d", powerType), game.NewPowerAction(playerID, powerType), "power", int(powerType)))
		if powerType == game.PowerActionSpade1 || powerType == game.PowerActionSpade2 {
			for _, hex := range actionTargetHexes(gs, playerID) {
				out = append(out,
					option(playerID, "power_spade", "Power spade", game.NewPowerActionWithTransform(playerID, powerType, hex, false), "power_spade", int(powerType), hex.Q, hex.R, 0),
					option(playerID, "power_spade_build", "Power spade build", game.NewPowerActionWithTransform(playerID, powerType, hex, true), "power_spade_build", int(powerType), hex.Q, hex.R, 1),
				)
			}
		}
		if powerType == game.PowerActionBridge {
			for _, pair := range bridgePairs(gs, playerID) {
				out = append(out, option(playerID, "power_bridge", "Power bridge", game.NewPowerActionWithBridge(playerID, pair[0], pair[1]), "power_bridge", pair[0].Q, pair[0].R, pair[1].Q, pair[1].R))
			}
		}
	}
	return out
}

func specialActionCandidates(gs *game.GameState, playerID string) []Option {
	var out []Option
	player := gs.GetPlayer(playerID)
	faction := playerFaction(player)
	if faction == models.FactionUnknown {
		return out
	}
	if specialActionReady(player, game.SpecialActionAurenCultAdvance) && faction == models.FactionAuren {
		for _, track := range allCultTracks() {
			out = append(out, option(playerID, "special_auren", "Auren cult advance", game.NewAurenCultAdvanceAction(playerID, track), "special_auren", int(track)))
		}
	}
	if specialActionUnused(player, game.SpecialActionWater2CultAdvance) && hasFavor(gs, playerID, game.FavorWater2) {
		for _, track := range allCultTracks() {
			out = append(out, option(playerID, "special_water2", "Water+2 cult action", game.NewWater2CultAdvanceAction(playerID, track), "special_water2", int(track)))
		}
	}
	if specialActionUnused(player, game.SpecialActionBonusCardCultAdvance) && hasBonusCard(gs, playerID, game.BonusCardCultAdvance) {
		for _, track := range allCultTracks() {
			out = append(out, option(playerID, "special_bonus_cult", "Bonus cult action", game.NewBonusCardCultAction(playerID, track), "special_bonus_cult", int(track)))
		}
	}
	for _, hex := range actionTargetHexes(gs, playerID) {
		if specialActionReady(player, game.SpecialActionWitchesRide) && faction == models.FactionWitches {
			out = append(out, option(playerID, "special_witches", "Witches ride", game.NewWitchesRideAction(playerID, hex), "special_witches", hex.Q, hex.R))
		}
		if specialActionReady(player, game.SpecialActionGiantsTransform) && faction == models.FactionGiants {
			out = append(out,
				option(playerID, "special_giants", "Giants transform", game.NewGiantsTransformAction(playerID, hex, false), "special_giants", hex.Q, hex.R, 0),
				option(playerID, "special_giants_build", "Giants transform/build", game.NewGiantsTransformAction(playerID, hex, true), "special_giants_build", hex.Q, hex.R, 1),
			)
		}
		if specialActionReady(player, game.SpecialActionNomadsSandstorm) && faction == models.FactionNomads {
			out = append(out,
				option(playerID, "special_nomads", "Nomads sandstorm", game.NewNomadsSandstormAction(playerID, hex, false), "special_nomads", hex.Q, hex.R, 0),
				option(playerID, "special_nomads_build", "Nomads sandstorm/build", game.NewNomadsSandstormAction(playerID, hex, true), "special_nomads_build", hex.Q, hex.R, 1),
			)
		}
		if specialActionUnused(player, game.SpecialActionBonusCardSpade) && hasBonusCard(gs, playerID, game.BonusCardSpade) {
			for _, terrain := range buildableTerrains() {
				out = append(out,
					option(playerID, "special_bonus_spade", "Bonus spade", game.NewBonusCardSpadeAction(playerID, hex, false, terrain), "special_bonus_spade", hex.Q, hex.R, int(terrain), 0),
					option(playerID, "special_bonus_spade_build", "Bonus spade build", game.NewBonusCardSpadeAction(playerID, hex, true, terrain), "special_bonus_spade_build", hex.Q, hex.R, int(terrain), 1),
				)
			}
		}
		if specialActionReady(player, game.SpecialActionSelkiesStronghold) && faction == models.FactionSelkies {
			for _, terrain := range buildableTerrains() {
				out = append(out,
					option(playerID, "special_selkies", "Selkies stronghold", game.NewSelkiesStrongholdAction(playerID, hex, false, terrain), "special_selkies", hex.Q, hex.R, int(terrain), 0),
					option(playerID, "special_selkies_build", "Selkies stronghold/build", game.NewSelkiesStrongholdAction(playerID, hex, true, terrain), "special_selkies_build", hex.Q, hex.R, int(terrain), 1),
				)
			}
		}
	}
	if faction == models.FactionMermaids {
		for _, hex := range sortedRiverHexes(gs) {
			out = append(out, option(playerID, "special_mermaids_town", "Mermaids river town", game.NewMermaidsRiverTownAction(playerID, hex), "special_mermaids_town", hex.Q, hex.R))
		}
	}
	if faction == models.FactionEngineers || faction == models.FactionAtlanteans || faction == models.FactionArchitects {
		for _, pair := range bridgePairs(gs, playerID) {
			out = append(out, option(playerID, "engineers_bridge", "Engineers bridge", game.NewEngineersBridgeAction(playerID, pair[0], pair[1]), "engineers_bridge", pair[0].Q, pair[0].R, pair[1].Q, pair[1].R))
		}
	}
	if faction == models.FactionGoblins && player.GoblinTreasureTokens > 0 {
		for _, reward := range []game.GoblinsTreasureRewardType{game.GoblinsTreasureDwellings, game.GoblinsTreasureTradingPosts, game.GoblinsTreasureTemples, game.GoblinsTreasureBigStructures} {
			out = append(out, option(playerID, "goblins_treasure", fmt.Sprintf("Goblins treasure %s", reward), game.NewUseGoblinsTreasureAction(playerID, reward), "goblins_treasure", reward))
		}
	}
	if specialActionReady(player, game.SpecialActionSwarmlingsUpgrade) && faction == models.FactionSwarmlings {
		for _, hex := range ownedBuildingHexes(gs, playerID, models.BuildingDwelling) {
			out = append(out, option(playerID, "special_swarmlings", "Swarmlings upgrade", game.NewSwarmlingsUpgradeAction(playerID, hex), "special_swarmlings", hex.Q, hex.R))
		}
	}
	if specialActionReady(player, game.SpecialActionEnlightenedGainPower) && faction == models.FactionTheEnlightened {
		out = append(out, option(playerID, "special_enlightened", "Enlightened gain power", game.NewEnlightenedGainPowerAction(playerID), "special_enlightened"))
	}
	if specialActionReady(player, game.SpecialActionProspectorsGainCoins) && faction == models.FactionProspectors {
		out = append(out, option(playerID, "special_prospectors", "Prospectors gain coins", game.NewProspectorsGainCoinsAction(playerID), "special_prospectors"))
	}
	if specialActionReady(player, game.SpecialActionTimeTravelersPowerShift) && faction == models.FactionTimeTravelers {
		for amount := 1; amount <= 4; amount++ {
			out = append(out, option(playerID, "special_timetravelers", "Time Travelers power shift", game.NewTimeTravelersPowerShiftAction(playerID, amount), "special_timetravelers", amount))
		}
	}
	if player.HasStrongholdAbility && faction == models.FactionShapeshifters {
		for _, terrain := range buildableTerrains() {
			out = append(out, option(playerID, "special_shapeshift", fmt.Sprintf("Shapeshift to %s", terrain), game.NewShapeshiftersShiftTerrainAction(playerID, terrain), "special_shapeshift", int(terrain)))
		}
	}
	return out
}

func playerFaction(player *game.Player) models.FactionType {
	if player == nil || player.Faction == nil {
		return models.FactionUnknown
	}
	return player.Faction.GetType()
}

func specialActionReady(player *game.Player, actionType game.SpecialActionType) bool {
	if player == nil || !player.HasStrongholdAbility {
		return false
	}
	return specialActionUnused(player, actionType)
}

func specialActionUnused(player *game.Player, actionType game.SpecialActionType) bool {
	if player == nil {
		return false
	}
	if !specialActionTracksUsage(actionType) {
		return true
	}
	return player.SpecialActionsUsed == nil || !player.SpecialActionsUsed[actionType]
}

func specialActionTracksUsage(actionType game.SpecialActionType) bool {
	return actionType != game.SpecialActionMermaidsRiverTown &&
		actionType != game.SpecialActionDjinniSwapCults &&
		actionType != game.SpecialActionShapeshiftersShiftTerrain
}

func hasBonusCard(gs *game.GameState, playerID string, card game.BonusCardType) bool {
	if gs == nil || gs.BonusCards == nil {
		return false
	}
	for _, held := range gs.BonusCards.GetPlayerCards(playerID) {
		if held == card {
			return true
		}
	}
	return false
}

func hasFavor(gs *game.GameState, playerID string, tile game.FavorTileType) bool {
	if gs == nil || gs.FavorTiles == nil {
		return false
	}
	return gs.FavorTiles.HasTileType(playerID, tile)
}

func availableFavorTiles(gs *game.GameState) []game.FavorTileType {
	if gs == nil || gs.FavorTiles == nil {
		return allFavorTiles()
	}
	out := make([]game.FavorTileType, 0, len(gs.FavorTiles.Available))
	for tile, count := range gs.FavorTiles.Available {
		if count > 0 {
			out = append(out, tile)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func availableTownTiles(gs *game.GameState) []models.TownTileType {
	if gs == nil || gs.TownTiles == nil {
		return allTownTiles()
	}
	out := make([]models.TownTileType, 0, len(gs.TownTiles.Available))
	for tile, count := range gs.TownTiles.Available {
		if count > 0 {
			out = append(out, tile)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func availablePassCards(gs *game.GameState, playerID string) []game.BonusCardType {
	if gs == nil || gs.BonusCards == nil {
		return allBonusCards()
	}
	seen := make(map[game.BonusCardType]bool)
	out := make([]game.BonusCardType, 0, len(gs.BonusCards.Available)+1)
	for card := range gs.BonusCards.Available {
		seen[card] = true
		out = append(out, card)
	}
	for _, card := range gs.BonusCards.GetPlayerCards(playerID) {
		if !seen[card] {
			seen[card] = true
			out = append(out, card)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func ownedBuildingHexes(gs *game.GameState, playerID string, buildingType models.BuildingType) []board.Hex {
	var out []board.Hex
	for _, hex := range sortedHexes(gs) {
		mapHex := gs.Map.GetHex(hex)
		if mapHex == nil || mapHex.Building == nil {
			continue
		}
		if mapHex.Building.PlayerID == playerID && mapHex.Building.Type == buildingType {
			out = append(out, hex)
		}
	}
	return out
}

func strategicConversionCandidates(gs *game.GameState, playerID string) []Option {
	var out []Option
	player := gs.GetPlayer(playerID)
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		return out
	}
	maxAmounts := map[game.ConversionType]int{}
	if player.Faction != nil && player.Faction.GetType() == models.FactionTheEnlightened {
		maxAmounts[game.ConversionCoinToPower] = player.Resources.Coins
	}
	addPowerConversionIfLeechCapacityImproves(player, maxAmounts, game.ConversionPowerToCoin, 1)
	addPowerConversionIfLeechCapacityImproves(player, maxAmounts, game.ConversionPowerToWorker, 3)
	addPowerConversionIfLeechCapacityImproves(player, maxAmounts, game.ConversionPowerToPriest, 5)
	for _, conv := range sortedConversions(maxAmounts) {
		for amount := 1; amount <= minInt(8, maxAmounts[conv]); amount++ {
			out = append(out, option(playerID, "conversion", fmt.Sprintf("Convert %s x%d", conv, amount), &game.ConversionAction{
				BaseAction:     game.BaseAction{Type: game.ActionConversion, PlayerID: playerID},
				ConversionType: conv,
				Amount:         amount,
			}, "conversion", string(conv), amount))
		}
	}
	return out
}

func addPowerConversionIfLeechCapacityImproves(player *game.Player, maxAmounts map[game.ConversionType]int, conversion game.ConversionType, powerPerAmount int) {
	if player == nil || player.Resources == nil || player.Resources.Power == nil || powerPerAmount <= 0 {
		return
	}
	maxAmount := player.Resources.Power.Bowl3 / powerPerAmount
	if maxAmount <= 0 {
		return
	}
	before := leechCapacity(player.Resources.Power)
	spentPower := maxAmount * powerPerAmount
	after := before + 2*spentPower
	if after <= before {
		return
	}
	maxAmounts[conversion] = maxAmount
}

func leechCapacity(power *game.PowerSystem) int {
	if power == nil {
		return 0
	}
	return power.Bowl2 + 2*power.Bowl1
}

func sortedConversions(maxAmounts map[game.ConversionType]int) []game.ConversionType {
	conversions := make([]game.ConversionType, 0, len(maxAmounts))
	for conversion, maxAmount := range maxAmounts {
		if maxAmount > 0 {
			conversions = append(conversions, conversion)
		}
	}
	sort.Slice(conversions, func(i, j int) bool {
		return conversions[i] < conversions[j]
	})
	return conversions
}

func option(playerID, typ, label string, action game.Action, parts ...interface{}) Option {
	idParts := make([]string, 0, 2+len(parts))
	idParts = append(idParts, playerID)
	for _, part := range parts {
		idParts = append(idParts, fmt.Sprint(part))
	}
	return Option{
		ID:       strings.Join(idParts, ":"),
		PlayerID: playerID,
		Type:     typ,
		Label:    label,
		Params:   actionParams(action),
		Action:   action,
	}
}

func actionParams(action game.Action) map[string]any {
	if action == nil {
		return nil
	}
	raw, err := json.Marshal(action)
	if err != nil {
		return nil
	}
	params := make(map[string]any)
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil
	}
	normalized, _ := normalizeParamValue(params).(map[string]any)
	delete(normalized, "type")
	delete(normalized, "playerID")
	delete(normalized, "playerId")
	return normalized
}

func normalizeParamValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, child := range typed {
			if key != "" {
				key = strings.ToLower(key[:1]) + key[1:]
			}
			normalized[key] = normalizeParamValue(child)
		}
		return normalized
	case []any:
		normalized := make([]any, len(typed))
		for i, child := range typed {
			normalized[i] = normalizeParamValue(child)
		}
		return normalized
	default:
		return value
	}
}

func sortedPlayerIDs(gs *game.GameState) []string {
	if gs == nil {
		return nil
	}
	ids := make([]string, 0, len(gs.Players))
	for id := range gs.Players {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedHexes(gs *game.GameState) []board.Hex {
	if gs == nil || gs.Map == nil {
		return nil
	}
	hexes := make([]board.Hex, 0, len(gs.Map.Hexes))
	for hex := range gs.Map.Hexes {
		hexes = append(hexes, hex)
	}
	sort.Slice(hexes, func(i, j int) bool {
		if hexes[i].Q != hexes[j].Q {
			return hexes[i].Q < hexes[j].Q
		}
		return hexes[i].R < hexes[j].R
	})
	return hexes
}

func sortedRiverHexes(gs *game.GameState) []board.Hex {
	if gs == nil || gs.Map == nil {
		return nil
	}
	all := sortedHexes(gs)
	rivers := make([]board.Hex, 0)
	for _, hex := range all {
		mh := gs.Map.GetHex(hex)
		if mh != nil && mh.Terrain == models.TerrainRiver {
			rivers = append(rivers, hex)
		}
	}
	return rivers
}

func actionTargetHexes(gs *game.GameState, playerID string) []board.Hex {
	if gs == nil || gs.Map == nil {
		return nil
	}
	all := sortedHexes(gs)
	owned := make([]board.Hex, 0)
	for _, hex := range all {
		mh := gs.Map.GetHex(hex)
		if mh != nil && mh.Building != nil && mh.Building.PlayerID == playerID {
			owned = append(owned, hex)
		}
	}
	if len(owned) == 0 {
		return all
	}
	out := make([]board.Hex, 0)
	for _, hex := range all {
		mh := gs.Map.GetHex(hex)
		if mh == nil || mh.Building != nil {
			continue
		}
		for _, source := range owned {
			if axialDistance(source, hex) <= 4 {
				out = append(out, hex)
				break
			}
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

func bridgePairs(gs *game.GameState, playerID string) [][2]board.Hex {
	hexes := sortedHexes(gs)
	pairs := make([][2]board.Hex, 0)
	for i := range hexes {
		for j := i + 1; j < len(hexes); j++ {
			a, b := hexes[i], hexes[j]
			if axialDistance(a, b) > 2 {
				continue
			}
			if !endpointOwned(gs, playerID, a) && !endpointOwned(gs, playerID, b) {
				continue
			}
			pairs = append(pairs, [2]board.Hex{a, b})
		}
	}
	return pairs
}

func endpointOwned(gs *game.GameState, playerID string, hex board.Hex) bool {
	if gs == nil || gs.Map == nil {
		return false
	}
	mh := gs.Map.GetHex(hex)
	return mh != nil && mh.Building != nil && mh.Building.PlayerID == playerID
}

func townAnchors(pending *game.PendingTownFormation) []board.Hex {
	if pending == nil {
		return nil
	}
	if pending.SkippedRiverHex != nil {
		return []board.Hex{*pending.SkippedRiverHex}
	}
	return append([]board.Hex(nil), pending.Hexes...)
}

func cultTrackSelections(candidates []game.CultTrack, count int) [][]game.CultTrack {
	if count <= 0 || count > len(candidates) {
		return nil
	}
	var out [][]game.CultTrack
	var walk func(int, []game.CultTrack)
	walk = func(start int, chosen []game.CultTrack) {
		if len(chosen) == count {
			out = append(out, append([]game.CultTrack(nil), chosen...))
			return
		}
		for i := start; i < len(candidates); i++ {
			walk(i+1, append(chosen, candidates[i]))
		}
	}
	walk(0, nil)
	return out
}

func trackInts(tracks []game.CultTrack) []interface{} {
	out := make([]interface{}, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, int(track))
	}
	return out
}

func allCultTracks() []game.CultTrack {
	return []game.CultTrack{game.CultFire, game.CultWater, game.CultEarth, game.CultAir}
}

func standardTerrains() []models.TerrainType {
	return []models.TerrainType{models.TerrainPlains, models.TerrainSwamp, models.TerrainLake, models.TerrainForest, models.TerrainMountain, models.TerrainWasteland, models.TerrainDesert}
}

func buildableTerrains() []models.TerrainType {
	return []models.TerrainType{models.TerrainPlains, models.TerrainSwamp, models.TerrainLake, models.TerrainForest, models.TerrainMountain, models.TerrainWasteland, models.TerrainDesert, models.TerrainIce, models.TerrainVolcano}
}

func allBonusCards() []game.BonusCardType {
	return []game.BonusCardType{
		game.BonusCardPriest, game.BonusCardShipping, game.BonusCardDwellingVP, game.BonusCardWorkerPower, game.BonusCardSpade,
		game.BonusCardTradingHouseVP, game.BonusCard6Coins, game.BonusCardCultAdvance, game.BonusCardStrongholdSanctuary, game.BonusCardShippingVP,
	}
}

func allFavorTiles() []game.FavorTileType {
	return []game.FavorTileType{
		game.FavorFire3, game.FavorWater3, game.FavorEarth3, game.FavorAir3,
		game.FavorFire2, game.FavorWater2, game.FavorEarth2, game.FavorAir2,
		game.FavorFire1, game.FavorWater1, game.FavorEarth1, game.FavorAir1,
	}
}

func allTownTiles() []models.TownTileType {
	return []models.TownTileType{
		models.TownTile5Points, models.TownTile6Points, models.TownTile7Points, models.TownTile4Points,
		models.TownTile8Points, models.TownTile9Points, models.TownTile11Points, models.TownTile2Points,
	}
}

func baseFactions() []models.FactionType {
	return []models.FactionType{
		models.FactionNomads, models.FactionFakirs, models.FactionChaosMagicians, models.FactionGiants,
		models.FactionSwarmlings, models.FactionMermaids, models.FactionWitches, models.FactionAuren,
		models.FactionHalflings, models.FactionCultists, models.FactionAlchemists, models.FactionDarklings,
		models.FactionEngineers, models.FactionDwarves,
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func axialDistance(a, b board.Hex) int {
	dq := a.Q - b.Q
	dr := a.R - b.R
	ds := (a.Q + a.R) - (b.Q + b.R)
	return (abs(dq) + abs(dr) + abs(ds)) / 2
}
