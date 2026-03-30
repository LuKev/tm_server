package board

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestAvailableMaps_ContainsBaseAndArchipelago(t *testing.T) {
	infos := AvailableMaps()
	if len(infos) < 2 {
		t.Fatalf("expected at least two maps, got %d", len(infos))
	}

	found := map[MapID]bool{}
	for _, info := range infos {
		found[info.ID] = true
	}

	if !found[MapBase] {
		t.Fatalf("base map missing from catalog")
	}
	if !found[MapArchipelago] {
		t.Fatalf("archipelago map missing from catalog")
	}
}

func TestArchipelagoLayout_TopAndFourthRows(t *testing.T) {
	layout, err := LayoutForMap(MapArchipelago)
	if err != nil {
		t.Fatalf("load archipelago: %v", err)
	}

	topRow := []models.TerrainType{
		models.TerrainSwamp,
		models.TerrainLake,
		models.TerrainWasteland,
		models.TerrainForest,
		models.TerrainLake,
		models.TerrainPlains,
		models.TerrainRiver,
		models.TerrainWasteland,
		models.TerrainPlains,
		models.TerrainSwamp,
		models.TerrainRiver,
		models.TerrainLake,
		models.TerrainForest,
	}
	for i, want := range topRow {
		got := layout[NewHex(i, 0)]
		if got != want {
			t.Fatalf("row A slot %d: got %v, want %v", i, got, want)
		}
	}

	fourthRow := []models.TerrainType{
		models.TerrainRiver,
		models.TerrainMountain,
		models.TerrainWasteland,
		models.TerrainRiver,
		models.TerrainRiver,
		models.TerrainRiver,
		models.TerrainPlains,
		models.TerrainDesert,
		models.TerrainRiver,
		models.TerrainWasteland,
		models.TerrainDesert,
		models.TerrainRiver,
	}
	for i, want := range fourthRow {
		got := layout[NewHex(i-1, 3)]
		if got != want {
			t.Fatalf("row D slot %d: got %v, want %v", i, got, want)
		}
	}
}
