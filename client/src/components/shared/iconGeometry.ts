// One pawn silhouette for SVG icons and the canvas cult/town artwork.
export const PRIEST_PAWN_PATH = 'M10 2 A3 3 0 1 1 10 8 A3 3 0 1 1 10 2 Z M7 9 L4 18 H16 L13 9 Z';

export function drawPriestPawn(ctx: CanvasRenderingContext2D, x: number, y: number, size: number, color: string): void {
    ctx.save();
    ctx.translate(x - size / 2, y - size / 2);
    ctx.scale(size / 20, size / 20);
    ctx.fillStyle = color;
    ctx.strokeStyle = '#404040';
    ctx.lineWidth = 1.2;
    const path = new Path2D(PRIEST_PAWN_PATH);
    ctx.fill(path);
    ctx.stroke(path);
    ctx.restore();
}
