import { type TownTileData } from '../../data/townTiles';
import { drawTownReward } from './drawTownReward';

// Supply, owned, tooltip, and canvas tiles use the same face proportions.
export const TOWN_FACE = {
    width: 100, height: 115,
    outline: 'M50 3 L97 29 L97 86 L50 112 L3 86 L3 29 Z',
    vpX: 50, vpY: 36, vpOnlyY: 58, vpSize: 23, labelSize: 10,
    rewardX: 29, rewardY: 57, rewardSize: 42,
} as const;

export function drawTownTileFace(ctx: CanvasRenderingContext2D, tile: TownTileData, x: number, y: number, height: number): void {
    const scale = height / TOWN_FACE.height;
    ctx.save();
    ctx.translate(x - TOWN_FACE.width * scale / 2, y - height / 2);
    ctx.scale(scale, scale);
    const outline = new Path2D(TOWN_FACE.outline);
    ctx.fillStyle = 'white'; ctx.fill(outline);
    ctx.strokeStyle = '#252b32'; ctx.lineWidth = 3; ctx.stroke(outline);
    ctx.fillStyle = '#252b32';
    ctx.textAlign = 'left'; ctx.textBaseline = 'middle';
    ctx.font = `700 ${String(TOWN_FACE.vpSize)}px sans-serif`;
    const vp = String(tile.vp);
    const vpWidth = ctx.measureText(vp).width;
    ctx.font = `700 ${String(TOWN_FACE.labelSize)}px sans-serif`;
    const labelWidth = ctx.measureText(' VP').width;
    const start = TOWN_FACE.vpX - (vpWidth + labelWidth) / 2;
    const baseline = tile.reward === 'none' ? TOWN_FACE.vpOnlyY : TOWN_FACE.vpY;
    ctx.font = `700 ${String(TOWN_FACE.vpSize)}px sans-serif`;
    ctx.fillText(vp, start, baseline);
    ctx.font = `700 ${String(TOWN_FACE.labelSize)}px sans-serif`;
    ctx.fillText(' VP', start + vpWidth, baseline);
    ctx.translate(TOWN_FACE.rewardX, TOWN_FACE.rewardY);
    ctx.scale(TOWN_FACE.rewardSize / 30, TOWN_FACE.rewardSize / 30);
    drawTownReward(ctx, tile);
    ctx.restore();
}
