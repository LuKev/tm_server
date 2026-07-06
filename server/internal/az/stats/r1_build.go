package stats

import (
	"sort"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

type R1BuildRate struct {
	Samples                 int     `json:"samples"`
	TempleOrSanctuaryCount  int     `json:"templeOrSanctuaryCount"`
	StrongholdCount         int     `json:"strongholdCount"`
	TempleOrStrongholdCount int     `json:"templeOrStrongholdCount"`
	TempleOrSanctuaryRate   float64 `json:"templeOrSanctuaryRate"`
	StrongholdRate          float64 `json:"strongholdRate"`
	TempleOrStrongholdRate  float64 `json:"templeOrStrongholdRate"`
}

type R1BuildRates map[string]R1BuildRate

type R1BuildSample struct {
	PlayerID          string
	Faction           string
	TempleOrSanctuary bool
	Stronghold        bool
}

// R1BuildTracker captures the first point at which a game has left round 1,
// then records whether each tracked player had built a Temple/Sanctuary or SH.
type R1BuildTracker struct {
	playerIDs []string
	captured  bool
	samples   []R1BuildSample
}

func NewR1BuildTracker(playerIDs []string) *R1BuildTracker {
	ids := append([]string(nil), playerIDs...)
	sort.Strings(ids)
	return &R1BuildTracker{playerIDs: ids}
}

func (t *R1BuildTracker) Observe(gs *game.GameState) {
	if t == nil || t.captured || gs == nil || gs.Round <= 1 {
		return
	}
	t.captured = true
	t.samples = buildSamples(gs, t.playerIDs)
}

func (t *R1BuildTracker) Finalize(gs *game.GameState) {
	if t == nil || t.captured || gs == nil || gs.Round <= 1 {
		return
	}
	t.captured = true
	t.samples = buildSamples(gs, t.playerIDs)
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
		total.TempleOrSanctuaryCount += part.TempleOrSanctuaryCount
		total.StrongholdCount += part.StrongholdCount
		total.TempleOrStrongholdCount += part.TempleOrStrongholdCount
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
		if sample.TempleOrSanctuary {
			entry.TempleOrSanctuaryCount++
		}
		if sample.Stronghold {
			entry.StrongholdCount++
		}
		if sample.TempleOrSanctuary || sample.Stronghold {
			entry.TempleOrStrongholdCount++
		}
		rates[sample.Faction] = entry
	}
	FinalizeR1BuildRates(rates)
}

func FinalizeR1BuildRates(rates R1BuildRates) {
	for faction, entry := range rates {
		if entry.Samples > 0 {
			entry.TempleOrSanctuaryRate = float64(entry.TempleOrSanctuaryCount) / float64(entry.Samples)
			entry.StrongholdRate = float64(entry.StrongholdCount) / float64(entry.Samples)
			entry.TempleOrStrongholdRate = float64(entry.TempleOrStrongholdCount) / float64(entry.Samples)
		}
		rates[faction] = entry
	}
}

func buildSamples(gs *game.GameState, playerIDs []string) []R1BuildSample {
	out := make([]R1BuildSample, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		player := gs.GetPlayer(playerID)
		if player == nil || player.Faction == nil {
			continue
		}
		sample := R1BuildSample{PlayerID: playerID, Faction: player.Faction.GetType().String()}
		for _, hex := range gs.Map.Hexes {
			if hex == nil || hex.Building == nil || hex.Building.PlayerID != playerID {
				continue
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
