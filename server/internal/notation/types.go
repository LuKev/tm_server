package notation

import (
	"fmt"
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
	PlayerID    string
	PowerAmount int
	VPCost      int
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

	// Try to infer from pending offers
	if offers, ok := gs.PendingLeechOffers[a.PlayerID]; ok && len(offers) > 0 {
		// Accept the first offer
		// Use AcceptLeechOffer helper which handles VP cost and removal
		if err := gs.AcceptLeechOffer(a.PlayerID, 0); err != nil {
			return fmt.Errorf("failed to accept pending leech offer: %w", err)
		}
		return nil
	}

	// Fallback: use explicit amount (or default 1)
	// This handles cases where pending offers might be missing (e.g. partial replay or bug)
	player.Resources.Power.GainPower(a.PowerAmount)
	player.VictoryPoints -= a.VPCost
	return nil
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
func (a *LogPowerAction) Validate(gs *game.GameState) error { return nil }

// Execute applies the action to the game state.
func (a *LogPowerAction) Execute(gs *game.GameState) error {
	player := gs.GetPlayer(a.PlayerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", a.PlayerID)
	}

	actionType := ParsePowerActionCode(a.ActionCode)
	if actionType == game.PowerActionUnknown {
		return fmt.Errorf("unknown power action code: %s", a.ActionCode)
	}

	// Check availability
	// if !gs.PowerActions.IsAvailable(actionType) {
	// 	// For replay, we might want to warn but proceed?
	// 	// Or assume the log is correct and maybe we missed a reset?
	// 	// But strictly, it's an error.
	// 	// fmt.Printf("Warning: Power action %v already used\n", actionType)
	// }

	// Spend power (manual implementation to avoid validation errors if resources mismatch slightly)
	powerCost := game.GetPowerCost(actionType)
	player.Resources.Power.Bowl3 -= powerCost
	player.Resources.Power.Bowl1 += powerCost

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
		// Note: PowerAction.Execute increments this too, but we are bypassing PowerAction.Execute
		// because we are implementing LogPowerAction.Execute directly.
		// Wait, LogPowerAction.Execute does NOT call PowerAction.Execute.
		// It implements the logic itself.
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
	// Logic: Burn X/2 from Bowl 2 to remove from game, move X/2 from Bowl 2 to Bowl 3.
	// Wait, standard burn is: Burn 1 from Bowl 2 to move 1 from Bowl 2 to Bowl 3.
	// So cost is 1 burned per 1 gained.
	// The log says "sacrificed X power ... to get Y power".
	// So we remove X from Bowl 2, and move Y from Bowl 2 to Bowl 3.
	// Actually, usually X = Y.
	// Let's assume Amount is the amount BURNED (removed).
	// And we gain the same amount in Bowl 3 (from Bowl 2).

	// Check bga_parser.go regex:
	// reBurn := regexp.MustCompile(`(.*) sacrificed (\d+) power in Bowl 2 to get (\d+) power from Bowl 2 to Bowl 3`)
	// LogBurnAction has Amount. bga_parser sets Amount to matches[2] (sacrificed).

	burned := a.Amount
	gained := burned // usually 1:1

	if player.Resources.Power.Bowl2 < burned+gained {
		// This might happen if log is out of sync or we missed something.
		// For replay, we might just force it.
		_ = 0 // No-op to avoid empty block lint
	}

	player.Resources.Power.Bowl2 -= burned
	// burned power is gone.

	player.Resources.Power.Bowl2 -= gained
	player.Resources.Power.Bowl3 += gained

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
	cost := factions.Cost{
		Workers: a.Cost[models.ResourceWorker],
		Priests: a.Cost[models.ResourcePriest],
		Coins:   a.Cost[models.ResourceCoin],
	}
	// Power cost in conversion usually means "Spend Power" (Bowl III -> Bowl I)
	// But factions.Cost.Power usually means "Gain Power" or "Spend Power" depending on context.
	// Player.Resources.Spend handles Power as "Spend from Bowl III".
	cost.Power = a.Cost[models.ResourcePower]

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

// Execute applies the action to the game state.
func (a *LogCompoundAction) Execute(gs *game.GameState) error {
	// Suppress turn advancement during sub-actions to prevent multiple advances
	// (e.g. Transform calls NextTurn, then Build calls NextTurn)
	gs.SuppressTurnAdvance = true
	defer func() {
		gs.SuppressTurnAdvance = false
		// Advance turn once at the end of the compound action
		gs.NextTurn()
	}()

	for _, action := range a.Actions {
		if err := action.Execute(gs); err != nil {
			return err
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
