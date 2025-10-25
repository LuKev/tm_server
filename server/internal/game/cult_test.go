package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
)

func TestCultTrackState_InitializePlayer(t *testing.T) {
	cts := NewCultTrackState()
	cts.InitializePlayer("player1")

	// Verify all tracks start at 0
	if cts.GetPosition("player1", CultFire) != 0 {
		t.Error("expected Fire track to start at 0")
	}
	if cts.GetPosition("player1", CultWater) != 0 {
		t.Error("expected Water track to start at 0")
	}
	if cts.GetPosition("player1", CultEarth) != 0 {
		t.Error("expected Earth track to start at 0")
	}
	if cts.GetPosition("player1", CultAir) != 0 {
		t.Error("expected Air track to start at 0")
	}
}

func TestCultTrackState_AdvancePlayer(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up power for gaining
	player.Resources.Power.Bowl1 = 10
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Advance 3 spaces on Fire track
	advanced, err := gs.CultTracks.AdvancePlayer("player1", CultFire, 3, player)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if advanced != 3 {
		t.Errorf("expected to advance 3 spaces, got %d", advanced)
	}

	// Verify position
	if gs.CultTracks.GetPosition("player1", CultFire) != 3 {
		t.Errorf("expected position 3, got %d", gs.CultTracks.GetPosition("player1", CultFire))
	}

	// Verify power was gained (only 1 bonus for reaching position 3, no base power)
	if player.Resources.Power.Bowl1 != 9 {
		t.Errorf("expected 9 power in Bowl1, got %d", player.Resources.Power.Bowl1)
	}
	if player.Resources.Power.Bowl2 != 1 {
		t.Errorf("expected 1 power in Bowl2 (bonus only), got %d", player.Resources.Power.Bowl2)
	}
}

func TestCultTrackState_AdvancePlayer_MaxPosition(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up power
	player.Resources.Power.Bowl1 = 20

	// Give player a key to reach position 10
	player.Keys = 1

	// Advance to position 8
	gs.CultTracks.AdvancePlayer("player1", CultWater, 8, player)

	// Try to advance 5 more spaces (should only advance 2 to reach max of 10)
	advanced, err := gs.CultTracks.AdvancePlayer("player1", CultWater, 5, player)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if advanced != 2 {
		t.Errorf("expected to advance 2 spaces (to max), got %d", advanced)
	}

	// Verify position is at max
	if gs.CultTracks.GetPosition("player1", CultWater) != 10 {
		t.Errorf("expected position 10 (max), got %d", gs.CultTracks.GetPosition("player1", CultWater))
	}

	// Try to advance further (should return 0)
	advanced, err = gs.CultTracks.AdvancePlayer("player1", CultWater, 3, player)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if advanced != 0 {
		t.Errorf("expected to advance 0 spaces (already at max), got %d", advanced)
	}
}

func TestCultTrackState_GetRankings(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren()) // Forest
	gs.AddPlayer("player2", factions.NewAlchemists()) // Swamp - different from Auren
	gs.AddPlayer("player3", factions.NewGiants())

	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")

	// Set up power for all players
	player1.Resources.Power.Bowl1 = 20
	player2.Resources.Power.Bowl1 = 20
	player3.Resources.Power.Bowl1 = 20

	// Advance players to different positions
	gs.CultTracks.AdvancePlayer("player1", CultFire, 7, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 3, player2)
	gs.CultTracks.AdvancePlayer("player3", CultFire, 5, player3)

	// Get rankings
	rankings := gs.CultTracks.GetRankings(CultFire)

	// Verify order: player1 (7), player3 (5), player2 (3)
	if len(rankings) != 3 {
		t.Fatalf("expected 3 players in rankings, got %d", len(rankings))
	}
	if rankings[0] != "player1" {
		t.Errorf("expected player1 in 1st place, got %s", rankings[0])
	}
	if rankings[1] != "player3" {
		t.Errorf("expected player3 in 2nd place, got %s", rankings[1])
	}
	if rankings[2] != "player2" {
		t.Errorf("expected player2 in 3rd place, got %s", rankings[2])
	}
}

func TestCultTrackState_EndGameScoring_Simple(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	gs.AddPlayer("player2", factions.NewAlchemists()) // Swamp - different from Auren (Forest)
	gs.AddPlayer("player3", factions.NewGiants())

	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")

	// Set up power
	player1.Resources.Power.Bowl1 = 20
	player2.Resources.Power.Bowl1 = 20
	player3.Resources.Power.Bowl1 = 20

	// Give player1 a key to reach position 10
	player1.Keys = 1

	// Fire track: player1 (10), player2 (5), player3 (2)
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 5, player2)
	gs.CultTracks.AdvancePlayer("player3", CultFire, 2, player3)

	// Calculate scoring
	vpByPlayer := gs.CultTracks.CalculateEndGameScoring()

	// Expected: player1=8, player2=4, player3=2
	if vpByPlayer["player1"] != 8 {
		t.Errorf("expected player1 to get 8 VP, got %d", vpByPlayer["player1"])
	}
	if vpByPlayer["player2"] != 4 {
		t.Errorf("expected player2 to get 4 VP, got %d", vpByPlayer["player2"])
	}
	if vpByPlayer["player3"] != 2 {
		t.Errorf("expected player3 to get 2 VP, got %d", vpByPlayer["player3"])
	}
}

func TestCultTrackState_EndGameScoring_Tie(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	gs.AddPlayer("player2", factions.NewAlchemists()) // Swamp - different from Auren (Forest)
	gs.AddPlayer("player3", factions.NewGiants())

	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")
	player3 := gs.GetPlayer("player3")

	// Set up power
	player1.Resources.Power.Bowl1 = 20
	player2.Resources.Power.Bowl1 = 20
	player3.Resources.Power.Bowl1 = 20

	// Fire track: player1 (7), player2 (7), player3 (3)
	// Tied for 1st: split 8+4=12 points -> 6 each
	// 3rd place: 2 points
	gs.CultTracks.AdvancePlayer("player1", CultFire, 7, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 7, player2)
	gs.CultTracks.AdvancePlayer("player3", CultFire, 3, player3)

	// Calculate scoring
	vpByPlayer := gs.CultTracks.CalculateEndGameScoring()

	// Expected: player1=6, player2=6, player3=2
	if vpByPlayer["player1"] != 6 {
		t.Errorf("expected player1 to get 6 VP (tied 1st), got %d", vpByPlayer["player1"])
	}
	if vpByPlayer["player2"] != 6 {
		t.Errorf("expected player2 to get 6 VP (tied 1st), got %d", vpByPlayer["player2"])
	}
	if vpByPlayer["player3"] != 2 {
		t.Errorf("expected player3 to get 2 VP (3rd), got %d", vpByPlayer["player3"])
	}
}

func TestCultTrackState_EndGameScoring_MultipleTracks(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	gs.AddPlayer("player2", factions.NewAlchemists()) // Swamp - different from Auren (Forest)

	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")

	// Set up power
	player1.Resources.Power.Bowl1 = 40
	player2.Resources.Power.Bowl1 = 40

	// Give player1 a key to reach position 10
	player1.Keys = 1

	// Fire track: player1 (10), player2 (5)
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1)
	gs.CultTracks.AdvancePlayer("player2", CultFire, 5, player2)

	// Water track: player2 (8), player1 (3)
	gs.CultTracks.AdvancePlayer("player2", CultWater, 8, player2)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 3, player1)

	// Calculate scoring
	vpByPlayer := gs.CultTracks.CalculateEndGameScoring()

	// Expected:
	// Fire: player1=8, player2=4
	// Water: player2=8, player1=4
	// Total: player1=12, player2=12
	if vpByPlayer["player1"] != 12 {
		t.Errorf("expected player1 to get 12 VP total, got %d", vpByPlayer["player1"])
	}
	if vpByPlayer["player2"] != 12 {
		t.Errorf("expected player2 to get 12 VP total, got %d", vpByPlayer["player2"])
	}
}

func TestCultTrackState_EndGameScoring_NoAdvancement(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	gs.AddPlayer("player2", factions.NewAlchemists()) // Swamp - different from Auren (Forest)

	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")

	// Set up power
	player1.Resources.Power.Bowl1 = 20
	player2.Resources.Power.Bowl1 = 20

	// Only player1 advances on Fire
	gs.CultTracks.AdvancePlayer("player1", CultFire, 5, player1)
	// player2 doesn't advance at all

	// Calculate scoring
	vpByPlayer := gs.CultTracks.CalculateEndGameScoring()

	// Expected: player1=8 (1st place), player2=0 (no advancement)
	if vpByPlayer["player1"] != 8 {
		t.Errorf("expected player1 to get 8 VP, got %d", vpByPlayer["player1"])
	}
	if vpByPlayer["player2"] != 0 {
		t.Errorf("expected player2 to get 0 VP (no advancement), got %d", vpByPlayer["player2"])
	}
}

func TestCultTrackState_BonusPower(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up power
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player a key to reach position 10
	player.Keys = 1

	// Advance to position 5 (should get 1 bonus at pos 3 + 2 bonus at pos 5 = 3 total, no base power)
	advanced, err := gs.CultTracks.AdvancePlayer("player1", CultFire, 5, player)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if advanced != 5 {
		t.Errorf("expected to advance 5 spaces, got %d", advanced)
	}

	// Verify power gained: 1 (pos 3) + 2 (pos 5) = 3 total
	expectedBowl1 := 20 - 3
	expectedBowl2 := 3
	if player.Resources.Power.Bowl1 != expectedBowl1 {
		t.Errorf("expected %d power in Bowl1, got %d", expectedBowl1, player.Resources.Power.Bowl1)
	}
	if player.Resources.Power.Bowl2 != expectedBowl2 {
		t.Errorf("expected %d power in Bowl2, got %d", expectedBowl2, player.Resources.Power.Bowl2)
	}

	// Advance to position 10 (should get 2 bonus at pos 7 + 3 bonus at pos 10 = 5 total, no base power)
	advanced, err = gs.CultTracks.AdvancePlayer("player1", CultFire, 5, player)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if advanced != 5 {
		t.Errorf("expected to advance 5 spaces, got %d", advanced)
	}

	// Verify power gained: previous 3 + 2 (pos 7) + 3 (pos 10) = 8 total
	expectedBowl1 = 20 - 8
	expectedBowl2 = 8
	if player.Resources.Power.Bowl1 != expectedBowl1 {
		t.Errorf("expected %d power in Bowl1, got %d", expectedBowl1, player.Resources.Power.Bowl1)
	}
	if player.Resources.Power.Bowl2 != expectedBowl2 {
		t.Errorf("expected %d power in Bowl2, got %d", expectedBowl2, player.Resources.Power.Bowl2)
	}
}

func TestCultTrackState_Position10Blocked(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("player1", factions.NewAuren())
	gs.AddPlayer("player2", factions.NewAlchemists()) // Swamp - different from Auren (Forest)

	player1 := gs.GetPlayer("player1")
	player2 := gs.GetPlayer("player2")

	// Set up power
	player1.Resources.Power.Bowl1 = 20
	player2.Resources.Power.Bowl1 = 20

	// Give player1 a key to reach position 10
	player1.Keys = 1

	// Player 1 reaches position 10
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player1)

	if gs.CultTracks.GetPosition("player1", CultFire) != 10 {
		t.Errorf("expected player1 at position 10")
	}

	// Player 2 tries to advance to position 10 (should be blocked at 9)
	advanced, err := gs.CultTracks.AdvancePlayer("player2", CultFire, 10, player2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if advanced != 9 {
		t.Errorf("expected to advance 9 spaces (blocked at 9), got %d", advanced)
	}
	if gs.CultTracks.GetPosition("player2", CultFire) != 9 {
		t.Errorf("expected player2 at position 9 (blocked), got %d", gs.CultTracks.GetPosition("player2", CultFire))
	}
}

func TestSendPriestToCult_Basic(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources (clear starting power and add priests)
	player.Resources.Priests = 3
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Send priest to Fire track (use slot value 3)
	action := &SendPriestToCultAction{
		BaseAction: BaseAction{
			Type:     ActionSendPriestToCult,
			PlayerID: "player1",
		},
		Track:         CultFire,
		UsePriestSlot: true,
		SlotValue:     3,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify priest was consumed
	if player.Resources.Priests != 2 {
		t.Errorf("expected 2 priests remaining, got %d", player.Resources.Priests)
	}

	// Verify position advanced by 3
	if gs.CultTracks.GetPosition("player1", CultFire) != 3 {
		t.Errorf("expected position 3, got %d", gs.CultTracks.GetPosition("player1", CultFire))
	}

	// Verify power gained: 1 bonus at position 3 only (no base power)
	if player.Resources.Power.Bowl2 != 1 {
		t.Errorf("expected 1 power in Bowl2, got %d", player.Resources.Power.Bowl2)
	}
}

func TestSendPriestToCult_ReturnToSupply(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources (clear starting power and add priests)
	player.Resources.Priests = 3
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Send priest to Water track (return to supply, only 1 space)
	action := &SendPriestToCultAction{
		BaseAction: BaseAction{
			Type:     ActionSendPriestToCult,
			PlayerID: "player1",
		},
		Track:         CultWater,
		UsePriestSlot: false, // Return to supply
		SlotValue:     0,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify priest was consumed
	if player.Resources.Priests != 2 {
		t.Errorf("expected 2 priests remaining, got %d", player.Resources.Priests)
	}

	// Verify position advanced by 1 only
	if gs.CultTracks.GetPosition("player1", CultWater) != 1 {
		t.Errorf("expected position 1, got %d", gs.CultTracks.GetPosition("player1", CultWater))
	}

	// Verify power gained: 0 (no milestone reached)
	if player.Resources.Power.Bowl2 != 0 {
		t.Errorf("expected 0 power in Bowl2, got %d", player.Resources.Power.Bowl2)
	}
}

func TestSendPriestToCult_NoPriest(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// No priests available
	player.Resources.Priests = 0

	action := &SendPriestToCultAction{
		BaseAction: BaseAction{
			Type:     ActionSendPriestToCult,
			PlayerID: "player1",
		},
		Track:         CultFire,
		UsePriestSlot: true,
		SlotValue:     3,
	}

	err := action.Execute(gs)
	if err == nil {
		t.Error("expected error when no priests available")
	}
}

func TestSendPriestToCult_AlreadyAtMax(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up resources
	player.Resources.Priests = 3
	player.Resources.Power.Bowl1 = 20

	// Give player a key to reach position 10
	player.Keys = 1

	// Advance to position 10
	gs.CultTracks.AdvancePlayer("player1", CultEarth, 10, player)

	// Send priest even though already at max (valid move - priest is sacrificed)
	action := &SendPriestToCultAction{
		BaseAction: BaseAction{
			Type:     ActionSendPriestToCult,
			PlayerID: "player1",
		},
		Track:         CultEarth,
		UsePriestSlot: true,
		SlotValue:     3,
	}

	err := action.Execute(gs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify priest was consumed (not refunded - valid sacrifice)
	if player.Resources.Priests != 2 {
		t.Errorf("expected 2 priests (priest consumed), got %d", player.Resources.Priests)
	}
	
	// Position should still be 10
	if gs.CultTracks.GetPosition("player1", CultEarth) != 10 {
		t.Errorf("expected position to remain at 10, got %d", gs.CultTracks.GetPosition("player1", CultEarth))
	}
}

func TestTownCultBonus_8Points(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up power
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Apply 8-point town bonus (+1 on all tracks)
	gs.CultTracks.ApplyTownCultBonus("player1", TownTile8Points, player)

	// Verify all tracks advanced by 1
	if gs.CultTracks.GetPosition("player1", CultFire) != 1 {
		t.Errorf("expected Fire position 1, got %d", gs.CultTracks.GetPosition("player1", CultFire))
	}
	if gs.CultTracks.GetPosition("player1", CultWater) != 1 {
		t.Errorf("expected Water position 1, got %d", gs.CultTracks.GetPosition("player1", CultWater))
	}
	if gs.CultTracks.GetPosition("player1", CultEarth) != 1 {
		t.Errorf("expected Earth position 1, got %d", gs.CultTracks.GetPosition("player1", CultEarth))
	}
	if gs.CultTracks.GetPosition("player1", CultAir) != 1 {
		t.Errorf("expected Air position 1, got %d", gs.CultTracks.GetPosition("player1", CultAir))
	}

	// Verify no power gained (no milestones reached at position 1)
	if player.Resources.Power.Bowl2 != 0 {
		t.Errorf("expected 0 power in Bowl2, got %d", player.Resources.Power.Bowl2)
	}
}

func TestTownCultBonus_2Keys(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up power
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Apply 2-key town bonus (+2 on all tracks)
	gs.CultTracks.ApplyTownCultBonus("player1", TownTile2Points, player)

	// Verify all tracks advanced by 2
	if gs.CultTracks.GetPosition("player1", CultFire) != 2 {
		t.Errorf("expected Fire position 2, got %d", gs.CultTracks.GetPosition("player1", CultFire))
	}
	if gs.CultTracks.GetPosition("player1", CultWater) != 2 {
		t.Errorf("expected Water position 2, got %d", gs.CultTracks.GetPosition("player1", CultWater))
	}
	if gs.CultTracks.GetPosition("player1", CultEarth) != 2 {
		t.Errorf("expected Earth position 2, got %d", gs.CultTracks.GetPosition("player1", CultEarth))
	}
	if gs.CultTracks.GetPosition("player1", CultAir) != 2 {
		t.Errorf("expected Air position 2, got %d", gs.CultTracks.GetPosition("player1", CultAir))
	}

	// Verify no power gained (no milestones reached at position 2)
	if player.Resources.Power.Bowl2 != 0 {
		t.Errorf("expected 0 power in Bowl2, got %d", player.Resources.Power.Bowl2)
	}
}

func TestTownCultBonus_WithMilestones(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up power
	player.Resources.Power.Bowl1 = 20
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Advance to position 2 on all tracks first
	gs.CultTracks.AdvancePlayer("player1", CultFire, 2, player)
	gs.CultTracks.AdvancePlayer("player1", CultWater, 2, player)
	gs.CultTracks.AdvancePlayer("player1", CultEarth, 2, player)
	gs.CultTracks.AdvancePlayer("player1", CultAir, 2, player)

	// Apply 8-point town bonus (+1 on all tracks, reaching position 3 on all)
	gs.CultTracks.ApplyTownCultBonus("player1", TownTile8Points, player)

	// Verify all tracks advanced to position 3
	if gs.CultTracks.GetPosition("player1", CultFire) != 3 {
		t.Errorf("expected Fire position 3, got %d", gs.CultTracks.GetPosition("player1", CultFire))
	}
	if gs.CultTracks.GetPosition("player1", CultWater) != 3 {
		t.Errorf("expected Water position 3, got %d", gs.CultTracks.GetPosition("player1", CultWater))
	}
	if gs.CultTracks.GetPosition("player1", CultEarth) != 3 {
		t.Errorf("expected Earth position 3, got %d", gs.CultTracks.GetPosition("player1", CultEarth))
	}
	if gs.CultTracks.GetPosition("player1", CultAir) != 3 {
		t.Errorf("expected Air position 3, got %d", gs.CultTracks.GetPosition("player1", CultAir))
	}

	// Verify power gained: 4 tracks × 1 bonus power at position 3 = 4 total
	if player.Resources.Power.Bowl2 != 4 {
		t.Errorf("expected 4 power in Bowl2 (4 milestones), got %d", player.Resources.Power.Bowl2)
	}
}

func TestTownCultBonus_Position10Capped(t *testing.T) {
	gs := NewGameState()
	faction := factions.NewAuren()
	gs.AddPlayer("player1", faction)
	player := gs.GetPlayer("player1")

	// Set up power
	player.Resources.Power.Bowl1 = 50
	player.Resources.Power.Bowl2 = 0
	player.Resources.Power.Bowl3 = 0

	// Give player keys to reach position 10 on multiple tracks
	player.Keys = 4

	// Advance to position 10 on Fire track
	gs.CultTracks.AdvancePlayer("player1", CultFire, 10, player)
	
	// Advance to position 9 on other tracks
	gs.CultTracks.AdvancePlayer("player1", CultWater, 9, player)
	gs.CultTracks.AdvancePlayer("player1", CultEarth, 9, player)
	gs.CultTracks.AdvancePlayer("player1", CultAir, 9, player)

	// Reset power for clean test
	initialBowl2 := player.Resources.Power.Bowl2
	
	// Apply 2-key town bonus (+2 on all tracks)
	// Fire: 10 → 10 (capped, no advancement)
	// Others: 9 → 10 (advance 1, not 2, due to cap)
	gs.CultTracks.ApplyTownCultBonus("player1", TownTile2Points, player)

	// Verify Fire stayed at 10
	if gs.CultTracks.GetPosition("player1", CultFire) != 10 {
		t.Errorf("expected Fire position 10 (capped), got %d", gs.CultTracks.GetPosition("player1", CultFire))
	}
	
	// Verify others advanced to 10
	if gs.CultTracks.GetPosition("player1", CultWater) != 10 {
		t.Errorf("expected Water position 10, got %d", gs.CultTracks.GetPosition("player1", CultWater))
	}
	if gs.CultTracks.GetPosition("player1", CultEarth) != 10 {
		t.Errorf("expected Earth position 10, got %d", gs.CultTracks.GetPosition("player1", CultEarth))
	}
	if gs.CultTracks.GetPosition("player1", CultAir) != 10 {
		t.Errorf("expected Air position 10, got %d", gs.CultTracks.GetPosition("player1", CultAir))
	}

	// Verify power gained: 3 tracks × 3 bonus power at position 10 = 9 total
	expectedPower := initialBowl2 + 9
	if player.Resources.Power.Bowl2 != expectedPower {
		t.Errorf("expected %d power in Bowl2 (3 position 10 bonuses), got %d", expectedPower, player.Resources.Power.Bowl2)
	}
}
