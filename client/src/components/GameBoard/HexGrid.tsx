// HexGrid component - renders all hexes in the game
import React from 'react';
import { Hex } from './Hex';
import type { MapHexData } from '../../data/baseGameMap';
import { hexCenter, HEX_SIZE } from '../../utils/hexUtils';
import type { Building } from '../../types/game.types';

interface HexGridProps {
  hexes: MapHexData[];
  buildings?: Map<string, Building>; // Map from "q,r" to Building
  highlightedHexes?: Set<string>;
  onHexClick?: (q: number, r: number) => void;
  onHexHover?: (q: number, r: number) => void;
}

export const HexGrid: React.FC<HexGridProps> = ({
  hexes: hexesProp, // Rename the prop to avoid conflict
  buildings = new Map<string, Building>(),
  highlightedHexes = new Set(),
  onHexClick,
  onHexHover,
}) => {
  // Create a Map of Hex components from the hexesProp array
  const hexes = new Map<string, JSX.Element>();
  hexesProp.forEach((hexData) => {
    const { coord, terrain, isRiver } = hexData;
    const center = hexCenter(coord.r, coord.q);
    const key = `${String(coord.q)},${String(coord.r)}`;
    const building = buildings.get(key);
    const isHighlighted = highlightedHexes.has(key);

    hexes.set(key, (
      <Hex
        key={key}
        coord={coord}
        center={center}
        terrain={terrain}
        isRiver={isRiver}
        building={building}
        isHighlighted={isHighlighted}
        hexSize={HEX_SIZE}
        onClick={() => onHexClick?.(coord.q, coord.r)}
        onMouseEnter={() => onHexHover?.(coord.q, coord.r)}
      />
    ));
  });

  // Calculate actual min/max from the hexes data
  let minQ = Infinity, maxQ = -Infinity;
  let minR = Infinity, maxR = -Infinity;

  hexesProp.forEach(hex => { // Use hexesProp for calculations as it's the original data
    minQ = Math.min(minQ, hex.coord.q);
    maxQ = Math.max(maxQ, hex.coord.q);
    minR = Math.min(minR, hex.coord.r);
    maxR = Math.max(maxR, hex.coord.r);
  });

  // console.log(`Q range: ${minQ} to ${maxQ}, R range: ${minR} to ${maxR}`);

  // Find actual leftmost/rightmost and top/bottom hex positions
  let minX = Infinity, maxX = -Infinity;
  let minY = Infinity, maxY = -Infinity;
  hexesProp.forEach(hex => { // Use hexesProp for calculations
    const center = hexCenter(hex.coord.r, hex.coord.q);
    minX = Math.min(minX, center.x);
    maxX = Math.max(maxX, center.x);
    minY = Math.min(minY, center.y);
    maxY = Math.max(maxY, center.y);
  });

  const paddingY = HEX_SIZE * 2; // Vertical padding (70px)
  const paddingX = HEX_SIZE; // Horizontal padding (35px)

  // Calculate bounds from actual positions
  const width = maxX - minX + paddingX * 2;
  const height = maxY - minY + paddingY * 2;

  // Offset to position the grid
  const offsetX = -minX + paddingX;
  const offsetY = -minY + paddingY;

  return (
    <svg
      width={width}
      height={height}
      style={{ border: '1px solid #ccc', backgroundColor: '#f0f0f0' }}
    >
      <g transform={`translate(${String(offsetX)}, ${String(offsetY)})`}>
        {Array.from(hexes.values())}
      </g>
    </svg>
  );
};
