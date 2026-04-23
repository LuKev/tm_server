package websocket

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/lobby"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
	"github.com/lukev/tm_server/internal/replay"
)

var errReplayEnded = errors.New("replay reached end")
var errGoldenCannotAutoFundUpgrade = errors.New("cannot auto-fund upgrade cost")

//go:embed testdata/*.txt
var snellmanFixtureFS embed.FS

type goldenFixtureSpec struct {
	id       string
	fixture  string
	playerID []string
	expected map[string]int
}

func goldenFixtureCatalog() []goldenFixtureSpec {
	return []goldenFixtureSpec{
		{
			id:       "s69_g2",
			fixture:  "testdata/4pLeague_S69_D1L1_G2.txt",
			playerID: []string{"Witches", "Nomads", "Darklings", "Mermaids"},
			expected: map[string]int{
				"Nomads":    166,
				"Darklings": 137,
				"Mermaids":  130,
				"Witches":   124,
			},
		},
		{
			id:       "s60_g4",
			fixture:  "testdata/4pLeague_S60_D1L1_G4.txt",
			playerID: []string{"Cultists", "Darklings", "Dwarves", "Giants"},
			expected: map[string]int{
				"Dwarves":   167,
				"Darklings": 151,
				"Cultists":  149,
				"Giants":    115,
			},
		},
		{
			id:       "s61_g3",
			fixture:  "testdata/4pLeague_S61_D1L1_G3.txt",
			playerID: []string{"Darklings", "Cultists", "Engineers", "Witches"},
			expected: map[string]int{
				"Cultists":  151,
				"Darklings": 151,
				"Witches":   126,
				"Engineers": 125,
			},
		},
		{
			id:       "s69_g7",
			fixture:  "testdata/4pLeague_S69_D1L1_G7.txt",
			playerID: []string{"Cultists", "Engineers", "Swarmlings", "Nomads"},
			expected: map[string]int{
				"Swarmlings": 139,
				"Nomads":     135,
				"Engineers":  133,
				"Cultists":   131,
			},
		},
	}
}

func findGoldenFixtureSpec(id string) (goldenFixtureSpec, bool) {
	for _, spec := range goldenFixtureCatalog() {
		if spec.id == id {
			return spec, true
		}
	}
	return goldenFixtureSpec{}, false
}

func goldenFixtureIDs() []string {
	catalog := goldenFixtureCatalog()
	ids := make([]string, 0, len(catalog))
	for _, spec := range catalog {
		ids = append(ids, spec.id)
	}
	return ids
}

type goldenExportAction struct {
	PlayerID string         `json:"playerId"`
	Type     string         `json:"type"`
	Params   map[string]any `json:"params"`
}

type goldenExportScript struct {
	Fixture             string               `json:"fixture"`
	PlayerIDs           []string             `json:"playerIds"`
	ScoringTiles        []string             `json:"scoringTiles"`
	BonusCards          []string             `json:"bonusCards"`
	TurnOrderPolicy     string               `json:"turnOrderPolicy"`
	Actions             []goldenExportAction `json:"actions"`
	ExpectedFinalScores map[string]int       `json:"expectedFinalScores"`
}

func TestWebsocketGolden_SnellmanS69D1L1G2_CompletesWithExpectedScores(t *testing.T) {
	runGoldenSnellmanFixture(
		t,
		"testdata/4pLeague_S69_D1L1_G2.txt",
		[]string{"Witches", "Nomads", "Darklings", "Mermaids"},
		map[string]int{
			"Nomads":    166,
			"Darklings": 137,
			"Mermaids":  130,
			"Witches":   124,
		},
		true,
	)
}

func TestWebsocketGolden_SnellmanS61D1L1G3_CompletesWithExpectedScores(t *testing.T) {
	runGoldenSnellmanFixture(
		t,
		"testdata/4pLeague_S61_D1L1_G3.txt",
		[]string{"Darklings", "Cultists", "Engineers", "Witches"},
		map[string]int{
			"Cultists":  151,
			"Darklings": 151,
			"Witches":   126,
			"Engineers": 125,
		},
		true,
	)
}

func TestWebsocketGolden_ExportActionScript(t *testing.T) {
	normalizedID := strings.TrimSpace(os.Getenv("TM_EXPORT_GOLDEN_ID"))
	if normalizedID == "" {
		t.Skip("set TM_EXPORT_GOLDEN_ID to export a fixture action script")
	}
	outputPath := strings.TrimSpace(os.Getenv("TM_EXPORT_GOLDEN_ACTIONS_PATH"))
	if outputPath == "" {
		t.Skip("set TM_EXPORT_GOLDEN_ACTIONS_PATH to export a fixture action script")
	}

	spec, ok := findGoldenFixtureSpec(normalizedID)
	if !ok {
		t.Fatalf("unknown TM_EXPORT_GOLDEN_ID=%q (valid: %s)", normalizedID, strings.Join(goldenFixtureIDs(), ", "))
	}

	script := runGoldenSnellmanFixture(t, spec.fixture, spec.playerID, spec.expected, false)
	encoded, err := json.MarshalIndent(script, "", "  ")
	if err != nil {
		t.Fatalf("marshal export script: %v", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(outputPath, encoded, 0o644); err != nil {
		t.Fatalf("write export script %s: %v", outputPath, err)
	}
	t.Logf("exported %d actions for %s to %s", len(script.Actions), normalizedID, outputPath)
}

func TestWebsocketGoldenCandidateFixtures_TargetedCoverageInventory(t *testing.T) {
	type candidate struct {
		fixture  string
		patterns []string
	}
	candidates := []candidate{
		{
			fixture: "testdata/4pLeague_S1_D1L1_G3.txt",
			patterns: []string{
				"fakirs",
				"dwarves",
				"action ACT6",
				"Bridge",
			},
		},
		{
			fixture: "testdata/4pLeague_S60_D1L1_G4.txt",
			patterns: []string{
				"dwarves",
				"cultists",
				"[opponent accepted power]",
				"Bridge",
			},
		},
		{
			fixture: "testdata/4pLeague_S61_D1L1_G3.txt",
			patterns: []string{
				"engineers",
				"cultists",
				"ACTE",
				"Bridge",
				"[opponent accepted power]",
			},
		},
		{
			fixture: "testdata/4pLeague_S69_D1L1_G4.txt",
			patterns: []string{
				"dwarves",
				"cultists",
				"[opponent accepted power]",
			},
		},
		{
			fixture: "testdata/4pLeague_S69_D1L1_G7.txt",
			patterns: []string{
				"engineers",
				"cultists",
				"Bridge",
				"[opponent accepted power]",
			},
		},
	}

	for _, c := range candidates {
		data, err := snellmanFixtureFS.ReadFile(c.fixture)
		if err != nil {
			t.Fatalf("read candidate fixture %s: %v", c.fixture, err)
		}
		content := strings.ToLower(string(data))
		for _, pattern := range c.patterns {
			if !strings.Contains(content, strings.ToLower(pattern)) {
				t.Fatalf("candidate fixture %s missing expected pattern %q", c.fixture, pattern)
			}
		}
	}
}

func runGoldenSnellmanFixture(
	t *testing.T,
	fixturePath string,
	playerIDs []string,
	expected map[string]int,
	validateFinal bool,
) *goldenExportScript {
	t.Helper()

	fixtureBytes, err := snellmanFixtureFS.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read embedded fixture: %v", err)
	}

	concise, err := notation.ConvertSnellmanToConciseForReplay(string(fixtureBytes))
	if err != nil {
		t.Fatalf("convert fixture to concise: %v", err)
	}

	items, err := notation.ParseConciseLogStrict(concise)
	if err != nil {
		t.Fatalf("parse concise fixture: %v", err)
	}

	compareReplay := strings.EqualFold(strings.TrimSpace(os.Getenv("TM_COMPARE_REPLAY_ON_RUN")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("TM_COMPARE_REPLAY_ON_RUN")), "true")
	var replayComparator *replayActionComparator
	if compareReplay {
		replayComparator, err = newReplayActionComparator(string(fixtureBytes), playerIDs)
		if err != nil {
			t.Fatalf("create replay comparator: %v", err)
		}
	}

	settings, actions := extractGoldenSettingsAndActions(items)
	if len(actions) == 0 {
		t.Fatalf("no actions parsed from fixture")
	}

	deps, server, gameID, clients, state := setupWebsocketGoldenGame(t, playerIDs)
	defer server.Close()
	defer closeConnections(clients)

	gs, ok := deps.Games.GetGame(gameID)
	if !ok {
		t.Fatalf("golden game not found: %s", gameID)
	}
	gs.TurnOrderPolicy = game.TurnOrderPolicyPassOrder
	if err := applyGoldenSettings(gs, settings); err != nil {
		t.Fatalf("apply golden settings: %v", err)
	}

	// Refresh state after deterministic fixture setup.
	sendJSON(t, clients[playerIDs[0]], map[string]any{
		"type": "get_game_state",
		"payload": map[string]any{
			"gameID":   gameID,
			"playerID": playerIDs[0],
		},
	})
	state = asMap(readUntilType(t, clients[playerIDs[0]], "game_state_update", 4*time.Second)["payload"])

	runner := &goldenRunner{
		t:                  t,
		gameID:             gameID,
		deps:               deps.Games,
		clients:            clients,
		state:              state,
		gs:                 gs,
		preExecutedActions: make(map[game.Action]bool),
	}

	// Snellman fixtures start after factions are already selected; execute the
	// faction-selection phase explicitly in multiplayer before replaying setup rows.
	for _, playerID := range playerIDs {
		faction := models.FactionTypeFromString(playerID)
		if faction == models.FactionUnknown {
			t.Fatalf("unknown faction for player id %s", playerID)
		}
		if err := runner.perform(playerID, "select_faction", map[string]any{
			"faction": faction.String(),
		}); err != nil {
			t.Fatalf("select faction %s: %v", playerID, err)
		}
	}

	for i, action := range actions {
		if runner.preExecutedActions[action] {
			continue
		}
		if runner.isNoOpLeechAction(action) {
			if testing.Verbose() {
				runner.t.Logf("golden skip no-op leech idx=%d action=%s", i, describeGoldenAction(action))
			}
			if replayComparator != nil {
				if err := replayComparator.advanceAndAssertPostState(action, i, runner.state, runner.gs); err != nil {
					t.Fatalf("replay post-action mismatch after action %d: %v", i, err)
				}
			}
			continue
		}
		if strings.EqualFold(strings.TrimSpace(os.Getenv("TM_GOLDEN_TRACE")), "1") || strings.EqualFold(strings.TrimSpace(os.Getenv("TM_GOLDEN_TRACE")), "true") {
			pd := asMap(runner.state["pendingDecision"])
			runner.t.Logf(
				"golden trace idx=%d action=%s player=%s pendingType=%s pendingPlayer=%s phase=%v round=%v turn=%s\n",
				i,
				describeGoldenAction(action),
				strings.TrimSpace(action.GetPlayerID()),
				asString(pd["type"]),
				asString(pd["playerId"]),
				asInt(runner.state["phase"]),
				asInt(asMap(runner.state["round"])["round"]),
				currentTurnPlayerID(runner.state),
			)
		}
		if replayComparator != nil {
			if err := replayComparator.expectPreState(i, action, runner.state, runner.gs); err != nil {
				t.Fatalf("replay pre-state mismatch before action %d: %v", i, err)
			}
		}
		nextPlayer := strings.TrimSpace(action.GetPlayerID())
		if testing.Verbose() {
			pd := asMap(runner.state["pendingDecision"])
			playerState := asMap(asMap(runner.state["players"])[nextPlayer])
			res := asMap(playerState["resources"])
			power := asMap(res["power"])
			runner.t.Logf(
				"golden idx=%d action=%s actionType=%T actionPlayer=%s turn=%s phase=%d round=%d pendingType=%s pendingPlayer=%s pre=vp=%d c=%d w=%d p=%d pw=%d/%d/%d",
				i,
				describeGoldenAction(action),
				action,
				strings.TrimSpace(action.GetPlayerID()),
				currentTurnPlayerID(runner.state),
				asInt(runner.state["phase"]),
				asInt(asMap(runner.state["round"])["round"]),
				asString(pd["type"]),
				asString(pd["playerId"]),
				asInt(playerState["victoryPoints"]),
				asInt(res["coins"]),
				asInt(res["workers"]),
				asInt(res["priests"]),
				asInt(power["powerI"]),
				asInt(power["powerII"]),
				asInt(power["powerIII"]),
			)
		}
		if err := runner.resolveBlockingPendingBefore(nextPlayer, actions[i:]); err != nil {
			t.Fatalf("resolve pending before action %d (%T): %v", i, action, err)
		}
		if err := runner.executeActionWithUpcoming(action, actions[i:]); err != nil {
			t.Fatalf("execute action %d (%T): %v", i, action, err)
		}
		if replayComparator != nil {
			if err := replayComparator.advanceAndAssertPostState(action, i, runner.state, runner.gs); err != nil {
				t.Fatalf("replay post-action mismatch after action %d: %v", i, err)
			}
		}
	}

	if err := runner.resolveBlockingPendingBefore("", nil); err != nil {
		t.Fatalf("resolve trailing pending decisions: %v", err)
	}

	if asInt(runner.state["phase"]) != int(game.PhaseEnd) {
		t.Fatalf("expected game phase end (%d), got %v", int(game.PhaseEnd), runner.state["phase"])
	}

	finalScoring := asMap(runner.state["finalScoring"])
	if len(finalScoring) == 0 {
		t.Fatalf("expected non-empty final scoring")
	}

	if !validateFinal {
		if expected == nil {
			expected = make(map[string]int, len(finalScoring))
		} else {
			for key := range expected {
				delete(expected, key)
			}
		}
		for playerID, playerEntry := range finalScoring {
			entry := asMap(playerEntry)
			if len(entry) == 0 {
				continue
			}
			expected[playerID] = asInt(entry["totalVp"])
		}
	} else {
		for playerID, want := range expected {
			entry := asMap(finalScoring[playerID])
			if len(entry) == 0 {
				t.Fatalf("missing final scoring entry for %s; got keys=%v", playerID, mapKeys(finalScoring))
			}
			got := asInt(entry["totalVp"])
			if got != want {
				t.Fatalf("final score mismatch for %s: got %d, want %d entry=%v", playerID, got, want, entry)
			}
		}
	}

	turnOrderPolicy := string(game.TurnOrderPolicyPassOrder)

	return &goldenExportScript{
		Fixture:             fixturePath,
		PlayerIDs:           slices.Clone(playerIDs),
		ScoringTiles:        splitCSV(settings["ScoringTiles"]),
		BonusCards:          splitCSV(settings["BonusCards"]),
		TurnOrderPolicy:     turnOrderPolicy,
		Actions:             slices.Clone(runner.recordedActions),
		ExpectedFinalScores: cloneIntMap(expected),
	}
}

type goldenRunner struct {
	t                  *testing.T
	gameID             string
	deps               *game.Manager
	clients            map[string]*gws.Conn
	state              map[string]any
	gs                 *game.GameState
	step               int
	compoundDepth      int
	recordedActions    []goldenExportAction
	preExecutedActions map[game.Action]bool
}

func (r *goldenRunner) executeActionWithUpcoming(action game.Action, upcoming []game.Action) error {
	if action == nil {
		return nil
	}

	switch a := action.(type) {
	case *notation.LogCompoundAction:
		return r.executeCompound(a, upcoming)
	case *notation.LogPreIncomeAction:
		if testing.Verbose() {
			innerType := "<nil>"
			innerPlayer := ""
			if a.Action != nil {
				innerType = fmt.Sprintf("%T", a.Action)
				innerPlayer = strings.TrimSpace(a.Action.GetPlayerID())
			}
			pd := asMap(r.state["pendingDecision"])
			r.t.Logf(
				"golden pre-income encountered inner=%s innerPlayer=%s turn=%s pendingType=%s pendingPlayer=%s",
				innerType,
				innerPlayer,
				currentTurnPlayerID(r.state),
				asString(pd["type"]),
				asString(pd["playerId"]),
			)
		}
		if transform, ok := a.Action.(*game.TransformAndBuildAction); ok {
			r.ensurePendingCultRewardSpadesForLeadingPreIncomeTransforms(transform.PlayerID, upcoming)
		}
		return r.executeActionWithUpcoming(a.Action, upcoming)
	case *notation.LogPostIncomeAction:
		return r.executeActionWithUpcoming(a.Action, upcoming)
	case *game.SelectFactionAction:
		return r.perform(a.PlayerID, "select_faction", map[string]any{
			"faction": a.FactionType.String(),
		})
	case *game.SetupDwellingAction:
		return r.perform(a.PlayerID, "setup_dwelling", map[string]any{
			"hex": toHexParam(a.Hex),
		})
	case *notation.LogBonusCardSelectionAction:
		card := notation.ParseBonusCardCode(strings.ToUpper(strings.TrimSpace(a.BonusCard)))
		if card == game.BonusCardUnknown {
			return fmt.Errorf("unknown setup bonus card code: %q", a.BonusCard)
		}
		return r.perform(a.PlayerID, "setup_bonus_card", map[string]any{
			"bonusCard": int(card),
		})
	case *game.TransformAndBuildAction:
		return r.executeTransformAction(a)
	case *notation.LogDigTransformAction:
		if testing.Verbose() {
			r.t.Logf(
				"golden dig transform action idx=%d player=%s target=%v spades=%d",
				len(r.recordedActions),
				strings.TrimSpace(a.PlayerID),
				a.Target,
				a.Spades,
			)
		}
		return r.perform(a.PlayerID, "transform_build", map[string]any{
			"targetHex": toHexParam(a.Target),
		})
	case *game.UpgradeBuildingAction:
		if err := r.tryAutoFundUpgradeByCost(a.PlayerID, func(player *game.Player) (int, int, int) {
			return r.getUpgradeActionCost(a.PlayerID, a)
		}); err != nil {
			return err
		}
		return r.perform(a.PlayerID, "upgrade_building", map[string]any{
			"targetHex":       toHexParam(a.TargetHex),
			"newBuildingType": int(a.NewBuildingType),
		})
	case *game.AdvanceShippingAction:
		if err := r.tryAutoFundUpgradeByCost(
			a.PlayerID,
			func(player *game.Player) (int, int, int) {
				cost := player.Faction.GetShippingCost(player.ShippingLevel)
				return cost.Coins, cost.Workers, cost.Priests
			},
		); err != nil {
			return err
		}
		return r.perform(a.PlayerID, "advance_shipping", map[string]any{})
	case *game.AdvanceDiggingAction:
		if err := r.tryAutoFundUpgradeByCost(
			a.PlayerID,
			func(player *game.Player) (int, int, int) {
				cost := player.Faction.GetDiggingCost(player.DiggingLevel)
				return cost.Coins, cost.Workers, cost.Priests
			},
		); err != nil {
			return err
		}
		return r.perform(a.PlayerID, "advance_digging", map[string]any{})
	case *game.SendPriestToCultAction:
		return r.perform(a.PlayerID, "send_priest", map[string]any{
			"track":  int(a.Track),
			"spaces": a.SpacesToClimb,
		})
	case *game.PassAction:
		params := map[string]any{}
		if a.BonusCard != nil {
			params["bonusCard"] = int(*a.BonusCard)
		}
		return r.perform(a.PlayerID, "pass", params)
	case *notation.LogBurnAction:
		amount := a.Amount
		maxBurn := maxBurnPossible(r.state, a.PlayerID)
		if maxBurn <= 0 {
			return fmt.Errorf("cannot execute burn for %s: no bowl II power available", strings.TrimSpace(a.PlayerID))
		}
		if amount > maxBurn {
			amount = maxBurn
		}
		if amount <= 0 {
			return fmt.Errorf("cannot execute burn for %s: amount <= 0", strings.TrimSpace(a.PlayerID))
		}
		return r.perform(a.PlayerID, "burn_power", map[string]any{"amount": amount})
	case *notation.LogConversionAction:
		if testing.Verbose() && strings.EqualFold(strings.TrimSpace(a.PlayerID), "Witches") {
			r.t.Logf(
				"golden debug conversion pre step=%d action=%T player=%s state=%s gs=%s",
				r.step,
				a,
				strings.TrimSpace(a.PlayerID),
				r.playerResourceSummary(r.state, a.PlayerID),
				r.playerResourceSummaryFromGS(a.PlayerID),
			)
		}
		if amount, ok := detectDarklingsWorkerToPriestConversion(a); ok {
			if !strings.EqualFold(strings.TrimSpace(a.PlayerID), models.FactionDarklings.String()) {
				return fmt.Errorf(
					"unsupported conversion shape for non-darklings: cost=%v reward=%v",
					a.Cost, a.Reward,
				)
			}
			if err := r.recordReplayDarklingsOrdination(a.PlayerID, amount); err != nil {
				return err
			}
			return nil
		}
		if amount, ok := detectPriestToCoinConversion(a); ok {
			if amount <= 0 {
				return fmt.Errorf("cannot execute priest->coin conversion for %s: amount <= 0", strings.TrimSpace(a.PlayerID))
			}
			if err := r.recordReplayConversion(a.PlayerID, game.ConversionPriestToWorker, amount); err != nil {
				return err
			}
			if err := r.recordReplayConversion(a.PlayerID, game.ConversionWorkerToCoin, amount); err != nil {
				return err
			}
			return nil
		}
		convType, amount, err := mapLogConversion(a)
		if err != nil {
			return err
		}
		amount, err = r.prepareConversionAmount(a.PlayerID, convType, amount)
		if err != nil {
			return err
		}
		if amount <= 0 {
			return fmt.Errorf(
				"cannot execute conversion for %s: prepared amount <= 0 (convType=%s)",
				strings.TrimSpace(a.PlayerID),
				convType,
			)
		}
		if testing.Verbose() && strings.EqualFold(strings.TrimSpace(a.PlayerID), "Witches") {
			r.t.Logf(
				"golden debug conversion record step=%d player=%s convType=%s amount=%d pre state=%s pre gs=%s",
				r.step,
				strings.TrimSpace(a.PlayerID),
				convType,
				amount,
				r.playerResourceSummary(r.state, a.PlayerID),
				r.playerResourceSummaryFromGS(a.PlayerID),
			)
		}
		return r.recordReplayConversion(a.PlayerID, convType, amount)
	case *notation.LogAcceptLeechAction:
		if !hasPendingLeechOffer(r.state, a.PlayerID) {
			if r.isNoOpLeechAction(a) {
				return nil
			}
			return fmt.Errorf("accept leech for %s without pending offer", strings.TrimSpace(a.PlayerID))
		}
		offerIndex, err := findLeechOfferIndex(r.state, a.PlayerID, a.FromPlayerID, a.PowerAmount, a.Explicit)
		if err != nil {
			if a.Explicit || strings.TrimSpace(a.FromPlayerID) != "" {
				return err
			}
			offerIndex, err = findLeechOfferIndex(r.state, a.PlayerID, "", 0, false)
			if err != nil {
				return err
			}
		}
		params := map[string]any{
			"offerIndex": offerIndex,
		}
		if a.Explicit && a.PowerAmount > 0 {
			params["amount"] = a.PowerAmount
		}
		return r.perform(a.PlayerID, "accept_leech", params)
	case *notation.LogDeclineLeechAction:
		if !hasPendingLeechOffer(r.state, a.PlayerID) {
			if r.isNoOpLeechAction(a) {
				return nil
			}
			return fmt.Errorf("decline leech for %s without pending offer", strings.TrimSpace(a.PlayerID))
		}
		offerIndex, err := findLeechOfferIndex(r.state, a.PlayerID, a.FromPlayerID, 0, false)
		if err != nil {
			if strings.TrimSpace(a.FromPlayerID) != "" {
				return err
			}
			offerIndex, err = findLeechOfferIndex(r.state, a.PlayerID, "", 0, false)
			if err != nil {
				return err
			}
		}
		return r.perform(a.PlayerID, "decline_leech", map[string]any{
			"offerIndex": offerIndex,
		})
	case *notation.LogFavorTileAction:
		tile, err := notation.ParseFavorTileCode(strings.ToUpper(strings.TrimSpace(a.Tile)))
		if err != nil {
			return err
		}
		return r.perform(a.PlayerID, "select_favor_tile", map[string]any{
			"tileType": int(tile),
		})
	case *notation.LogTownAction:
		if err := r.ensureTownTileSelectionPending(a.PlayerID); err != nil {
			return err
		}
		if !r.canSelectTownTile(a.PlayerID) {
			return fmt.Errorf("cannot select town tile for %s: no pending town tile selection", strings.TrimSpace(a.PlayerID))
		}
		tile, err := notation.GetTownTileFromVP(a.VP)
		if err != nil {
			return err
		}
		params, err := r.townTileSelectionParams(a.PlayerID, tile)
		if err != nil {
			return err
		}
		if err := r.perform(a.PlayerID, "select_town_tile", params); err != nil {
			return err
		}
		return r.resolveTownCultTopChoice(a.PlayerID, nil)
	case *notation.LogSpecialAction:
		return r.executeSpecialAction(a)
	case *game.UseDarklingsPriestOrdinationAction:
		return r.recordReplayDarklingsOrdination(
			a.PlayerID,
			a.WorkersToConvert,
		)
	case *notation.LogPowerAction:
		return r.executeStandalonePowerAction(a)
	case *notation.LogCultistAdvanceAction:
		pd := asMap(r.state["pendingDecision"])
		if asString(pd["type"]) == "cultists_cult_choice" && asString(pd["playerId"]) == a.PlayerID {
			return r.performCultistsTrackChoice(a.PlayerID, int(a.Track))
		}
		return fmt.Errorf(
			"cannot execute cultists track advance for %s: pending decision mismatch (type=%s player=%s)",
			strings.TrimSpace(a.PlayerID),
			asString(pd["type"]),
			asString(pd["playerId"]),
		)
	case *notation.LogCultTrackDecreaseAction:
		// Handled in compound context for town-cult-top selection disambiguation.
		return nil
	default:
		return fmt.Errorf("unsupported action type: %T", action)
	}
}

func (r *goldenRunner) executeCompound(compound *notation.LogCompoundAction, upcoming []game.Action) error {
	if compound == nil {
		return fmt.Errorf("nil compound action")
	}

	r.compoundDepth++
	defer func() { r.compoundDepth-- }()

	compoundDigTargets := map[string]struct{}{}
	pendingTownTracks := make([]game.CultTrack, 0, 2)
	preExecuted := make(map[int]bool, len(compound.Actions))

	for i := 0; i < len(compound.Actions); i++ {
		if preExecuted[i] || r.preExecutedActions[compound.Actions[i]] {
			continue
		}
		action := compound.Actions[i]
		if r.isNoOpLeechAction(action) {
			continue
		}
		shouldClearDigTargets := !isCompoundDigFollowerPreserver(action)

		nextPlayer := strings.TrimSpace(action.GetPlayerID())
		nestedUpcoming := compound.Actions[i:]
		if len(upcoming) > 1 {
			nestedUpcoming = append(slices.Clone(nestedUpcoming), upcoming[1:]...)
		}
		if err := r.resolveBlockingPendingBefore(nextPlayer, nestedUpcoming); err != nil {
			return err
		}
		if err := r.preExecuteReplayFundingForCompoundAction(action, compound.Actions, i, upcoming, preExecuted); err != nil {
			return err
		}

		switch a := action.(type) {
		case *notation.LogDigTransformAction:
			if hasLaterCompoundTransformAction(compound.Actions[i+1:], a.PlayerID, a.Target) {
				continue
			}
			if err := r.executeActionWithUpcoming(a, nestedUpcoming); err != nil {
				return err
			}
			compoundDigTargets[compoundActionLocationKey(a.PlayerID, a.Target)] = struct{}{}
			continue

		case *notation.LogCultTrackDecreaseAction:
			pendingTownTracks = append(pendingTownTracks, a.Track)
			continue

		case *notation.LogPowerAction:
			var nextTown *notation.LogTownAction
			if i+1 < len(compound.Actions) {
				if candidate, ok := compound.Actions[i+1].(*notation.LogTownAction); ok &&
					strings.TrimSpace(candidate.PlayerID) == strings.TrimSpace(a.PlayerID) {
					nextTown = candidate
				}
			}

			powerType := notation.ParsePowerActionCode(strings.ToUpper(strings.TrimSpace(a.ActionCode)))
			if powerType == game.PowerActionUnknown {
				return fmt.Errorf("unknown power action code: %s", a.ActionCode)
			}
			if err := r.burnToReachPowerActionCost(a.PlayerID, powerType); err != nil {
				return err
			}

			if powerType == game.PowerActionSpade1 || powerType == game.PowerActionSpade2 {
				if i+1 >= len(compound.Actions) {
					return fmt.Errorf("spade power action without following transform: %s", a.ActionCode)
				}
				nextTransform, ok := compound.Actions[i+1].(*game.TransformAndBuildAction)
				if !ok {
					return fmt.Errorf("spade power action expected transform follow-up, got %T", compound.Actions[i+1])
				}
				if err := r.tryAutoFundPowerSpadeClaim(a.PlayerID, powerType, nextTransform); err != nil {
					return err
				}

				params := transformBuildParams(nextTransform)
				params["actionType"] = int(powerType)
				if err := r.perform(a.PlayerID, "power_action_claim", params); err != nil {
					return err
				}
				i++

				for i+1 < len(compound.Actions) {
					followTransform, ok := compound.Actions[i+1].(*game.TransformAndBuildAction)
					if !ok {
						break
					}
					pd := asMap(r.state["pendingDecision"])
					if asString(pd["type"]) != "spade_followup" || asString(pd["playerId"]) != a.PlayerID {
						break
					}
					if err := r.perform(a.PlayerID, "transform_build", transformBuildParams(followTransform)); err != nil {
						return err
					}
					i++
				}

				pd := asMap(r.state["pendingDecision"])
				if asString(pd["type"]) == "spade_followup" && asString(pd["playerId"]) == a.PlayerID {
					remaining := asInt(pd["spadesRemaining"])
					if remaining <= 0 {
						remaining = 1
					}
					if err := r.perform(a.PlayerID, "discard_pending_spade", map[string]any{"count": remaining}); err != nil {
						return err
					}
				}
				continue
			}

			if powerType == game.PowerActionBridge {
				h1, h2, err := parseBridgeFromACT1(a.ActionCode)
				if err != nil {
					return err
				}
				if err := r.perform(a.PlayerID, "power_action_claim", map[string]any{
					"actionType": int(powerType),
					"bridgeHex1": toHexParam(h1),
					"bridgeHex2": toHexParam(h2),
				}); err != nil {
					return err
				}
				continue
			}

			if err := r.perform(a.PlayerID, "power_action_claim", map[string]any{"actionType": int(powerType)}); err != nil {
				return err
			}
			if nextTown != nil {
				if err := r.ensureTownTileSelectionPending(a.PlayerID); err != nil {
					return err
				}
				if r.canSelectTownTile(a.PlayerID) {
					tile, err := notation.GetTownTileFromVP(nextTown.VP)
					if err != nil {
						return err
					}
					params, err := r.townTileSelectionParams(a.PlayerID, tile)
					if err != nil {
						return err
					}
					if err := r.perform(a.PlayerID, "select_town_tile", params); err != nil {
						return err
					}
					if err := r.resolveTownCultTopChoice(a.PlayerID, pendingTownTracks); err != nil {
						return err
					}
					pendingTownTracks = pendingTownTracks[:0]
				}
				i++
			}
			continue

		case *notation.LogSpecialAction:
			code := strings.ToUpper(strings.TrimSpace(a.ActionCode))
			if strings.HasPrefix(code, "ACT-SH-S-") && i+1 < len(compound.Actions) {
				if followTransform, ok := compound.Actions[i+1].(*game.TransformAndBuildAction); ok && strings.TrimSpace(followTransform.PlayerID) == strings.TrimSpace(a.PlayerID) {
					params := map[string]any{
						"actionType":    int(game.SpecialActionGiantsTransform),
						"targetHex":     toHexParam(followTransform.TargetHex),
						"buildDwelling": followTransform.BuildDwelling,
					}
					if followTransform.TargetTerrain != models.TerrainTypeUnknown {
						params["targetTerrain"] = int(followTransform.TargetTerrain)
					}
					if err := r.perform(a.PlayerID, "special_action_use", params); err != nil {
						return err
					}
					i++
					continue
				}
			}

		case *notation.LogTownAction:
			if err := r.ensureTownTileSelectionPending(a.PlayerID); err != nil {
				return err
			}
			if !r.canSelectTownTile(a.PlayerID) {
				return fmt.Errorf("cannot select compound town tile for %s: no pending town tile selection", strings.TrimSpace(a.PlayerID))
			}
			tile, err := notation.GetTownTileFromVP(a.VP)
			if err != nil {
				return err
			}
			params, err := r.townTileSelectionParams(a.PlayerID, tile)
			if err != nil {
				return err
			}
			if err := r.perform(a.PlayerID, "select_town_tile", params); err != nil {
				return err
			}
			if err := r.resolveTownCultTopChoice(a.PlayerID, pendingTownTracks); err != nil {
				return err
			}
			pendingTownTracks = pendingTownTracks[:0]
			continue
		}

		if a, ok := action.(*game.TransformAndBuildAction); ok {
			if _, found := compoundDigTargets[compoundActionLocationKey(a.PlayerID, a.TargetHex)]; found {
				if a.BuildDwelling {
					delete(compoundDigTargets, compoundActionLocationKey(a.PlayerID, a.TargetHex))
				} else {
					if testing.Verbose() {
						r.t.Logf(
							"golden skip compound transform duplicate idx=%d actionPlayer=%s target=%v",
							len(r.recordedActions),
							strings.TrimSpace(a.PlayerID),
							a.TargetHex,
						)
					}
					delete(compoundDigTargets, compoundActionLocationKey(a.PlayerID, a.TargetHex))
					if shouldClearDigTargets {
						for k := range compoundDigTargets {
							delete(compoundDigTargets, k)
						}
					}
					continue
				}
			}
		}

		if err := r.executeActionWithUpcoming(action, nestedUpcoming); err != nil {
			return err
		}
		if shouldClearDigTargets {
			for k := range compoundDigTargets {
				delete(compoundDigTargets, k)
			}
		}
	}

	return nil
}

type goldenReplayInsufficientResources struct {
	needCoins, needWorkers, needPriests, needPower int
	haveCoins, haveWorkers, havePriests, havePower int
}

func goldenIsReplayAffordabilityError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not enough resources") ||
		strings.Contains(msg, "insufficient resources") ||
		strings.Contains(msg, "not enough workers") ||
		strings.Contains(msg, "not enough priests") ||
		strings.Contains(msg, "cannot afford")
}

func parseGoldenReplayInsufficientResources(err error) (goldenReplayInsufficientResources, bool) {
	if err == nil {
		return goldenReplayInsufficientResources{}, false
	}
	re := regexp.MustCompile(`insufficient resources: need \(coins:(\d+), workers:(\d+), priests:(\d+), power:(\d+)\), have \(coins:(\d+), workers:(\d+), priests:(\d+), power:(\d+)\)`)
	m := re.FindStringSubmatch(strings.ToLower(err.Error()))
	if len(m) != 9 {
		return goldenReplayInsufficientResources{}, false
	}
	toInt := func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	}
	return goldenReplayInsufficientResources{
		needCoins:   toInt(m[1]),
		needWorkers: toInt(m[2]),
		needPriests: toInt(m[3]),
		needPower:   toInt(m[4]),
		haveCoins:   toInt(m[5]),
		haveWorkers: toInt(m[6]),
		havePriests: toInt(m[7]),
		havePower:   toInt(m[8]),
	}, true
}

func parseGoldenReplayNotEnoughResources(err error) (goldenReplayInsufficientResources, bool) {
	if err == nil {
		return goldenReplayInsufficientResources{}, false
	}
	re := regexp.MustCompile(`not enough resources for [^:]+: need \{(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\}, have &\{(\d+)\s+(\d+)\s+(\d+)\s+`)
	m := re.FindStringSubmatch(strings.ToLower(err.Error()))
	if len(m) != 8 {
		return goldenReplayInsufficientResources{}, false
	}
	toInt := func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	}
	return goldenReplayInsufficientResources{
		needCoins:   toInt(m[1]),
		needWorkers: toInt(m[2]),
		needPriests: toInt(m[3]),
		needPower:   toInt(m[4]),
		haveCoins:   toInt(m[5]),
		haveWorkers: toInt(m[6]),
		havePriests: toInt(m[7]),
		havePower:   0,
	}, true
}

func goldenIsReplayPreExecutableActionForError(action game.Action, err error) bool {
	if action == nil || err == nil {
		return false
	}

	need, ok := parseGoldenReplayInsufficientResources(err)
	if !ok {
		need, ok = parseGoldenReplayNotEnoughResources(err)
		if !ok {
			_, isBurn := action.(*notation.LogBurnAction)
			return isBurn
		}
	}

	defCoins := need.needCoins - need.haveCoins
	defWorkers := need.needWorkers - need.haveWorkers
	defPriests := need.needPriests - need.havePriests
	defPower := need.needPower - need.havePower

	switch v := action.(type) {
	case *notation.LogBurnAction:
		return true
	case *notation.LogConversionAction:
		if defCoins > 0 && v.Reward[models.ResourceCoin] > 0 {
			return true
		}
		if defWorkers > 0 && v.Reward[models.ResourceWorker] > 0 {
			return true
		}
		if defPriests > 0 && v.Reward[models.ResourcePriest] > 0 {
			return true
		}
		if defPower > 0 && v.Reward[models.ResourcePower] > 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (r *goldenRunner) preExecuteReplayFundingForCompoundAction(
	action game.Action,
	actions []game.Action,
	index int,
	upcoming []game.Action,
	preExecuted map[int]bool,
) error {
	if action == nil {
		return nil
	}
	validateErr := action.Validate(r.gs)
	if !goldenIsReplayAffordabilityError(validateErr) {
		return nil
	}

	replayed := false
	for j := index + 1; j < len(actions); j++ {
		later := actions[j]
		if preExecuted[j] || !goldenIsReplayPreExecutableActionForError(later, validateErr) {
			continue
		}
		laterUpcoming := actions[j:]
		if len(upcoming) > 1 {
			laterUpcoming = append(slices.Clone(laterUpcoming), upcoming[1:]...)
		}
		if err := r.resolveBlockingPendingBefore(strings.TrimSpace(later.GetPlayerID()), laterUpcoming); err != nil {
			return err
		}
		if err := r.executeActionWithUpcoming(later, laterUpcoming); err != nil {
			return err
		}
		preExecuted[j] = true
		replayed = true
	}
	if !replayed {
		return nil
	}
	return nil
}

func (r *goldenRunner) isNoOpLeechAction(action game.Action) bool {
	switch a := action.(type) {
	case *notation.LogAcceptLeechAction:
		if hasPendingLeechOffer(r.state, a.PlayerID) {
			return false
		}
		player := r.gs.GetPlayer(a.PlayerID)
		if player == nil || player.Resources == nil || player.Resources.Power == nil {
			return false
		}
		capacity := player.Resources.Power.Bowl2 + 2*player.Resources.Power.Bowl1
		return capacity <= 0
	case *notation.LogDeclineLeechAction:
		return !hasPendingLeechOffer(r.state, a.PlayerID)
	default:
		return false
	}
}

func (r *goldenRunner) executeTransformAction(action *game.TransformAndBuildAction) error {
	if action == nil {
		return fmt.Errorf("nil transform action")
	}
	pd := asMap(r.state["pendingDecision"])
	if asString(pd["type"]) == "cult_reward_spade" && asString(pd["playerId"]) == action.PlayerID {
		params := map[string]any{"targetHex": toHexParam(action.TargetHex)}
		if action.TargetTerrain != models.TerrainTypeUnknown {
			params["targetTerrain"] = int(action.TargetTerrain)
		}
		if err := r.perform(action.PlayerID, "use_cult_spade", params); err != nil {
			return err
		}
		// Pre-income transform rows in concise can represent multi-spade transforms.
		// When a pure transform still isn't at home terrain, continue consuming the
		// same player's pending cult-reward spades on that hex until complete.
		if !action.BuildDwelling && action.TargetTerrain == models.TerrainTypeUnknown {
			for guard := 0; guard < 4; guard++ {
				pd = asMap(r.state["pendingDecision"])
				if asString(pd["type"]) != "cult_reward_spade" || asString(pd["playerId"]) != action.PlayerID {
					break
				}
				player := r.gs.GetPlayer(action.PlayerID)
				if player == nil {
					break
				}
				mapHex := r.gs.Map.GetHex(action.TargetHex)
				if mapHex == nil || mapHex.Terrain == player.Faction.GetHomeTerrain() {
					break
				}
				if err := r.perform(action.PlayerID, "use_cult_spade", params); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := r.tryAutoFundTransformBuild(action); err != nil {
		return err
	}
	return r.perform(action.PlayerID, "transform_build", transformBuildParams(action))
}

func (r *goldenRunner) executeSpecialAction(action *notation.LogSpecialAction) error {
	if action == nil {
		return fmt.Errorf("nil special action")
	}
	code := strings.ToUpper(strings.TrimSpace(action.ActionCode))
	params := map[string]any{}

	switch {
	case strings.HasPrefix(code, "ACT-BON-"):
		trackLetter := strings.TrimPrefix(code, "ACT-BON-")
		track, err := parseCultTrackLetter(trackLetter)
		if err != nil {
			return err
		}
		params["actionType"] = int(game.SpecialActionBonusCardCultAdvance)
		params["cultTrack"] = int(track)
		return r.perform(action.PlayerID, "special_action_use", params)

	case strings.HasPrefix(code, "ACTS-"):
		targetHex, build, targetTerrain, err := parseACTSSpecial(code)
		if err != nil {
			return err
		}
		params["actionType"] = int(game.SpecialActionBonusCardSpade)
		params["targetHex"] = toHexParam(targetHex)
		params["buildDwelling"] = build
		if targetTerrain != models.TerrainTypeUnknown {
			params["targetTerrain"] = int(targetTerrain)
		}
		return r.perform(action.PlayerID, "special_action_use", params)

	case strings.HasPrefix(code, "ACT-SH-D-"):
		hexCode := strings.TrimPrefix(code, "ACT-SH-D-")
		hex, err := notation.ConvertLogCoordToAxial(hexCode)
		if err != nil {
			return err
		}
		params["actionType"] = int(game.SpecialActionWitchesRide)
		params["targetHex"] = toHexParam(hex)
		return r.perform(action.PlayerID, "special_action_use", params)

	case strings.HasPrefix(code, "ACT-SH-T-"):
		targetHex, build, _, err := parseStrongholdTransform(code, "ACT-SH-T-")
		if err != nil {
			return err
		}
		params["actionType"] = int(game.SpecialActionNomadsSandstorm)
		params["targetHex"] = toHexParam(targetHex)
		params["buildDwelling"] = build
		return r.perform(action.PlayerID, "special_action_use", params)

	case strings.HasPrefix(code, "ACT-SH-S-"):
		hexCode := strings.TrimPrefix(code, "ACT-SH-S-")
		hex, err := notation.ConvertLogCoordToAxial(hexCode)
		if err != nil {
			return err
		}
		params["actionType"] = int(game.SpecialActionGiantsTransform)
		params["targetHex"] = toHexParam(hex)
		params["buildDwelling"] = false
		return r.perform(action.PlayerID, "special_action_use", params)

	case strings.HasPrefix(code, "ACT-SH-TP-"):
		hexCode := strings.TrimPrefix(code, "ACT-SH-TP-")
		hex, err := notation.ConvertLogCoordToAxial(hexCode)
		if err != nil {
			return err
		}
		params["actionType"] = int(game.SpecialActionSwarmlingsUpgrade)
		params["targetHex"] = toHexParam(hex)
		return r.perform(action.PlayerID, "special_action_use", params)
	case strings.HasPrefix(code, "ACT-BR-"):
		parts := strings.Split(code, "-")
		if len(parts) < 4 {
			return fmt.Errorf("invalid engineers bridge action code: %s", code)
		}
		hex1, err := notation.ConvertLogCoordToAxial(parts[2])
		if err != nil {
			return err
		}
		hex2, err := notation.ConvertLogCoordToAxial(parts[3])
		if err != nil {
			return err
		}
		return r.perform(action.PlayerID, "engineers_bridge", map[string]any{
			"bridgeHex1": toHexParam(hex1),
			"bridgeHex2": toHexParam(hex2),
		})

	case strings.HasPrefix(code, "ACT-FAV-"):
		trackLetter := strings.TrimPrefix(code, "ACT-FAV-")
		track, err := parseCultTrackLetter(trackLetter)
		if err != nil {
			return err
		}
		params["actionType"] = int(game.SpecialActionWater2CultAdvance)
		params["cultTrack"] = int(track)
		return r.perform(action.PlayerID, "special_action_use", params)

	default:
		return fmt.Errorf("unsupported special action code: %s", code)
	}
}

func (r *goldenRunner) resolveTownCultTopChoice(playerID string, preferred []game.CultTrack) error {
	pd := asMap(r.state["pendingDecision"])
	if asString(pd["type"]) != "town_cult_top_choice" || asString(pd["playerId"]) != playerID {
		return nil
	}

	maxSelections := asInt(pd["maxSelections"])
	if maxSelections <= 0 {
		maxSelections = 1
	}
	candidate := parseTrackListAny(pd["candidateTracks"])
	if len(candidate) == 0 {
		return fmt.Errorf("town_cult_top_choice without candidate tracks")
	}

	selected := make([]int, 0, maxSelections)
	for _, track := range preferred {
		if len(selected) >= maxSelections {
			break
		}
		if slices.Contains(candidate, int(track)) && !slices.Contains(selected, int(track)) {
			selected = append(selected, int(track))
		}
	}
	for _, track := range candidate {
		if len(selected) >= maxSelections {
			break
		}
		if !slices.Contains(selected, track) {
			selected = append(selected, track)
		}
	}
	if len(selected) == 0 {
		return fmt.Errorf("failed to resolve town cult top choice")
	}

	return r.perform(playerID, "select_town_cult_top", map[string]any{"tracks": selected})
}

func (r *goldenRunner) ensureTownTileSelectionPending(playerID string) error {
	pd := asMap(r.state["pendingDecision"])
	if asString(pd["type"]) == "town_tile_selection" && asString(pd["playerId"]) == playerID {
		return nil
	}
	if r.hasAnyPendingTownFormation(playerID) {
		return nil
	}

	// Snellman rows can represent delayed Mermaids town claims without an explicit
	// actionable token in concise. Rebuild a legal pending town formation from the
	// current board state when we need the town-tile decision.
	if strings.TrimSpace(playerID) != models.FactionMermaids.String() {
		return nil
	}
	if !r.synthesizeMermaidsPendingTown(playerID) {
		return nil
	}

	if r.gs.GetPendingTownSelectionPlayer() != playerID {
		return fmt.Errorf("expected pending town tile selection for %s after mermaids connect", playerID)
	}
	return nil
}

func (r *goldenRunner) townTileSelectionParams(playerID string, tile models.TownTileType) (map[string]any, error) {
	params := map[string]any{"tileType": int(tile)}
	anchor, ok := r.pendingTownAnchorHex(playerID)
	if !ok {
		return nil, fmt.Errorf("cannot select town tile for %s: no pending town anchor", strings.TrimSpace(playerID))
	}
	params["anchorHex"] = toHexParam(anchor)
	return params, nil
}

func (r *goldenRunner) pendingTownAnchorHex(playerID string) (board.Hex, bool) {
	if r == nil || r.gs == nil {
		return board.Hex{}, false
	}
	pendingTowns, ok := r.gs.PendingTownFormations[playerID]
	if !ok || len(pendingTowns) == 0 || pendingTowns[0] == nil {
		return board.Hex{}, false
	}
	pending := pendingTowns[0]
	if pending.SkippedRiverHex != nil {
		return *pending.SkippedRiverHex, true
	}
	for _, hex := range pending.Hexes {
		mapHex := r.gs.Map.GetHex(hex)
		if mapHex != nil && mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			return hex, true
		}
	}
	return board.Hex{}, false
}

func hasPendingTownSelection(state map[string]any, playerID string) bool {
	pd := asMap(state["pendingDecision"])
	return asString(pd["type"]) == "town_tile_selection" && asString(pd["playerId"]) == playerID
}

func (r *goldenRunner) hasPendingTownSelection(playerID string) bool {
	if hasPendingTownSelection(r.state, playerID) {
		return true
	}
	if r.gs != nil && r.gs.GetPendingTownSelectionPlayer() == playerID {
		return true
	}
	return false
}

func (r *goldenRunner) hasAnyPendingTownFormation(playerID string) bool {
	if r.gs == nil {
		return false
	}
	return len(r.gs.PendingTownFormations[playerID]) > 0
}

func (r *goldenRunner) canSelectTownTile(playerID string) bool {
	return r.hasPendingTownSelection(playerID) || r.hasAnyPendingTownFormation(playerID)
}

func (r *goldenRunner) resolveBlockingPendingBefore(nextPlayerID string, upcoming []game.Action) error {
	for guard := 0; guard < 12; guard++ {
		pd := asMap(r.state["pendingDecision"])
		decisionType := asString(pd["type"])
		playerID := asString(pd["playerId"])
		if decisionType == "" {
			return nil
		}
		if decisionType == "post_action_free_actions" {
			if playerID == "" {
				return nil
			}
			if len(upcoming) > 0 && canUsePostActionFreeWindow(upcoming[0], playerID) {
				return nil
			}
			if err := r.performSynthetic(playerID, "confirm_turn", nil); err != nil {
				return err
			}
			continue
		}
		if decisionType == "turn_confirmation" {
			if playerID == "" {
				return nil
			}
			if err := r.performSynthetic(playerID, "confirm_turn", nil); err != nil {
				return err
			}
			continue
		}
		if len(upcoming) > 0 && r.canUseImmediateLeechResponse(upcoming[0]) {
			return nil
		}
		if decisionType == "cultists_cult_choice" &&
			(len(upcoming) == 0 || !actionResolvesPendingDecision(decisionType, playerID, upcoming[0])) {
			resolved, err := r.resolveFutureCultistsChoice(playerID, upcoming)
			if err != nil {
				return err
			}
			if resolved {
				continue
			}
		}
		if upcomingActionResolvesPending(decisionType, upcoming, playerID) {
			if nextPlayerID != "" && playerID != nextPlayerID && actionRequiresTurnOwnership(upcoming[0]) {
				return fmt.Errorf("pending decision %q for player %q cannot be resolved by upcoming action for %q", decisionType, playerID, nextPlayerID)
			}
			return nil
		}
		if decisionType == "leech_offer" {
			resolved, err := r.resolveFutureLeechResponse(playerID, upcoming)
			if err != nil {
				return err
			}
			if resolved {
				continue
			}
		}
		if decisionType == "leech_offer" && canContinueFreeActionBeforeLeech(r.state, upcoming) {
			return nil
		}

		switch decisionType {
		default:
			return fmt.Errorf("pending decision %q for player %q unresolved before next action for %q", decisionType, playerID, nextPlayerID)
		}
	}
	return fmt.Errorf("pending decision resolution exceeded iteration guard")
}

func (r *goldenRunner) resolveFutureLeechResponse(playerID string, upcoming []game.Action) (bool, error) {
	for i := 1; i < len(upcoming); i++ {
		action := upcoming[i]
		if action == nil || r.preExecutedActions[action] {
			continue
		}
		response := r.findMatchingLeechResponseInAction(action, playerID)
		if response == nil || r.preExecutedActions[response] {
			continue
		}
		if err := r.executeActionWithUpcoming(response, upcoming[i:]); err != nil {
			return false, err
		}
		r.preExecutedActions[response] = true
		return true, nil
	}
	return false, nil
}

func (r *goldenRunner) findMatchingLeechResponseInAction(action game.Action, playerID string) game.Action {
	response := findLeechResponseInAction(action, playerID)
	if response == nil {
		return nil
	}
	switch a := response.(type) {
	case *notation.LogAcceptLeechAction:
		if _, err := findLeechOfferIndex(r.state, a.PlayerID, a.FromPlayerID, a.PowerAmount, a.Explicit); err != nil {
			return nil
		}
	case *notation.LogDeclineLeechAction:
		if _, err := findLeechOfferIndex(r.state, a.PlayerID, a.FromPlayerID, 0, false); err != nil {
			return nil
		}
	}
	return response
}

func findLeechResponseInAction(action game.Action, playerID string) game.Action {
	switch a := action.(type) {
	case *notation.LogAcceptLeechAction:
		if strings.TrimSpace(a.PlayerID) == strings.TrimSpace(playerID) {
			return a
		}
	case *notation.LogDeclineLeechAction:
		if strings.TrimSpace(a.PlayerID) == strings.TrimSpace(playerID) {
			return a
		}
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if found := findLeechResponseInAction(nested, playerID); found != nil {
				return found
			}
		}
	case *notation.LogPreIncomeAction:
		return findLeechResponseInAction(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return findLeechResponseInAction(a.Action, playerID)
	}
	return nil
}

func (r *goldenRunner) resolveFutureCultistsChoice(playerID string, upcoming []game.Action) (bool, error) {
	for i := 1; i < len(upcoming); i++ {
		action := upcoming[i]
		if action == nil {
			continue
		}
		found := findCultistAdvanceInAction(action, playerID)
		if found == nil || r.preExecutedActions[found] {
			continue
		}
		if err := r.executeActionWithUpcoming(found, upcoming[i:]); err != nil {
			return false, err
		}
		r.preExecutedActions[found] = true
		return true, nil
	}
	return false, nil
}

func canUsePostActionFreeWindow(action game.Action, playerID string) bool {
	if action == nil {
		return false
	}

	switch a := action.(type) {
	case *notation.LogConversionAction, *notation.LogBurnAction:
		return strings.TrimSpace(action.GetPlayerID()) == strings.TrimSpace(playerID)
	case *notation.LogCompoundAction:
		if len(a.Actions) == 0 {
			return false
		}
		return canUsePostActionFreeWindow(a.Actions[0], playerID)
	case *notation.LogPreIncomeAction:
		return canUsePostActionFreeWindow(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return canUsePostActionFreeWindow(a.Action, playerID)
	default:
		return false
	}
}

func (r *goldenRunner) canUseImmediateLeechResponse(action game.Action) bool {
	if r == nil || action == nil {
		return false
	}
	playerID := leechActionPlayerID(action)
	if playerID == "" {
		return false
	}
	if !hasPendingLeechOffer(r.state, playerID) {
		return false
	}
	pd := asMap(r.state["pendingDecision"])
	switch asString(pd["type"]) {
	case "town_cult_top_choice", "town_tile_selection", "darklings_ordination":
		return false
	default:
		return true
	}
}

func leechActionPlayerID(action game.Action) string {
	switch a := action.(type) {
	case *notation.LogAcceptLeechAction:
		return strings.TrimSpace(a.PlayerID)
	case *notation.LogDeclineLeechAction:
		return strings.TrimSpace(a.PlayerID)
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if playerID := leechActionPlayerID(nested); playerID != "" {
				return playerID
			}
		}
	case *notation.LogPreIncomeAction:
		return leechActionPlayerID(a.Action)
	case *notation.LogPostIncomeAction:
		return leechActionPlayerID(a.Action)
	}
	return ""
}

func findCultistAdvanceInAction(action game.Action, playerID string) *notation.LogCultistAdvanceAction {
	switch a := action.(type) {
	case *notation.LogCultistAdvanceAction:
		if strings.TrimSpace(a.PlayerID) != playerID {
			return nil
		}
		return a
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if found := findCultistAdvanceInAction(nested, playerID); found != nil {
				return found
			}
		}
	case *notation.LogPreIncomeAction:
		return findCultistAdvanceInAction(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return findCultistAdvanceInAction(a.Action, playerID)
	}
	return nil
}

func findPendingFavorTileAction(playerID string, upcoming []game.Action) *notation.LogFavorTileAction {
	for _, action := range upcoming {
		if found := findFavorTileInAction(action, strings.TrimSpace(playerID)); found != nil {
			return found
		}
	}
	return nil
}

func findFavorTileInAction(action game.Action, playerID string) *notation.LogFavorTileAction {
	switch a := action.(type) {
	case *notation.LogFavorTileAction:
		if strings.TrimSpace(a.PlayerID) == playerID {
			return a
		}
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if found := findFavorTileInAction(nested, playerID); found != nil {
				return found
			}
		}
	case *notation.LogPreIncomeAction:
		return findFavorTileInAction(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return findFavorTileInAction(a.Action, playerID)
	}
	return nil
}

func actionRequiresTurnOwnership(action game.Action) bool {
	if action == nil {
		return true
	}

	actionType := action.GetType()
	switch actionType {
	case game.ActionSelectFavorTile,
		game.ActionSelectTownTile,
		game.ActionSelectTownCultTop,
		game.ActionUseDarklingsPriestOrdination,
		game.ActionApplyHalflingsSpade,
		game.ActionBuildHalflingsDwelling,
		game.ActionSkipHalflingsDwelling,
		game.ActionSelectCultistsCultTrack,
		game.ActionDiscardPendingSpade,
		game.ActionSetPlayerOptions:
		return false
	default:
		return true
	}
}

func upcomingActionResolvesPending(decisionType string, upcoming []game.Action, playerID string) bool {
	if len(upcoming) == 0 {
		return false
	}
	playerID = strings.TrimSpace(playerID)

	switch decisionType {
	case "setup_bonus_card":
		return actionResolvesPendingDecision(decisionType, playerID, upcoming[0])
	case "cultists_cult_choice":
		for _, action := range upcoming {
			if actionResolvesPendingDecision(decisionType, playerID, action) {
				return true
			}
		}
	case "favor_tile_selection":
		return actionResolvesPendingDecision(decisionType, playerID, upcoming[0])
	case "cult_reward_spade":
		return actionResolvesPendingDecision(decisionType, playerID, upcoming[0])
	case "leech_offer":
		return actionResolvesPendingDecision(decisionType, playerID, upcoming[0])
	case "town_tile_selection":
		return actionResolvesPendingDecision(decisionType, playerID, upcoming[0])
	case "town_cult_top_choice":
		return actionResolvesPendingDecision(decisionType, playerID, upcoming[0])
	case "darklings_ordination":
		return actionResolvesPendingDecision(decisionType, playerID, upcoming[0])
	}

	return false
}

func actionResolvesPendingDecision(decisionType, playerID string, action game.Action) bool {
	if action == nil {
		return false
	}
	playerID = strings.TrimSpace(playerID)

	switch v := action.(type) {
	case *notation.LogCompoundAction:
		if len(v.Actions) == 0 {
			return false
		}
		return actionResolvesPendingDecision(decisionType, playerID, v.Actions[0])
	case *notation.LogPreIncomeAction:
		return actionResolvesPendingDecision(decisionType, playerID, v.Action)
	case *notation.LogPostIncomeAction:
		return actionResolvesPendingDecision(decisionType, playerID, v.Action)
	}

	switch decisionType {
	case "setup_bonus_card":
		if action, ok := action.(*notation.LogBonusCardSelectionAction); ok {
			return strings.TrimSpace(action.PlayerID) == playerID
		}
	case "cultists_cult_choice":
		return findCultistAdvanceInAction(action, playerID) != nil
	case "favor_tile_selection":
		return findFavorTileInAction(action, playerID) != nil
	case "cult_reward_spade":
		return actionContainsCultRewardSpadeUse(action, playerID)
	case "leech_offer":
		switch action := action.(type) {
		case *notation.LogAcceptLeechAction:
			return strings.TrimSpace(action.PlayerID) == playerID
		case *notation.LogDeclineLeechAction:
			return strings.TrimSpace(action.PlayerID) == playerID
		}
	case "town_tile_selection":
		return findTownActionInAction(action, playerID) != nil
	case "town_cult_top_choice":
		return findTownCultTopChoiceInAction(action, playerID) != nil
	case "darklings_ordination":
		if action, ok := action.(*game.UseDarklingsPriestOrdinationAction); ok {
			return strings.TrimSpace(action.PlayerID) == playerID
		}
	}

	return false
}

func findTownActionInAction(action game.Action, playerID string) *notation.LogTownAction {
	switch a := action.(type) {
	case *notation.LogTownAction:
		if strings.TrimSpace(a.PlayerID) == playerID {
			return a
		}
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if found := findTownActionInAction(nested, playerID); found != nil {
				return found
			}
		}
	case *notation.LogPreIncomeAction:
		return findTownActionInAction(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return findTownActionInAction(a.Action, playerID)
	}
	return nil
}

func findTownCultTopChoiceInAction(action game.Action, playerID string) *notation.LogCultTrackDecreaseAction {
	switch a := action.(type) {
	case *notation.LogCultTrackDecreaseAction:
		if strings.TrimSpace(a.PlayerID) == playerID {
			return a
		}
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if found := findTownCultTopChoiceInAction(nested, playerID); found != nil {
				return found
			}
		}
	case *notation.LogPreIncomeAction:
		return findTownCultTopChoiceInAction(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return findTownCultTopChoiceInAction(a.Action, playerID)
	}
	return nil
}

func canContinueFreeActionBeforeLeech(state map[string]any, upcoming []game.Action) bool {
	if len(upcoming) == 0 {
		return false
	}
	current := upcoming[0]
	if strings.TrimSpace(currentTurnPlayerID(state)) != strings.TrimSpace(current.GetPlayerID()) {
		return false
	}
	if !isReorderableFreeAction(current, current.GetPlayerID()) {
		return false
	}
	return true
}

func isCompoundDigFollowerPreserver(action game.Action) bool {
	switch action.(type) {
	case *notation.LogConversionAction, *notation.LogBurnAction:
		return true
	default:
		return false
	}
}

func compoundActionLocationKey(playerID string, coord board.Hex) string {
	return fmt.Sprintf("%s|%d|%d", strings.TrimSpace(playerID), coord.Q, coord.R)
}

func hasLaterCompoundTransformAction(actions []game.Action, playerID string, target board.Hex) bool {
	key := compoundActionLocationKey(playerID, target)
	for _, action := range actions {
		transform, ok := action.(*game.TransformAndBuildAction)
		if !ok {
			continue
		}
		if compoundActionLocationKey(transform.PlayerID, transform.TargetHex) == key {
			return true
		}
	}
	return false
}

func actionContainsCultRewardSpadeUse(action game.Action, playerID string) bool {
	if action == nil {
		return false
	}
	switch v := action.(type) {
	case *notation.LogPreIncomeAction:
		return actionContainsCultRewardSpadeUse(v.Action, playerID)
	case *notation.LogPostIncomeAction:
		return actionContainsCultRewardSpadeUse(v.Action, playerID)
	case *notation.LogCompoundAction:
		for _, sub := range v.Actions {
			if actionContainsCultRewardSpadeUse(sub, playerID) {
				return true
			}
		}
		return false
	case *game.TransformAndBuildAction:
		return strings.TrimSpace(v.PlayerID) == strings.TrimSpace(playerID) && !v.BuildDwelling
	default:
		return false
	}
}

func (r *goldenRunner) perform(playerID, actionType string, params map[string]any) error {
	if params == nil {
		params = map[string]any{}
	}
	r.recordAction(playerID, actionType, params)

	conn := r.clients[playerID]
	if conn == nil {
		return fmt.Errorf("missing websocket client for player %s", playerID)
	}
	expectedRevision := asInt(r.state["revision"])
	actionID := fmt.Sprintf("golden-%04d-%s-%s", r.step, playerID, actionType)
	if testing.Verbose() {
		pd := asMap(r.state["pendingDecision"])
		playerState := asMap(asMap(r.state["players"])[playerID])
		res := asMap(playerState["resources"])
		power := asMap(res["power"])
		cults := asMap(playerState["cults"])
		pendingSpadesForPlayer := asInt(asMap(r.state["pendingSpades"])[playerID])
		pendingCultSpadesForPlayer := asInt(asMap(r.state["pendingCultRewardSpades"])[playerID])
		r.t.Logf(
			"golden perform step=%d actionID=%s player=%s type=%s turn=%s phase=%d pendingType=%s pendingPlayer=%s vp=%d c=%d w=%d p=%d pw=%d/%d/%d cult=%d/%d/%d/%d pendingSpades=%d pendingCultSpades=%d params=%v",
			r.step,
			actionID,
			playerID,
			actionType,
			currentTurnPlayerID(r.state),
			asInt(r.state["phase"]),
			asString(pd["type"]),
			asString(pd["playerId"]),
			asInt(playerState["victoryPoints"]),
			asInt(res["coins"]),
			asInt(res["workers"]),
			asInt(res["priests"]),
			asInt(power["powerI"]),
			asInt(power["powerII"]),
			asInt(power["powerIII"]),
			asInt(cults["0"]),
			asInt(cults["1"]),
			asInt(cults["2"]),
			asInt(cults["3"]),
			pendingSpadesForPlayer,
			pendingCultSpadesForPlayer,
			params,
		)
	}
	r.step++

	sendJSON(r.t, conn, map[string]any{
		"type": "perform_action",
		"payload": map[string]any{
			"type":             actionType,
			"gameID":           r.gameID,
			"actionId":         actionID,
			"expectedRevision": expectedRevision,
			"params":           params,
		},
	})

	_ = readUntilType(r.t, conn, "action_accepted", 6*time.Second)
	r.state = readUntilStateRevisionAtLeast(r.t, conn, expectedRevision+1, 6*time.Second)
	if testing.Verbose() {
		players := asMap(r.state["players"])
		passed := map[string]bool{}
		for id, raw := range players {
			passed[id] = asBool(asMap(raw)["hasPassed"])
		}
		r.t.Logf(
			"golden post step=%d actionID=%s rev=%d turn=%s phase=%d round=%d pendingType=%s pendingPlayer=%s passed=%v playerState=%s gsState=%s",
			r.step-1,
			actionID,
			asInt(r.state["revision"]),
			currentTurnPlayerID(r.state),
			asInt(r.state["phase"]),
			asInt(asMap(r.state["round"])["round"]),
			asString(asMap(r.state["pendingDecision"])["type"]),
			asString(asMap(r.state["pendingDecision"])["playerId"]),
			passed,
			r.playerResourceSummary(r.state, playerID),
			r.playerResourceSummaryFromGS(playerID),
		)
	}
	return nil
}

func (r *goldenRunner) performSynthetic(playerID, actionType string, params map[string]any) error {
	if params == nil {
		params = map[string]any{}
	}

	conn := r.clients[playerID]
	if conn == nil {
		return fmt.Errorf("missing websocket client for player %s", playerID)
	}
	expectedRevision := asInt(r.state["revision"])
	actionID := fmt.Sprintf("golden-auto-%04d-%s-%s", r.step, playerID, actionType)

	sendJSON(r.t, conn, map[string]any{
		"type": "perform_action",
		"payload": map[string]any{
			"type":             actionType,
			"gameID":           r.gameID,
			"actionId":         actionID,
			"expectedRevision": expectedRevision,
			"params":           params,
		},
	})

	_ = readUntilType(r.t, conn, "action_accepted", 6*time.Second)
	r.state = readUntilStateRevisionAtLeast(r.t, conn, expectedRevision+1, 6*time.Second)
	return nil
}

func (r *goldenRunner) recordAction(playerID, actionType string, params map[string]any) {
	if params == nil {
		params = map[string]any{}
	}
	r.recordedActions = append(r.recordedActions, goldenExportAction{
		PlayerID: playerID,
		Type:     actionType,
		Params:   cloneAnyMap(params),
	})
}

func (r *goldenRunner) recordReplayConversion(playerID string, conversionType game.ConversionType, amount int) error {
	if amount <= 0 {
		return nil
	}
	if strings.TrimSpace(playerID) == "" || conversionType == "" {
		return fmt.Errorf("missing conversion player or type")
	}

	params := map[string]any{
		"conversionType": string(conversionType),
		"amount":         amount,
	}
	if testing.Verbose() && strings.EqualFold(strings.TrimSpace(playerID), "Witches") {
		r.t.Logf(
			"golden debug replay_conversion before step=%d player=%s type=%s amount=%d state=%s gs=%s",
			r.step,
			strings.TrimSpace(playerID),
			conversionType,
			amount,
			r.playerResourceSummary(r.state, playerID),
			r.playerResourceSummaryFromGS(playerID),
		)
	}

	recordAsReplay := false
	action := &game.ConversionAction{
		BaseAction:     game.BaseAction{Type: game.ActionConversion, PlayerID: playerID},
		ConversionType: conversionType,
		Amount:         amount,
	}

	_, err := r.deps.ExecuteActionWithMeta(r.gameID, action, game.ActionMeta{
		ActionID:         fmt.Sprintf("golden-%04d-%s-conversion", r.step, playerID),
		ExpectedRevision: asInt(r.state["revision"]),
		SeatID:           playerID,
	})
	if err != nil {
		if !isTurnValidationFailure(err) {
			return fmt.Errorf("conversion action failed: %w", err)
		}
		recordAsReplay = true
		if _, err := r.deps.ApplyConversionWithoutTurnCheck(r.gameID, playerID, conversionType, amount); err != nil {
			return fmt.Errorf("replay conversion failed: %w", err)
		}
	}

	actionType := "conversion"
	if recordAsReplay {
		actionType = "replay_conversion"
	}
	r.recordAction(playerID, actionType, params)
	r.step++

	gameState := r.deps.SerializeGameState(r.gameID)
	if gameState == nil {
		return fmt.Errorf("failed to serialize game state")
	}
	r.state = asMap(gameState)
	if testing.Verbose() && strings.EqualFold(strings.TrimSpace(playerID), "Witches") {
		r.t.Logf(
			"golden debug replay_conversion after step=%d player=%s state=%s gs=%s",
			r.step,
			strings.TrimSpace(playerID),
			r.playerResourceSummary(r.state, playerID),
			r.playerResourceSummaryFromGS(playerID),
		)
	}

	return nil
}

func (r *goldenRunner) playerResourceSummary(state map[string]any, playerID string) string {
	playerData := asMap(asMap(state["players"])[playerID])
	if len(playerData) == 0 {
		return "missing"
	}
	res := asMap(playerData["resources"])
	power := asMap(res["power"])
	return fmt.Sprintf(
		"vp=%d c=%d w=%d p=%d pw=%d/%d/%d",
		asInt(playerData["victoryPoints"]),
		asInt(res["coins"]),
		asInt(res["workers"]),
		asInt(res["priests"]),
		asInt(power["powerI"]),
		asInt(power["powerII"]),
		asInt(power["powerIII"]),
	)
}

func (r *goldenRunner) playerResourceSummaryFromGS(playerID string) string {
	player := r.gs.GetPlayer(playerID)
	if player == nil {
		return "missing"
	}
	res := player.Resources
	var pw3, pw2, pw1 int
	if res.Power != nil {
		pw1 = res.Power.Bowl1
		pw2 = res.Power.Bowl2
		pw3 = res.Power.Bowl3
	}
	return fmt.Sprintf(
		"vp=%d c=%d w=%d p=%d pw=%d/%d/%d",
		player.VictoryPoints,
		res.Coins,
		res.Workers,
		res.Priests,
		pw1,
		pw2,
		pw3,
	)
}

func (r *goldenRunner) recordReplayDarklingsOrdination(playerID string, workersToConvert int) error {
	if workersToConvert <= 0 {
		return nil
	}
	if strings.TrimSpace(playerID) == "" {
		return fmt.Errorf("missing darklings ordination player")
	}
	if workersToConvert > 3 {
		return fmt.Errorf("too many workers for darklings ordination: %d", workersToConvert)
	}

	params := map[string]any{
		"workersToConvert": workersToConvert,
	}
	action := &game.UseDarklingsPriestOrdinationAction{
		BaseAction:       game.BaseAction{Type: game.ActionUseDarklingsPriestOrdination, PlayerID: playerID},
		WorkersToConvert: workersToConvert,
	}

	_, err := r.deps.ExecuteActionWithMeta(r.gameID, action, game.ActionMeta{
		ActionID:         fmt.Sprintf("golden-%04d-%s-darklings_ordination", r.step, playerID),
		ExpectedRevision: asInt(r.state["revision"]),
		SeatID:           playerID,
	})
	if err != nil {
		if isTurnValidationFailure(err) {
			return fmt.Errorf("darklings ordination turn mismatch: %w", err)
		}
		return fmt.Errorf("darklings ordination action failed: %w", err)
	}
	r.recordAction(playerID, "darklings_ordination", params)

	gameState := r.deps.SerializeGameState(r.gameID)
	if gameState == nil {
		return fmt.Errorf("failed to serialize game state")
	}
	r.state = asMap(gameState)
	r.step++

	return nil
}

func isTurnValidationFailure(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "action turn validation failed")
}

func cloneAnyMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = cloneAnyValue(v)
	}
	return out
}

func cloneAnyValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneAnyMap(x)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = cloneAnyValue(x[i])
		}
		return out
	default:
		return x
	}
}

func cloneIntMap(input map[string]int) map[string]int {
	out := make(map[string]int, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func (r *goldenRunner) ensurePendingCultRewardSpadesForLeadingPreIncomeTransforms(playerID string, upcoming []game.Action) {
	if strings.TrimSpace(playerID) == "" || len(upcoming) == 0 {
		return
	}
	pd := asMap(r.state["pendingDecision"])
	if asString(pd["type"]) != "cult_reward_spade" || asString(pd["playerId"]) != strings.TrimSpace(playerID) {
		return
	}

	required := countLeadingPreIncomeTransforms(upcoming, playerID)
	if required <= 0 {
		return
	}
	current := asInt(pd["spadesRemaining"])
	if current >= required {
		return
	}

	if r.gs.PendingCultRewardSpades == nil {
		r.gs.PendingCultRewardSpades = make(map[string]int)
	}
	r.gs.PendingCultRewardSpades[playerID] += required - current

	pd["spadesRemaining"] = required
	r.state["pendingDecision"] = pd
}

func countLeadingPreIncomeTransforms(actions []game.Action, playerID string) int {
	count := 0
	for _, action := range actions {
		pre, ok := action.(*notation.LogPreIncomeAction)
		if !ok || pre.Action == nil {
			break
		}
		transform, ok := pre.Action.(*game.TransformAndBuildAction)
		if !ok {
			break
		}
		if strings.TrimSpace(transform.PlayerID) != strings.TrimSpace(playerID) {
			break
		}
		if transform.BuildDwelling {
			break
		}
		count++
	}
	return count
}

func (r *goldenRunner) tryAutoFundUpgradeByPowerToCoin(action *game.UpgradeBuildingAction) error {
	if action == nil {
		return nil
	}
	for attempts := 0; attempts < 8; attempts++ {
		if err := action.Validate(r.gs); err == nil {
			return nil
		} else if !strings.Contains(strings.ToLower(err.Error()), "cannot afford") {
			return nil
		}

		playerState := asMap(asMap(r.state["players"])[action.PlayerID])
		resources := asMap(playerState["resources"])
		power := asMap(resources["power"])
		if asInt(power["powerIII"]) <= 0 {
			return nil
		}

		if err := r.perform(action.PlayerID, "conversion", map[string]any{
			"conversionType": string(game.ConversionPowerToCoin),
			"amount":         1,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *goldenRunner) getUpgradeActionCost(playerID string, action *game.UpgradeBuildingAction) (coins, workers, priests int) {
	if action == nil {
		return 0, 0, 0
	}
	player := r.gs.GetPlayer(playerID)
	if player == nil {
		return 0, 0, 0
	}

	switch action.NewBuildingType {
	case models.BuildingTradingHouse:
		cost := player.Faction.GetTradingHouseCost()
		if r.isAdjacentToOpponent(action.TargetHex, playerID) {
			cost.Coins /= 2
		}
		return cost.Coins, cost.Workers, cost.Priests
	case models.BuildingTemple:
		cost := player.Faction.GetTempleCost()
		return cost.Coins, cost.Workers, cost.Priests
	case models.BuildingSanctuary:
		cost := player.Faction.GetSanctuaryCost()
		return cost.Coins, cost.Workers, cost.Priests
	case models.BuildingStronghold:
		cost := player.Faction.GetStrongholdCost()
		return cost.Coins, cost.Workers, cost.Priests
	default:
		return 0, 0, 0
	}
}

func (r *goldenRunner) isAdjacentToOpponent(targetHex board.Hex, playerID string) bool {
	mapHex := r.gs.Map.GetHex(targetHex)
	if mapHex == nil {
		return false
	}
	for _, neighbor := range mapHex.Coord.Neighbors() {
		neighborHex := r.gs.Map.GetHex(neighbor)
		if neighborHex == nil || neighborHex.Building == nil {
			continue
		}
		if neighborHex.Building.PlayerID != strings.TrimSpace(playerID) {
			return true
		}
	}
	return false
}

func (r *goldenRunner) tryAutoFundUpgradeByCost(
	playerID string,
	getCost func(player *game.Player) (int, int, int),
) error {
	if getCost == nil {
		return nil
	}
	for attempts := 0; attempts < 10; attempts++ {
		player := r.gs.GetPlayer(playerID)
		if player == nil {
			return errGoldenCannotAutoFundUpgrade
		}
		needCoins, needWorkers, needPriests := getCost(player)
		if testing.Verbose() {
			p := player.Resources
			cost3 := 0
			if p.Power != nil {
				cost3 = p.Power.Bowl3
			}
			r.t.Logf(
				"golden auto-fund upgrade cost idx=%d player=%s need={c:%d w:%d p:%d} have={c:%d w:%d p:%d pw3:%d}",
				len(r.recordedActions),
				playerID,
				needCoins,
				needWorkers,
				needPriests,
				p.Coins,
				p.Workers,
				p.Priests,
				cost3,
			)
		}
		if player.Resources.Coins >= needCoins &&
			player.Resources.Workers >= needWorkers &&
			player.Resources.Priests >= needPriests {
			return nil
		}

		if player.Resources.Priests < needPriests {
			amount := needPriests - player.Resources.Priests
			amount, err := r.prepareConversionAmount(playerID, game.ConversionPowerToPriest, amount)
			if err != nil {
				return err
			}
			if amount > 0 {
				if err := r.perform(playerID, "conversion", map[string]any{
					"conversionType": string(game.ConversionPowerToPriest),
					"amount":         amount,
				}); err != nil {
					return err
				}
				continue
			}
			return errGoldenCannotAutoFundUpgrade
		}

		if player.Resources.Workers < needWorkers {
			amount := needWorkers - player.Resources.Workers
			if amount > 0 {
				amount, err := r.prepareConversionAmount(playerID, game.ConversionPowerToWorker, amount)
				if err != nil {
					return err
				}
				if amount > 0 {
					if err := r.perform(playerID, "conversion", map[string]any{
						"conversionType": string(game.ConversionPowerToWorker),
						"amount":         amount,
					}); err != nil {
						return err
					}
					continue
				}
			}
			if player.Resources.Priests > 0 {
				if err := r.perform(playerID, "conversion", map[string]any{
					"conversionType": string(game.ConversionPriestToWorker),
					"amount":         1,
				}); err != nil {
					return err
				}
				continue
			}
			return errGoldenCannotAutoFundUpgrade
		}

		if player.Resources.Coins < needCoins {
			amount := needCoins - player.Resources.Coins
			if amount > 0 {
				amount, err := r.prepareConversionAmount(playerID, game.ConversionPowerToCoin, amount)
				if err != nil {
					return err
				}
				if amount > 0 {
					if err := r.perform(playerID, "conversion", map[string]any{
						"conversionType": string(game.ConversionPowerToCoin),
						"amount":         amount,
					}); err != nil {
						return err
					}
					continue
				}
			}
			if player.Resources.Workers > 0 {
				if err := r.perform(playerID, "conversion", map[string]any{
					"conversionType": string(game.ConversionWorkerToCoin),
					"amount":         1,
				}); err != nil {
					return err
				}
				continue
			}
			return errGoldenCannotAutoFundUpgrade
		}

		return nil
	}
	return errGoldenCannotAutoFundUpgrade
}

func (r *goldenRunner) tryAutoFundTransformBuild(action *game.TransformAndBuildAction) error {
	if action == nil {
		return nil
	}
	for attempts := 0; attempts < 8; attempts++ {
		validateErr := action.Validate(r.gs)
		if validateErr == nil {
			return nil
		}

		msg := strings.ToLower(validateErr.Error())
		isWorkersMissing := strings.Contains(msg, "not enough workers")
		isDwellingMissing := strings.Contains(msg, "not enough resources for dwelling")
		if !isWorkersMissing && !isDwellingMissing && !strings.Contains(msg, "not enough priests for terraform") {
			return nil
		}

		player := r.gs.GetPlayer(action.PlayerID)
		if player == nil {
			return nil
		}
		playerState := asMap(asMap(r.state["players"])[action.PlayerID])
		resources := asMap(playerState["resources"])

		if isDwellingMissing {
			requiredWorkers, estimateErr := r.estimatePowerSpadeWorkersNeeded(action.PlayerID, game.PowerActionSpade1, action)
			if estimateErr != nil {
				return nil
			}
			requiredCoins := 0
			if action.BuildDwelling {
				requiredCoins = player.Faction.GetDwellingCost().Coins
			}

			if player.Resources.Workers < requiredWorkers {
				if player.Resources.Priests > 0 {
					if err := r.perform(action.PlayerID, "conversion", map[string]any{
						"conversionType": string(game.ConversionPriestToWorker),
						"amount":         1,
					}); err != nil {
						return err
					}
					continue
				}
				if player.Resources.Power != nil && player.Resources.Power.Bowl3 >= 3 {
					if err := r.perform(action.PlayerID, "conversion", map[string]any{
						"conversionType": string(game.ConversionPowerToWorker),
						"amount":         1,
					}); err != nil {
						return err
					}
					continue
				}
				return nil
			}
			if player.Resources.Coins < requiredCoins {
				if player.Resources.Power != nil && player.Resources.Power.Bowl3 >= 1 {
					if err := r.perform(action.PlayerID, "conversion", map[string]any{
						"conversionType": string(game.ConversionPowerToCoin),
						"amount":         1,
					}); err != nil {
						return err
					}
					continue
				}
				if player.Resources.Workers > 0 {
					if err := r.perform(action.PlayerID, "conversion", map[string]any{
						"conversionType": string(game.ConversionWorkerToCoin),
						"amount":         1,
					}); err != nil {
						return err
					}
					continue
				}
				return nil
			}

			return nil
		}

		if player.Resources.Power != nil && player.Resources.Power.Bowl3 >= 3 {
			if err := r.perform(action.PlayerID, "conversion", map[string]any{
				"conversionType": string(game.ConversionPowerToWorker),
				"amount":         1,
			}); err != nil {
				return err
			}
			continue
		}

		priests := asInt(resources["priests"])
		if priests > 0 {
			if err := r.perform(action.PlayerID, "conversion", map[string]any{
				"conversionType": string(game.ConversionPriestToWorker),
				"amount":         1,
			}); err != nil {
				return err
			}
			continue
		}

		return nil
	}
	return nil
}

func (r *goldenRunner) tryAutoFundPowerSpadeClaim(playerID string, actionType game.PowerActionType, transform *game.TransformAndBuildAction) error {
	if transform == nil {
		return nil
	}
	for attempts := 0; attempts < 10; attempts++ {
		requiredWorkers, err := r.estimatePowerSpadeWorkersNeeded(playerID, actionType, transform)
		if err != nil {
			return nil
		}
		player := r.gs.GetPlayer(playerID)
		if player == nil {
			return nil
		}
		if player.Resources.Workers >= requiredWorkers {
			return nil
		}
		deficit := requiredWorkers - player.Resources.Workers
		if deficit <= 0 {
			return nil
		}
		powerReserve := game.GetPowerCost(actionType)
		if player.Resources.Power.Bowl3 >= 3 && player.Resources.Power.Bowl3-3 >= powerReserve {
			if err := r.perform(playerID, "conversion", map[string]any{
				"conversionType": string(game.ConversionPowerToWorker),
				"amount":         1,
			}); err != nil {
				return err
			}
			continue
		}
		if player.Resources.Priests > 0 {
			if err := r.perform(playerID, "conversion", map[string]any{
				"conversionType": string(game.ConversionPriestToWorker),
				"amount":         1,
			}); err != nil {
				return err
			}
			continue
		}
		return nil
	}
	return nil
}

func (r *goldenRunner) estimatePowerSpadeWorkersNeeded(playerID string, actionType game.PowerActionType, transform *game.TransformAndBuildAction) (int, error) {
	if transform == nil {
		return 0, nil
	}
	player := r.gs.GetPlayer(playerID)
	if player == nil {
		return 0, fmt.Errorf("player not found: %s", playerID)
	}
	mapHex := r.gs.Map.GetHex(transform.TargetHex)
	if mapHex == nil {
		return 0, fmt.Errorf("target hex does not exist: %v", transform.TargetHex)
	}

	requiredWorkers := 0
	useSkip := transform.UseSkip
	isAdjacent := r.gs.IsAdjacentToPlayerBuilding(transform.TargetHex, playerID)
	if !isAdjacent && !useSkip {
		factionType := player.Faction.GetType()
		if factionType == models.FactionDwarves || factionType == models.FactionFakirs {
			useSkip = true
		}
	}
	if useSkip && player.Faction.GetType() == models.FactionDwarves {
		tunnelCost := 2
		if player.HasStrongholdAbility {
			tunnelCost = 1
		}
		requiredWorkers += tunnelCost
	}

	targetTerrain := player.Faction.GetHomeTerrain()
	if transform.TargetTerrain != models.TerrainTypeUnknown {
		targetTerrain = transform.TargetTerrain
	}
	requiredSpades := r.gs.Map.GetTerrainDistance(mapHex.Terrain, targetTerrain)
	if requiredSpades < 0 {
		requiredSpades = 0
	}
	if player.Faction.GetType() == models.FactionGiants && requiredSpades > 0 {
		requiredSpades = 2
	}
	freeSpades := 1
	if actionType == game.PowerActionSpade2 {
		freeSpades = 2
	}
	if freeSpades > requiredSpades {
		freeSpades = requiredSpades
	}
	remainingSpades := requiredSpades - freeSpades
	if remainingSpades > 0 && player.Faction.GetType() != models.FactionDarklings {
		requiredWorkers += player.Faction.GetTerraformCost(remainingSpades)
	}

	if transform.BuildDwelling {
		requiredWorkers += player.Faction.GetDwellingCost().Workers
	}

	return requiredWorkers, nil
}

func (r *goldenRunner) prepareConversionAmount(playerID string, convType game.ConversionType, amount int) (int, error) {
	if amount <= 0 {
		return 0, nil
	}

	powerCostPerUnit := 0
	switch convType {
	case game.ConversionPowerToCoin:
		powerCostPerUnit = 1
	case game.ConversionPowerToWorker:
		powerCostPerUnit = 3
	case game.ConversionPowerToPriest:
		powerCostPerUnit = 5
	default:
		return amount, nil
	}

	playerState := asMap(asMap(r.state["players"])[playerID])
	resources := asMap(playerState["resources"])
	power := asMap(resources["power"])
	power2 := asInt(power["powerII"])
	power3 := asInt(power["powerIII"])
	maxPower3Reachable := power3 + power2/2
	maxUnits := maxPower3Reachable / powerCostPerUnit
	if maxUnits <= 0 {
		return 0, nil
	}
	if amount > maxUnits {
		amount = maxUnits
	}

	requiredPower3 := amount * powerCostPerUnit
	if requiredPower3 <= power3 {
		return amount, nil
	}

	burnNeeded := requiredPower3 - power3
	if burnNeeded > 0 {
		if err := r.perform(playerID, "burn_power", map[string]any{"amount": burnNeeded}); err != nil {
			return 0, err
		}
	}

	return amount, nil
}

func (r *goldenRunner) performCultistsTrackChoice(playerID string, preferredTrack int) error {
	tryTracks := []int{preferredTrack, int(game.CultFire), int(game.CultWater), int(game.CultEarth), int(game.CultAir)}
	seen := map[int]bool{}
	ordered := make([]int, 0, len(tryTracks))
	for _, track := range tryTracks {
		if seen[track] {
			continue
		}
		seen[track] = true
		ordered = append(ordered, track)
	}

	players := asMap(r.state["players"])
	player := asMap(players[playerID])
	cults := asMap(player["cults"])
	for _, track := range ordered {
		if asInt(cults[fmt.Sprintf("%d", track)]) >= 10 {
			continue
		}
		return r.perform(playerID, "select_cultists_track", map[string]any{"track": track})
	}

	return fmt.Errorf("no legal cultists track choice for %s", playerID)
}

func (r *goldenRunner) executeStandalonePowerAction(action *notation.LogPowerAction) error {
	if action == nil {
		return fmt.Errorf("nil power action")
	}
	powerType := notation.ParsePowerActionCode(strings.ToUpper(strings.TrimSpace(action.ActionCode)))
	if powerType == game.PowerActionUnknown {
		return fmt.Errorf("unknown power action code: %s", action.ActionCode)
	}
	if err := r.burnToReachPowerActionCost(action.PlayerID, powerType); err != nil {
		return err
	}
	if powerType == game.PowerActionSpade1 || powerType == game.PowerActionSpade2 {
		return fmt.Errorf("standalone spade power action without transform follow-up: %s", action.ActionCode)
	}
	if powerType == game.PowerActionBridge {
		h1, h2, err := parseBridgeFromACT1(action.ActionCode)
		if err != nil {
			return err
		}
		return r.perform(action.PlayerID, "power_action_claim", map[string]any{
			"actionType": int(powerType),
			"bridgeHex1": toHexParam(h1),
			"bridgeHex2": toHexParam(h2),
		})
	}
	return r.perform(action.PlayerID, "power_action_claim", map[string]any{"actionType": int(powerType)})
}

func (r *goldenRunner) burnToReachPowerActionCost(playerID string, actionType game.PowerActionType) error {
	cost := game.GetPowerCost(actionType)
	if cost <= 0 {
		return nil
	}
	player := asMap(asMap(r.state["players"])[playerID])
	resources := asMap(player["resources"])
	power := asMap(resources["power"])
	bowl2 := asInt(power["powerII"])
	bowl3 := asInt(power["powerIII"])
	need := cost - bowl3
	if need <= 0 {
		return nil
	}
	maxBurn := bowl2 / 2
	if maxBurn <= 0 {
		return nil
	}
	if need > maxBurn {
		need = maxBurn
	}
	if need <= 0 {
		return nil
	}
	return r.perform(playerID, "burn_power", map[string]any{"amount": need})
}

func maxBurnPossible(state map[string]any, playerID string) int {
	player := asMap(asMap(state["players"])[playerID])
	resources := asMap(player["resources"])
	power := asMap(resources["power"])
	return asInt(power["powerII"]) / 2
}

func isReorderableFreeAction(action game.Action, playerID string) bool {
	switch a := action.(type) {
	case *notation.LogBurnAction:
		return strings.TrimSpace(a.PlayerID) == strings.TrimSpace(playerID)
	case *notation.LogConversionAction:
		return strings.TrimSpace(a.PlayerID) == strings.TrimSpace(playerID)
	default:
		return false
	}
}

func setupWebsocketGoldenGame(t *testing.T, playerIDs []string) (ServerDeps, *httptest.Server, string, map[string]*gws.Conn, map[string]any) {
	t.Helper()

	hub := NewHub()
	go hub.Run()

	deps := ServerDeps{
		Lobby: lobby.NewManager(),
		Games: game.NewManager(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, deps, w, r)
	}))

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clients := make(map[string]*gws.Conn, len(playerIDs))
	for _, playerID := range playerIDs {
		clients[playerID] = dialWS(t, wsURL)
	}

	creator := playerIDs[0]
	sendJSON(t, clients[creator], map[string]any{
		"type": "create_game",
		"payload": map[string]any{
			"name":       "golden-snellman",
			"maxPlayers": len(playerIDs),
			"creator":    creator,
		},
	})
	created := readUntilType(t, clients[creator], "game_created", 4*time.Second)
	gameID := asString(asMap(created["payload"])["gameId"])
	if gameID == "" {
		t.Fatalf("missing game id in create response")
	}
	_ = readUntilType(t, clients[creator], "lobby_state", 4*time.Second)

	for _, playerID := range playerIDs[1:] {
		sendJSON(t, clients[playerID], map[string]any{
			"type": "join_game",
			"payload": map[string]any{
				"id":   gameID,
				"name": playerID,
			},
		})
		_ = readUntilType(t, clients[playerID], "game_joined", 4*time.Second)
	}

	sendJSON(t, clients[creator], map[string]any{
		"type": "start_game",
		"payload": map[string]any{
			"gameID":             gameID,
			"randomizeTurnOrder": false,
		},
	})

	state := asMap(readUntilType(t, clients[creator], "game_state_update", 4*time.Second)["payload"])
	return deps, server, gameID, clients, state
}

func applyGoldenSettings(gs *game.GameState, settings map[string]string) error {
	if gs == nil {
		return fmt.Errorf("nil game state")
	}

	scoringCodes := splitCSV(settings["ScoringTiles"])
	if len(scoringCodes) != 6 {
		return fmt.Errorf("expected 6 scoring tile codes, got %d (%v)", len(scoringCodes), scoringCodes)
	}
	scoringTiles, err := scoringTilesFromCodes(scoringCodes)
	if err != nil {
		return err
	}
	gs.ScoringTiles.Tiles = scoringTiles

	bonusCodes := splitCSV(settings["BonusCards"])
	if len(bonusCodes) == 0 {
		return fmt.Errorf("missing BonusCards setting")
	}
	cards := make([]game.BonusCardType, 0, len(bonusCodes))
	for _, code := range bonusCodes {
		card := notation.ParseBonusCardCode(strings.ToUpper(strings.TrimSpace(code)))
		if card == game.BonusCardUnknown {
			return fmt.Errorf("unknown bonus card code: %s", code)
		}
		cards = append(cards, card)
	}
	gs.BonusCards.SetAvailableBonusCards(cards)

	return nil
}

func extractGoldenSettingsAndActions(items []notation.LogItem) (map[string]string, []game.Action) {
	settings := map[string]string{}
	actions := make([]game.Action, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case notation.GameSettingsItem:
			for k, val := range v.Settings {
				settings[k] = val
			}
		case notation.ActionItem:
			actions = append(actions, v.Action)
		}
	}
	return settings, actions
}

func splitCSV(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func scoringTilesFromCodes(codes []string) ([]game.ScoringTile, error) {
	typeByCode := map[string]game.ScoringTileType{
		"SCORE1": game.ScoringSpades,
		"SCORE2": game.ScoringTown,
		"SCORE3": game.ScoringDwellingWater,
		"SCORE4": game.ScoringStrongholdFire,
		"SCORE5": game.ScoringDwellingFire,
		"SCORE6": game.ScoringTradingHouseWater,
		"SCORE7": game.ScoringStrongholdAir,
		"SCORE8": game.ScoringTradingHouseAir,
		"SCORE9": game.ScoringTemplePriest,
	}

	all := game.GetAllScoringTiles()
	tileByType := make(map[game.ScoringTileType]game.ScoringTile, len(all))
	for _, tile := range all {
		tileByType[tile.Type] = tile
	}

	out := make([]game.ScoringTile, 0, len(codes))
	for _, code := range codes {
		typeVal, ok := typeByCode[strings.ToUpper(strings.TrimSpace(code))]
		if !ok {
			return nil, fmt.Errorf("unknown scoring tile code: %s", code)
		}
		tile, ok := tileByType[typeVal]
		if !ok {
			return nil, fmt.Errorf("missing scoring tile type in registry: %v", typeVal)
		}
		out = append(out, tile)
	}

	return out, nil
}

func transformBuildParams(action *game.TransformAndBuildAction) map[string]any {
	params := map[string]any{
		"targetHex":     toHexParam(action.TargetHex),
		"buildDwelling": action.BuildDwelling,
	}
	if action.TargetTerrain != models.TerrainTypeUnknown {
		params["targetTerrain"] = int(action.TargetTerrain)
	}
	if action.UseSkip {
		params["useSkip"] = true
	}
	return params
}

func toHexParam(hex board.Hex) map[string]any {
	return map[string]any{"q": hex.Q, "r": hex.R}
}

func mapLogConversion(action *notation.LogConversionAction) (game.ConversionType, int, error) {
	if action == nil {
		return "", 0, fmt.Errorf("nil conversion action")
	}

	costKinds := nonZeroResourceKinds(action.Cost)
	rewardKinds := nonZeroResourceKinds(action.Reward)
	if len(costKinds) != 1 || len(rewardKinds) != 1 {
		return "", 0, fmt.Errorf("unsupported multi-resource conversion: cost=%v reward=%v", action.Cost, action.Reward)
	}

	costKind := costKinds[0]
	rewardKind := rewardKinds[0]
	costAmt := action.Cost[costKind]
	rewardAmt := action.Reward[rewardKind]
	if costAmt <= 0 || rewardAmt <= 0 {
		return "", 0, fmt.Errorf("invalid conversion amounts: cost=%v reward=%v", action.Cost, action.Reward)
	}

	switch {
	case costKind == models.ResourcePower && rewardKind == models.ResourceCoin && costAmt == rewardAmt:
		return game.ConversionPowerToCoin, rewardAmt, nil
	case costKind == models.ResourcePower && rewardKind == models.ResourceWorker && costAmt == rewardAmt*3:
		return game.ConversionPowerToWorker, rewardAmt, nil
	case costKind == models.ResourcePower && rewardKind == models.ResourcePriest && costAmt == rewardAmt*5:
		return game.ConversionPowerToPriest, rewardAmt, nil
	case costKind == models.ResourcePriest && rewardKind == models.ResourceWorker && costAmt == rewardAmt:
		return game.ConversionPriestToWorker, rewardAmt, nil
	case costKind == models.ResourceWorker && rewardKind == models.ResourceCoin && costAmt == rewardAmt:
		return game.ConversionWorkerToCoin, rewardAmt, nil
	case costKind == models.ResourceVictoryPoint && rewardKind == models.ResourceCoin && costAmt == rewardAmt:
		return game.ConversionAlchVPToCoin, rewardAmt, nil
	case costKind == models.ResourceCoin && rewardKind == models.ResourceVictoryPoint && costAmt == rewardAmt*2:
		return game.ConversionAlchCoinToVP, rewardAmt, nil
	default:
		return "", 0, fmt.Errorf("unsupported conversion shape: cost=%v reward=%v", action.Cost, action.Reward)
	}
}

func (r *goldenRunner) synthesizeMermaidsPendingTown(playerID string) bool {
	if r.gs == nil || r.gs.Map == nil {
		return false
	}
	player := r.gs.GetPlayer(playerID)
	if player == nil || player.Faction.GetType() != models.FactionMermaids {
		return false
	}
	if pending := r.gs.PendingTownFormations[playerID]; len(pending) > 0 {
		return true
	}

	type componentCandidate struct {
		hexes   []board.Hex
		power   int
		river   *board.Hex
		hasTown bool
	}
	minPower := r.gs.GetTownPowerRequirement(playerID)
	candidates := make([]componentCandidate, 0, 8)
	seen := map[string]bool{}

	for _, mapHex := range r.gs.Map.Hexes {
		if mapHex == nil || mapHex.Building == nil || mapHex.Building.PlayerID != playerID {
			continue
		}
		component, river := r.gs.Map.GetConnectedBuildingsForMermaids(mapHex.Coord, playerID)
		if len(component) == 0 {
			continue
		}
		key := componentKey(component)
		if seen[key] {
			continue
		}
		seen[key] = true
		hasTown := componentContainsTownHex(r.gs, component)
		power := componentPower(r.gs, component)
		candidates = append(candidates, componentCandidate{hexes: component, power: power, river: river, hasTown: hasTown})
	}

	if len(candidates) == 0 {
		return false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].power != candidates[j].power {
			return candidates[i].power > candidates[j].power
		}
		return len(candidates[i].hexes) > len(candidates[j].hexes)
	})

	pickBest := func(allowTowned bool) (componentCandidate, bool) {
		for _, c := range candidates {
			if c.power < minPower {
				continue
			}
			if !allowTowned && c.hasTown {
				continue
			}
			return c, true
		}
		return componentCandidate{}, false
	}
	best, ok := pickBest(false)
	if !ok {
		best, ok = pickBest(true)
		if !ok {
			return false
		}
	}
	r.gs.PendingTownFormations[playerID] = append(r.gs.PendingTownFormations[playerID], &game.PendingTownFormation{
		PlayerID:        playerID,
		Hexes:           best.hexes,
		SkippedRiverHex: best.river,
		CanBeDelayed:    best.river != nil,
	})
	return true
}

func componentKey(component []board.Hex) string {
	if len(component) == 0 {
		return ""
	}
	copyComponent := make([]board.Hex, len(component))
	copy(copyComponent, component)
	sort.Slice(copyComponent, func(i, j int) bool {
		if copyComponent[i].Q != copyComponent[j].Q {
			return copyComponent[i].Q < copyComponent[j].Q
		}
		return copyComponent[i].R < copyComponent[j].R
	})
	parts := make([]string, 0, len(copyComponent))
	for _, h := range copyComponent {
		parts = append(parts, fmt.Sprintf("%d,%d", h.Q, h.R))
	}
	return strings.Join(parts, "|")
}

func componentContainsTownHex(gs *game.GameState, component []board.Hex) bool {
	for _, h := range component {
		mapHex := gs.Map.GetHex(h)
		if mapHex != nil && mapHex.PartOfTown {
			return true
		}
	}
	return false
}

func componentPower(gs *game.GameState, component []board.Hex) int {
	total := 0
	for _, h := range component {
		mapHex := gs.Map.GetHex(h)
		if mapHex == nil || mapHex.Building == nil {
			continue
		}
		total += game.GetPowerValue(mapHex.Building.Type)
	}
	return total
}

func nonZeroResourceKinds(m map[models.ResourceType]int) []models.ResourceType {
	out := make([]models.ResourceType, 0, len(m))
	for k, v := range m {
		if v > 0 {
			out = append(out, k)
		}
	}
	return out
}

func detectPriestToCoinConversion(action *notation.LogConversionAction) (int, bool) {
	if action == nil {
		return 0, false
	}
	costKinds := nonZeroResourceKinds(action.Cost)
	rewardKinds := nonZeroResourceKinds(action.Reward)
	if len(costKinds) != 1 || len(rewardKinds) != 1 {
		return 0, false
	}
	if costKinds[0] != models.ResourcePriest || rewardKinds[0] != models.ResourceCoin {
		return 0, false
	}
	costAmt := action.Cost[models.ResourcePriest]
	rewardAmt := action.Reward[models.ResourceCoin]
	if costAmt <= 0 || rewardAmt <= 0 || costAmt != rewardAmt {
		return 0, false
	}
	return costAmt, true
}

func detectDarklingsWorkerToPriestConversion(action *notation.LogConversionAction) (int, bool) {
	if action == nil {
		return 0, false
	}
	costKinds := nonZeroResourceKinds(action.Cost)
	rewardKinds := nonZeroResourceKinds(action.Reward)
	if len(costKinds) != 1 || len(rewardKinds) != 1 {
		return 0, false
	}
	if costKinds[0] != models.ResourceWorker || rewardKinds[0] != models.ResourcePriest {
		return 0, false
	}
	costAmt := action.Cost[models.ResourceWorker]
	rewardAmt := action.Reward[models.ResourcePriest]
	if costAmt <= 0 || rewardAmt <= 0 || costAmt != rewardAmt {
		return 0, false
	}
	return costAmt, true
}

func findLeechOfferIndex(state map[string]any, playerID, fromPlayerID string, amount int, strictAmount bool) (int, error) {
	pending := asMap(state["pendingLeechOffers"])
	offersRaw := pending[playerID]
	offers, ok := offersRaw.([]any)
	if !ok || len(offers) == 0 {
		return 0, fmt.Errorf("no pending leech offers for player %s", playerID)
	}

	normalizedFrom := strings.ToLower(strings.TrimSpace(fromPlayerID))
	fallbackIdx := -1
	fallbackAmount := 0
	for i, raw := range offers {
		offer := asMap(raw)
		offerFrom := strings.ToLower(strings.TrimSpace(firstNonEmptyString(offer["fromPlayerID"], offer["FromPlayerID"])))
		offerAmount := asInt(firstNonNil(offer["amount"], offer["Amount"]))
		if normalizedFrom != "" && offerFrom != normalizedFrom {
			continue
		}
		if fallbackIdx < 0 {
			fallbackIdx = i
			fallbackAmount = offerAmount
		}
		if strictAmount && amount > 0 && offerAmount != amount {
			continue
		}
		return i, nil
	}
	if strictAmount && amount > 0 && fallbackIdx >= 0 && fallbackAmount > amount {
		return fallbackIdx, nil
	}
	if !strictAmount && fallbackIdx >= 0 {
		return fallbackIdx, nil
	}

	if normalizedFrom == "" && !strictAmount {
		return 0, nil
	}
	return 0, fmt.Errorf("could not match leech offer for player=%s from=%s amount=%d strict=%v offers=%v", playerID, fromPlayerID, amount, strictAmount, offersRaw)
}

func hasPendingLeechOffer(state map[string]any, playerID string) bool {
	pending := asMap(state["pendingLeechOffers"])
	offersRaw := pending[playerID]
	offers, ok := offersRaw.([]any)
	return ok && len(offers) > 0
}

type leechIntent struct {
	accept         bool
	fromPlayerID   string
	amount         int
	explicitAmount bool
	found          bool
}

func inspectLeechAction(action game.Action, playerID string) (game.Action, leechIntent, bool) {
	switch a := action.(type) {
	case *notation.LogAcceptLeechAction:
		if strings.TrimSpace(a.PlayerID) != strings.TrimSpace(playerID) {
			return nil, leechIntent{}, false
		}
		return a, leechIntent{
			accept:         true,
			fromPlayerID:   a.FromPlayerID,
			amount:         a.PowerAmount,
			explicitAmount: a.Explicit,
			found:          true,
		}, true
	case *notation.LogDeclineLeechAction:
		if strings.TrimSpace(a.PlayerID) != strings.TrimSpace(playerID) {
			return nil, leechIntent{}, false
		}
		return a, leechIntent{
			accept:       false,
			fromPlayerID: a.FromPlayerID,
			found:        true,
		}, true
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if matchedAction, intent, ok := inspectLeechAction(nested, playerID); ok {
				return matchedAction, intent, true
			}
		}
	case *notation.LogPreIncomeAction:
		return inspectLeechAction(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return inspectLeechAction(a.Action, playerID)
	}
	return nil, leechIntent{}, false
}

func findUpcomingTownTileSelection(actions []game.Action, playerID string) (models.TownTileType, bool) {
	tiles := findUpcomingTownTileSelections(actions, playerID)
	if len(tiles) == 0 {
		return models.TownTileUnknown, false
	}
	return tiles[0], true
}

func findUpcomingTownTileSelections(actions []game.Action, playerID string) []models.TownTileType {
	out := make([]models.TownTileType, 0, 2)
	for _, action := range actions {
		out = append(out, inspectTownTileSelection(action, playerID)...)
	}
	return out
}

func inspectTownTileSelection(action game.Action, playerID string) []models.TownTileType {
	switch a := action.(type) {
	case *notation.LogTownAction:
		if strings.TrimSpace(a.PlayerID) != strings.TrimSpace(playerID) {
			return nil
		}
		tile, err := notation.GetTownTileFromVP(a.VP)
		if err != nil {
			return nil
		}
		return []models.TownTileType{tile}
	case *notation.LogCompoundAction:
		out := make([]models.TownTileType, 0, 2)
		for _, nested := range a.Actions {
			out = append(out, inspectTownTileSelection(nested, playerID)...)
		}
		return out
	case *notation.LogPreIncomeAction:
		return inspectTownTileSelection(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return inspectTownTileSelection(a.Action, playerID)
	}
	return nil
}

func findUpcomingCultistsTrack(actions []game.Action, playerID string) (int, bool) {
	for _, action := range actions {
		track, ok := inspectCultistsTrack(action, playerID)
		if ok {
			return track, true
		}
	}
	return 0, false
}

func inspectCultistsTrack(action game.Action, playerID string) (int, bool) {
	switch a := action.(type) {
	case *notation.LogCultistAdvanceAction:
		if strings.TrimSpace(a.PlayerID) != strings.TrimSpace(playerID) {
			return 0, false
		}
		return int(a.Track), true
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if track, ok := inspectCultistsTrack(nested, playerID); ok {
				return track, true
			}
		}
	case *notation.LogPreIncomeAction:
		return inspectCultistsTrack(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return inspectCultistsTrack(a.Action, playerID)
	}
	return 0, false
}

func parseBridgeFromACT1(code string) (board.Hex, board.Hex, error) {
	upper := strings.ToUpper(strings.TrimSpace(code))
	parts := strings.Split(upper, "-")
	if len(parts) != 3 || parts[0] != "ACT1" {
		return board.Hex{}, board.Hex{}, fmt.Errorf("invalid ACT1 bridge code: %s", code)
	}
	hex1, err := notation.ConvertLogCoordToAxial(parts[1])
	if err != nil {
		return board.Hex{}, board.Hex{}, err
	}
	hex2, err := notation.ConvertLogCoordToAxial(parts[2])
	if err != nil {
		return board.Hex{}, board.Hex{}, err
	}
	return hex1, hex2, nil
}

func parseACTSSpecial(code string) (board.Hex, bool, models.TerrainType, error) {
	raw := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(code)), "ACTS-")
	if raw == code {
		return board.Hex{}, false, models.TerrainTypeUnknown, fmt.Errorf("not an ACTS code: %s", code)
	}

	build := false
	if dotIdx := strings.Index(raw, "."); dotIdx >= 0 {
		build = true
		raw = raw[:dotIdx]
	}

	parts := strings.Split(raw, "-")
	hex, err := notation.ConvertLogCoordToAxial(parts[0])
	if err != nil {
		return board.Hex{}, false, models.TerrainTypeUnknown, err
	}

	targetTerrain := models.TerrainTypeUnknown
	if len(parts) > 1 {
		targetTerrain = parseTerrainCode(parts[1])
	}
	return hex, build, targetTerrain, nil
}

func parseStrongholdTransform(code, prefix string) (board.Hex, bool, models.TerrainType, error) {
	raw := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(code)), prefix)
	if raw == code {
		return board.Hex{}, false, models.TerrainTypeUnknown, fmt.Errorf("invalid stronghold transform code: %s", code)
	}
	build := false
	if dotIdx := strings.Index(raw, "."); dotIdx >= 0 {
		build = true
		raw = raw[:dotIdx]
	}
	hex, err := notation.ConvertLogCoordToAxial(raw)
	if err != nil {
		return board.Hex{}, false, models.TerrainTypeUnknown, err
	}
	return hex, build, models.TerrainTypeUnknown, nil
}

func parseCultTrackLetter(letter string) (game.CultTrack, error) {
	switch strings.ToUpper(strings.TrimSpace(letter)) {
	case "F":
		return game.CultFire, nil
	case "W":
		return game.CultWater, nil
	case "E":
		return game.CultEarth, nil
	case "A":
		return game.CultAir, nil
	default:
		return game.CultUnknown, fmt.Errorf("unknown cult track letter: %s", letter)
	}
}

func parseTerrainCode(code string) models.TerrainType {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "P", "BR":
		return models.TerrainPlains
	case "S", "BK":
		return models.TerrainSwamp
	case "L", "BL":
		return models.TerrainLake
	case "F", "G":
		return models.TerrainForest
	case "M", "GY":
		return models.TerrainMountain
	case "W", "R":
		return models.TerrainWasteland
	case "D", "Y":
		return models.TerrainDesert
	default:
		return models.TerrainTypeUnknown
	}
}

func parseTrackListAny(raw any) []int {
	if raw == nil {
		return nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(entries))
	for _, entry := range entries {
		out = append(out, asInt(entry))
	}
	return out
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func describeGoldenAction(a game.Action) string {
	switch v := a.(type) {
	case *notation.LogCompoundAction:
		parts := make([]string, 0, len(v.Actions))
		for _, sub := range v.Actions {
			parts = append(parts, describeGoldenAction(sub))
		}
		return fmt.Sprintf("compound[%s]", strings.Join(parts, ", "))
	case *notation.LogConversionAction:
		return fmt.Sprintf("convert(cost=%v reward=%v)", v.Cost, v.Reward)
	case *notation.LogBurnAction:
		return fmt.Sprintf("burn(%d)", v.Amount)
	case *notation.LogPowerAction:
		return fmt.Sprintf("power(%s)", v.ActionCode)
	case *notation.LogSpecialAction:
		return fmt.Sprintf("special(%s)", v.ActionCode)
	case *notation.LogAcceptLeechAction:
		return fmt.Sprintf("accept_leech(from=%s amount=%d explicit=%t)", v.FromPlayerID, v.PowerAmount, v.Explicit)
	case *notation.LogDeclineLeechAction:
		return fmt.Sprintf("decline_leech(from=%s)", v.FromPlayerID)
	case *notation.LogCultistAdvanceAction:
		return fmt.Sprintf("cultists(track=%d)", int(v.Track))
	case *notation.LogPreIncomeAction:
		return fmt.Sprintf("pre[%s]", describeGoldenAction(v.Action))
	case *notation.LogPostIncomeAction:
		return fmt.Sprintf("post[%s]", describeGoldenAction(v.Action))
	case *game.TransformAndBuildAction:
		return fmt.Sprintf("transform(q=%d r=%d build=%t terrain=%d skip=%t)", v.TargetHex.Q, v.TargetHex.R, v.BuildDwelling, int(v.TargetTerrain), v.UseSkip)
	case *game.UpgradeBuildingAction:
		return fmt.Sprintf("upgrade(q=%d r=%d to=%d)", v.TargetHex.Q, v.TargetHex.R, int(v.NewBuildingType))
	case *game.AdvanceShippingAction:
		return "advance_shipping"
	case *game.AdvanceDiggingAction:
		return "advance_digging"
	case *game.SendPriestToCultAction:
		return fmt.Sprintf("send_priest(track=%d spaces=%d)", int(v.Track), v.SpacesToClimb)
	case *game.PassAction:
		return "pass"
	default:
		return fmt.Sprintf("%T", a)
	}
}

type replayActionComparator struct {
	manager *replay.ReplayManager
	session *replay.ReplaySession

	gameID string
}

func newReplayActionComparator(fixtureText string, playerOrder []string) (*replayActionComparator, error) {
	manager := replay.NewReplayManager("/tmp")
	gameID := "cmp-snellman-golden"
	if err := manager.ImportText(gameID, fixtureText, "snellman"); err != nil {
		return nil, err
	}

	session, err := manager.StartReplay(gameID, true)
	if err != nil {
		return nil, err
	}

	if session.Simulator != nil && len(session.Simulator.CurrentState.TurnOrder) == 0 && len(playerOrder) > 0 {
		session.Simulator.CurrentState.TurnOrder = append([]string(nil), playerOrder...)
	}

	return &replayActionComparator{
		manager: manager,
		session: session,
		gameID:  gameID,
	}, nil
}

func (c *replayActionComparator) expectPreState(index int, action game.Action, wsState map[string]any, wsGS *game.GameState) error {
	if c == nil || c.session == nil || c.session.Simulator == nil {
		return nil
	}
	if err := c.syncToNextReplayAction(index, false); err != nil {
		return err
	}

	if c.session.Simulator.CurrentIndex >= len(c.session.Simulator.Actions) {
		return fmt.Errorf("replay exhausted before fixture action %d (%s)", index, describeGoldenAction(action))
	}

	if err := c.assertReplayActionMatches(index, action); err != nil {
		return err
	}

	replayState := c.session.Simulator.GetState()
	if replayState == nil {
		return fmt.Errorf("nil replay state before action %d", index)
	}

	replayAction := c.currentReplayAction()
	if cmpIndex := strings.TrimSpace(os.Getenv("TM_COMPARE_DEBUG_INDEX")); cmpIndex != "" {
		if idx, err := strconv.Atoi(cmpIndex); err == nil && idx == index {
			fmt.Printf(
				"DEBUG_IDX_PRE index=%d replayCurrentIndex=%d replayType=%T replayPlayer=%q\n",
				index,
				c.session.Simulator.CurrentIndex,
				replayAction,
				replayActionPlayerID(replayAction),
			)
		}
	}

	if err := compareWebsocketAndReplayState(wsState, wsGS, replayState, "pre", index, action); err != nil {
		return err
	}

	return nil
}

func (c *replayActionComparator) advanceAndAssertPostState(action game.Action, index int, wsState map[string]any, wsGS *game.GameState) error {
	if c == nil || c.session == nil || c.session.Simulator == nil {
		return nil
	}

	if err := c.session.Simulator.StepForward(); err != nil {
		return fmt.Errorf("replay step failed at fixture action %d (%s): %w", index, describeGoldenAction(action), err)
	}
	if err := c.syncToNextReplayAction(index+1, true); err != nil && !errors.Is(err, errReplayEnded) {
		return err
	}

	replayState := c.session.Simulator.GetState()
	if replayState == nil {
		return fmt.Errorf("nil replay state after action %d", index)
	}
	if c.session.Simulator.CurrentIndex >= len(c.session.Simulator.Actions) &&
		replayState.Phase == game.PhaseAction &&
		replayState.Round >= 6 &&
		replayState.AllPlayersPassed() &&
		!replayState.HasLateRoundPendingDecisions() {
		replayState.ExecuteCleanupPhase()
	}

	replayAction := c.currentReplayAction()
	if cmpIndex := strings.TrimSpace(os.Getenv("TM_COMPARE_DEBUG_INDEX")); cmpIndex != "" {
		if idx, err := strconv.Atoi(cmpIndex); err == nil && idx == index {
			fmt.Printf(
				"DEBUG_IDX_POST index=%d replayCurrentIndex=%d replayType=%T replayPlayer=%q\n",
				index,
				c.session.Simulator.CurrentIndex,
				replayAction,
				replayActionPlayerID(replayAction),
			)
		}
	}

	if err := compareWebsocketAndReplayState(wsState, wsGS, replayState, "post", index, action); err != nil {
		return err
	}

	return nil
}

func (c *replayActionComparator) syncToNextReplayAction(index int, allowEnd bool) error {
	if c.session == nil || c.session.Simulator == nil {
		return nil
	}

	for guard := 0; guard < 200; guard++ {
		if c.session.Simulator.CurrentIndex >= len(c.session.Simulator.Actions) {
			if allowEnd {
				return errReplayEnded
			}
			return fmt.Errorf("replay exhausted while syncing to action %d", index)
		}

		if _, ok := c.session.Simulator.Actions[c.session.Simulator.CurrentIndex].(notation.ActionItem); ok {
			return nil
		}

		if err := c.session.Simulator.StepForward(); err != nil {
			return err
		}
	}

	return fmt.Errorf("failed to find next replay action while syncing action %d", index)
}

func (c *replayActionComparator) assertReplayActionMatches(index int, action game.Action) error {
	if action == nil {
		return fmt.Errorf("nil fixture action at index %d", index)
	}
	item := c.session.Simulator.Actions[c.session.Simulator.CurrentIndex]
	actionItem, ok := item.(notation.ActionItem)
	if !ok || actionItem.Action == nil {
		return fmt.Errorf("expected replay action item at index %d for %s, got %T", index, describeGoldenAction(action), item)
	}

	want := strings.TrimSpace(action.GetPlayerID())
	gotPlayer := strings.TrimSpace(actionItem.Action.GetPlayerID())
	if want != "" && gotPlayer != "" && want != gotPlayer {
		return fmt.Errorf("player mismatch at fixture action %d: fixture=%q replay=%q action=%s", index, want, gotPlayer, describeGoldenAction(action))
	}

	if reflect.TypeOf(action) != reflect.TypeOf(actionItem.Action) {
		return fmt.Errorf("action type mismatch at %d: fixture=%T replay=%T (%s)", index, action, actionItem.Action, describeGoldenAction(action))
	}

	if !actionPlayerSignatureMatches(action, actionItem.Action) {
		return fmt.Errorf("fixture/replay action identity mismatch at %d: fixture=%s replay=%T(%s)", index, describeGoldenAction(action), actionItem.Action, actionItem.Action.GetPlayerID())
	}

	return nil
}

func (c *replayActionComparator) currentReplayAction() game.Action {
	if c == nil || c.session == nil || c.session.Simulator == nil {
		return nil
	}
	if c.session.Simulator.CurrentIndex < 0 || c.session.Simulator.CurrentIndex >= len(c.session.Simulator.Actions) {
		return nil
	}
	item := c.session.Simulator.Actions[c.session.Simulator.CurrentIndex]
	actionItem, ok := item.(notation.ActionItem)
	if !ok || actionItem.Action == nil {
		return nil
	}
	return actionItem.Action
}

func replayActionPlayerID(action game.Action) string {
	if action == nil {
		return ""
	}
	return strings.TrimSpace(action.GetPlayerID())
}

func actionPlayerSignatureMatches(actionA game.Action, actionB game.Action) bool {
	if actionA == nil || actionB == nil {
		return actionA == nil && actionB == nil
	}

	if strings.TrimSpace(actionA.GetPlayerID()) != strings.TrimSpace(actionB.GetPlayerID()) {
		return false
	}

	if strings.TrimSpace(fmt.Sprintf("%d", actionA.GetType())) == "" || strings.TrimSpace(fmt.Sprintf("%d", actionB.GetType())) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%d", actionA.GetType())), strings.TrimSpace(fmt.Sprintf("%d", actionB.GetType())))
}

func compareWebsocketAndReplayState(wsState map[string]any, wsGS *game.GameState, replayState *game.GameState, phase string, index int, action game.Action) error {
	if wsState == nil || replayState == nil {
		// keep behavior unchanged; explicit nil checks below still handle details
	}
	if cmpIndex := strings.TrimSpace(os.Getenv("TM_COMPARE_DEBUG_INDEX")); cmpIndex != "" {
		if v, err := strconv.Atoi(cmpIndex); err == nil && v == index {
			fmt.Printf("DEBUG_COMPARE_DETAILED phase=%s index=%d action=%s\n", phase, index, describeGoldenAction(action))
			wsDecision := asMap(wsState["pendingDecision"])
			wsOffers := asMap(wsState["pendingLeechOffers"])
			fmt.Printf(
				"DEBUG_WS_PENDING phase=%s pendingType=%s pendingPlayer=%s nextLeech=%v\n",
				phase,
				asString(wsDecision["type"]),
				asString(wsDecision["playerId"]),
				wsOffers != nil,
			)
			if wsPlayers, ok := wsState["players"].(map[string]any); ok {
				for playerID, raw := range wsPlayers {
					playerRaw := asMap(raw)
					options := asMap(playerRaw["options"])
					fmt.Printf("DEBUG_WS_PLAYER player=%s autoLeech=%s autoConvert=%v showIncome=%v\n", playerID, asString(options["autoLeechMode"]), asBool(options["autoConvertOnPass"]), asBool(options["showIncomePreview"]))
					fmt.Printf(
						"DEBUG_WS_LEECH player=%s offers=%v\n",
						playerID,
						wsOffers[playerID],
					)
					cult := asMap(playerRaw["cults"])
					fmt.Printf(
						"DEBUG_WS_CULT player=%s passed=%v passedThisRound=%v hasPassed=%v\n",
						playerID,
						asString(cult["0"]),
						asString(cult["1"]),
						asBool(playerRaw["hasPassed"]),
					)
				}
			}

			if replayState != nil {
				replayOffers := replayState.PendingLeechOffers
				replayDecision := replayState.GetNextLeechResponder()
				fmt.Printf("DEBUG_REPLAY_PENDING phase=%s nextLeechResponder=%s\n", phase, replayDecision)
				for playerID, offers := range replayOffers {
					fmt.Printf("DEBUG_REPLAY_LEECH player=%s offers=%v\n", playerID, offers)
				}
				for playerID := range replayOffers {
					player := replayState.GetPlayer(playerID)
					if player == nil {
						continue
					}
					fmt.Printf(
						"DEBUG_REPLAY_PLAYER player=%s autoLeech=%s autoConvert=%v showIncome=%v\n",
						playerID,
						player.Options.AutoLeechMode,
						player.Options.AutoConvertOnPass,
						player.Options.ShowIncomePreview,
					)
					if offers := replayOffers[playerID]; len(offers) > 0 {
						fmt.Printf("DEBUG_REPLAY_PLAYER_OFFERS player=%s %v\n", playerID, offers)
					}
				}
			}
			wsPlayers := asMap(wsState["players"])
			replayPlayers := replayState.Players
			fmt.Printf("DEBUG_COMPARE index=%d phase=%s action=%s\n", index, phase, describeGoldenAction(action))
			for playerID, wsPlayerRaw := range wsPlayers {
				r := gameStatePlayerSummary(replayPlayers[playerID])
				wsSum := websocketPlayerSummary(wsPlayerRaw)
				fmt.Printf("DEBUG_PLAYER player=%s ws=%d/%d/%d/%d/%d/%d/%d replay=%d/%d/%d/%d/%d/%d/%d\n",
					playerID,
					wsSum.vp, wsSum.coins, wsSum.workers, wsSum.priests, wsSum.pw1, wsSum.pw2, wsSum.pw3,
					r.vp, r.coins, r.workers, r.priests, r.pw1, r.pw2, r.pw3,
				)
			}
		}
	}
	if wsGS == nil {
		return fmt.Errorf("nil websocket game state before %s action %d (%s)", phase, index, describeGoldenAction(action))
	}

	wsRound := 0
	wsRoundAny := asMap(wsState["round"])
	if wsRoundAny != nil {
		wsRound = asInt(wsRoundAny["round"])
	}
	replayRound := replayState.Round
	if wsRound != replayRound {
		return fmt.Errorf("round mismatch at %s action %d (%s): websocket=%d replay=%d", phase, index, describeGoldenAction(action), wsRound, replayRound)
	}

	wsTurn := currentTurnPlayerID(wsState)
	replayTurn := ""
	if cp := replayState.GetCurrentPlayer(); cp != nil {
		replayTurn = cp.ID
	}
	if strings.TrimSpace(wsTurn) != "" && strings.TrimSpace(replayTurn) != "" && strings.TrimSpace(wsTurn) != strings.TrimSpace(replayTurn) {
		return fmt.Errorf("turn mismatch at %s action %d (%s): websocket=%q replay=%q", phase, index, describeGoldenAction(action), wsTurn, replayTurn)
	}

	wsPlayers := asMap(wsState["players"])
	for playerID, wsPlayerRaw := range wsPlayers {
		replayPlayer := replayState.Players[playerID]
		if replayPlayer == nil {
			return fmt.Errorf("player missing in replay during %s action %d (%s): %s", phase, index, describeGoldenAction(action), playerID)
		}

		wsSummary := websocketPlayerSummary(wsPlayerRaw)
		replaySummary := gameStatePlayerSummary(replayPlayer)
		if wsSummary.vp != replaySummary.vp {
			return fmt.Errorf("resource mismatch at %s action %d (%s) player=%s: vp websocket=%d replay=%d", phase, index, describeGoldenAction(action), playerID, wsSummary.vp, replaySummary.vp)
		}
		if wsSummary.coins != replaySummary.coins {
			return fmt.Errorf("coin mismatch at %s action %d (%s) player=%s: websocket=%d replay=%d", phase, index, describeGoldenAction(action), playerID, wsSummary.coins, replaySummary.coins)
		}
		if wsSummary.workers != replaySummary.workers {
			return fmt.Errorf("worker mismatch at %s action %d (%s) player=%s: websocket=%d replay=%d", phase, index, describeGoldenAction(action), playerID, wsSummary.workers, replaySummary.workers)
		}
		if wsSummary.priests != replaySummary.priests {
			return fmt.Errorf("priest mismatch at %s action %d (%s) player=%s: websocket=%d replay=%d", phase, index, describeGoldenAction(action), playerID, wsSummary.priests, replaySummary.priests)
		}
		if wsSummary.pw1 != replaySummary.pw1 || wsSummary.pw2 != replaySummary.pw2 || wsSummary.pw3 != replaySummary.pw3 {
			return fmt.Errorf("power mismatch at %s action %d (%s) player=%s: websocket=%d/%d/%d replay=%d/%d/%d", phase, index, describeGoldenAction(action), playerID, wsSummary.pw1, wsSummary.pw2, wsSummary.pw3, replaySummary.pw1, replaySummary.pw2, replaySummary.pw3)
		}
	}

	for playerID := range replayState.Players {
		if _, ok := wsPlayers[playerID]; !ok {
			return fmt.Errorf("extra replay player missing in websocket during %s action %d (%s): %s", phase, index, describeGoldenAction(action), playerID)
		}
	}

	return nil
}

type playerResourceSummary struct {
	vp      int
	coins   int
	workers int
	priests int
	pw1     int
	pw2     int
	pw3     int
}

func websocketPlayerSummary(raw any) playerResourceSummary {
	player := asMap(raw)
	res := asMap(player["resources"])
	power := asMap(res["power"])
	return playerResourceSummary{
		vp:      asInt(player["victoryPoints"]),
		coins:   asInt(res["coins"]),
		workers: asInt(res["workers"]),
		priests: asInt(res["priests"]),
		pw1:     asInt(power["powerI"]),
		pw2:     asInt(power["powerII"]),
		pw3:     asInt(power["powerIII"]),
	}
}

func gameStatePlayerSummary(player *game.Player) playerResourceSummary {
	if player == nil {
		return playerResourceSummary{}
	}
	res := player.Resources
	if res == nil {
		return playerResourceSummary{}
	}
	pw1, pw2, pw3 := 0, 0, 0
	if res.Power != nil {
		pw1, pw2, pw3 = res.Power.Bowl1, res.Power.Bowl2, res.Power.Bowl3
	}
	return playerResourceSummary{
		vp:      player.VictoryPoints,
		coins:   res.Coins,
		workers: res.Workers,
		priests: res.Priests,
		pw1:     pw1,
		pw2:     pw2,
		pw3:     pw3,
	}
}

func firstNonEmptyString(values ...any) string {
	for _, v := range values {
		s := asString(v)
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}
