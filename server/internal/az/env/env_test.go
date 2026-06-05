package env

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/lukev/tm_server/internal/replay"
)

func TestBuiltInScenarioHasLegalActionsAndApplies(t *testing.T) {
	position, err := BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := position.LegalActions()
	if len(legal) == 0 {
		t.Fatal("expected legal actions")
	}
	next, err := position.Apply(legal[0])
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if next == nil || next.State == nil {
		t.Fatal("expected next position")
	}
	if got := len(next.Encode()); got == 0 {
		t.Fatal("expected non-empty encoding")
	}
}

func TestBaseOrderedMatchupScenariosCoverLegalBasePairs(t *testing.T) {
	scenarios := BaseOrderedMatchupScenarios()
	if len(scenarios) != 168 {
		t.Fatalf("ordered base matchups = %d, want 168", len(scenarios))
	}
	seen := make(map[string]bool, len(scenarios))
	for _, scenario := range scenarios {
		if seen[scenario] {
			t.Fatalf("duplicate matchup scenario: %s", scenario)
		}
		seen[scenario] = true
	}
	if !seen["matchup:Nomads:Witches"] || !seen["matchup:Witches:Nomads"] {
		t.Fatalf("expected both ordered Nomads/Witches matchups in scenario set")
	}
}

func TestSampleScenarioMatchupMetadata(t *testing.T) {
	position, name, err := SampleScenario("matchup:Nomads:Witches", rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("SampleScenario failed: %v", err)
	}
	if name != "matchup:Nomads:Witches" {
		t.Fatalf("scenario name = %s, want matchup:Nomads:Witches", name)
	}
	if position.Metadata.OrderedMatchup != "Nomads_vs_Witches" {
		t.Fatalf("ordered matchup = %s", position.Metadata.OrderedMatchup)
	}
	if position.Metadata.UnorderedMatchup != "Nomads_vs_Witches" {
		t.Fatalf("unordered matchup = %s", position.Metadata.UnorderedMatchup)
	}
	next, err := position.Apply(position.LegalActions()[0])
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if next.Metadata.OrderedMatchup != position.Metadata.OrderedMatchup {
		t.Fatalf("metadata was not preserved across apply: %#v", next.Metadata)
	}
}

func TestObservationIncludesBoardSchema(t *testing.T) {
	position, err := BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	observation := position.Observation()
	if observation.Schema != ObservationSchema {
		t.Fatalf("unexpected schema: %s", observation.Schema)
	}
	if len(observation.Shape) != 3 {
		t.Fatalf("expected 3-part shape, got %#v", observation.Shape)
	}
	if observation.Shape[0] <= 0 || observation.Shape[1] <= 0 || observation.Shape[2] <= 0 {
		t.Fatalf("expected non-empty global and board shape, got %#v", observation.Shape)
	}
	expectedSize := observation.Shape[0] + observation.Shape[1]*observation.Shape[2]
	if len(observation.Features) != expectedSize {
		t.Fatalf("feature length %d does not match shape size %d (%#v)", len(observation.Features), expectedSize, observation.Shape)
	}
	if len(observation.FeatureNames) != len(observation.Features) {
		t.Fatalf("feature name length %d does not match feature length %d", len(observation.FeatureNames), len(observation.Features))
	}
	if len(observation.Features) <= 100 {
		t.Fatalf("expected board-aware encoding, got only %d features", len(observation.Features))
	}
}

func TestSampleScenarioRandomBase(t *testing.T) {
	position, name, err := SampleScenario("random_base", rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("SampleScenario failed: %v", err)
	}
	if name == "" {
		t.Fatal("expected scenario name")
	}
	if position == nil || len(position.LegalActions()) == 0 {
		t.Fatal("expected sampled scenario with legal actions")
	}
}

func TestSampleScenarioTrainingMixIncludesRoundAssets(t *testing.T) {
	position, name, err := SampleScenario("training_mix", rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("SampleScenario failed: %v", err)
	}
	if name == "" {
		t.Fatal("expected scenario name")
	}
	if position == nil || position.State == nil {
		t.Fatal("expected sampled position")
	}
	if len(position.State.ScoringTiles.Tiles) != 6 {
		t.Fatalf("expected six scoring tiles, got %d", len(position.State.ScoringTiles.Tiles))
	}
	heldCards := 0
	for _, playerID := range []string{"p1", "p2"} {
		if _, ok := position.State.BonusCards.GetPlayerCard(playerID); ok {
			heldCards++
		}
	}
	if heldCards != 2 {
		t.Fatalf("expected both players to hold bonus cards, got %d", heldCards)
	}
	if len(position.LegalActions()) == 0 {
		t.Fatal("expected legal actions")
	}
}

func TestSampleScenarioSnapshotsFile(t *testing.T) {
	position, err := BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "seeds.jsonl")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	seed := SnapshotSeed{
		Name:         "midgame_fixture",
		RootPlayerID: position.RootPlayerID,
		Snapshot:     replay.GenerateSnapshot(position.State),
	}
	if err := json.NewEncoder(file).Encode(seed); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	sampled, name, err := SampleScenario("snapshots:"+path, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("SampleScenario failed: %v", err)
	}
	if name != "midgame_fixture" {
		t.Fatalf("unexpected scenario name: %s", name)
	}
	if sampled == nil || sampled.State == nil {
		t.Fatal("expected sampled snapshot position")
	}
	if len(sampled.LegalActions()) == 0 {
		t.Fatal("expected sampled snapshot with legal actions")
	}
}
