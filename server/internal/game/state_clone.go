package game

import (
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// CloneForUndo creates a deep copy of the current game state that is safe to
// restore later when an undo is requested.
func (gs *GameState) CloneForUndo() *GameState {
	if gs == nil {
		return nil
	}

	clone := &GameState{
		Round:                           gs.Round,
		Phase:                           gs.Phase,
		SetupMode:                       gs.SetupMode,
		SetupSubphase:                   gs.SetupSubphase,
		SetupDwellingIndex:              gs.SetupDwellingIndex,
		SetupBonusIndex:                 gs.SetupBonusIndex,
		TurnOrderPolicy:                 gs.TurnOrderPolicy,
		CurrentPlayerIndex:              gs.CurrentPlayerIndex,
		PassOrder:                       append([]string(nil), gs.PassOrder...),
		SetupDwellingOrder:              append([]string(nil), gs.SetupDwellingOrder...),
		SetupBonusOrder:                 append([]string(nil), gs.SetupBonusOrder...),
		TurnOrder:                       append([]string(nil), gs.TurnOrder...),
		SetupPlacedDwellings:            cloneStringIntMap(gs.SetupPlacedDwellings),
		PendingSpades:                   cloneStringIntMap(gs.PendingSpades),
		PendingSpadeBuildAllowed:        cloneStringBoolMap(gs.PendingSpadeBuildAllowed),
		PendingCultRewardSpades:         cloneStringIntMap(gs.PendingCultRewardSpades),
		NextLeechEventID:                gs.NextLeechEventID,
		PendingFreeActionsPlayerID:      gs.PendingFreeActionsPlayerID,
		PendingCultistsLeech:            clonePendingCultistsLeech(gs.PendingCultistsLeech),
		SkipAbilityUsedThisAction:       cloneSkipAbilityUsedThisAction(gs.SkipAbilityUsedThisAction),
		PendingWispsTradingPostSpade:    cloneHexMap(gs.PendingWispsTradingPostSpade),
		ReplayMode:                      cloneStringBoolMap(gs.ReplayMode),
		SuppressTurnAdvance:             gs.SuppressTurnAdvance,
		PendingTurnConfirmationPlayerID: "",
	}

	clone.Map = cloneMap(gs.Map)
	clone.Players = clonePlayers(gs.Players)
	clone.AuctionState = cloneAuctionState(gs.AuctionState)
	clone.PowerActions = clonePowerActionState(gs.PowerActions)
	clone.CultTracks = cloneCultTrackState(gs.CultTracks)
	clone.FavorTiles = cloneFavorTileState(gs.FavorTiles)
	clone.BonusCards = cloneBonusCardState(gs.BonusCards)
	clone.TownTiles = cloneTownTileState(gs.TownTiles)
	clone.ScoringTiles = cloneScoringTileState(gs.ScoringTiles)
	clone.PendingLeechOffers = clonePendingLeechOffers(gs.PendingLeechOffers)
	clone.PendingTownFormations = clonePendingTownFormations(gs.PendingTownFormations)
	clone.PendingCultistsLeech = clonePendingCultistsLeech(gs.PendingCultistsLeech)
	clone.PendingFavorTileSelection = clonePendingFavorTileSelection(gs.PendingFavorTileSelection)
	clone.PendingHalflingsSpades = clonePendingHalflingsSpades(gs.PendingHalflingsSpades)
	clone.PendingWispsStrongholdDwelling = clonePendingWispsStrongholdDwelling(gs.PendingWispsStrongholdDwelling)
	clone.PendingDarklingsPriestOrdination = clonePendingDarklingsPriestOrdination(gs.PendingDarklingsPriestOrdination)
	clone.PendingCultistsCultSelection = clonePendingCultistsCultSelection(gs.PendingCultistsCultSelection)
	clone.PendingTownCultTopChoice = clonePendingTownCultTopChoice(gs.PendingTownCultTopChoice)
	clone.FinalScoring = cloneFinalScoring(gs.FinalScoring)
	clone.TurnTimer = cloneTurnTimerState(gs.TurnTimer)

	if gs.RiverTownHex != nil {
		hex := *gs.RiverTownHex
		clone.RiverTownHex = &hex
	}

	// Confirmation snapshots are intentionally not carried over so restored
	// states never recurse into older undo windows.
	clone.PendingTurnConfirmationSnapshot = nil

	return clone
}

func clonePlayers(src map[string]*Player) map[string]*Player {
	if src == nil {
		return nil
	}
	dst := make(map[string]*Player, len(src))
	for id, player := range src {
		dst[id] = clonePlayer(player)
	}
	return dst
}

func clonePlayer(src *Player) *Player {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Resources != nil {
		dst.Resources = src.Resources.Clone()
	}
	if src.CultPositions != nil {
		dst.CultPositions = make(map[CultTrack]int, len(src.CultPositions))
		for track, pos := range src.CultPositions {
			dst.CultPositions[track] = pos
		}
	}
	if src.SpecialActionsUsed != nil {
		dst.SpecialActionsUsed = make(map[SpecialActionType]bool, len(src.SpecialActionsUsed))
		for actionType, used := range src.SpecialActionsUsed {
			dst.SpecialActionsUsed[actionType] = used
		}
	}
	if src.TownTiles != nil {
		dst.TownTiles = append([]models.TownTileType(nil), src.TownTiles...)
	}
	if src.AtlanteansTownHexes != nil {
		dst.AtlanteansTownHexes = append([]board.Hex(nil), src.AtlanteansTownHexes...)
	}
	if src.AtlanteansTownRewards != nil {
		dst.AtlanteansTownRewards = make(map[int]bool, len(src.AtlanteansTownRewards))
		for threshold, claimed := range src.AtlanteansTownRewards {
			dst.AtlanteansTownRewards[threshold] = claimed
		}
	}
	return &dst
}

func cloneHexMap(src map[string]board.Hex) map[string]board.Hex {
	if src == nil {
		return nil
	}
	dst := make(map[string]board.Hex, len(src))
	for playerID, hex := range src {
		dst[playerID] = hex
	}
	return dst
}

func cloneMap(src *board.TerraMysticaMap) *board.TerraMysticaMap {
	if src == nil {
		return nil
	}
	dst := &board.TerraMysticaMap{
		Hexes:      make(map[board.Hex]*board.MapHex, len(src.Hexes)),
		Bridges:    make(map[board.BridgeKey]string, len(src.Bridges)),
		RiverHexes: make(map[board.Hex]bool, len(src.RiverHexes)),
	}
	for coord, hex := range src.Hexes {
		if hex == nil {
			dst.Hexes[coord] = nil
			continue
		}
		hexClone := *hex
		if hex.Building != nil {
			building := *hex.Building
			hexClone.Building = &building
		}
		dst.Hexes[coord] = &hexClone
	}
	for key, owner := range src.Bridges {
		dst.Bridges[key] = owner
	}
	for coord, isRiver := range src.RiverHexes {
		dst.RiverHexes[coord] = isRiver
	}
	return dst
}

func cloneAuctionState(src *AuctionState) *AuctionState {
	if src == nil {
		return nil
	}
	dst := *src
	dst.NominationOrder = append([]models.FactionType(nil), src.NominationOrder...)
	dst.SeatOrder = append([]string(nil), src.SeatOrder...)
	dst.CurrentBids = make(map[models.FactionType]int, len(src.CurrentBids))
	for faction, bid := range src.CurrentBids {
		dst.CurrentBids[faction] = bid
	}
	dst.FactionHolders = make(map[models.FactionType]string, len(src.FactionHolders))
	for faction, holder := range src.FactionHolders {
		dst.FactionHolders[faction] = holder
	}
	dst.PlayerHasFaction = make(map[string]bool, len(src.PlayerHasFaction))
	for playerID, hasFaction := range src.PlayerHasFaction {
		dst.PlayerHasFaction[playerID] = hasFaction
	}
	dst.FastBids = make(map[string]map[models.FactionType]int, len(src.FastBids))
	for playerID, bids := range src.FastBids {
		playerBids := make(map[models.FactionType]int, len(bids))
		for faction, bid := range bids {
			playerBids[faction] = bid
		}
		dst.FastBids[playerID] = playerBids
	}
	dst.FastSubmitted = make(map[string]bool, len(src.FastSubmitted))
	for playerID, submitted := range src.FastSubmitted {
		dst.FastSubmitted[playerID] = submitted
	}
	return &dst
}

func cloneTurnTimerState(src *TurnTimerState) *TurnTimerState {
	if src == nil {
		return nil
	}
	dst := &TurnTimerState{
		Config:  src.Config,
		Players: make(map[string]*PlayerTurnTimer, len(src.Players)),
	}
	for playerID, timer := range src.Players {
		if timer == nil {
			dst.Players[playerID] = nil
			continue
		}
		timerCopy := *timer
		dst.Players[playerID] = &timerCopy
	}
	return dst
}

func clonePowerActionState(src *PowerActionState) *PowerActionState {
	if src == nil {
		return nil
	}
	dst := &PowerActionState{
		UsedActions: make(map[PowerActionType]bool, len(src.UsedActions)),
	}
	for actionType, used := range src.UsedActions {
		dst.UsedActions[actionType] = used
	}
	return dst
}

func cloneCultTrackState(src *CultTrackState) *CultTrackState {
	if src == nil {
		return nil
	}
	dst := &CultTrackState{
		PlayerPositions:       make(map[string]map[CultTrack]int, len(src.PlayerPositions)),
		Position10Occupied:    make(map[CultTrack]string, len(src.Position10Occupied)),
		BonusPositionsClaimed: make(map[string]map[CultTrack]map[int]bool, len(src.BonusPositionsClaimed)),
		PriestsOnActionSpaces: make(map[string]map[CultTrack]int, len(src.PriestsOnActionSpaces)),
		PriestsOnTrack:        make(map[CultTrack]map[int][]string, len(src.PriestsOnTrack)),
	}
	for playerID, positions := range src.PlayerPositions {
		playerPositions := make(map[CultTrack]int, len(positions))
		for track, pos := range positions {
			playerPositions[track] = pos
		}
		dst.PlayerPositions[playerID] = playerPositions
	}
	for track, playerID := range src.Position10Occupied {
		dst.Position10Occupied[track] = playerID
	}
	for playerID, claimedByTrack := range src.BonusPositionsClaimed {
		playerClaimed := make(map[CultTrack]map[int]bool, len(claimedByTrack))
		for track, positions := range claimedByTrack {
			trackClaimed := make(map[int]bool, len(positions))
			for pos, claimed := range positions {
				trackClaimed[pos] = claimed
			}
			playerClaimed[track] = trackClaimed
		}
		dst.BonusPositionsClaimed[playerID] = playerClaimed
	}
	for playerID, trackCounts := range src.PriestsOnActionSpaces {
		playerCounts := make(map[CultTrack]int, len(trackCounts))
		for track, count := range trackCounts {
			playerCounts[track] = count
		}
		dst.PriestsOnActionSpaces[playerID] = playerCounts
	}
	for track, spots := range src.PriestsOnTrack {
		trackSpots := make(map[int][]string, len(spots))
		for spot, playerIDs := range spots {
			trackSpots[spot] = append([]string(nil), playerIDs...)
		}
		dst.PriestsOnTrack[track] = trackSpots
	}
	return dst
}

func cloneFavorTileState(src *FavorTileState) *FavorTileState {
	if src == nil {
		return nil
	}
	dst := &FavorTileState{
		Available:   make(map[FavorTileType]int, len(src.Available)),
		PlayerTiles: make(map[string][]FavorTileType, len(src.PlayerTiles)),
	}
	for tileType, count := range src.Available {
		dst.Available[tileType] = count
	}
	for playerID, tiles := range src.PlayerTiles {
		dst.PlayerTiles[playerID] = append([]FavorTileType(nil), tiles...)
	}
	return dst
}

func cloneBonusCardState(src *BonusCardState) *BonusCardState {
	if src == nil {
		return nil
	}
	dst := &BonusCardState{
		Available:     make(map[BonusCardType]int, len(src.Available)),
		PlayerCards:   make(map[string]BonusCardType, len(src.PlayerCards)),
		PlayerHasCard: make(map[string]bool, len(src.PlayerHasCard)),
	}
	for cardType, count := range src.Available {
		dst.Available[cardType] = count
	}
	for playerID, cardType := range src.PlayerCards {
		dst.PlayerCards[playerID] = cardType
	}
	for playerID, hasCard := range src.PlayerHasCard {
		dst.PlayerHasCard[playerID] = hasCard
	}
	return dst
}

func cloneTownTileState(src *TownTileState) *TownTileState {
	if src == nil {
		return nil
	}
	dst := &TownTileState{
		Available: make(map[models.TownTileType]int, len(src.Available)),
	}
	for tileType, count := range src.Available {
		dst.Available[tileType] = count
	}
	return dst
}

func cloneScoringTileState(src *ScoringTileState) *ScoringTileState {
	if src == nil {
		return nil
	}
	dst := &ScoringTileState{
		Tiles:       append([]ScoringTile(nil), src.Tiles...),
		PriestsSent: make(map[string]int, len(src.PriestsSent)),
	}
	for playerID, count := range src.PriestsSent {
		dst.PriestsSent[playerID] = count
	}
	return dst
}

func clonePendingLeechOffers(src map[string][]*PowerLeechOffer) map[string][]*PowerLeechOffer {
	if src == nil {
		return nil
	}
	dst := make(map[string][]*PowerLeechOffer, len(src))
	for playerID, offers := range src {
		clonedOffers := make([]*PowerLeechOffer, 0, len(offers))
		for _, offer := range offers {
			if offer == nil {
				clonedOffers = append(clonedOffers, nil)
				continue
			}
			offerClone := *offer
			clonedOffers = append(clonedOffers, &offerClone)
		}
		dst[playerID] = clonedOffers
	}
	return dst
}

func clonePendingTownFormations(src map[string][]*PendingTownFormation) map[string][]*PendingTownFormation {
	if src == nil {
		return nil
	}
	dst := make(map[string][]*PendingTownFormation, len(src))
	for playerID, formations := range src {
		clonedFormations := make([]*PendingTownFormation, 0, len(formations))
		for _, formation := range formations {
			if formation == nil {
				clonedFormations = append(clonedFormations, nil)
				continue
			}
			formationClone := *formation
			formationClone.Hexes = append([]board.Hex(nil), formation.Hexes...)
			if formation.SkippedRiverHex != nil {
				hex := *formation.SkippedRiverHex
				formationClone.SkippedRiverHex = &hex
			}
			clonedFormations = append(clonedFormations, &formationClone)
		}
		dst[playerID] = clonedFormations
	}
	return dst
}

func clonePendingCultistsLeech(src map[int]*CultistsLeechBonus) map[int]*CultistsLeechBonus {
	if src == nil {
		return nil
	}
	dst := make(map[int]*CultistsLeechBonus, len(src))
	for eventID, bonus := range src {
		if bonus == nil {
			dst[eventID] = nil
			continue
		}
		bonusClone := *bonus
		dst[eventID] = &bonusClone
	}
	return dst
}

func clonePendingFavorTileSelection(src *PendingFavorTileSelection) *PendingFavorTileSelection {
	if src == nil {
		return nil
	}
	dst := *src
	dst.SelectedTiles = append([]FavorTileType(nil), src.SelectedTiles...)
	return &dst
}

func clonePendingHalflingsSpades(src *PendingHalflingsSpades) *PendingHalflingsSpades {
	if src == nil {
		return nil
	}
	dst := *src
	dst.TransformedHexes = append([]board.Hex(nil), src.TransformedHexes...)
	return &dst
}

func clonePendingWispsStrongholdDwelling(src *PendingWispsStrongholdDwelling) *PendingWispsStrongholdDwelling {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func clonePendingDarklingsPriestOrdination(src *PendingDarklingsPriestOrdination) *PendingDarklingsPriestOrdination {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func clonePendingCultistsCultSelection(src *PendingCultistsCultSelection) *PendingCultistsCultSelection {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func clonePendingTownCultTopChoice(src *PendingTownCultTopChoice) *PendingTownCultTopChoice {
	if src == nil {
		return nil
	}
	dst := *src
	dst.CandidateTracks = append([]CultTrack(nil), src.CandidateTracks...)
	return &dst
}

func cloneFinalScoring(src map[string]*PlayerFinalScore) map[string]*PlayerFinalScore {
	if src == nil {
		return nil
	}
	dst := make(map[string]*PlayerFinalScore, len(src))
	for playerID, score := range src {
		if score == nil {
			dst[playerID] = nil
			continue
		}
		scoreClone := *score
		dst[playerID] = &scoreClone
	}
	return dst
}

func cloneStringIntMap(src map[string]int) map[string]int {
	if src == nil {
		return nil
	}
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneStringBoolMap(src map[string]bool) map[string]bool {
	if src == nil {
		return nil
	}
	dst := make(map[string]bool, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneSkipAbilityUsedThisAction(src map[string][]board.Hex) map[string][]board.Hex {
	if src == nil {
		return nil
	}
	dst := make(map[string][]board.Hex, len(src))
	for playerID, hexes := range src {
		dst[playerID] = append([]board.Hex(nil), hexes...)
	}
	return dst
}
