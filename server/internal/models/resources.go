package models

// Resources tracks the player's resources
// Power is tracked as bowls: I, II, III

type Resources struct {
	Coins   int `json:"coins"`
	Workers int `json:"workers"`
	Priests int `json:"priests"`
	PowerI  int `json:"powerI"`
	PowerII int `json:"powerII"`
	PowerIII int `json:"powerIII"`
}

func (r *Resources) TotalPower() int { return r.PowerI + r.PowerII + r.PowerIII }
