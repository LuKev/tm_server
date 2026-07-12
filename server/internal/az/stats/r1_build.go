package stats

import (
	"fmt"
	"sort"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

type R1BuildRate struct {
	Samples                  int                `json:"samples"`
	AnyBuildCount            int                `json:"anyBuildCount"`
	TempleOrSanctuaryCount   int                `json:"templeOrSanctuaryCount"`
	StrongholdCount          int                `json:"strongholdCount"`
	TempleOrStrongholdCount  int                `json:"templeOrStrongholdCount"`
	PassedBeforeActionCount  int                `json:"passedBeforeActionCount"`
	ActionsBeforePassTotal   int                `json:"actionsBeforePassTotal"`
	ActionsBeforePassCounts  map[int]int        `json:"actionsBeforePassCounts,omitempty"`
	BuildingCounts           map[string]int     `json:"buildingCounts,omitempty"`
	AverageBuildings         map[string]float64 `json:"averageBuildings,omitempty"`
	AnyBuildRate             float64            `json:"anyBuildRate"`
	TempleOrSanctuaryRate    float64            `json:"templeOrSanctuaryRate"`
	StrongholdRate           float64            `json:"strongholdRate"`
	TempleOrStrongholdRate   float64            `json:"templeOrStrongholdRate"`
	PassedBeforeActionRate   float64            `json:"passedBeforeActionRate"`
	AverageActionsBeforePass float64            `json:"averageActionsBeforePass"`
}

type R1BuildRates map[string]R1BuildRate

type R1BuildSample struct {
	PlayerID          string
	Faction           string
	AnyBuild          bool
	TempleOrSanctuary bool
	Stronghold        bool
	ActionsBeforePass int
	BuildingCounts    map[string]int
}

// R1BuildTracker captures the first point at which a game has left round 1,
// then records whether each tracked player had built a Temple/Sanctuary or SH.
type R1BuildTracker struct {
	playerIDs         []string
	initialBuildings  map[string]string
	currentBuildings  map[string]string
	buildingCounts    map[string]map[string]int
	actionsBeforePass map[string]int
	passed            map[string]bool
	captured          bool
	samples           []R1BuildSample
}

func NewR1BuildTracker(gs *game.GameState, playerIDs []string) *R1BuildTracker {
	ids := append([]string(nil), playerIDs...)
	sort.Strings(ids)
	return &R1BuildTracker{
		playerIDs:         ids,
		initialBuildings:  buildingStates(gs),
		currentBuildings:  buildingStates(gs),
		buildingCounts:    make(map[string]map[string]int),
		actionsBeforePass: make(map[string]int),
		passed:            make(map[string]bool),
	}
}

func (t *R1BuildTracker) ObserveAction(round int, playerID, actionType string) {
	if t == nil || round != 1 || playerID == "" || t.passed[playerID] {
		return
	}
	if actionType == "pass" || actionType == "pass_final" {
		t.passed[playerID] = true
		return
	}
	t.actionsBeforePass[playerID]++
}

func (t *R1BuildTracker) Observe(gs *game.GameState) {
	if t == nil || t.captured || gs == nil {
		return
	}
	if gs.Round <= 1 {
		t.observeBuildingChanges(gs)
		return
	}
	t.captured = true
	t.samples = t.buildSamples(gs)
}

func (t *R1BuildTracker) Finalize(gs *game.GameState) {
	if t == nil || t.captured || gs == nil || gs.Round <= 1 {
		return
	}
	t.captured = true
	t.samples = t.buildSamples(gs)
}

func (t *R1BuildTracker) Samples() []R1BuildSample {
	if t == nil {
		return nil
	}
	return append([]R1BuildSample(nil), t.samples...)
}

func MergeR1BuildRates(dst R1BuildRates, src R1BuildRates) {
	for faction, part := range src {
		total := dst[faction]
		total.Samples += part.Samples
		total.AnyBuildCount += part.AnyBuildCount
		total.TempleOrSanctuaryCount += part.TempleOrSanctuaryCount
		total.StrongholdCount += part.StrongholdCount
		total.TempleOrStrongholdCount += part.TempleOrStrongholdCount
		total.PassedBeforeActionCount += part.PassedBeforeActionCount
		total.ActionsBeforePassTotal += part.ActionsBeforePassTotal
		if total.ActionsBeforePassCounts == nil {
			total.ActionsBeforePassCounts = make(map[int]int)
		}
		for actions, count := range part.ActionsBeforePassCounts {
			total.ActionsBeforePassCounts[actions] += count
		}
		if total.BuildingCounts == nil {
			total.BuildingCounts = make(map[string]int)
		}
		for building, count := range part.BuildingCounts {
			total.BuildingCounts[building] += count
		}
		dst[faction] = total
	}
	FinalizeR1BuildRates(dst)
}

func AddR1BuildSamples(rates R1BuildRates, samples []R1BuildSample) {
	for _, sample := range samples {
		if sample.Faction == "" {
			continue
		}
		entry := rates[sample.Faction]
		entry.Samples++
		if sample.AnyBuild {
			entry.AnyBuildCount++
		}
		if sample.TempleOrSanctuary {
			entry.TempleOrSanctuaryCount++
		}
		if sample.Stronghold {
			entry.StrongholdCount++
		}
		if sample.TempleOrSanctuary || sample.Stronghold {
			entry.TempleOrStrongholdCount++
		}
		if sample.ActionsBeforePass == 0 {
			entry.PassedBeforeActionCount++
		}
		entry.ActionsBeforePassTotal += sample.ActionsBeforePass
		if entry.ActionsBeforePassCounts == nil {
			entry.ActionsBeforePassCounts = make(map[int]int)
		}
		entry.ActionsBeforePassCounts[sample.ActionsBeforePass]++
		if entry.BuildingCounts == nil {
			entry.BuildingCounts = make(map[string]int)
		}
		for building, count := range sample.BuildingCounts {
			entry.BuildingCounts[building] += count
		}
		rates[sample.Faction] = entry
	}
	FinalizeR1BuildRates(rates)
}

func FinalizeR1BuildRates(rates R1BuildRates) {
	for faction, entry := range rates {
		if entry.Samples > 0 {
			entry.AnyBuildRate = float64(entry.AnyBuildCount) / float64(entry.Samples)
			entry.TempleOrSanctuaryRate = float64(entry.TempleOrSanctuaryCount) / float64(entry.Samples)
			entry.StrongholdRate = float64(entry.StrongholdCount) / float64(entry.Samples)
			entry.TempleOrStrongholdRate = float64(entry.TempleOrStrongholdCount) / float64(entry.Samples)
			entry.PassedBeforeActionRate = float64(entry.PassedBeforeActionCount) / float64(entry.Samples)
			entry.AverageActionsBeforePass = float64(entry.ActionsBeforePassTotal) / float64(entry.Samples)
			entry.AverageBuildings = make(map[string]float64, len(entry.BuildingCounts))
			for building, count := range entry.BuildingCounts {
				entry.AverageBuildings[building] = float64(count) / float64(entry.Samples)
			}
		}
		rates[faction] = entry
	}
}

func (t *R1BuildTracker) buildSamples(gs *game.GameState) []R1BuildSample {
	out := make([]R1BuildSample, 0, len(t.playerIDs))
	for _, playerID := range t.playerIDs {
		player := gs.GetPlayer(playerID)
		if player == nil || player.Faction == nil {
			continue
		}
		sample := R1BuildSample{PlayerID: playerID, Faction: player.Faction.GetType().String(), ActionsBeforePass: t.actionsBeforePass[playerID], BuildingCounts: cloneIntMap(t.buildingCounts[playerID])}
		for _, hex := range gs.Map.Hexes {
			if hex == nil || hex.Building == nil || hex.Building.PlayerID != playerID {
				continue
			}
			key := fmt.Sprintf("%d,%d", hex.Coord.Q, hex.Coord.R)
			if t.initialBuildings[key] != buildingState(hex.Building) {
				sample.AnyBuild = true
			}
			switch hex.Building.Type {
			case models.BuildingTemple, models.BuildingSanctuary:
				sample.TempleOrSanctuary = true
			case models.BuildingStronghold:
				sample.Stronghold = true
			}
		}
		out = append(out, sample)
	}
	return out
}

func (t *R1BuildTracker) observeBuildingChanges(gs *game.GameState) {
	if t == nil || gs == nil || gs.Map == nil {
		return
	}
	next := buildingStates(gs)
	for _, hex := range gs.Map.Hexes {
		if hex == nil || hex.Building == nil {
			continue
		}
		key := fmt.Sprintf("%d,%d", hex.Coord.Q, hex.Coord.R)
		if t.currentBuildings[key] == next[key] {
			continue
		}
		counts := t.buildingCounts[hex.Building.PlayerID]
		if counts == nil {
			counts = make(map[string]int)
			t.buildingCounts[hex.Building.PlayerID] = counts
		}
		counts[hex.Building.Type.String()]++
	}
	t.currentBuildings = next
}

func cloneIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func buildingStates(gs *game.GameState) map[string]string {
	states := make(map[string]string)
	if gs == nil || gs.Map == nil {
		return states
	}
	for _, hex := range gs.Map.Hexes {
		if hex != nil && hex.Building != nil {
			states[fmt.Sprintf("%d,%d", hex.Coord.Q, hex.Coord.R)] = buildingState(hex.Building)
		}
	}
	return states
}

func buildingState(building *models.Building) string {
	if building == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", building.PlayerID, building.Type)
}
