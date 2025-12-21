package board

import (
	"testing"
)

// These tests use the BaseGameTerrainLayout via NewTerraMysticaMap()
// and assert shipping (indirect adjacency) over rivers only.
//
// Base map layout reference (rows 0-2):
// Row 0: Plains, Mountain, Forest, Lake, Desert, Wasteland, Plains, Swamp, Wasteland, Forest, Lake, Wasteland, Swamp
// Row 1: Desert, River, River, Plains, Swamp, River, River, Desert, Swamp, River, River, Desert
// Row 2: River, River, Swamp, River, Mountain, River, Forest, River, Forest, River, Mountain, River, River

func TestIndirectAdjacency_Shipping1_SingleRiverNeighbor(t *testing.T) {
	m := NewTerraMysticaMap()
	// b1 is Desert
	// b2 is Plains
	// c1 is Swamp
	// This is the north western corner of the map
	b1 := NewHex(0, 1)
	b2 := NewHex(3, 1)
	c1 := NewHex(1, 2)

	if ok := m.IsIndirectlyAdjacent(b1, b2, 2); !ok {
		t.Fatalf("expected shipping=2 to reach from %v to %v", b1, c1)
	}
	if ok := m.IsIndirectlyAdjacent(b1, b2, 1); ok {
		t.Fatalf("expected shipping=1 to NOT reach from %v to %v", b1, c1)
	}
	if ok := m.IsIndirectlyAdjacent(b1, c1, 1); !ok {
		t.Fatalf("expected shipping=1 to reach from %v to %v", b1, c1)
	}
	if ok := m.IsIndirectlyAdjacent(b2, c1, 1); !ok {
		t.Fatalf("expected shipping=1 to reach from %v to %v", b2, c1)
	}

}

func TestIndirectAdjacency_Shipping2_RiverChain(t *testing.T) {
	m := NewTerraMysticaMap()
	// Use northern black swamp 2 ship path
	a8 := NewHex(7, 0)
	b3 := NewHex(4, 1)
	c1 := NewHex(1, 2)
	e5 := NewHex(2, 4)

	pairs := [][2]Hex{
		{c1, e5},
		{e5, b3},
		{b3, a8},
	}
	for _, p := range pairs {
		if ok := m.IsIndirectlyAdjacent(p[0], p[1], 2); !ok {
			t.Fatalf("expected shipping=2 to reach from %v to %v via river chain", p[0], p[1])
		}
		// With shipping 1 should not reach
		if ok := m.IsIndirectlyAdjacent(p[0], p[1], 1); ok {
			t.Fatalf("expected shipping=1 to NOT reach from %v to %v", p[0], p[1])
		}
	}
}

func TestIndirectAdjacency_EndpointsMustBeLand_BaseMap(t *testing.T) {
	m := NewTerraMysticaMap()
	// (0,1) is Desert (land), (1,1) is River
	h1 := NewHex(0, 1)
	hRiver := NewHex(1, 1) // river
	if ok := m.IsIndirectlyAdjacent(h1, hRiver, 1); ok {
		t.Fatalf("expected endpoints must be land; got true")
	}
}

func TestIndirectAdjacency_DirectAdjacencyExcluded_BaseMap(t *testing.T) {
	m := NewTerraMysticaMap()
	// Two land hexes sharing edge: (0,0) Plains and (1,0) Mountain
	h1 := NewHex(0, 0)
	h2 := NewHex(1, 0)
	if ok := m.IsIndirectlyAdjacent(h1, h2, 3); ok {
		t.Fatalf("expected false for directly adjacent land hexes")
	}
}

func TestIndirectAdjacency_Shipping3_LongerChain(t *testing.T) {
	m := NewTerraMysticaMap()
	// Test western mermaid 3 ship path
	a4 := NewHex(3, 0)
	d2 := NewHex(0, 3)
	e4 := NewHex(1, 4)
	h4 := NewHex(3, 7)
	i4 := NewHex(-1, 8)

	pairs := [][2]Hex{
		{a4, d2},
		{a4, e4},
		{e4, i4},
		{i4, h4},
	}
	for _, p := range pairs {
		if ok := m.IsIndirectlyAdjacent(p[0], p[1], 3); !ok {
			t.Fatalf("expected shipping=3 to reach from %v to %v via river chain", p[0], p[1])
		}
		// With shipping 2 should not reach
		if ok := m.IsIndirectlyAdjacent(p[0], p[1], 2); ok {
			t.Fatalf("expected shipping=2 to NOT reach from %v to %v", p[0], p[1])
		}
	}
}

func TestIndirectAdjacency_Shipping3_VerifyReachability(t *testing.T) {
	m := NewTerraMysticaMap()
	// Test various shipping=3 scenarios to verify the algorithm works correctly
	h1 := NewHex(0, 1) // Desert

	// Test that we can reach various land hexes with shipping=3
	// Just verify the algorithm handles shipping=3 correctly
	reachable := false
	for q := 0; q < 13; q++ {
		for r := 0; r < 9; r++ {
			h2 := NewHex(q, r)
			if m.IsValidHex(h2) && !m.IsRiver(h2) && !h1.Equals(h2) {
				if m.IsIndirectlyAdjacent(h1, h2, 3) {
					reachable = true
					break
				}
			}
		}
		if reachable {
			break
		}
	}
	if !reachable {
		t.Fatalf("expected shipping=3 to reach at least some hexes from %v", h1)
	}
}

func TestIndirectAdjacency_Shipping4_VerifyReachability(t *testing.T) {
	m := NewTerraMysticaMap()
	// Test that shipping=4 works correctly (may reach same or more hexes than shipping=3)
	h1 := NewHex(0, 1) // Desert

	count3 := 0
	count4 := 0
	for q := 0; q < 13; q++ {
		for r := 0; r < 9; r++ {
			h2 := NewHex(q, r)
			if m.IsValidHex(h2) && !m.IsRiver(h2) && !h1.Equals(h2) {
				if m.IsIndirectlyAdjacent(h1, h2, 3) {
					count3++
				}
				if m.IsIndirectlyAdjacent(h1, h2, 4) {
					count4++
				}
			}
		}
	}
	// Shipping=4 should reach at least as many as shipping=3
	if count4 < count3 {
		t.Fatalf("expected shipping=4 to reach at least as many hexes as shipping=3, got %d vs %d", count4, count3)
	}
	// Verify shipping=4 actually works
	if count4 == 0 {
		t.Fatalf("expected shipping=4 to reach at least some hexes")
	}
}

func TestIndirectAdjacency_Shipping5_VerifyReachability(t *testing.T) {
	m := NewTerraMysticaMap()
	// Test that shipping=5 works correctly
	h1 := NewHex(0, 1) // Desert

	count4 := 0
	count5 := 0
	for q := 0; q < 13; q++ {
		for r := 0; r < 9; r++ {
			h2 := NewHex(q, r)
			if m.IsValidHex(h2) && !m.IsRiver(h2) && !h1.Equals(h2) {
				if m.IsIndirectlyAdjacent(h1, h2, 4) {
					count4++
				}
				if m.IsIndirectlyAdjacent(h1, h2, 5) {
					count5++
				}
			}
		}
	}
	// Shipping=5 should reach at least as many as shipping=4
	if count5 < count4 {
		t.Fatalf("expected shipping=5 to reach at least as many hexes as shipping=4, got %d vs %d", count5, count4)
	}
}

func TestIndirectAdjacency_Shipping6_MaxReachability(t *testing.T) {
	m := NewTerraMysticaMap()
	// Test that shipping=6 (max) works correctly
	h1 := NewHex(0, 1) // Desert

	count5 := 0
	count6 := 0
	for q := 0; q < 13; q++ {
		for r := 0; r < 9; r++ {
			h2 := NewHex(q, r)
			if m.IsValidHex(h2) && !m.IsRiver(h2) && !h1.Equals(h2) {
				if m.IsIndirectlyAdjacent(h1, h2, 5) {
					count5++
				}
				if m.IsIndirectlyAdjacent(h1, h2, 6) {
					count6++
				}
			}
		}
	}
	// Shipping=6 should reach at least as many as shipping=5
	if count6 < count5 {
		t.Fatalf("expected shipping=6 to reach at least as many hexes as shipping=5, got %d vs %d", count6, count5)
	}
	// Verify max shipping actually works
	if count6 == 0 {
		t.Fatalf("expected shipping=6 to reach at least some hexes")
	}
}

func TestIndirectAdjacency_NoRiverPath(t *testing.T) {
	m := NewTerraMysticaMap()
	// Test two land hexes with no connecting river path
	// (0,0) Plains and (12,0) Swamp - opposite ends of row 0, no river between
	h1 := NewHex(0, 0)
	h2 := NewHex(12, 0)

	// Even with max shipping, cannot reach if no river path exists
	if ok := m.IsIndirectlyAdjacent(h1, h2, 6); ok {
		t.Fatalf("expected no river path from %v to %v even with shipping=6", h1, h2)
	}
}

func TestIndirectAdjacency_CrossMapShipping(t *testing.T) {
	m := NewTerraMysticaMap()
	// Test a 6 ship path
	d5 := NewHex(5, 3)
	i10 := NewHex(6, 8)

	// Check if reachable via river path with appropriate shipping
	if ok := m.IsIndirectlyAdjacent(d5, i10, 6); !ok {
		t.Fatalf("expected shipping=6 to potentially reach from %v to %v if river path exists", d5, i10)
	}
	// With shipping 5 should not reach
	if ok := m.IsIndirectlyAdjacent(d5, i10, 5); ok {
		t.Fatalf("expected shipping=5 to NOT reach from %v to %v", d5, i10)
	}
}
