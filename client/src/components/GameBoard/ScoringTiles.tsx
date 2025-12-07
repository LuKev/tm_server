import React from 'react';
import {
    Shovel,
    ArrowRight
} from 'lucide-react';
import './ScoringTiles.css';

// Types for tile configuration
enum ActionType {
    Dwelling,
    TradingHouse,
    Temple,
    Stronghold,
    Spade,
    Town
}

enum CultType {
    Fire,
    Water,
    Earth,
    Air
}

enum RewardType {
    Priest,
    Power,
    Spade,
    Worker,
    Coin
}

interface TileConfig {
    id: number;
    action: ActionType;
    vp: number;
    cult: CultType;
    cultSteps: number;
    reward: RewardType;
    rewardAmount: number;
}

// Configuration for all 9 tiles
const TILE_CONFIGS: Record<number, TileConfig> = {
    0: { id: 0, action: ActionType.Dwelling, vp: 2, cult: CultType.Water, cultSteps: 4, reward: RewardType.Priest, rewardAmount: 1 },
    1: { id: 1, action: ActionType.Dwelling, vp: 2, cult: CultType.Fire, cultSteps: 4, reward: RewardType.Power, rewardAmount: 4 },
    2: { id: 2, action: ActionType.TradingHouse, vp: 3, cult: CultType.Water, cultSteps: 4, reward: RewardType.Spade, rewardAmount: 1 },
    3: { id: 3, action: ActionType.TradingHouse, vp: 3, cult: CultType.Air, cultSteps: 4, reward: RewardType.Spade, rewardAmount: 1 },
    4: { id: 4, action: ActionType.Temple, vp: 4, cult: CultType.Fire, cultSteps: 0, reward: RewardType.Coin, rewardAmount: 2 },
    5: { id: 5, action: ActionType.Stronghold, vp: 5, cult: CultType.Fire, cultSteps: 2, reward: RewardType.Worker, rewardAmount: 1 },
    6: { id: 6, action: ActionType.Stronghold, vp: 5, cult: CultType.Air, cultSteps: 2, reward: RewardType.Worker, rewardAmount: 1 },
    7: { id: 7, action: ActionType.Spade, vp: 2, cult: CultType.Earth, cultSteps: 1, reward: RewardType.Coin, rewardAmount: 1 },
    8: { id: 8, action: ActionType.Town, vp: 5, cult: CultType.Earth, cultSteps: 4, reward: RewardType.Spade, rewardAmount: 1 },
};

// SVG Components from Building.tsx
// SVG Components from Building.tsx
const DwellingIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 30 30" className={`icon-lg ${className || ''}`}>
        <path d="M 15 5 L 25 15 L 25 25 L 5 25 L 5 15 Z" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

const TradingHouseIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 40 40" className={`icon-lg ${className || ''}`}>
        <path d="M 10 10 L 20 20 L 20 27 L 30 27 L 30 40 L 0 40 L 0 30 L 0 20 Z" transform="translate(5, -5)" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

const TempleIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 30 30" className={`icon-lg ${className || ''}`}>
        <circle cx="15" cy="15" r="12" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

const SanctuaryIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 40 30" className={`icon-lg ${className || ''}`}>
        {/* Pill shape: Two arcs connected by lines */}
        {/* Left arc center roughly (13, 15), Right arc center roughly (27, 15) for 40 width */}
        {/* Let's try to match the canvas logic: center +/- 7. Radius 12. */}
        {/* Canvas coords were relative to hex center. Here we fit in 40x30 box. Center (20, 15). */}
        {/* Left center: (13, 15). Right center: (27, 15). Radius 12. */}
        <path d="M 13 27 A 12 12 0 0 1 13 3 L 27 3 A 12 12 0 0 1 27 27 Z" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

const StrongholdIcon = ({ className }: { className?: string }) => (
    <svg viewBox="0 0 30 30" className={`icon-lg ${className || ''}`}>
        <path d="M 5 5 Q 10 15 5 25 Q 15 20 25 25 Q 20 15 25 5 Q 15 10 5 5 Z" fill="#D4B483" stroke="#5C4033" strokeWidth="2" />
    </svg>
);

const TownIcon = ({ className }: { className?: string }) => (
    <div className={`icon-lg town-icon ${className || ''}`}>
        TOWN
    </div>
);

const ActionIcon = ({ type, className }: { type: ActionType, className?: string }) => {
    switch (type) {
        case ActionType.Dwelling: return <DwellingIcon className={className} />;
        case ActionType.TradingHouse: return <TradingHouseIcon className={className} />;
        case ActionType.Temple: return <TempleIcon className={className} />;
        case ActionType.Stronghold: return (
            <div className="flex items-center gap-1">
                <StrongholdIcon className={className} />
                <SanctuaryIcon className={className} />
            </div>
        );
        case ActionType.Spade: return <Shovel className={`icon-lg ${className || ''}`} color="#5C4033" />;
        case ActionType.Town: return <TownIcon className={className} />;
        default: return null;
    }
};

const RewardIcon = ({ type, amount, className }: { type: RewardType, amount: number, className?: string }) => {
    switch (type) {
        case RewardType.Priest: return (
            <svg viewBox="0 0 20 20" className={`icon-md ${className || ''}`}>
                <path d="M 10 2 L 14 6 L 14 18 L 6 18 L 6 6 Z" fill="#A0A0A0" stroke="#404040" strokeWidth="1" />
            </svg>
        );
        case RewardType.Power: return (
            <div className={`icon-xs reward-power ${className || ''}`}>
                {amount}
            </div>
        );
        case RewardType.Spade: return <Shovel className={`icon-md ${className || ''}`} color="#5C4033" />;
        case RewardType.Worker: return <div className={`icon-sm reward-worker ${className || ''}`} />;
        case RewardType.Coin: return (
            <div className={`icon-sm reward-coin ${className || ''}`} />
        );
        default: return null;
    }
};

const CultDots = ({ type, count }: { type: CultType, count: number }) => {
    const getColorClass = (t: CultType) => {
        switch (t) {
            case CultType.Fire: return 'bg-red-500';
            case CultType.Water: return 'bg-blue-500';
            case CultType.Earth: return 'bg-amber-700';
            case CultType.Air: return 'bg-gray-400';
        }
    };

    // Special case for 0 steps (Temple/Priest tile)
    if (count === 0) {
        return (
            <div className="priest-icon-container">
                {/* Priest Icon */}
                <svg viewBox="0 0 20 20" className="icon-sm">
                    <path d="M 10 2 L 14 6 L 14 18 L 6 18 L 6 6 Z" fill="#A0A0A0" stroke="#404040" strokeWidth="1" />
                </svg>
                <ArrowRight className="icon-xs arrow-rotated" />
                <div className={`cult-dot ${getColorClass(type)}`} />
            </div>
        );
    }

    return (
        <div className="cult-dots-container">
            {Array.from({ length: count }).map((_, i) => (
                <div key={i} className={`cult-dot ${getColorClass(type)}`} />
            ))}
        </div>
    );
};

interface ScoringTilesProps {
    tiles: number[];
    currentRound: number;
}

export const ScoringTiles: React.FC<ScoringTilesProps> = ({ tiles, currentRound }) => {
    if (!tiles || tiles.length === 0) {
        return null;
    }

    console.log('ScoringTiles rendering', { tiles, currentRound, timestamp: Date.now() });

    // Reverse tiles to show R6 at top, R1 at bottom
    const reversedTiles = [...tiles].reverse();

    return (
        <div className="scoring-tiles-container">
            {reversedTiles.map((tileId, reverseIndex) => {
                const config = TILE_CONFIGS[tileId];
                if (!config) return null;

                // Calculate actual round number (1-6)
                const roundNum = 6 - reverseIndex;
                const isCurrentRound = roundNum === currentRound;
                const isPastRound = roundNum < currentRound;

                return (
                    <div
                        key={`${tileId}-${roundNum}`}
                        className={`scoring-tile ${isCurrentRound ? 'current-round' : ''} ${isPastRound ? 'past-round' : ''}`}
                    >
                        {/* Left Side: Scoring Action */}
                        <div className="tile-section left">
                            <div className="content-row">
                                <ActionIcon type={config.action} />
                                <ArrowRight className="icon-sm" style={{ color: 'black' }} />
                                <div className="vp-text">{config.vp}</div>
                            </div>
                        </div>

                        {/* Right Side: Cult Reward */}
                        <div className="tile-section">
                            {/* Round 6 has no cult reward */}
                            {roundNum !== 6 ? (
                                <div className="content-row">
                                    <CultDots type={config.cult} count={config.cultSteps} />
                                    <ArrowRight className="icon-sm" style={{ color: 'black' }} />
                                    <RewardIcon type={config.reward} amount={config.rewardAmount} />
                                </div>
                            ) : (
                                <div className="final-round-text">Final Round</div>
                            )}
                        </div>
                    </div>
                );
            })}
        </div>
    );
};
