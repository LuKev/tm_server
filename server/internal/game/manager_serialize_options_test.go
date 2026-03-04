package game

import "testing"

func TestSerializeStateWithRevision_IncludesPlayerOptions(t *testing.T) {
	gs := NewGameState()
	if err := gs.AddPlayer("p1", nil); err != nil {
		t.Fatalf("add player: %v", err)
	}

	state := SerializeStateWithRevision(gs, "g1", 0)
	playersRaw, ok := state["players"].(map[string]interface{})
	if !ok {
		t.Fatalf("players missing from serialized state")
	}
	playerRaw, ok := playersRaw["p1"].(map[string]interface{})
	if !ok {
		t.Fatalf("player p1 missing from serialized state")
	}
	optionsRaw, ok := playerRaw["options"].(PlayerOptions)
	if !ok {
		t.Fatalf("player options missing from serialized state")
	}
	if optionsRaw.AutoLeechMode != LeechAutoModeOff {
		t.Fatalf("unexpected auto leech mode: got %q", optionsRaw.AutoLeechMode)
	}
	if optionsRaw.AutoConvertOnPass {
		t.Fatalf("expected auto convert on pass to default false")
	}
	if !optionsRaw.ConfirmActions {
		t.Fatalf("expected confirm actions to default true")
	}
	if optionsRaw.ShowIncomePreview {
		t.Fatalf("expected show income preview to default false")
	}
}
