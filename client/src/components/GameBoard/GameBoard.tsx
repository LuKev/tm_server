// GameBoard component - main container for the hex map
import React, { useState } from 'react';
import { HexGridCanvas } from './HexGridCanvas';
import { BASE_GAME_MAP } from '../../data/baseGameMap';
import { useGameStore } from '../../stores/gameStore';
import type { Building } from '../../types/game.types';
import { PowerActions } from './PowerActions';

import { type PowerActionType } from '../../types/game.types';

interface GameBoardProps {
  onHexClick?: (q: number, r: number) => void;
  onBridgeEdgeClick?: (from: { q: number; r: number }, to: { q: number; r: number }) => void;
  bridgeEdgeSelectionEnabled?: boolean;
  onPowerActionClick?: (action: PowerActionType) => void;
  isReplayMode?: boolean;
}

export const GameBoard: React.FC<GameBoardProps> = ({
  onHexClick,
  onBridgeEdgeClick,
  bridgeEdgeSelectionEnabled,
  onPowerActionClick,
  isReplayMode
}): React.ReactElement => {
  const gameState = useGameStore(s => s.gameState);
  const [hoveredHex, setHoveredHex] = useState<string | null>(null);

  // In replay mode, don't track hover state
  const handleHexHover = (q: number, r: number): void => {
    if (!isReplayMode) {
      setHoveredHex(`${String(q)},${String(r)}`);
    }
  };

  // Get buildings from game state
  const buildings = new Map<string, Building>();
  if (gameState?.map?.hexes) {
    Object.entries(gameState.map.hexes).forEach(([key, hex]) => {
      if (hex.building) {
        buildings.set(key, hex.building);
      }
    });
  }

  const handleHexClick = (q: number, r: number): void => {
    // console.log(`GameBoard: Hex clicked: (${q}, ${r})`);
    onHexClick?.(q, r);
  };

  // Highlight hovered hex
  const highlightedHexes = new Set<string>();
  if (hoveredHex) {
    highlightedHexes.add(hoveredHex);
  }

  const handlePowerActionClick = (action: PowerActionType): void => {
    onPowerActionClick?.(action);
  };

  // Merge dynamic terrain data from gameState
  const currentHexes = React.useMemo(() => {
    if (!gameState?.map?.hexes) return BASE_GAME_MAP;

    return BASE_GAME_MAP.map(baseHex => {
      const key = `${String(baseHex.coord.q)},${String(baseHex.coord.r)}`;
      const dynamicHex = gameState.map.hexes[key];
      // Check if dynamicHex exists and has a valid terrain (0 is a valid enum value)
      if (dynamicHex?.terrain !== undefined) {
        return {
          ...baseHex,
          terrain: dynamicHex.terrain
        };
      }
      return baseHex;
    });
  }, [gameState?.map?.hexes]);

  return (
    <div className="game-board-container bg-white rounded-lg shadow-md p-4 flex flex-col gap-4 h-full w-full overflow-y-auto">
      <div className="overflow-auto flex-shrink-0">
        <HexGridCanvas
          hexes={currentHexes}
          buildings={buildings}
          bridges={gameState?.map?.bridges || []}
          highlightedHexes={highlightedHexes}
          onHexClick={handleHexClick}
          onBridgeEdgeClick={onBridgeEdgeClick}
          bridgeEdgeSelectionEnabled={bridgeEdgeSelectionEnabled}
          onHexHover={handleHexHover}
          showCoords={!isReplayMode}
          disableHover={isReplayMode}
        />
      </div>

      {/* Power Actions Section */}
      <div className="border-t pt-4 flex-1 min-h-0">
        <PowerActions onActionClick={handlePowerActionClick} />
      </div>

      {/* Player Boards Section */}
      <div className="border-t pt-4">

      </div>
    </div>
  );
};
