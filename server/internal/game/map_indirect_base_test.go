package game

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
	// h1 at (0,1) is Desert (land); neighbor (0,2) is River
	// h2 at (1,2) is River - but endpoint must be land, so use (2,2) which is Swamp
	h1 := NewHex(0, 1) // Desert
	h2 := NewHex(2, 2) // Swamp, adjacent to river (1,2)

	// From (0,1) Desert, river neighbor is (0,2)
	// From (0,2) river, can reach (1,2) river in 1 step
	// (1,2) river is adjacent to (2,2) Swamp
	if ok := m.IsIndirectlyAdjacent(h1, h2, 2); !ok {
		t.Fatalf("expected shipping=2 to reach from %v to %v", h1, h2)
	}
	// With shipping 1 should not reach
	if ok := m.IsIndirectlyAdjacent(h1, h2, 1); ok {
		t.Fatalf("expected shipping=1 to NOT reach from %v to %v", h1, h2)
	}
}

func TestIndirectAdjacency_Shipping2_RiverChain(t *testing.T) {
	m := NewTerraMysticaMap()
	// Use row 1-2 river chain: (1,1) and (1,2) are both rivers
	// Start from (0,1) Desert, end at (2,2) Swamp
	h1 := NewHex(0, 1) // Desert
	h2 := NewHex(2, 2) // Swamp

	if ok := m.IsIndirectlyAdjacent(h1, h2, 2); !ok {
		t.Fatalf("expected shipping=2 to reach from %v to %v via river chain", h1, h2)
	}
	// With shipping 1 should not reach
	if ok := m.IsIndirectlyAdjacent(h1, h2, 1); ok {
		t.Fatalf("expected shipping=1 to NOT reach from %v to %v", h1, h2)
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
	// Test shipping=3 along row 2 river chain
	// Row 2: River, River, Swamp, River, Mountain, River, Forest, River, Forest, River, Mountain, River, River
	// Path: (0,1) Desert -> (0,2) River -> (1,2) River -> (3,2) River -> (2,2) Swamp
	h1 := NewHex(0, 1) // Desert
	h2 := NewHex(2, 2) // Swamp
	
	// Already tested this as shipping=2, so test that shipping=3 also works
	if ok := m.IsIndirectlyAdjacent(h1, h2, 3); !ok {
		t.Fatalf("expected shipping=3 to reach from %v to %v", h1, h2)
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
			if m.IsValidHex(h2) && !m.isRiver(h2) && !h1.Equals(h2) {
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
			if m.IsValidHex(h2) && !m.isRiver(h2) && !h1.Equals(h2) {
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
			if m.IsValidHex(h2) && !m.isRiver(h2) && !h1.Equals(h2) {
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
			if m.IsValidHex(h2) && !m.isRiver(h2) && !h1.Equals(h2) {
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
	// Test shipping across different rows via river network
	// From row 1 to row 3 through river hexes
	h1 := NewHex(3, 1) // Plains (row 1)
	h2 := NewHex(5, 3) // Wasteland (row 3)
	
	// Check if reachable via river path with appropriate shipping
	if ok := m.IsIndirectlyAdjacent(h1, h2, 6); !ok {
		t.Fatalf("expected shipping=6 to potentially reach from %v to %v if river path exists", h1, h2)
	}
}
