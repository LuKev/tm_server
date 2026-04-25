package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func mustAddPlayer(t *testing.T, gs *GameState, id string, faction factions.Faction) {
	t.Helper()
	if err := gs.AddPlayer(id, faction); err != nil {
		t.Fatalf("add player %s: %v", id, err)
	}
}

func TestResolveAutoLeechOffers_AcceptsWithinThreshold(t *testing.T) {
	gs := NewGameState()
	mustAddPlayer(t, gs, "src", factions.NewNomads())
	mustAddPlayer(t, gs, "dst", factions.NewAuren())
	gs.TurnOrder = []string{"src", "dst"}
	gs.CurrentPlayerIndex = 0

	dst := gs.GetPlayer("dst")
	dst.Options.AutoLeechMode = LeechAutoModeAccept2
	dst.Resources.Power = NewPowerSystem(0, 4, 0)
	dst.VictoryPoints = 20

	gs.PendingLeechOffers["dst"] = []*PowerLeechOffer{
		{Amount: 2, FromPlayerID: "src", EventID: 1},
	}

	if err := gs.ResolveAutoLeechOffers(); err != nil {
		t.Fatalf("resolve auto leech: %v", err)
	}

	if got := len(gs.PendingLeechOffers["dst"]); got != 0 {
		t.Fatalf("expected no pending offers, got %d", got)
	}
	if got := dst.Resources.Power.Bowl3; got != 2 {
		t.Fatalf("expected bowlIII=2 after accepting leech, got %d", got)
	}
	if got := dst.VictoryPoints; got != 19 {
		t.Fatalf("expected VP=19 after accepting 2-power leech, got %d", got)
	}
}

func TestResolveAutoLeechOffers_CultistsSourceRequiresManualChoice(t *testing.T) {
	gs := NewGameState()
	mustAddPlayer(t, gs, "src", factions.NewCultists())
	mustAddPlayer(t, gs, "dst", factions.NewAuren())
	gs.TurnOrder = []string{"src", "dst"}
	gs.CurrentPlayerIndex = 0

	dst := gs.GetPlayer("dst")
	dst.Options.AutoLeechMode = LeechAutoModeAccept4
	dst.VictoryPoints = 20

	gs.PendingLeechOffers["dst"] = []*PowerLeechOffer{
		{Amount: 2, FromPlayerID: "src", EventID: 1},
	}

	if err := gs.ResolveAutoLeechOffers(); err != nil {
		t.Fatalf("resolve auto leech: %v", err)
	}

	if got := len(gs.PendingLeechOffers["dst"]); got != 1 {
		t.Fatalf("expected offer to remain pending for manual decision, got %d", got)
	}
	if got := dst.VictoryPoints; got != 20 {
		t.Fatalf("expected VP unchanged, got %d", got)
	}
}

func TestResolveAutoLeechOffers_ShapeshiftersSourceRequiresManualChoice(t *testing.T) {
	gs := NewGameState()
	mustAddPlayer(t, gs, "src", factions.NewShapeshifters())
	mustAddPlayer(t, gs, "dst", factions.NewAuren())
	gs.TurnOrder = []string{"src", "dst"}
	gs.CurrentPlayerIndex = 0

	dst := gs.GetPlayer("dst")
	dst.Options.AutoLeechMode = LeechAutoModeAccept4
	dst.VictoryPoints = 20

	gs.PendingLeechOffers["dst"] = []*PowerLeechOffer{
		{Amount: 2, FromPlayerID: "src", EventID: 1},
	}

	if err := gs.ResolveAutoLeechOffers(); err != nil {
		t.Fatalf("resolve auto leech: %v", err)
	}

	if got := len(gs.PendingLeechOffers["dst"]); got != 1 {
		t.Fatalf("expected offer to remain pending for manual decision, got %d", got)
	}
	if got := dst.VictoryPoints; got != 20 {
		t.Fatalf("expected VP unchanged, got %d", got)
	}
}

func TestResolveAutoLeechOffers_PassedIncomeSaturationAutoDeclines(t *testing.T) {
	gs := NewGameState()
	gs.Round = 1
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				CultTrack:        CultFire,
				CultThreshold:    1,
				CultRewardType:   CultRewardPower,
				CultRewardAmount: 8,
			},
		},
		PriestsSent: map[string]int{},
	}

	mustAddPlayer(t, gs, "src", factions.NewNomads())
	mustAddPlayer(t, gs, "dst", factions.NewAuren())
	gs.TurnOrder = []string{"src", "dst"}
	gs.CurrentPlayerIndex = 0

	gs.CultTracks.PlayerPositions["dst"][CultFire] = 1

	dst := gs.GetPlayer("dst")
	dst.Options.AutoLeechMode = LeechAutoModeAccept4
	dst.HasPassed = true
	dst.Resources.Power = NewPowerSystem(1, 6, 3)
	dst.VictoryPoints = 20

	gs.PendingLeechOffers["dst"] = []*PowerLeechOffer{
		{Amount: 2, FromPlayerID: "src", EventID: 3},
	}

	if err := gs.ResolveAutoLeechOffers(); err != nil {
		t.Fatalf("resolve auto leech: %v", err)
	}

	if got := len(gs.PendingLeechOffers["dst"]); got != 0 {
		t.Fatalf("expected offer to be auto-declined and removed, got %d", got)
	}
	if got := dst.VictoryPoints; got != 20 {
		t.Fatalf("expected VP unchanged from auto-decline, got %d", got)
	}
	if dst.Resources.Power.Bowl1 != 1 || dst.Resources.Power.Bowl2 != 6 || dst.Resources.Power.Bowl3 != 3 {
		t.Fatalf("expected power unchanged on auto-decline, got %d/%d/%d", dst.Resources.Power.Bowl1, dst.Resources.Power.Bowl2, dst.Resources.Power.Bowl3)
	}
}

func TestApplyAutoConvertOnPass_PriestOverflowToWorkers(t *testing.T) {
	gs := NewGameState()
	gs.Round = 1
	gs.ScoringTiles = &ScoringTileState{
		Tiles: []ScoringTile{
			{
				CultTrack:        CultFire,
				CultThreshold:    1,
				CultRewardType:   CultRewardPriest,
				CultRewardAmount: 4,
			},
		},
		PriestsSent: map[string]int{},
	}

	mustAddPlayer(t, gs, "p1", factions.NewNomads())
	gs.CultTracks.PlayerPositions["p1"][CultFire] = 1
	gs.CultTracks.PriestsOnActionSpaces["p1"][CultFire] = 2

	player := gs.GetPlayer("p1")
	player.Options.AutoConvertOnPass = true
	player.Resources.Priests = 2
	player.Resources.Workers = 0

	gs.ApplyAutoConvertOnPass("p1")

	if got := player.Resources.Priests; got != 1 {
		t.Fatalf("expected priests to reduce by overflow conversion, got %d", got)
	}
	if got := player.Resources.Workers; got != 1 {
		t.Fatalf("expected workers to increase from priest overflow conversion, got %d", got)
	}
}

func TestApplyAutoConvertOnPass_PowerToCoinOnlyWhenGuaranteedFullBowl(t *testing.T) {
	tests := []struct {
		name          string
		incomePower   int
		wantPower     [3]int
		wantCoinDelta int
	}{
		{
			name:          "converts when full-bowl guarantee is preserved",
			incomePower:   7,
			wantPower:     [3]int{1, 5, 2},
			wantCoinDelta: 1,
		},
		{
			name:          "does not convert when full-bowl guarantee would be lost",
			incomePower:   6,
			wantPower:     [3]int{0, 5, 3},
			wantCoinDelta: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGameState()
			gs.Round = 1
			gs.ScoringTiles = &ScoringTileState{
				Tiles: []ScoringTile{
					{
						CultTrack:        CultFire,
						CultThreshold:    1,
						CultRewardType:   CultRewardPower,
						CultRewardAmount: tt.incomePower,
					},
				},
				PriestsSent: map[string]int{},
			}

			mustAddPlayer(t, gs, "p1", factions.NewNomads())
			gs.CultTracks.PlayerPositions["p1"][CultFire] = 1
			player := gs.GetPlayer("p1")
			player.Options.AutoConvertOnPass = true
			player.Resources.Power = NewPowerSystem(0, 5, 3)
			coinsBefore := player.Resources.Coins

			gs.ApplyAutoConvertOnPass("p1")

			if player.Resources.Power.Bowl1 != tt.wantPower[0] ||
				player.Resources.Power.Bowl2 != tt.wantPower[1] ||
				player.Resources.Power.Bowl3 != tt.wantPower[2] {
				t.Fatalf(
					"unexpected power bowls: got %d/%d/%d want %d/%d/%d",
					player.Resources.Power.Bowl1, player.Resources.Power.Bowl2, player.Resources.Power.Bowl3,
					tt.wantPower[0], tt.wantPower[1], tt.wantPower[2],
				)
			}
			if gotDelta := player.Resources.Coins - coinsBefore; gotDelta != tt.wantCoinDelta {
				t.Fatalf("unexpected coin delta: got %d want %d", gotDelta, tt.wantCoinDelta)
			}
		})
	}
}
