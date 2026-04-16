import { TerrainType } from './game.types'
import { TownTileId } from './game.types'
import type { AxialCoord } from '../utils/hexUtils'

export interface MapHexData {
  coord: AxialCoord
  terrain: TerrainType
  isRiver: boolean
  displayCoord?: string
  hasTownTile?: boolean
  townTileType?: TownTileId
  townTileOwnerPlayerId?: string
  powerTokenOwnerPlayerId?: string
}

export interface CustomMapDefinition {
  name?: string
  rowCount: number
  firstRowColumns: number
  firstRowLonger: boolean
  rows: TerrainType[][]
}

export interface MapSummary {
  id: string
  name: string
}
