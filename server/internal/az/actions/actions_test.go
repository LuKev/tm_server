package actions_test

import (
	"testing"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
)

func TestLegalActionsAreExecutableOnScenario(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	if len(legal) == 0 {
		t.Fatal("expected legal actions")
	}
	for _, option := range legal {
		if _, err := actions.ApplyToClone(position.State, option.Action); err != nil {
			t.Fatalf("legal action %s did not apply: %v", option.ID, err)
		}
	}
}
