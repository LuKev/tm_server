package main

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/notation"
)

func TestCreateInitialState_EnablesReplayMode(t *testing.T) {
	state := createInitialState([]notation.LogItem{})
	if state == nil {
		t.Fatal("createInitialState returned nil")
	}
	if state.ReplayMode == nil || !state.ReplayMode["__replay__"] {
		t.Fatalf("ReplayMode[__replay__] = %v, want true", state.ReplayMode)
	}
}

func TestCreateInitialState_UsesParsedMap(t *testing.T) {
	state := createInitialState([]notation.LogItem{
		notation.GameSettingsItem{Settings: map[string]string{"Game": "Lakes"}},
	})
	if state == nil || state.Map == nil {
		t.Fatal("createInitialState returned nil map")
	}
	if state.Map.ID != board.MapLakes {
		t.Fatalf("map ID = %q, want %q", state.Map.ID, board.MapLakes)
	}
}

func TestInjectStartingCultChoices_InsertsDjinniChoiceAfterSettings(t *testing.T) {
	items := []notation.LogItem{
		notation.GameSettingsItem{Settings: map[string]string{"Player:alice": "Djinni"}},
		notation.ActionItem{Action: game.NewAdvanceShippingAction("Djinni")},
	}

	got := injectStartingCultChoices(items, map[string]string{"Djinni": "earth"})
	if len(got) != 3 {
		t.Fatalf("len(injected items) = %d, want 3", len(got))
	}

	actionItem, ok := got[1].(notation.ActionItem)
	if !ok {
		t.Fatalf("item[1] type = %T, want notation.ActionItem", got[1])
	}
	action, ok := actionItem.Action.(*game.SelectDjinniStartingCultTrackAction)
	if !ok {
		t.Fatalf("item[1].Action type = %T, want *game.SelectDjinniStartingCultTrackAction", actionItem.Action)
	}
	if action.PlayerID != "Djinni" || action.CultTrack != game.CultEarth {
		t.Fatalf("injected action = %+v, want Djinni earth choice", action)
	}
}
