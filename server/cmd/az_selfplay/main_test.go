package main

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestOrderedBaseMatchupsAreCompleteAndLegal(t *testing.T) {
	matchups := orderedBaseMatchups()
	if len(matchups) != 168 {
		t.Fatalf("ordered matchup count = %d, want 168", len(matchups))
	}
	seen := make(map[[2]int]bool, len(matchups))
	for _, matchup := range matchups {
		key := [2]int{int(matchup[0]), int(matchup[1])}
		if seen[key] {
			t.Fatalf("duplicate matchup %v", matchup)
		}
		seen[key] = true
		if matchup[0] == matchup[1] || factions.NewFaction(matchup[0]).GetHomeTerrain() == factions.NewFaction(matchup[1]).GetHomeTerrain() {
			t.Fatalf("illegal same-faction/home matchup %v", matchup)
		}
	}
	for _, matchup := range matchups {
		if !seen[[2]int{int(matchup[1]), int(matchup[0])}] {
			t.Fatalf("matchup %v has no reverse-seat pair", matchup)
		}
	}
}

func TestOrderedBaseMatchupsKeepSmallBatchesFactionDiverse(t *testing.T) {
	matchups := orderedBaseMatchups()
	for start := range matchups {
		firstFactions := make(map[models.FactionType]bool)
		for offset := 0; offset < 8; offset++ {
			firstFactions[matchups[(start+offset)%len(matchups)][0]] = true
		}
		if len(firstFactions) != 8 {
			t.Fatalf("eight-game window at %d has only %d first factions", start, len(firstFactions))
		}
	}
}

func TestGameBatchEndBoundsStreamingBatches(t *testing.T) {
	want := [][2]int{{0, 4}, {4, 8}, {8, 10}}
	got := make([][2]int, 0, len(want))
	for start := 0; start < 10; start += 4 {
		got = append(got, [2]int{start, gameBatchEnd(start, 10, 4)})
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("batch %d = %v, want %v", index, got[index], want[index])
		}
	}
}

func TestSelfPlaySeedsUseIndependentDomains(t *testing.T) {
	setup, search := selfPlaySeeds(1000, 2000, 7)
	if setup != 1007 || search != 2007 {
		t.Fatalf("seeds = (%d,%d), want (1007,2007)", setup, search)
	}
	_, sameSearch := selfPlaySeeds(9000, 2000, 7)
	if sameSearch != search {
		t.Fatalf("changing setup seed changed search seed: got %d, want %d", sameSearch, search)
	}
	sameSetup, _ := selfPlaySeeds(1000, 8000, 7)
	if sameSetup != setup {
		t.Fatalf("changing search seed changed setup seed: got %d, want %d", sameSetup, setup)
	}
}

func TestPositiveSearchSeedRangeCannotCrossZero(t *testing.T) {
	for index := 0; index < 168; index++ {
		_, search := selfPlaySeeds(1, 1, index)
		if search <= 0 {
			t.Fatalf("positive search seed range crossed zero at game %d: %d", index, search)
		}
	}
}
