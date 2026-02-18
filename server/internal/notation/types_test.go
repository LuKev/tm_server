package notation

import (
	"fmt"
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestLogCompoundAction_AllowsAuxiliaryOnlySequence(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}
	// Ensure the auxiliary actions are executable.
	gs.Players[playerID].Resources.Power = game.NewPowerSystem(0, 2, 1)

	compound := &LogCompoundAction{
		Actions: []game.Action{
			&LogBurnAction{PlayerID: playerID, Amount: 1},
			&LogConversionAction{
				PlayerID: playerID,
				Cost: map[models.ResourceType]int{
					models.ResourcePower: 1,
				},
				Reward: map[models.ResourceType]int{
					models.ResourceCoin: 1,
				},
			},
		},
	}

	if err := compound.Execute(gs); err != nil {
		t.Fatalf("compound.Execute(auxiliary-only) error = %v, want nil", err)
	}
}

func TestLogDeclineLeechAction_NoPendingOffers_NoError(t *testing.T) {
	gs := game.NewGameState()
	playerID := "p1"
	gs.Players[playerID] = &game.Player{ID: playerID, Resources: game.NewResourcePool()}

	action := &LogDeclineLeechAction{PlayerID: playerID}
	if err := action.Execute(gs); err != nil {
		t.Fatalf("LogDeclineLeechAction.Execute(no pending) error = %v, want nil", err)
	}
}

func TestParseReplayInsufficientResources_WrappedPrefix(t *testing.T) {
	err := fmt.Errorf("failed to pay for dwelling: insufficient resources: need (coins:2, workers:1, priests:0, power:0), have (coins:0, workers:5, priests:0, power:0)")
	got, ok := parseReplayInsufficientResources(err)
	if !ok {
		t.Fatalf("parseReplayInsufficientResources() did not match wrapped message")
	}
	if got.needCoins != 2 || got.needWorkers != 1 || got.needPriests != 0 || got.needPower != 0 {
		t.Fatalf("parsed need = %+v, want coins=2 workers=1 priests=0 power=0", got)
	}
	if got.haveCoins != 0 || got.haveWorkers != 5 || got.havePriests != 0 || got.havePower != 0 {
		t.Fatalf("parsed have = %+v, want coins=0 workers=5 priests=0 power=0", got)
	}
}

func TestIncomeWrapper_TransformOnlyRetriesWithSyntheticSpade(t *testing.T) {
	testCases := []struct {
		name  string
		wrap  func(game.Action) game.Action
	}{
		{
			name: "pre_income",
			wrap: func(a game.Action) game.Action { return &LogPreIncomeAction{Action: a} },
		},
		{
			name: "post_income",
			wrap: func(a game.Action) game.Action { return &LogPostIncomeAction{Action: a} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := game.NewGameState()
			playerID := "cultists"
			if err := gs.AddPlayer(playerID, factions.NewCultists()); err != nil {
				t.Fatalf("AddPlayer failed: %v", err)
			}
			player := gs.GetPlayer(playerID)
			if player == nil {
				t.Fatalf("player %q missing after AddPlayer", playerID)
			}

			target, anchor, ok := findAdjacentNonRiverPair(gs)
			if !ok {
				t.Fatalf("failed to find adjacent non-river hex pair")
			}

			anchorHex := gs.Map.GetHex(anchor)
			if anchorHex == nil {
				t.Fatalf("anchor hex missing")
			}
			anchorHex.Building = &models.Building{
				Type:       models.BuildingDwelling,
				Faction:    player.Faction.GetType(),
				PlayerID:   playerID,
				PowerValue: game.GetPowerValue(models.BuildingDwelling),
			}

			home := player.Faction.GetHomeTerrain()
			oneStepTerrain, ok := findTerrainDistanceOneToHome(gs, home)
			if !ok {
				t.Fatalf("failed to find distance-1 terrain for home %v", home)
			}
			targetHex := gs.Map.GetHex(target)
			if targetHex == nil {
				t.Fatalf("target hex missing")
			}
			targetHex.Terrain = oneStepTerrain

			player.Resources.Workers = 0
			action := game.NewTransformAndBuildAction(playerID, target, false, home)
			wrapped := tc.wrap(action)

			if err := wrapped.Execute(gs); err != nil {
				t.Fatalf("wrapped.Execute() error = %v, want nil", err)
			}
			if got := gs.Map.GetHex(target).Terrain; got != home {
				t.Fatalf("target terrain = %v, want %v", got, home)
			}
			if got := player.Resources.Workers; got != 0 {
				t.Fatalf("workers = %d, want 0", got)
			}
		})
	}
}

func findAdjacentNonRiverPair(gs *game.GameState) (board.Hex, board.Hex, bool) {
	if gs == nil || gs.Map == nil {
		return board.Hex{}, board.Hex{}, false
	}
	for h, hex := range gs.Map.Hexes {
		if hex == nil || hex.Terrain == models.TerrainRiver {
			continue
		}
		for _, n := range h.Neighbors() {
			neighbor := gs.Map.GetHex(n)
			if neighbor == nil || neighbor.Terrain == models.TerrainRiver {
				continue
			}
			return h, n, true
		}
	}
	return board.Hex{}, board.Hex{}, false
}

func findTerrainDistanceOneToHome(gs *game.GameState, home models.TerrainType) (models.TerrainType, bool) {
	candidates := []models.TerrainType{
		models.TerrainDesert,
		models.TerrainPlains,
		models.TerrainSwamp,
		models.TerrainLake,
		models.TerrainForest,
		models.TerrainMountain,
		models.TerrainWasteland,
	}
	for _, t := range candidates {
		if t == home {
			continue
		}
		if gs.Map.GetTerrainDistance(t, home) == 1 {
			return t, true
		}
	}
	return models.TerrainTypeUnknown, false
}
