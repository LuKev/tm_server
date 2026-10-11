import { type FAVOR_TILES, getCultColorClass } from '../../data/favorTiles';
import '../GameBoard/FavorTiles.css';

// One bounded face for the supply and owned tiles, including narrow panels.
export function FavorTileFace({ tile, stackCount = 1, used = false }: {
    tile: (typeof FAVOR_TILES)[number]; stackCount?: number; used?: boolean;
}): React.ReactElement {
    return <div className="favor-tile-face" data-favor={tile.id}>
        {Array.from({ length: stackCount }, (_, index) => <svg key={index} className="favor-tile-bg" viewBox="0 0 200 100" aria-hidden="true" style={{ transform: `translateY(-${String(index * 3)}px)`, zIndex: -index }}>
            <ellipse cx="100" cy="50" rx="97" ry="47" fill="white" stroke="#cdd1d4" strokeWidth="3" />
        </svg>)}
        <div className={`favor-tile-content ${tile.reward ? '' : 'favor-steps-only'}`}>
            <div className="favor-cult-steps">
                {Array.from({ length: tile.steps }, (_, index) => <div key={index} className={`cult-step-circle ${getCultColorClass(tile.cult)}`} />)}
            </div>
            {tile.reward && <div className="favor-reward">
                <div className="favor-reward-artwork">{tile.reward}
                    {used && <svg className="favor-used" viewBox="-2 -2 44 44" aria-hidden="true">
                        <path d="M12 0 H28 L40 12 V28 L28 40 H12 L0 28 V12 Z" fill="#d6d3d1" stroke="#78716c" strokeWidth="2" fillOpacity="0.9" />
                        <path d="M10 10 L30 30 M30 10 L10 30" stroke="#78716c" strokeWidth="3" />
                    </svg>}
                </div>
            </div>}
        </div>
    </div>;
}
