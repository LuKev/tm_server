import React from 'react';
import { useGameStore } from '../../stores/gameStore';
import { GamePhase, BuildingType, FactionType, SpecialActionType, FavorTileType, BonusCardType, type PlayerState } from '../../types/game.types';
import { FACTION_BOARDS, type BuildingSlot } from '../../data/factionBoards';
import { FACTIONS } from '../../data/factions';
import { CoinIcon, WorkerIcon, PriestIcon, PowerIcon, PowerCircleIcon, DwellingIcon, TradingHouseIcon, TempleIcon, StrongholdIcon, SanctuaryIcon, CultRhombusIcon } from '../shared/Icons';
import { FACTION_COLORS } from '../../utils/colors';
import { FAVOR_TILES, getCultColorClass } from '../../data/favorTiles';
import { TownTileId } from '../../types/game.types';
import { ShippingDiggingDisplay } from '../shared/ShippingDiggingDisplay';
import './PlayerBoards.css';
import './FavorTiles.css';
import './TownTiles.css';

// Helper to get Stronghold Action Type for a faction
const getStrongholdActionType = (faction: FactionType): SpecialActionType | null => {
    switch (faction) {
        case FactionType.Auren: return SpecialActionType.AurenCultAdvance;
        case FactionType.Witches: return SpecialActionType.WitchesRide;
        case FactionType.Swarmlings: return SpecialActionType.SwarmlingsUpgrade;
        case FactionType.ChaosMagicians: return SpecialActionType.ChaosMagiciansDoubleTurn;
        case FactionType.Giants: return SpecialActionType.GiantsTransform;
        case FactionType.Nomads: return SpecialActionType.NomadsSandstorm;
        default: return null;
    }
};

// Helper to get town tile config with VP and rewards
const getTownTileConfig = (tileId: TownTileId): { vp: number; rewards: React.ReactNode } => {
    switch (tileId) {
        case TownTileId.Vp5Coins6:
            return { vp: 5, rewards: <CoinIcon className="icon-sm">6</CoinIcon> };
        case TownTileId.Vp6Power8:
            return { vp: 6, rewards: <PowerIcon amount={8} className="icon-sm" /> };
        case TownTileId.Vp7Workers2:
            return { vp: 7, rewards: <WorkerIcon className="icon-sm">2</WorkerIcon> };
        case TownTileId.Vp4Ship1:
            return { vp: 4, rewards: <span style={{ fontSize: '0.6em' }}>Ship</span> };
        case TownTileId.Vp8Cult1:
            return { vp: 8, rewards: <CultRhombusIcon className="icon-sm" /> };
        case TownTileId.Vp9Priest1:
            return { vp: 9, rewards: <PriestIcon className="icon-sm" /> };
        case TownTileId.Vp11:
            return { vp: 11, rewards: null };
        case TownTileId.Vp2Cult2:
            return { vp: 2, rewards: <CultRhombusIcon className="icon-sm" showNumber={true} /> };
        default:
            return { vp: 0, rewards: null };
    }
};

const StrongholdOctagon: React.FC<{ isUsed?: boolean; onClick?: () => void; disabled?: boolean }> = ({ isUsed, onClick, disabled }) => (
    <button
        type="button"
        onClick={onClick}
        disabled={disabled}
        style={{
            position: 'relative',
            width: '2.4rem',
            height: '2.4rem',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'transparent',
            border: 'none',
            padding: 0,
            cursor: disabled ? 'not-allowed' : 'pointer',
            opacity: disabled ? 0.6 : 1,
        }}
    >
        <svg viewBox="-2 -2 44 44" style={{ width: '100%', height: '100%', filter: 'drop-shadow(0 1px 1px rgba(0,0,0,0.1))' }}>
            <path
                d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z"
                fill="#f97316" // orange-500
                stroke="#c2410c" // orange-700
                strokeWidth="2"
            />
        </svg>
        {isUsed && (
            <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 10, pointerEvents: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <svg viewBox="-2 -2 44 44" style={{ width: '100%', height: '100%', display: 'block' }}>
                    <path d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z" fill="#d6d3d1" stroke="#78716c" strokeWidth="2" fillOpacity="0.9" />
                    <path d="M 10 10 L 30 30 M 30 10 L 10 30" stroke="#78716c" strokeWidth="3" strokeLinecap="round" />
                </svg>
            </div>
        )}
    </button>
);

const StrongholdSquare: React.FC<{ onClick?: () => void; disabled?: boolean; label?: string }> = ({ onClick, disabled, label }) => (
    <button
        type="button"
        onClick={onClick}
        disabled={disabled}
        title={label}
        style={{
            width: '2.4rem',
            height: '2.4rem',
            borderRadius: '0.3rem',
            border: '2px solid #c2410c',
            background: '#f97316',
            color: '#111827',
            fontSize: '0.75rem',
            fontWeight: 700,
            cursor: disabled ? 'not-allowed' : 'pointer',
            opacity: disabled ? 0.6 : 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            boxShadow: '0 1px 2px rgba(0,0,0,0.15)',
        }}
    >
        {label ?? 'ACT'}
    </button>
);

const IncomeDisplay: React.FC<{ income: BuildingSlot['income']; compact?: boolean }> = ({ income, compact }) => {
    if (!income) return null;
    const scale = compact ? 0.8 : 1;
    const style = { transform: `scale(${String(scale)})` };

    return (
        <div className="income-reveal" style={compact ? { gap: '0' } : undefined}>
            {income.workers && <WorkerIcon style={style}>{income.workers}</WorkerIcon>}
            {income.coins && <CoinIcon style={style}>{income.coins}</CoinIcon>}
            {income.priests && <PriestIcon style={{ width: '1.5em', height: '1.5em', ...style }}>{income.priests}</PriestIcon>}
            {income.power && <PowerIcon amount={income.power} style={style} />}

        </div>
    );
};

const CostIndicator: React.FC<{ workers?: number; coins?: number }> = ({ workers, coins }) => {
    if (!workers && !coins) return null;
    return (
        <div className="cost-indicator">
            {workers && <div className="cost-worker">{workers}</div>}
            {coins && <div className="cost-coin">{coins}</div>}
        </div>
    );
};

const BuildingTrackSlot: React.FC<{
    slot: BuildingSlot;
    type: BuildingType;
    faction: FactionType;
    isBuilt: boolean;
}> = ({ slot, type, faction, isBuilt }): React.ReactElement => {
    const renderIcon = (): React.ReactElement | null => {
        const className = "building-piece";
        const color = FACTION_COLORS[faction];
        switch (type) {
            case BuildingType.Dwelling: return <DwellingIcon className={className} color={color} />;
            case BuildingType.TradingHouse: return <TradingHouseIcon className={className} color={color} />;
            case BuildingType.Temple: return <TempleIcon className={className} color={color} />;
            case BuildingType.Stronghold: return <StrongholdIcon className={className} color={color} />;
            case BuildingType.Sanctuary: return <SanctuaryIcon className={className} color={color} />;
            default: return null;
        }
    };

    return (
        <div className={`building-slot ${isBuilt ? 'built' : 'unbuilt'}`}>
            {/* Income is always rendered behind, revealed when built or hovered */}
            <IncomeDisplay income={slot.income} compact={type === BuildingType.TradingHouse} />

            {/* Piece is rendered on top, fades out on hover */}
            {!isBuilt && renderIcon()}
        </div>
    );
};

interface PlayerBoardProps {
    playerId: string;
    turnOrder: number | string;
    isCurrentPlayer?: boolean;
    isReplayMode?: boolean;
    onConversion?: (playerId: string, conversionType: string) => void;
    onBurnPower?: (playerId: string, amount: number) => void;
    onAdvanceShipping?: (playerId: string) => void;
    onAdvanceDigging?: (playerId: string) => void;
    onStrongholdAction?: (playerId: string, actionType: SpecialActionType) => void;
    onEngineersBridgeAction?: (playerId: string) => void;
    onMermaidsConnectAction?: (playerId: string) => void;
    onWater2Action?: (playerId: string) => void;
}

const PlayerBoard: React.FC<PlayerBoardProps> = ({
    playerId,
    turnOrder,
    isCurrentPlayer,
    isReplayMode,
    onConversion,
    onBurnPower,
    onAdvanceShipping,
    onAdvanceDigging,
    onStrongholdAction,
    onEngineersBridgeAction,
    onMermaidsConnectAction,
    onWater2Action
}) => {
    const gameState = useGameStore(s => s.gameState);
    const localPlayerId = useGameStore(s => s.localPlayerId);
    const player: PlayerState | undefined = gameState?.players[playerId];

    if (!player || (!player.Faction && !player.faction)) return null;

    // Resolve FactionType safely
    let factionType: FactionType = FactionType.Nomads;

    // Check 'Faction' (uppercase)
    if (player.Faction) {
        if (typeof player.Faction === 'object' && 'Type' in player.Faction) {
            factionType = player.Faction.Type;
        } else if (typeof player.Faction === 'number') {
            factionType = player.Faction;
        } else if (typeof player.Faction === 'string') {
            const factionName = player.Faction;
            const found = FACTIONS.find(f => f.type === factionName || f.name === factionName);
            if (found) factionType = found.id;
        }
    }
    // Check 'faction' (lowercase) fallback
    else if (player.faction) {
        // The Go struct has json:"faction", so this is the likely path.
        // However, the embedded BaseFaction fields (like Type) are exported but untagged,
        // so they serialize as "Type" (uppercase).
        if (typeof player.faction === 'object') {
            // Check for "Type" (uppercase) which comes from Go serialization
            if ('Type' in player.faction) {
                factionType = (player.faction as { Type: number }).Type;
            }
            // Check for "type" (lowercase) just in case
            else if ('type' in player.faction) {
                factionType = (player.faction as { type: FactionType }).type;
            }
        } else if (typeof player.faction === 'string') {
            const factionName = player.faction;
            const found = FACTIONS.find(f => f.type === factionName || f.name === factionName);
            if (found) factionType = found.id;
        } else if (typeof player.faction === 'number') {
            factionType = player.faction;
        }
    }
    const boardLayout = FACTION_BOARDS[factionType];
    const factionColor = FACTION_COLORS[factionType];

    // Count built buildings
    const buildings = Object.values(gameState?.map.hexes ?? {})
        .map(h => h.building)
        .filter(b => b && b.ownerPlayerId === playerId);

    const dwellingCount = buildings.filter(b => b?.type === BuildingType.Dwelling).length;
    const tradingHouseCount = buildings.filter(b => b?.type === BuildingType.TradingHouse).length;
    const templeCount = buildings.filter(b => b?.type === BuildingType.Temple).length;
    const sanctuaryCount = buildings.filter(b => b?.type === BuildingType.Sanctuary).length;
    const strongholdCount = buildings.filter(b => b?.type === BuildingType.Stronghold).length;

    const strongholdActionType = getStrongholdActionType(factionType);
    const isStrongholdActionUsed = strongholdActionType !== null && player.specialActionsUsed?.[strongholdActionType];
    const isLocalPlayer = localPlayerId === playerId;
    const isEngineersSquareAction = factionType === FactionType.Engineers && !!player.hasStrongholdAbility;
    const isMermaidsSquareAction = factionType === FactionType.Mermaids && !!player.hasStrongholdAbility;

    const hasTempShippingBonus = gameState?.bonusCards?.playerCards?.[playerId] === BonusCardType.Shipping;
    const shippingLevel = (player as unknown as { shipping?: number }).shipping ?? 0;
    const diggingLevel = (player as unknown as { digging?: number }).digging ?? 0;

    return (
        <div className="pb-resize-container">
            <div
                className="player-board-section"
                style={{
                    borderLeft: `5px solid ${factionColor}`,
                    transition: 'all 0.3s ease',
                    boxShadow: isCurrentPlayer ? '0 0 0 4px #FACC15' : 'none', // Yellow-400 ring
                    zIndex: isCurrentPlayer ? 10 : 1
                }}
            >
                {/* Row 1: Header */}
                <div className="pb-header">
                    <div
                        className="turn-order-badge"
                        style={isCurrentPlayer ? { backgroundColor: '#FACC15', color: 'black' } : undefined}
                    >
                        {turnOrder}
                    </div>
                    <div className="pb-player-name">{player.name} ({FactionType[factionType]})</div>

                    <div className="resource-display">
                        <div className="resource-item"><CoinIcon /> {player.resources.coins}</div>
                        <div className="resource-item"><WorkerIcon /> {player.resources.workers}</div>
                        <div className="resource-item"><PriestIcon style={{ width: '1.5em', height: '1.5em' }} /> {player.resources.priests}</div>
                        <div className="resource-item">
                            <div className="pb-power-bowl">
                                <PowerCircleIcon style={{ width: '1.15em', height: '1.15em' }} />
                                <span>{player.resources.power.powerI}/{player.resources.power.powerII}/{player.resources.power.powerIII}</span>
                            </div>
                        </div>
                        <div className="resource-item">
                            <ShippingDiggingDisplay
                                factionType={factionType}
                                shipping={shippingLevel}
                                diggingLevel={diggingLevel}
                                hasTempShippingBonus={hasTempShippingBonus}
                            />
                            {!isReplayMode && isLocalPlayer && (
                                <div style={{ display: 'flex', gap: '0.25em', marginLeft: '0.5em' }}>
                                    <button
                                        type="button"
                                        className="conversion-btn"
                                        style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                        onClick={() => { onAdvanceShipping?.(playerId); }}
                                    >
                                        +Ship
                                    </button>
                                    <button
                                        type="button"
                                        className="conversion-btn"
                                        style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                        onClick={() => { onAdvanceDigging?.(playerId); }}
                                    >
                                        +Dig
                                    </button>
                                </div>
                            )}
                        </div>
                        <div className="resource-item" style={{ marginLeft: 'auto' }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.25em', fontWeight: 'bold' }}>
                                <span>{player.victoryPoints ?? player.VictoryPoints ?? 0} VP</span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Row 2: Main Content (Board | Conversions | Favor Tiles) */}
                <div className="pb-wrapper">

                    {/* Column 1: Player Board (Buildings) - Vertical Stack */}
                    <div className="pb-board-area">
                        <div className="pb-upper-section">
                            {/* Left Panel: SH, TPs */}
                            <div className="pb-panel-left">
                                {/* Row 1: Stronghold */}
                                <div style={{ display: 'flex', flexDirection: 'row', alignItems: 'center' }}>
                                    <CostIndicator
                                        workers={boardLayout.stronghold.cost.workers}
                                        coins={boardLayout.stronghold.cost.coins}
                                    />
                                    <div className="pb-slot-sh-sa" style={{ position: 'relative' }}>
                                        <BuildingTrackSlot
                                            slot={boardLayout.stronghold}
                                            type={BuildingType.Stronghold}
                                            faction={factionType}
                                            isBuilt={strongholdCount > 0}
                                        />
                                        {/* Stronghold Action Octagon */}
                                        {strongholdActionType !== null && (
                                            <div style={{ position: 'absolute', right: '-3em', top: '50%', transform: 'translateY(-50%)' }}>
                                                <StrongholdOctagon
                                                    isUsed={isStrongholdActionUsed}
                                                    onClick={() => { if (strongholdActionType !== null) onStrongholdAction?.(playerId, strongholdActionType); }}
                                                    disabled={!isLocalPlayer || !!isStrongholdActionUsed}
                                                />
                                            </div>
                                        )}
                                        {isEngineersSquareAction && (
                                            <div style={{ position: 'absolute', right: '-3em', top: '50%', transform: 'translateY(-50%)' }}>
                                                <StrongholdSquare
                                                    label="BR"
                                                    onClick={() => { onEngineersBridgeAction?.(playerId); }}
                                                    disabled={!isLocalPlayer}
                                                />
                                            </div>
                                        )}
                                        {isMermaidsSquareAction && (
                                            <div style={{ position: 'absolute', right: '-3em', top: '50%', transform: 'translateY(-50%)' }}>
                                                <StrongholdSquare
                                                    label="CT"
                                                    onClick={() => { onMermaidsConnectAction?.(playerId); }}
                                                    disabled={!isLocalPlayer}
                                                />
                                            </div>
                                        )}
                                    </div>
                                </div>

                                {/* Row 2: Trading Houses (4) */}
                                <div className="pb-building-row">
                                    <CostIndicator
                                        workers={boardLayout.tradingHouses[0].cost.workers}
                                        coins={boardLayout.tradingHouses[0].cost.coins}
                                    />
                                    {boardLayout.tradingHouses.map((slot, i) => (
                                        <div key={`tp-${String(i)}`} className="pb-slot-tp-temple">
                                            <BuildingTrackSlot
                                                slot={slot}
                                                type={BuildingType.TradingHouse}
                                                faction={factionType}
                                                isBuilt={tradingHouseCount > i}
                                            />
                                        </div>
                                    ))}
                                </div>
                            </div>

                            {/* Right Panel: SA, Temples */}
                            <div className="pb-panel-right">
                                {/* Row 1: Sanctuary */}
                                <div style={{ display: 'flex', flexDirection: 'row', alignItems: 'center' }}>
                                    <CostIndicator
                                        workers={boardLayout.sanctuary.cost.workers}
                                        coins={boardLayout.sanctuary.cost.coins}
                                    />
                                    <div className="pb-slot-sh-sa">
                                        <BuildingTrackSlot
                                            slot={boardLayout.sanctuary}
                                            type={BuildingType.Sanctuary}
                                            faction={factionType}
                                            isBuilt={sanctuaryCount > 0}
                                        />
                                    </div>
                                </div>

                                {/* Row 2: Temples (3) */}
                                <div className="pb-building-row">
                                    <CostIndicator
                                        workers={boardLayout.temples[0].cost.workers}
                                        coins={boardLayout.temples[0].cost.coins}
                                    />
                                    {boardLayout.temples.map((slot, i) => (
                                        <div key={`temple-${String(i)}`} className="pb-slot-tp-temple">
                                            <BuildingTrackSlot
                                                slot={slot}
                                                type={BuildingType.Temple}
                                                faction={factionType}
                                                isBuilt={templeCount > i}
                                            />
                                        </div>
                                    ))}
                                </div>
                            </div>
                        </div>

                        {/* Lower Section: Dwellings (8) */}
                        <div className="pb-lower-section">
                            <div className="pb-dwellings-row">
                                <CostIndicator
                                    workers={boardLayout.dwellings[0].cost.workers}
                                    coins={boardLayout.dwellings[0].cost.coins}
                                />

                                {/* Base Income Display */}
                                <div style={{ display: 'flex', alignItems: 'center', marginRight: '0.5em' }}>
                                    <WorkerIcon style={{ width: '1.2em', height: '1.2em' }}>
                                        {factionType === FactionType.Swarmlings ? 2 :
                                            (factionType === FactionType.Engineers) ? 0 : 1}
                                    </WorkerIcon>
                                </div>

                                {boardLayout.dwellings.map((slot, i) => (
                                    <div key={`dw-${String(i)}`} className="pb-slot-dwelling">
                                        <BuildingTrackSlot
                                            slot={slot}
                                            type={BuildingType.Dwelling}
                                            faction={factionType}
                                            isBuilt={dwellingCount > i}
                                        />
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>

                    {/* Column 2: Conversions (normal) or Towns (replay mode) */}
                    <div className="pb-conversions-col">
                        {isReplayMode ? (
                            <>
                                <div className="pb-section-title">Towns</div>
                                <div className="pb-towns-area">
                                    {(() => {
                                        const townTiles = player.townTiles ?? [];
                                        if (townTiles.length === 0) {
                                            return <div className="pb-empty-text">None</div>;
                                        }
                                        return (
                                            <div className="pb-towns-list">
                                                {townTiles.map((tileId, idx) => {
                                                    const config = getTownTileConfig(tileId as TownTileId);
                                                    return (
                                                        <div key={`town-${String(idx)}`} className="pb-town-slot">
                                                            <div className="town-tile">
                                                                <div className="town-tile-content">
                                                                    <div className="town-tile-top">
                                                                        <span className="vp-value">{config.vp}</span>
                                                                        <span className="vp-label">VP</span>
                                                                    </div>
                                                                    {config.rewards && (
                                                                        <div className="town-tile-bottom">
                                                                            {config.rewards}
                                                                        </div>
                                                                    )}
                                                                </div>
                                                            </div>
                                                        </div>
                                                    );
                                                })}
                                            </div>
                                        );
                                    })()}
                                </div>
                            </>
                        ) : (
                            <>
                                <div className="pb-section-title">Conversions</div>
                                <div className="conversion-area">
                                    <button className="conversion-btn" onClick={() => { onConversion?.(playerId, 'priest_to_worker'); }} disabled={!isLocalPlayer}>1 Priest → 1 Worker</button>
                                    <button className="conversion-btn" onClick={() => { onConversion?.(playerId, 'worker_to_coin'); }} disabled={!isLocalPlayer}>1 Worker → 1 Coin</button>
                                    <button className="conversion-btn" onClick={() => { onConversion?.(playerId, 'power_to_priest'); }} disabled={!isLocalPlayer}>5 PW → 1 Priest</button>
                                    <button className="conversion-btn" onClick={() => { onConversion?.(playerId, 'power_to_worker'); }} disabled={!isLocalPlayer}>3 PW → 1 Worker</button>
                                    <button className="conversion-btn" onClick={() => { onConversion?.(playerId, 'power_to_coin'); }} disabled={!isLocalPlayer}>1 PW → 1 Coin</button>
                                    <button className="conversion-btn" onClick={() => { onBurnPower?.(playerId, 1); }} disabled={!isLocalPlayer}>Burn 2PW → +1 Bowl III</button>
                                    {factionType === FactionType.Alchemists && (
                                        <button className="conversion-btn special" onClick={() => { onConversion?.(playerId, 'alchemists_vp_to_coin'); }} disabled={!isLocalPlayer}>1 VP → 1 Coin</button>
                                    )}
                                </div>
                            </>
                        )}
                    </div>

                    {/* Column 3: Favor Tiles & Bonus Card */}
                    <div className="pb-favors-col">
                        <div className="pb-section-title">Favor Tiles</div>
                        <div className="favor-tiles-area">
                            {(() => {
                                const playerTiles = gameState?.favorTiles?.playerTiles[playerId] ?? [];
                                if (playerTiles.length === 0) {
                                    return <div className="pb-empty-text">None</div>;
                                }

                                return (
                                    <div className="pb-favors-list">
                                        {playerTiles.map((tileType, idx) => {
                                            const tileData = FAVOR_TILES.find(t => t.type === tileType);
                                            if (!tileData) return null;

                                            const isWater2 = tileType === FavorTileType.Water2;
                                            const isUsed = isWater2 && player.specialActionsUsed?.[SpecialActionType.Water2CultAdvance];

                                            return (
                                                <div key={`${String(tileType)}-${String(idx)}`} className="pb-favor-tile">
                                                    <svg className="favor-tile-bg" viewBox="0 0 200 100" preserveAspectRatio="xMidYMid meet" style={{ zIndex: 0 }}>
                                                        <ellipse cx="100" cy="50" rx="98" ry="48"
                                                            className={`${getCultColorClass(tileData.cult)} opacity-20`}
                                                            stroke="currentColor"
                                                            strokeWidth="3"
                                                        />
                                                        <ellipse cx="100" cy="50" rx="95" ry="45"
                                                            className="fill-white"
                                                            fill="white"
                                                        />
                                                    </svg>

                                                    <div className={`favor-tile-content ${!tileData.reward ? 'justify-center' : ''}`}>
                                                        <div className={`favor-cult-steps ${!tileData.reward ? 'w-full' : ''}`}>
                                                            {Array.from({ length: tileData.steps }).map((_, i) => (
                                                                <div
                                                                    key={i}
                                                                    className={`cult-step-circle ${getCultColorClass(tileData.cult)}`}
                                                                />
                                                            ))}
                                                        </div>

                                                        {tileData.reward && (
                                                            <div className="favor-reward">
                                                                <div style={{ position: 'relative', display: 'inline-flex' }}>
                                                                    {tileData.reward}
                                                                    {isUsed && (
                                                                        <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', zIndex: 10, pointerEvents: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                                                            <svg viewBox="-2 -2 44 44" style={{ width: '100%', height: '100%', display: 'block' }} preserveAspectRatio="xMidYMid meet">
                                                                                <path d="M 12 0 L 28 0 L 40 12 L 40 28 L 28 40 L 12 40 L 0 28 L 0 12 Z" fill="#d6d3d1" stroke="#78716c" strokeWidth="2" fillOpacity="0.9" />
                                                                                <path d="M 10 10 L 30 30 M 30 10 L 10 30" stroke="#78716c" strokeWidth="3" strokeLinecap="round" />
                                                                            </svg>
                                                                        </div>
                                                                    )}
                                                                </div>
                                                            </div>
                                                        )}
                                                    </div>
                                                    {tileType === FavorTileType.Water2 && !isUsed && isLocalPlayer && (
                                                        <button
                                                            type="button"
                                                            onClick={() => { onWater2Action?.(playerId); }}
                                                            style={{
                                                                position: 'absolute',
                                                                inset: 0,
                                                                background: 'transparent',
                                                                border: 'none',
                                                                cursor: 'pointer',
                                                            }}
                                                        />
                                                    )}
                                                </div>
                                            );
                                        })}
                                    </div>
                                );
                            })()}
                        </div>


                    </div>
                </div>
            </div>
        </div>
    );
};

interface PlayerBoardsProps {
    isReplayMode?: boolean;
    onConversion?: (playerId: string, conversionType: string) => void;
    onBurnPower?: (playerId: string, amount: number) => void;
    onAdvanceShipping?: (playerId: string) => void;
    onAdvanceDigging?: (playerId: string) => void;
    onStrongholdAction?: (playerId: string, actionType: SpecialActionType) => void;
    onEngineersBridgeAction?: (playerId: string) => void;
    onMermaidsConnectAction?: (playerId: string) => void;
    onWater2Action?: (playerId: string) => void;
}

export const PlayerBoards: React.FC<PlayerBoardsProps> = ({
    isReplayMode,
    onConversion,
    onBurnPower,
    onAdvanceShipping,
    onAdvanceDigging,
    onStrongholdAction,
    onEngineersBridgeAction,
    onMermaidsConnectAction,
    onWater2Action
}) => {
    const gameState = useGameStore(s => s.gameState);

    // JS-based scaling to ensure reliability
    // Hooks must be called unconditionally
    const containerRef = React.useRef<HTMLDivElement>(null);
    const [scaleFontSize, setScaleFontSize] = React.useState<number>(16); // Default 16px

    React.useEffect(() => {
        if (!containerRef.current) return;

        const observer = new ResizeObserver((entries) => {
            for (const entry of entries) {
                const width = entry.contentRect.width;
                // 2% of width, minimum 10px
                const newSize = Math.max(width * 0.02, 10);
                setScaleFontSize(newSize);
            }
        });

        observer.observe(containerRef.current);
        return () => { observer.disconnect(); };
    }, [gameState?.phase, gameState?.players]); // Re-run when game state loads/changes

    if (!gameState?.players) return null;

    // Only show after faction selection
    if (gameState.phase === GamePhase.FactionSelection) {
        return (
            <div className="pb-waiting">
                Waiting for all players to select factions...
            </div>
        );
    }

    // Sort players by ID to keep board order stable across rounds
    const sortedPlayerIds = Object.keys(gameState.players).sort();

    // Determine current player based on turn order and current turn index
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    const turnOrderList = gameState.turnOrder ?? [];
    const currentPlayerId = turnOrderList[gameState.currentTurn];


    return (
        <div className="pb-resize-container" ref={containerRef}>
            <div
                className="player-boards-container"
                style={{ fontSize: `${String(scaleFontSize)}px` }}
            >
                {sortedPlayerIds.map((pid) => {
                    // Calculate turn order (1-based) for this player
                    const turnOrderIndex = turnOrderList.indexOf(pid);
                    const turnOrder = turnOrderIndex !== -1 ? turnOrderIndex + 1 : '-';
                    const isCurrentPlayer = pid === currentPlayerId;

                    return (
                        <PlayerBoard
                            key={pid}
                            playerId={pid}
                            turnOrder={turnOrder}
                            isCurrentPlayer={isCurrentPlayer}
                            isReplayMode={isReplayMode}
                            onConversion={onConversion}
                            onBurnPower={onBurnPower}
                            onAdvanceShipping={onAdvanceShipping}
                            onAdvanceDigging={onAdvanceDigging}
                            onStrongholdAction={onStrongholdAction}
                            onEngineersBridgeAction={onEngineersBridgeAction}
                            onMermaidsConnectAction={onMermaidsConnectAction}
                            onWater2Action={onWater2Action}
                        />
                    );
                })}
            </div>
        </div>
    );
};
