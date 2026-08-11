package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/lukev/tm_server/internal/az"
	"github.com/lukev/tm_server/internal/models"
)

func main() {
	seed := int64(20260810)
	firstID, secondID := "p0", "p1"
	if len(os.Args) > 1 {
		parsed, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil {
			fatal(err)
		}
		seed = parsed
	}
	if len(os.Args) > 3 {
		firstID, secondID = os.Args[2], os.Args[3]
	}
	position, err := az.NewBaseGameWithPlayerIDs(seed, models.FactionWitches, models.FactionEngineers, firstID, secondID)
	if err != nil {
		fatal(err)
	}
	canonical, err := position.CanonicalJSON()
	if err != nil {
		fatal(err)
	}
	record := struct {
		State   json.RawMessage   `json:"state"`
		Actions []az.SearchAction `json:"actions"`
	}{State: canonical, Actions: position.LegalActions()}
	if err := json.NewEncoder(os.Stdout).Encode(record); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
