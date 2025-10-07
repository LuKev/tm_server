# Terra Mystica Online

A web-based multiplayer implementation of the strategic board game Terra Mystica, allowing players to compete against each other in real-time over the internet.

## About Terra Mystica

Terra Mystica is a complex strategy board game where players control one of 14 factions, each with unique abilities. Players terraform terrain, build structures, develop their faction's capabilities, and compete for resources and victory points across six rounds.

## Project Overview

This project aims to create a fully functional online version of Terra Mystica with:

- **Multiplayer Support**: 2-5 players can join and play together
- **Real-time Gameplay**: Live game state synchronization across all players
- **Complete Game Rules**: Full implementation of Terra Mystica mechanics including:
  - 14 unique factions with special abilities
  - Terrain transformation and building placement
  - Resource management (coins, workers, priests, power)
  - Cult track progression
  - Favor tiles, town tiles, and bonus cards
  - Round scoring and final scoring
- **Modern Web Interface**: Intuitive, responsive UI for desktop and tablet
- **Game Lobby System**: Create, join, and manage game sessions
- **Turn-based Mechanics**: Proper turn order and action validation

## Technology Stack

### Frontend
- **React** with TypeScript for type-safe UI development
- **TailwindCSS** for modern, responsive styling
- **shadcn/ui** for polished UI components
- **Lucide React** for icons
- **WebSocket Client** for real-time communication

### Backend
- **Go** (Golang) for high-performance game server
- **Gorilla WebSocket** for WebSocket-based real-time updates
- **Gorilla Mux** for HTTP routing
- **Strong typing** with Go's type system
- **In-memory game state** with concurrent access patterns (with future database expansion)

## Project Structure

```
tm_server/
├── client/                 # React frontend application
│   ├── src/
│   │   ├── components/    # React components
│   │   ├── game/          # Game logic and state management
│   │   ├── types/         # TypeScript type definitions
│   │   └── utils/         # Helper functions
│   └── package.json
├── server/                # Go backend server
│   ├── cmd/
│   │   └── server/        # Main application entry point
│   ├── internal/
│   │   ├── game/          # Game engine and rules
│   │   ├── models/        # Data models and types
│   │   ├── websocket/     # WebSocket handlers
│   │   └── lobby/         # Lobby management
│   ├── go.mod
│   └── go.sum
└── README.md
```

## Getting Started

### Prerequisites
- Go 1.21+ for backend development
- Node.js 18+ and npm for frontend development
- Modern web browser (Chrome, Firefox, Safari, Edge)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd tm_server
```

2. Install Go dependencies:
```bash
cd server
go mod download
```

3. Install client dependencies:
```bash
cd ../client
npm install
```

### Running the Application

1. Start the Go server:
```bash
cd server
go run cmd/server/main.go
```

2. In a separate terminal, start the client:
```bash
cd client
npm run dev
```

3. Open your browser to `http://localhost:5173` (or the port shown in the terminal)

## Development Roadmap

### Phase 1: Foundation (Current)
- [x] Project setup and structure
- [ ] Basic server with WebSocket
- [ ] Client application scaffold
- [ ] Game lobby system

### Phase 2: Core Game Engine
- [ ] Game state management
- [ ] Faction implementations
- [ ] Map and terrain system
- [ ] Building placement logic
- [ ] Resource management

### Phase 3: Game Mechanics
- [ ] Turn order and action system
- [ ] Power cycle mechanics
- [ ] Cult track system
- [ ] Favor tiles and bonus cards
- [ ] Town formation and scoring

### Phase 4: UI/UX
- [ ] Game board visualization
- [ ] Player dashboards
- [ ] Action selection interface
- [ ] Game log and history
- [ ] Responsive design

### Phase 5: Polish & Features
- [ ] Game rules validation
- [ ] Error handling and recovery
- [ ] Player reconnection support
- [ ] Game save/load functionality
- [ ] Tutorial and help system

## Contributing

This is a personal project, but suggestions and feedback are welcome!

## License

This project is for educational and personal use. Terra Mystica is a trademark of Feuerland Spiele.

## Acknowledgments

- Original game design by Helge Ostertag and Jens Drögemüller
- Published by Feuerland Spiele
