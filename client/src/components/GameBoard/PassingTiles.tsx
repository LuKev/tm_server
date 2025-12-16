import React from 'react';
import { BonusCardType } from '../../types/game.types';
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
import { useGameStore } from '../../stores/gameStore';

interface PassingTilesProps {
    availableCards?: number[]; // Array of BonusCardType values
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

const BonusCardContent: React.FC<{ type: BonusCardType }> = ({ type }) => {
    const split = isSplitCard(type);

    const renderContent = (): React.ReactNode => {
        switch (type) {
            case BonusCardType.Priest:
                return <PriestIcon className="flex-shrink-0" style={{ width: '60%' }} />; // Centered, larger
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
                        <div className="passing-tile-top">
                            <SpadeActionIcon className="flex-shrink-0" style={{ width: '75%' }} />
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
                        <div className="passing-tile-top">
                            <CultActionIcon className="flex-shrink-0" style={{ width: '75%' }} />
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
                return <div>?</div>;
        }
    };

    if (split) {
        return <>{renderContent()}</>;
    } else {
        return <div className="passing-tile-single">{renderContent()}</div>;
    }
};

export const PassingTiles: React.FC<PassingTilesProps> = () => {
    const gameState = useGameStore((state) => state.gameState);
    const availableCards = gameState?.bonusCards ?? [];

    const numCards = availableCards.length;

    return (
        <div className="passing-tiles-container" style={{ gridTemplateColumns: `repeat(${String(numCards)}, 1fr)`, aspectRatio: `${String(numCards)} / 4` }}>
            {availableCards.map((cardTypeVal, index) => {
                const cardType = cardTypeVal as BonusCardType;
                const showDivider = shouldShowDivider(cardType);

                return (
                    <div key={index} className="passing-tile" style={{ containerType: 'inline-size' }}>
                        {/* Background - Scroll texture simulation */}
                        <svg className="passing-tile-bg" viewBox="0 0 100 400" preserveAspectRatio="none">
                            <rect x="0" y="0" width="100" height="400" fill="white" stroke="black" strokeWidth="3" rx="5" />
                            {/* Divider line - only for split cards that should show it */}
                            {showDivider && (
                                <line x1="10" y1="200" x2="90" y2="200" stroke="black" strokeWidth="1" />
                            )}
                        </svg>

                        <div className="passing-tile-content">
                            <BonusCardContent type={cardType} />
                        </div>
                    </div>
                );
            })}
        </div>
    );
};
