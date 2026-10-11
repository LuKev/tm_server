package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"

	"github.com/lukev/tm_server/internal/az"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

// This is a test-only legal trajectory generator, not an expert policy or a
// second rules engine. Never fund resources or inject pending decisions here.
type traceSample struct {
	Ply     int               `json:"ply"`
	State   json.RawMessage   `json:"state"`
	Actions []az.SearchAction `json:"actions"`
}

func exportTraces() error {
	pairs := [][2]models.FactionType{
		{models.FactionChaosMagicians, models.FactionEngineers},
		{models.FactionGiants, models.FactionDwarves},
		{models.FactionFakirs, models.FactionAuren},
		{models.FactionNomads, models.FactionWitches},
		{models.FactionHalflings, models.FactionMermaids},
		{models.FactionCultists, models.FactionDarklings},
		{models.FactionAlchemists, models.FactionSwarmlings},
	}
	for pairIndex, pair := range pairs {
		for replicate := 0; replicate < 2; replicate++ {
			seed := int64(9100 + pairIndex*10 + replicate)
			position, err := az.NewBaseGame(seed, pair[0], pair[1])
			if err != nil {
				return err
			}
			renamed, err := az.NewBaseGameWithPlayerIDs(seed, pair[0], pair[1], "alice", "bob")
			if err != nil {
				return err
			}
			rng := rand.New(rand.NewSource(seed))
			var trace []az.SearchAction
			var samples []traceSample
			seen := make(map[string]bool)
			for ply := 0; !position.IsTerminal(); ply++ {
				if ply >= 1000 {
					return fmt.Errorf("seed %d exceeded trace tripwire", seed)
				}
				raw, err := position.CanonicalJSON()
				if err != nil {
					return err
				}
				other, err := renamed.CanonicalJSON()
				if err != nil || !bytes.Equal(raw, other) {
					return fmt.Errorf("seed %d ply %d player-relative replay differs: %v", seed, ply, err)
				}
				actions := position.LegalActions()
				if len(actions) == 0 {
					return fmt.Errorf("seed %d ply %d has no actions", seed, ply)
				}
				// Retain the first position for each round/phase and pending-field
				// decision player and pending owner. Keep the full action trace.
				var record struct {
					State map[string]json.RawMessage `json:"state"`
				}
				if err := json.Unmarshal(raw, &record); err != nil {
					return err
				}
				keys := []string{fmt.Sprintf("phase:%s/round:%s", record.State["phase"], record.State["round"])}
				for _, field := range []string{"pendingLeechOffers", "pendingCultistsLeech", "pendingFavorTileSelection", "pendingHalflingsSpades", "pendingDarklingsPriestOrdination", "pendingCultistsCultSelection", "pendingTownCultTopChoice", "pendingChaosMagiciansDoubleTurn", "pendingTownFormations", "pendingFreeActionsPlayerId"} {
					value := record.State[field]
					if field == "pendingLeechOffers" || field == "pendingTownFormations" {
						var groups map[string][]json.RawMessage
						if err := json.Unmarshal(value, &groups); err != nil {
							return err
						}
						if len(groups["self"])+len(groups["opponent"]) == 0 {
							continue
						}
					}
					if len(value) > 0 && string(value) != "null" && string(value) != "{}" && string(value) != "\"\"" {
						var pending map[string]json.RawMessage
						if field != "pendingFreeActionsPlayerId" {
							if err := json.Unmarshal(value, &pending); err != nil {
								return err
							}
						}
						keys = append(keys, field+":"+string(position.DecisionPlayer())+":"+string(pending["PlayerID"])+string(pending["playerId"]))
					}
				}
				keep := false
				for _, key := range keys {
					if !seen[key] {
						seen[key] = true
						keep = true
					}
				}
				if keep {
					if !reflect.DeepEqual(actions, renamed.LegalActions()) {
						return fmt.Errorf("seed %d ply %d renamed legal choices differ", seed, ply)
					}
					samples = append(samples, traceSample{Ply: ply, State: raw, Actions: actions})
				}
				choice := actions[rng.Intn(len(actions))]
				if replicate == 1 {
					choice = constructionChoice(position, actions, rng)
				}
				trace = append(trace, choice)
				if err := position.Apply(choice); err != nil {
					return fmt.Errorf("seed %d ply %d: %w", seed, ply, err)
				}
				if err := renamed.Apply(choice); err != nil {
					return err
				}
			}
			if !renamed.IsTerminal() || position.CanonicalHash() != renamed.CanonicalHash() {
				return fmt.Errorf("terminal replay differs")
			}
			if err := json.NewEncoder(os.Stdout).Encode(struct {
				Seed     int64                 `json:"seed"`
				Factions [2]models.FactionType `json:"factions"`
				Trace    []az.SearchAction     `json:"trace"`
				Samples  []traceSample         `json:"samples"`
				Terminal bool                  `json:"terminal"`
			}{seed, pair, trace, samples, true}); err != nil {
				return err
			}
		}
	}
	return nil
}

// The second replicate seeks construction/stronghold boundaries using only
// complete legal candidates. This coverage policy is not a strength baseline.
func constructionChoice(position *az.GamePosition, actions []az.SearchAction, rng *rand.Rand) az.SearchAction {
	state := position.StateClone()
	owner := string(position.DecisionPlayer())
	stronghold, tradingHouse := false, false
	for _, space := range state.Map.Hexes {
		if space.Building != nil && space.Building.PlayerID == owner {
			stronghold = stronghold || space.Building.Type == models.BuildingStronghold
			tradingHouse = tradingHouse || space.Building.Type == models.BuildingTradingHouse
		}
	}
	best, score := actions[0], -1
	for _, index := range rng.Perm(len(actions)) {
		action := actions[index]
		value := 10
		switch action.Kind {
		case game.ActionFinishTurn:
			value = 1000
		case game.ActionSpecialAction:
			if action.Special == game.SpecialActionChaosMagiciansDoubleTurn {
				value = 1000
			}
		case game.ActionUpgradeBuilding:
			value = 500
			if !stronghold && action.Building == models.BuildingTradingHouse {
				value = 850
			}
			if action.Building == models.BuildingStronghold {
				value = 900
			}
		case game.ActionTransformAndBuild:
			if action.Build {
				value = 600
				for _, neighbor := range action.Hexes[0].Neighbors() {
					space := state.Map.GetHex(neighbor)
					if space != nil && space.Building != nil && space.Building.PlayerID == owner {
						value += 30
					}
				}
			}
		case game.ActionPowerAction:
			value = 100
		case game.ActionPass:
			value = 0
		}
		// Save income for the stronghold instead of repeatedly spending every
		// worker on dwellings. This seeks continuation coverage, not strength.
		if !stronghold && tradingHouse {
			if action.Kind == game.ActionPass {
				value = 200
			}
			if action.Kind == game.ActionTransformAndBuild || (action.Kind == game.ActionUpgradeBuilding && action.Building != models.BuildingStronghold) {
				value = 100
			}
		}
		if value > score {
			best, score = action, value
		}
	}
	return best
}
