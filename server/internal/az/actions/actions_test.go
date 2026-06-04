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

func TestLegalActionsExcludeMainTurnTransformOnly(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	hasTransformBuild := false
	for _, option := range legal {
		if option.Type == "transform" {
			t.Fatalf("main-turn transform-only action should be pruned from AZ surface: %s", option.ID)
		}
		if option.Type == "transform_build" {
			hasTransformBuild = true
		}
	}
	if !hasTransformBuild {
		t.Fatal("expected transform/build actions to remain legal")
	}
}

func TestLegalActionsIncludeExecutablePass(t *testing.T) {
	position, err := env.BuiltInScenario("base_nomads_witches")
	if err != nil {
		t.Fatalf("BuiltInScenario failed: %v", err)
	}
	legal := actions.LegalActions(position.State)
	for _, option := range legal {
		if option.Type != "pass" {
			continue
		}
		if _, err := actions.ApplyToClone(position.State, option.Action); err != nil {
			t.Fatalf("pass action %s did not apply: %v", option.ID, err)
		}
		return
	}
	t.Fatal("expected at least one legal pass action")
}
