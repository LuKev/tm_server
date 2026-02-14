import React from 'react';
import { FactionType } from '../../types/game.types';
import { ShippingIcon, SpadeIcon } from './Icons';

export type ShippingDiggingDisplayProps = {
  factionType: FactionType;
  shipping: number;
  digging: number;
  hasTempShippingBonus?: boolean;
  compact?: boolean;
};

const canShowShippingForFaction = (factionType: FactionType): boolean => {
  // Fakirs and Dwarves have no shipping track.
  return factionType !== FactionType.Fakirs && factionType !== FactionType.Dwarves;
};

const canShowDiggingForFaction = (factionType: FactionType): boolean => {
  // Darklings have no digging upgrades (they pay priests instead).
  return factionType !== FactionType.Darklings;
};

export const ShippingDiggingDisplay: React.FC<ShippingDiggingDisplayProps> = ({
  factionType,
  shipping,
  digging,
  hasTempShippingBonus,
  compact,
}) => {
  const showShipping = canShowShippingForFaction(factionType);
  const showDigging = canShowDiggingForFaction(factionType);

  if (!showShipping && !showDigging) return null;

  const iconSize = compact ? '1em' : '1.15em';
  const textSize = compact ? '0.9em' : '1em';

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: compact ? '0.5em' : '0.75em' }}>
      {showShipping && (
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.25em', fontSize: textSize }}>
          <ShippingIcon style={{ width: iconSize, height: iconSize }} />
          <span>{shipping}</span>
          {hasTempShippingBonus && (
            <span style={{ opacity: 0.85, fontSize: '0.85em' }}>(+1)</span>
          )}
        </div>
      )}
      {showDigging && (
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.25em', fontSize: textSize }}>
          <span style={{ width: iconSize, height: iconSize, display: 'inline-flex' }}>
            <SpadeIcon />
          </span>
          <span>{digging}</span>
        </div>
      )}
    </div>
  );
};

