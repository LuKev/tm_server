// Individual hex component - renders a single hex with terrain and optional building
import React from 'react';
import type { Building } from '../../types/game.types';
import { type TerrainType } from '../../types/game.types';
import type { AxialCoord } from '../../utils/hexUtils';
import { getHexCorners } from '../../utils/hexUtils';
import { TERRAIN_COLORS, getContrastColor } from '../../utils/colors';
import { BuildingComponent } from './Building';

interface HexProps {
  coord: AxialCoord;
  center: { x: number; y: number };
  terrain: TerrainType;
  isRiver?: boolean;
  building?: Building;
  isHighlighted?: boolean;
  hexSize: number;
  onClick?: () => void;
  onMouseEnter?: () => void;
}

export const Hex: React.FC<HexProps> = ({
  coord,
  center,
  terrain,
  isRiver = false,
  building,
  isHighlighted = false,
  hexSize,
  onClick,
  onMouseEnter,
}) => {
  const corners = getHexCorners(center, hexSize);

  // Create SVG path from corners
  const pathData = corners
    .map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x},${p.y}`)
    .join(' ') + ' Z';

  // River hexes use a special light blue color
  const fillColor = isRiver ? '#b3d9ff' : TERRAIN_COLORS[terrain];
  const strokeColor = isHighlighted ? '#00ff00' : '#333';
  const strokeWidth = isHighlighted ? 3 : 1;

  return (
    <g
      onClick={onClick}
      onMouseEnter={onMouseEnter}
      style={{ cursor: onClick ? 'pointer' : 'default' }}
    >
      {/* Hex background */}
      <path
        d={pathData}
        fill={fillColor}
        stroke={strokeColor}
        strokeWidth={strokeWidth}
      />

      {/* Building (if present) */}
      {building && (
        <BuildingComponent
          building={building}
          center={center}
        />
      )}

      {/* Debug: Show coordinates in dev mode */}
      {process.env.NODE_ENV === 'development' && (
        <text
          x={center.x}
          y={center.y}
          textAnchor="middle"
          fontSize={10}
          fill={getContrastColor(fillColor)}
          style={{ pointerEvents: 'none', userSelect: 'none' }}
        >
          {coord.q},{coord.r}
        </text>
      )}
    </g>
  );
};
