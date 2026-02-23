package websocket

import (
	"embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/lobby"
	"github.com/lukev/tm_server/internal/models"
	"github.com/lukev/tm_server/internal/notation"
)

//go:embed testdata/*.txt
var snellmanFixtureFS embed.FS

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
	)
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

func runGoldenSnellmanFixture(t *testing.T, fixturePath string, playerIDs []string, expected map[string]int) {
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
		t:       t,
		gameID:  gameID,
		clients: clients,
		state:   state,
		gs:      gs,
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
		nextPlayer := strings.TrimSpace(action.GetPlayerID())
		if err := runner.resolveBlockingPendingBefore(nextPlayer, actions[i:]); err != nil {
			t.Fatalf("resolve pending before action %d (%T): %v", i, action, err)
		}
		if err := runner.executeActionWithUpcoming(action, actions[i:]); err != nil {
			t.Fatalf("execute action %d (%T): %v", i, action, err)
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

type goldenRunner struct {
	t       *testing.T
	gameID  string
	clients map[string]*gws.Conn
	state   map[string]any
	gs      *game.GameState
	step    int
}

func (r *goldenRunner) executeActionWithUpcoming(action game.Action, upcoming []game.Action) error {
	switch a := action.(type) {
	case *notation.LogCompoundAction:
		return r.executeCompound(a, upcoming)
	case *notation.LogPreIncomeAction:
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
	case *game.UpgradeBuildingAction:
		return r.perform(a.PlayerID, "upgrade_building", map[string]any{
			"targetHex":       toHexParam(a.TargetHex),
			"newBuildingType": int(a.NewBuildingType),
		})
	case *game.AdvanceShippingAction:
		return r.perform(a.PlayerID, "advance_shipping", map[string]any{})
	case *game.AdvanceDiggingAction:
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
			return nil
		}
		if amount > maxBurn {
			amount = maxBurn
		}
		if amount <= 0 {
			return nil
		}
		return r.perform(a.PlayerID, "burn_power", map[string]any{"amount": amount})
	case *notation.LogConversionAction:
		convType, amount, err := mapLogConversion(a)
		if err != nil {
			return err
		}
		return r.perform(a.PlayerID, "conversion", map[string]any{
			"conversionType": string(convType),
			"amount":         amount,
		})
	case *notation.LogAcceptLeechAction:
		if !hasPendingLeechOffer(r.state, a.PlayerID) {
			return nil
		}
		offerIndex, err := findLeechOfferIndex(r.state, a.PlayerID, a.FromPlayerID, a.PowerAmount, a.Explicit)
		if err != nil {
			offerIndex, err = findLeechOfferIndex(r.state, a.PlayerID, "", 0, false)
			if err != nil {
				return err
			}
		}
		return r.perform(a.PlayerID, "accept_leech", map[string]any{
			"offerIndex": offerIndex,
		})
	case *notation.LogDeclineLeechAction:
		if !hasPendingLeechOffer(r.state, a.PlayerID) {
			return nil
		}
		offerIndex, err := findLeechOfferIndex(r.state, a.PlayerID, a.FromPlayerID, 0, false)
		if err != nil {
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
			return nil
		}
		tile, err := notation.GetTownTileFromVP(a.VP)
		if err != nil {
			return err
		}
		if err := r.perform(a.PlayerID, "select_town_tile", map[string]any{"tileType": int(tile)}); err != nil {
			return err
		}
		return r.resolveTownCultTopChoice(a.PlayerID, nil)
	case *notation.LogSpecialAction:
		return r.executeSpecialAction(a)
	case *notation.LogPowerAction:
		return r.executeStandalonePowerAction(a)
	case *notation.LogCultistAdvanceAction:
		pd := asMap(r.state["pendingDecision"])
		if asString(pd["type"]) == "cultists_cult_choice" && asString(pd["playerId"]) == a.PlayerID {
			return r.perform(a.PlayerID, "select_cultists_track", map[string]any{"track": int(a.Track)})
		}
		if currentTurnPlayerID(r.state) != strings.TrimSpace(a.PlayerID) {
			// Some Snellman rows emit cult-advance bookkeeping outside turn order.
			// If no pending cultists choice is blocking and it is not this player's
			// turn, skip the row to preserve strict multiplayer turn ownership.
			return nil
		}
		return r.perform(a.PlayerID, "special_action_use", map[string]any{
			"actionType": int(game.SpecialActionBonusCardCultAdvance),
			"cultTrack":  int(a.Track),
		})
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

	pendingTownTracks := make([]game.CultTrack, 0, 2)

	for i := 0; i < len(compound.Actions); i++ {
		action := compound.Actions[i]
		nextPlayer := strings.TrimSpace(action.GetPlayerID())
		nestedUpcoming := compound.Actions[i:]
		if len(upcoming) > 1 {
			nestedUpcoming = append(slices.Clone(nestedUpcoming), upcoming[1:]...)
		}
		if err := r.resolveBlockingPendingBefore(nextPlayer, nestedUpcoming); err != nil {
			return err
		}

		// Snellman rows can serialize free actions (burn/conversion) after a
		// turn-ending main action. In strict multiplayer flow these must be
		// applied before the main action if the next non-free action belongs to
		// another player.
		if shouldReorderTrailingFreeActions(compound.Actions, i, nextPlayer) {
			reorderEnd := i + 1
			for reorderEnd < len(compound.Actions) && isReorderableFreeAction(compound.Actions[reorderEnd], nextPlayer) {
				follow := compound.Actions[reorderEnd]
				followUpcoming := compound.Actions[reorderEnd:]
				if len(upcoming) > 1 {
					followUpcoming = append(slices.Clone(followUpcoming), upcoming[1:]...)
				}
				if err := r.resolveBlockingPendingBefore(nextPlayer, followUpcoming); err != nil {
					return err
				}
				if err := r.executeActionWithUpcoming(follow, followUpcoming); err != nil {
					return err
				}
				reorderEnd++
			}
			if err := r.executeActionWithUpcoming(action, nestedUpcoming); err != nil {
				return err
			}
			i = reorderEnd - 1
			continue
		}

		switch a := action.(type) {
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
					if err := r.perform(a.PlayerID, "select_town_tile", map[string]any{"tileType": int(tile)}); err != nil {
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

		case *notation.LogTownAction:
			if err := r.ensureTownTileSelectionPending(a.PlayerID); err != nil {
				return err
			}
			if !r.canSelectTownTile(a.PlayerID) {
				continue
			}
			tile, err := notation.GetTownTileFromVP(a.VP)
			if err != nil {
				return err
			}
			if err := r.perform(a.PlayerID, "select_town_tile", map[string]any{"tileType": int(tile)}); err != nil {
				return err
			}
			if err := r.resolveTownCultTopChoice(a.PlayerID, pendingTownTracks); err != nil {
				return err
			}
			pendingTownTracks = pendingTownTracks[:0]
			continue
		}

		if err := r.executeActionWithUpcoming(action, nestedUpcoming); err != nil {
			return err
		}
	}

	return nil
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
		return r.perform(action.PlayerID, "use_cult_spade", params)
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
	townChoiceCursor := map[string]int{}
	for guard := 0; guard < 12; guard++ {
		pd := asMap(r.state["pendingDecision"])
		decisionType := asString(pd["type"])
		if decisionType == "" {
			return nil
		}
		playerID := asString(pd["playerId"])
		if nextPlayerID != "" && playerID == nextPlayerID {
			if decisionType != "cultists_cult_choice" {
				return nil
			}
			if upcomingActionResolvesPending(decisionType, upcoming, playerID) {
				return nil
			}
		}

		switch decisionType {
		case "spade_followup":
			remaining := asInt(pd["spadesRemaining"])
			if remaining <= 0 {
				remaining = 1
			}
			if err := r.perform(playerID, "discard_pending_spade", map[string]any{"count": remaining}); err != nil {
				return err
			}
		case "town_cult_top_choice":
			if err := r.resolveTownCultTopChoice(playerID, nil); err != nil {
				return err
			}
		case "town_tile_selection":
			choices := findUpcomingTownTileSelections(upcoming, playerID)
			cursor := townChoiceCursor[playerID]
			if cursor >= len(choices) {
				return fmt.Errorf("pending town tile selection for %s but no upcoming town choice found", playerID)
			}
			tile := choices[cursor]
			townChoiceCursor[playerID] = cursor + 1
			if err := r.perform(playerID, "select_town_tile", map[string]any{"tileType": int(tile)}); err != nil {
				return err
			}
			if err := r.resolveTownCultTopChoice(playerID, nil); err != nil {
				return err
			}
		case "cultists_cult_choice":
			if err := r.perform(playerID, "select_cultists_track", map[string]any{"track": int(game.CultFire)}); err != nil {
				return err
			}
		case "darklings_ordination":
			if err := r.perform(playerID, "darklings_ordination", map[string]any{"workersToConvert": 0}); err != nil {
				return err
			}
		case "leech_offer":
			intent := findUpcomingLeechIntent(upcoming, playerID)
			if !intent.found {
				return fmt.Errorf("pending leech offer for %s but no upcoming leech intent found", playerID)
			}
			offerIndex, err := findLeechOfferIndex(r.state, playerID, intent.fromPlayerID, intent.amount, intent.explicitAmount)
			if err != nil {
				offerIndex, err = findLeechOfferIndex(r.state, playerID, "", 0, false)
				if err != nil {
					return err
				}
			}
			actionType := "decline_leech"
			if intent.accept {
				actionType = "accept_leech"
			}
			if err := r.perform(playerID, actionType, map[string]any{"offerIndex": offerIndex}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("pending decision %q for player %q unresolved before next action for %q", decisionType, playerID, nextPlayerID)
		}
	}
	return fmt.Errorf("pending decision resolution exceeded iteration guard")
}

func upcomingActionResolvesPending(decisionType string, upcoming []game.Action, playerID string) bool {
	if len(upcoming) == 0 {
		return false
	}

	switch decisionType {
	case "cultists_cult_choice":
		if action, ok := upcoming[0].(*notation.LogCultistAdvanceAction); ok {
			return strings.TrimSpace(action.PlayerID) == strings.TrimSpace(playerID)
		}
	case "leech_offer":
		switch action := upcoming[0].(type) {
		case *notation.LogAcceptLeechAction:
			return strings.TrimSpace(action.PlayerID) == strings.TrimSpace(playerID)
		case *notation.LogDeclineLeechAction:
			return strings.TrimSpace(action.PlayerID) == strings.TrimSpace(playerID)
		}
	case "town_tile_selection":
		if action, ok := upcoming[0].(*notation.LogTownAction); ok {
			return strings.TrimSpace(action.PlayerID) == strings.TrimSpace(playerID)
		}
	case "town_cult_top_choice":
		if action, ok := upcoming[0].(*notation.LogCultTrackDecreaseAction); ok {
			return strings.TrimSpace(action.PlayerID) == strings.TrimSpace(playerID)
		}
	case "darklings_ordination":
		if action, ok := upcoming[0].(*game.UseDarklingsPriestOrdinationAction); ok {
			return strings.TrimSpace(action.PlayerID) == strings.TrimSpace(playerID)
		}
	}

	return false
}

func (r *goldenRunner) perform(playerID, actionType string, params map[string]any) error {
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
		r.t.Logf(
			"golden perform step=%d actionID=%s player=%s type=%s turn=%s phase=%d pendingType=%s pendingPlayer=%s vp=%d c=%d w=%d p=%d pw=%d/%d/%d cult=%d/%d/%d/%d params=%v",
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
			"golden post step=%d actionID=%s rev=%d turn=%s phase=%d round=%d pendingType=%s pendingPlayer=%s passed=%v",
			r.step-1,
			actionID,
			asInt(r.state["revision"]),
			currentTurnPlayerID(r.state),
			asInt(r.state["phase"]),
			asInt(asMap(r.state["round"])["round"]),
			asString(asMap(r.state["pendingDecision"])["type"]),
			asString(asMap(r.state["pendingDecision"])["playerId"]),
			passed,
		)
	}
	return nil
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

func shouldReorderTrailingFreeActions(actions []game.Action, index int, playerID string) bool {
	if index < 0 || index >= len(actions) || strings.TrimSpace(playerID) == "" {
		return false
	}
	if !isTurnEndingMainAction(actions[index]) {
		return false
	}

	reorderEnd := index + 1
	for reorderEnd < len(actions) && isReorderableFreeAction(actions[reorderEnd], playerID) {
		reorderEnd++
	}
	if reorderEnd == index+1 {
		return false
	}
	if reorderEnd >= len(actions) {
		return true
	}

	nextNonFreePlayer := strings.TrimSpace(actions[reorderEnd].GetPlayerID())
	return nextNonFreePlayer != strings.TrimSpace(playerID)
}

func isTurnEndingMainAction(action game.Action) bool {
	switch action.(type) {
	case *game.TransformAndBuildAction:
		return true
	case *game.UpgradeBuildingAction:
		return true
	case *game.AdvanceShippingAction:
		return true
	case *game.AdvanceDiggingAction:
		return true
	case *game.SendPriestToCultAction:
		return true
	case *game.PassAction:
		return true
	case *notation.LogPowerAction:
		return true
	case *notation.LogSpecialAction:
		return true
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

func findLeechOfferIndex(state map[string]any, playerID, fromPlayerID string, amount int, strictAmount bool) (int, error) {
	pending := asMap(state["pendingLeechOffers"])
	offersRaw := pending[playerID]
	offers, ok := offersRaw.([]any)
	if !ok || len(offers) == 0 {
		return 0, fmt.Errorf("no pending leech offers for player %s", playerID)
	}

	normalizedFrom := strings.ToLower(strings.TrimSpace(fromPlayerID))
	for i, raw := range offers {
		offer := asMap(raw)
		offerFrom := strings.ToLower(strings.TrimSpace(firstNonEmptyString(offer["fromPlayerID"], offer["FromPlayerID"])))
		offerAmount := asInt(firstNonNil(offer["amount"], offer["Amount"]))
		if normalizedFrom != "" && offerFrom != normalizedFrom {
			continue
		}
		if strictAmount && amount > 0 && offerAmount != amount {
			continue
		}
		return i, nil
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

func findUpcomingLeechIntent(actions []game.Action, playerID string) leechIntent {
	for _, action := range actions {
		intent, ok := inspectLeechIntent(action, playerID)
		if ok {
			return intent
		}
	}
	return leechIntent{}
}

func inspectLeechIntent(action game.Action, playerID string) (leechIntent, bool) {
	switch a := action.(type) {
	case *notation.LogAcceptLeechAction:
		if strings.TrimSpace(a.PlayerID) != strings.TrimSpace(playerID) {
			return leechIntent{}, false
		}
		return leechIntent{
			accept:         true,
			fromPlayerID:   a.FromPlayerID,
			amount:         a.PowerAmount,
			explicitAmount: a.Explicit,
			found:          true,
		}, true
	case *notation.LogDeclineLeechAction:
		if strings.TrimSpace(a.PlayerID) != strings.TrimSpace(playerID) {
			return leechIntent{}, false
		}
		return leechIntent{
			accept: false,
			found:  true,
		}, true
	case *notation.LogCompoundAction:
		for _, nested := range a.Actions {
			if intent, ok := inspectLeechIntent(nested, playerID); ok {
				return intent, true
			}
		}
	case *notation.LogPreIncomeAction:
		return inspectLeechIntent(a.Action, playerID)
	case *notation.LogPostIncomeAction:
		return inspectLeechIntent(a.Action, playerID)
	}
	return leechIntent{}, false
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
