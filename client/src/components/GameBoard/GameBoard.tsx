// GameBoard component - main container for the hex map
import React, { useState } from 'react';
import { HexGridCanvas } from './HexGridCanvas';
import { BASE_GAME_MAP } from '../../data/baseGameMap';
import { useGameStore } from '../../stores/gameStore';
import type { Building } from '../../types/game.types';

interface GameBoardProps {
  onHexClick?: (q: number, r: number) => void
}

export const GameBoard: React.FC<GameBoardProps> = ({ onHexClick }) => {
  const gameState = useGameStore(s => s.gameState);
  const [hoveredHex, setHoveredHex] = useState<string | null>(null);

  // Get buildings from game state
  const buildings = new Map<string, Building>();
  if (gameState?.map?.hexes) {
    Object.entries(gameState.map.hexes).forEach(([key, hex]) => {
      if (hex.building) {
        buildings.set(key, hex.building);
      }
    });
  }

  const handleHexClick = (q: number, r: number) => {
    console.log(`GameBoard: Hex clicked: (${q}, ${r})`);
    onHexClick?.(q, r);
  };

  const handleHexHover = (q: number, r: number) => {
    setHoveredHex(`${q},${r}`);
  };

  // Highlight hovered hex
  const highlightedHexes = new Set<string>();
  if (hoveredHex) {
    highlightedHexes.add(hoveredHex);
  }

  return (
    <div className="game-board-container bg-white rounded-lg shadow-md p-4">
      <div className="overflow-auto">
        <HexGridCanvas
          hexes={BASE_GAME_MAP}
          buildings={buildings}
          bridges={[]} // TODO: Get bridges from game state
          highlightedHexes={highlightedHexes}
          onHexClick={handleHexClick}
          onHexHover={handleHexHover}
        />
      </div>
    </div>
  );
};
