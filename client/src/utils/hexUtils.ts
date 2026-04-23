// Hex coordinate utilities - TypeScript version of terra-mystica/stc/game.js

export interface AxialCoord {
  q: number;
  r: number;
}

interface PixelCoord {
  x: number;
  y: number;
}

interface DisplayCoordSource {
  coord: AxialCoord
  isRiver: boolean
  displayCoord?: string
}

export function buildDisplayCoordinateMap(hexes: DisplayCoordSource[]): Map<string, string> {
  const byRow = new Map<number, number[]>()
  const explicitLabels = new Map<string, string>()
  hexes.forEach((hex) => {
    if (hex.isRiver) return
    if (hex.displayCoord) {
      explicitLabels.set(`${String(hex.coord.q)},${String(hex.coord.r)}`, hex.displayCoord)
    }
    const qValues = byRow.get(hex.coord.r)
    if (qValues) {
      qValues.push(hex.coord.q)
      return
    }
    byRow.set(hex.coord.r, [hex.coord.q])
  })

  if (explicitLabels.size > 0) {
    return explicitLabels
  }

  const labels = new Map<string, string>()
  byRow.forEach((qValues, row) => {
    qValues
      .sort((left, right) => left - right)
      .forEach((q, landIndex) => {
        labels.set(`${String(q)},${String(row)}`, `${String.fromCharCode(65 + row)}${String(landIndex + 1)}`)
      })
  })

  return labels
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

export function getDisplayCoordinate(coord: AxialCoord, labels?: Map<string, string>): string | null {
  return labels?.get(`${String(coord.q)},${String(coord.r)}`) ?? null
}

export function formatDisplayCoordinate(coord: AxialCoord, labels?: Map<string, string>): string {
  return getDisplayCoordinate(coord, labels) ?? `(${String(coord.q)}, ${String(coord.r)})`
}
