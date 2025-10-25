package models

// CultType represents the four cult tracks

type CultType int

const (
	CultFire CultType = iota
	CultWater
	CultEarth
	CultAir
)

// PlayerState tracks per-player info used across the game

type PlayerState struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Faction   FactionType `json:"faction"`
	Resources Resources   `json:"resources"`
	Shipping  int         `json:"shipping"`
	Digging   int         `json:"digging"`
	// Cult positions by track (0-10)
	Cults map[CultType]int `json:"cults"`
	// Buildings on the map keyed by hex key
	Buildings map[string]Building `json:"buildings"`
}

// RoundState stores current round info

type RoundState struct {
	Round int `json:"round"` // 1..6
	// TODO: scoring tiles, bonus tiles ref ids later
}

// GameState is the authoritative state stored on the server

type GameState struct {
	ID          string                 `json:"id"`
	Players     map[string]*PlayerState `json:"players"`
	Order       []string               `json:"order"` // player ID order
	ActiveIndex int                    `json:"activeIndex"`
	Map         MapState               `json:"map"`
	Round       RoundState             `json:"round"`
	Started     bool                   `json:"started"`
	Finished    bool                   `json:"finished"`
}
