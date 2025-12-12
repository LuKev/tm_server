import {
    ArrowRight
} from 'lucide-react';
import './ScoringTiles.css';
import {
    DwellingIcon,
    TradingHouseIcon,
    TempleIcon,
    StrongholdIcon,
    SanctuaryIcon,
    SpadeIcon,
    PriestIcon,
    WorkerIcon,
    CoinIcon,
    PowerIcon
} from '../shared/Icons';

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

const TownIcon = ({ className }: { className?: string }): React.ReactElement => (
    <div className={`icon-lg town-icon ${className || ''}`}>
        TOWN
    </div>
);

const ActionIcon = ({ type, className }: { type: ActionType, className?: string }): React.ReactElement | null => {
    switch (type) {
        case ActionType.Dwelling: return <DwellingIcon className={`icon-lg ${className || ''}`} />;
        case ActionType.TradingHouse: return <TradingHouseIcon className={`icon-lg ${className || ''}`} />;
        case ActionType.Temple: return <TempleIcon className={`icon-lg ${className || ''}`} />;
        case ActionType.Stronghold: return (
            <div className="flex items-center gap-1">
                <StrongholdIcon className={`icon-md ${className || ''}`} />
                <SanctuaryIcon className={`icon-md ${className || ''}`} />
            </div>
        );
        case ActionType.Spade: return <SpadeIcon className={`icon-lg ${className || ''}`} />;
        case ActionType.Town: return <TownIcon className={className} />;
        default: return null;
    }
};

const RewardIcon = ({ type, amount, className }: { type: RewardType, amount: number, className?: string }): React.ReactElement | null => {
    switch (type) {
        case RewardType.Priest: return <PriestIcon className={`icon-md ${className || ''}`} />;
        case RewardType.Power: return <PowerIcon amount={amount} className={`icon-xs reward-power ${className || ''}`} />;
        case RewardType.Spade: return <SpadeIcon className={`icon-md ${className || ''}`} />;
        case RewardType.Worker: return <WorkerIcon className={`icon-sm reward-worker ${className || ''}`} />;
        case RewardType.Coin: return <CoinIcon className={`icon-sm reward-coin ${className || ''}`} />;
        default: return null;
    }
};

const CultDots = ({ type, count }: { type: CultType, count: number }): React.ReactElement => {
    const getColorClass = (t: CultType): string => {
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
                <PriestIcon className="icon-sm" />
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

export const ScoringTiles: React.FC<ScoringTilesProps> = ({ tiles, currentRound }): React.ReactElement | null => {
    if (!tiles || tiles.length === 0) {
        return null;
    }

    // console.log('ScoringTiles rendering', { tiles, currentRound, timestamp: Date.now() });

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
                        key={`${String(tileId)}-${String(roundNum)}`}
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
