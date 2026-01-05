import React from 'react';
import { CultType, FavorTileType } from '../types/game.types';
import { CoinIcon, WorkerIcon, PowerIcon, DwellingIcon, TradingHouseIcon, CultActionIcon } from '../components/shared/Icons';

export interface FavorTileData {
    id: string;
    type: FavorTileType;
    cult: CultType;
    steps: 1 | 2 | 3;
    reward: React.ReactNode | null;
    initialCount: number;
}

export const FAVOR_TILES: FavorTileData[] = [
    // Row 1: Level 3s
    { id: 'fav_fire_3', type: FavorTileType.Fire3, cult: CultType.Fire, steps: 3, reward: null, initialCount: 1 },
    { id: 'fav_water_3', type: FavorTileType.Water3, cult: CultType.Water, steps: 3, reward: null, initialCount: 1 },
    { id: 'fav_earth_3', type: FavorTileType.Earth3, cult: CultType.Earth, steps: 3, reward: null, initialCount: 1 },
    { id: 'fav_air_3', type: FavorTileType.Air3, cult: CultType.Air, steps: 3, reward: null, initialCount: 1 },

    // Row 2: Level 2s
    { id: 'fav_fire_2', type: FavorTileType.Fire2, cult: CultType.Fire, steps: 2, reward: <div style={{ lineHeight: '1.1' }}>Town<br />Size 6</div>, initialCount: 3 },
    { id: 'fav_water_2', type: FavorTileType.Water2, cult: CultType.Water, steps: 2, reward: <CultActionIcon className="favor-icon" style={{ width: '1.5em', height: '1.5em' }} />, initialCount: 3 },
    { id: 'fav_earth_2', type: FavorTileType.Earth2, cult: CultType.Earth, steps: 2, reward: <div className="flex items-center gap-1"><WorkerIcon className="favor-icon" style={{ width: '1em', height: '1em' }}>1</WorkerIcon><PowerIcon amount={1} className="favor-icon" style={{ width: '1em', height: '1em' }} /></div>, initialCount: 3 },
    { id: 'fav_air_2', type: FavorTileType.Air2, cult: CultType.Air, steps: 2, reward: <PowerIcon amount={4} className="favor-icon" style={{ width: '1em', height: '1em' }} />, initialCount: 3 },

    // Row 3: Level 1s
    { id: 'fav_fire_1', type: FavorTileType.Fire1, cult: CultType.Fire, steps: 1, reward: <CoinIcon className="favor-icon" style={{ width: '1em', height: '1em' }}>3</CoinIcon>, initialCount: 3 },
    { id: 'fav_water_1', type: FavorTileType.Water1, cult: CultType.Water, steps: 1, reward: <div className="flex items-center gap-1"><TradingHouseIcon className="favor-icon" style={{ width: '1em', height: '1em' }} /><span>→</span><span>3</span></div>, initialCount: 3 },
    { id: 'fav_earth_1', type: FavorTileType.Earth1, cult: CultType.Earth, steps: 1, reward: <div className="flex items-center gap-1"><DwellingIcon className="favor-icon" style={{ width: '1em', height: '1em' }} /><span>→</span><span>2</span></div>, initialCount: 3 },
    { id: 'fav_air_1', type: FavorTileType.Air1, cult: CultType.Air, steps: 1, reward: <div className="flex items-center gap-1" style={{ fontSize: '0.8em' }}><span className="font-bold">Pass</span><span>→</span><TradingHouseIcon className="favor-icon" style={{ width: '1em', height: '1em' }} /><span>→</span><span>2-4</span></div>, initialCount: 3 },
];

export const getCultColorClass = (cult: CultType): string => {
    switch (cult) {
        case CultType.Fire: return 'bg-fire';
        case CultType.Water: return 'bg-water';
        case CultType.Earth: return 'bg-earth';
        case CultType.Air: return 'bg-air';
    }
};
