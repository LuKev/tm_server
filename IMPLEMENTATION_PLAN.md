# Terra Mystica Online - Implementation Plan

## Architecture Overview

### High-Level Architecture
```
┌─────────────────┐         WebSocket           ┌─────────────────┐
│                 │ ◄─────────────────────────► │                 │
│  React Client   │   (Gorilla WebSocket)       │   Go Server     │
│   (Frontend)    │                             │    (Backend)    │
│                 │                             │                 │
└─────────────────┘                             └─────────────────┘
        │                                               │
        │                                               │
        ▼                                               ▼
┌─────────────────┐                             ┌─────────────────┐
│  UI Components  │                             │  Game Engine    │
│  - Game Board   │                             │  - State Mgmt   │
│  - Player Panel │                             │  - Rules Logic  │
│  - Action Menu  │                             │  - Validation   │
└─────────────────┘                             └─────────────────┘
```

### Technology Decisions

**Frontend:**
- React 18+ with TypeScript for type safety
- Vite for fast development and building
- TailwindCSS for utility-first styling
- shadcn/ui for pre-built accessible components
- Native WebSocket API for real-time communication
- Zustand or Context API for client-side state management

**Backend:**
- Go 1.21+ for high-performance, concurrent server
- Gorilla Mux for HTTP routing
- Gorilla WebSocket for WebSocket connections
- Strong typing with Go's type system and structs
- In-memory storage with sync.Map for concurrent access
- Goroutines for handling concurrent player connections
- Future: PostgreSQL with pgx driver for persistence

## Implementation Phases

### Phase 1: Project Foundation (Days 1-2) ✅

#### 1.1 Server Setup
- [x] Initialize Go module
- [x] Set up Gorilla Mux HTTP server
- [x] Configure Gorilla WebSocket
- [x] Create basic project structure (cmd/internal layout)
- [x] Add CORS middleware for development

**Files created:**
- ✅ `server/go.mod`
- ✅ `server/cmd/server/main.go`
- ✅ `server/internal/websocket/hub.go`
- ✅ `server/internal/websocket/client.go`
- ✅ `server/internal/websocket/handler.go`

#### 1.2 Client Setup
- [x] Initialize React + TypeScript with Vite
- [x] Configure TailwindCSS
- [ ] Install shadcn/ui (not yet needed)
- [x] Set up WebSocket client connection
- [x] Create basic routing structure

**Files created:**
- ✅ `client/package.json`
- ✅ `client/tsconfig.json`
- ✅ `client/vite.config.ts`
- ✅ `client/tailwind.config.js`
- ✅ `client/src/main.tsx`
- ✅ `client/src/App.tsx`
- ✅ `client/src/services/WebSocketContext.tsx`

#### 1.3 Basic Lobby System
- [x] Create lobby data structure
- [x] Implement create/join/leave game endpoints
- [x] Add player connection/disconnection handling
- [x] Build simple lobby UI

### Phase 2: Core Data Models (Days 3-4) ✅

#### 2.1 Type Definitions
Define Go structs and TypeScript interfaces for all game entities:

**Core Types:**
- ✅ `TerrainType`: 7 terrain types (Plains, Swamp, Lake, Forest, Mountain, Wasteland, Desert)
- ✅ `FactionType`: 14 faction names
- ✅ `BuildingType`: Dwelling, Trading House, Temple, Sanctuary, Stronghold
- ✅ `ResourceType`: Coins, Workers, Priests, Power (in 3 bowls)
- ✅ `CultTrack`: Fire, Water, Earth, Air

**Game State Types:**
- ✅ `GameState`: Complete game state
- ✅ `PlayerState`: Individual player resources, buildings, etc.
- ✅ `MapState`: Hex grid with terrain and buildings
- ✅ `RoundState`: Current round, bonus tiles, scoring tiles

**Files created:**
- ✅ `server/internal/models/game.go`
- ✅ `server/internal/models/faction.go`
- ✅ `server/internal/models/map.go`
- ✅ `server/internal/models/building.go`
- ✅ `server/internal/models/terrain.go`
- ✅ `server/internal/models/resources.go`
- ✅ `client/src/types/game.types.ts` (mirror server types for frontend)

#### 2.2 Game State Manager
- [x] Create GameState struct
- [x] Implement state initialization (basic)
- [x] Add state mutation methods with mutex locks
- [x] Create JSON marshaling/unmarshaling
- [ ] Add state validation (to be expanded in Phase 3+)

**Files created:**
- ✅ `server/internal/game/state.go`
- ✅ `server/internal/game/manager.go`

**Additional files created:**
- ✅ `server/internal/lobby/lobby.go` (sequential game IDs, join/leave logic)
- ✅ `scripts/dev.sh` (dev server launcher)
- ✅ `scripts/dev.ps1` (PowerShell dev launcher)

### Phase 3: Map and Terrain System (Days 5-7) ✅

#### 3.1 Hex Grid Implementation ✅
- [x] Define hex coordinate system (axial coordinates)
- [x] Create hex grid data structure
- [x] Implement neighbor finding algorithm
- [x] Add distance calculation
- [x] Create reachability checks (direct & indirect adjacency)

**Key Concepts:**
- Use axial coordinates (q, r) for hex positioning
- Implement standard Terra Mystica map layout
- Support for base game map (later: expansion maps)

**Additional implementations:**
- [x] Bridge system with precise geometry validation
- [x] Shipping system with river-only BFS
- [x] Complete base game terrain layout (9 rows, 113 hexes)

#### 3.2 Terrain and Building Logic ✅
- [x] Implement terrain transformation rules
- [x] Add building placement validation
- [x] Create adjacency bonus calculation
- [x] Implement town formation detection
- [x] Add power distribution from buildings (power leech)

**Files created:**
- ✅ `server/internal/game/map.go`
- ✅ `server/internal/game/hex.go`
- ✅ `server/internal/game/terraform.go`
- ✅ `server/internal/game/town.go`
- ✅ `server/internal/game/terrain_layout.go`

**Tests created:**
- ✅ `server/internal/game/map_bridge_shipping_test.go` (bridge geometry tests)
- ✅ `server/internal/game/map_indirect_base_test.go` (shipping tests 1-6)
- ✅ `server/internal/game/terraform_test.go` (terrain & building tests)
- ✅ `server/internal/game/town_test.go` (town formation & power leech tests)

### Phase 4: Faction System (Days 8-12) ✅

#### 4.1 Base Faction Interface ✅
- [x] Create Faction interface
- [x] Define common faction properties in base struct
- [x] Implement base resource management
- [x] Add building cost calculations
- [x] Create terraform cost logic

#### 4.2 Implement All 14 Factions ✅
Each faction needs:
- Starting resources
- Home terrain type
- Special ability implementation
- Stronghold ability
- Unique mechanics

**Factions implemented:**
1. [x] **Nomads** (Yellow/Desert): 3 dwellings, sandstorm ability
2. [x] **Fakirs** (Yellow/Desert): Carpet flying with range bonuses
3. [x] **Chaos Magicians** (Red/Wasteland): 2 favor tiles, double-turn
4. [x] **Giants** (Red/Wasteland): Always 2 spades terraform
5. [x] **Swarmlings** (Blue/Lake): +3 workers from towns, expensive buildings
6. [x] **Mermaids** (Blue/Lake): Skip river for town, shipping bonuses
7. [x] **Witches** (Green/Forest): +5 VP for towns, Witches' Ride
8. [x] **Auren** (Green/Forest): Favor tile + cult advance
9. [x] **Halflings** (Brown/Plains): +1 VP per spade, cheap digging
10. [x] **Cultists** (Brown/Plains): Cult advance from power leech
11. [x] **Alchemists** (Black/Swamp): VP/Coin conversion, power bonuses
12. [x] **Darklings** (Black/Swamp): Priests for terraform, expensive sanctuary
13. [x] **Engineers** (Gray/Mountain): Cheap buildings, bridge bonuses
14. [x] **Dwarves** (Gray/Mountain): Tunneling, no shipping

**Files created:**
- ✅ `server/internal/game/factions/faction.go` (interface and base)
- ✅ `server/internal/game/factions/registry.go`
- ✅ All 14 faction implementation files
- ✅ All 14 faction test files
- ✅ 166 test cases passing

### Phase 5: Resource Management (Days 13-14) ✅

#### 5.1 Power System ✅
- [x] Implement 3-bowl power cycle
- [x] Add power gain mechanics
- [x] Create power action costs
- [x] Handle power leech (neighbor building)
- [x] Implement power conversion (burn: 2:1 bowl 2→bowl 3)

#### 5.2 Resource Tracking ✅
- [x] Create resource pool management
- [ ] Add income phase calculations (deferred - faction-specific)
- [x] Implement resource spending validation
- [x] Create resource conversion actions
- [x] Add resource display helpers (ToResources, Clone)

**Files created:**
- ✅ `server/internal/game/power.go` (PowerSystem with 3-bowl cycle)
- ✅ `server/internal/game/power_test.go` (15 test cases)
- ✅ `server/internal/game/resources.go` (ResourcePool + PowerLeech)
- ✅ `server/internal/game/resources_test.go` (28 test cases)

**Key implementations:**
- Power cycle: Bowl 1→Bowl 2→Bowl 3 priority system
- Resource conversions: Power→Coins/Workers/Priests, Priest→Worker, Worker→Coin
- Power leech with capacity limiting
- Priest system: 7 total per player (pool + cult tracks + supply)

### Phase 6: Turn System and Actions (Days 15-18) ✅ COMPLETE

#### 6.1 Turn Order Management ✅
- [x] Implement turn order tracking
- [x] Add pass mechanism (pass order determines next round turn order)
- [x] Implement round transitions
- [x] Action phase end detection (AllPlayersPassed)

#### 6.2 Action System ✅ COMPLETE
Implement all player actions:
- [x] **Transform and Build**: Terraform + place dwelling (13 tests)
- [x] **Upgrade Building**: Dwelling → TP → Temple/SH/SA (12 tests)
- [x] **Advance Shipping**: Increase shipping range
- [x] **Advance Digging**: Reduce terraform cost
- [x] **Power Actions**: 6 power actions from game board (13 tests)
  - Bridge (3 power), Priest (3 power), Workers (4 power), Coins (4 power)
  - 1 Spade (4 power), 2 Spades (6 power)
- [x] **Special Actions**: Faction-specific stronghold abilities (21 tests)
  - [x] Auren: Stronghold cult advance (7 tests)
  - [x] Witches: Stronghold Witches' Ride (7 tests)
  - [x] Swarmlings: Free dwelling→TP upgrade (3 tests)
  - [x] Chaos Magicians: Double-turn (implemented, test deferred)
  - [x] Giants: 2 free spades transform (2 tests)
  - [x] Nomads: Sandstorm adjacent transform (2 tests)
  - 📝 8 other factions: Documented, passive abilities deferred
  - [ ] 2 bonus card special actions (defer to Phase 7)
- [x] **Pass**: End turn and select bonus card (bonus card selection deferred to Phase 7)

#### 6.3 Action Validation ✅ COMPLETE
- [x] Create action validator (integrated into each action)
- [x] Check resource availability
- [x] Validate placement rules
- [x] Verify turn order (HasPassed check)
- [x] Add error messages

#### 6.4 Game Flow Structure ✅ COMPLETE
**Game Structure**: 6 rounds, each with 3 phases
- [x] **Income Phase**: Players receive resources based on buildings (17 tests)
  - [x] Implement base income system (0/1/2 workers by faction)
  - [x] Add faction-specific income modifiers
  - [x] Track income from buildings (coins, workers, power)
    - [x] Dwellings (8th dwelling rule, Engineers exception)
    - [x] Trading Houses (4 faction variations)
    - [x] Temples (Engineers 2nd temple exception)
    - [x] Sanctuaries (standard)
    - [x] Strongholds (14 faction variations)
  - [x] Power properly cycles through bowls using GainPower()
  - [ ] Bonus tile income (Phase 7)
- [x] **Action Phase**: Players take turns performing actions
  - [x] Turn order management
  - [x] Action execution
  - [x] Pass mechanism
  - [x] Action phase end detection (AllPlayersPassed)
- [x] **Cleanup Phase**: End-of-round maintenance ✅ COMPLETE
  - [x] Cult track rewards (multiple thresholds calculated correctly)
  - [x] Add coins to leftover bonus cards
  - [x] Return bonus cards to available pool
  - [x] Reset round-specific state (power actions, HasPassed, PassOrder)
  - [x] Pending spades system (from cult rewards, used in pass order)
  - [x] Check for game end (IsGameOver - after round 6)
- [ ] **End Game Scoring**: Final VP calculation (Phase 8.2)

**Files created:**
- ✅ `server/internal/game/state.go` (turn order, pass order, phase tracking, PendingSpades)
- ✅ `server/internal/game/actions.go` (all basic actions)
- ✅ `server/internal/game/action_cult_spade.go` (cult spade action)
- ✅ `server/internal/game/power_actions.go` (power actions from game board)
- ✅ `server/internal/game/special_actions.go` (stronghold + bonus card special actions)
- ✅ `server/internal/game/income.go` (complete income system)
- ✅ `server/internal/game/cleanup.go` (cleanup phase orchestration)
- ✅ `server/internal/game/scoring_tiles.go` (scoring tiles + cult rewards)
- 📝 `server/internal/game/scoring.go` (final scoring - Phase 8.2)
- ✅ **200 tests passing**

### Phase 7: Cult Tracks and Tiles (Days 19-20) ✅ COMPLETE

#### 7.1 Cult Track System ✅ COMPLETE
- [x] Create 4 cult tracks (Fire, Water, Earth, Air)
- [x] Implement advancement logic
  - [x] Milestone power bonuses at positions 3/5/7/10
  - [x] Position 10 restriction (only one player per track)
  - [x] Bonus tracking (prevent double-claiming)
- [x] Add cult track scoring (end-game majority bonuses)
- [x] Handle town cult bonuses (8-point and 2-key towns)
- [x] Implement end-game cult majority scoring (8/4/2 VP for top 3)
- [x] Send Priest to Cult Track action

#### 7.2 Favor Tiles ✅ COMPLETE
- [x] Define all 12 favor tiles (+3, +2, +1 variants)
- [x] Implement tile availability tracking
- [x] Add favor tile benefits
  - [x] Income bonuses (Fire+1, Earth+2, Air+2)
  - [x] VP scoring (Earth+1, Water+1, Air+1)
  - [x] Helper functions (Fire+2 town power, Air+1 pass VP)
  - [x] Water+2 special action (advance 1 on any cult track)
- [x] Chaos Magicians special logic (2 favor tiles instead of 1)
- [ ] Favor tile selection UI/prompt system (deferred to Phase 9+)

#### 7.3 Bonus Cards ✅ COMPLETE
- [x] Define all 10 bonus cards
- [x] Implement random selection during setup (playerCount + 3)
- [x] Implement selection during pass
- [x] Add bonus card scoring (pass VP bonuses)
- [x] Create immediate benefits (income bonuses)
- [x] Implement 2 bonus card special actions (spade, cult advance)
- [x] Coin accumulation on leftover cards

#### 7.4 Town Tiles ✅ COMPLETE
- [x] Define all 7 town tiles (5, 6, 7, 8, 9, 11, 2 points)
- [x] Implement town tile availability tracking (2 copies of most, 1 of 11/2)
- [x] Implement town formation detection
  - [x] Standard: 4+ buildings, 7+ power
  - [x] Sanctuary rule: 3+ buildings allowed
  - [x] Fire 2 favor tile: 6+ power requirement
  - [x] Bridge connections count as adjacent
- [x] Implement town tile selection system (PendingTownFormation)
- [x] Handle all town formation rewards
  - [x] VP (2, 5, 6, 7, 8, 9, 11)
  - [x] Resources (coins, workers, priests, power)
  - [x] Keys (for cult advancement)
  - [x] Cult advancement (8-point: +1 all, 2-point: +2 all)
- [x] Add faction-specific town bonuses
  - [x] Witches: +5 VP per town
  - [x] Swarmlings: +3 workers per town
- [x] Integrate with build/upgrade actions

**Files created:**
- ✅ `server/internal/game/cult.go` (cult track system - 286 lines)
- ✅ `server/internal/game/favor.go` (favor tile system - 338 lines)
- ✅ `server/internal/game/bonus_cards.go` (bonus card system - 420 lines)
- ✅ `server/internal/game/town.go` (complete town system - 350+ lines)

**Test Coverage:**
- ✅ 18 cult track tests
- ✅ 10 favor tile tests
- ✅ 8 action scoring tests (favor tile & bonus card VP)
- ✅ 8 bonus card special action tests
- ✅ 7 scoring tile tests
- ✅ 24 cleanup phase tests (all cult rewards, bonus cards, full integration)
- ✅ 1 income test (favor tile & bonus card income)
- ✅ 4 turn order tests (bonus card selection)
- ✅ 17 town formation tests (all 7 tiles, Sanctuary, Fire 2, faction bonuses, prompting)
- ✅ **157 total tests passing**

### Phase 8: Scoring System (Days 21-22)

#### 8.1 Round Scoring ✅ COMPLETE
- [x] Implement scoring tile evaluation (9 tiles with action VP)
- [x] Add bonus card scoring (pass VP bonuses)
- [x] Cult track end-of-round rewards (all 5 types: priests, power, workers, coins, spades)
- [x] Spade VP calculation (faction-specific: Giants always 2 spades)
- [x] Dwelling/Trading House/Stronghold VP from scoring tiles
- [x] Town VP from scoring tiles
- [x] Track victory points (awarded throughout game)

#### 8.2 Final Scoring ✅ COMPLETE
- [x] Calculate area majority (largest connected area - 18 VP, ties split)
- [x] Score cult track positions (8/4/2 VP for top 3 per track, all 4 tracks)
- [x] Add resource conversion to VP (3 coins = 1 VP, 1 worker = 1 VP, 1 priest = 1 VP)
- [x] Determine winner (highest VP, tiebreaker: total resource value)
- [x] Ranked player list for leaderboard

#### 8.3 Auction System ✅ COMPLETE
- [x] Implement Standard Auction (nomination + bidding phases)
- [x] Turn order by nomination order
- [x] Starting VP based on bids (40 - bid)
- [x] Overbidding mechanics (must reduce VP by at least 1)
- [x] GameSetupOptions for auction vs direct selection
- [ ] Fast Auction algorithm (optional, deferred)

**Files created:**
- ✅ `server/internal/game/scoring_tiles.go` (round scoring with cult rewards)
- ✅ `server/internal/game/final_scoring.go` (end-game scoring system)
- ✅ `server/internal/game/cleanup.go` (cleanup phase orchestration)
- ✅ `server/internal/game/auction.go` (standard auction system)

**Test Coverage:**
- ✅ 7 scoring tile tests
- ✅ 24 cleanup phase tests
- ✅ 12 final scoring tests
- ✅ 13 auction tests
- ✅ **228 total tests passing**

### Phase 8.5: Faction Special Abilities (Days 22-23) - IN PROGRESS

Complete implementation of all faction-specific mechanics and special abilities.

#### Status by Faction:

**✅ FULLY IMPLEMENTED (with Integration Tests):**
1. **Halflings** - Cheap dig (1P/2W/1C), +1 VP per spade always, stronghold: 3 spades ✅
2. **Swarmlings** - Expensive dwellings (2W+3C), stronghold: free D→TH upgrade/round, +3 workers/town ✅
3. **Alchemists** - VP↔Coin conversion (1:1 and 2:1), +2 power per spade after stronghold ✅
4. **Cultists** - +7 VP on stronghold, +1 power if all refuse leech ✅
5. **Giants** - 2 spades/terraform always, stronghold: 2 free spades/round (awards scoring tile VP) ✅
6. **Engineers** - Reduced building costs, bridge cost 2W, stronghold: 3VP/bridge on pass ✅
7. **Witches** - +5 VP per town, stronghold: Witches' Ride (free dwelling on any forest, once/round, ignores adjacency) ✅
8. **Auren** - Expensive sanctuary (8C/4W), stronghold: 1 favor tile on build + cult advance special action (2 spaces, once/round) ✅
9. **Fakirs** - Carpet flight (1P, +4 VP, range 1+stronghold+shipping tile), cannot upgrade shipping/digging past 1, expensive stronghold ✅
10. **Dwarves** - Tunneling (2W before stronghold/1W after, +4 VP, range 1), cannot upgrade shipping ✅
11. **Darklings** - Terraform with priests (1P per spade, +2 VP per spade), priest ordination (convert up to 3 workers to priests, once/game, 7-priest limit), cannot upgrade digging, expensive sanctuary ✅
12. **Chaos Magicians** - 2 favor tiles, double turn (take 2 actions, once/round), start with 4W/15C, 1 dwelling placed last, cheap stronghold (4C), expensive sanctuary (8C) ✅
13. **Nomads** - Sandstorm (transform adjacent hex, once/round), start with 3 dwellings, 2W/15C ✅

**⚠️ PARTIALLY IMPLEMENTED:**
14. **Mermaids** - Water building, town bonuses (needs water hex placement validation)

#### Work Needed:

**High Priority (Core Mechanics):**
- [x] Witches Ride action (place dwelling on any forest, once per round after stronghold) ✅
- [x] Fakirs carpet flight action (place dwelling ignoring adjacency, pay priest) ✅
- [x] Dwarves tunneling in Transform+Build (skip terrain/river by paying workers) ✅
- [x] Nomads Sandstorm action (place dwelling on any desert, once per round after stronghold) ✅
- [x] Chaos Magicians double turn (take 2 actions instead of 1, once per round after stronghold) ✅
- [x] Auren cult advance action (advance 2 on any cult track, once per round after stronghold) ✅
- [x] Darklings priest ordination (convert up to 3 workers to priests, once per game after stronghold) ✅
- [x] Giants 2 free spades transform (once per round after stronghold) ✅
- [x] Swarmlings free dwelling upgrade (once per round after stronghold) ✅

**Remaining Work:**
- [ ] Mermaids river-skipping for town formation (skip 1 river hex when founding town)
- [ ] Halflings stronghold: Prompt for applying 3 spades immediately + optional dwelling
- [ ] Darklings stronghold: Prompt for converting 0-3 workers to priests immediately
- [ ] Auren stronghold: Prompt for selecting favor tile immediately

**Low Priority (Already Functional):**
- [x] All passive abilities (cost reductions, VP bonuses, etc.)
- [x] Town bonuses (Witches, Swarmlings, Mermaids)
- [x] Area scoring (Fakirs, Dwarves)
- [x] Resource conversion (Alchemists)

### Phase 9: Frontend UI - Game Board (Days 23-28)

#### 9.1 Hex Grid Rendering
- [ ] Create SVG-based hex grid component
- [ ] Implement hex coordinate conversion
- [ ] Add terrain coloring
- [ ] Create building sprites/icons
- [ ] Add hover effects and selection

#### 9.2 Interactive Map
- [ ] Implement click handlers for hexes
- [ ] Add valid placement highlighting
- [ ] Show reachability ranges

#### 9.3 Visual Polish/Accessability
- [ ] Add animations for building placement
- [ ] Add tooltips for game elements
- [ ] Create confirmation and undo system

**Files to create:**
- `client/src/components/GameBoard/GameBoard.tsx`
- `client/src/components/GameBoard/HexGrid.tsx`
- `client/src/components/GameBoard/Hex.tsx`
- `client/src/components/GameBoard/Building.tsx`
- `client/src/utils/hexUtils.ts`

### Phase 10: Frontend UI - Player Interface (Days 29-32)

#### 10.1 Player Dashboard
- [ ] Create resource display panel
- [ ] Show power bowls visualization
- [ ] Display cult track positions
- [ ] Show available actions
- [ ] Add faction card display

#### 10.2 Action Interface
- [ ] Create action selection menu
  - There should not be a menu - clicking the appropriate spots on the player board or map should imply what action is being taken
  - Click on a map hex that has no building on it: terraform + build action
  - Click on a map hex that has your own building on it: upgrade building
  - Click on spades upgrade track: upgrade digging
  - Click on shipping upgrade track: upgrade shipping
  - Click on a cult: send priest to a cult
  - Click on a power action: Use a special power action
  - Click on a special action octagon: Use a special action
  - Click on a pass tile: Passing action
- [ ] Build terraform/build workflow
- [ ] Add upgrade building interface
- [ ] Create power action menu
  - Power actions and special actions when available should look like an orange hexagon with an icon in them. When used, they should be covered by an "X" token.
- [ ] Implement pass and bonus selection

#### 10.3 Game Information
- [ ] Show current round and phase
- [ ] Display turn order
- [ ] Create game log/history
- [ ] Add scoring breakdown
- [ ] Show available town and favor tiles

**Files to create:**
- `client/src/components/PlayerDashboard/PlayerDashboard.tsx`
- `client/src/components/PlayerDashboard/ResourcePanel.tsx`
- `client/src/components/PlayerDashboard/PowerBowls.tsx`
- `client/src/components/PlayerDashboard/CultTrackDisplay.tsx`
- `client/src/components/ActionMenu/ActionMenu.tsx`
- `client/src/components/ActionMenu/ActionButton.tsx`
- `client/src/components/GameLog/GameLog.tsx`

### Phase 11: Lobby and Multiplayer (Days 33-35)

#### 11.1 Lobby System
- [ ] Create game lobby UI
- [ ] Implement game creation
- [ ] Add join game functionality
- [ ] Show player list
- [ ] Add faction selection
- [ ] Implement ready/start mechanism

#### 11.2 Real-time Synchronization
- [ ] Set up Socket.io event handlers
- [ ] Implement state broadcasting
- [ ] Add optimistic updates
- [ ] Handle disconnections
- [ ] Create reconnection logic

#### 11.3 Game Session Management
- [ ] Create session storage
- [ ] Implement game persistence
- [ ] Add spectator mode
- [ ] Create game history

**Files to create:**
- `client/src/components/Lobby/Lobby.tsx`
- `client/src/components/Lobby/GameList.tsx`
- `client/src/components/Lobby/CreateGame.tsx`
- `client/src/components/Lobby/FactionSelector.tsx`
- `client/src/hooks/useWebSocket.ts`
- `client/src/hooks/useGameState.ts`
- `server/internal/websocket/game_events.go`
- `server/internal/lobby/lobby.go`

### Phase 12: Polish and Testing (Days 36-40)

#### 12.1 Error Handling
- [ ] Add comprehensive error messages
- [ ] Implement error boundaries
- [ ] Create user-friendly notifications
- [ ] Add validation feedback
- [ ] Handle edge cases

#### 12.2 Performance Optimization
- [ ] Optimize state updates
- [ ] Add memoization where needed
- [ ] Reduce unnecessary re-renders
- [ ] Optimize WebSocket messages
- [ ] Add loading states

#### 12.3 Testing
- [ ] Write unit tests for game logic
- [ ] Add integration tests for actions
- [ ] Test multiplayer scenarios
- [ ] Validate all faction abilities
- [ ] Test edge cases and rule interactions
- [ ] Create end to end testing by validating whole games are processed correctly

#### 12.4 Documentation
- [ ] Add code comments
- [ ] Create API documentation
- [ ] Write game rules reference
- [ ] Add developer setup guide
- [ ] Create user manual

### Phase 13: Polish and Testing (Days 41-44)

#### 13.1 Notation system
- [ ] Create notation system for logging games
  - Consider using something snellman compatible, or maybe a simplified version of the snellman system?
  - Full games should be replayable purely from the notation

- [ ] Create a replay viewer. Key features:
  - Go forwards/backwards one turn
  - Forwards/backwards custom number of turns
  - Jump to end of round/beginning of next round/end of auction
  - Jump to beginning of game/eng of game

## Key Technical Challenges

### 1. Complex Game State Management
**Challenge**: Terra Mystica has intricate state with many interdependencies.
**Solution**: 
- Use immutable state updates
- Implement state validation at each mutation
- Create comprehensive TypeScript types
- Use event sourcing pattern for action history

### 2. Faction Special Abilities
**Challenge**: Each faction has unique mechanics that break standard rules.
**Solution**:
- Use strategy pattern for faction-specific behavior
- Create hooks/callbacks for ability triggers
- Implement ability as composable modifiers
- Extensive testing for each faction

### 3. Real-time Synchronization
**Challenge**: Keep game state consistent across multiple clients.
**Solution**:
- Server is source of truth
- Clients send actions, server validates and broadcasts
- Implement optimistic updates with rollback
- Add conflict resolution

### 4. Power Leech Mechanics
**Challenge**: When a player builds, neighbors can gain power (requires interaction).
**Solution**:
- Pause action execution for leech decisions
- Implement timeout for automatic decline
- Queue multiple leech opportunities
- Clear UI for leech decisions

### 5. Town Formation Detection
**Challenge**: Detecting when buildings form a town (complex graph problem).
**Solution**:
- Implement connected component algorithm
- Check town criteria (4+ buildings, total power value ≥7)
- Cache town formations for performance
- Recalculate only when buildings change

## Development Best Practices

### Code Organization
- Separate concerns: UI, game logic, networking
- Use TypeScript strictly (no `any` types)
- Follow consistent naming conventions
- Keep functions small and focused

### State Management
- Single source of truth on server
- Immutable state updates
- Clear action → state mutation flow
- Validate all state transitions

### Testing Strategy
- Unit tests for pure game logic
- Integration tests for action flows
- Manual testing for UI/UX
- Multiplayer scenario testing

### Version Control
- Commit frequently with clear messages
- Use feature branches for major changes
- Keep main branch stable
- Tag releases

## Future Enhancements

### Post-MVP Features
- [ ] AI opponents
- [ ] Replay system
- [ ] Tournament mode
- [ ] Expansion content (Fire & Ice, etc.)
- [ ] Mobile responsive design
- [ ] Game statistics and analytics
- [ ] User accounts and authentication
- [ ] Persistent game storage (database)
- [ ] Chat system
- [ ] Elo rating system

### Performance Improvements
- [ ] Database integration (PostgreSQL)
- [ ] Redis for session management
- [ ] CDN for static assets
- [ ] Server-side rendering
- [ ] Progressive Web App (PWA)

## Timeline Estimate

- **Phase 1-2**: Foundation & Models (4 days)
- **Phase 3-4**: Map & Factions (10 days)
- **Phase 5-8**: Game Mechanics (10 days)
- **Phase 9-10**: Frontend UI (10 days)
- **Phase 11**: Multiplayer (3 days)
- **Phase 12**: Polish & Testing (5 days)

**Total Estimated Time**: ~40 days of focused development

This is an aggressive timeline. A more realistic estimate with testing and iteration would be 60-90 days.

## Next Steps

1. Set up the development environment
2. Initialize server and client projects
3. Implement basic WebSocket connection
4. Create simple lobby to test multiplayer
5. Begin implementing core game state and map system

Let's start building! 🎮
