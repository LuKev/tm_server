// Cult Tracks component - based on terra-mystica/stc/game.js drawCults()
import React, { useRef, useEffect } from 'react';
import { CultType, FactionType } from '../../types/game.types';
import { FACTION_COLORS, CULT_COLORS, getContrastColor } from '../../utils/colors';

interface CultPosition {
  faction: FactionType;
  position: number;
  hasKey: boolean; // Power key (shows hex instead of circle at position 10)
}

interface BonusTile {
  priests?: number;
  power?: number;
  faction?: FactionType; // For colored priest markers
}

interface CultTracksProps {
  cultPositions: Map<CultType, CultPosition[]>; // For each cult, list of faction positions
  bonusTiles?: Map<CultType, BonusTile[]>; // Bonus tiles at bottom
}

export const CultTracks: React.FC<CultTracksProps> = ({ cultPositions, bonusTiles }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  
  const cults = [CultType.Fire, CultType.Water, CultType.Earth, CultType.Air];
  const cultWidth = 250 / 4; // 62.5px per cult
  const height = 500;
  
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    ctx.save();
    ctx.scale(2, 2); // Scale for better rendering
    
    // Draw each cult track
    for (let j = 0; j < 4; j++) {
      const cult = cults[j];
      
      ctx.save();
      ctx.translate(cultWidth * j, 0);
      
      // Background
      ctx.beginPath();
      ctx.rect(0, 0, cultWidth, height);
      ctx.fillStyle = CULT_COLORS[cult as CultType];
      ctx.fill();
      
      ctx.translate(0, 20);
      
      // Track positions (10 down to 0)
      let seen10 = false;
      const positions = cultPositions.get(cult) || [];
      
      for (let i = 10; i >= 0; i--) {
        ctx.save();
        ctx.translate(0, (10 - i) * 40 + 20);
        
        // Draw markers for factions at this position (centered)
        const factionsAtPosition = positions.filter(p => p.position === i);
        if (factionsAtPosition.length > 0) {
          ctx.save();
          
          // If multiple factions, space them out horizontally
          const markerWidth = 12; // Width taken by each marker
          const totalWidth = factionsAtPosition.length * markerWidth;
          
          // Start position to center the group
          ctx.translate(cultWidth / 2 - totalWidth / 2 + markerWidth / 2, 0);
          
          factionsAtPosition.forEach(pos => {
            drawCultMarker(
              ctx,
              pos.faction,
              !seen10 && (i === 10 || pos.hasKey)
            );
            if (i === 10) {
              seen10 = true;
            }
            // Move to the right for next marker
            ctx.translate(markerWidth, 0);
          });
          
          ctx.restore();
        } else {
          // Show position number only if no factions at this position (centered vertically)
          ctx.fillStyle = '#000';
          ctx.strokeStyle = '#000';
          ctx.font = '15px Verdana';
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle'; // Center text vertically
          ctx.lineWidth = 0.1;
          ctx.fillText(String(i), cultWidth / 2, 10); // Center at same Y as circles
          ctx.strokeText(String(i), cultWidth / 2, 10);
        }
        
        ctx.restore();
      }
      
      // Bonus tiles at bottom
      ctx.save();
      ctx.translate(8, 470);
      ctx.lineWidth = 0.2;
      
      const tiles = bonusTiles?.get(cult) || [];
      for (let i = 0; i < 4; i++) {
        const tile = tiles[i];
        let text = i === 0 ? '3' : '2'; // Default power values
        let color = '#000';
        let font = '15px Verdana';
        
        if (tile) {
          if (tile.priests && tile.priests > 0 && tile.faction !== undefined) {
            text = 'P'; // Capital P for priest
            color = FACTION_COLORS[tile.faction];
            font = 'bold 15px Verdana'; // Bold font for priest
          } else if (tile.power) {
            text = String(tile.power);
          }
        }
        
        ctx.font = font;
        ctx.fillStyle = color;
        ctx.strokeStyle = color;
        ctx.fillText(text, 0, 0);
        ctx.strokeText(text, 0, 0);
        ctx.translate(12, 0);
      }
      
      ctx.restore();
      ctx.restore();
    }
    
    // Draw horizontal divider lines
    ctx.beginPath();
    ctx.strokeStyle = '#000';
    ctx.lineWidth = 1;
    ctx.translate(0, 60.5);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);
    ctx.moveTo(0, 3); ctx.lineTo(250, 3);
    ctx.moveTo(0, 6); ctx.lineTo(250, 6);
    
    ctx.translate(0, 120);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);
    ctx.moveTo(0, 3); ctx.lineTo(250, 3);
    
    ctx.translate(0, 80);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);
    ctx.moveTo(0, 3); ctx.lineTo(250, 3);
    
    ctx.translate(0, 80);
    ctx.moveTo(0, 0); ctx.lineTo(250, 0);
    
    ctx.stroke();
    ctx.restore();
  }, [cultPositions, bonusTiles]);
  
  // Draw a single cult marker (from terra-mystica/stc/game.js)
  const drawCultMarker = (
    ctx: CanvasRenderingContext2D,
    faction: FactionType,
    isHex: boolean
  ) => {
    const color = FACTION_COLORS[faction];
    const contrastColor = getContrastColor(color);
    
    ctx.save();
    ctx.beginPath();
    
    if (isHex) {
      // Draw hex shape for position 10 or with key
      drawHexPath(ctx, -8, 14, 8.5);
    } else {
      // Draw circle
      ctx.arc(0, 10, 8, 0, Math.PI * 2);
    }
    
    ctx.fillStyle = color;
    ctx.fill();
    ctx.strokeStyle = '#000';
    ctx.lineWidth = 1;
    ctx.stroke();
    ctx.restore();
    
    // Draw faction letter
    ctx.save();
    ctx.fillStyle = contrastColor;
    ctx.strokeStyle = contrastColor;
    ctx.textAlign = 'center';
    ctx.font = 'bold 10px Verdana';
    ctx.lineWidth = 0.1;
    
    const letter = getFactionLetter(faction);
    ctx.fillText(letter, -1, 14);
    ctx.strokeText(letter, -1, 14);
    ctx.restore();
  };
  
  // Draw hex path (simplified from makeHexPath)
  const drawHexPath = (ctx: CanvasRenderingContext2D, x: number, y: number, size: number) => {
    let angle = 0;
    ctx.moveTo(x, y);
    for (let i = 0; i < 6; i++) {
      ctx.lineTo(x, y);
      angle += Math.PI / 3;
      x += Math.sin(angle) * size;
      y += Math.cos(angle) * size;
    }
    ctx.closePath();
  };
  
  // Get first letter of faction (all uppercase)
  const getFactionLetter = (faction: FactionType): string => {
    const names: Record<FactionType, string> = {
      [FactionType.Nomads]: 'N',
      [FactionType.Fakirs]: 'F',
      [FactionType.ChaosMagicians]: 'C',
      [FactionType.Giants]: 'G',
      [FactionType.Swarmlings]: 'S',
      [FactionType.Mermaids]: 'M',
      [FactionType.Witches]: 'W',
      [FactionType.Auren]: 'A',
      [FactionType.Halflings]: 'H',
      [FactionType.Cultists]: 'C',
      [FactionType.Alchemists]: 'A',
      [FactionType.Darklings]: 'D',
      [FactionType.Engineers]: 'E',
      [FactionType.Dwarves]: 'D',
    };
    return names[faction];
  };
  
  return (
    <div className="cult-tracks w-full">
      <canvas
        ref={canvasRef}
        width={500} // 250 * 2 for scale
        height={1000} // 500 * 2 for scale
        style={{ width: '250px', height: '500px', border: '1px solid #333', display: 'block' }}
      />
    </div>
  );
};
