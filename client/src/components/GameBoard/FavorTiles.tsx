import React from 'react';
import { useGameStore } from '../../stores/gameStore';
import { FAVOR_TILES, getCultColorClass } from '../../data/favorTiles';
import './FavorTiles.css';

export const FavorTiles: React.FC = () => {
    const gameState = useGameStore(state => state.gameState);
    const available = gameState?.favorTiles?.available || {};

    return (
        <div className="favor-tiles-container">
            {FAVOR_TILES.map((tile) => {
                // Use available count if present, otherwise fallback to initialCount (for setup/loading)
                const count = available[tile.type] !== undefined ? available[tile.type] : tile.initialCount;
                const stackHeight = Math.min(count, 3);

                if (count === 0) {
                    return <div key={tile.id} className="favor-tile opacity-0" />;
                }

                return (
                    <div key={tile.id} className="favor-tile">
                        {/* Render underlying stack layers (if any) */}
                        {Array.from({ length: stackHeight - 1 }).map((_, index) => {
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
                                </div>
                            )}
                            {!tile.reward && (
                                <div className="absolute top-1 right-1 bg-gray-800 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center border border-white">
                                    {count}
                                </div>
                            )}
                        </div>
                    </div>
                );
            })}
        </div>
    );
};
