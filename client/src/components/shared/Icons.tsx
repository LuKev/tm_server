import React from 'react';
import { Shovel, Sailboat } from 'lucide-react';
import { PRIEST_PAWN_PATH } from './iconGeometry';

// Common Icon Styles
// These rely on the parent component or CSS to define sizing classes like 'icon-lg', 'icon-md', etc.

export const KeyIcon = ({ className }: { className?: string }): React.ReactElement => (
    <svg viewBox="0 0 30 30" className={className} aria-hidden="true">
        <path d="M 12 15 L 25 15 L 25 21 L 21 21 L 21 18 L 18 18 L 18 15" fill="none" stroke="#8c682c" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
        <circle cx="8" cy="15" r="5" fill="#f4d58a" stroke="#8c682c" strokeWidth="2.5" />
        <circle cx="8" cy="15" r="1.5" fill="#fffaf0" />
    </svg>
);

export const DwellingIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 30 30" className={className} style={style}>
        <path d="M 15 5 L 25 15 L 25 25 L 5 25 L 5 15 Z" fill={color ?? "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const TradingHouseIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="4 4 32 32" className={className} style={style}>
        <path d="M 10 10 L 20 20 L 20 27 L 30 27 L 30 40 L 0 40 L 0 30 L 0 20 Z" transform="translate(5, -5)" fill={color ?? "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const TempleIcon = ({ className, color }: { className?: string, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 30 30" className={className}>
        <circle cx="15" cy="15" r="12" fill={color ?? "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const StrongholdIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 30 30" className={className} style={style}>
        <path d="M 5 5 Q 10 15 5 25 Q 15 20 25 25 Q 20 15 25 5 Q 15 10 5 5 Z" fill={color ?? "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const SanctuaryIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 40 30" className={className} style={style}>
        <path d="M 13 27 A 12 12 0 0 1 13 3 L 27 3 A 12 12 0 0 1 27 27 Z" fill={color ?? "#D4B483"} stroke="#5C4033" strokeWidth="2.5" />
    </svg>
);

export const PriestIcon = ({ className, style, children, color = '#A0A0A0' }: { className?: string, style?: React.CSSProperties, children?: number | string, color?: string }): React.ReactElement => (
    <svg data-icon="priest" viewBox="0 0 20 20" className={className} style={{ width: '1.5em', height: '1.5em', flexShrink: 0, display: 'block', ...style }}>
        <path d={PRIEST_PAWN_PATH} fill={color} stroke="#404040" strokeWidth="1.2" />
        <text x="10" y="14.5" textAnchor="middle" dominantBaseline="central" fontSize="7" fontWeight="700" fill="#252b32">{children}</text>
    </svg>
);

export const BridgeIcon = ({ className }: { className?: string }): React.ReactElement => (
    <svg viewBox="0 0 40 20" className={className}>
        <rect x="5" y="5" width="30" height="10" fill="#8B4513" stroke="#5C4033" strokeWidth="2" />
        <line x1="5" y1="5" x2="35" y2="15" stroke="#5C4033" strokeWidth="1" />
        <line x1="5" y1="15" x2="35" y2="5" stroke="#5C4033" strokeWidth="1" />
    </svg>
);

export const ShippingIcon = ({ className, style }: { className?: string, style?: React.CSSProperties }): React.ReactElement => (
    <Sailboat className={className} color="#5C4033" strokeWidth={2} style={{ width: '1.5em', height: '1.5em', display: 'block', flexShrink: 0, ...style }} />
);

export const SpadeIcon = ({ className }: { className?: string }): React.ReactElement => (
    <Shovel className={className} color="#5C4033" size="100%" strokeWidth={2} />
);

export const WorkerIcon = ({ className, children, style }: { className?: string, children?: number | string, style?: React.CSSProperties }): React.ReactElement => (
    <svg data-icon="worker" viewBox="0 0 30 30" className={className} style={{ width: '1.5em', height: '1.5em', flexShrink: 0, display: 'block', ...style }}>
        <rect x="3" y="3" width="24" height="24" fill="white" stroke="#5C4033" strokeWidth="2.5" />
        <text x="15" y="15" textAnchor="middle" dominantBaseline="central" fontSize="16" fontWeight="700" fill="#5C4033">{children}</text>
    </svg>
);

export const CoinIcon = ({ className, children, style }: { className?: string, children?: number | string, style?: React.CSSProperties }): React.ReactElement => (
    <svg data-icon="coin" viewBox="0 0 30 30" className={className} style={{ width: '1.5em', height: '1.5em', flexShrink: 0, display: 'block', ...style }}>
        <circle cx="15" cy="15" r="12" fill="#FBBF24" stroke="#D97706" strokeWidth="2.5" />
        <text x="15" y="15" textAnchor="middle" dominantBaseline="central" fontSize={String(children).length > 1 ? 13 : 16} fontWeight="700" fill="#5C4033">{children}</text>
    </svg>
);

export const PowerIcon = ({ amount, className, style }: { amount: number, className?: string, style?: React.CSSProperties }): React.ReactElement => (
    <svg data-icon="power" viewBox="0 0 30 30" className={className} style={{ width: '1.5em', height: '1.5em', flexShrink: 0, display: 'block', ...style }}>
        <circle cx="15" cy="15" r="12" fill="#7C3AED" stroke="#5C4033" strokeWidth="2" />
        <text x="15" y="15" textAnchor="middle" dominantBaseline="central" fontSize={amount > 9 ? 13 : 16} fontWeight="700" fill="white">{amount}</text>
    </svg>
);

// Purple power circle (used as a label/icon without a number).
export const PowerCircleIcon = ({ className, style }: { className?: string, style?: React.CSSProperties }): React.ReactElement => (
    <div
        className={className}
        style={{
            width: '1.15em',
            height: '1.15em',
            borderRadius: '50%',
            backgroundColor: '#7C3AED',
            border: '0.1em solid #5C4033',
            boxShadow: '0 0.05em 0.1em rgba(0,0,0,0.1)',
            flexShrink: 0,
            boxSizing: 'border-box',
            ...style
        }}
    />
);

export const CultRhombusIcon = ({ className, showNumber = false, style }: { className?: string, showNumber?: boolean, style?: React.CSSProperties }): React.ReactElement => (
    <svg viewBox="0 0 40 34" className={className} style={{ width: '1.8em', height: '1.5em', flexShrink: 0, display: 'block', ...style }}>
        <circle cx="20" cy="7" r="6" fill="#E5E7EB" stroke="#333" />
        <circle cx="8" cy="17" r="6" fill="#92400E" stroke="#333" />
        <circle cx="32" cy="17" r="6" fill="#3B82F6" stroke="#333" />
        <circle cx="20" cy="27" r="6" fill="#EF4444" stroke="#333" />
        {showNumber && <text x="20" y="17" textAnchor="middle" dominantBaseline="central" fontSize="12" fontWeight="700" fill="#333">2</text>}
    </svg>
);

export const CultActionIcon = ({ className, style, children }: { className?: string, style?: React.CSSProperties, children?: React.ReactNode }): React.ReactElement => (
    <div className={className} style={{ position: 'relative', aspectRatio: '1/1', ...style }}>
        <svg viewBox="-2 -2 44 44" style={{ width: '100%', height: '100%', display: 'block' }}>
            {/* Octagon Background - Orange/Brownish for action - exact same as grey overlay */}
            <path d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z" fill="#D97706" stroke="#92400E" strokeWidth="2" />

            {/* Split Circle Group - Centered */}
            <g transform="translate(20, 20)">
                {/* Top Left - Red */}
                <path d="M 0 0 L 0 -11 A 11 11 0 0 0 -11 0 Z" fill="#EF4444" stroke="white" strokeWidth="1" />

                {/* Bottom Left - Blue */}
                <path d="M 0 0 L -11 0 A 11 11 0 0 0 0 11 Z" fill="#3B82F6" stroke="white" strokeWidth="1" />

                {/* Bottom Right - Orange */}
                <path d="M 0 0 L 0 11 A 11 11 0 0 0 11 0 Z" fill="#F59E0B" stroke="white" strokeWidth="1" />

                {/* Top Right - White/Gray */}
                <path d="M 0 0 L 11 0 A 11 11 0 0 0 0 -11 Z" fill="#E5E7EB" stroke="white" strokeWidth="1" />
            </g>
        </svg>
        {children}
    </div>
);

export const SpadeActionIcon = ({ className, style, children }: { className?: string, style?: React.CSSProperties, children?: React.ReactNode }): React.ReactElement => (
    <div className={className} style={{ position: 'relative', aspectRatio: '1/1', ...style }}>
        <svg viewBox="-2 -2 44 44" style={{ width: '100%', height: '100%', display: 'block' }}>
            {/* Octagon Background - Orange/Brownish for action - exact same as grey overlay */}
            <path d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z" fill="#D97706" stroke="#92400E" strokeWidth="2" />
        </svg>
        <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: '40%', height: '40%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Shovel color="#5C4033" size="100%" strokeWidth={2} />
        </div>
        {children}
    </div>
);
