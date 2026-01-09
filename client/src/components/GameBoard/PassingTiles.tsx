import React from 'react';
import { BonusCardType, SpecialActionType, type FactionType, type PlayerState } from '../../types/game.types';
import {
    CoinIcon,
    WorkerIcon,
    PowerIcon,
    PriestIcon,
    DwellingIcon,
    TradingHouseIcon,
    StrongholdIcon,
    SanctuaryIcon,
    CultActionIcon,
    SpadeActionIcon,
    ShippingIcon
} from '../shared/Icons';
import './PassingTiles.css';
import { FACTION_COLORS } from '../../utils/colors';

interface PassingTilesProps {
    availableCards?: number[]; // Array of BonusCardType values
    bonusCardCoins?: Record<string, number>; // Map of BonusCardType -> coins
    bonusCardOwners?: Record<string, string>; // Map of BonusCardType -> PlayerID
    players?: Record<string, PlayerState>; // Map of PlayerID -> Player
    passedPlayers?: Set<string>; // Set of PlayerIDs who have passed
}

const isSplitCard = (type: BonusCardType): boolean => {
    switch (type) {
        case BonusCardType.Priest:
        case BonusCardType.Coins6:
            return false;
        default:
            return true;
    }
};

const shouldShowDivider = (type: BonusCardType): boolean => {
    switch (type) {
        case BonusCardType.Spade:
        case BonusCardType.CultAdvance:
        case BonusCardType.Shipping:
            return false;
        default:
            return isSplitCard(type);
    }
};

export const BonusCardContent: React.FC<{
    type: BonusCardType;
    isUsed?: boolean;
    coins?: number;
    playerColor?: string;
    isPassed?: boolean;
}> = ({ type, isUsed, coins, playerColor, isPassed }) => {
    const split = isSplitCard(type);

    const renderContent = (): React.ReactNode => {
        switch (type) {
            case BonusCardType.Priest:
                return <PriestIcon className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto' }} />; // Centered, larger
            case BonusCardType.Coins6:
                return <div className="flex-shrink-0" style={{ width: '100%', display: 'flex', justifyContent: 'center' }}><CoinIcon className="" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }}>6</CoinIcon></div>; // Centered, smaller

            case BonusCardType.Shipping:
                return (
                    <>
                        <div className="passing-tile-top">
                            <div className="flex flex-col items-center gap-0" style={{ width: '100%', justifyContent: 'center', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
                                <ShippingIcon className="flex-shrink-0" style={{ width: '60%' }} />
                                <span className="font-bold text-[#5C4033]" style={{ fontSize: '25cqw' }}>+1</span>
                            </div>
                        </div>
                        <div className="passing-tile-bottom">
                            <PowerIcon amount={3} className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }} />
                        </div>
                    </>
                );
            case BonusCardType.DwellingVP:
                return (
                    <>
                        <div className="passing-tile-top">
                            <div className="flex flex-col items-center gap-0" style={{ width: '100%', justifyContent: 'center', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
                                <DwellingIcon className="flex-shrink-0" style={{ width: '60%' }} />
                                <span className="passing-tile-arrow" style={{ fontSize: '25cqw', lineHeight: '1' }}>↓</span>
                                <span className="vp-number" style={{ fontSize: '25cqw', lineHeight: '1' }}>1</span>
                            </div>
                        </div>
                        <div className="passing-tile-bottom">
                            <CoinIcon className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }}>2</CoinIcon>
                        </div>
                    </>
                );
            case BonusCardType.WorkerPower:
                return (
                    <>
                        <div className="passing-tile-top">
                            <WorkerIcon className="flex-shrink-0" style={{ width: '35%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }}>1</WorkerIcon>
                        </div>
                        <div className="passing-tile-bottom">
                            <PowerIcon amount={3} className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }} />
                        </div>
                    </>
                );
            case BonusCardType.Spade:
                return (
                    <>
                        <div className="passing-tile-top relative flex items-center justify-center">
                            <div className="relative w-[60%]">
                                <SpadeActionIcon className="w-full">
                                    {isUsed && (
                                        <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 10, pointerEvents: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                            <svg viewBox="-2 -2 44 44" style={{ width: '100%', height: '100%', display: 'block' }}>
                                                <path d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z" fill="#d6d3d1" stroke="#78716c" strokeWidth="2" fillOpacity="0.9" />
                                                <path d="M 10 10 L 30 30 M 30 10 L 10 30" stroke="#78716c" strokeWidth="3" strokeLinecap="round" />
                                            </svg>
                                        </div>
                                    )}
                                </SpadeActionIcon>
                            </div>
                        </div>
                        <div className="passing-tile-bottom">
                            <CoinIcon className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }}>2</CoinIcon>
                        </div>
                    </>
                );
            case BonusCardType.TradingHouseVP:
                return (
                    <>
                        <div className="passing-tile-top">
                            <div className="flex flex-col items-center gap-0" style={{ width: '100%', justifyContent: 'center', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
                                <TradingHouseIcon className="flex-shrink-0" style={{ width: '60%' }} />
                                <span className="passing-tile-arrow" style={{ fontSize: '25cqw' }}>↓</span>
                                <span className="vp-number" style={{ fontSize: '25cqw' }}>2</span>
                            </div>
                        </div>
                        <div className="passing-tile-bottom">
                            <WorkerIcon className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }}>1</WorkerIcon>
                        </div>
                    </>
                );
            case BonusCardType.CultAdvance:
                return (
                    <>
                        <div className="passing-tile-top relative flex items-center justify-center">
                            <div className="relative w-[60%]">
                                <CultActionIcon className="w-full">
                                    {isUsed && (
                                        <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 10, pointerEvents: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                            <svg viewBox="-2 -2 44 44" style={{ width: '100%', height: '100%', display: 'block' }}>
                                                <path d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z" fill="#d6d3d1" stroke="#78716c" strokeWidth="2" fillOpacity="0.9" />
                                                <path d="M 10 10 L 30 30 M 30 10 L 10 30" stroke="#78716c" strokeWidth="3" strokeLinecap="round" />
                                            </svg>
                                        </div>
                                    )}
                                </CultActionIcon>
                            </div>
                        </div>
                        <div className="passing-tile-bottom">
                            <CoinIcon className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }}>4</CoinIcon>
                        </div>
                    </>
                );
            case BonusCardType.StrongholdSanctuaryVP:
                return (
                    <>
                        <div className="passing-tile-top">
                            <div className="flex flex-col items-center gap-0" style={{ width: '100%', justifyContent: 'center', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
                                <StrongholdIcon className="flex-shrink-0" style={{ width: '45%' }} />
                                <SanctuaryIcon className="flex-shrink-0" style={{ width: '45%' }} />
                                <span className="passing-tile-arrow" style={{ fontSize: '25cqw' }}>↓</span>
                                <span className="vp-number" style={{ fontSize: '25cqw' }}>4</span>
                            </div>
                        </div>
                        <div className="passing-tile-bottom">
                            <WorkerIcon className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }}>2</WorkerIcon>
                        </div>
                    </>
                );
            case BonusCardType.ShippingVP:
                return (
                    <>
                        <div className="passing-tile-top">
                            <div className="flex flex-col items-center gap-0" style={{ width: '100%', justifyContent: 'center', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
                                <ShippingIcon className="flex-shrink-0" style={{ width: '50%' }} />
                                <span className="passing-tile-arrow" style={{ fontSize: '25cqw' }}>↓</span>
                                <span className="vp-number" style={{ fontSize: '25cqw' }}>3</span>
                            </div>
                        </div>
                        <div className="passing-tile-bottom">
                            <PowerIcon amount={3} className="flex-shrink-0" style={{ width: '60%', aspectRatio: '1/1', height: 'auto', fontSize: '25cqw' }} />
                        </div>
                    </>
                );
            default:
                return null;
        }
    };

    return (
        <div className={`passing-tile ${isPassed ? 'brightness-75' : ''}`} style={{ containerType: 'inline-size' }}>
            {/* Player Indicator */}
            {playerColor && (
                <div style={{
                    position: 'absolute',
                    top: '5%',
                    left: '50%',
                    transform: 'translateX(-50%)',
                    width: '15%',
                    aspectRatio: '1/1',
                    borderRadius: '50%',
                    backgroundColor: playerColor,
                    border: '1px solid white',
                    boxShadow: '0 1px 2px rgba(0,0,0,0.3)',
                    zIndex: 10
                }} />
            )}

            {/* Coin Indicator */}
            {coins !== undefined && coins > 0 && (
                <div style={{
                    position: 'absolute',
                    top: '5%',
                    left: '50%',
                    transform: 'translateX(-50%)',
                    zIndex: 10,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                }}>
                    <CoinIcon style={{ width: '100%', height: '100%', aspectRatio: '1/1', fontSize: '20cqw' }}>{coins}</CoinIcon>
                </div>
            )}

            {split ? renderContent() : <div className="passing-tile-single">{renderContent()}</div>}
            {shouldShowDivider(type) && <div className="passing-tile-divider" />}
        </div>
    );
};

export const PassingTiles: React.FC<PassingTilesProps> = ({
    availableCards,
    bonusCardCoins,
    bonusCardOwners,
    players,
    passedPlayers
}) => {
    if (!availableCards || availableCards.length === 0) return null;

    const numCards = availableCards.length;

    return (
        <div className="passing-tiles-container" style={{ gridTemplateColumns: `repeat(${String(numCards)}, 1fr)`, aspectRatio: `${String(numCards)} / 4` }}>
            {availableCards.map((cardTypeVal) => {
                const cardType = cardTypeVal as BonusCardType;

                const ownerId = bonusCardOwners?.[String(cardType)];
                const player = ownerId && players ? players[ownerId] : null;

                // Resolve faction color
                let playerColor = undefined;
                if (player) {
                    // Logic from PlayerBoards.tsx to resolve faction type
                    let factionType = 1; // Default Nomads
                    if (typeof player.faction === 'number') {
                        factionType = player.faction;
                    } else if (player.Faction && typeof player.Faction === 'object' && 'Type' in player.Faction) {
                        factionType = player.Faction.Type;
                    }
                    playerColor = FACTION_COLORS[factionType as FactionType];
                }

                const isPassed = ownerId ? passedPlayers?.has(ownerId) : false;
                const coins = bonusCardCoins?.[String(cardType)] ?? 0;

                // Determine if this card is used (for special action cards)
                let isUsed = false;
                if (player?.specialActionsUsed) {
                    if (cardType === BonusCardType.Spade) {
                        isUsed = player.specialActionsUsed[SpecialActionType.BonusCardSpade];
                    } else if (cardType === BonusCardType.CultAdvance) {
                        isUsed = player.specialActionsUsed[SpecialActionType.BonusCardCultAdvance];
                    }
                }

                return (
                    <BonusCardContent
                        key={cardType}
                        type={cardType}
                        isUsed={isUsed}
                        playerColor={playerColor}
                        coins={coins}
                        isPassed={isPassed}
                    />
                );
            })}
        </div>
    );
};

export default PassingTiles;
