import React from 'react';
import { Shovel, Sailboat } from 'lucide-react';

// Common Icon Styles
// These rely on the parent component or CSS to define sizing classes like 'icon-lg', 'icon-md', etc.

export const DwellingIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 30 30" className={className} style={style}>
        <path d="M 15 5 L 25 15 L 25 25 L 5 25 L 5 15 Z" fill={color || "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const TradingHouseIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="4 4 32 32" className={className} style={style}>
        <path d="M 10 10 L 20 20 L 20 27 L 30 27 L 30 40 L 0 40 L 0 30 L 0 20 Z" transform="translate(5, -5)" fill={color || "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const TempleIcon = ({ className, color }: { className?: string, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 30 30" className={className}>
        <circle cx="15" cy="15" r="12" fill={color || "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const StrongholdIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 30 30" className={className} style={style}>
        <path d="M 5 5 Q 10 15 5 25 Q 15 20 25 25 Q 20 15 25 5 Q 15 10 5 5 Z" fill={color || "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const SanctuaryIcon = ({ className, style, color }: { className?: string, style?: React.CSSProperties, color?: string }): React.ReactElement => (
    <svg viewBox="0 0 40 30" className={className} style={style}>
        <path d="M 13 27 A 12 12 0 0 1 13 3 L 27 3 A 12 12 0 0 1 27 27 Z" fill={color || "#D4B483"} stroke="#5C4033" strokeWidth="2" />
    </svg>
);

export const PriestIcon = ({ className, style, children }: { className?: string, style?: React.CSSProperties, children?: React.ReactNode }): React.ReactElement => (
    <div className={className} style={{ position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center', ...style }}>
        <svg viewBox="0 0 20 20" style={{ width: '100%', height: '100%', position: 'absolute', top: 0, left: 0 }}>
            <path d="M 10 2 L 14 6 L 14 18 L 6 18 L 6 6 Z" fill="#A0A0A0" stroke="#404040" strokeWidth="1.5" />
        </svg>
        <span style={{ position: 'relative', zIndex: 1, fontSize: '0.7em', fontWeight: 'bold', color: '#333' }}>{children}</span>
    </div>
);

export const BridgeIcon = ({ className }: { className?: string }): React.ReactElement => (
    <svg viewBox="0 0 40 20" className={className}>
        <rect x="5" y="5" width="30" height="10" fill="#8B4513" stroke="#5C4033" strokeWidth="2" />
        <line x1="5" y1="5" x2="35" y2="15" stroke="#5C4033" strokeWidth="1" />
        <line x1="5" y1="15" x2="35" y2="5" stroke="#5C4033" strokeWidth="1" />
    </svg>
);

export const ShippingIcon = ({ className, style }: { className?: string, style?: React.CSSProperties }): React.ReactElement => (
    <div className={className} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', ...style }}>
        <Sailboat color="#5C4033" size="100%" strokeWidth={2} />
    </div>
);

export const SpadeIcon = ({ className }: { className?: string }): React.ReactElement => (
    <Shovel className={className} color="#5C4033" size="100%" strokeWidth={2} />
);

export const WorkerIcon = ({ className, children, style }: { className?: string, children?: React.ReactNode, style?: React.CSSProperties }): React.ReactElement => (
    <div className={className} style={{ width: '1.5em', height: '1.5em', backgroundColor: 'white', border: '0.15em solid #5C4033', boxShadow: '0 0.05em 0.1em rgba(0,0,0,0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', fontSize: '1em', borderRadius: 0, flexShrink: 0, color: '#5C4033', ...style }}>
        {children}
    </div>
);

export const CoinIcon = ({ className, children, style }: { className?: string, children?: React.ReactNode, style?: React.CSSProperties }): React.ReactElement => (
    <div className={className} style={{ width: '1.5em', height: '1.5em', borderRadius: '50%', backgroundColor: '#FBBF24', border: '0.15em solid #D97706', boxShadow: '0 0.05em 0.1em rgba(0,0,0,0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', fontSize: '1em', flexShrink: 0, color: '#5C4033', ...style }}>
        {children}
    </div>
);

export const PowerIcon = ({ amount, className, style }: { amount: number, className?: string, style?: React.CSSProperties }): React.ReactElement => (
    <div className={className} style={{ width: '1.5em', height: '1.5em', display: 'flex', alignItems: 'center', justifyContent: 'center', backgroundColor: '#7C3AED', color: 'white', borderRadius: '50%', fontWeight: 'bold', fontSize: '1em', flexShrink: 0, border: '0.1em solid #5C4033', ...style }}>
        {amount}
    </div>
);

export const CultIcon = ({ className }: { className?: string }): React.ReactElement => (
    <div className={className} style={{ borderRadius: '50%', backgroundColor: '#E5E7EB', border: '0.15em solid #5C4033', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', fontSize: '0.75rem', aspectRatio: '1 / 1', color: '#5C4033' }}>
        C
    </div>
);

export const CultRhombusIcon = ({ className, showNumber = false }: { className?: string, showNumber?: boolean }): React.ReactElement => {
    // Rhombus where circles just touch at their borders (not intersecting)
    // Using em units for responsive scaling
    // Base: container 2.47em × 2em (matches ratio 37:30)

    const horizontalOffset = '0.866'; // em units: ~35% of width for rhombus geometry

    return (
        <div className={className} style={{ position: 'relative', width: '2.47em', height: '2em', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            {/* Top circle - white (Air) */}
            <div style={{ position: 'absolute', top: '0.5em', left: '50%', transform: 'translate(-50%, -50%)', width: '0.8em', height: '0.8em', borderRadius: '50%', backgroundColor: '#E5E7EB', border: '0.1em solid #333' }} />

            {/* Left circle - brown (Earth) */}
            <div style={{ position: 'absolute', top: '50%', left: `calc(50% - ${horizontalOffset}em)`, transform: 'translate(-50%, -50%)', width: '0.8em', height: '0.8em', borderRadius: '50%', backgroundColor: '#92400E', border: '0.1em solid #333' }} />

            {/* Right circle - blue (Water) */}
            <div style={{ position: 'absolute', top: '50%', left: `calc(50% + ${horizontalOffset}em)`, transform: 'translate(-50%, -50%)', width: '0.8em', height: '0.8em', borderRadius: '50%', backgroundColor: '#3B82F6', border: '0.1em solid #333' }} />

            {/* Bottom circle - red (Fire) */}
            <div style={{ position: 'absolute', top: '1.5em', left: '50%', transform: 'translate(-50%, -50%)', width: '0.8em', height: '0.8em', borderRadius: '50%', backgroundColor: '#EF4444', border: '0.1em solid #333' }} />

            {/* Number "2" in the center - only for 2VP town */}
            {showNumber && (
                <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', fontSize: '1em', fontWeight: 'bold', color: '#333', textShadow: '0 0 0.2em rgba(255, 255, 255, 0.8)', zIndex: 10 }}>
                    2
                </div>
            )}
        </div>
    );
};
export const VPIcon = ({ className, children }: { className?: string, children?: React.ReactNode }): React.ReactElement => (
    <div className={className} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', backgroundColor: '#D4B483', border: '0.15em solid #5C4033', borderRadius: '50%', width: '1.5em', height: '1.5em', fontWeight: 'bold', fontSize: '1em', color: '#5C4033' }}>
        {children ?? 'VP'}
    </div>
);

export const CultActionIcon = ({ className, style }: { className?: string, style?: React.CSSProperties }): React.ReactElement => (
    <svg viewBox="0 0 100 100" className={className} style={style}>
        {/* Octagon Background - Orange/Brownish for action */}
        <polygon points="30,0 70,0 100,30 100,70 70,100 30,100 0,70 0,30" fill="#D97706" stroke="#92400E" strokeWidth="2" />

        {/* Split Circle Group - Centered */}
        <g transform="translate(50, 50)">
            {/* Top Left - Red */}
            <path d="M 0 0 L 0 -35 A 35 35 0 0 0 -35 0 Z" fill="#EF4444" stroke="white" strokeWidth="1.5" />

            {/* Bottom Left - Blue */}
            <path d="M 0 0 L -35 0 A 35 35 0 0 0 0 35 Z" fill="#3B82F6" stroke="white" strokeWidth="1.5" />

            {/* Bottom Right - Orange */}
            <path d="M 0 0 L 0 35 A 35 35 0 0 0 35 0 Z" fill="#F59E0B" stroke="white" strokeWidth="1.5" />

            {/* Top Right - White/Gray */}
            <path d="M 0 0 L 35 0 A 35 35 0 0 0 0 -35 Z" fill="#E5E7EB" stroke="white" strokeWidth="1.5" />
        </g>
    </svg>
);

export const SpadeActionIcon = ({ className, style }: { className?: string, style?: React.CSSProperties }): React.ReactElement => (
    <div className={className} style={{ position: 'relative', ...style }}>
        <svg viewBox="0 0 100 100" style={{ width: '100%', height: '100%' }}>
            {/* Octagon Background - Orange/Brownish for action */}
            <polygon points="30,0 70,0 100,30 100,70 70,100 30,100 0,70 0,30" fill="#D97706" stroke="#92400E" strokeWidth="2" />
        </svg>
        <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: '50%', height: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Shovel color="#5C4033" size="100%" strokeWidth={2} />
        </div>
    </div>
);
