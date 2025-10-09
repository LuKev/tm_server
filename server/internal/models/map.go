package models

import "fmt"

// Axial coordinates for hex grid (q, r)
// See https://www.redblobgames.com/grids/hexagons/

type Hex struct {
	Q int `json:"q"`
	R int `json:"r"`
}

type MapHex struct {
	Coord   Hex         `json:"coord"`
	Terrain TerrainType `json:"terrain"`
	// Building present on this hex (optional)
	Building *Building `json:"building,omitempty"`
}

type Building struct {
	OwnerPlayerID string       `json:"ownerPlayerId"`
	Faction       FactionType  `json:"faction"`
	Type          BuildingType `json:"type"`
	PowerValue    int          `json:"powerValue"` // Power value for town formation and leech
}

type MapState struct {
	Hexes map[string]*MapHex `json:"hexes"` // key: keyFromHex
}

func keyFromHex(h Hex) string { return fmtKey(h.Q, h.R) }

func fmtKey(q, r int) string {
	// compact string key to address map cells
	return fmt.Sprintf("%d,%d", q, r)
}
