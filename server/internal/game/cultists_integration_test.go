package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// Test Cultists gain power when all opponents decline leech
func TestCultistsIntegration_PowerWhenAllDecline(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	halflingsFaction := factions.NewHalflings()
	
	gs.AddPlayer("cultists", cultistsFaction)
	gs.AddPlayer("halflings", halflingsFaction)
	
	cultistsPlayer := gs.GetPlayer("cultists")
	halflingsPlayer := gs.GetPlayer("halflings")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	halflingsPlayer.Resources.Power.Bowl1 = 10
	
	// Place a Halflings dwelling adjacent to where Cultists will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = halflingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    halflingsFaction.GetType(),
		PlayerID:   "halflings",
		PowerValue: 1,
	})
	
	// Cultists build a dwelling
	cultistsHex := NewHex(0, 0)
	gs.Map.GetHex(cultistsHex).Terrain = cultistsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(cultistsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    cultistsFaction.GetType(),
		PlayerID:   "cultists",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := cultistsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(cultistsHex, "cultists")
	
	// Verify leech offer was created for Halflings
	offers := gs.GetPendingLeechOffers("halflings")
	if len(offers) != 1 {
		t.Fatalf("expected 1 leech offer, got %d", len(offers))
	}
	
	// Halflings decline the offer
	declineAction := NewDeclinePowerLeechAction("halflings", 0)
	err := declineAction.Execute(gs)
	if err != nil {
		t.Fatalf("decline failed: %v", err)
	}
	
	// Cultists should gain 1 power (all opponents declined)
	powerGained := cultistsPlayer.Resources.Power.Bowl1 - initialPower
	expectedPower := 1
	if powerGained != expectedPower {
		t.Errorf("expected %d power gained, got %d", expectedPower, powerGained)
	}
	
	// Verify the pending bonus was resolved
	if gs.PendingCultistsLeech["cultists"] != nil {
		t.Error("Cultists leech bonus should be resolved")
	}
}

// Test Cultists get cult advance when at least one opponent accepts
func TestCultistsIntegration_CultAdvanceWhenOneAccepts(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	halflingsFaction := factions.NewHalflings()
	
	gs.AddPlayer("cultists", cultistsFaction)
	gs.AddPlayer("halflings", halflingsFaction)
	
	cultistsPlayer := gs.GetPlayer("cultists")
	halflingsPlayer := gs.GetPlayer("halflings")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	halflingsPlayer.Resources.Power.Bowl1 = 10
	halflingsPlayer.VictoryPoints = 10
	
	// Place a Halflings dwelling adjacent to where Cultists will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = halflingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    halflingsFaction.GetType(),
		PlayerID:   "halflings",
		PowerValue: 1,
	})
	
	// Cultists build a dwelling
	cultistsHex := NewHex(0, 0)
	gs.Map.GetHex(cultistsHex).Terrain = cultistsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(cultistsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    cultistsFaction.GetType(),
		PlayerID:   "cultists",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := cultistsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(cultistsHex, "cultists")
	
	// Verify leech offer was created
	offers := gs.GetPendingLeechOffers("halflings")
	if len(offers) != 1 {
		t.Fatalf("expected 1 leech offer, got %d", len(offers))
	}
	
	// Halflings accept the offer
	acceptAction := NewAcceptPowerLeechAction("halflings", 0)
	err := acceptAction.Execute(gs)
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}
	
	// Cultists should NOT gain power (opponent accepted)
	powerGained := cultistsPlayer.Resources.Power.Bowl1 - initialPower
	if powerGained != 0 {
		t.Errorf("expected 0 power gained (opponent accepted), got %d", powerGained)
	}
	
	// Verify the pending bonus was resolved
	if gs.PendingCultistsLeech["cultists"] != nil {
		t.Error("Cultists leech bonus should be resolved")
	}
	
	// TODO: When cult track selection is implemented, verify cult advance happened
}

// Test Cultists with multiple opponents - one accepts, one declines
func TestCultistsIntegration_MultipleOpponents_MixedResponses(t *testing.T) {
	gs := NewGameState()
	cultistsFaction := factions.NewCultists()
	halflingsFaction := factions.NewHalflings()
	nomadsFaction := factions.NewNomads()
	
	gs.AddPlayer("cultists", cultistsFaction)
	gs.AddPlayer("halflings", halflingsFaction)
	gs.AddPlayer("nomads", nomadsFaction)
	
	cultistsPlayer := gs.GetPlayer("cultists")
	halflingsPlayer := gs.GetPlayer("halflings")
	nomadsPlayer := gs.GetPlayer("nomads")
	
	// Set up power
	cultistsPlayer.Resources.Power.Bowl1 = 10
	halflingsPlayer.Resources.Power.Bowl1 = 10
	halflingsPlayer.VictoryPoints = 10
	nomadsPlayer.Resources.Power.Bowl1 = 10
	nomadsPlayer.VictoryPoints = 10
	
	// Place Halflings dwelling adjacent to Cultists
	hex1 := NewHex(1, 0)
	gs.Map.GetHex(hex1).Terrain = halflingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex1, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    halflingsFaction.GetType(),
		PlayerID:   "halflings",
		PowerValue: 1,
	})
	
	// Place Nomads dwelling adjacent to Cultists
	hex2 := NewHex(0, 1)
	gs.Map.GetHex(hex2).Terrain = nomadsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(hex2, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    nomadsFaction.GetType(),
		PlayerID:   "nomads",
		PowerValue: 1,
	})
	
	// Cultists build a dwelling
	cultistsHex := NewHex(0, 0)
	gs.Map.GetHex(cultistsHex).Terrain = cultistsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(cultistsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    cultistsFaction.GetType(),
		PlayerID:   "cultists",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := cultistsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(cultistsHex, "cultists")
	
	// Verify leech offers were created
	halflingsOffers := gs.GetPendingLeechOffers("halflings")
	nomadsOffers := gs.GetPendingLeechOffers("nomads")
	if len(halflingsOffers) != 1 || len(nomadsOffers) != 1 {
		t.Fatalf("expected 1 offer each, got %d and %d", len(halflingsOffers), len(nomadsOffers))
	}
	
	// Halflings accept
	acceptAction := NewAcceptPowerLeechAction("halflings", 0)
	err := acceptAction.Execute(gs)
	if err != nil {
		t.Fatalf("halflings accept failed: %v", err)
	}
	
	// Nomads decline
	declineAction := NewDeclinePowerLeechAction("nomads", 0)
	err = declineAction.Execute(gs)
	if err != nil {
		t.Fatalf("nomads decline failed: %v", err)
	}
	
	// Cultists should NOT gain power (at least one opponent accepted)
	powerGained := cultistsPlayer.Resources.Power.Bowl1 - initialPower
	if powerGained != 0 {
		t.Errorf("expected 0 power gained (one accepted), got %d", powerGained)
	}
	
	// Verify the pending bonus was resolved
	if gs.PendingCultistsLeech["cultists"] != nil {
		t.Error("Cultists leech bonus should be resolved")
	}
}

// Test non-Cultists don't get the bonus
func TestCultistsIntegration_OnlyCultistsGetBonus(t *testing.T) {
	gs := NewGameState()
	halflingsFaction := factions.NewHalflings()
	nomadsFaction := factions.NewNomads()
	
	gs.AddPlayer("halflings", halflingsFaction)
	gs.AddPlayer("nomads", nomadsFaction)
	
	halflingsPlayer := gs.GetPlayer("halflings")
	nomadsPlayer := gs.GetPlayer("nomads")
	
	// Set up power
	halflingsPlayer.Resources.Power.Bowl1 = 10
	nomadsPlayer.Resources.Power.Bowl1 = 10
	
	// Place a Nomads dwelling adjacent to where Halflings will build
	adjacentHex := NewHex(1, 0)
	gs.Map.GetHex(adjacentHex).Terrain = nomadsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(adjacentHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    nomadsFaction.GetType(),
		PlayerID:   "nomads",
		PowerValue: 1,
	})
	
	// Halflings build a dwelling
	halflingsHex := NewHex(0, 0)
	gs.Map.GetHex(halflingsHex).Terrain = halflingsFaction.GetHomeTerrain()
	gs.Map.PlaceBuilding(halflingsHex, &models.Building{
		Type:       models.BuildingDwelling,
		Faction:    halflingsFaction.GetType(),
		PlayerID:   "halflings",
		PowerValue: 1,
	})
	
	// Trigger power leech
	initialPower := halflingsPlayer.Resources.Power.Bowl1
	gs.TriggerPowerLeech(halflingsHex, "halflings")
	
	// Nomads decline
	declineAction := NewDeclinePowerLeechAction("nomads", 0)
	err := declineAction.Execute(gs)
	if err != nil {
		t.Fatalf("decline failed: %v", err)
	}
	
	// Halflings should NOT gain power (not Cultists)
	powerGained := halflingsPlayer.Resources.Power.Bowl1 - initialPower
	if powerGained != 0 {
		t.Errorf("expected 0 power gained (not Cultists), got %d", powerGained)
	}
	
	// Verify no pending bonus was created
	if gs.PendingCultistsLeech["halflings"] != nil {
		t.Error("non-Cultists should not have pending leech bonus")
	}
}
