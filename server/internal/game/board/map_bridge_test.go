package board

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

// helper to make a minimal map
func makeMap(cells map[Hex]models.TerrainType) *TerraMysticaMap {
	m := &TerraMysticaMap{
		Hexes:      make(map[Hex]*MapHex),
		Bridges:    make(map[BridgeKey]bool),
		RiverHexes: make(map[Hex]bool),
	}
	for h, t := range cells {
		m.Hexes[h] = &MapHex{Coord: h, Terrain: t}
		if t == models.TerrainRiver {
			m.RiverHexes[h] = true
		}
	}
	return m
}

func TestBridgeGeometry_Valid(t *testing.T) {
	// Use base orientation from problem statement
	// h1 -> h2 delta (1,-2) with midpoints (0,-1) and (1,-1)
	h1 := NewHex(0, 0)
	midA := NewHex(0, -1)
	midB := NewHex(1, -1)
	h2 := NewHex(1, -2)
	cells := map[Hex]models.TerrainType{
		h1:   models.TerrainPlains,
		midA: models.TerrainRiver,
		midB: models.TerrainRiver,
		h2:   models.TerrainForest,
	}
	m := makeMap(cells)

	if err := m.BuildBridge(h1, h2); err != nil {
		t.Fatalf("expected valid bridge, got error: %v", err)
	}
	if !m.IsDirectlyAdjacent(h1, h2) {
		t.Fatalf("expected directly adjacent via bridge")
	}
}

func TestBridgeGeometry_InvalidMidpoint(t *testing.T) {
	h1 := NewHex(0, 0)
	midA := NewHex(0, -1)
	midB := NewHex(1, -1)
	h2 := NewHex(1, -2)
	cells := map[Hex]models.TerrainType{
		h1:   models.TerrainPlains,
		midA: models.TerrainRiver,
		midB: models.TerrainPlains, // not river
		h2:   models.TerrainForest,
	}
	m := makeMap(cells)

	if err := m.BuildBridge(h1, h2); err == nil {
		t.Fatalf("expected error due to non-river midpoint")
	}
}

func TestBridgeGeometry_EndpointRiver(t *testing.T) {
	h1 := NewHex(0, 0)
	midA := NewHex(0, -1)
	midB := NewHex(1, -1)
	h2 := NewHex(1, -2)
	cells := map[Hex]models.TerrainType{
		h1:   models.TerrainRiver, // endpoint river -> invalid
		midA: models.TerrainRiver,
		midB: models.TerrainRiver,
		h2:   models.TerrainForest,
	}
	m := makeMap(cells)

	if err := m.BuildBridge(h1, h2); err == nil {
		t.Fatalf("expected error due to river endpoint")
	}
}

func TestBridgeGeometry_WrongDelta(t *testing.T) {
	// Use delta (2,-2) which is not in allowed set
	h1 := NewHex(0, 0)
	midA := NewHex(1, -1)
	midB := NewHex(2, -1)
	h2 := NewHex(2, -2)
	cells := map[Hex]models.TerrainType{
		h1:   models.TerrainPlains,
		midA: models.TerrainRiver,
		midB: models.TerrainRiver,
		h2:   models.TerrainForest,
	}
	m := makeMap(cells)
	if err := m.BuildBridge(h1, h2); err == nil {
		t.Fatalf("expected error for invalid bridge delta")
	}
}

func TestShipping_RiverOnlyBFS_ReachesAtExactShipping(t *testing.T) {
	// Land h1 touches rv1; chain rv1->rv2->rv3; h2 touches rv3
	h1 := NewHex(0, 0)
	rv1 := NewHex(1, 0)
	rv2 := NewHex(2, 0)
	rv3 := NewHex(3, 0)
	h2 := NewHex(4, 0)
	cells := map[Hex]models.TerrainType{
		h1: models.TerrainPlains,
		rv1: models.TerrainRiver,
		rv2: models.TerrainRiver,
		rv3: models.TerrainRiver,
		h2: models.TerrainForest,
	}
	m := makeMap(cells)

	if ok := m.IsIndirectlyAdjacent(h1, h2, 3); !ok {
		t.Fatalf("expected reachable with shipping=3")
	}
	if ok := m.IsIndirectlyAdjacent(h1, h2, 2); ok {
		t.Fatalf("expected NOT reachable with shipping=2")
	}
}

func TestShipping_EndpointsMustBeLand(t *testing.T) {
	h1 := NewHex(0, 0)
	rv := NewHex(1, 0)
	h2 := NewHex(2, 0)
	cells := map[Hex]models.TerrainType{
		h1: models.TerrainRiver, // invalid endpoint
		rv: models.TerrainRiver,
		h2: models.TerrainPlains,
	}
	m := makeMap(cells)
	if ok := m.IsIndirectlyAdjacent(h1, h2, 1); ok {
		t.Fatalf("expected false when endpoint is river")
	}
}

func TestShipping_DirectAdjacencyExcluded(t *testing.T) {
	h1 := NewHex(0, 0)
	h2 := NewHex(1, 0)
	cells := map[Hex]models.TerrainType{
		h1: models.TerrainPlains,
		h2: models.TerrainForest,
	}
	m := makeMap(cells)
	if ok := m.IsIndirectlyAdjacent(h1, h2, 3); ok {
		t.Fatalf("expected false for directly adjacent hexes")
	}
}
