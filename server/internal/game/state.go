package game

import (
	"fmt"
	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

// State is an alias to the authoritative models.GameState.
// Keeping the alias in this package lets us evolve engine-specific helpers
// without leaking them to external packages.
type State = models.GameState

// GameState represents the complete game state
type GameState struct {
	Map                *TerraMysticaMap
	Players            map[string]*Player
	Round              int
	Phase              GamePhase
	TurnOrder          []string                      // Player IDs in turn order
	CurrentPlayerIndex int                           // Index into TurnOrder
	PassOrder          []string                      // Player IDs in the order they passed (for next round's turn order)
	PowerActions       *PowerActionState             // Tracks which power actions have been used this round
	CultTracks         *CultTrackState               // Tracks all players' cult track positions
	FavorTiles         *FavorTileState               // Tracks available favor tiles and player selections
	BonusCards         *BonusCardState               // Tracks available bonus cards and player selections
	TownTiles               *TownTileState                      // Tracks available town tiles
	ScoringTiles            *ScoringTileState                   // Tracks scoring tiles for each round
	PendingLeechOffers      map[string][]*PowerLeechOffer       // Key: playerID who can accept
	PendingTownFormations   map[string]*PendingTownFormation    // Key: playerID who can form town
	PendingSpades           map[string]int                      // Key: playerID, Value: number of spades to use (from cult rewards)
	PendingCultistsLeech    map[string]*CultistsLeechBonus      // Key: playerID (Cultists), tracks pending cult advance/power bonus
	PendingFavorTileSelection *PendingFavorTileSelection        // Player who needs to select favor tile(s)
}

// PendingTownFormation represents a town that can be formed but awaits tile selection
type PendingTownFormation struct {
	PlayerID        string
	Hexes           []Hex // The connected buildings that form the town
	SkippedRiverHex *Hex  // For Mermaids: the river hex that was skipped to form the town (town tile goes here)
	CanBeDelayed    bool  // For Mermaids: true if town uses river skipping (can be claimed later), false if only land tiles (must claim now)
}

// CultistsLeechBonus tracks Cultists' pending cult advance or power bonus from power leech
type CultistsLeechBonus struct {
	PlayerID       string
	OffersCreated  int  // Number of offers created
	AcceptedCount  int  // Number of offers accepted
	DeclinedCount  int  // Number of offers declined
}

// PendingFavorTileSelection represents a player who needs to select favor tile(s)
// Triggered by: Temple, Sanctuary, or Auren Stronghold
type PendingFavorTileSelection struct {
	PlayerID      string
	Count         int  // Number of tiles to select (1 for most factions, 2 for Chaos Magicians)
	SelectedTiles []FavorTileType // Tiles selected so far
}

// GamePhase represents the current phase of the game
type GamePhase int

const (
	PhaseSetup   GamePhase = iota // Initial game setup
	PhaseIncome                   // Players receive resources
	PhaseAction                   // Players take actions
	PhaseCleanup                  // End-of-round maintenance and scoring
	PhaseEnd                      // Game over (after round 6)
)

// CultTrack represents the four cult tracks
type CultTrack int

const (
	CultFire CultTrack = iota
	CultWater
	CultEarth
	CultAir
)

// Player represents a player in the game
type Player struct {
	ID                      string
	Faction                 factions.Faction
	Resources               *ResourcePool
	ShippingLevel           int
	DiggingLevel            int
	BridgesBuilt            int // Number of bridges built (max 3)
	CultPositions           map[CultTrack]int // Position on each cult track (0-10)
	HasStrongholdAbility    bool // Whether the stronghold special ability is available
	SpecialActionsUsed      map[SpecialActionType]bool // Track which special actions have been used this round
	HasPassed               bool
	VictoryPoints           int
	Keys                    int // Keys for advancing to position 10 on cult tracks
	TownsFormed             int // Number of towns formed
	TownTiles               []TownTileType // Town tiles selected by this player
}

// NewGameState creates a new game state with an initialized map
func NewGameState() *GameState {
	return &GameState{
		Map:                   NewTerraMysticaMap(),
		Players:               make(map[string]*Player),
		Round:                 1,
		Phase:                 PhaseSetup,
		PowerActions:          NewPowerActionState(),
		CultTracks:            NewCultTrackState(),
		FavorTiles:            NewFavorTileState(),
		BonusCards:            NewBonusCardState(),
		TownTiles:             NewTownTileState(),
		ScoringTiles:          NewScoringTileState(),
		PendingLeechOffers:    make(map[string][]*PowerLeechOffer),
		PendingTownFormations: make(map[string]*PendingTownFormation),
	}
}

// AddPlayer adds a player to the game
func (gs *GameState) AddPlayer(playerID string, faction factions.Faction) error {
	if _, exists := gs.Players[playerID]; exists {
		return fmt.Errorf("player already exists: %s", playerID)
	}
	
	// Check if another player already has the same home terrain
	newHomeTerrain := faction.GetHomeTerrain()
	for _, existingPlayer := range gs.Players {
		if existingPlayer.Faction.GetHomeTerrain() == newHomeTerrain {
			return fmt.Errorf("faction %s cannot be added: another player already has home terrain %v (faction %s)", 
				faction.GetType(), newHomeTerrain, existingPlayer.Faction.GetType())
		}
	}

	player := &Player{
		ID:            playerID,
		Faction:       faction,
		Resources:     NewResourcePool(faction.GetStartingResources()),
		ShippingLevel: 0,
		DiggingLevel:  0,
		BridgesBuilt:  0,
		CultPositions: map[CultTrack]int{
			CultFire:  0,
			CultWater: 0,
			CultEarth: 0,
			CultAir:   0,
		},
		HasStrongholdAbility:  false,
		SpecialActionsUsed:    make(map[SpecialActionType]bool),
		HasPassed:             false,
		VictoryPoints:         20, // Starting VP
		Keys:                  0,
		TownsFormed:           0,
		TownTiles:             []TownTileType{},
	}

	gs.Players[playerID] = player
	
	// Initialize cult track positions for this player
	gs.CultTracks.InitializePlayer(playerID)
	
	// Initialize favor tiles for this player
	gs.FavorTiles.InitializePlayer(playerID)
	
	// Initialize bonus cards for this player
	gs.BonusCards.InitializePlayer(playerID)
	
	return nil
}

// GetPlayer returns a player by ID
func (gs *GameState) GetPlayer(playerID string) *Player {
	return gs.Players[playerID]
}

// SelectTownTile allows a player to select a town tile for their pending town formation
func (gs *GameState) SelectTownTile(playerID string, tileType TownTileType) error {
	// Check if player has a pending town formation
	pending, ok := gs.PendingTownFormations[playerID]
	if !ok || pending == nil {
		return fmt.Errorf("no pending town formation for player %s", playerID)
	}
	
	// Check if tile is available
	if !gs.TownTiles.IsAvailable(tileType) {
		return fmt.Errorf("town tile %v is not available", tileType)
	}
	
	// Form the town with the selected tile (and skipped river hex for Mermaids)
	if err := gs.FormTown(playerID, pending.Hexes, tileType, pending.SkippedRiverHex); err != nil {
		return err
	}
	
	// Remove pending town formation
	delete(gs.PendingTownFormations, playerID)
	
	return nil
}

// IsAdjacentToPlayerBuilding checks if a hex is adjacent to any of the player's buildings
// According to Terra Mystica rules, adjacency can be:
// 1. Direct adjacency (shared edge or connected via bridge)
// 2. Indirect adjacency (connected via river navigation with shipping)
func (gs *GameState) IsAdjacentToPlayerBuilding(targetHex Hex, playerID string) bool {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return false
	}

	// Check if player has any buildings at all (for first dwelling placement)
	hasAnyBuilding := false
	for _, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			hasAnyBuilding = true
			break
		}
	}
	
	// If player has no buildings yet, allow placement anywhere (first dwelling)
	if !hasAnyBuilding {
		return true
	}

	// Check adjacency to each of the player's buildings
	for _, mapHex := range gs.Map.Hexes {
		if mapHex.Building != nil && mapHex.Building.PlayerID == playerID {
			buildingHex := mapHex.Coord
			
			// Check direct adjacency (includes bridges)
			if gs.Map.IsDirectlyAdjacent(targetHex, buildingHex) {
				return true
			}
			
			// Check indirect adjacency via shipping (river navigation)
			if player.ShippingLevel > 0 {
				if gs.Map.IsIndirectlyAdjacent(targetHex, buildingHex, player.ShippingLevel) {
					return true
				}
			}
		}
	}

	// TODO: Check for special abilities (Witches flying, Fakirs carpet, Dwarves tunneling)

	return false
}

// TriggerPowerLeech triggers power leech offers for all adjacent players
// This is called when a building is placed or upgraded
// According to Terra Mystica rules, each adjacent player receives ONE offer equal to
// the sum of power values from ALL their buildings adjacent to the new building
// Special: If building player is Cultists, they get cult advance or power bonus based on responses
func (gs *GameState) TriggerPowerLeech(buildingHex Hex, buildingPlayerID string) {
	// Find all adjacent players and calculate total power from their adjacent buildings
	neighbors := buildingHex.Neighbors()
	adjacentPlayerPower := make(map[string]int) // playerID -> total power from their adjacent buildings

	for _, neighbor := range neighbors {
		mapHex := gs.Map.GetHex(neighbor)
		if mapHex != nil && mapHex.Building != nil {
			neighborPlayerID := mapHex.Building.PlayerID
			if neighborPlayerID != buildingPlayerID {
				// Add this building's power value to the player's total
				adjacentPlayerPower[neighborPlayerID] += mapHex.Building.PowerValue
			}
		}
	}

	// Track if any offers were created (for Cultists ability)
	offersCreated := 0
	
	// Create ONE power leech offer per adjacent player based on their total adjacent power
	for neighborPlayerID, totalPower := range adjacentPlayerPower {
		neighborPlayer := gs.GetPlayer(neighborPlayerID)
		if neighborPlayer == nil {
			continue
		}

		// Create offer based on TOTAL power from all adjacent buildings
		offer := NewPowerLeechOffer(totalPower, buildingPlayerID, neighborPlayer.Resources.Power)
		if offer != nil {
			// Store offer for player to accept/decline
			if gs.PendingLeechOffers[neighborPlayerID] == nil {
				gs.PendingLeechOffers[neighborPlayerID] = []*PowerLeechOffer{}
			}
			gs.PendingLeechOffers[neighborPlayerID] = append(gs.PendingLeechOffers[neighborPlayerID], offer)
			offersCreated++
		}
	}
	
	// Cultists special ability: If at least one offer was created, mark for cult advance/power bonus
	// The actual bonus is applied when all offers are resolved (accepted/declined)
	if offersCreated > 0 {
		buildingPlayer := gs.GetPlayer(buildingPlayerID)
		if buildingPlayer != nil {
			if _, ok := buildingPlayer.Faction.(*factions.Cultists); ok {
				// Store that this building triggered Cultists ability
				// We'll resolve it when all leech offers are processed
				if gs.PendingCultistsLeech == nil {
					gs.PendingCultistsLeech = make(map[string]*CultistsLeechBonus)
				}
				gs.PendingCultistsLeech[buildingPlayerID] = &CultistsLeechBonus{
					PlayerID:      buildingPlayerID,
					OffersCreated: offersCreated,
				}
			}
		}
	}
}

// GetPendingLeechOffers returns all pending leech offers for a player
func (gs *GameState) GetPendingLeechOffers(playerID string) []*PowerLeechOffer {
	return gs.PendingLeechOffers[playerID]
}

// HasPendingLeechOffers checks if any player has pending leech offers
func (gs *GameState) HasPendingLeechOffers() bool {
	for _, offers := range gs.PendingLeechOffers {
		if len(offers) > 0 {
			return true
		}
	}
	return false
}

// AcceptLeechOffer allows a player to accept a power leech offer
func (gs *GameState) AcceptLeechOffer(playerID string, offerIndex int) error {
	offers := gs.PendingLeechOffers[playerID]
	if offerIndex < 0 || offerIndex >= len(offers) {
		return fmt.Errorf("invalid offer index: %d", offerIndex)
	}

	offer := offers[offerIndex]
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	// Gain power
	player.Resources.Power.GainPower(offer.Amount)

	// Lose VP
	player.VictoryPoints -= offer.VPCost

	// Remove the offer
	gs.PendingLeechOffers[playerID] = append(offers[:offerIndex], offers[offerIndex+1:]...)

	return nil
}

// DeclineLeechOffer allows a player to decline a power leech offer
func (gs *GameState) DeclineLeechOffer(playerID string, offerIndex int) error {
	offers := gs.PendingLeechOffers[playerID]
	if offerIndex < 0 || offerIndex >= len(offers) {
		return fmt.Errorf("invalid offer index: %d", offerIndex)
	}

	// Simply remove the offer without gaining power or losing VP
	gs.PendingLeechOffers[playerID] = append(offers[:offerIndex], offers[offerIndex+1:]...)

	return nil
}

// ClearPendingLeechOffers clears all pending leech offers for a player
func (gs *GameState) ClearPendingLeechOffers(playerID string) {
	delete(gs.PendingLeechOffers, playerID)
}

// GetCurrentPlayer returns the player whose turn it is
func (gs *GameState) GetCurrentPlayer() *Player {
	if len(gs.TurnOrder) == 0 || gs.CurrentPlayerIndex >= len(gs.TurnOrder) {
		return nil
	}
	return gs.GetPlayer(gs.TurnOrder[gs.CurrentPlayerIndex])
}

// NextTurn advances to the next player's turn
// Returns true if we've completed a full round of turns
func (gs *GameState) NextTurn() bool {
	// Skip players who have passed
	for {
		gs.CurrentPlayerIndex++
		
		// If we've gone through all players, check if everyone has passed
		if gs.CurrentPlayerIndex >= len(gs.TurnOrder) {
			gs.CurrentPlayerIndex = 0
			
			// Check if all players have passed
			allPassed := true
			for _, playerID := range gs.TurnOrder {
				player := gs.GetPlayer(playerID)
				if player != nil && !player.HasPassed {
					allPassed = false
					break
				}
			}
			
			if allPassed {
				return true // Round complete
			}
		}
		
		// Get current player
		currentPlayer := gs.GetCurrentPlayer()
		if currentPlayer == nil {
			continue
		}
		
		// If player hasn't passed, it's their turn
		if !currentPlayer.HasPassed {
			break
		}
	}
	
	return false
}

// StartNewRound prepares the game for a new round
// This transitions from PhaseCleanup (or PhaseSetup for round 1) to PhaseIncome
func (gs *GameState) StartNewRound() {
	gs.Round++
	gs.CurrentPlayerIndex = 0
	
	// Set turn order based on pass order (first to pass goes first next round)
	if len(gs.PassOrder) > 0 {
		gs.TurnOrder = make([]string, len(gs.PassOrder))
		copy(gs.TurnOrder, gs.PassOrder)
	}
	
	// Reset pass order for the new round
	gs.PassOrder = []string{}
	
	// Reset power actions for the new round
	gs.PowerActions.ResetForNewRound()
	
	// Reset all players' passed status and special action usage
	for _, player := range gs.Players {
		player.HasPassed = false
		player.SpecialActionsUsed = make(map[SpecialActionType]bool)
	}
	
	// Start with income phase
	gs.Phase = PhaseIncome
}

// StartIncomePhase transitions to the income phase
func (gs *GameState) StartIncomePhase() {
	gs.Phase = PhaseIncome
	// TODO: Grant income to all players (implemented in income.go)
}

// StartActionPhase transitions to the action phase
func (gs *GameState) StartActionPhase() {
	gs.Phase = PhaseAction
	gs.CurrentPlayerIndex = 0
}

// StartCleanupPhase transitions to the cleanup phase
func (gs *GameState) StartCleanupPhase() {
	gs.Phase = PhaseCleanup
	// TODO: Execute cleanup logic (implemented in cleanup.go)
	// - Cult track rewards
	// - Add coins to bonus tiles
	// - Check for game end
}

// EndGame transitions to the end game phase
func (gs *GameState) EndGame() {
	gs.Phase = PhaseEnd
	// TODO: Calculate final scores (implemented in scoring.go)
}

// AllPlayersPassed checks if all players have passed
func (gs *GameState) AllPlayersPassed() bool {
	for _, player := range gs.Players {
		if !player.HasPassed {
			return false
		}
	}
	return true
}

// GainPriests grants priests to a player while enforcing the 7-priest limit
// Terra Mystica rule: priests in hand + priests on cult tracks <= 7
// Returns the actual number of priests gained (may be less than requested)
func (gs *GameState) GainPriests(playerID string, amount int) int {
	if amount <= 0 {
		return 0
	}
	
	player := gs.GetPlayer(playerID)
	if player == nil {
		return 0
	}
	
	priestsInHand := player.Resources.Priests
	priestsOnCult := gs.CultTracks.GetTotalPriestsOnCultTracks(playerID)
	totalPriests := priestsInHand + priestsOnCult
	maxNewPriests := 7 - totalPriests
	
	if maxNewPriests <= 0 {
		// Already at or above the 7-priest limit, cannot gain any priests
		return 0
	}
	
	// Cap the amount to not exceed the limit
	priestsToGain := amount
	if priestsToGain > maxNewPriests {
		priestsToGain = maxNewPriests
	}
	
	player.Resources.Priests += priestsToGain
	return priestsToGain
}

// IsGameOver checks if the game has ended (after round 6)
func (gs *GameState) IsGameOver() bool {
	return gs.Round > 6
}

// AlchemistsConvertVPToCoins allows Alchemists to convert VP to Coins (1 VP -> 1 Coin)
// This can be done at any time and does not count as an action
func (gs *GameState) AlchemistsConvertVPToCoins(playerID string, vp int) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	
	// Check if player is Alchemists
	alchemists, ok := player.Faction.(*factions.Alchemists)
	if !ok {
		return fmt.Errorf("only Alchemists can use Philosopher's Stone")
	}
	
	// Validate conversion
	coins, err := alchemists.ConvertVPToCoins(vp)
	if err != nil {
		return err
	}
	
	// Check if player has enough VP
	if player.VictoryPoints < vp {
		return fmt.Errorf("not enough VP (have %d, need %d)", player.VictoryPoints, vp)
	}
	
	// Execute conversion
	player.VictoryPoints -= vp
	player.Resources.Coins += coins
	
	return nil
}

// AlchemistsConvertCoinsToVP allows Alchemists to convert Coins to VP (2 Coins -> 1 VP)
// This can be done at any time and does not count as an action
func (gs *GameState) AlchemistsConvertCoinsToVP(playerID string, coins int) error {
	player := gs.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found")
	}
	
	// Check if player is Alchemists
	alchemists, ok := player.Faction.(*factions.Alchemists)
	if !ok {
		return fmt.Errorf("only Alchemists can use Philosopher's Stone")
	}
	
	// Validate conversion
	vp, err := alchemists.ConvertCoinsToVP(coins)
	if err != nil {
		return err
	}
	
	// Check if player has enough coins
	if player.Resources.Coins < coins {
		return fmt.Errorf("not enough coins (have %d, need %d)", player.Resources.Coins, coins)
	}
	
	// Execute conversion
	player.Resources.Coins -= coins
	player.VictoryPoints += vp
	
	return nil
}
