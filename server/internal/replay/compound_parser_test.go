package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

func TestParseConversion(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantType ConversionType
		wantAmt  int
		wantFrom string
		wantTo   string
		wantOk   bool
	}{
		{
			name:     "burn power",
			token:    "burn 3",
			wantType: ConvBurn,
			wantAmt:  0,
			wantOk:   false, // burn is not handled by parseConversion
		},
		{
			name:     "power to coins",
			token:    "convert 1PW to 1C",
			wantType: ConvPowerToCoins,
			wantAmt:  1,
			wantFrom: "PW",
			wantTo:   "C",
			wantOk:   true,
		},
		{
			name:     "power to workers",
			token:    "convert 3PW to 1W",
			wantType: ConvPowerToWorkers,
			wantAmt:  1,
			wantFrom: "PW",
			wantTo:   "W",
			wantOk:   true,
		},
		{
			name:     "power to priests",
			token:    "convert 5PW to 1P",
			wantType: ConvPowerToPriests,
			wantAmt:  1,
			wantFrom: "PW",
			wantTo:   "P",
			wantOk:   true,
		},
		{
			name:     "priest to worker",
			token:    "convert 1P to 1W",
			wantType: ConvPriestToWorker,
			wantAmt:  1,
			wantFrom: "P",
			wantTo:   "W",
			wantOk:   true,
		},
		{
			name:     "worker to coin",
			token:    "convert 1W to 1C",
			wantType: ConvWorkerToCoin,
			wantAmt:  1,
			wantFrom: "W",
			wantTo:   "C",
			wantOk:   true,
		},
		{
			name:     "multiple workers to coins",
			token:    "convert 3W to 3C",
			wantType: ConvWorkerToCoin,
			wantAmt:  3,
			wantFrom: "W",
			wantTo:   "C",
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, ok := parseConversion(tt.token)
			if ok != tt.wantOk {
				t.Errorf("parseConversion(%q) ok = %v, want %v", tt.token, ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if conv.Type != tt.wantType {
				t.Errorf("parseConversion(%q) type = %v, want %v", tt.token, conv.Type, tt.wantType)
			}
			if conv.Amount != tt.wantAmt {
				t.Errorf("parseConversion(%q) amount = %v, want %v", tt.token, conv.Amount, tt.wantAmt)
			}
			if conv.From != tt.wantFrom {
				t.Errorf("parseConversion(%q) from = %v, want %v", tt.token, conv.From, tt.wantFrom)
			}
			if conv.To != tt.wantTo {
				t.Errorf("parseConversion(%q) to = %v, want %v", tt.token, conv.To, tt.wantTo)
			}
		})
	}
}

func TestParseAuxiliary(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantType AuxiliaryType
		wantTile string
		wantOk   bool
	}{
		{
			name:     "favor tile",
			token:    "+FAV5",
			wantType: AuxFavorTile,
			wantTile: "+FAV5",
			wantOk:   true,
		},
		{
			name:     "town tile",
			token:    "+TW3",
			wantType: AuxTownTile,
			wantTile: "+TW3",
			wantOk:   true,
		},
		{
			name:   "not auxiliary",
			token:  "build E5",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aux, ok := parseAuxiliary(tt.token)
			if ok != tt.wantOk {
				t.Errorf("parseAuxiliary(%q) ok = %v, want %v", tt.token, ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if aux.Type != tt.wantType {
				t.Errorf("parseAuxiliary(%q) type = %v, want %v", tt.token, aux.Type, tt.wantType)
			}
			if aux.Params["tile"] != tt.wantTile {
				t.Errorf("parseAuxiliary(%q) tile = %v, want %v", tt.token, aux.Params["tile"], tt.wantTile)
			}
		})
	}
}

func TestParseMainActionToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantType ActionType
		wantOk   bool
	}{
		{
			name:     "build action",
			token:    "build E7",
			wantType: ActionBuild,
			wantOk:   true,
		},
		{
			name:     "upgrade action",
			token:    "upgrade E5 to TP",
			wantType: ActionUpgrade,
			wantOk:   true,
		},
		{
			name:     "send priest",
			token:    "send p to WATER",
			wantType: ActionSendPriest,
			wantOk:   true,
		},
		{
			name:     "advance shipping",
			token:    "advance ship",
			wantType: ActionAdvanceShipping,
			wantOk:   true,
		},
		{
			name:     "advance digging",
			token:    "advance dig",
			wantType: ActionAdvanceDigging,
			wantOk:   true,
		},
		{
			name:     "pass",
			token:    "pass BON1",
			wantType: ActionPass,
			wantOk:   true,
		},
		{
			name:   "not a main action",
			token:  "convert 1W to 1C",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			part, ok := parseMainActionToken(tt.token)
			if ok != tt.wantOk {
				t.Errorf("parseMainActionToken(%q) ok = %v, want %v", tt.token, ok, tt.wantOk)
				return
			}
			if !tt.wantOk {
				return
			}
			if part.Type != tt.wantType {
				t.Errorf("parseMainActionToken(%q) type = %v, want %v", tt.token, part.Type, tt.wantType)
			}
		})
	}
}

func TestParseCompoundAction_Simple(t *testing.T) {
	// Create a minimal game state for testing
	gs := game.NewGameState()
	gs.AddPlayer("engineers", &factions.Engineers{})

	entry := &LogEntry{
		Faction: models.FactionEngineers,
		Action:  "convert 1PW to 1C",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	if len(compound.Components) != 1 {
		t.Errorf("expected 1 component, got %d", len(compound.Components))
	}

	// Check it's a conversion
	conv, ok := compound.Components[0].(*ConversionComponent)
	if !ok {
		t.Fatalf("expected ConversionComponent, got %T", compound.Components[0])
	}

	if conv.Type != ConvPowerToCoins {
		t.Errorf("expected ConvPowerToCoins, got %v", conv.Type)
	}
}

func TestParseCompoundAction_ConvertAndPass(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("engineers", &factions.Engineers{})

	entry := &LogEntry{
		Faction: models.FactionEngineers,
		Action:  "convert 1PW to 1C. pass BON7",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	if len(compound.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(compound.Components))
	}

	// Check first is conversion
	_, ok := compound.Components[0].(*ConversionComponent)
	if !ok {
		t.Errorf("component 0: expected ConversionComponent, got %T", compound.Components[0])
	}

	// Check second is main action (pass)
	_, ok = compound.Components[1].(*MainActionComponent)
	if !ok {
		t.Errorf("component 1: expected MainActionComponent, got %T", compound.Components[1])
	}
}

func TestParseCompoundAction_MultipleConversions(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("engineers", &factions.Engineers{})

	entry := &LogEntry{
		Faction: models.FactionEngineers,
		Action:  "convert 2PW to 2C. convert 1W to 1C. upgrade F2 to TP. +TW5",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have: 2 conversions + 1 main action + 1 auxiliary
	if len(compound.Components) != 4 {
		t.Errorf("expected 4 components, got %d", len(compound.Components))
		for i, comp := range compound.Components {
			t.Logf("  %d: %s", i, comp.String())
		}
	}
}

func TestParseCompoundAction_UpgradeFavorTile(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("cultists", &factions.Cultists{})

	entry := &LogEntry{
		Faction: models.FactionCultists,
		Action:  "upgrade F5 to TE. +FAV11",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have: 1 main action (upgrade) + 1 auxiliary (favor tile)
	if len(compound.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(compound.Components))
		for i, comp := range compound.Components {
			t.Logf("  %d: %s", i, comp.String())
		}
		t.FailNow()
	}

	// Component 0 should be MainAction (upgrade)
	if _, ok := compound.Components[0].(*MainActionComponent); !ok {
		t.Errorf("component 0 should be MainActionComponent, got %T", compound.Components[0])
	}

	// Component 1 should be AuxiliaryComponent (favor tile)
	if aux, ok := compound.Components[1].(*AuxiliaryComponent); !ok {
		t.Errorf("component 1 should be AuxiliaryComponent, got %T", compound.Components[1])
	} else if aux.Type != AuxFavorTile {
		t.Errorf("auxiliary type should be AuxFavorTile, got %v", aux.Type)
	}
}

func TestParseCompoundAction_BurnConvertAdvance(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("engineers", &factions.Engineers{})

	entry := &LogEntry{
		Faction: models.FactionEngineers,
		Action:  "burn 1. convert 1PW to 1C. convert 3W to 3C. advance ship",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have: burn + 2 conversions + 1 main action
	if len(compound.Components) != 4 {
		t.Errorf("expected 4 components, got %d", len(compound.Components))
		for i, comp := range compound.Components {
			t.Logf("  %d: %s", i, comp.String())
		}
	}

	// Check first is burn conversion
	conv, ok := compound.Components[0].(*ConversionComponent)
	if !ok {
		t.Fatalf("component 0: expected ConversionComponent, got %T", compound.Components[0])
	}
	if conv.Type != ConvBurn {
		t.Errorf("component 0: expected ConvBurn, got %v", conv.Type)
	}
}

func TestParseCompoundAction_ConvertBetweenActions(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("engineers", &factions.Engineers{})

	entry := &LogEntry{
		Faction: models.FactionEngineers,
		Action:  "convert 1PW to 1C. send p to EARTH. convert 1PW to 1C",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have: conversion + main action + conversion
	if len(compound.Components) != 3 {
		t.Errorf("expected 3 components, got %d", len(compound.Components))
		for i, comp := range compound.Components {
			t.Logf("  %d: %s", i, comp.String())
		}
	}

	// Check pattern: conversion, main, conversion
	if _, ok := compound.Components[0].(*ConversionComponent); !ok {
		t.Errorf("component 0: expected ConversionComponent, got %T", compound.Components[0])
	}
	if _, ok := compound.Components[1].(*MainActionComponent); !ok {
		t.Errorf("component 1: expected MainActionComponent, got %T", compound.Components[1])
	}
	if _, ok := compound.Components[2].(*ConversionComponent); !ok {
		t.Errorf("component 2: expected ConversionComponent, got %T", compound.Components[2])
	}
}
