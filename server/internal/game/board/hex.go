package board

import (
	"fmt"
)

// Hex represents an axial coordinate in a pointy-top hex grid
// Terra Mystica uses pointy-top hexagons with 9 rows alternating 13/12 hexagons
type Hex struct {
	Q int // column (axial coordinate)
	R int // row (axial coordinate)
}

// NewHex creates a new hex coordinate
func NewHex(q, r int) Hex {
	return Hex{Q: q, R: r}
}

// String returns a string representation of the hex
func (h Hex) String() string {
	return fmt.Sprintf("(%d,%d)", h.Q, h.R)
}

// Equals checks if two hexes are equal
func (h Hex) Equals(other Hex) bool {
	return h.Q == other.Q && h.R == other.R
}

// Add adds two hex coordinates
func (h Hex) Add(other Hex) Hex {
	return Hex{Q: h.Q + other.Q, R: h.R + other.R}
}

// Subtract subtracts two hex coordinates
func (h Hex) Subtract(other Hex) Hex {
	return Hex{Q: h.Q - other.Q, R: h.R - other.R}
}

// Scale multiplies a hex coordinate by a scalar
func (h Hex) Scale(k int) Hex {
	return Hex{Q: h.Q * k, R: h.R * k}
}

// DirectionVectors returns the 6 direction vectors for pointy-top hexagons
// In axial coordinates, the 6 neighbors are at these offsets
var DirectionVectors = []Hex{
	{Q: 1, R: 0},  // East
	{Q: 1, R: -1}, // Northeast
	{Q: 0, R: -1}, // Northwest
	{Q: -1, R: 0}, // West
	{Q: -1, R: 1}, // Southwest
	{Q: 0, R: 1},  // Southeast
}

// Neighbor returns the neighbor in the given direction (0-5)
func (h Hex) Neighbor(direction int) Hex {
	return h.Add(DirectionVectors[direction])
}

// Neighbors returns all 6 neighbors of this hex
func (h Hex) Neighbors() []Hex {
	neighbors := make([]Hex, 6)
	for i := 0; i < 6; i++ {
		neighbors[i] = h.Neighbor(i)
	}
	return neighbors
}

// Distance calculates the Manhattan distance between two hexes in axial coordinates
// For axial coordinates, distance = (abs(q1-q2) + abs(r1-r2) + abs(s1-s2)) / 2
// where s = -q - r (derived from cube coordinate constraint)
func (h Hex) Distance(other Hex) int {
	dq := abs(h.Q - other.Q)
	dr := abs(h.R - other.R)
	// s = -q - r, so ds = abs((-h.Q - h.R) - (-other.Q - other.R))
	ds := abs((h.Q + h.R) - (other.Q + other.R))
	return (dq + dr + ds) / 2
}

// IsDirectlyAdjacent checks if two hexes share an edge (distance = 1)
func (h Hex) IsDirectlyAdjacent(other Hex) bool {
	return h.Distance(other) == 1
}

// IsWithinRange checks if another hex is within a given distance
func (h Hex) IsWithinRange(other Hex, distance int) bool {
	return h.Distance(other) <= distance
}

// Ring returns all hexes at exactly the given distance from this hex
func (h Hex) Ring(radius int) []Hex {
	if radius == 0 {
		return []Hex{h}
	}

	results := make([]Hex, 0, radius*6)
	// Start at a hex 'radius' steps away in direction 4 (southwest in our system)
	hex := h.Add(DirectionVectors[4].Scale(radius))

	// Walk around the ring
	for i := 0; i < 6; i++ {
		for j := 0; j < radius; j++ {
			results = append(results, hex)
			hex = hex.Neighbor(i)
		}
	}
	return results
}

// SpiralRange returns all hexes within a given radius (inclusive)
// Potentially useful for Fakirs with upgraded flights - revisit this and remove it if it turns out to be unnecessary
func (h Hex) SpiralRange(radius int) []Hex {
	results := []Hex{h}
	for k := 1; k <= radius; k++ {
		results = append(results, h.Ring(k)...)
	}
	return results
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
