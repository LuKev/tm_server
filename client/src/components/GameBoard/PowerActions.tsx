import React from 'react';
import { PowerActionType } from '../../types/game.types';
import {
    BridgeIcon,
    PriestIcon,
    WorkerIcon,
    CoinIcon,
    SpadeIcon
} from '../shared/Icons';
import './PowerActions.css';

interface PowerActionConfig {
    type: PowerActionType;
    cost: number;
    label: string;
    icon: React.ReactNode;
}

// Octagon SVG background wrapper
const OctagonWrapper = ({ children }: { children: React.ReactNode }): React.ReactElement => (
    <div className="octagon-wrapper">
        <svg viewBox="-2 -2 44 44" className="octagon-svg">
            <path
                d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z"
                fill="#f97316" // orange-500
                stroke="#c2410c" // orange-700
                strokeWidth="2"
            />
        </svg>
        <div className="octagon-content">
            {children}
        </div>
    </div>
);

const ACTIONS: PowerActionConfig[] = [
    {
        type: PowerActionType.Bridge,
        cost: 3,
        label: "Bridge",
        icon: <BridgeIcon className="icon-bridge" />
    },
    {
        type: PowerActionType.Priest,
        cost: 3,
        label: "Priest",
        icon: <PriestIcon className="icon-priest" />
    },
    {
        type: PowerActionType.Workers,
        cost: 4,
        label: "2 Workers",
        icon: (
            <div className="workers-container">
                <WorkerIcon className="worker-icon-styled" style={{ width: '12px', height: '12px' }} />
                <WorkerIcon className="worker-icon-styled" style={{ width: '12px', height: '12px' }} />
            </div>
        )
    },
    {
        type: PowerActionType.Coins,
        cost: 4,
        label: "7 Coins",
        icon: (
            <CoinIcon className="coin-icon-large" style={{ width: '24px', height: '24px' }}>
                <span className="coins-text">7</span>
            </CoinIcon>
        )
    },
    {
        type: PowerActionType.Spade,
        cost: 4,
        label: "Spade",
        icon: <SpadeIcon className="icon-spade" />
    },
    {
        type: PowerActionType.DoubleSpade,
        cost: 6,
        label: "2 Spades",
        icon: (
            <div className="double-spade-container">
                <SpadeIcon className="icon-spade-small" />
                <SpadeIcon className="icon-spade-small" />
            </div>
        )
    }
];

import { useGameStore } from '../../stores/gameStore';

// ...

interface PowerActionsProps {
    onActionClick?: (action: PowerActionType) => void;
}

export const PowerActions: React.FC<PowerActionsProps> = ({ onActionClick }): React.ReactElement => {
    const gameState = useGameStore(state => state.gameState);
    const usedActions = gameState?.powerActions?.UsedActions ?? {};

    return (
        <div className="power-actions-container">
            {ACTIONS.map((action) => {
                const isUsed = usedActions[action.type];

                return (
                    <div
                        key={action.type}
                        className={`power-action-tile ${isUsed ? 'used' : ''}`}
                        onClick={() => !isUsed && onActionClick?.(action.type)}
                        title={action.label}
                        style={{ cursor: isUsed ? 'not-allowed' : 'pointer', opacity: isUsed ? 0.7 : 1 }}
                    >
                        {/* Power Cost */}
                        <div className="power-cost">
                            <div className="power-cost-circle">
                                {action.cost}
                            </div>
                        </div>

                        {/* Action Result (Octagon with Icon) */}
                        <div className="action-result" style={{ position: 'relative' }}>
                            <OctagonWrapper>
                                {action.icon}
                            </OctagonWrapper>

                            {isUsed && (
                                <div style={{
                                    position: 'absolute',
                                    top: 0,
                                    left: 0,
                                    width: '100%',
                                    height: '100%',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    zIndex: 10
                                }}>
                                    <svg viewBox="-2 -2 44 44" className="octagon-svg-overlay" style={{ width: '100%', height: '100%' }}>
                                        <path
                                            d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z"
                                            fill="#d6d3d1" // stone-300 (tan-ish)
                                            stroke="#78716c" // stone-500
                                            strokeWidth="2"
                                            fillOpacity="0.9"
                                        />
                                        <line x1="10" y1="10" x2="30" y2="30" stroke="#78716c" strokeWidth="4" strokeLinecap="round" />
                                        <line x1="30" y1="10" x2="10" y2="30" stroke="#78716c" strokeWidth="4" strokeLinecap="round" />
                                    </svg>
                                </div>
                            )}
                        </div>
                    </div>
                );
            })}
        </div>
    );
};
