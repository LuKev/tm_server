package notation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// LogItem represents an item in the game log, which can be a player action or metadata.
type LogItem interface {
	isLogItem()
}

// LogLocation represents a specific location in the concise log (line and column)
type LogLocation struct {
	LineIndex   int `json:"lineIndex"`
	ColumnIndex int `json:"columnIndex"`
}

// ActionItem wraps a standard game action.
type ActionItem struct {
	Action game.Action
}

func (i ActionItem) isLogItem() {}

// GameSettingsItem represents the game configuration header.
type GameSettingsItem struct {
	Settings map[string]string
}

func (i GameSettingsItem) isLogItem() {}

// RoundStartItem marks the start of a new round.
type RoundStartItem struct {
	Round     int
	TurnOrder []string
}

// LogAcceptLeechAction is a log-only representation of accepting leech
type LogAcceptLeechAction struct {
	PlayerID     string
	FromPlayerID string // optional: expected leech source
	PowerAmount  int
	VPCost       int
	Explicit     bool
}

// GetType returns the action type.
func (a *LogAcceptLeechAction) GetType() game.ActionType { return game.ActionAcceptPowerLeech }

// GetPlayerID returns the player ID.
func (a *LogAcceptLeechAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogAcceptLeechAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogAcceptLeechAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	offers := gs.PendingLeechOffers[a.PlayerID]
	if len(offers) == 0 {
		// If the player has no capacity to gain power (Bowl I+II empty), Snellman can still
		// include a "Leech ..." row. Treat it as an automatic decline/no-op.
		if player.Resources != nil && player.Resources.Power != nil {
			capacity := player.Resources.Power.Bowl2 + 2*player.Resources.Power.Bowl1
			if capacity <= 0 {
				return nil
			}
		}
		return fmt.Errorf("no pending leech offers for %q", a.PlayerID)
	}

	idx := 0
	if a.FromPlayerID != "" {
		foundIdx := -1
		for i, offer := range offers {
			if offer == nil || !strings.EqualFold(offer.FromPlayerID, a.FromPlayerID) {
				continue
			}
			// If amount is explicit, prefer an exact amount match when available.
			if a.Explicit && a.PowerAmount > 0 && offer.Amount != a.PowerAmount {
				continue
			}
			foundIdx = i
			break
		}
		if foundIdx < 0 {
			// Capacity 0: treat as auto-decline/no-op even if Snellman row claims an accept.
			if player.Resources != nil && player.Resources.Power != nil {
				capacity := player.Resources.Power.Bowl2 + 2*player.Resources.Power.Bowl1
				if capacity <= 0 {
					return nil
				}
			}
			// Include current offers to help diagnose mis-bindings in imported logs.
			summary := make([]string, 0, len(offers))
			for _, o := range offers {
				if o == nil {
					continue
				}
				summary = append(summary, fmt.Sprintf("%s:%d", o.FromPlayerID, o.Amount))
			}
			return fmt.Errorf("no pending leech offer from %q for %q (pending=%v)", a.FromPlayerID, a.PlayerID, summary)
		}
		idx = foundIdx
	}

	if a.Explicit && a.PowerAmount > 0 && offers[idx] != nil {
		offers[idx].Amount = a.PowerAmount
		offers[idx].VPCost = a.VPCost
	}
	// Use the strict game action so Cultists leech bonuses resolve consistently.
	if err := game.NewAcceptPowerLeechAction(a.PlayerID, idx).Execute(gs); err != nil {
		return fmt.Errorf("failed to accept pending leech offer: %w", err)
	}
	return nil
}

// LogDeclineLeechAction is a log-only representation of declining leech.
// Unlike the strict game decline action, this is tolerant when no offer is pending.
type LogDeclineLeechAction struct {
	PlayerID     string
	FromPlayerID string // optional: expected leech source
}

// GetType returns the action type.
func (a *LogDeclineLeechAction) GetType() game.ActionType { return game.ActionDeclinePowerLeech }

// GetPlayerID returns the player ID.
func (a *LogDeclineLeechAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogDeclineLeechAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogDeclineLeechAction) Execute(gs *game.GameState) error {
	offers := gs.PendingLeechOffers[a.PlayerID]
	if len(offers) == 0 {
		// Snellman logs can include delayed/defensive decline rows where
		// our reconstructed offer state has already been resolved.
		return nil
	}
	idx := 0
	if a.FromPlayerID != "" {
		found := false
		for i, offer := range offers {
			if offer != nil && strings.EqualFold(offer.FromPlayerID, a.FromPlayerID) {
				idx = i
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no pending leech offer from %q for %q", a.FromPlayerID, a.PlayerID)
		}
	}
	return game.NewDeclinePowerLeechAction(a.PlayerID, idx).Execute(gs)
}

// LogPowerAction is a log-only representation of a power action
type LogPowerAction struct {
	PlayerID   string
	ActionCode string // e.g. "ACT1", "ACT6"
}

// GetType returns the action type.
func (a *LogPowerAction) GetType() game.ActionType { return game.ActionPowerAction }

// GetPlayerID returns the player ID.
func (a *LogPowerAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogPowerAction) Validate(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	actionType := ParsePowerActionCode(a.ActionCode)
	if actionType == game.PowerActionUnknown {
		return fmt.Errorf("unknown power action code: %s", a.ActionCode)
	}

	if !gs.PowerActions.IsAvailable(actionType) {
		return fmt.Errorf("power action %v has already been taken this round", actionType)
	}

	powerCost := game.GetPowerCost(actionType)
	if player.Resources.Power.Bowl3 < powerCost {
		return fmt.Errorf("not enough power in Bowl III: need %d, have %d", powerCost, player.Resources.Power.Bowl3)
	}

	return nil
}

// Execute applies the action to the game state.
func (a *LogPowerAction) Execute(gs *game.GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}

	player := gs.GetPlayer(a.PlayerID)
	actionType := ParsePowerActionCode(a.ActionCode)

	// Spend power from Bowl III only. Explicit burns are represented as separate log actions.
	powerCost := game.GetPowerCost(actionType)
	if err := player.Resources.Power.SpendPower(powerCost); err != nil {
		return fmt.Errorf("failed to spend power for %s: %w", a.ActionCode, err)
	}

	// Mark used
	gs.PowerActions.MarkUsed(actionType)

	// Apply effects
	switch actionType {
	case game.PowerActionBridge:
		// Parse coordinates if present: ACT1-C2-D4
		parts := strings.Split(a.ActionCode, "-")
		if len(parts) == 3 {
			hex1, err := ConvertLogCoordToAxial(parts[1])
			if err != nil {
				return fmt.Errorf("invalid bridge hex1: %w", err)
			}
			hex2, err := ConvertLogCoordToAxial(parts[2])
			if err != nil {
				return fmt.Errorf("invalid bridge hex2: %w", err)
			}

			if err := gs.Map.BuildBridge(hex1, hex2, a.PlayerID); err != nil {
				return fmt.Errorf("failed to build bridge: %w", err)
			}

			// Check for town formation after building bridge
			gs.CheckAllTownFormations(a.PlayerID)
		}

		// Increment bridge count (game logic handles limit check in Validate, but we are in Execute)
		player.BridgesBuilt++

	case game.PowerActionPriest:
		gs.GainPriests(a.PlayerID, 1)
	case game.PowerActionWorkers:
		player.Resources.Workers += 2
	case game.PowerActionCoins:
		player.Resources.Coins += 7
	case game.PowerActionSpade1:
		gs.PendingSpades[a.PlayerID]++
	case game.PowerActionSpade2:
		gs.PendingSpades[a.PlayerID] += 2
	}

	return nil
}

// LogBurnAction is a log-only representation of burning power
type LogBurnAction struct {
	PlayerID string
	Amount   int
}

// GetType returns the action type.
func (a *LogBurnAction) GetType() game.ActionType { return game.ActionSpecialAction } // Placeholder
// GetPlayerID returns the player ID.
func (a *LogBurnAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogBurnAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogBurnAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	// Burn power: move from Bowl 2 to Bowl 3
	// Amount is 2 * power gained.
	burned := a.Amount
	gained := burned // usually 1:1

	if player.Resources.Power.Bowl2 < burned+gained {
		return fmt.Errorf("insufficient power in bowl 2 to burn: %d available, %d required", player.Resources.Power.Bowl2, burned+gained)
	}

	player.Resources.Power.Bowl2 -= burned
	player.Resources.Power.Bowl2 -= gained
	player.Resources.Power.Bowl3 += gained

	return nil
}

// LogDigTransformAction represents a Snellman "dig N" step (terraform by N spades)
// against a specific hex. This is used to preserve intra-row ordering for cases like
// Alchemists stronghold (gain power per spade), where conversions can be interleaved
// with dig steps in the ledger.
type LogDigTransformAction struct {
	PlayerID string
	Spades   int
	Target   board.Hex
}

func (a *LogDigTransformAction) GetType() game.ActionType { return game.ActionSpecialAction } // Placeholder
func (a *LogDigTransformAction) GetPlayerID() string      { return a.PlayerID }
func (a *LogDigTransformAction) Validate(gs *game.GameState) error {
	if a == nil {
		return fmt.Errorf("nil dig action")
	}
	if a.Spades <= 0 {
		return fmt.Errorf("invalid dig spades: %d", a.Spades)
	}
	return nil
}

func (a *LogDigTransformAction) Execute(gs *game.GameState) error {
	if err := a.Validate(gs); err != nil {
		return err
	}
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}
	mapHex := gs.Map.GetHex(a.Target)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", a.Target)
	}

	home := player.Faction.GetHomeTerrain()
	target := board.CalculateIntermediateTerrain(mapHex.Terrain, home, a.Spades)

	// Apply ONLY the terraform step (no building). This must not call TransformAndBuildAction
	// because that would charge/award full terraform logic again when the later build action
	// runs in the same compound row, causing resource divergence.
	needsTransform := mapHex.Terrain != target
	if !needsTransform {
		return nil
	}

	// Distance should match the spade count implied by the intermediate terrain, but compute
	// it from the board state to keep this robust to odd terrain encodings.
	distance := gs.Map.GetTerrainDistance(mapHex.Terrain, target)
	if distance <= 0 {
		return nil
	}

	// Consume free spades (BON1) first. These count for VP.
	vpEligibleFreeSpades := 0
	if gs.PendingSpades != nil && gs.PendingSpades[a.PlayerID] > 0 {
		vpEligibleFreeSpades = gs.PendingSpades[a.PlayerID]
		if vpEligibleFreeSpades > distance {
			vpEligibleFreeSpades = distance
		}
		gs.PendingSpades[a.PlayerID] -= vpEligibleFreeSpades
		if gs.PendingSpades[a.PlayerID] == 0 {
			delete(gs.PendingSpades, a.PlayerID)
		}
	}

	// Consume cult-reward spades next. These do NOT count for VP.
	remainingDistance := distance - vpEligibleFreeSpades
	cultRewardSpades := 0
	if remainingDistance > 0 && gs.PendingCultRewardSpades != nil && gs.PendingCultRewardSpades[a.PlayerID] > 0 {
		cultRewardSpades = gs.PendingCultRewardSpades[a.PlayerID]
		if cultRewardSpades > remainingDistance {
			cultRewardSpades = remainingDistance
		}
		gs.PendingCultRewardSpades[a.PlayerID] -= cultRewardSpades
		if gs.PendingCultRewardSpades[a.PlayerID] == 0 {
			delete(gs.PendingCultRewardSpades, a.PlayerID)
		}
	}

	totalFreeSpades := vpEligibleFreeSpades + cultRewardSpades
	paidSpades := distance - totalFreeSpades

	// Pay for remaining spades.
	if paidSpades > 0 {
		// Darklings pay priests for terraform (instead of workers).
		if player.Faction.GetType() == models.FactionDarklings {
			player.Resources.Priests -= paidSpades
			// Darklings: +2 VP per PAID spade.
			player.VictoryPoints += paidSpades * 2
		} else {
			player.Resources.Workers -= player.Faction.GetTerraformCost(paidSpades)
		}
	}

	// Apply terrain change.
	if err := gs.Map.TransformTerrain(a.Target, target); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Award VP + faction bonuses for VP-eligible spades (paid + BON1 spades).
	vpEligibleDistance := paidSpades + vpEligibleFreeSpades
	if vpEligibleDistance > 0 {
		spadesForVP := vpEligibleDistance
		if player.Faction.GetType() == models.FactionGiants {
			spadesForVP = 2
		}
		for i := 0; i < spadesForVP; i++ {
			gs.AwardActionVP(a.PlayerID, game.ScoringActionSpades)
		}
		game.AwardFactionSpadeBonuses(player, spadesForVP)
	}

	// Award faction bonuses for cult-reward spades too (no VP).
	if cultRewardSpades > 0 {
		spadesUsed := cultRewardSpades
		if player.Faction.GetType() == models.FactionGiants {
			spadesUsed = 2
		}
		game.AwardFactionSpadeBonuses(player, spadesUsed)
	}

	return nil
}

// LogFavorTileAction is a log-only representation of taking a favor tile
type LogFavorTileAction struct {
	PlayerID string
	Tile     string // e.g. "FAV-F1"
}

// GetType returns the action type.
func (a *LogFavorTileAction) GetType() game.ActionType { return game.ActionSelectFavorTile }

// GetPlayerID returns the player ID.
func (a *LogFavorTileAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogFavorTileAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogFavorTileAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	tileType, err := ParseFavorTileCode(a.Tile)
	if err != nil {
		return err
	}

	// Take the tile
	if err := gs.FavorTiles.TakeFavorTile(a.PlayerID, tileType); err != nil {
		return err
	}

	// Check for town formation (e.g. Fire+2 reduces requirement)
	gs.CheckAllTownFormations(a.PlayerID)

	// Apply immediate effects
	if err := game.ApplyFavorTileImmediate(gs, a.PlayerID, tileType); err != nil {
		return err
	}

	// Clear pending selection if any
	if gs.PendingFavorTileSelection != nil && gs.PendingFavorTileSelection.PlayerID == a.PlayerID {
		gs.PendingFavorTileSelection.Count--
		if gs.PendingFavorTileSelection.Count <= 0 {
			gs.PendingFavorTileSelection = nil
		}
	}

	return nil
}

// LogSpecialAction is a log-only representation of special faction actions (e.g. Witches Ride)
type LogSpecialAction struct {
	PlayerID   string
	ActionCode string // e.g. "ACT-SH-D-C4"
}

// GetType returns the action type.
func (a *LogSpecialAction) GetType() game.ActionType { return game.ActionSpecialAction }

// GetPlayerID returns the player ID.
func (a *LogSpecialAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogSpecialAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogSpecialAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	parts := strings.Split(a.ActionCode, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid special action code: %s", a.ActionCode)
	}

	switch parts[0] {
	case "ACT":
		if parts[1] == "SH" {
			return a.executeStrongholdAction(gs, player, parts)
		} else if parts[1] == "FAV" {
			// ACT-FAV-W
			if len(parts) < 3 {
				return fmt.Errorf("invalid favor action code")
			}
			track := GetCultTrackFromCode(parts[2])
			action := game.NewWater2CultAdvanceAction(a.PlayerID, track)
			return action.Execute(gs)
		} else if parts[1] == "BON" {
			// ACT-BON-W (Bonus Card cult advance, BON2)
			if len(parts) < 3 {
				return fmt.Errorf("invalid bonus card cult action code")
			}
			track := GetCultTrackFromCode(parts[2])
			action := game.NewBonusCardCultAction(a.PlayerID, track)
			return action.Execute(gs)
		} else if parts[1] == "TOWN" {
			// ACT-TOWN-Q_R (Mermaids river town)
			if len(parts) < 3 {
				return fmt.Errorf("invalid mermaids town action code")
			}
			// Parse axial coordinates from "Q_R" format
			coordParts := strings.Split(parts[2], "_")
			if len(coordParts) != 2 {
				return fmt.Errorf("invalid river hex coordinates: %s", parts[2])
			}
			var q, r int
			if _, err := fmt.Sscanf(coordParts[0], "%d", &q); err != nil {
				return fmt.Errorf("invalid Q coordinate: %w", err)
			}
			if _, err := fmt.Sscanf(coordParts[1], "%d", &r); err != nil {
				return fmt.Errorf("invalid R coordinate: %w", err)
			}
			riverHex := board.NewHex(q, r)

			// Mermaids river town: Mark the river hex as part of a pending town
			// This will be completed by the subsequent TW[VP] action
			action := game.NewMermaidsRiverTownAction(a.PlayerID, riverHex)
			return action.Execute(gs)
		} else if parts[1] == "BR" {
			// Engineers Bridge: ACT-BR-[Coord]-[Coord]
			if len(parts) < 4 {
				return fmt.Errorf("invalid engineers bridge action code")
			}
			if player.Faction.GetType() != models.FactionEngineers {
				return fmt.Errorf("engineers bridge action only valid for engineers")
			}
			// Engineers stronghold bridge costs 2 workers (no coins).
			if player.Resources.Workers < 2 {
				return fmt.Errorf("not enough workers for engineers bridge: need 2, have %d", player.Resources.Workers)
			}
			player.Resources.Workers -= 2
			hex1, err := ConvertLogCoordToAxial(parts[2])
			if err != nil {
				return err
			}
			hex2, err := ConvertLogCoordToAxial(parts[3])
			if err != nil {
				return err
			}

			if err := gs.Map.BuildBridge(hex1, hex2, a.PlayerID); err != nil {
				return fmt.Errorf("failed to build bridge: %w", err)
			}

			// Check for town formation after building bridge
			gs.CheckAllTownFormations(a.PlayerID)

			// Track bridge usage (used for the 3-bridge limit).
			player.BridgesBuilt++
			return nil
		}
	case "ACTS":
		// Bonus Card Spade: ACTS-[Coord] or ACTS-[Coord].[coord] (with dwelling build)
		if len(parts) < 2 {
			return fmt.Errorf("missing coord for ACTS")
		}

		// Check for combined format: ACTS-G3.g3 (transform at G3, build dwelling at g3)
		coordPart := parts[1]
		buildDwelling := false
		if dotIdx := strings.Index(coordPart, "."); dotIdx > 0 {
			// Combined action - extract just the coord and set BuildDwelling=true
			coordPart = coordPart[:dotIdx]
			buildDwelling = true
		}

		hex, err := ConvertLogCoordToAxial(coordPart)
		if err != nil {
			return err
		}

		targetTerrain := models.TerrainTypeUnknown
		if len(parts) > 2 {
			targetTerrain = parseTerrainShortCode(parts[2])
		}

		action := game.NewBonusCardSpadeAction(a.PlayerID, hex, buildDwelling, targetTerrain)
		return action.Execute(gs)
	case "ORD":
		// Darklings Ordination: ORD-[N]
		if len(parts) < 2 {
			return fmt.Errorf("missing amount for ORD")
		}
		amount := 0
		if _, err := fmt.Sscanf(parts[1], "%d", &amount); err != nil {
			return err
		}
		if amount > 0 {
			if player.Resources.Workers >= amount {
				player.Resources.Workers -= amount
				gs.GainPriests(a.PlayerID, amount)
			}
		}
		return nil
	}

	return nil
}

func (a *LogSpecialAction) executeStrongholdAction(gs *game.GameState, player *game.Player, parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("invalid stronghold action code: %s", a.ActionCode)
	}

	switch parts[2] {
	case "D": // Witches Ride (Free Dwelling)
		if len(parts) < 4 {
			return fmt.Errorf("missing coord for ACT-SH-D")
		}
		hex, err := ConvertLogCoordToAxial(parts[3])
		if err != nil {
			return err
		}

		if player.Faction.GetType() == models.FactionWitches {
			// Witches Ride
			action := game.NewWitchesRideAction(a.PlayerID, hex)
			return action.Execute(gs)
		}
		return fmt.Errorf("ACT-SH-D is only valid for Witches, got faction %v", player.Faction.GetType())

	case "T": // Nomads Sandstorm (Transform to Desert)
		if len(parts) < 4 {
			return fmt.Errorf("missing coord for ACT-SH-T")
		}

		// Check for combined format: ACT-SH-T-F4.f4 (sandstorm + build dwelling)
		coordPart := parts[3]
		buildDwelling := false
		if dotIdx := strings.Index(coordPart, "."); dotIdx > 0 {
			// Combined action - extract just the coord and set BuildDwelling=true
			coordPart = coordPart[:dotIdx]
			buildDwelling = true
		}

		hex, err := ConvertLogCoordToAxial(coordPart)
		if err != nil {
			return err
		}

		if player.Faction.GetType() == models.FactionNomads {
			action := game.NewNomadsSandstormAction(a.PlayerID, hex, buildDwelling)
			return action.Execute(gs)
		}
		return fmt.Errorf("ACT-SH-T is only valid for Nomads, got faction %v", player.Faction.GetType())

	case "S": // Giants (2 Spades)
		if len(parts) < 4 {
			return fmt.Errorf("missing coord for ACT-SH-S")
		}
		hex, err := ConvertLogCoordToAxial(parts[3])
		if err != nil {
			return err
		}
		// Assume BuildDwelling=false for "S" (Spade)
		action := game.NewGiantsTransformAction(a.PlayerID, hex, false)
		return action.Execute(gs)

	case "TP": // Swarmlings (Upgrade to TP)
		if len(parts) < 4 {
			return fmt.Errorf("missing coord for ACT-SH-TP")
		}
		hex, err := ConvertLogCoordToAxial(parts[3])
		if err != nil {
			return err
		}
		action := game.NewSwarmlingsUpgradeAction(a.PlayerID, hex)
		return action.Execute(gs)

	case "2X": // Chaos Magicians (Double Turn)
		// Just mark the ability as used. Sub-actions follow in the log.
		player.SpecialActionsUsed[game.SpecialActionChaosMagiciansDoubleTurn] = true
		return nil

	case "W", "F", "E", "A": // Auren Cult Advance
		track := GetCultTrackFromCode(parts[2])
		action := game.NewAurenCultAdvanceAction(a.PlayerID, track)
		return action.Execute(gs)
	}
	return nil
}

// GetCultTrackFromCode converts a code to a CultTrack.
func GetCultTrackFromCode(code string) game.CultTrack {
	switch code {
	case "F":
		return game.CultFire
	case "W":
		return game.CultWater
	case "E":
		return game.CultEarth
	case "A":
		return game.CultAir
	}
	return game.CultFire
}

// LogConversionAction is a log-only representation of a conversion action
type LogConversionAction struct {
	PlayerID string
	Cost     map[models.ResourceType]int
	Reward   map[models.ResourceType]int
}

// GetType returns the action type.
func (a *LogConversionAction) GetType() game.ActionType { return game.ActionSpecialAction } // Placeholder
// GetPlayerID returns the player ID.
func (a *LogConversionAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogConversionAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogConversionAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Spend cost
	// We need to map models.ResourceType to factions.Cost
	powerCost := a.Cost[models.ResourcePower]
	cost := factions.Cost{
		Workers: a.Cost[models.ResourceWorker],
		Priests: a.Cost[models.ResourcePriest],
		Coins:   a.Cost[models.ResourceCoin],
	}
	// Power cost in conversion usually means "Spend Power" (Bowl III -> Bowl I)
	// But factions.Cost.Power usually means "Gain Power" or "Spend Power" depending on context.
	// Player.Resources.Spend handles Power as "Spend from Bowl III".
	cost.Power = powerCost

	// VP cost? (Alchemists)
	vpCost := a.Cost[models.ResourceVictoryPoint]
	if vpCost > 0 {
		player.VictoryPoints -= vpCost
	}

	if err := player.Resources.Spend(cost); err != nil {
		return err
	}

	// Gain reward
	player.Resources.Workers += a.Reward[models.ResourceWorker]
	player.Resources.Priests += a.Reward[models.ResourcePriest]
	player.Resources.Coins += a.Reward[models.ResourceCoin]

	if p := a.Reward[models.ResourcePower]; p > 0 {
		player.Resources.GainPower(p)
	}

	if vp := a.Reward[models.ResourceVictoryPoint]; vp > 0 {
		player.VictoryPoints += vp
	}

	return nil
}

// LogCompoundAction is a log-only representation of multiple actions chained together
type LogCompoundAction struct {
	Actions []game.Action
}

// GetType returns the action type.
func (a *LogCompoundAction) GetType() game.ActionType { return game.ActionSpecialAction } // Placeholder
// GetPlayerID returns the player ID.
func (a *LogCompoundAction) GetPlayerID() string {
	if len(a.Actions) > 0 {
		return a.Actions[0].GetPlayerID()
	}
	return ""
}

// Validate checks if the action is valid.
func (a *LogCompoundAction) Validate(gs *game.GameState) error { return nil }

// LogPreIncomeAction marks an action that occurs before the round's normal income is granted
// (e.g. Snellman interlude actions between two "Round X income" blocks). These actions must
// not advance turn order.
type LogPreIncomeAction struct {
	Action game.Action
}

func (a *LogPreIncomeAction) GetType() game.ActionType {
	if a == nil || a.Action == nil {
		return game.ActionSpecialAction
	}
	return a.Action.GetType()
}

func (a *LogPreIncomeAction) GetPlayerID() string {
	if a == nil || a.Action == nil {
		return ""
	}
	return a.Action.GetPlayerID()
}

func (a *LogPreIncomeAction) Validate(gs *game.GameState) error {
	if a == nil || a.Action == nil {
		return fmt.Errorf("nil pre-income action")
	}
	return a.Action.Validate(gs)
}

func (a *LogPreIncomeAction) Execute(gs *game.GameState) error {
	if a == nil || a.Action == nil {
		return fmt.Errorf("nil pre-income action")
	}
	// Pre-income actions occur outside normal turn order; suppress any implicit NextTurn.
	prev := gs.SuppressTurnAdvance
	gs.SuppressTurnAdvance = true
	defer func() { gs.SuppressTurnAdvance = prev }()
	return a.Action.Execute(gs)
}

// LogPostIncomeAction marks an action that occurs during the round's income phase
// after normal income has been granted but before the action phase begins.
// These actions must not advance turn order.
type LogPostIncomeAction struct {
	Action game.Action
}

func (a *LogPostIncomeAction) GetType() game.ActionType {
	if a == nil || a.Action == nil {
		return game.ActionSpecialAction
	}
	return a.Action.GetType()
}

func (a *LogPostIncomeAction) GetPlayerID() string {
	if a == nil || a.Action == nil {
		return ""
	}
	return a.Action.GetPlayerID()
}

func (a *LogPostIncomeAction) Validate(gs *game.GameState) error {
	if a == nil || a.Action == nil {
		return fmt.Errorf("nil post-income action")
	}
	return a.Action.Validate(gs)
}

func (a *LogPostIncomeAction) Execute(gs *game.GameState) error {
	if a == nil || a.Action == nil {
		return fmt.Errorf("nil post-income action")
	}
	// Post-income actions occur outside normal turn order; suppress any implicit NextTurn.
	prev := gs.SuppressTurnAdvance
	gs.SuppressTurnAdvance = true
	defer func() { gs.SuppressTurnAdvance = prev }()
	return a.Action.Execute(gs)
}

func isReplayAuxiliaryOnlyAction(action game.Action) bool {
	switch action.(type) {
	case *LogPreIncomeAction:
		if v, ok := action.(*LogPreIncomeAction); ok && v != nil {
			return isReplayAuxiliaryOnlyAction(v.Action)
		}
		return true
	case *LogPostIncomeAction:
		if v, ok := action.(*LogPostIncomeAction); ok && v != nil {
			return isReplayAuxiliaryOnlyAction(v.Action)
		}
		return true
	case *LogConversionAction,
		*LogBurnAction,
		*LogFavorTileAction,
		*LogTownAction,
		*LogCultistAdvanceAction,
		*LogHalflingsSpadeAction,
		*LogAcceptLeechAction,
		*LogDeclineLeechAction,
		*LogBonusCardSelectionAction:
		return true
	}

	// Decline leech is also a reaction/auxiliary action and not a turn main action.
	if action.GetType() == game.ActionDeclinePowerLeech {
		return true
	}
	return false
}

func compoundHasMainAction(actions []game.Action) bool {
	for _, action := range actions {
		if action == nil {
			continue
		}
		if !isReplayAuxiliaryOnlyAction(action) {
			return true
		}
	}
	return false
}

func isReplayAffordabilityError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not enough resources") ||
		strings.Contains(msg, "insufficient resources") ||
		strings.Contains(msg, "not enough workers") ||
		strings.Contains(msg, "cannot afford")
}

func isReplayPreExecutableAction(action game.Action) bool {
	// Deprecated: use isReplayPreExecutableActionForError for selective retry.
	switch action.(type) {
	case *LogBurnAction:
		return true
	default:
		return false
	}
}

type replayInsufficientResources struct {
	needCoins, needWorkers, needPriests, needPower int
	haveCoins, haveWorkers, havePriests, havePower int
}

func parseReplayInsufficientResources(err error) (replayInsufficientResources, bool) {
	if err == nil {
		return replayInsufficientResources{}, false
	}
	var r replayInsufficientResources
	n, _ := fmt.Sscanf(
		strings.ToLower(err.Error()),
		"insufficient resources: need (coins:%d, workers:%d, priests:%d, power:%d), have (coins:%d, workers:%d, priests:%d, power:%d)",
		&r.needCoins, &r.needWorkers, &r.needPriests, &r.needPower,
		&r.haveCoins, &r.haveWorkers, &r.havePriests, &r.havePower,
	)
	return r, n == 8
}

func parseReplayNotEnoughResources(err error) (replayInsufficientResources, bool) {
	if err == nil {
		return replayInsufficientResources{}, false
	}
	// Example:
	// "not enough resources for dwelling: need {2 1 0 0}, have &{0 2 0 0xc000...}"
	re := regexp.MustCompile(`not enough resources for [^:]+: need \{(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\}, have &\{(\d+)\s+(\d+)\s+(\d+)\s+`)
	m := re.FindStringSubmatch(strings.ToLower(err.Error()))
	if len(m) != 8 {
		return replayInsufficientResources{}, false
	}
	toInt := func(s string) int {
		n, _ := strconv.Atoi(s)
		return n
	}
	return replayInsufficientResources{
		needCoins:   toInt(m[1]),
		needWorkers: toInt(m[2]),
		needPriests: toInt(m[3]),
		needPower:   toInt(m[4]),
		haveCoins:   toInt(m[5]),
		haveWorkers: toInt(m[6]),
		havePriests: toInt(m[7]),
		havePower:   0, // not present (pointer in output)
	}, true
}

func isReplayPreExecutableActionForError(action game.Action, err error) bool {
	if action == nil || err == nil {
		return false
	}

	need, ok := parseReplayInsufficientResources(err)
	if !ok {
		need, ok = parseReplayNotEnoughResources(err)
		if !ok {
			// Unknown affordability error shape: only allow burns (they can only help power availability).
			_, isBurn := action.(*LogBurnAction)
			return isBurn
		}
	}

	defCoins := need.needCoins - need.haveCoins
	defWorkers := need.needWorkers - need.haveWorkers
	defPriests := need.needPriests - need.havePriests
	defPower := need.needPower - need.havePower

	switch v := action.(type) {
	case *LogBurnAction:
		// Burns are free actions and can only increase spendable power (Bowl III).
		// Snellman sometimes logs them late in the row even when they fund earlier steps.
		return true
	case *LogConversionAction:
		// Only pre-execute conversions that produce a resource we are short on.
		if defCoins > 0 && v.Reward[models.ResourceCoin] > 0 {
			return true
		}
		if defWorkers > 0 && v.Reward[models.ResourceWorker] > 0 {
			return true
		}
		if defPriests > 0 && v.Reward[models.ResourcePriest] > 0 {
			return true
		}
		if defPower > 0 && v.Reward[models.ResourcePower] > 0 {
			return true
		}
		return false
	default:
		return false
	}
}

// Execute applies the action to the game state.
func (a *LogCompoundAction) Execute(gs *game.GameState) error {
	if len(a.Actions) == 0 {
		return fmt.Errorf("empty compound action")
	}
	hasMain := compoundHasMainAction(a.Actions)

	// Suppress turn advancement during sub-actions to prevent multiple advances
	// (e.g. Transform calls NextTurn, then Build calls NextTurn)
	prevSuppress := gs.SuppressTurnAdvance
	gs.SuppressTurnAdvance = true
	defer func() {
		gs.SuppressTurnAdvance = prevSuppress
		// Advance turn once at the end of the compound action if it contains a legal main action.
		// Reaction-only compound rows (e.g. "+AIR. Leech 2 from witches") must not advance turn.
		if hasMain && !prevSuppress {
			gs.NextTurn()
		}
	}()

	preExecuted := make(map[int]bool, len(a.Actions))

	for i, action := range a.Actions {
		if preExecuted[i] {
			continue
		}
		if err := action.Execute(gs); err != nil {
			// Snellman rows often place resource conversions at the end of the
			// compound token even when they fund an earlier build/upgrade.
			// If an affordability check fails, opportunistically run trailing
			// burn/convert steps once and retry the failed action.
			if !isReplayAffordabilityError(err) {
				return err
			}

			replayed := false
			for j := i + 1; j < len(a.Actions); j++ {
				if preExecuted[j] || !isReplayPreExecutableActionForError(a.Actions[j], err) {
					continue
				}
				if execErr := a.Actions[j].Execute(gs); execErr != nil {
					return execErr
				}
				preExecuted[j] = true
				replayed = true
			}
			if !replayed {
				return err
			}
			if retryErr := action.Execute(gs); retryErr != nil {
				return retryErr
			}
		}
	}
	return nil
}

// LogTownAction is a log-only representation of founding a town
type LogTownAction struct {
	PlayerID string
	VP       int
}

// GetType returns the action type.
func (a *LogTownAction) GetType() game.ActionType { return game.ActionSpecialAction } // Placeholder
// GetPlayerID returns the player ID.
func (a *LogTownAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogTownAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogTownAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	tileType, err := GetTownTileFromVP(a.VP)
	if err != nil {
		return err
	}

	// Select the town tile
	// This assumes PendingTownFormations was populated by the previous action (Build/Upgrade)
	if err := gs.SelectTownTile(a.PlayerID, tileType); err != nil {
		return fmt.Errorf("failed to select town tile: %w", err)
	}

	return nil
}

func (i RoundStartItem) isLogItem() {}

// LogBonusCardSelectionAction is a log-only representation of selecting a bonus card
type LogBonusCardSelectionAction struct {
	PlayerID  string
	BonusCard string // e.g. "BON1"
}

// GetType returns the action type.
func (a *LogBonusCardSelectionAction) GetType() game.ActionType { return game.ActionSpecialAction } // Placeholder
// GetPlayerID returns the player ID.
func (a *LogBonusCardSelectionAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogBonusCardSelectionAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogBonusCardSelectionAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	cardType := ParseBonusCardCode(a.BonusCard)
	if cardType == game.BonusCardUnknown {
		return fmt.Errorf("unknown bonus card code: %s", a.BonusCard)
	}

	// Ensure card is available (hack for replay if missing)
	if !gs.BonusCards.IsAvailable(cardType) {
		gs.BonusCards.Available[cardType] = 0
	}

	if _, err := gs.BonusCards.TakeBonusCard(a.PlayerID, cardType); err != nil {
		return fmt.Errorf("failed to take bonus card: %w", err)
	}

	return nil
}

// ParseBonusCardCode converts a code to a BonusCardType.
func ParseBonusCardCode(code string) game.BonusCardType {
	switch code {
	case "BON-SPD":
		return game.BonusCardSpade
	case "BON-4C":
		return game.BonusCardCultAdvance
	case "BON-6C":
		return game.BonusCard6Coins
	case "BON-SHIP":
		return game.BonusCardShipping
	case "BON-WP":
		return game.BonusCardWorkerPower
	case "BON-TP":
		return game.BonusCardTradingHouseVP
	case "BON-BB":
		return game.BonusCardStrongholdSanctuary
	case "BON-P":
		return game.BonusCardPriest
	case "BON-DW":
		return game.BonusCardDwellingVP
	case "BON-SHIP-VP":
		return game.BonusCardShippingVP
	}
	return game.BonusCardUnknown
}

// LogHalflingsSpadeAction represents Halflings Stronghold 3 spades for transform
type LogHalflingsSpadeAction struct {
	PlayerID        string
	TransformCoords []string
	TargetTerrains  []string // Target terrain for each transform (e.g., "desert", "plains")
}

// GetType returns the action type.
func (a *LogHalflingsSpadeAction) GetType() game.ActionType { return game.ActionSpecialAction }

// GetPlayerID returns the player ID.
func (a *LogHalflingsSpadeAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogHalflingsSpadeAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the Halflings stronghold spades transformations.
func (a *LogHalflingsSpadeAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Apply each transform using ApplyHalflingsSpadeAction
	for i, coordStr := range a.TransformCoords {
		hex, err := ConvertLogCoordToAxial(coordStr)
		if err != nil {
			return err
		}

		// Get target terrain from stored terrains, or default to home terrain
		var targetTerrain models.TerrainType
		if i < len(a.TargetTerrains) && a.TargetTerrains[i] != "" {
			targetTerrain = getTerrainTypeFromName(a.TargetTerrains[i])
		} else {
			targetTerrain = player.Faction.GetHomeTerrain()
		}

		action := &game.ApplyHalflingsSpadeAction{
			BaseAction:    game.BaseAction{Type: game.ActionApplyHalflingsSpade, PlayerID: a.PlayerID},
			TargetHex:     hex,
			TargetTerrain: targetTerrain,
		}
		if err := action.Execute(gs); err != nil {
			return err
		}
	}

	// For BGA log replay, just clear the pending state directly
	// The formal SkipHalflingsDwellingAction checks spade count, but BGA logs
	// may use fewer hexes (e.g., 2 hexes using 3 spades total)
	gs.PendingHalflingsSpades = nil
	return nil
}

// getTerrainTypeFromName converts a terrain name string to TerrainType
func getTerrainTypeFromName(name string) models.TerrainType {
	switch strings.ToLower(name) {
	case "plains":
		return models.TerrainPlains
	case "swamp":
		return models.TerrainSwamp
	case "lakes", "lake":
		return models.TerrainLake
	case "forest":
		return models.TerrainForest
	case "mountains", "mountain":
		return models.TerrainMountain
	case "wasteland":
		return models.TerrainWasteland
	case "desert":
		return models.TerrainDesert
	}
	return models.TerrainPlains // Default to plains
}

// LogCultistAdvanceAction represents the Cultists' faction ability to advance on a cult track
type LogCultistAdvanceAction struct {
	PlayerID string
	Track    game.CultTrack
}

// GetType returns the action type.
func (a *LogCultistAdvanceAction) GetType() game.ActionType {
	return game.ActionSelectCultistsCultTrack
}

// GetPlayerID returns the player ID.
func (a *LogCultistAdvanceAction) GetPlayerID() string { return a.PlayerID }

// Validate checks if the action is valid.
func (a *LogCultistAdvanceAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogCultistAdvanceAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	// Cultists ability: Advance 1 step on the chosen track
	if _, err := gs.AdvanceCultTrack(a.PlayerID, a.Track, 1); err != nil {
		return fmt.Errorf("failed to advance cult track: %w", err)
	}

	// Note: In the real game, this consumes a pending permission/state.
	// For log replay, we just apply the effect.
	return nil
}
