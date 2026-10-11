import { TOWN_FACE } from './townTileGeometry';
import { TOWN_TILES } from '../../data/townTiles';
import { type TownTileId } from '../../types/game.types';
import { CoinIcon, CultRhombusIcon, PowerIcon, PriestIcon, ShippingIcon, WorkerIcon } from './Icons';

export function TownTileFace({ tileId, borderColor = '#252b32' }: { tileId: TownTileId; borderColor?: string }): React.ReactElement {
    const tile = TOWN_TILES[tileId];
    const size = { width: '100%', height: '100%' };
    const reward = (() => {
        switch (tile.reward) {
            case 'coin': return <CoinIcon style={size}>{tile.amount}</CoinIcon>;
            case 'power': return <PowerIcon style={size} amount={tile.amount} />;
            case 'worker': return <WorkerIcon style={size}>{tile.amount}</WorkerIcon>;
            case 'priest': return <PriestIcon style={size} />;
            case 'shipping': return <ShippingIcon style={size} />;
            case 'cult': return <CultRhombusIcon style={size} showNumber={tile.amount === 2} />;
            case 'none': return null;
        }
    })();
    return <svg data-testid={`town-face-${String(tileId)}`} viewBox={`0 0 ${String(TOWN_FACE.width)} ${String(TOWN_FACE.height)}`} fontFamily="sans-serif" aria-hidden="true" style={{ width: '100%', height: '100%', display: 'block' }}>
        <path d={TOWN_FACE.outline} fill="white" stroke={borderColor} strokeWidth="3" />
        <text x={TOWN_FACE.vpX} y={reward ? TOWN_FACE.vpY : TOWN_FACE.vpOnlyY} textAnchor="middle" dominantBaseline="middle" fontWeight="700" fill="#252b32">
            <tspan fontSize={TOWN_FACE.vpSize}>{tile.vp}</tspan><tspan fontSize={TOWN_FACE.labelSize}> VP</tspan>
        </text>
        {reward && <svg x={TOWN_FACE.rewardX} y={TOWN_FACE.rewardY} width={TOWN_FACE.rewardSize} height={TOWN_FACE.rewardSize} viewBox="0 0 30 30">{reward}</svg>}
    </svg>;
}
