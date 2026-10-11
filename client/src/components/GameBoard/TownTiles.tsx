import React from 'react';
import { type TownTileId } from '../../types/game.types';
import './TownTiles.css';
import { TOWN_TILES } from '../../data/townTiles';
import { TownTileFace } from '../shared/TownTileFace';

interface TownTilesProps {
    availableTiles?: number[]; // List of available TownTileIds (can include duplicates)
    onTileClick?: (tile: TownTileId) => void;
    isTileClickable?: (tile: TownTileId, count: number) => boolean;
}

export const TownTiles: React.FC<TownTilesProps> = ({ availableTiles, onTileClick, isTileClickable }) => {
    // Default: 2 of tiles 0-5, 1 of tiles 6 and 7
    const defaultTiles = [0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 7];
    const tiles = availableTiles ?? defaultTiles;

    // Group tiles by ID
    const tileCounts = tiles.reduce<Record<number, number>>((acc, id) => {
        acc[id] = (acc[id] || 0) + 1;
        return acc;
    }, {});

    // Only show slots that have tiles
    const filledSlots = Object.keys(tileCounts).map(Number).sort((a, b) => a - b);

    return (
        <div className="tt-resize-container">
            <div className="town-tiles-container" data-testid="town-tiles">
                {filledSlots.map((id) => {
                    const count = tileCounts[id] || 0;
                    const config = TOWN_TILES[id as TownTileId];

                    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
                    if (!config) return null;

                    const clickable = isTileClickable ? isTileClickable(id as TownTileId, count) : true;

                    return (
                        <button
                            type="button"
                            key={id}
                            data-testid={`town-tile-${String(id)}`}
                            aria-label={`${String(config.vp)} victory point town tile, ${config.description}, ${String(count)} available`}
                            className={`game-tile town-tile-slot ${clickable ? 'town-tile-slot-selectable' : ''}`}
                            onClick={() => { onTileClick?.(id as TownTileId); }}
                            disabled={isTileClickable ? !clickable : false}
                            style={{
                                cursor: onTileClick ? (clickable ? 'pointer' : 'not-allowed') : 'default',

                                background: 'transparent',
                                border: 'none',
                                padding: 0,
                            }}
                        >
                            {/* Render stack */}
                            {Array.from({ length: Math.min(count, 3) }).map((_, index) => (
                                <div
                                    key={index}
                                    className={`town-tile town-tile-stack-${String(index)}`}
                                >
                                    <TownTileFace tileId={id as TownTileId} />
                                </div>
                            ))}
                        </button>
                    );
                })}
            </div>
        </div>
    );
};
