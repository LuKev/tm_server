# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terra Mystica Online is a web-based multiplayer implementation of the board game Terra Mystica. It features a Go backend with WebSocket-based real-time communication and a React + TypeScript frontend.

## Build System & Commands

### Go Backend (Bazel)

The server uses **Bazel** as the build system. **Always use Bazel commands, never `go` directly**.

```bash
# Build the server
cd server
bazel build //cmd/server:server

# Run the server (starts on port 8080)
bazel run //cmd/server:server

# Run all tests
bazel test //...

# Run specific package tests
bazel test //internal/game:game_test
bazel test //internal/game/factions:factions_test

# Run single test
bazel test //internal/game:game_test --test_filter=TestTransformAndBuild
```

### React Frontend (npm)

```bash
cd client

# Development server (port 5173)
npm run dev

# Build for production
npm run build

# Lint
npm run lint

# Preview production build
npm run preview
```

## Architecture

### Server Structure (`server/`)

- `cmd/server/main.go` - Entry point, WebSocket hub initialization, HTTP router setup
- `internal/game/` - Core game engine
  - `state.go` - GameState struct, game phases (Setup, Income, Action, Cleanup, End)
  - `actions.go` - Action interface and base implementations
  - `map.go` - Hex grid with axial coordinates (q, r), adjacency, bridges, shipping
  - `terraform.go`, `power.go`, `cult.go`, `town.go` - Core game mechanics
  - `income.go`, `cleanup.go`, `final_scoring.go` - Phase-specific logic
  - `factions/` - 14 faction implementations (Witches, Nomads, Halflings, Cultists, Alchemists, Darklings, Engineers, Swarmlings, Chaos Magicians, Giants, Fakirs, Dwarves, Mermaids, Auren)
    - Each faction has unique abilities, costs, and special actions
    - `faction.go` - Faction interface definition
    - `registry.go` - Faction registry for lookups
- `internal/websocket/` - WebSocket hub, client management, message broadcasting
- `internal/lobby/` - Game lobby system (create, join, leave)
- `internal/models/` - Shared data models for WebSocket messages

### Client Structure (`client/src/`)

- `components/` - React UI components
- `services/` - WebSocket client connection and context
- `types/` - TypeScript type definitions (mirror server models)

### Key Design Patterns

**Action-based Game Engine:**
All player actions implement the `Action` interface:
```go
type Action interface {
    GetType() ActionType
    GetPlayerID() string
    Validate(gs *GameState) error
    Execute(gs *GameState) error
}
```

Actions validate game state before execution. Common actions:
- `TransformAndBuildAction` - Terraform hex and optionally build dwelling
- `UpgradeBuildingAction` - Upgrade buildings (Dwelling → Trading House/Temple)
- `AdvanceShippingAction`, `AdvanceDiggingAction` - Track advancement
- `SendPriestToCultAction` - Cult track progression
- `PowerAction` - Power actions from power bowls
- `PassAction` - End turn and select bonus card

**Faction System:**
Each faction implements the `Faction` interface with:
- Starting resources and home terrain
- Building/shipping/digging costs
- Special abilities (flying, water building, bridge bonuses, etc.)
- Stronghold abilities
- Income modifiers

**Hex Map:**
- Axial coordinate system (q, r)
- 113 hexes in base game layout (9 rows)
- Supports direct adjacency, indirect adjacency via bridges/shipping
- River detection for shipping mechanics

**Game Phases:**
1. `PhaseSetup` - Initial faction selection and dwelling placement
2. `PhaseIncome` - Resource income based on buildings and bonus cards
3. `PhaseAction` - Turn-based player actions
4. `PhaseCleanup` - End-of-round scoring, cult track rewards
5. `PhaseEnd` - Final scoring after round 6

## Testing Conventions

- All test files use `_test.go` suffix
- Tests use table-driven testing pattern where applicable
- Faction tests validate special abilities, costs, and unique mechanics
- Game engine tests cover action validation and state transitions
- Tests are organized by feature (e.g., `action_upgrade_building_test.go`, `alchemists_test.go`)

Run tests frequently during development to catch regressions.

## WebSocket Message Protocol

Client ↔ Server communication uses JSON messages over WebSocket:
- Connection endpoint: `ws://localhost:8080/ws`
- Messages are broadcast to all clients in a game
- Game state updates sent on every action
- Lobby messages for create/join/leave game

## Development Workflow

1. **Start server**: `cd server && bazel run //cmd/server:server`
2. **Start client**: `cd client && npm run dev`
3. **Access UI**: http://localhost:5173
4. **Test changes**: `cd server && bazel test //...`

## Important Notes

- **Always use Bazel** for Go builds and tests, never invoke `go` commands directly
- The game engine validates all actions before execution - check existing validation patterns
- Faction-specific behavior is isolated in `internal/game/factions/` - modify faction files for special abilities
- Power mechanics use 3 bowls (power cycling from bowl 1 → 2 → 3)
- Cult tracks range from 0-10 with special rewards at positions 3, 5, 7, 10
- Buildings provide power to adjacent opponents (power leech mechanic)
- Town formation requires 4+ buildings with total power value ≥7
