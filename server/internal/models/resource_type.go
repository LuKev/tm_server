package models

type ResourceType int

const (
	ResourcePower ResourceType = iota
	ResourcePriest
	ResourceWorker
	ResourceCoin
	ResourceVictoryPoint
)

func (r ResourceType) String() string {
	switch r {
	case ResourcePower:
		return "Power"
	case ResourcePriest:
		return "Priest"
	case ResourceWorker:
		return "Worker"
	case ResourceCoin:
		return "Coin"
	case ResourceVictoryPoint:
		return "VP"
	default:
		return "Unknown"
	}
}
