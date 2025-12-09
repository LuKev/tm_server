import React from 'react';
import { Shovel } from 'lucide-react';

// Common Icon Styles
// These rely on the parent component or CSS to define sizing classes like 'icon-lg', 'icon-md', etc.

export const DwellingIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 30 30" className={className}>
        <path d="M 15 5 L 25 15 L 25 25 L 5 25 L 5 15 Z" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const TradingHouseIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 40 40" className={className}>
        <path d="M 10 10 L 20 20 L 20 27 L 30 27 L 30 40 L 0 40 L 0 30 L 0 20 Z" transform="translate(5, -5)" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const TempleIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 30 30" className={className}>
        <circle cx="15" cy="15" r="12" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const StrongholdIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 30 30" className={className}>
        <path d="M 5 5 Q 10 15 5 25 Q 15 20 25 25 Q 20 15 25 5 Q 15 10 5 5 Z" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const SanctuaryIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 40 30" className={className}>
        <path d="M 13 27 A 12 12 0 0 1 13 3 L 27 3 A 12 12 0 0 1 27 27 Z" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const PriestIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 20 20" className={className}>
        <path d="M 10 2 L 14 6 L 14 18 L 6 18 L 6 6 Z" fill="#A0A0A0" stroke="#404040" strokeWidth="1" />
    </svg>
);

export const BridgeIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 40 20" className={className}>
        <rect x="5" y="5" width="30" height="10" fill="#8B4513" stroke="#5C4033" strokeWidth="2" />
        <line x1="5" y1="5" x2="35" y2="15" stroke="#5C4033" strokeWidth="1" />
        <line x1="5" y1="15" x2="35" y2="5" stroke="#5C4033" strokeWidth="1" />
    </svg>
);

export const SpadeIcon = ({ className }: { className?: string }) => (
    <Shovel className={className} color="#5C4033" />
);

export const WorkerIcon = ({ className }: { className?: string }) => (
    <div className={`bg-white border-2 border-gray-400 shadow-sm ${className}`} />
);

export const CoinIcon = ({ className, children }: { className?: string, children?: React.ReactNode }) => (
    <div className={`rounded-full bg-yellow-400 border border-yellow-600 shadow-sm flex items-center justify-center ${className}`}>
        {children}
    </div>
);

export const PowerIcon = ({ amount, className }: { amount: number, className?: string }) => (
    <div className={`flex items-center justify-center bg-purple-700 text-white rounded-full font-bold ${className}`}>
        {amount}
    </div>
);
