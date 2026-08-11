package az

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"sort"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestPositionDeterministicAndPlayerIDRelativeHash(t *testing.T) {
	first, err := newBaseGame(17, models.FactionWitches, models.FactionEngineers, "p0", "p1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := newBaseGame(17, models.FactionWitches, models.FactionEngineers, "alice", "bob")
	if err != nil {
		t.Fatal(err)
	}
	if first.CanonicalHash() != second.CanonicalHash() {
		t.Fatalf("renaming internal player IDs changed canonical hash: %s != %s", first.CanonicalHash(), second.CanonicalHash())
	}
	firstJSON, err := first.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := second.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("renaming internal player IDs changed canonical encoder input")
	}
	repeat, err := NewBaseGame(17, models.FactionWitches, models.FactionEngineers)
	if err != nil {
		t.Fatal(err)
	}
	if first.CanonicalHash() != repeat.CanonicalHash() {
		t.Fatal("same seed did not reproduce setup")
	}
}

func TestCanonicalCacheIsImmutableAndInvalidatedByApply(t *testing.T) {
	position := forcedActionPosition(t, 1701, models.FactionWitches, models.FactionEngineers)
	callerJSON, err := position.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	wantJSON := append([]byte(nil), callerJSON...)
	wantHash := position.CanonicalHash()
	if !position.canonicalReady {
		t.Fatal("canonical record was not cached")
	}

	callerJSON[0] ^= 0xff
	gotJSON, err := position.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotJSON, wantJSON) || position.CanonicalHash() != wantHash {
		t.Fatal("mutating caller-owned canonical bytes poisoned the cache")
	}

	clone := position.Clone().(*GamePosition)
	if clone.CanonicalHash() != wantHash || !clone.canonicalReady {
		t.Fatal("clone did not preserve the immutable canonical cache")
	}
	actions := position.LegalActions()
	if len(actions) == 0 {
		t.Fatal("fixture has no legal actions")
	}
	if err := position.Apply(representativeBenchmarkAction(actions)); err != nil {
		t.Fatal(err)
	}
	if position.canonicalReady || position.canonicalRecord != nil || position.cachedHash != (Hash128{}) {
		t.Fatal("Apply did not invalidate the canonical cache")
	}
	if position.CanonicalHash() == wantHash {
		t.Fatal("applied successor retained the parent canonical hash")
	}
	if clone.CanonicalHash() != wantHash {
		t.Fatal("applying to one clone mutated another clone's canonical cache")
	}
}

func TestCanonicalHashDoesNotRewriteUnrelatedStringsMatchingPlayerIDs(t *testing.T) {
	want, err := newBaseGame(18, models.FactionWitches, models.FactionEngineers, "p0", "p1")
	if err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][2]string{{"0", "1"}, {"pass_order", "base"}} {
		got, err := newBaseGame(18, models.FactionWitches, models.FactionEngineers, ids[0], ids[1])
		if err != nil {
			t.Fatal(err)
		}
		if got.CanonicalHash() != want.CanonicalHash() {
			t.Fatalf("IDs %q/%q changed canonical hash", ids[0], ids[1])
		}
	}
}

func TestCanonicalHashIncludesPrivateFactionRuleState(t *testing.T) {
	base, err := NewBaseGame(19, models.FactionFakirs, models.FactionEngineers)
	if err != nil {
		t.Fatal(err)
	}
	state := base.StateClone()
	fakir, ok := state.GetPlayer("p0").Faction.(*factions.Fakirs)
	if !ok {
		t.Fatal("expected Fakirs implementation")
	}
	fakir.IncrementFlightRange()
	changed, err := NewPosition(state)
	if err != nil {
		t.Fatal(err)
	}
	if changed.CanonicalHash() == base.CanonicalHash() {
		t.Fatal("Fakirs flight range did not change canonical hash")
	}
}

func TestCanonicalHashIgnoresOrderWithinSemanticSets(t *testing.T) {
	base := forcedActionPosition(t, 20, models.FactionWitches, models.FactionEngineers).StateClone()
	firstHex := ownedBuildingHex(t, base, "p0")
	secondHex := placeOwnedBuilding(t, base, "p0")
	base.PendingTownFormations["p0"] = []*game.PendingTownFormation{{
		PlayerID: "p0", Hexes: []board.Hex{firstHex, secondHex},
	}}
	base.PendingTownCultTopChoice = &game.PendingTownCultTopChoice{
		PlayerID: "p0", AdvanceAmount: 2,
		CandidateTracks: []game.CultTrack{game.CultFire, game.CultWater, game.CultAir},
		MaxSelections:   2,
	}
	base.FavorTiles.PlayerTiles["p0"] = []game.FavorTileType{game.FavorWater1, game.FavorFire1}
	base.GetPlayer("p0").TownTiles = []models.TownTileType{models.TownTile9Points, models.TownTile5Points}
	if base.CultTracks.PriestsOnTrack[game.CultFire] == nil {
		base.CultTracks.PriestsOnTrack[game.CultFire] = make(map[int][]string)
	}
	base.CultTracks.PriestsOnTrack[game.CultFire][2] = []string{"p1", "p0"}

	first, err := NewPosition(base)
	if err != nil {
		t.Fatal(err)
	}
	reordered := base.CloneForUndo()
	reordered.PendingTownFormations["p0"][0].Hexes = []board.Hex{secondHex, firstHex}
	reordered.PendingTownCultTopChoice.CandidateTracks = []game.CultTrack{game.CultAir, game.CultFire, game.CultWater}
	reordered.FavorTiles.PlayerTiles["p0"] = []game.FavorTileType{game.FavorFire1, game.FavorWater1}
	reordered.GetPlayer("p0").TownTiles = []models.TownTileType{models.TownTile5Points, models.TownTile9Points}
	reordered.CultTracks.PriestsOnTrack[game.CultFire][2] = []string{"p0", "p1"}
	second, err := NewPosition(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if first.CanonicalHash() != second.CanonicalHash() {
		t.Fatal("ordering a semantic set changed canonical hash")
	}
	assertActionSetsEqual(t, first.LegalActions(), second.LegalActions())

	halflings := forcedActionPosition(t, 21, models.FactionHalflings, models.FactionWitches).StateClone()
	halflings.PendingHalflingsSpades = &game.PendingHalflingsSpades{
		PlayerID: "p0", SpadesRemaining: 1, TransformedHexes: []board.Hex{firstHex, secondHex},
	}
	halflingsReordered := halflings.CloneForUndo()
	halflingsReordered.PendingHalflingsSpades.TransformedHexes = []board.Hex{secondHex, firstHex}
	assertCanonicalHashesEqual(t, halflings, halflingsReordered)

	chaos := forcedActionPosition(t, 22, models.FactionChaosMagicians, models.FactionEngineers).StateClone()
	chaos.PendingFavorTileSelection = &game.PendingFavorTileSelection{
		PlayerID: "p0", Count: 1, SelectedTiles: []game.FavorTileType{game.FavorWater1, game.FavorFire1},
	}
	chaosReordered := chaos.CloneForUndo()
	chaosReordered.PendingFavorTileSelection.SelectedTiles = []game.FavorTileType{game.FavorFire1, game.FavorWater1}
	assertCanonicalHashesEqual(t, chaos, chaosReordered)
}

func assertCanonicalHashesEqual(t *testing.T, firstState, secondState *game.GameState) {
	t.Helper()
	first, err := NewPosition(firstState)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewPosition(secondState)
	if err != nil {
		t.Fatal(err)
	}
	if first.CanonicalHash() != second.CanonicalHash() {
		t.Fatal("ordering a semantic set changed canonical hash")
	}
}

func TestLegalOracleMatchesProductionAndActionsApply(t *testing.T) {
	position, err := NewBaseGame(9, models.FactionWitches, models.FactionEngineers)
	if err != nil {
		t.Fatal(err)
	}
	assertActionSetsEqual(t, position.LegalActions(), oracleLegalActions(position))
	for _, action := range position.LegalActions() {
		clone := position.Clone()
		if err := clone.Apply(action); err != nil {
			t.Fatalf("emitted action does not apply: %s: %v", action.Key(), err)
		}
		raw, err := json.Marshal(action)
		if err != nil {
			t.Fatal(err)
		}
		var decoded SearchAction
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		if action.Key() != decoded.Key() {
			t.Fatalf("action schema failed round trip: %s != %s", action.Key(), decoded.Key())
		}
		engineAction, err := action.GameAction(string(position.DecisionPlayer()))
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := SearchActionFromGameAction(engineAction)
		if err != nil {
			t.Fatal(err)
		}
		if action.Key() != roundTrip.Key() {
			t.Fatalf("game action round trip failed: %s != %s", action.Key(), roundTrip.Key())
		}
	}
}

func TestSeededRandomGameCompletesNaturally(t *testing.T) {
	position, err := NewBaseGame(42, models.FactionWitches, models.FactionEngineers)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(42))
	const tripwire = 700
	for ply := 0; ply < tripwire && !position.IsTerminal(); ply++ {
		actions := position.LegalActions()
		if len(actions) == 0 {
			t.Fatalf("non-terminal state has no legal actions at ply %d, phase=%v round=%d actor=%q hash=%s", ply, position.state.Phase, position.state.Round, position.DecisionPlayer(), position.CanonicalHash())
		}
		// Check the independent oracle at phase boundaries and periodically; the
		// exhaustive pass is deliberately too slow to run at every ply.
		if ply < 5 || ply%50 == 0 {
			assertActionSetsEqual(t, actions, oracleLegalActions(position))
		}
		if err := position.Apply(actions[rng.Intn(len(actions))]); err != nil {
			t.Fatalf("apply failed at ply %d: %v", ply, err)
		}
	}
	if !position.IsTerminal() {
		t.Fatalf("game hit %d-ply tripwire at round=%d phase=%v hash=%s", tripwire, position.state.Round, position.state.Phase, position.CanonicalHash())
	}
	if got := position.Outcome("p0"); got < -1 || got > 1 {
		t.Fatalf("invalid outcome %v", got)
	}
	if _, ok := position.FinalVP("p0"); !ok {
		t.Fatal("missing final VP for p0")
	}
	if _, ok := position.FinalVP("p1"); !ok {
		t.Fatal("missing final VP for p1")
	}
}

func TestActionLogReplayReproducesHashesAndTerminalScore(t *testing.T) {
	original, err := NewBaseGame(73, models.FactionNomads, models.FactionAuren)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(73))
	actions := make([]SearchAction, 0, 256)
	hashes := make([]Hash128, 0, 256)
	records := make([][]byte, 0, 256)
	for ply := 0; ply < 700 && !original.IsTerminal(); ply++ {
		legal := original.LegalActions()
		if len(legal) == 0 {
			t.Fatalf("original game stuck at ply %d", ply)
		}
		choice := legal[rng.Intn(len(legal))]
		actions = append(actions, choice)
		if err := original.Apply(choice); err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, original.CanonicalHash())
		record, _ := original.CanonicalJSON()
		records = append(records, record)
	}
	if !original.IsTerminal() {
		t.Fatal("original game did not finish")
	}

	replayed, err := NewBaseGame(73, models.FactionNomads, models.FactionAuren)
	if err != nil {
		t.Fatal(err)
	}
	for i, action := range actions {
		if err := replayed.Apply(action); err != nil {
			t.Fatalf("replay action %d failed: %v", i, err)
		}
		if got := replayed.CanonicalHash(); got != hashes[i] {
			gotJSON, _ := replayed.CanonicalJSON()
			at := firstDifferentByte(gotJSON, records[i])
			start := at - 80
			if start < 0 {
				start = 0
			}
			end := at + 160
			if end > len(gotJSON) {
				end = len(gotJSON)
			}
			wantEnd := end
			if wantEnd > len(records[i]) {
				wantEnd = len(records[i])
			}
			t.Fatalf("replay diverged at action %d (%s): got %s want %s, canonical byte=%d\ngot=%s\nwant=%s", i, action.Key(), got, hashes[i], at, gotJSON[start:end], records[i][start:wantEnd])
		}
		if i%50 == 0 && !replayed.IsTerminal() {
			assertActionSetsEqual(t, replayed.LegalActions(), oracleLegalActions(replayed))
		}
	}
	for _, player := range []PlayerID{"p0", "p1"} {
		got, gotOK := replayed.FinalVP(player)
		want, wantOK := original.FinalVP(player)
		if !gotOK || !wantOK || got != want {
			t.Fatalf("terminal score mismatch for %s: got %d/%v want %d/%v", player, got, gotOK, want, wantOK)
		}
	}
}

func firstDifferentByte(a, b []byte) int {
	if bytes.Equal(a, b) {
		return -1
	}
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return limit
}

func TestSearchProfileRejectsReplayAndConfirmation(t *testing.T) {
	position, err := NewBaseGame(1, models.FactionWitches, models.FactionEngineers)
	if err != nil {
		t.Fatal(err)
	}
	state := position.StateClone()
	state.ReplayMode = make(map[string]bool)
	state.ReplayMode["p0"] = true
	if _, err := NewPosition(state); err == nil {
		t.Fatal("expected replay mode to be rejected")
	}
	state.ReplayMode["p0"] = false
	state.BeginPendingTurnConfirmation("p0", state.CloneForUndo())
	if _, err := NewPosition(state); err == nil {
		t.Fatal("expected pending turn confirmation to be rejected")
	}
}

func assertActionSetsEqual(t *testing.T, production, oracle []SearchAction) {
	t.Helper()
	if len(production) != len(oracle) {
		productionKeys := make(map[string]bool, len(production))
		oracleKeys := make(map[string]bool, len(oracle))
		for _, action := range production {
			productionKeys[action.Key()] = true
		}
		for _, action := range oracle {
			oracleKeys[action.Key()] = true
		}
		var missing, extra []string
		for key := range oracleKeys {
			if !productionKeys[key] {
				missing = append(missing, key)
			}
		}
		for key := range productionKeys {
			if !oracleKeys[key] {
				extra = append(extra, key)
			}
		}
		sort.Strings(missing)
		sort.Strings(extra)
		t.Fatalf("legal action count differs: production=%d oracle=%d\nmissing=%v\nextra=%v", len(production), len(oracle), missing, extra)
	}
	for i := range production {
		if production[i].Key() != oracle[i].Key() {
			t.Fatalf("legal action mismatch at %d\nproduction=%s\noracle=%s", i, production[i].Key(), oracle[i].Key())
		}
	}
}

var _ Position = (*GamePosition)(nil)
var _ = game.PhaseEnd
