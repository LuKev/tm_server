package main

import (
	"testing"

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
