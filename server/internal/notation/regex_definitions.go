package notation

import "regexp"

var (
	reSendPriest      = regexp.MustCompile(`(.*) sends a Priest to the Order of the Cult of (\w+)`)
	reReclaimPriest   = regexp.MustCompile(`(.*) sends a Priest to the Order of the Cult of (\w+) then reclaims it`)
	reAdvanceCult     = regexp.MustCompile(`(.*) gains (\d+) on the Cult of (\w+) track`)
	reAurenStronghold = regexp.MustCompile(`(.*) gains 2 on the Cult of (\w+) track \(Auren Stronghold\)`)
	reFavorTileAction = regexp.MustCompile(`(.*) gains (\d+) on the Cult of (\w+) track \(Favor tile(?: action)?\)`)
	reBonusCardCult   = regexp.MustCompile(`(.*) gains 1 on the Cult of (\w+) track \(Bonus card action\)`)
	reBonusCardSpade  = regexp.MustCompile(`(.*) transforms a Terrain space.* \(Bonus card action\)`)
	reBridgePower     = regexp.MustCompile(`(.*) spends \d+ power to build a Bridge \(Power action\)`)
	reConversion      = regexp.MustCompile(`(.*) does some Conversions \(spent: (.*) ; collects: (.*)\)`)
	reAlchemistsVP    = regexp.MustCompile(`(.*) converts (\d+) VP into (\d+) coins \(Alchemists ability\)`)
)
