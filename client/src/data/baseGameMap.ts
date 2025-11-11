// Base game map layout - 113 hexes total
// Matches server/internal/game/terrain_layout.go

import { TerrainType } from '../types/game.types';
import type { AxialCoord } from '../utils/hexUtils';

export interface MapHexData {
  coord: AxialCoord;
  terrain: TerrainType;
  isRiver: boolean;
}

// Note: Backend uses (q, r) where q is column offset and r is row
// River hexes have terrain type but are marked as isRiver
export const BASE_GAME_MAP: MapHexData[] = [
  // Row 0 (13 hexes) - Top row
  { coord: { q: 0, r: 0 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 1, r: 0 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 2, r: 0 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 3, r: 0 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 4, r: 0 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 5, r: 0 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 6, r: 0 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 7, r: 0 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 8, r: 0 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 9, r: 0 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 10, r: 0 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 11, r: 0 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 12, r: 0 }, terrain: TerrainType.Swamp, isRiver: false },
  
  // Row 1 (12 hexes)
  { coord: { q: 0, r: 1 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 1, r: 1 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 2, r: 1 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 3, r: 1 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 4, r: 1 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 5, r: 1 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 6, r: 1 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 7, r: 1 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 8, r: 1 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 9, r: 1 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 10, r: 1 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 11, r: 1 }, terrain: TerrainType.Desert, isRiver: false },
  
  // Row 2 (13 hexes)
  { coord: { q: -1, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 0, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 1, r: 2 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 2, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 3, r: 2 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 4, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 5, r: 2 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 6, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 7, r: 2 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 8, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 9, r: 2 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 10, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 11, r: 2 }, terrain: TerrainType.Lake, isRiver: true }, // River
  
  // Row 3 (12 hexes)
  { coord: { q: -1, r: 3 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 0, r: 3 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 1, r: 3 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 2, r: 3 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 3, r: 3 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 4, r: 3 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 5, r: 3 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 6, r: 3 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 7, r: 3 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 8, r: 3 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 9, r: 3 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 10, r: 3 }, terrain: TerrainType.Plains, isRiver: false },
  
  // Row 4 (13 hexes)
  { coord: { q: -2, r: 4 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: -1, r: 4 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 0, r: 4 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 1, r: 4 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 2, r: 4 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 3, r: 4 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 4, r: 4 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 5, r: 4 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 6, r: 4 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 7, r: 4 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 8, r: 4 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 9, r: 4 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 10, r: 4 }, terrain: TerrainType.Lake, isRiver: false },
  
  // Row 5 (12 hexes)
  { coord: { q: -2, r: 5 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: -1, r: 5 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 0, r: 5 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 1, r: 5 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 2, r: 5 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 3, r: 5 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 4, r: 5 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 5, r: 5 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 6, r: 5 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 7, r: 5 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 8, r: 5 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 9, r: 5 }, terrain: TerrainType.Plains, isRiver: false },
  
  // Row 6 (13 hexes)
  { coord: { q: -3, r: 6 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: -2, r: 6 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: -1, r: 6 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 0, r: 6 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 1, r: 6 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 2, r: 6 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 3, r: 6 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 4, r: 6 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 5, r: 6 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 6, r: 6 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 7, r: 6 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 8, r: 6 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 9, r: 6 }, terrain: TerrainType.Desert, isRiver: false },
  
  // Row 7 (12 hexes)
  { coord: { q: -3, r: 7 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: -2, r: 7 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: -1, r: 7 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 0, r: 7 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 1, r: 7 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 2, r: 7 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 3, r: 7 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 4, r: 7 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: 5, r: 7 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 6, r: 7 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 7, r: 7 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 8, r: 7 }, terrain: TerrainType.Mountain, isRiver: false },
  
  // Row 8 (13 hexes) - Bottom row
  { coord: { q: -4, r: 8 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: -3, r: 8 }, terrain: TerrainType.Swamp, isRiver: false },
  { coord: { q: -2, r: 8 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: -1, r: 8 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 0, r: 8 }, terrain: TerrainType.Wasteland, isRiver: false },
  { coord: { q: 1, r: 8 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 2, r: 8 }, terrain: TerrainType.Desert, isRiver: false },
  { coord: { q: 3, r: 8 }, terrain: TerrainType.Plains, isRiver: false },
  { coord: { q: 4, r: 8 }, terrain: TerrainType.Mountain, isRiver: false },
  { coord: { q: 5, r: 8 }, terrain: TerrainType.Lake, isRiver: true }, // River
  { coord: { q: 6, r: 8 }, terrain: TerrainType.Lake, isRiver: false },
  { coord: { q: 7, r: 8 }, terrain: TerrainType.Forest, isRiver: false },
  { coord: { q: 8, r: 8 }, terrain: TerrainType.Wasteland, isRiver: false },
];

// Helper to get map as a dictionary keyed by "q,r"
export function getMapDictionary(): Map<string, MapHexData> {
  const dict = new Map<string, MapHexData>();
  BASE_GAME_MAP.forEach(hex => {
    const key = `${hex.coord.q},${hex.coord.r}`;
    dict.set(key, hex);
  });
  return dict;
}
