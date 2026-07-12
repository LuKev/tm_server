package stats

type FinalScoreRate struct {
	Samples      int     `json:"samples"`
	TotalScore   int     `json:"totalScore"`
	AverageScore float64 `json:"averageScore"`
}

type FinalScoreRates map[string]FinalScoreRate

func AddFinalScore(rates FinalScoreRates, faction string, score int) {
	if faction == "" {
		return
	}
	entry := rates[faction]
	entry.Samples++
	entry.TotalScore += score
	entry.AverageScore = float64(entry.TotalScore) / float64(entry.Samples)
	rates[faction] = entry
}

func MergeFinalScoreRates(dst, src FinalScoreRates) {
	for faction, part := range src {
		entry := dst[faction]
		entry.Samples += part.Samples
		entry.TotalScore += part.TotalScore
		if entry.Samples > 0 {
			entry.AverageScore = float64(entry.TotalScore) / float64(entry.Samples)
		}
		dst[faction] = entry
	}
}
