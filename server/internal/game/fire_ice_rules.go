package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func isStandardLandTerrain(terrain models.TerrainType) bool {
	return terrain >= models.TerrainPlains && terrain <= models.TerrainDesert
}

func isPermanentFireIceTerrain(terrain models.TerrainType) bool {
	return terrain == models.TerrainIce || terrain == models.TerrainVolcano
}

func isIceFactionType(factionType models.FactionType) bool {
	switch factionType {
	case models.FactionIceMaidens, models.FactionYetis, models.FactionSelkies, models.FactionSnowShamans:
		return true
	default:
		return false
	}
}

func isVolcanoFactionType(factionType models.FactionType) bool {
	switch factionType {
	case models.FactionDragonlords, models.FactionAcolytes, models.FactionFirewalkers:
		return true
	default:
		return false
	}
}

func isRiverwalkers(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionRiverwalkers
}

func isShapeshifters(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionShapeshifters
}

func isSelkies(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionSelkies
}

func isSnowShamans(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionSnowShamans
}

func isDragonlords(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionDragonlords
}

func isAcolytes(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionAcolytes
}

func isFirewalkers(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionFirewalkers
}

func isIceMaidens(player *Player) bool {
	return player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionIceMaidens
}

func iceTerraformCost(player *Player, spades int) factions.Cost {
	if player == nil || player.Faction == nil || spades <= 0 {
		return factions.Cost{}
	}
	return factions.Cost{Workers: player.Faction.GetTerraformCost(spades)}
}

func isAdjacentToRiver(gs *GameState, hex board.Hex) bool {
	if gs == nil || gs.Map == nil {
		return false
	}
	for _, neighbor := range gs.Map.GetDirectNeighbors(hex) {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex != nil && mapHex.Terrain == models.TerrainRiver {
			return true
		}
	}
	return false
}

func riverwalkersCanSettleTerrain(player *Player, terrain models.TerrainType) bool {
	return isRiverwalkers(player) && isStandardLandTerrain(terrain) && player.UnlockedTerrains != nil && player.UnlockedTerrains[terrain]
}

func (gs *GameState) riverwalkersUnlockCost(player *Player, terrain models.TerrainType) int {
	if isOpponentHomeTerrain(gs, player, terrain) {
		return 2
	}
	return 1
}

func effectiveHomeTerrain(player *Player) models.TerrainType {
	if player == nil || player.Faction == nil {
		return models.TerrainTypeUnknown
	}
	if isShapeshifters(player) && player.HasStartingTerrain {
		return player.StartingTerrain
	}
	return player.Faction.GetHomeTerrain()
}

func resolveActionTargetTerrain(player *Player, currentTerrain, requested models.TerrainType) models.TerrainType {
	if requested != models.TerrainTypeUnknown {
		return requested
	}
	if isRiverwalkers(player) {
		return currentTerrain
	}
	return effectiveHomeTerrain(player)
}

func fireIceTerraformDistance(player *Player, from, to models.TerrainType) (int, error) {
	if from == models.TerrainRiver || to == models.TerrainRiver {
		return 0, fmt.Errorf("cannot terraform river terrain")
	}
	if isPermanentFireIceTerrain(from) && from != to {
		return 0, fmt.Errorf("%s terrain cannot be transformed", from)
	}
	if to == models.TerrainIce {
		if from == models.TerrainIce {
			return 0, nil
		}
		if !player.HasStartingTerrain || !isStandardLandTerrain(player.StartingTerrain) {
			return 0, fmt.Errorf("ice faction has no selected starting terrain")
		}
		distance := board.TerrainDistance(from, player.StartingTerrain)
		if distance == 0 {
			return 1, nil
		}
		return distance, nil
	}
	if to == models.TerrainVolcano {
		if from == models.TerrainVolcano {
			return 0, nil
		}
		if !isVolcanoFactionType(player.Faction.GetType()) {
			return 0, fmt.Errorf("only volcano factions may create volcano terrain")
		}
		return 0, nil
	}
	return board.TerrainDistance(from, to), nil
}

func volcanoTransformCost(gs *GameState, player *Player, from models.TerrainType) int {
	if gs == nil || player == nil {
		return 1
	}
	if isOpponentHomeTerrain(gs, player, from) {
		return 2
	}
	return 1
}

func acolytesCultTransformCost(gs *GameState, player *Player, from models.TerrainType) int {
	if isOpponentHomeTerrain(gs, player, from) {
		return 4
	}
	return 3
}

func firewalkersVPTransformCost(gs *GameState, player *Player, from models.TerrainType) int {
	if isOpponentHomeTerrain(gs, player, from) {
		return 6
	}
	return 4
}

func firewalkersAvailableVP(player *Player) int {
	if player == nil {
		return 0
	}
	available := player.VictoryPoints - player.FirewalkersBlockerVP
	if available < 0 {
		return 0
	}
	return available
}

func isOpponentHomeTerrain(gs *GameState, player *Player, terrain models.TerrainType) bool {
	if gs == nil || player == nil || !isStandardLandTerrain(terrain) {
		return false
	}
	for _, other := range gs.Players {
		if other == nil || other.ID == player.ID || other.Faction == nil {
			continue
		}
		if effectiveHomeTerrain(other) == terrain {
			return true
		}
	}
	return false
}

func canSelkiesBuildRiverDwelling(gs *GameState, player *Player, targetHex board.Hex) bool {
	if !isSelkies(player) || gs == nil || gs.Map == nil {
		return false
	}
	iceNeighbors := []board.Hex{}
	for _, neighbor := range gs.Map.GetDirectNeighbors(targetHex) {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex == nil || mapHex.Terrain != models.TerrainIce || mapHex.Building == nil || mapHex.Building.PlayerID != player.ID {
			continue
		}
		iceNeighbors = append(iceNeighbors, neighbor)
	}
	for i := 0; i < len(iceNeighbors); i++ {
		for j := i + 1; j < len(iceNeighbors); j++ {
			if !iceNeighbors[i].IsDirectlyAdjacent(iceNeighbors[j]) {
				return true
			}
		}
	}
	return false
}

func isAdjacentToPlayerBuildingWithExtraShipping(gs *GameState, targetHex board.Hex, playerID string, extraShipping int) bool {
	if gs == nil || gs.Map == nil {
		return false
	}
	if gs.IsAdjacentToPlayerBuilding(targetHex, playerID) {
		return true
	}

	effectiveShipping := gs.effectiveShippingLevel(playerID) + extraShipping
	if effectiveShipping <= 0 {
		return false
	}

	for buildingHex, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			if gs.Map.IsIndirectlyAdjacent(targetHex, buildingHex, effectiveShipping) {
				return true
			}
		}
	}

	return false
}

func factionConvertsSpadeRewards(player *Player) bool {
	return isDragonlords(player) || isAcolytes(player) || isFirewalkers(player)
}

func (gs *GameState) convertFactionSpadeReward(playerID string, spades int, vpEligible bool) {
	if gs == nil || spades <= 0 {
		return
	}
	player := gs.GetPlayer(playerID)
	if player == nil || player.Faction == nil {
		return
	}
	if vpEligible {
		for i := 0; i < spades; i++ {
			gs.AwardActionVP(playerID, ScoringActionSpades)
		}
	}
	switch player.Faction.GetType() {
	case models.FactionDragonlords:
		player.Resources.Power.Bowl1 += spades
	case models.FactionAcolytes:
		for i := 0; i < spades; i++ {
			track := gs.bestAcolytesCultTrackForGain(player)
			gs.AdvanceCultTrack(playerID, track, 1)
		}
	case models.FactionFirewalkers:
		player.FirewalkersBlockerVP -= spades * 4
		if player.FirewalkersBlockerVP < 0 {
			player.FirewalkersBlockerVP = 0
		}
	}
}

func (gs *GameState) bestAcolytesCultTrackForGain(player *Player) CultTrack {
	if player == nil {
		return CultFire
	}
	bestTrack := CultFire
	bestPosition := -1
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		position := player.CultPositions[track]
		if position < 10 && position > bestPosition {
			bestTrack = track
			bestPosition = position
		}
	}
	return bestTrack
}

func (gs *GameState) spendAcolytesCultSteps(playerID string, amount int) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}
	track, ok := gs.acolytesCultPaymentTrack(player, amount)
	if !ok {
		return fmt.Errorf("acolytes need %d cult steps on one track", amount)
	}
	if _, err := gs.DecreaseCultTrack(playerID, track, amount); err != nil {
		return err
	}
	gs.consumeReplayAcolytesCultPaymentTrack(playerID, track)
	return nil
}

func (gs *GameState) acolytesCultPaymentTrack(player *Player, amount int) (CultTrack, bool) {
	if player == nil {
		return CultFire, false
	}
	if gs != nil && gs.ReplayMode != nil && gs.ReplayMode["__replay__"] {
		if track, ok, configured := gs.replayConfiguredAcolytesCultPaymentTrack(player, amount); configured {
			return track, ok
		}
		if track, ok := gs.replayAcolytesCultPaymentTrack(player, amount); ok {
			return track, true
		}
	}
	bestTrack := CultFire
	bestPosition := -1
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		position := player.CultPositions[track]
		if position >= amount && position > bestPosition {
			bestTrack = track
			bestPosition = position
		}
	}
	return bestTrack, bestPosition >= amount
}

func (gs *GameState) replayConfiguredAcolytesCultPaymentTrack(player *Player, amount int) (CultTrack, bool, bool) {
	if gs == nil || player == nil || gs.ReplayAcolytesCultTracks == nil {
		return CultFire, false, false
	}

	queue := gs.ReplayAcolytesCultTracks[player.ID]
	index := gs.ReplayAcolytesCultTrackIndex[player.ID]
	if index >= len(queue) {
		return CultFire, false, false
	}

	track := queue[index]
	if player.CultPositions[track] < amount {
		return track, false, true
	}
	return track, true, true
}

func (gs *GameState) consumeReplayAcolytesCultPaymentTrack(playerID string, track CultTrack) {
	if gs == nil || gs.ReplayAcolytesCultTracks == nil || gs.ReplayAcolytesCultTrackIndex == nil {
		return
	}

	queue := gs.ReplayAcolytesCultTracks[playerID]
	index := gs.ReplayAcolytesCultTrackIndex[playerID]
	if index >= len(queue) || queue[index] != track {
		return
	}

	gs.ReplayAcolytesCultTrackIndex[playerID] = index + 1
}

func (gs *GameState) replayAcolytesCultPaymentTrack(player *Player, amount int) (CultTrack, bool) {
	if player == nil {
		return CultFire, false
	}

	currentRoundCultTrack, hasRoundCultTrack := gs.currentRoundCultRewardTrack()
	bestTrack := CultFire
	bestPosition := 1<<30 - 1
	found := false
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		position := player.CultPositions[track]
		if position < amount {
			continue
		}
		if hasRoundCultTrack && track == currentRoundCultTrack {
			continue
		}
		if position < bestPosition {
			bestTrack = track
			bestPosition = position
			found = true
		}
	}
	if found {
		return bestTrack, true
	}

	bestTrack = CultFire
	bestPosition = 1<<30 - 1
	for _, track := range []CultTrack{CultFire, CultWater, CultEarth, CultAir} {
		position := player.CultPositions[track]
		if position < amount {
			continue
		}
		if position < bestPosition {
			bestTrack = track
			bestPosition = position
			found = true
		}
	}
	return bestTrack, found
}

func (gs *GameState) currentRoundCultRewardTrack() (CultTrack, bool) {
	if gs == nil || gs.ScoringTiles == nil {
		return CultFire, false
	}
	tile := gs.ScoringTiles.GetTileForRound(gs.Round)
	if tile == nil || tile.CultThreshold <= 0 {
		return CultFire, false
	}
	return tile.CultTrack, true
}

func getPowerActionCostForPlayer(player *Player, actionType PowerActionType) int {
	cost := GetPowerCost(actionType)
	if player != nil && player.Faction != nil && player.Faction.GetType() == models.FactionYetis {
		cost--
		if cost < 0 {
			return 0
		}
	}
	return cost
}

func (gs *GameState) grantSnowShamansStrongholdDwellings(playerID string) {
	player := gs.GetPlayer(playerID)
	if !isSnowShamans(player) || gs == nil || gs.Map == nil {
		return
	}

	visited := make(map[board.Hex]bool)
	for hex, mapHex := range gs.Map.Hexes {
		if visited[hex] || mapHex == nil || mapHex.Terrain != models.TerrainIce {
			continue
		}
		component := []board.Hex{}
		queue := []board.Hex{hex}
		visited[hex] = true
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			component = append(component, current)
			for _, neighbor := range gs.Map.GetDirectNeighbors(current) {
				if visited[neighbor] {
					continue
				}
				neighborHex := gs.Map.GetHex(neighbor)
				if neighborHex == nil || neighborHex.Terrain != models.TerrainIce {
					continue
				}
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}

		for _, candidate := range component {
			mapHex := gs.Map.GetHex(candidate)
			if mapHex == nil || mapHex.Building != nil {
				continue
			}
			if err := gs.CheckBuildingLimit(playerID, models.BuildingDwelling); err != nil {
				return
			}
			if err := gs.BuildDwelling(playerID, candidate); err != nil {
				return
			}
			break
		}
	}
}

func (gs *GameState) removePowerTokens(playerID string, count int) error {
	player := gs.GetPlayer(playerID)
	if player == nil || player.Resources == nil || player.Resources.Power == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}
	if count < 0 {
		return fmt.Errorf("cannot remove negative power tokens")
	}
	if player.Resources.Power.TotalPower() < count {
		return fmt.Errorf("not enough power tokens available")
	}
	remaining := count
	remove := remaining
	if player.Resources.Power.Bowl1 < remove {
		remove = player.Resources.Power.Bowl1
	}
	player.Resources.Power.Bowl1 -= remove
	remaining -= remove
	remove = remaining
	if player.Resources.Power.Bowl2 < remove {
		remove = player.Resources.Power.Bowl2
	}
	player.Resources.Power.Bowl2 -= remove
	remaining -= remove
	player.Resources.Power.Bowl3 -= remaining
	return nil
}
