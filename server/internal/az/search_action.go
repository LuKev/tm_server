package az

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/game/board"
	"github.com/lukev/tm_server/internal/models"
)

// ActionSchemaVersion changes whenever SearchAction's semantic encoding changes.
const ActionSchemaVersion = 1

// SearchAction is the stable, player-relative action representation persisted
// in trajectories. Fields are interpreted by Kind; the acting player is stored
// on the position record rather than duplicated in every candidate.
type SearchAction struct {
	Kind       game.ActionType        `json:"kind"`
	Hexes      []board.Hex            `json:"hexes,omitempty"`
	Terrain    models.TerrainType     `json:"terrain,omitempty"`
	Building   models.BuildingType    `json:"building,omitempty"`
	Track      game.CultTrack         `json:"track,omitempty"`
	Tracks     []game.CultTrack       `json:"tracks,omitempty"`
	Card       game.BonusCardType     `json:"card,omitempty"`
	FavorTile  game.FavorTileType     `json:"favor_tile,omitempty"`
	TownTile   models.TownTileType    `json:"town_tile,omitempty"`
	Faction    models.FactionType     `json:"faction,omitempty"`
	Power      game.PowerActionType   `json:"power,omitempty"`
	Special    game.SpecialActionType `json:"special,omitempty"`
	Conversion game.ConversionType    `json:"conversion,omitempty"`
	Amount     int                    `json:"amount,omitempty"`
	Amount2    int                    `json:"amount_2,omitempty"`
	Build      bool                   `json:"build,omitempty"`
	UseSkip    bool                   `json:"use_skip,omitempty"`
	Subactions []SearchAction         `json:"subactions,omitempty"`
}

// Key returns a deterministic semantic key suitable for sorting and equality.
func (a SearchAction) Key() string {
	a = a.normalized()
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// GameAction reconstructs the authoritative engine action for playerID.
func (a SearchAction) GameAction(playerID string) (game.Action, error) {
	a = a.normalized()
	hex := func(i int) (board.Hex, error) {
		if i < 0 || i >= len(a.Hexes) {
			return board.Hex{}, fmt.Errorf("action %d requires hex %d", a.Kind, i)
		}
		return a.Hexes[i], nil
	}

	switch a.Kind {
	case game.ActionTransformAndBuild:
		h, err := hex(0)
		if err != nil {
			return nil, err
		}
		out := game.NewTransformAndBuildAction(playerID, h, a.Build, a.Terrain)
		out.UseSkip = a.UseSkip
		return out, nil
	case game.ActionUpgradeBuilding:
		h, err := hex(0)
		if err != nil {
			return nil, err
		}
		return game.NewUpgradeBuildingAction(playerID, h, a.Building), nil
	case game.ActionAdvanceShipping:
		return game.NewAdvanceShippingAction(playerID), nil
	case game.ActionAdvanceDigging:
		return game.NewAdvanceDiggingAction(playerID), nil
	case game.ActionSendPriestToCult:
		return &game.SendPriestToCultAction{
			BaseAction:    game.BaseAction{Type: game.ActionSendPriestToCult, PlayerID: playerID},
			Track:         a.Track,
			SpacesToClimb: a.Amount,
		}, nil
	case game.ActionPowerAction:
		var out *game.PowerAction
		switch a.Power {
		case game.PowerActionBridge:
			h1, err := hex(0)
			if err != nil {
				return nil, err
			}
			h2, err := hex(1)
			if err != nil {
				return nil, err
			}
			out = game.NewPowerActionWithBridge(playerID, h1, h2)
		case game.PowerActionSpade1, game.PowerActionSpade2:
			h, err := hex(0)
			if err != nil {
				return nil, err
			}
			out = game.NewPowerActionWithTransform(playerID, a.Power, h, a.Build)
			out.UseSkip = a.UseSkip
		default:
			out = game.NewPowerAction(playerID, a.Power)
		}
		return out, nil
	case game.ActionSpecialAction:
		return a.specialGameAction(playerID)
	case game.ActionPass:
		if a.Card == game.BonusCardUnknown {
			return game.NewPassAction(playerID, nil), nil
		}
		card := a.Card
		return game.NewPassAction(playerID, &card), nil
	case game.ActionSetupDwelling:
		h, err := hex(0)
		if err != nil {
			return nil, err
		}
		return game.NewSetupDwellingAction(playerID, h), nil
	case game.ActionUseCultSpade:
		h, err := hex(0)
		if err != nil {
			return nil, err
		}
		return game.NewUseCultSpadeActionWithTerrain(playerID, h, a.Terrain), nil
	case game.ActionAcceptPowerLeech:
		return game.NewAcceptPowerLeechAmountAction(playerID, a.Amount, a.Amount2), nil
	case game.ActionDeclinePowerLeech:
		return game.NewDeclinePowerLeechAction(playerID, a.Amount), nil
	case game.ActionSelectFavorTile:
		return &game.SelectFavorTileAction{BaseAction: game.BaseAction{Type: game.ActionSelectFavorTile, PlayerID: playerID}, TileType: a.FavorTile}, nil
	case game.ActionApplyHalflingsSpade:
		h, err := hex(0)
		if err != nil {
			return nil, err
		}
		return &game.ApplyHalflingsSpadeAction{BaseAction: game.BaseAction{Type: game.ActionApplyHalflingsSpade, PlayerID: playerID}, TargetHex: h, TargetTerrain: a.Terrain}, nil
	case game.ActionBuildHalflingsDwelling:
		h, err := hex(0)
		if err != nil {
			return nil, err
		}
		return &game.BuildHalflingsDwellingAction{BaseAction: game.BaseAction{Type: game.ActionBuildHalflingsDwelling, PlayerID: playerID}, TargetHex: h}, nil
	case game.ActionSkipHalflingsDwelling:
		return &game.SkipHalflingsDwellingAction{BaseAction: game.BaseAction{Type: game.ActionSkipHalflingsDwelling, PlayerID: playerID}}, nil
	case game.ActionUseDarklingsPriestOrdination:
		return &game.UseDarklingsPriestOrdinationAction{BaseAction: game.BaseAction{Type: game.ActionUseDarklingsPriestOrdination, PlayerID: playerID}, WorkersToConvert: a.Amount}, nil
	case game.ActionSelectCultistsCultTrack:
		return game.NewSelectCultistsCultTrackAction(playerID, a.Track), nil
	case game.ActionSelectFaction:
		return &game.SelectFactionAction{PlayerID: playerID, FactionType: a.Faction}, nil
	case game.ActionSetupBonusCard:
		return &game.SetupBonusCardAction{BaseAction: game.BaseAction{Type: game.ActionSetupBonusCard, PlayerID: playerID}, BonusCard: a.Card}, nil
	case game.ActionSelectTownTile:
		var anchor *board.Hex
		if len(a.Hexes) > 0 {
			h := a.Hexes[0]
			anchor = &h
		}
		return &game.SelectTownTileAction{BaseAction: game.BaseAction{Type: game.ActionSelectTownTile, PlayerID: playerID}, TileType: a.TownTile, AnchorHex: anchor}, nil
	case game.ActionSelectTownCultTop:
		return &game.SelectTownCultTopAction{BaseAction: game.BaseAction{Type: game.ActionSelectTownCultTop, PlayerID: playerID}, Tracks: append([]game.CultTrack(nil), a.Tracks...)}, nil
	case game.ActionDiscardPendingSpade:
		return game.NewDiscardPendingSpadeAction(playerID, a.Amount), nil
	case game.ActionConversion:
		return &game.ConversionAction{BaseAction: game.BaseAction{Type: game.ActionConversion, PlayerID: playerID}, ConversionType: a.Conversion, Amount: a.Amount}, nil
	case game.ActionBurnPower:
		return &game.BurnPowerAction{BaseAction: game.BaseAction{Type: game.ActionBurnPower, PlayerID: playerID}, Amount: a.Amount}, nil
	case game.ActionEngineersBridge:
		h1, err := hex(0)
		if err != nil {
			return nil, err
		}
		h2, err := hex(1)
		if err != nil {
			return nil, err
		}
		return game.NewEngineersBridgeAction(playerID, h1, h2), nil
	default:
		return nil, fmt.Errorf("action kind %d is outside AlphaZero schema v%d", a.Kind, ActionSchemaVersion)
	}
}

func (a SearchAction) normalized() SearchAction {
	a.Hexes = append([]board.Hex(nil), a.Hexes...)
	a.Tracks = append([]game.CultTrack(nil), a.Tracks...)
	a.Subactions = append([]SearchAction(nil), a.Subactions...)
	if a.Kind == game.ActionSelectTownCultTop {
		sort.Slice(a.Tracks, func(i, j int) bool { return a.Tracks[i] < a.Tracks[j] })
	}
	if (a.Kind == game.ActionEngineersBridge || (a.Kind == game.ActionPowerAction && a.Power == game.PowerActionBridge)) && len(a.Hexes) == 2 {
		if a.Hexes[1].R < a.Hexes[0].R || (a.Hexes[1].R == a.Hexes[0].R && a.Hexes[1].Q < a.Hexes[0].Q) {
			a.Hexes[0], a.Hexes[1] = a.Hexes[1], a.Hexes[0]
		}
	}
	for i := range a.Subactions {
		a.Subactions[i] = a.Subactions[i].normalized()
	}
	return a
}

func (a SearchAction) specialGameAction(playerID string) (game.Action, error) {
	firstHex := func() (board.Hex, error) {
		if len(a.Hexes) == 0 {
			return board.Hex{}, fmt.Errorf("special action %d requires a hex", a.Special)
		}
		return a.Hexes[0], nil
	}
	switch a.Special {
	case game.SpecialActionAurenCultAdvance:
		return game.NewAurenCultAdvanceAction(playerID, a.Track), nil
	case game.SpecialActionWitchesRide:
		h, err := firstHex()
		if err != nil {
			return nil, err
		}
		return game.NewWitchesRideAction(playerID, h), nil
	case game.SpecialActionSwarmlingsUpgrade:
		h, err := firstHex()
		if err != nil {
			return nil, err
		}
		return game.NewSwarmlingsUpgradeAction(playerID, h), nil
	case game.SpecialActionChaosMagiciansDoubleTurn:
		if len(a.Subactions) == 0 {
			return game.NewChaosMagiciansDoubleTurnAction(playerID, nil, nil), nil
		}
		if len(a.Subactions) != 2 {
			return nil, fmt.Errorf("chaos double turn requires zero or two subactions")
		}
		first, err := a.Subactions[0].GameAction(playerID)
		if err != nil {
			return nil, err
		}
		second, err := a.Subactions[1].GameAction(playerID)
		if err != nil {
			return nil, err
		}
		return game.NewChaosMagiciansDoubleTurnAction(playerID, first, second), nil
	case game.SpecialActionGiantsTransform:
		h, err := firstHex()
		if err != nil {
			return nil, err
		}
		return game.NewGiantsTransformAction(playerID, h, a.Build), nil
	case game.SpecialActionNomadsSandstorm:
		h, err := firstHex()
		if err != nil {
			return nil, err
		}
		return game.NewNomadsSandstormAction(playerID, h, a.Build), nil
	case game.SpecialActionWater2CultAdvance:
		return game.NewWater2CultAdvanceAction(playerID, a.Track), nil
	case game.SpecialActionBonusCardSpade:
		h, err := firstHex()
		if err != nil {
			return nil, err
		}
		out := game.NewBonusCardSpadeAction(playerID, h, a.Build, a.Terrain)
		out.UseSkip = a.UseSkip
		return out, nil
	case game.SpecialActionBonusCardCultAdvance:
		return game.NewBonusCardCultAction(playerID, a.Track), nil
	case game.SpecialActionMermaidsRiverTown:
		h, err := firstHex()
		if err != nil {
			return nil, err
		}
		return game.NewMermaidsRiverTownAction(playerID, h), nil
	default:
		return nil, fmt.Errorf("special action %d is outside base-game schema", a.Special)
	}
}

// SearchActionFromGameAction converts a supported authoritative action back to
// its stable player-relative representation.
func SearchActionFromGameAction(action game.Action) (SearchAction, error) {
	if action == nil {
		return SearchAction{}, fmt.Errorf("action is nil")
	}
	switch typed := action.(type) {
	case *game.TransformAndBuildAction:
		return SearchAction{Kind: game.ActionTransformAndBuild, Hexes: []board.Hex{typed.TargetHex}, Terrain: typed.TargetTerrain, Build: typed.BuildDwelling, UseSkip: typed.UseSkip}, nil
	case *game.UpgradeBuildingAction:
		return SearchAction{Kind: game.ActionUpgradeBuilding, Hexes: []board.Hex{typed.TargetHex}, Building: typed.NewBuildingType}, nil
	case *game.AdvanceShippingAction:
		return SearchAction{Kind: game.ActionAdvanceShipping}, nil
	case *game.AdvanceDiggingAction:
		return SearchAction{Kind: game.ActionAdvanceDigging}, nil
	case *game.SendPriestToCultAction:
		return SearchAction{Kind: game.ActionSendPriestToCult, Track: typed.Track, Amount: typed.SpacesToClimb}, nil
	case *game.PowerAction:
		out := SearchAction{Kind: game.ActionPowerAction, Power: typed.ActionType, Build: typed.BuildDwelling, UseSkip: typed.UseSkip}
		if typed.TargetHex != nil {
			out.Hexes = append(out.Hexes, *typed.TargetHex)
		}
		if typed.BridgeHex1 != nil && typed.BridgeHex2 != nil {
			out.Hexes = []board.Hex{*typed.BridgeHex1, *typed.BridgeHex2}
		}
		return out.normalized(), nil
	case *game.SpecialAction:
		out := SearchAction{Kind: game.ActionSpecialAction, Special: typed.ActionType, Build: typed.BuildDwelling, UseSkip: typed.UseSkip, Amount: typed.Amount}
		if typed.TargetHex != nil {
			out.Hexes = append(out.Hexes, *typed.TargetHex)
		}
		if typed.UpgradeHex != nil && len(out.Hexes) == 0 {
			out.Hexes = append(out.Hexes, *typed.UpgradeHex)
		}
		if typed.TargetTerrain != nil {
			out.Terrain = *typed.TargetTerrain
		}
		if typed.CultTrack != nil {
			out.Track = *typed.CultTrack
		}
		if typed.ActionType == game.SpecialActionChaosMagiciansDoubleTurn && typed.FirstAction != nil && typed.SecondAction != nil {
			first, err := SearchActionFromGameAction(typed.FirstAction)
			if err != nil {
				return SearchAction{}, err
			}
			second, err := SearchActionFromGameAction(typed.SecondAction)
			if err != nil {
				return SearchAction{}, err
			}
			out.Subactions = []SearchAction{first, second}
		}
		return out, nil
	case *game.PassAction:
		card := game.BonusCardUnknown
		if typed.BonusCard != nil {
			card = *typed.BonusCard
		}
		return SearchAction{Kind: game.ActionPass, Card: card}, nil
	case *game.SetupDwellingAction:
		return SearchAction{Kind: game.ActionSetupDwelling, Hexes: []board.Hex{typed.Hex}}, nil
	case *game.UseCultSpadeAction:
		return SearchAction{Kind: game.ActionUseCultSpade, Hexes: []board.Hex{typed.TargetHex}, Terrain: typed.TargetTerrain}, nil
	case *game.AcceptPowerLeechAction:
		return SearchAction{Kind: game.ActionAcceptPowerLeech, Amount: typed.OfferIndex, Amount2: typed.Amount}, nil
	case *game.DeclinePowerLeechAction:
		return SearchAction{Kind: game.ActionDeclinePowerLeech, Amount: typed.OfferIndex}, nil
	case *game.SelectFavorTileAction:
		return SearchAction{Kind: game.ActionSelectFavorTile, FavorTile: typed.TileType}, nil
	case *game.ApplyHalflingsSpadeAction:
		return SearchAction{Kind: game.ActionApplyHalflingsSpade, Hexes: []board.Hex{typed.TargetHex}, Terrain: typed.TargetTerrain}, nil
	case *game.BuildHalflingsDwellingAction:
		return SearchAction{Kind: game.ActionBuildHalflingsDwelling, Hexes: []board.Hex{typed.TargetHex}}, nil
	case *game.SkipHalflingsDwellingAction:
		return SearchAction{Kind: game.ActionSkipHalflingsDwelling}, nil
	case *game.UseDarklingsPriestOrdinationAction:
		return SearchAction{Kind: game.ActionUseDarklingsPriestOrdination, Amount: typed.WorkersToConvert}, nil
	case *game.SelectCultistsCultTrackAction:
		return SearchAction{Kind: game.ActionSelectCultistsCultTrack, Track: typed.CultTrack}, nil
	case *game.SelectFactionAction:
		return SearchAction{Kind: game.ActionSelectFaction, Faction: typed.FactionType}, nil
	case *game.SetupBonusCardAction:
		return SearchAction{Kind: game.ActionSetupBonusCard, Card: typed.BonusCard}, nil
	case *game.SelectTownTileAction:
		out := SearchAction{Kind: game.ActionSelectTownTile, TownTile: typed.TileType}
		if typed.AnchorHex != nil {
			out.Hexes = []board.Hex{*typed.AnchorHex}
		}
		return out, nil
	case *game.SelectTownCultTopAction:
		return (SearchAction{Kind: game.ActionSelectTownCultTop, Tracks: append([]game.CultTrack(nil), typed.Tracks...)}).normalized(), nil
	case *game.DiscardPendingSpadeAction:
		return SearchAction{Kind: game.ActionDiscardPendingSpade, Amount: typed.Count}, nil
	case *game.ConversionAction:
		return SearchAction{Kind: game.ActionConversion, Conversion: typed.ConversionType, Amount: typed.Amount}, nil
	case *game.BurnPowerAction:
		return SearchAction{Kind: game.ActionBurnPower, Amount: typed.Amount}, nil
	case *game.EngineersBridgeAction:
		return (SearchAction{Kind: game.ActionEngineersBridge, Hexes: []board.Hex{typed.BridgeHex1, typed.BridgeHex2}}).normalized(), nil
	default:
		return SearchAction{}, fmt.Errorf("unsupported game action %T", action)
	}
}
