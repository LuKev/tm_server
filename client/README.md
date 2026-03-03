# Terra Mystica Frontend

React + TypeScript frontend for Terra Mystica multiplayer board game with Canvas-based rendering.

## 🎯 Overview

Complete frontend implementation featuring:
- **Canvas-based hex map** with 113 hexes, buildings, bridges, and hover effects
- **Cult tracks sidebar** with faction markers, position tracking, and bonus tiles
- **Real-time WebSocket** integration with game server
- **Type-safe architecture** mirroring Go backend models

## 🚀 Quick Start

```bash
cd client
npm install
npm run dev
```

Visit `http://localhost:5173` to access the game.

## 📁 Project Structure

```
src/
├── components/
│   ├── GameBoard/
│   │   ├── HexGridCanvas.tsx    # Canvas hex renderer (ported from game.js)
│   │   └── GameBoard.tsx        # Main game container
│   ├── CultTracks/
│   │   ├── CultTracks.tsx       # Cult track rendering
│   │   └── CultTracksTest.tsx   # Test page
│   ├── Game.tsx                 # Full game view (board + cult tracks)
│   ├── Lobby.tsx                # Game lobby
│   └── MapTest.tsx              # Interactive map testing
├── types/
│   └── game.types.ts            # TypeScript types (mirrors Go backend)
├── utils/
│   ├── hexUtils.ts              # Hex coordinate utilities
│   └── colors.ts                # Color constants
├── stores/
│   └── gameStore.ts             # Zustand state management
└── services/
    └── WebSocketContext.tsx     # WebSocket connection
```

## 🎮 Features

### ✅ Completed

- **Hex Map Rendering**
  - All 113 base game hexes with correct terrain colors
  - 5 building types (Dwelling, Trading House, Temple, Sanctuary, Stronghold)
  - Bridge rendering along hex edges between distance-2 hexes
  - Proper Z-order layering and hover highlighting

- **Cult Tracks**
  - 4 vertical tracks (Fire, Water, Earth, Air) with positions 0-10
  - Faction markers with uppercase letters
  - Position 10 hexagons and tied position handling
  - Bonus tiles with power values and faction-colored priests
  - Integrated sidebar layout adjacent to game board

- **Test Pages**
  - Map test with interactive controls
  - Cult tracks test with sample data
  - Full game view with proper layout

### 🚧 In Progress

- WebSocket game state integration
- Action handling and move inference
- Player resource displays

## 🔧 Development

### Reference Implementation
Based on `terra-mystica/stc/game.js` with direct port of drawing logic to TypeScript/React.

### Key Differences
- **Coordinate System**: Progressive offset for backend compatibility
- **Component Architecture**: React components with Zustand state management
- **Type Safety**: Full TypeScript integration with Go backend types

## 📖 Documentation

- [Implementation Notes](./IMPLEMENTATION_NOTES.md) - Detailed technical documentation
- [Quick Start Guide](./QUICKSTART.md) - Development setup and approach
- [Map Test Guide](./MAP_TEST_GUIDE.md) - Testing instructions

## 🌐 Available Routes

- `/` - Game lobby
- `/game/:id` - Main game view
- `/maptest` - Interactive map testing (development only)
- `/culttrackstest` - Cult tracks testing (development only)

## Expanding the ESLint configuration

If you are developing a production application, we recommend updating the configuration to enable type-aware lint rules:

```js
export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...

      // Remove tseslint.configs.recommended and replace with this
      tseslint.configs.recommendedTypeChecked,
      // Alternatively, use this for stricter rules
      tseslint.configs.strictTypeChecked,
      // Optionally, add this for stylistic rules
      tseslint.configs.stylisticTypeChecked,

      // Other configs...
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```

You can also install [eslint-plugin-react-x](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-x) and [eslint-plugin-react-dom](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-dom) for React-specific lint rules:

```js
// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...
      // Enable lint rules for React
      reactX.configs['recommended-typescript'],
      // Enable lint rules for React DOM
      reactDom.configs.recommended,
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```
