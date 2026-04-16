import React from 'react';
import { useGameStore } from '../../stores/gameStore';
import { GamePhase, BuildingType, FactionType, SpecialActionType, FavorTileType, BonusCardType, type PlayerState, type TurnTimerState } from '../../types/game.types';
import { FACTION_BOARDS, type BuildingSlot } from '../../data/factionBoards';
import { FACTIONS } from '../../data/factions';
import { CoinIcon, WorkerIcon, PriestIcon, PowerIcon, PowerCircleIcon, DwellingIcon, TradingHouseIcon, TempleIcon, StrongholdIcon, SanctuaryIcon, CultRhombusIcon, ShippingIcon } from '../shared/Icons';
import { FACTION_COLORS } from '../../utils/colors';
import { FAVOR_TILES, getCultColorClass } from '../../data/favorTiles';
import { TownTileId } from '../../types/game.types';
import { ShippingDiggingDisplay, canShowDiggingForFaction, canShowShippingForFaction } from '../shared/ShippingDiggingDisplay';
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
        case FactionType.TheEnlightened: return SpecialActionType.EnlightenedGainPower;
        case FactionType.Conspirators: return SpecialActionType.ConspiratorsSwapFavor;
        case FactionType.ChildrenOfTheWyrm: return SpecialActionType.ChildrenPlacePowerTokens;
        case FactionType.Prospectors: return SpecialActionType.ProspectorsGainCoins;
        case FactionType.TimeTravelers: return SpecialActionType.TimeTravelersPowerShift;
        case FactionType.Architects: return SpecialActionType.ArchitectsMoveBridge;
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
            return { vp: 4, rewards: <ShippingIcon className="icon-sm" /> };
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

const StrongholdOctagon: React.FC<{ isUsed?: boolean; isActive?: boolean; onClick?: () => void; disabled?: boolean; testId?: string }> = ({ isUsed, isActive, onClick, disabled, testId }) => (
    <button
        type="button"
        data-testid={testId}
        onClick={onClick}
        disabled={disabled}
        className={isActive ? 'pb-special-action-active' : 'pb-special-action-hover'}
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
            borderRadius: '0.35rem',
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

const renderIncomeIcons = (income: BuildingSlot['income'], compact?: boolean): React.ReactNode => {
    if (!income) return null;
    const scale = compact ? 0.8 : 1;
    const style = { transform: `scale(${String(scale)})` };

    return (
        <>
            {income.workers && <WorkerIcon style={style}>{income.workers}</WorkerIcon>}
            {income.coins && <CoinIcon style={style}>{income.coins}</CoinIcon>}
            {income.priests && <PriestIcon style={{ width: '1.5em', height: '1.5em', ...style }}>{income.priests}</PriestIcon>}
            {income.power && <PowerIcon amount={income.power} style={style} />}
        </>
    );
};

const IncomeDisplay: React.FC<{ income: BuildingSlot['income']; compact?: boolean }> = ({ income, compact }) => {
    if (!income) return null;
    return (
        <div className="income-reveal" style={compact ? { gap: '0' } : undefined}>
            {renderIncomeIcons(income, compact)}
        </div>
    );
};

const InlineIncomeDisplay: React.FC<{ income: BuildingSlot['income'] }> = ({ income }) => {
    if (!income) return null;
    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.125em' }}>
            {renderIncomeIcons(income)}
        </div>
    );
};

const getBaseIncome = (faction: FactionType): BuildingSlot['income'] => {
    switch (faction) {
        case FactionType.Engineers:
        case FactionType.Treasurers:
            return null;
        case FactionType.Swarmlings:
        case FactionType.Archivists:
        case FactionType.DynionGeifr:
            return { workers: 2 };
        case FactionType.TheEnlightened:
            return { power: 3 };
        default:
            return { workers: 1 };
    }
};

const CostIndicator: React.FC<{ workers?: number; coins?: number; priests?: number; power?: number }> = ({ workers, coins, priests, power }) => {
    if (!workers && !coins && !priests && !power) return null;
    return (
        <div className="cost-indicator">
            {workers && <WorkerIcon style={{ width: '1.25em', height: '1.25em' }}>{workers}</WorkerIcon>}
            {coins && <CoinIcon style={{ width: '1.25em', height: '1.25em' }}>{coins}</CoinIcon>}
            {priests && <PriestIcon style={{ width: '1.25em', height: '1.25em' }}>{priests}</PriestIcon>}
            {power && <PowerIcon amount={power} style={{ width: '1.25em', height: '1.25em' }} />}
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

const formatTimerRemaining = (remainingMs: number): string => {
    const sign = remainingMs < 0 ? '-' : '';
    const absoluteSeconds = Math.max(0, Math.ceil(Math.abs(remainingMs) / 1000));
    const minutes = Math.floor(absoluteSeconds / 60);
    const seconds = absoluteSeconds % 60;
    return `${sign}${String(minutes)}:${String(seconds).padStart(2, '0')}`;
};

const getDisplayedRemainingMs = (
    turnTimer: TurnTimerState | null | undefined,
    playerId: string,
    nowMs: number,
): number | null => {
    if (!turnTimer) return null;
    const playerTimer = turnTimer.players?.[playerId];
    if (!playerTimer) return null;
    if (!playerTimer.isActive) return playerTimer.remainingMs;
    return playerTimer.remainingMs - Math.max(0, nowMs - turnTimer.serverNowMs);
};

interface PlayerBoardProps {
    playerId: string;
    turnOrder: number | string;
    displayNowMs?: number;
    isCurrentPlayer?: boolean;
    isReplayMode?: boolean;
    canUseTurnActions?: boolean;
    canUseConversions?: boolean;
    onConversion?: (playerId: string, conversionType: string) => void;
    onBurnPower?: (playerId: string, amount: number) => void;
    onAdvanceShipping?: (playerId: string) => void;
    onAdvanceDigging?: (playerId: string) => void;
    onAdvanceChashTrack?: (playerId: string) => void;
    onStrongholdAction?: (playerId: string, actionType: SpecialActionType) => void;
    onGoblinsTreasureAction?: (playerId: string) => void;
    onDjinniLampAction?: (playerId: string) => void;
    onEngineersBridgeAction?: (playerId: string) => void;
    onMermaidsConnectAction?: (playerId: string) => void;
    onWater2Action?: (playerId: string) => void;
    activeStrongholdActionType?: SpecialActionType | null;
    isEngineersBridgeActive?: boolean;
    isMermaidsConnectActive?: boolean;
    isWater2Active?: boolean;
}

const PlayerBoard: React.FC<PlayerBoardProps> = ({
    playerId,
    turnOrder,
    displayNowMs = Date.now(),
    isCurrentPlayer,
    isReplayMode,
    canUseTurnActions = false,
    canUseConversions = false,
    onConversion,
    onBurnPower,
    onAdvanceShipping,
    onAdvanceDigging,
    onAdvanceChashTrack,
    onStrongholdAction,
    onGoblinsTreasureAction,
    onDjinniLampAction,
    onEngineersBridgeAction,
    onMermaidsConnectAction,
    onWater2Action,
    activeStrongholdActionType,
    isEngineersBridgeActive,
    isMermaidsConnectActive,
    isWater2Active
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
    const childrenBoardPowerTokens = Object.values(gameState?.map.hexes ?? {}).filter(h => h.powerTokenOwnerPlayerId === playerId).length;
    const goblinTreasureTokens = player.goblinTreasureTokens ?? 0;
    const djinniLampTokens = player.djinniLampTokens ?? 0;
    const treasuryCoins = player.treasuryCoins ?? 0;
    const treasuryWorkers = player.treasuryWorkers ?? 0;
    const treasuryPriests = player.treasuryPriests ?? 0;

    const dwellingCount = buildings.filter(b => b?.type === BuildingType.Dwelling).length;
    const tradingHouseCount = buildings.filter(b => b?.type === BuildingType.TradingHouse).length;
    const templeCount = buildings.filter(b => b?.type === BuildingType.Temple).length;
    const sanctuaryCount = buildings.filter(b => b?.type === BuildingType.Sanctuary).length;
    const strongholdCount = buildings.filter(b => b?.type === BuildingType.Stronghold).length;

    const strongholdActionType = getStrongholdActionType(factionType);
    const isStrongholdActionUsed = strongholdActionType !== null && player.specialActionsUsed?.[strongholdActionType];
    const isLocalPlayer = localPlayerId === playerId;
    const pendingDecision = gameState?.pendingDecision as { type?: string; playerId?: string } | null | undefined;
    const isLocalPostActionFreeWindow = isLocalPlayer
        && pendingDecision?.playerId === playerId
        && pendingDecision?.type === 'post_action_free_actions';
    const conversionActionsEnabled = canUseConversions || isLocalPostActionFreeWindow;
    const hasReusableBridgeAction = factionType === FactionType.Engineers || factionType === FactionType.Atlanteans || factionType === FactionType.Architects;
    const hasMermaidsConnectAction = factionType === FactionType.Mermaids && !!player.hasStrongholdAbility;
    const hasGoblinsTreasureAction = factionType === FactionType.Goblins;
    const isStrongholdActionActive = strongholdActionType !== null && activeStrongholdActionType === strongholdActionType && isLocalPlayer;
    const isLocalEngineersBridgeActive = !!isEngineersBridgeActive && isLocalPlayer;
    const isLocalMermaidsConnectActive = !!isMermaidsConnectActive && isLocalPlayer;
    const isLocalWater2Active = !!isWater2Active && isLocalPlayer;

    const heldBonusCards = [
        ...(gameState?.bonusCards?.playerCards?.[playerId] !== undefined ? [gameState.bonusCards.playerCards[playerId]] : []),
        ...((gameState?.bonusCards?.playerExtraCards?.[playerId] ?? []) as BonusCardType[]),
    ];
    const hasTempShippingBonus = heldBonusCards.includes(BonusCardType.Shipping);
    const shippingLevel = (player as unknown as { shipping?: number }).shipping ?? 0;
    const diggingLevel = (player as unknown as { digging?: number }).digging ?? 0;
    const showShippingUpgrade = canShowShippingForFaction(factionType);
    const showDiggingUpgrade = canShowDiggingForFaction(factionType);
    const showChashTrackUpgrade = factionType === FactionType.ChashDallah;
    const chashTrackLevel = player.chashIncomeTrackLevel ?? 0;
    const townTiles = player.townTiles ?? [];
    const displayedRemainingMs = getDisplayedRemainingMs(gameState?.turnTimer ?? null, playerId, displayNowMs);
    const timerIsActive = !!gameState?.turnTimer?.players?.[playerId]?.isActive;

    const renderTownTiles = (): React.ReactNode => {
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
    };

    return (
        <div className="pb-resize-container">
            <div
                className="player-board-section"
                data-testid={`player-board-${playerId}`}
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
                    {displayedRemainingMs !== null && (
                        <div
                            className="resource-item"
                            data-testid={`player-${playerId}-turn-timer`}
                            style={{
                                fontWeight: timerIsActive ? 700 : 500,
                                color: timerIsActive ? '#0f766e' : '#334155',
                            }}
                        >
                            {formatTimerRemaining(displayedRemainingMs)}
                        </div>
                    )}

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
                        {factionType === FactionType.Goblins && (
                            <div className="resource-item">
                                <span style={{ fontWeight: 600 }}>Treasure {goblinTreasureTokens}</span>
                            </div>
                        )}
                        {factionType === FactionType.ChildrenOfTheWyrm && (
                            <div className="resource-item">
                                <span style={{ fontWeight: 600 }}>Board PW {childrenBoardPowerTokens}</span>
                            </div>
                        )}
                        {factionType === FactionType.Djinni && (
                            <div className="resource-item">
                                <span style={{ fontWeight: 600 }}>Lamps {djinniLampTokens}</span>
                            </div>
                        )}
                        {factionType === FactionType.Treasurers && (
                            <div className="resource-item">
                                <span style={{ fontWeight: 600 }}>Treasury {treasuryCoins}/{treasuryWorkers}/{treasuryPriests}</span>
                            </div>
                        )}
                        <div className="resource-item">
                            <ShippingDiggingDisplay
                                factionType={factionType}
                                shipping={shippingLevel}
                                diggingLevel={diggingLevel}
                                hasTempShippingBonus={hasTempShippingBonus}
                            />
                            {!isReplayMode && isLocalPlayer && (showShippingUpgrade || showDiggingUpgrade || showChashTrackUpgrade || hasReusableBridgeAction || hasMermaidsConnectAction || hasGoblinsTreasureAction) && (
                                <div style={{ display: 'flex', gap: '0.25em', marginLeft: '0.5em' }}>
                                    {showShippingUpgrade && (
                                        <button
                                            type="button"
                                            data-testid={`player-${playerId}-upgrade-shipping`}
                                            className="conversion-btn"
                                            style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                            onClick={() => { onAdvanceShipping?.(playerId); }}
                                            disabled={!canUseTurnActions}
                                        >
                                            +Ship
                                        </button>
                                    )}
                                    {showDiggingUpgrade && (
                                        <button
                                            type="button"
                                            data-testid={`player-${playerId}-upgrade-digging`}
                                            className="conversion-btn"
                                            style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                            onClick={() => { onAdvanceDigging?.(playerId); }}
                                            disabled={!canUseTurnActions}
                                        >
                                            +Dig
                                        </button>
                                    )}
                                    {showChashTrackUpgrade && (
                                        <button
                                            type="button"
                                            data-testid={`player-${playerId}-advance-chash-track`}
                                            className="conversion-btn"
                                            style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                            onClick={() => { onAdvanceChashTrack?.(playerId); }}
                                            disabled={!canUseTurnActions}
                                            title={`Chash income track ${String(chashTrackLevel)}/4`}
                                        >
                                            +Track {String(chashTrackLevel)}/4
                                        </button>
                                    )}
                                    {hasReusableBridgeAction && (
                                        <button
                                            type="button"
                                            data-testid={`player-${playerId}-bridge-action`}
                                            className={`conversion-btn ${isLocalEngineersBridgeActive ? 'special' : ''}`}
                                            style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                            onClick={() => { onEngineersBridgeAction?.(playerId); }}
                                            disabled={!canUseTurnActions}
                                            title={factionType === FactionType.Architects ? 'Build a bridge for 1 priest' : factionType === FactionType.Atlanteans ? 'Build a bridge for 2 workers' : 'Build a bridge for 2 workers'}
                                        >
                                            BR
                                        </button>
                                    )}
                                    {hasMermaidsConnectAction && (
                                        <button
                                            type="button"
                                            data-testid={`player-${playerId}-mermaids-connect`}
                                            className={`conversion-btn ${isLocalMermaidsConnectActive ? 'special' : ''}`}
                                            style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                            onClick={() => { onMermaidsConnectAction?.(playerId); }}
                                            disabled={!canUseTurnActions}
                                            title="Mermaids stronghold: connect across one river"
                                        >
                                            CT
                                        </button>
                                    )}
                                    {hasGoblinsTreasureAction && (
                                        <button
                                            type="button"
                                            data-testid={`player-${playerId}-goblins-treasure`}
                                            className="conversion-btn special"
                                            style={{ padding: '0.1em 0.45em', fontSize: '0.75em' }}
                                            onClick={() => { onGoblinsTreasureAction?.(playerId); }}
                                            disabled={!canUseTurnActions || goblinTreasureTokens <= 0}
                                            title="Spend 1 Goblins treasure"
                                        >
                                            Treasure
                                        </button>
                                    )}
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
                                        priests={boardLayout.stronghold.cost.priests}
                                        power={boardLayout.stronghold.cost.power}
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
                                                    isActive={isStrongholdActionActive}
                                                    onClick={() => { if (strongholdActionType !== null) onStrongholdAction?.(playerId, strongholdActionType); }}
                                                    disabled={!isLocalPlayer || !canUseTurnActions || !!isStrongholdActionUsed}
                                                    testId={`player-${playerId}-stronghold-action`}
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
                                        priests={boardLayout.tradingHouses[0].cost.priests}
                                        power={boardLayout.tradingHouses[0].cost.power}
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
                                        priests={boardLayout.sanctuary.cost.priests}
                                        power={boardLayout.sanctuary.cost.power}
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
                                        priests={boardLayout.temples[0].cost.priests}
                                        power={boardLayout.temples[0].cost.power}
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
                                    priests={boardLayout.dwellings[0].cost.priests}
                                    power={boardLayout.dwellings[0].cost.power}
                                />

                                {/* Base Income Display */}
                                <div style={{ display: 'flex', alignItems: 'center', marginRight: '0.5em' }}>
                                    <InlineIncomeDisplay income={getBaseIncome(factionType)} />
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

                    {/* Column 2: Local players get conversions plus towns; others show towns */}
                    <div className="pb-conversions-col">
                        {(isReplayMode || !isLocalPlayer) ? (
                            <>
                                <div className="pb-section-title">Towns</div>
                                <div className="pb-towns-area">
                                    {renderTownTiles()}
                                </div>
                            </>
                        ) : (
                            <>
                                <div className="pb-section-title">Conversions</div>
                                <div className="conversion-area">
                                    <button data-testid={`player-${playerId}-conversion-priest_to_worker`} className="conversion-btn" onClick={() => { onConversion?.(playerId, 'priest_to_worker'); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>{factionType === FactionType.DynionGeifr ? '1 Priest → 2 Workers + 2 Coins' : '1 Priest → 1 Worker'}</button>
                                    <button data-testid={`player-${playerId}-conversion-worker_to_coin`} className="conversion-btn" onClick={() => { onConversion?.(playerId, 'worker_to_coin'); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>1 Worker → 1 Coin</button>
                                    <button data-testid={`player-${playerId}-conversion-power_to_priest`} className="conversion-btn" onClick={() => { onConversion?.(playerId, 'power_to_priest'); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>5 PW → {factionType === FactionType.TheEnlightened && player.hasStrongholdAbility ? '2 Priests' : '1 Priest'}</button>
                                    <button data-testid={`player-${playerId}-conversion-power_to_worker`} className="conversion-btn" onClick={() => { onConversion?.(playerId, 'power_to_worker'); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>3 PW → {factionType === FactionType.TheEnlightened && player.hasStrongholdAbility ? '2 Workers' : '1 Worker'}</button>
                                    <button data-testid={`player-${playerId}-conversion-power_to_coin`} className="conversion-btn" onClick={() => { onConversion?.(playerId, 'power_to_coin'); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>1 PW → {factionType === FactionType.TheEnlightened && player.hasStrongholdAbility ? '2 Coins' : '1 Coin'}</button>
                                    {factionType === FactionType.TheEnlightened && (
                                        <button data-testid={`player-${playerId}-conversion-coin_to_power`} className="conversion-btn special" onClick={() => { onConversion?.(playerId, 'coin_to_power'); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>1 Coin → 1 PW Token</button>
                                    )}
                                    <button data-testid={`player-${playerId}-burn-power-1`} className="conversion-btn" onClick={() => { onBurnPower?.(playerId, 1); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>{factionType === FactionType.ChildrenOfTheWyrm ? 'Burn 3PW → +2 Bowl III' : 'Burn 2PW → +1 Bowl III'}</button>
                                    {factionType === FactionType.Alchemists && (
                                        <button data-testid={`player-${playerId}-conversion-alchemists_vp_to_coin`} className="conversion-btn special" onClick={() => { onConversion?.(playerId, 'alchemists_vp_to_coin'); }} disabled={!isLocalPlayer || !conversionActionsEnabled}>1 VP → 1 Coin</button>
                                    )}
                                    {factionType === FactionType.Djinni && (
                                        <button data-testid={`player-${playerId}-djinni-lamp`} className="conversion-btn special" onClick={() => { onDjinniLampAction?.(playerId); }} disabled={!isLocalPlayer || !canUseTurnActions || djinniLampTokens <= 0}>Use 1 Lamp</button>
                                    )}
                                </div>
                                <div className="pb-section-title">Towns</div>
                                <div className="pb-towns-area">
                                    {renderTownTiles()}
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
                                                            data-testid={`player-${playerId}-water2-action`}
                                                            className={isLocalWater2Active ? 'pb-special-action-active' : 'pb-special-action-hover'}
                                                            onClick={() => { onWater2Action?.(playerId); }}
                                                            disabled={!canUseTurnActions}
                                                            style={{
                                                                position: 'absolute',
                                                                inset: 0,
                                                                background: 'transparent',
                                                                border: 'none',
                                                                cursor: canUseTurnActions ? 'pointer' : 'not-allowed',
                                                                borderRadius: '0.45rem',
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
    canUseTurnActions?: boolean;
    canUseConversions?: boolean;
    onConversion?: (playerId: string, conversionType: string) => void;
    onBurnPower?: (playerId: string, amount: number) => void;
    onAdvanceShipping?: (playerId: string) => void;
    onAdvanceDigging?: (playerId: string) => void;
    onAdvanceChashTrack?: (playerId: string) => void;
    onStrongholdAction?: (playerId: string, actionType: SpecialActionType) => void;
    onGoblinsTreasureAction?: (playerId: string) => void;
    onDjinniLampAction?: (playerId: string) => void;
    onEngineersBridgeAction?: (playerId: string) => void;
    onMermaidsConnectAction?: (playerId: string) => void;
    onWater2Action?: (playerId: string) => void;
    activeStrongholdActionType?: SpecialActionType | null;
    isEngineersBridgeActive?: boolean;
    isMermaidsConnectActive?: boolean;
    isWater2Active?: boolean;
}

export const PlayerBoards: React.FC<PlayerBoardsProps> = ({
    isReplayMode,
    canUseTurnActions = false,
    canUseConversions = false,
    onConversion,
    onBurnPower,
    onAdvanceShipping,
    onAdvanceDigging,
    onAdvanceChashTrack,
    onStrongholdAction,
    onGoblinsTreasureAction,
    onDjinniLampAction,
    onEngineersBridgeAction,
    onMermaidsConnectAction,
    onWater2Action,
    activeStrongholdActionType,
    isEngineersBridgeActive,
    isMermaidsConnectActive,
    isWater2Active
}) => {
    const gameState = useGameStore(s => s.gameState);
    const [displayNowMs, setDisplayNowMs] = React.useState(() => Date.now());

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

    React.useEffect(() => {
        const hasActiveTimer = Object.values(gameState?.turnTimer?.players ?? {}).some((playerTimer) => playerTimer?.isActive);
        if (!hasActiveTimer) return;
        const interval = window.setInterval(() => {
            setDisplayNowMs(Date.now());
        }, 1000);
        return () => { window.clearInterval(interval); };
    }, [gameState?.turnTimer]);

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
        <div className="pb-resize-container" ref={containerRef} data-testid="player-boards">
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
                            displayNowMs={displayNowMs}
                            isCurrentPlayer={isCurrentPlayer}
                            isReplayMode={isReplayMode}
                            canUseTurnActions={canUseTurnActions}
                            canUseConversions={canUseConversions}
                            onConversion={onConversion}
                            onBurnPower={onBurnPower}
                            onAdvanceShipping={onAdvanceShipping}
                            onAdvanceDigging={onAdvanceDigging}
                            onAdvanceChashTrack={onAdvanceChashTrack}
                            onStrongholdAction={onStrongholdAction}
                            onGoblinsTreasureAction={onGoblinsTreasureAction}
                            onDjinniLampAction={onDjinniLampAction}
                            onEngineersBridgeAction={onEngineersBridgeAction}
                            onMermaidsConnectAction={onMermaidsConnectAction}
                            onWater2Action={onWater2Action}
                            activeStrongholdActionType={activeStrongholdActionType}
                            isEngineersBridgeActive={isEngineersBridgeActive}
                            isMermaidsConnectActive={isMermaidsConnectActive}
                            isWater2Active={isWater2Active}
                        />
                    );
                })}
            </div>
        </div>
    );
};
