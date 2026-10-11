import { TownTileId } from '../types/game.types';

export type TownReward = 'coin' | 'power' | 'worker' | 'shipping' | 'cult' | 'priest' | 'none';
export interface TownTileData { vp: number; reward: TownReward; amount: number; description: string }
export const TOWN_TILES: Record<TownTileId, TownTileData> = {
    [TownTileId.Vp5Coins6]: { vp: 5, reward: 'coin', amount: 6, description: '6 coins' },
    [TownTileId.Vp6Power8]: { vp: 6, reward: 'power', amount: 8, description: '8 power' },
    [TownTileId.Vp7Workers2]: { vp: 7, reward: 'worker', amount: 2, description: '2 workers' },
    [TownTileId.Vp4Ship1]: { vp: 4, reward: 'shipping', amount: 1, description: '1 shipping advancement' },
    [TownTileId.Vp8Cult1]: { vp: 8, reward: 'cult', amount: 1, description: '1 step on all cult tracks' },
    [TownTileId.Vp9Priest1]: { vp: 9, reward: 'priest', amount: 1, description: '1 priest' },
    [TownTileId.Vp11]: { vp: 11, reward: 'none', amount: 0, description: 'no extra reward' },
    [TownTileId.Vp2Cult2]: { vp: 2, reward: 'cult', amount: 2, description: '2 steps on all cult tracks' },
};
