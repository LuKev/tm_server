import { type TownTileData } from '../../data/townTiles';
import { drawPriestPawn } from './iconGeometry';

// Draw into the same 30 x 30 reward viewport used by TownTileFace's SVG icons.
export function drawTownReward(ctx: CanvasRenderingContext2D, tile: TownTileData): void {
    ctx.save();
    ctx.lineWidth = 2.5;
    ctx.strokeStyle = '#5c4033';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.font = '700 16px sans-serif';
    switch (tile.reward) {
        case 'coin':
        case 'power':
            ctx.beginPath(); ctx.arc(15, 15, 12, 0, Math.PI * 2);
            ctx.fillStyle = tile.reward === 'coin' ? '#fbbf24' : '#7c3aed';
            ctx.strokeStyle = tile.reward === 'coin' ? '#d97706' : '#5c4033';
            ctx.lineWidth = tile.reward === 'coin' ? 2.5 : 2;
            ctx.fill(); ctx.stroke();
            ctx.fillStyle = tile.reward === 'coin' ? '#5c4033' : '#fff';
            ctx.fillText(String(tile.amount), 15, 15);
            break;
        case 'worker':
            ctx.fillStyle = '#fff'; ctx.fillRect(3, 3, 24, 24); ctx.strokeRect(3, 3, 24, 24);
            ctx.fillStyle = '#5c4033'; ctx.fillText(String(tile.amount), 15, 15);
            break;
        case 'priest': drawPriestPawn(ctx, 15, 15, 30, '#a0a0a0'); break;
        case 'cult':
            // The SVG's 40 x 34 viewBox is centered in the square viewport.
            ctx.translate(0, 2.25); ctx.scale(0.75, 0.75);
            ctx.lineWidth = 1; ctx.strokeStyle = '#333';
            [[20, 7, '#e5e7eb'], [8, 17, '#92400e'], [32, 17, '#3b82f6'], [20, 27, '#ef4444']].forEach(([cx, cy, color]) => {
                ctx.beginPath(); ctx.arc(Number(cx), Number(cy), 6, 0, Math.PI * 2);
                ctx.fillStyle = String(color); ctx.fill(); ctx.stroke();
            });
            if (tile.amount === 2) { ctx.fillStyle = '#333'; ctx.font = '700 12px sans-serif'; ctx.fillText('2', 20, 17); }
            break;
        case 'shipping':
            ctx.scale(30 / 24, 30 / 24);
            ctx.lineWidth = 2; ctx.lineCap = 'round'; ctx.lineJoin = 'round';
            ctx.stroke(new Path2D('M10 2v15 M7 22a4 4 0 0 1-4-4 1 1 0 0 1 1-1h16a1 1 0 0 1 1 1 4 4 0 0 1-4 4z M9.159 2.46a1 1 0 0 1 1.521-.193l9.977 8.98A1 1 0 0 1 20 13H4a1 1 0 0 1-.824-1.567z'));
            break;
        case 'none': break;
    }
    ctx.restore();
}
