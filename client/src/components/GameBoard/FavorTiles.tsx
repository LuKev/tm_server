import React from 'react';
import { CultType } from '../../types/game.types';
import { CoinIcon, WorkerIcon, PowerIcon, DwellingIcon, TradingHouseIcon, CultActionIcon } from '../shared/Icons';
import './FavorTiles.css';

interface FavorTileData {
    id: string;
    cult: CultType;
    steps: 1 | 2 | 3;
    reward: React.ReactNode | null;
    initialCount: number;
}

const FAVOR_TILES: FavorTileData[] = [
    // Row 1: Level 3s
    { id: 'fav_fire_3', cult: CultType.Fire, steps: 3, reward: null, initialCount: 1 },
    { id: 'fav_water_3', cult: CultType.Water, steps: 3, reward: null, initialCount: 1 },
    { id: 'fav_earth_3', cult: CultType.Earth, steps: 3, reward: null, initialCount: 1 },
    { id: 'fav_air_3', cult: CultType.Air, steps: 3, reward: null, initialCount: 1 },

    // Row 2: Level 2s
    { id: 'fav_fire_2', cult: CultType.Fire, steps: 2, reward: <div style={{ lineHeight: '1.1' }}>Town<br />Size 6</div>, initialCount: 3 },
    { id: 'fav_water_2', cult: CultType.Water, steps: 2, reward: <CultActionIcon className="favor-icon" style={{ fontSize: '1.5em' }} />, initialCount: 3 },
    { id: 'fav_earth_2', cult: CultType.Earth, steps: 2, reward: <div className="flex items-center gap-1"><WorkerIcon className="favor-icon">1</WorkerIcon><PowerIcon amount={1} className="favor-icon" /></div>, initialCount: 3 },
    { id: 'fav_air_2', cult: CultType.Air, steps: 2, reward: <PowerIcon amount={4} className="favor-icon" />, initialCount: 3 },

    // Row 3: Level 1s
    { id: 'fav_fire_1', cult: CultType.Fire, steps: 1, reward: <CoinIcon className="favor-icon">3</CoinIcon>, initialCount: 3 },
    { id: 'fav_water_1', cult: CultType.Water, steps: 1, reward: <div className="flex items-center gap-1"><TradingHouseIcon className="favor-icon" /><span>→</span><span>3</span></div>, initialCount: 3 },
    { id: 'fav_earth_1', cult: CultType.Earth, steps: 1, reward: <div className="flex items-center gap-1"><DwellingIcon className="favor-icon" /><span>→</span><span>2</span></div>, initialCount: 3 },
    { id: 'fav_air_1', cult: CultType.Air, steps: 1, reward: <div className="flex items-center gap-1" style={{ fontSize: '0.8em' }}><span className="font-bold">Pass</span><span>→</span><TradingHouseIcon className="favor-icon" /><span>→</span><span>2-4</span></div>, initialCount: 3 },
];

const getCultColorClass = (cult: CultType): string => {
    switch (cult) {
        case CultType.Fire: return 'bg-fire';
        case CultType.Water: return 'bg-water';
        case CultType.Earth: return 'bg-earth';
        case CultType.Air: return 'bg-air';
    }
};

export const FavorTiles: React.FC = () => {
    return (
        <div className="favor-tiles-container">
            {FAVOR_TILES.map((tile) => {
                const count = tile.initialCount;
                const stackHeight = Math.min(count, 3);

                if (count === 0) {
                    return <div key={tile.id} className="favor-tile opacity-0" />;
                }

                return (
                    <div key={tile.id} className="favor-tile">
                        {/* Render underlying stack layers (if any) */}
                        {Array.from({ length: stackHeight - 1 }).map((_, index) => {
                            // index 0 is the one just below top. index 1 is below that?
                            // Let's say we want:
                            // Top: 0,0
                            // Below 1: 2px, 2px
                            // Below 2: 4px, 4px

                            // If stackHeight is 3:
                            // We need offsets 4px and 2px.
                            // Let's iterate from bottom up?
                            // Or just map 0..stackHeight-2

                            // Let's do:
                            // i=0 -> offset = (stackHeight - 1 - i) * 2
                            // If stackHeight=3:
                            // i=0 -> offset 4. (Bottom)
                            // i=1 -> offset 2. (Middle)
                            // Top is separate.

                            const offset = (stackHeight - 1 - index) * 3;

                            return (
                                <svg
                                    key={`bg-${String(index)}`}
                                    className="favor-tile-bg"
                                    viewBox="0 0 200 100"
                                    preserveAspectRatio="xMidYMid meet"
                                    style={{
                                        // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
                                        transform: `translate(0px, -${offset}cqw)`,
                                        zIndex: 0 // Keep them at base level
                                    }}
                                >
                                    <ellipse cx="100" cy="50" rx="98" ry="48"
                                        className={`${getCultColorClass(tile.cult)} opacity-20`}
                                        stroke="currentColor"
                                        strokeWidth="3"
                                    />
                                    <ellipse cx="100" cy="50" rx="95" ry="45"
                                        className="fill-white"
                                        fill="white"
                                    />
                                </svg>
                            );
                        })}

                        {/* Top Layer Background */}
                        <svg className="favor-tile-bg" viewBox="0 0 200 100" preserveAspectRatio="xMidYMid meet" style={{ zIndex: 0 }}>
                            <ellipse cx="100" cy="50" rx="98" ry="48"
                                className={`${getCultColorClass(tile.cult)} opacity-20`}
                                stroke="currentColor"
                                strokeWidth="3"
                            />
                            <ellipse cx="100" cy="50" rx="95" ry="45"
                                className="fill-white"
                                fill="white"
                            />
                        </svg>

                        {/* Top Layer Content */}
                        <div className={`favor-tile-content ${!tile.reward ? 'justify-center' : ''}`}>
                            {/* Cult Steps */}
                            <div className={`favor-cult-steps ${!tile.reward ? 'w-full' : ''}`}>
                                {Array.from({ length: tile.steps }).map((_, i) => (
                                    <div
                                        key={i}
                                        className={`cult-step-circle ${getCultColorClass(tile.cult)}`}
                                    />
                                ))}
                            </div>

                            {/* Right: Reward (only if exists) */}
                            {tile.reward && (
                                <div className="favor-reward">
                                    {tile.reward}
                                    {/* Optional: Show count badge if needed in future */}
                                </div>
                            )}
                        </div>
                    </div>
                );
            })}
        </div>
    );
};
