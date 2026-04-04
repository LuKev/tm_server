import { TerrainType } from '../types/game.types'
import type { CustomMapDefinition, MapHexData } from '../types/map.types'

interface CustomMapSettings {
  name?: string
  rowCount?: number
  firstRowColumns?: number
  firstRowLonger?: boolean
}

interface TerrainBrushOption {
  terrain: TerrainType
  label: string
  importCode: string
}

export const TERRAIN_BRUSH_OPTIONS: TerrainBrushOption[] = [
  { terrain: TerrainType.River, label: 'River', importCode: 'I' },
  { terrain: TerrainType.Plains, label: 'Plains', importCode: 'U' },
  { terrain: TerrainType.Swamp, label: 'Swamp', importCode: 'K' },
  { terrain: TerrainType.Lake, label: 'Lake', importCode: 'B' },
  { terrain: TerrainType.Forest, label: 'Forest', importCode: 'G' },
  { terrain: TerrainType.Mountain, label: 'Mountain', importCode: 'S' },
  { terrain: TerrainType.Wasteland, label: 'Wasteland', importCode: 'R' },
  { terrain: TerrainType.Desert, label: 'Desert', importCode: 'Y' },
]

const IMPORT_TOKEN_MAP = new Map<string, TerrainType>([
  ['i', TerrainType.River],
  ['river', TerrainType.River],
  ['u', TerrainType.Plains],
  ['plains', TerrainType.Plains],
  ['plain', TerrainType.Plains],
  ['brown', TerrainType.Plains],
  ['k', TerrainType.Swamp],
  ['swamp', TerrainType.Swamp],
  ['black', TerrainType.Swamp],
  ['b', TerrainType.Lake],
  ['lake', TerrainType.Lake],
  ['blue', TerrainType.Lake],
  ['g', TerrainType.Forest],
  ['forest', TerrainType.Forest],
  ['green', TerrainType.Forest],
  ['s', TerrainType.Mountain],
  ['mountain', TerrainType.Mountain],
  ['gray', TerrainType.Mountain],
  ['grey', TerrainType.Mountain],
  ['silver', TerrainType.Mountain],
  ['r', TerrainType.Wasteland],
  ['wasteland', TerrainType.Wasteland],
  ['red', TerrainType.Wasteland],
  ['y', TerrainType.Desert],
  ['desert', TerrainType.Desert],
  ['yellow', TerrainType.Desert],
])

function getRowLength(firstRowColumns: number, firstRowLonger: boolean, rowIndex: number): number {
  if (rowIndex % 2 === 0) {
    return firstRowColumns
  }
  return firstRowLonger ? firstRowColumns - 1 : firstRowColumns + 1
}

function getRowStartQ(firstRowLonger: boolean, rowIndex: number): number {
  if (firstRowLonger) {
    return rowIndex % 2 === 0 ? -(Math.floor(rowIndex / 2)) : -((rowIndex - 1) / 2)
  }
  return rowIndex % 2 === 0 ? -(Math.floor(rowIndex / 2)) : -((rowIndex + 1) / 2)
}

function buildRows(rowCount: number, firstRowColumns: number, firstRowLonger: boolean, previousRows?: TerrainType[][]): TerrainType[][] {
  const rows: TerrainType[][] = []
  for (let rowIndex = 0; rowIndex < rowCount; rowIndex += 1) {
    const rowLength = getRowLength(firstRowColumns, firstRowLonger, rowIndex)
    const previousRow = previousRows?.[rowIndex] ?? []
    rows.push(Array.from({ length: rowLength }, (_, colIndex) => previousRow[colIndex] ?? TerrainType.River))
  }
  return rows
}

export function createEmptyCustomMapDefinition(overrides: CustomMapSettings = {}): CustomMapDefinition {
  const rowCount = Math.max(1, Math.trunc(overrides.rowCount ?? 9))
  const firstRowColumns = Math.max(1, Math.trunc(overrides.firstRowColumns ?? 13))
  const firstRowLonger = overrides.firstRowLonger ?? true
  const safeFirstRowColumns = firstRowLonger && rowCount > 1 ? Math.max(2, firstRowColumns) : firstRowColumns

  return {
    name: overrides.name?.trim() || '',
    rowCount,
    firstRowColumns: safeFirstRowColumns,
    firstRowLonger,
    rows: buildRows(rowCount, safeFirstRowColumns, firstRowLonger),
  }
}

export function resizeCustomMapDefinition(definition: CustomMapDefinition, next: CustomMapSettings): CustomMapDefinition {
  const rowCount = Math.max(1, Math.trunc(next.rowCount ?? definition.rowCount))
  const firstRowLonger = next.firstRowLonger ?? definition.firstRowLonger
  const desiredColumns = Math.max(1, Math.trunc(next.firstRowColumns ?? definition.firstRowColumns))
  const firstRowColumns = firstRowLonger && rowCount > 1 ? Math.max(2, desiredColumns) : desiredColumns

  return {
    name: next.name ?? definition.name ?? '',
    rowCount,
    firstRowColumns,
    firstRowLonger,
    rows: buildRows(rowCount, firstRowColumns, firstRowLonger, definition.rows),
  }
}

export function buildCustomMapHexes(definition: CustomMapDefinition): MapHexData[] {
  const hexes: MapHexData[] = []
  for (let rowIndex = 0; rowIndex < definition.rowCount; rowIndex += 1) {
    const row = definition.rows[rowIndex] ?? []
    const startQ = getRowStartQ(definition.firstRowLonger, rowIndex)
    row.forEach((terrain, colIndex) => {
      hexes.push({
        coord: { q: startQ + colIndex, r: rowIndex },
        terrain,
        isRiver: terrain === TerrainType.River,
      })
    })
  }
  return hexes
}

export function applyTerrainToHex(definition: CustomMapDefinition, q: number, r: number, terrain: TerrainType): CustomMapDefinition {
  const row = definition.rows[r]
  if (!row) return definition

  const colIndex = q - getRowStartQ(definition.firstRowLonger, r)
  if (colIndex < 0 || colIndex >= row.length) return definition

  const rows = definition.rows.map((existingRow, rowIndex) => {
    if (rowIndex !== r) return [...existingRow]
    const nextRow = [...existingRow]
    nextRow[colIndex] = terrain
    return nextRow
  })

  return { ...definition, rows }
}

function parseTerrainToken(rawToken: string): TerrainType {
  const token = rawToken.trim().toLowerCase()
  const terrain = IMPORT_TOKEN_MAP.get(token)
  if (terrain === undefined) {
    throw new Error(`Unsupported terrain token "${rawToken.trim()}".`)
  }
  return terrain
}

export function parseCustomMapDefinition(text: string): CustomMapDefinition {
  const rows = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line !== '')
    .map((line) => line.split(',').map(parseTerrainToken))

  if (rows.length === 0) {
    throw new Error('Paste at least one map row before importing.')
  }

  const rowCount = rows.length
  const firstRowColumns = rows[0].length
  const secondRowColumns = rows[1]?.length ?? firstRowColumns
  const firstRowLonger = rowCount === 1 ? true : firstRowColumns >= secondRowColumns

  if (rowCount > 1 && Math.abs(firstRowColumns - secondRowColumns) !== 1) {
    throw new Error('The first two rows must differ by exactly one column.')
  }

  rows.forEach((row, rowIndex) => {
    const expected = getRowLength(firstRowColumns, firstRowLonger, rowIndex)
    if (row.length !== expected) {
      throw new Error(`Row ${rowIndex + 1} has ${row.length} columns, expected ${expected}.`)
    }
  })

  return {
    name: '',
    rowCount,
    firstRowColumns,
    firstRowLonger,
    rows,
  }
}

export function countLandHexes(definition: CustomMapDefinition): number {
  return definition.rows.reduce(
    (sum, row) => sum + row.filter((terrain) => terrain !== TerrainType.River).length,
    0,
  )
}
