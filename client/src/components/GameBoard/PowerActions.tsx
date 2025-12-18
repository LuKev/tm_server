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

interface PowerActionsProps {
    onActionClick?: (action: PowerActionType) => void;
}

export const PowerActions: React.FC<PowerActionsProps> = ({ onActionClick }): React.ReactElement => {
    return (
        <div className="power-actions-container">
            {ACTIONS.map((action) => (
                <div
                    key={action.type}
                    className="power-action-tile"
                    onClick={() => onActionClick?.(action.type)}
                    title={action.label}
                >
                    {/* Power Cost */}
                    <div className="power-cost">
                        <div className="power-cost-circle">
                            {action.cost}
                        </div>
                    </div>

                    {/* Action Result (Octagon with Icon) */}
                    <div className="action-result">
                        <OctagonWrapper>
                            {action.icon}
                        </OctagonWrapper>
                    </div>
                </div>
            ))}
        </div>
    );
};
