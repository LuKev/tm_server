import { TerrainType } from './game.types'
import type { AxialCoord } from '../utils/hexUtils'

export interface MapHexData {
  coord: AxialCoord
  terrain: TerrainType
  isRiver: boolean
  displayCoord?: string
}

export interface MapSummary {
  id: string
  name: string
}
