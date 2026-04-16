// GameBoard component - main container for the hex map
import React, { useState } from 'react';
import { HexGridCanvas } from './HexGridCanvas';
import { useGameStore } from '../../stores/gameStore';
import { FactionType, TerrainType, type Building } from '../../types/game.types';
import type { MapHexData } from '../../types/map.types';
import { PowerActions } from './PowerActions';

import { type PowerActionType } from '../../types/game.types';

interface GameBoardProps {
  onHexClick?: (q: number, r: number) => void;
  onBridgeEdgeClick?: (from: { q: number; r: number }, to: { q: number; r: number }) => void;
  bridgeEdgeSelectionEnabled?: boolean;
  onPowerActionClick?: (action: PowerActionType) => void;
  disablePowerActions?: boolean;
  isReplayMode?: boolean;
}

export const GameBoard: React.FC<GameBoardProps> = ({
  onHexClick,
  onBridgeEdgeClick,
  bridgeEdgeSelectionEnabled,
  onPowerActionClick,
  disablePowerActions = false,
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
  const playerFactions = new Map<string, FactionType>();
  if (gameState?.map?.hexes) {
    Object.entries(gameState.map.hexes).forEach(([key, hex]) => {
      if (hex.building) {
        buildings.set(key, hex.building);
      }
    });
  }
  if (gameState?.players) {
    Object.entries(gameState.players).forEach(([playerId, player]) => {
      playerFactions.set(playerId, player.faction);
    });
  }

  const handleHexClick = (q: number, r: number): void => {
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
    if (!gameState?.map?.hexes) return [] as MapHexData[]

    return Object.values(gameState.map.hexes)
      .map((hex): MapHexData => ({
        coord: hex.coord,
        terrain: hex.terrain,
        isRiver: hex.terrain === TerrainType.River,
        displayCoord: hex.displayCoord,
        hasTownTile: hex.hasTownTile,
        townTileType: hex.townTileType,
        townTileOwnerPlayerId: hex.townTileOwnerPlayerId,
        powerTokenOwnerPlayerId: hex.powerTokenOwnerPlayerId,
      }))
      .sort((left, right) => {
        if (left.coord.r !== right.coord.r) return left.coord.r - right.coord.r
        return left.coord.q - right.coord.q
      })
  }, [gameState?.map?.hexes]);

  return (
    <div className="game-board-container bg-white rounded-lg shadow-md p-4 flex flex-col gap-4 h-full w-full overflow-y-auto" data-testid="game-board">
      <div className="overflow-auto flex-shrink-0">
        <HexGridCanvas
          testId="hex-grid-canvas"
          hexes={currentHexes}
          buildings={buildings}
          playerFactions={playerFactions}
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
      <div className="border-t pt-4 flex-1 min-h-0" data-testid="power-actions-section">
        <PowerActions onActionClick={handlePowerActionClick} disabled={disablePowerActions} />
      </div>

      {/* Player Boards Section */}
      <div className="border-t pt-4">

      </div>
    </div>
  );
};
