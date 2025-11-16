package replay

import (
	"fmt"
	"strings"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// CompoundAction represents a complete log entry as an ordered sequence of components
type CompoundAction struct {
	Components []ActionComponent
}

// ActionComponent represents any part of a compound action (conversion, main action, or auxiliary)
type ActionComponent interface {
	Execute(gs *game.GameState, playerID string) error
	String() string
}

// ========== CONVERSIONS ==========

// ConversionType identifies the type of resource conversion
type ConversionType int

const (
	ConvBurn ConversionType = iota
	ConvPowerToCoins
	ConvPowerToWorkers
	ConvPowerToPriests
	ConvPriestToWorker
	ConvWorkerToCoin
	ConvVPToCoins    // Alchemists: VP -> Coins (1:1)
	ConvCoinsToVP    // Alchemists: Coins -> VP (2:1)
)

func (ct ConversionType) String() string {
	switch ct {
	case ConvBurn:
		return "Burn"
	case ConvPowerToCoins:
		return "PW→C"
	case ConvPowerToWorkers:
		return "PW→W"
	case ConvPowerToPriests:
		return "PW→P"
	case ConvPriestToWorker:
		return "P→W"
	case ConvWorkerToCoin:
		return "W→C"
	case ConvVPToCoins:
		return "VP→C"
	case ConvCoinsToVP:
		return "C→VP"
	default:
		return "Unknown"
	}
}

// ConversionComponent represents a resource conversion
type ConversionComponent struct {
	Type   ConversionType
	Amount int
	From   string
	To     string
}

func (c *ConversionComponent) Execute(gs *game.GameState, playerID string) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player %s not found", playerID)
	}

	switch c.Type {
	case ConvBurn:
		return player.Resources.BurnPower(c.Amount)
	case ConvPowerToCoins:
		return player.Resources.ConvertPowerToCoins(c.Amount)
	case ConvPowerToWorkers:
		return player.Resources.ConvertPowerToWorkers(c.Amount)
	case ConvPowerToPriests:
		return player.Resources.ConvertPowerToPriests(c.Amount)
	case ConvPriestToWorker:
		return player.Resources.ConvertPriestToWorker(c.Amount)
	case ConvWorkerToCoin:
		return player.Resources.ConvertWorkerToCoin(c.Amount)
	case ConvVPToCoins:
		// Alchemists: VP -> Coins (1:1)
		// c.Amount is target coins, but function expects source VP
		// Since ratio is 1:1, source VP = target coins
		return gs.AlchemistsConvertVPToCoins(playerID, c.Amount)
	case ConvCoinsToVP:
		// Alchemists: Coins -> VP (2:1, i.e., 2 coins = 1 VP)
		// c.Amount is target VP, but function expects source coins
		// source coins = target VP * 2
		return gs.AlchemistsConvertCoinsToVP(playerID, c.Amount*2)
	default:
		return fmt.Errorf("unknown conversion type: %v", c.Type)
	}
}

func (c *ConversionComponent) String() string {
	if c.Type == ConvBurn {
		return fmt.Sprintf("Burn(%d)", c.Amount)
	}
	return fmt.Sprintf("Convert(%s: %d %s→%s)", c.Type, c.Amount, c.From, c.To)
}

// ========== MAIN ACTIONS ==========

// MainActionComponent represents the core game action
type MainActionComponent struct {
	Action    game.Action
	Modifiers []ActionModifier
}

func (m *MainActionComponent) Execute(gs *game.GameState, playerID string) error {
	// Apply modifiers first (if any)
	for _, mod := range m.Modifiers {
		if err := mod.Apply(gs, playerID, m.Action); err != nil {
			return fmt.Errorf("failed to apply modifier: %w", err)
		}
	}

	// Execute main action
	if err := m.Action.Validate(gs); err != nil {
		return fmt.Errorf("action validation failed: %w", err)
	}
	return m.Action.Execute(gs)
}

func (m *MainActionComponent) String() string {
	modStr := ""
	if len(m.Modifiers) > 0 {
		modStr = fmt.Sprintf(" [%d modifiers]", len(m.Modifiers))
	}
	return fmt.Sprintf("MainAction(%s%s)", m.Action.GetType(), modStr)
}

// ========== POWER ACTION FOR FREE SPADES ==========

// PowerActionFreeSpades is a component that executes a spade power action (ACT5/ACT6)
// and grants free spades via PendingSpades. This is used when the power action is followed
// by transform/build at potentially different hexes.
type PowerActionFreeSpades struct {
	PowerActionType game.PowerActionType
	Burned          int // Amount of power burned before using action
}

func (p *PowerActionFreeSpades) Execute(gs *game.GameState, playerID string) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player %s not found", playerID)
	}

	// Burn power if needed
	if p.Burned > 0 {
		if err := player.Resources.BurnPower(p.Burned); err != nil {
			return fmt.Errorf("failed to burn power: %w", err)
		}
	}

	// Pay power cost
	powerCost := game.GetPowerCost(p.PowerActionType)
	if err := player.Resources.Power.SpendPower(powerCost); err != nil {
		return fmt.Errorf("failed to spend power: %w", err)
	}

	// Mark power action as used
	gs.PowerActions.MarkUsed(p.PowerActionType)

	// Grant free spades via PendingSpades
	freeSpades := 1
	if p.PowerActionType == game.PowerActionSpade2 {
		freeSpades = 2
	}

	if gs.PendingSpades == nil {
		gs.PendingSpades = make(map[string]int)
	}
	gs.PendingSpades[playerID] += freeSpades

	// Award VP from scoring tile for power action spades
	// Unlike cult reward spades, power action spades (ACT5/ACT6) count for scoring
	if _, isDarklings := player.Faction.(*factions.Darklings); !isDarklings {
		spadesUsed := player.Faction.GetTerraformSpades(freeSpades)
		for i := 0; i < spadesUsed; i++ {
			gs.AwardActionVP(playerID, game.ScoringActionSpades)
		}

		// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
		if halflings, ok := player.Faction.(*factions.Halflings); ok {
			vpBonus := halflings.GetVPPerSpade() * spadesUsed
			player.VictoryPoints += vpBonus
		}

		// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
		if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
			powerBonus := alchemists.GetPowerPerSpade() * spadesUsed
			if powerBonus > 0 {
				player.Resources.GainPower(powerBonus)
			}
		}
	}

	return nil
}

func (p *PowerActionFreeSpades) String() string {
	if p.Burned > 0 {
		return fmt.Sprintf("PowerActionFreeSpades(%s, burned=%d)", p.PowerActionType, p.Burned)
	}
	return fmt.Sprintf("PowerActionFreeSpades(%s)", p.PowerActionType)
}

// GrantSpadesComponent grants free spades for terraform (from digging level or other sources)
// Used for "dig X. build Y" patterns where X is the number of free spades
type GrantSpadesComponent struct {
	Spades int // Number of free spades to grant
}

func (d *GrantSpadesComponent) Execute(gs *game.GameState, playerID string) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player %s not found", playerID)
	}

	// For Darklings, "dig X" is just notation - they don't get free spades
	// They pay priests directly in TransformAndBuildAction
	if _, isDarklings := player.Faction.(*factions.Darklings); isDarklings {
		return nil
	}

	// Grant free spades for dig advancement
	// Note: VP will be awarded later by TransformAndBuildAction when spades are actually USED
	// We don't award VP here because the transformation might not happen (hex already home terrain)
	if gs.PendingSpades == nil {
		gs.PendingSpades = make(map[string]int)
	}
	gs.PendingSpades[playerID] += d.Spades

	return nil
}

func (d *GrantSpadesComponent) String() string {
	return fmt.Sprintf("GrantSpades(spades=%d)", d.Spades)
}

// ========== ACTION MODIFIERS ==========

// ActionModifier enhances a main action (e.g., power actions that provide free spades)
type ActionModifier interface {
	Apply(gs *game.GameState, playerID string, mainAction game.Action) error
	String() string
}

// PowerActionModifier provides free spades or other benefits from power actions
type PowerActionModifier struct {
	PowerActionType game.PowerActionType
	Burned          int // Amount of power burned before using action
}

func (p *PowerActionModifier) Apply(gs *game.GameState, playerID string, mainAction game.Action) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player %s not found", playerID)
	}

	// Burn power if needed
	if p.Burned > 0 {
		if err := player.Resources.BurnPower(p.Burned); err != nil {
			return fmt.Errorf("failed to burn power: %w", err)
		}
	}

	// Pay power cost
	powerCost := game.GetPowerCost(p.PowerActionType)
	if err := player.Resources.Power.SpendPower(powerCost); err != nil {
		return fmt.Errorf("failed to spend power: %w", err)
	}

	// Mark power action as used
	gs.PowerActions.MarkUsed(p.PowerActionType)

	// For spade actions (ACT5, ACT6), grant free spades via PendingSpades
	// These can be used by TransformTerrainComponent or TransformAndBuildAction
	if p.PowerActionType == game.PowerActionSpade1 || p.PowerActionType == game.PowerActionSpade2 {
		freeSpades := 1
		if p.PowerActionType == game.PowerActionSpade2 {
			freeSpades = 2
		}

		if gs.PendingSpades == nil {
			gs.PendingSpades = make(map[string]int)
		}
		gs.PendingSpades[playerID] += freeSpades
	}

	return nil
}

func (p *PowerActionModifier) String() string {
	if p.Burned > 0 {
		return fmt.Sprintf("PowerAction(%s, burned=%d)", p.PowerActionType, p.Burned)
	}
	return fmt.Sprintf("PowerAction(%s)", p.PowerActionType)
}

// SpecialActionModifier for faction-specific special actions
type SpecialActionModifier struct {
	SpecialActionType string // "ACTW" (Witches), "ACTA" (Auren), etc.
}

func (s *SpecialActionModifier) Apply(gs *game.GameState, playerID string, mainAction game.Action) error {
	// Special actions are typically handled by their own action types
	// This is mainly for documentation and future extensibility
	return nil
}

func (s *SpecialActionModifier) String() string {
	return fmt.Sprintf("SpecialAction(%s)", s.SpecialActionType)
}

// ========== TRANSFORM TERRAIN (for transform-only actions) ==========

// TransformTerrainComponent represents a transform-only action
// This is used for "dig X. transform Y to Z" patterns where the target
// terrain Z is explicitly specified and may not be the player's home terrain
type TransformTerrainComponent struct {
	TargetHex     game.Hex
	TargetTerrain models.TerrainType
}

func (t *TransformTerrainComponent) Execute(gs *game.GameState, playerID string) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player %s not found", playerID)
	}

	mapHex := gs.Map.GetHex(t.TargetHex)
	if mapHex == nil {
		return fmt.Errorf("hex does not exist: %v", t.TargetHex)
	}

	// Check if terrain needs transformation
	if mapHex.Terrain == t.TargetTerrain {
		// Already the target terrain, nothing to do
		return nil
	}

	// Calculate terraform cost
	distance := gs.Map.GetTerrainDistance(mapHex.Terrain, t.TargetTerrain)
	if distance == 0 {
		return fmt.Errorf("terrain distance calculation failed")
	}

	// Get actual spades needed (accounts for faction abilities like Giants who always use 2 spades)
	spadesNeeded := player.Faction.GetTerraformSpades(distance)

	// Check for free spades from power actions (ACT5/ACT6) or cult rewards
	freeSpades := 0
	if gs.PendingSpades != nil && gs.PendingSpades[playerID] > 0 {
		freeSpades = gs.PendingSpades[playerID]
		if freeSpades > spadesNeeded {
			freeSpades = spadesNeeded // Only use what we need
		}
		// Consume free spades
		gs.PendingSpades[playerID] -= freeSpades
		if gs.PendingSpades[playerID] == 0 {
			delete(gs.PendingSpades, playerID)
		}
	}

	remainingSpades := spadesNeeded - freeSpades

	// Validate and pay costs for remaining spades
	if remainingSpades > 0 {
		// Darklings pay priests for terraform (1 priest per spade)
		if darklings, ok := player.Faction.(*factions.Darklings); ok {
			priestCost := darklings.GetTerraformCostInPriests(remainingSpades)
			if player.Resources.Priests < priestCost {
				return fmt.Errorf("not enough priests for terraform: need %d, have %d", priestCost, player.Resources.Priests)
			}
			player.Resources.Priests -= priestCost

			// Award Darklings VP bonus (+2 VP per remaining spade, not free spades)
			vpBonus := darklings.GetTerraformVPBonus(remainingSpades)
			player.VictoryPoints += vpBonus
		} else {
			// Other factions pay workers
			workerCost := player.Faction.GetTerraformCost(remainingSpades)
			if player.Resources.Workers < workerCost {
				return fmt.Errorf("not enough workers for terraform: need %d, have %d", workerCost, player.Resources.Workers)
			}
			player.Resources.Workers -= workerCost
		}
	}

	// Transform terrain to target terrain
	if err := gs.Map.TransformTerrain(t.TargetHex, t.TargetTerrain); err != nil {
		return fmt.Errorf("failed to transform terrain: %w", err)
	}

	// Award VP from scoring tile (per spade PAID FOR, not free spades)
	// Only award VP for spades the player actually paid for (remainingSpades), not cult reward spades
	// Note: Darklings get BOTH their faction bonus AND scoring tile VP for paid spades
	if remainingSpades > 0 {
		spadesUsed := player.Faction.GetTerraformSpades(remainingSpades)
		for i := 0; i < spadesUsed; i++ {
			gs.AwardActionVP(playerID, game.ScoringActionSpades)
		}

		// Award faction-specific spade VP bonus (e.g., Halflings +1 VP per spade)
		// Note: Darklings faction bonus is already awarded above when paying priests
		if halflings, ok := player.Faction.(*factions.Halflings); ok {
			vpBonus := halflings.GetVPPerSpade() * spadesUsed
			player.VictoryPoints += vpBonus
		}

		// Award faction-specific spade power bonus (e.g., Alchemists +2 power per spade after stronghold)
		if alchemists, ok := player.Faction.(*factions.Alchemists); ok {
			powerBonus := alchemists.GetPowerPerSpade() * spadesUsed
			if powerBonus > 0 {
				player.Resources.GainPower(powerBonus)
			}
		}
	}

	return nil
}

func (t *TransformTerrainComponent) String() string {
	return fmt.Sprintf("TransformTerrain(%v → %v)", t.TargetHex, t.TargetTerrain)
}

// ========== AUXILIARY ACTIONS ==========

// AuxiliaryType identifies the type of auxiliary action
type AuxiliaryType int

const (
	AuxFavorTile AuxiliaryType = iota
	AuxTownTile
	AuxConnect // Mermaids river-skip town formation (informational only)
)

func (at AuxiliaryType) String() string {
	switch at {
	case AuxFavorTile:
		return "FavorTile"
	case AuxTownTile:
		return "TownTile"
	case AuxConnect:
		return "Connect"
	default:
		return "Unknown"
	}
}

// AuxiliaryComponent represents auxiliary decisions (favor/town tile selection)
type AuxiliaryComponent struct {
	Type   AuxiliaryType
	Params map[string]string
}

func (a *AuxiliaryComponent) Execute(gs *game.GameState, playerID string) error {
	switch a.Type {
	case AuxFavorTile:
		// Strip the "+" prefix from tile string (e.g., "+FAV11" -> "FAV11")
		tileStr := strings.TrimPrefix(a.Params["tile"], "+")
		tileType, err := ParseFavorTile(tileStr)
		if err != nil {
			return fmt.Errorf("failed to parse favor tile: %w", err)
		}
		// Create and execute SelectFavorTileAction
		action := &game.SelectFavorTileAction{
			BaseAction: game.BaseAction{
				Type:     game.ActionSelectFavorTile,
				PlayerID: playerID,
			},
			TileType: tileType,
		}
		if err := action.Validate(gs); err != nil {
			return err
		}
		return action.Execute(gs)
	case AuxTownTile:
		// Strip the "+" prefix from tile string (e.g., "+TW3" -> "TW3")
		tileStr := strings.TrimPrefix(a.Params["tile"], "+")
		tileType, err := ParseTownTile(tileStr)
		if err != nil {
			return fmt.Errorf("failed to parse town tile: %w", err)
		}
		// Use GameState method directly
		return gs.SelectTownTile(playerID, tileType)
	case AuxConnect:
		// Mermaids river-skip town formation: "connect r16"
		// This triggers a check for town formation using river-skip
		// We need to check all Mermaids buildings to find which ones form a town via this river
		player := gs.GetPlayer(playerID)
		if player == nil {
			return fmt.Errorf("player not found: %s", playerID)
		}

		// Only Mermaids can use river-skip town formation
		if player.Faction.GetType() != models.FactionMermaids {
			return fmt.Errorf("only Mermaids can use river-skip town formation")
		}

		// Check for town formation on all Mermaids buildings
		// Since PendingTownFormations may have been cleared during cleanup,
		// we need to re-check for river-skip towns when explicitly triggered
		for _, mapHex := range gs.Map.Hexes {
			if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
				gs.CheckForTownFormation(playerID, mapHex.Coord)
				// If we found a town, stop checking (avoid duplicates)
				if len(gs.PendingTownFormations[playerID]) > 0 {
					break
				}
			}
		}

		// After checking, there should be at least one pending town formation
		if len(gs.PendingTownFormations[playerID]) == 0 {
			return fmt.Errorf("no valid town formation found for river-skip connection")
		}

		return nil
	default:
		return fmt.Errorf("unknown auxiliary type: %v", a.Type)
	}
}

func (a *AuxiliaryComponent) String() string {
	return fmt.Sprintf("Auxiliary(%s: %s)", a.Type, a.Params["tile"])
}

// ========== DARKLINGS PRIEST ORDINATION ==========

// DarklingsPriestOrdinationComponent represents Darklings converting workers to priests
type DarklingsPriestOrdinationComponent struct {
	WorkersToConvert int
}

func (d *DarklingsPriestOrdinationComponent) Execute(gs *game.GameState, playerID string) error {
	action := &game.UseDarklingsPriestOrdinationAction{
		BaseAction: game.BaseAction{
			Type:     game.ActionUseDarklingsPriestOrdination,
			PlayerID: playerID,
		},
		WorkersToConvert: d.WorkersToConvert,
	}
	if err := action.Validate(gs); err != nil {
		return fmt.Errorf("priest ordination validation failed: %w", err)
	}
	return action.Execute(gs)
}

func (d *DarklingsPriestOrdinationComponent) String() string {
	return fmt.Sprintf("DarklingsPriestOrdination(%dW→%dP)", d.WorkersToConvert, d.WorkersToConvert)
}

// ========== COMPOUND ACTION EXECUTION ==========

// Execute runs all components of the compound action in sequence
func (ca *CompoundAction) Execute(gs *game.GameState, playerID string) error {
	for i, component := range ca.Components {
		if err := component.Execute(gs, playerID); err != nil {
			return fmt.Errorf("failed to execute component %d (%s): %w", i, component.String(), err)
		}
	}

	return nil
}

// String returns a human-readable representation of the compound action
func (ca *CompoundAction) String() string {
	if len(ca.Components) == 0 {
		return "CompoundAction{empty}"
	}
	result := "CompoundAction{\n"
	for i, comp := range ca.Components {
		result += fmt.Sprintf("  %d. %s\n", i+1, comp.String())
	}
	result += "}"
	return result
}
