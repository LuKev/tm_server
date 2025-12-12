// Hex coordinate utilities - TypeScript version of terra-mystica/stc/game.js

export interface AxialCoord {
  q: number;
  r: number;
}

export interface PixelCoord {
  x: number;
  y: number;
}

// Hex size constant (matches terra-mystica/stc/game.js)
export const HEX_SIZE = 35;
export const HEX_WIDTH = Math.cos(Math.PI / 6) * HEX_SIZE * 2;
export const HEX_HEIGHT = Math.sin(Math.PI / 6) * HEX_SIZE + HEX_SIZE;

/**
 * Convert axial coordinates to pixel coordinates for rendering
 * Terra Mystica uses offset coordinates where:
 * - Odd rows (1,3,5,7) are shifted RIGHT by half a hex width
 * - Each pair of rows is shifted RIGHT by an additional full hex width
 *   (rows 0-1: +0, rows 2-3: +1, rows 4-5: +2, rows 6-7: +3, row 8: +4)
 */
export function hexCenter(row: number, col: number, hasA1 = true): PixelCoord {
  const yOffset = hasA1 ? 0 : -HEX_HEIGHT;

  // Odd rows get half-hex offset
  const oddRowOffset = row % 2 ? HEX_WIDTH / 2 : 0;

  // Progressive shift: each pair of rows shifts right by one full hex
  const progressiveOffset = Math.floor(row / 2) * HEX_WIDTH;

  return {
    x: 5 + HEX_SIZE + col * HEX_WIDTH + oddRowOffset + progressiveOffset,
    y: 5 + HEX_SIZE + row * HEX_HEIGHT + yOffset,
  };
}

/**
 * Convert axial coordinates to row/col used in terra-mystica layout
 */
export function axialToRowCol(coord: AxialCoord): { row: number; col: number } {
  // Terra Mystica uses row/col notation, need to map from axial
  // This mapping depends on your specific layout
  return {
    row: coord.r,
    col: coord.q,
  };
}

/**
 * Get hex corners for drawing
 * Based on terra-mystica/stc/game.js makeHexPath
 */
export function getHexCorners(center: PixelCoord, size = HEX_SIZE): PixelCoord[] {
  const corners: PixelCoord[] = [];
  let x = center.x - Math.cos(Math.PI / 6) * size;
  let y = center.y + Math.sin(Math.PI / 6) * size;
  let angle = 0;

  for (let i = 0; i < 6; i++) {
    corners.push({ x, y });
    angle += Math.PI / 3;
    x += Math.sin(angle) * size;
    y += Math.cos(angle) * size;
  }

  return corners;
}

/**
 * Get neighboring hexes
 */
export function getNeighbors(coord: AxialCoord): AxialCoord[] {
  const directions = [
    { q: 1, r: 0 },
    { q: 1, r: -1 },
    { q: 0, r: -1 },
    { q: -1, r: 0 },
    { q: -1, r: 1 },
    { q: 0, r: 1 },
  ];

  return directions.map(dir => ({
    q: coord.q + dir.q,
    r: coord.r + dir.r,
  }));
}

/**
 * Calculate distance between two hexes
 */
export function distance(a: AxialCoord, b: AxialCoord): number {
  return (
    Math.abs(a.q - b.q) +
    Math.abs(a.q + a.r - b.q - b.r) +
    Math.abs(a.r - b.r)
  ) / 2;
}

/**
 * Convert coordinate to string key for maps/sets
 */
export function coordToKey(coord: AxialCoord): string {
  return `${String(coord.q)},${String(coord.r)}`;
}

/**
 * Parse coordinate from string key
 */
export function keyToCoord(key: string): AxialCoord {
  const [q, r] = key.split(',').map(Number);
  return { q, r };
}
