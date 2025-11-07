package replay

import (
	"testing"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// TestConvertAndUpgrade tests convert+upgrade compound actions
// This pattern was previously excluded and used state syncing to bypass validation
// Now we want to test that conversions execute first, then upgrade validates costs
func TestConvertAndUpgrade_PowerToCoins(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("Engineers", &factions.Engineers{})

	player := gs.GetPlayer("Engineers")
	player.Resources.Coins = 4       // Start with 4 coins (need 6 total, will get 2 from conversion)
	player.Resources.Workers = 4     // Stronghold needs 4 workers
	player.Resources.Priests = 0     // Don't need priests for Stronghold
	player.Resources.Power.Bowl3 = 6 // Have 6 power in bowl 3 (convert 2 -> 2 coins)

	// Place a trading house at E9 to upgrade to Stronghold
	hex, _ := ConvertLogCoordToAxial("E9")
	gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(hex).Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    models.FactionEngineers,
		PlayerID:   "Engineers",
		PowerValue: 2,
	}

	// Parse: "convert 2PW to 2C. upgrade E9 to SH"
	entry := &LogEntry{
		Faction: models.FactionEngineers,
		Action:  "convert 2PW to 2C. upgrade E9 to SH",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have 2 components: conversion + upgrade
	if len(compound.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(compound.Components))
	}

	// Verify first component is conversion
	conv, ok := compound.Components[0].(*ConversionComponent)
	if !ok {
		t.Fatalf("expected ConversionComponent, got %T", compound.Components[0])
	}
	if conv.Type != ConvPowerToCoins {
		t.Errorf("expected ConvPowerToCoins, got %v", conv.Type)
	}
	if conv.Amount != 2 {
		t.Errorf("expected amount 2, got %d", conv.Amount)
	}

	// Verify second component is upgrade
	mainAction, ok := compound.Components[1].(*MainActionComponent)
	if !ok {
		t.Fatalf("expected MainActionComponent, got %T", compound.Components[1])
	}
	if mainAction.Action.GetType() != game.ActionUpgradeBuilding {
		t.Errorf("expected ActionUpgradeBuilding, got %v", mainAction.Action.GetType())
	}

	// Execute the compound action
	err = compound.Execute(gs, "Engineers")
	if err != nil {
		t.Fatalf("compound.Execute() error = %v", err)
	}

	// Verify conversion happened: 6 power -> 4 power
	if player.Resources.Power.Bowl3 != 4 {
		t.Errorf("expected 4 power in bowl3 after conversion, got %d", player.Resources.Power.Bowl3)
	}
	// Note: Coins were spent on the upgrade (6C + 4W), so checking final coin count isn't useful

	// Verify upgrade happened: building is now Stronghold
	if gs.Map.GetHex(hex).Building.Type != models.BuildingStronghold {
		t.Errorf("expected Stronghold, got %v", gs.Map.GetHex(hex).Building.Type)
	}

	t.Logf("✓ Convert+Upgrade: conversion executed first, then upgrade validated costs")
}

// TestConvertAndUpgrade_WithFavorTile tests convert+upgrade with favor tile selection
func TestConvertAndUpgrade_WithFavorTile(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("Cultists", &factions.Cultists{})

	player := gs.GetPlayer("Cultists")
	player.Resources.Coins = 4     // Start with 4 coins (need 5 total, will get 1 from converting 1W)
	player.Resources.Workers = 3   // Need 3 workers (1 for conversion, 2 for Temple upgrade)
	player.Resources.Priests = 0   // Temple doesn't need priests
	player.Resources.Power.Bowl3 = 0

	// Place a trading house at F3 to upgrade to temple
	hex, _ := ConvertLogCoordToAxial("F3")
	gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(hex).Building = &models.Building{
		Type:       models.BuildingTradingHouse,
		Faction:    models.FactionCultists,
		PlayerID:   "Cultists",
		PowerValue: 2,
	}

	// Parse: "convert 1W to 1C. upgrade F3 to TE. +FAV9"
	entry := &LogEntry{
		Faction: models.FactionCultists,
		Action:  "convert 1W to 1C. upgrade F3 to TE. +FAV9",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have 3 components: conversion + upgrade + favor tile
	if len(compound.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(compound.Components))
	}

	// Verify component types
	if _, ok := compound.Components[0].(*ConversionComponent); !ok {
		t.Errorf("component 0: expected ConversionComponent, got %T", compound.Components[0])
	}
	if _, ok := compound.Components[1].(*MainActionComponent); !ok {
		t.Errorf("component 1: expected MainActionComponent, got %T", compound.Components[1])
	}
	if _, ok := compound.Components[2].(*AuxiliaryComponent); !ok {
		t.Errorf("component 2: expected AuxiliaryComponent, got %T", compound.Components[2])
	}

	// Execute the compound action
	err = compound.Execute(gs, "Cultists")
	if err != nil {
		t.Fatalf("compound.Execute() error = %v", err)
	}

	// Verify conversions and upgrade happened
	// Started with 3W, converted 1W to 1C, then spent 2W on Temple = 0W remaining
	if player.Resources.Workers != 0 {
		t.Errorf("expected 0 workers after conversion and upgrade, got %d", player.Resources.Workers)
	}

	// Verify upgrade happened
	if gs.Map.GetHex(hex).Building.Type != models.BuildingTemple {
		t.Errorf("expected Temple, got %v", gs.Map.GetHex(hex).Building.Type)
	}

	t.Logf("✓ Convert+Upgrade+FavorTile: all components executed in order")
}

// TestConvertAndUpgrade_MultipleConversions tests conversions before AND after upgrade
func TestConvertAndUpgrade_MultipleConversions(t *testing.T) {
	gs := game.NewGameState()
	gs.AddPlayer("Darklings", &factions.Darklings{})

	player := gs.GetPlayer("Darklings")
	player.Resources.Coins = 5       // Start with 5 (need 6 total, will get 1 from converting 1PW)
	player.Resources.Workers = 2     // TradingHouse needs 2 workers
	player.Resources.Priests = 1     // For conversion after upgrade (1P -> 1W)
	player.Resources.Power.Bowl3 = 1 // For first conversion (1PW -> 1C)

	// Place a dwelling at I3 to upgrade to trading house (TP)
	hex, _ := ConvertLogCoordToAxial("I3")
	gs.Map.GetHex(hex).Terrain = player.Faction.GetHomeTerrain()
	gs.Map.GetHex(hex).Building = &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    models.FactionDarklings,
		PlayerID:   "Darklings",
		PowerValue: 1,
	}

	// Parse: "convert 1PW to 1C. upgrade I3 to TP. convert 1P to 1W"
	// (Removed +TW3 since town tiles require a formed town with 4+ buildings)
	entry := &LogEntry{
		Faction: models.FactionDarklings,
		Action:  "convert 1PW to 1C. upgrade I3 to TP. convert 1P to 1W",
	}

	compound, err := ParseCompoundAction(entry.Action, entry, gs)
	if err != nil {
		t.Fatalf("ParseCompoundAction() error = %v", err)
	}

	// Should have 3 components: conv + upgrade + conv
	if len(compound.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(compound.Components))
	}

	// Verify component order
	if _, ok := compound.Components[0].(*ConversionComponent); !ok {
		t.Errorf("component 0: expected ConversionComponent, got %T", compound.Components[0])
	}
	if _, ok := compound.Components[1].(*MainActionComponent); !ok {
		t.Errorf("component 1: expected MainActionComponent, got %T", compound.Components[1])
	}
	if _, ok := compound.Components[2].(*ConversionComponent); !ok {
		t.Errorf("component 2: expected ConversionComponent, got %T", compound.Components[2])
	}

	// Execute the compound action
	err = compound.Execute(gs, "Darklings")
	if err != nil {
		t.Fatalf("compound.Execute() error = %v", err)
	}

	// Verify first conversion happened (1 power -> 1 coin)
	if player.Resources.Power.Bowl3 != 0 {
		t.Errorf("expected 0 power in bowl3 after conversion, got %d", player.Resources.Power.Bowl3)
	}

	// Verify upgrade happened (Dwelling -> TradingHouse)
	// Should have consumed 2 workers
	if gs.Map.GetHex(hex).Building.Type != models.BuildingTradingHouse {
		t.Errorf("expected TradingHouse, got %v", gs.Map.GetHex(hex).Building.Type)
	}

	// Verify second conversion happened (1 priest -> 1 worker)
	// Started with 1 priest, converted to 1 worker
	if player.Resources.Priests != 0 {
		t.Errorf("expected 0 priests after conversion, got %d", player.Resources.Priests)
	}
	if player.Resources.Workers != 1 {
		t.Errorf("expected 1 worker after second conversion, got %d", player.Resources.Workers)
	}

	t.Logf("✓ Multiple conversions: before and after upgrade work correctly")
}
