package game

import (
	"testing"
)

func TestHexEquals(t *testing.T) {
	h1 := NewHex(3, 4)
	h2 := NewHex(3, 4)
	h3 := NewHex(3, 5)
	
	if !h1.Equals(h2) {
		t.Errorf("Expected h1 to equal h2")
	}
	if h1.Equals(h3) {
		t.Errorf("Expected h1 to not equal h3")
	}
}

func TestHexAdd(t *testing.T) {
	h1 := NewHex(1, 2)
	h2 := NewHex(3, 4)
	result := h1.Add(h2)
	expected := NewHex(4, 6)
	
	if !result.Equals(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestHexSubtract(t *testing.T) {
	h1 := NewHex(5, 7)
	h2 := NewHex(2, 3)
	result := h1.Subtract(h2)
	expected := NewHex(3, 4)
	
	if !result.Equals(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestHexScale(t *testing.T) {
	h := NewHex(2, 3)
	result := h.Scale(3)
	expected := NewHex(6, 9)
	
	if !result.Equals(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestHexNeighbors(t *testing.T) {
	h := NewHex(5, 5)
	neighbors := h.Neighbors()
	
	if len(neighbors) != 6 {
		t.Errorf("Expected 6 neighbors, got %d", len(neighbors))
	}
	
	// Check specific neighbors
	expected := []Hex{
		NewHex(6, 5),  // East
		NewHex(6, 4),  // Northeast
		NewHex(5, 4),  // Northwest
		NewHex(4, 5),  // West
		NewHex(4, 6),  // Southwest
		NewHex(5, 6),  // Southeast
	}
	
	for i, exp := range expected {
		if !neighbors[i].Equals(exp) {
			t.Errorf("Neighbor %d: expected %v, got %v", i, exp, neighbors[i])
		}
	}
}

func TestHexDistance(t *testing.T) {
	tests := []struct {
		h1       Hex
		h2       Hex
		expected int
	}{
		{NewHex(0, 0), NewHex(0, 0), 0},
		{NewHex(0, 0), NewHex(1, 0), 1},
		{NewHex(0, 0), NewHex(0, 1), 1},
		{NewHex(0, 0), NewHex(1, -1), 1},
		{NewHex(0, 0), NewHex(2, 0), 2},
		{NewHex(0, 0), NewHex(2, -2), 2},
		{NewHex(0, 0), NewHex(3, -1), 3},
		{NewHex(3, 4), NewHex(6, 8), 7},
	}
	
	for _, tt := range tests {
		result := tt.h1.Distance(tt.h2)
		if result != tt.expected {
			t.Errorf("Distance from %v to %v: expected %d, got %d", 
				tt.h1, tt.h2, tt.expected, result)
		}
		
		// Distance should be symmetric
		reverseResult := tt.h2.Distance(tt.h1)
		if reverseResult != tt.expected {
			t.Errorf("Distance from %v to %v: expected %d, got %d (reverse)", 
				tt.h2, tt.h1, tt.expected, reverseResult)
		}
	}
}

func TestHexIsDirectlyAdjacent(t *testing.T) {
	center := NewHex(5, 5)
	
	// All 6 neighbors should be directly adjacent
	for _, neighbor := range center.Neighbors() {
		if !center.IsDirectlyAdjacent(neighbor) {
			t.Errorf("Expected %v to be adjacent to %v", neighbor, center)
		}
	}
	
	// Non-adjacent hexes
	notAdjacent := []Hex{
		NewHex(7, 5),  // Distance 2
		NewHex(3, 3),  // Distance 3
		NewHex(10, 10), // Far away
	}
	
	for _, hex := range notAdjacent {
		if center.IsDirectlyAdjacent(hex) {
			t.Errorf("Expected %v to not be adjacent to %v", hex, center)
		}
	}
}

func TestHexIsWithinRange(t *testing.T) {
	center := NewHex(5, 5)
	
	tests := []struct {
		target   Hex
		distance int
		expected bool
	}{
		{NewHex(5, 5), 0, true},
		{NewHex(6, 5), 1, true},
		{NewHex(7, 5), 1, false},
		{NewHex(7, 5), 2, true},
		{NewHex(8, 3), 3, true},
		{NewHex(8, 3), 2, false},
	}
	
	for _, tt := range tests {
		result := center.IsWithinRange(tt.target, tt.distance)
		if result != tt.expected {
			t.Errorf("IsWithinRange(%v, %v, %d): expected %v, got %v",
				center, tt.target, tt.distance, tt.expected, result)
		}
	}
}

func TestHexRing(t *testing.T) {
	center := NewHex(0, 0)
	
	// Ring of radius 0 should just be the center
	ring0 := center.Ring(0)
	if len(ring0) != 1 || !ring0[0].Equals(center) {
		t.Errorf("Ring(0) should return just the center hex")
	}
	
	// Ring of radius 1 should have 6 hexes
	ring1 := center.Ring(1)
	if len(ring1) != 6 {
		t.Errorf("Ring(1) should have 6 hexes, got %d", len(ring1))
	}
	
	// All hexes in ring 1 should be distance 1 from center
	for _, hex := range ring1 {
		if center.Distance(hex) != 1 {
			t.Errorf("Hex %v in ring 1 has distance %d from center", hex, center.Distance(hex))
		}
	}
	
	// Ring of radius 2 should have 12 hexes
	ring2 := center.Ring(2)
	if len(ring2) != 12 {
		t.Errorf("Ring(2) should have 12 hexes, got %d", len(ring2))
	}
	
	// All hexes in ring 2 should be distance 2 from center
	for _, hex := range ring2 {
		if center.Distance(hex) != 2 {
			t.Errorf("Hex %v in ring 2 has distance %d from center", hex, center.Distance(hex))
		}
	}
}

func TestHexSpiralRange(t *testing.T) {
	center := NewHex(0, 0)
	
	// Spiral range 0 should just be the center
	spiral0 := center.SpiralRange(0)
	if len(spiral0) != 1 {
		t.Errorf("SpiralRange(0) should have 1 hex, got %d", len(spiral0))
	}
	
	// Spiral range 1 should have 1 + 6 = 7 hexes
	spiral1 := center.SpiralRange(1)
	if len(spiral1) != 7 {
		t.Errorf("SpiralRange(1) should have 7 hexes, got %d", len(spiral1))
	}
	
	// Spiral range 2 should have 1 + 6 + 12 = 19 hexes
	spiral2 := center.SpiralRange(2)
	if len(spiral2) != 19 {
		t.Errorf("SpiralRange(2) should have 19 hexes, got %d", len(spiral2))
	}
	
	// All hexes should be within distance
	for _, hex := range spiral2 {
		if center.Distance(hex) > 2 {
			t.Errorf("Hex %v in spiral range 2 has distance %d from center", hex, center.Distance(hex))
		}
	}
}
