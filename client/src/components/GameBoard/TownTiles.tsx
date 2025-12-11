import React from 'react';
import { TownTileId } from '../../types/game.types';
import './TownTiles.css';
import {
    PriestIcon,
    PowerIcon,
    WorkerIcon,
    CoinIcon,
    CultRhombusIcon
} from '../shared/Icons';

// Config for the 8 distinct town tiles
interface TownTileConfig {
    id: TownTileId;
    vp: number;
    rewards: React.ReactNode;
}

const TOWN_TILE_CONFIGS: Record<TownTileId, TownTileConfig> = {
    [TownTileId.Vp5Coins6]: {
        id: TownTileId.Vp5Coins6,
        vp: 5,
        rewards: <CoinIcon className="icon-md">6</CoinIcon>
    },
    [TownTileId.Vp7Workers2]: {
        id: TownTileId.Vp7Workers2,
        vp: 7,
        rewards: <WorkerIcon className="icon-md">2</WorkerIcon>
    },
    [TownTileId.Vp9Priest1]: {
        id: TownTileId.Vp9Priest1,
        vp: 9,
        rewards: <PriestIcon className="icon-md" />
    },
    [TownTileId.Vp6Power8]: {
        id: TownTileId.Vp6Power8,
        vp: 6,
        rewards: <PowerIcon amount={8} className="icon-md" />
    },
    [TownTileId.Vp8Cult1]: {
        id: TownTileId.Vp8Cult1,
        vp: 8,
        rewards: <CultRhombusIcon className="icon-md" />
    },
    // Mini expansion tiles
    [TownTileId.Vp2Ship1]: {
        id: TownTileId.Vp2Ship1,
        vp: 2,
        rewards: <CultRhombusIcon className="icon-md" showNumber={true} />
    },
    [TownTileId.Vp4Carpet1]: {
        id: TownTileId.Vp4Carpet1,
        vp: 4,
        rewards: <div className="reward-container"><span className="reward-text">Ship/Carpet</span></div>
    },
    [TownTileId.Vp11]: {
        id: TownTileId.Vp11,
        vp: 11,
        rewards: null
    },
};

interface TownTilesProps {
    availableTiles?: number[]; // List of available TownTileIds (can include duplicates)
}

export const TownTiles: React.FC<TownTilesProps> = ({ availableTiles }) => {
    // Default: 2 of tiles 0-4 and 6, 1 of tiles 5 and 7
    const defaultTiles = [0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 6, 6, 7];
    const tiles = availableTiles && availableTiles.length > 0 ? availableTiles : defaultTiles;

    // Group tiles by ID
    const tileCounts = tiles.reduce((acc, id) => {
        acc[id] = (acc[id] || 0) + 1;
        return acc;
    }, {} as Record<number, number>);

    // Only show slots that have tiles
    const filledSlots = Object.keys(tileCounts).map(Number).sort((a, b) => a - b);

    return (
        <div className="town-tiles-container">
            {filledSlots.map((id) => {
                const count = tileCounts[id] || 0;
                const config = TOWN_TILE_CONFIGS[id as TownTileId];

                if (!config || count === 0) return null;

                return (
                    <div key={id} className="town-tile-slot">
                        {/* Render stack */}
                        {Array.from({ length: Math.min(count, 3) }).map((_, index) => (
                            <div
                                key={index}
                                className={`town-tile town-tile-stack-${index}`}
                            >
                                <div className="town-tile-content">
                                    <div className="town-tile-top">
                                        <span className="vp-value">{config.vp}</span>
                                        <span className="vp-label">VP</span>
                                    </div>
                                    <div className="town-tile-bottom">
                                        {config.rewards}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                );
            })}
        </div>
    );
};
