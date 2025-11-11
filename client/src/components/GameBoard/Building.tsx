// Building renderer - draws different building types based on terra-mystica/stc/game.js
import React from 'react';
import type { Building } from '../../types/game.types';
import { BuildingType } from '../../types/game.types';
import { FACTION_COLORS, getContrastColor } from '../../utils/colors';

interface BuildingComponentProps {
  building: Building;
  center: { x: number; y: number };
  hexSize: number;
}

export const BuildingComponent: React.FC<BuildingComponentProps> = ({
  building,
  center,
  hexSize,
}) => {
  const color = FACTION_COLORS[building.faction];
  const contrastColor = getContrastColor(color);
  
  switch (building.type) {
    case BuildingType.Dwelling:
      return <Dwelling center={center} color={color} contrastColor={contrastColor} />;
    case BuildingType.TradingHouse:
      return <TradingHouse center={center} color={color} contrastColor={contrastColor} />;
    case BuildingType.Temple:
      return <Temple center={center} color={color} contrastColor={contrastColor} />;
    case BuildingType.Stronghold:
      return <Stronghold center={center} color={color} contrastColor={contrastColor} />;
    case BuildingType.Sanctuary:
      return <Sanctuary center={center} color={color} contrastColor={contrastColor} />;
    default:
      return null;
  }
};

// Dwelling - simple house shape (pentagon)
// Based on terra-mystica/stc/game.js drawDwelling
const Dwelling: React.FC<{ center: { x: number; y: number }; color: string; contrastColor: string }> = ({ 
  center, 
  color, 
  contrastColor 
}) => {
  const { x, y } = center;
  const path = `
    M ${x} ${y - 10}
    L ${x + 10} ${y}
    L ${x + 10} ${y + 10}
    L ${x - 10} ${y + 10}
    L ${x - 10} ${y}
    Z
  `;
  
  return (
    <path
      d={path}
      fill={color}
      stroke={contrastColor}
      strokeWidth={2}
    />
  );
};

// Trading House - house with chimney
// Based on terra-mystica/stc/game.js drawTradingPost
const TradingHouse: React.FC<{ center: { x: number; y: number }; color: string; contrastColor: string }> = ({ 
  center, 
  color, 
  contrastColor 
}) => {
  const { x, y } = center;
  const path = `
    M ${x} ${y - 20}
    L ${x + 10} ${y - 10}
    L ${x + 10} ${y - 3}
    L ${x + 20} ${y - 3}
    L ${x + 20} ${y + 10}
    L ${x - 10} ${y + 10}
    L ${x - 10} ${y}
    L ${x - 10} ${y - 10}
    Z
  `;
  
  return (
    <path
      d={path}
      fill={color}
      stroke={contrastColor}
      strokeWidth={2}
    />
  );
};

// Temple - circle
// Based on terra-mystica/stc/game.js drawTemple
const Temple: React.FC<{ center: { x: number; y: number }; color: string; contrastColor: string }> = ({ 
  center, 
  color, 
  contrastColor 
}) => {
  return (
    <circle
      cx={center.x}
      cy={center.y - 5}
      r={14}
      fill={color}
      stroke={contrastColor}
      strokeWidth={2}
    />
  );
};

// Stronghold - rounded square shape
// Based on terra-mystica/stc/game.js drawStronghold
const Stronghold: React.FC<{ center: { x: number; y: number }; color: string; contrastColor: string }> = ({ 
  center, 
  color, 
  contrastColor 
}) => {
  const { x, y } = center;
  const size = 15;
  const yOffset = y - 5;
  
  return (
    <rect
      x={x - size}
      y={yOffset - size}
      width={size * 2}
      height={size * 2}
      rx={8}
      ry={8}
      fill={color}
      stroke={contrastColor}
      strokeWidth={2}
    />
  );
};

// Sanctuary - double circle (peanut shape)
// Based on terra-mystica/stc/game.js drawSanctuary
const Sanctuary: React.FC<{ center: { x: number; y: number }; color: string; contrastColor: string }> = ({ 
  center, 
  color, 
  contrastColor 
}) => {
  const { x, y } = center;
  const yOffset = y - 5;
  const circleSize = 12;
  const separation = 7;
  
  return (
    <g>
      <circle
        cx={x - separation}
        cy={yOffset}
        r={circleSize}
        fill={color}
        stroke={contrastColor}
        strokeWidth={2}
      />
      <circle
        cx={x + separation}
        cy={yOffset}
        r={circleSize}
        fill={color}
        stroke={contrastColor}
        strokeWidth={2}
      />
    </g>
  );
};
