package board

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestAvailableMaps_ContainsRegisteredMaps(t *testing.T) {
	infos := AvailableMaps()
	if len(infos) < 6 {
		t.Fatalf("expected at least six maps, got %d", len(infos))
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
	if !found[MapFireAndIce] {
		t.Fatalf("fire & ice map missing from catalog")
	}
	if !found[MapFjords] {
		t.Fatalf("fjords map missing from catalog")
	}
	if !found[MapLakes] {
		t.Fatalf("lakes map missing from catalog")
	}
	if !found[MapRevisedBase] {
		t.Fatalf("revised base map missing from catalog")
	}
}

func TestNormalizeMapID_FireAndIceAliases(t *testing.T) {
	cases := []string{
		"fire-and-ice",
		"fire_and_ice",
		"Fire and Ice",
		"fire&ice",
	}

	for _, raw := range cases {
		if got := NormalizeMapID(raw); got != MapFireAndIce {
			t.Fatalf("NormalizeMapID(%q): got %q, want %q", raw, got, MapFireAndIce)
		}
	}
}

func TestRevisedBaseLayout_KeyHexes(t *testing.T) {
	layout, err := LayoutForMap(MapRevisedBase)
	if err != nil {
		t.Fatalf("load revised base: %v", err)
	}

	checks := map[Hex]models.TerrainType{
		NewHex(4, 0):  models.TerrainPlains,
		NewHex(9, 0):  models.TerrainLake,
		NewHex(10, 0): models.TerrainForest,
		NewHex(3, 1):  models.TerrainDesert,
		NewHex(8, 1):  models.TerrainForest,
		NewHex(7, 2):  models.TerrainSwamp,
		NewHex(9, 2):  models.TerrainWasteland,
		NewHex(9, 3):  models.TerrainMountain,
		NewHex(4, 4):  models.TerrainForest,
		NewHex(3, 5):  models.TerrainMountain,
		NewHex(-4, 8): models.TerrainLake,
		NewHex(8, 8):  models.TerrainMountain,
		NewHex(1, 1):  models.TerrainRiver,
		NewHex(6, 4):  models.TerrainRiver,
	}

	for hex, want := range checks {
		if got := layout[hex]; got != want {
			t.Fatalf("hex %v: got %v, want %v", hex, got, want)
		}
	}
}

func TestLakesLayout_TopRows(t *testing.T) {
	layout, err := LayoutForMap(MapLakes)
	if err != nil {
		t.Fatalf("load lakes: %v", err)
	}

	topRow := []models.TerrainType{
		models.TerrainMountain,
		models.TerrainLake,
		models.TerrainWasteland,
		models.TerrainPlains,
		models.TerrainDesert,
		models.TerrainLake,
		models.TerrainDesert,
		models.TerrainWasteland,
		models.TerrainRiver,
		models.TerrainRiver,
		models.TerrainForest,
		models.TerrainLake,
	}
	for i, want := range topRow {
		got := layout[NewHex(i, 0)]
		if got != want {
			t.Fatalf("lakes row A slot %d: got %v, want %v", i, got, want)
		}
	}

	secondRow := []models.TerrainType{
		models.TerrainDesert,
		models.TerrainSwamp,
		models.TerrainForest,
		models.TerrainRiver,
		models.TerrainRiver,
		models.TerrainSwamp,
		models.TerrainPlains,
		models.TerrainRiver,
		models.TerrainForest,
		models.TerrainMountain,
		models.TerrainRiver,
		models.TerrainPlains,
		models.TerrainSwamp,
	}
	for i, want := range secondRow {
		got := layout[NewHex(i-1, 1)]
		if got != want {
			t.Fatalf("lakes row B slot %d: got %v, want %v", i, got, want)
		}
	}
}

func TestLakesLayout_RowOffsetsUseLeftStagger(t *testing.T) {
	layout, err := LayoutForMap(MapLakes)
	if err != nil {
		t.Fatalf("load lakes: %v", err)
	}

	if got := layout[NewHex(0, 0)]; got != models.TerrainMountain {
		t.Fatalf("expected A1 at (0,0), got %v", got)
	}
	if got := layout[NewHex(-1, 1)]; got != models.TerrainDesert {
		t.Fatalf("expected B1 at (-1,1), got %v", got)
	}
	if got := layout[NewHex(-1, 2)]; got != models.TerrainPlains {
		t.Fatalf("expected C1 at (-1,2), got %v", got)
	}
	if got := layout[NewHex(-2, 3)]; got != models.TerrainLake {
		t.Fatalf("expected D1 at (-2,3), got %v", got)
	}
}

func TestLakesDisplayCoordinates_RespectLeftStagger(t *testing.T) {
	hex, ok := HexForDisplayCoordinate(MapLakes, "B1")
	if !ok {
		t.Fatalf("expected to resolve Lakes B1")
	}
	if hex != NewHex(-1, 1) {
		t.Fatalf("Lakes B1: got %v, want %v", hex, NewHex(-1, 1))
	}

	display, ok := DisplayCoordinateForHex(MapLakes, NewHex(-1, 1))
	if !ok {
		t.Fatalf("expected display coordinate for Lakes (-1,1)")
	}
	if display != "B1" {
		t.Fatalf("Lakes (-1,1): got %q, want %q", display, "B1")
	}

	display, ok = DisplayCoordinateForHex(MapLakes, NewHex(0, 1))
	if !ok {
		t.Fatalf("expected display coordinate for Lakes (0,1)")
	}
	if display != "B2" {
		t.Fatalf("Lakes (0,1): got %q, want %q", display, "B2")
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

func TestFireAndIceLayout_RowCountsAndKeyRows(t *testing.T) {
	layout, err := LayoutForMap(MapFireAndIce)
	if err != nil {
		t.Fatalf("load fire & ice: %v", err)
	}

	rowSpecs := []struct {
		r          int
		startQ     int
		slotCount  int
		riverCount int
	}{
		{r: 0, startQ: 0, slotCount: 12, riverCount: 2},
		{r: 1, startQ: -1, slotCount: 13, riverCount: 4},
		{r: 2, startQ: -1, slotCount: 12, riverCount: 7},
		{r: 3, startQ: -2, slotCount: 13, riverCount: 2},
		{r: 4, startQ: -2, slotCount: 12, riverCount: 5},
		{r: 5, startQ: -3, slotCount: 13, riverCount: 6},
		{r: 6, startQ: -3, slotCount: 12, riverCount: 3},
		{r: 7, startQ: -4, slotCount: 13, riverCount: 3},
		{r: 8, startQ: -4, slotCount: 12, riverCount: 3},
	}

	for _, spec := range rowSpecs {
		rowRiverCount := 0
		for i := 0; i < spec.slotCount; i++ {
			hex := NewHex(spec.startQ+i, spec.r)
			terrain, ok := layout[hex]
			if !ok {
				t.Fatalf("expected fire & ice hex at %v", hex)
			}
			if terrain == models.TerrainRiver {
				rowRiverCount++
			}
		}
		if rowRiverCount != spec.riverCount {
			t.Fatalf("row %d river count: got %d, want %d", spec.r, rowRiverCount, spec.riverCount)
		}
	}

	topRow := []models.TerrainType{
		models.TerrainPlains,
		models.TerrainRiver,
		models.TerrainPlains,
		models.TerrainSwamp,
		models.TerrainDesert,
		models.TerrainRiver,
		models.TerrainMountain,
		models.TerrainForest,
		models.TerrainWasteland,
		models.TerrainLake,
		models.TerrainDesert,
		models.TerrainLake,
	}
	for i, want := range topRow {
		got := layout[NewHex(i, 0)]
		if got != want {
			t.Fatalf("fire & ice row A slot %d: got %v, want %v", i, got, want)
		}
	}

	fourthRow := []models.TerrainType{
		models.TerrainDesert,
		models.TerrainMountain,
		models.TerrainForest,
		models.TerrainDesert,
		models.TerrainSwamp,
		models.TerrainRiver,
		models.TerrainLake,
		models.TerrainWasteland,
		models.TerrainPlains,
		models.TerrainRiver,
		models.TerrainForest,
		models.TerrainLake,
		models.TerrainForest,
	}
	for i, want := range fourthRow {
		got := layout[NewHex(i-2, 3)]
		if got != want {
			t.Fatalf("fire & ice row D slot %d: got %v, want %v", i, got, want)
		}
	}

	bottomRow := []models.TerrainType{
		models.TerrainMountain,
		models.TerrainForest,
		models.TerrainRiver,
		models.TerrainWasteland,
		models.TerrainDesert,
		models.TerrainSwamp,
		models.TerrainDesert,
		models.TerrainRiver,
		models.TerrainLake,
		models.TerrainPlains,
		models.TerrainRiver,
		models.TerrainPlains,
	}
	for i, want := range bottomRow {
		got := layout[NewHex(i-4, 8)]
		if got != want {
			t.Fatalf("fire & ice row I slot %d: got %v, want %v", i, got, want)
		}
	}
}

func TestFjordsLayout_TopAndFourthRows(t *testing.T) {
	layout, err := LayoutForMap(MapFjords)
	if err != nil {
		t.Fatalf("load fjords: %v", err)
	}

	topRow := []models.TerrainType{
		models.TerrainForest,
		models.TerrainSwamp,
		models.TerrainRiver,
		models.TerrainPlains,
		models.TerrainDesert,
		models.TerrainMountain,
		models.TerrainSwamp,
		models.TerrainMountain,
		models.TerrainDesert,
		models.TerrainWasteland,
		models.TerrainSwamp,
		models.TerrainLake,
		models.TerrainDesert,
	}
	for i, want := range topRow {
		got := layout[NewHex(i, 0)]
		if got != want {
			t.Fatalf("fjords row A slot %d: got %v, want %v", i, got, want)
		}
	}

	fourthRow := []models.TerrainType{
		models.TerrainRiver,
		models.TerrainRiver,
		models.TerrainRiver,
		models.TerrainMountain,
		models.TerrainRiver,
		models.TerrainRiver,
		models.TerrainForest,
		models.TerrainWasteland,
		models.TerrainLake,
		models.TerrainForest,
		models.TerrainWasteland,
		models.TerrainRiver,
	}
	for i, want := range fourthRow {
		got := layout[NewHex(i-1, 3)]
		if got != want {
			t.Fatalf("fjords row D slot %d: got %v, want %v", i, got, want)
		}
	}
}
