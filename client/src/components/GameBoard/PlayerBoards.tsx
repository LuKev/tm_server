import React from 'react';
import { useGameStore } from '../../stores/gameStore';
import { GamePhase, BuildingType, FactionType } from '../../types/game.types';
import { FACTION_BOARDS, type BuildingSlot } from '../../data/factionBoards';
import { FACTIONS } from '../../data/factions';
import { CoinIcon, WorkerIcon, PriestIcon, PowerIcon, DwellingIcon, TradingHouseIcon, TempleIcon, StrongholdIcon, SanctuaryIcon } from '../shared/Icons';
import { FACTION_COLORS } from '../../utils/colors';
import './PlayerBoards.css';

const IncomeDisplay: React.FC<{ income: BuildingSlot['income']; compact?: boolean }> = ({ income, compact }) => {
    if (!income) return null;
    const scale = compact ? 0.8 : 1;
    const style = { transform: `scale(${scale})` };

    return (
        <div className="income-reveal" style={compact ? { gap: '0' } : undefined}>
            {income.workers && <WorkerIcon style={style}>{income.workers}</WorkerIcon>}
            {income.coins && <CoinIcon style={style}>{income.coins}</CoinIcon>}
            {income.priests && <PriestIcon style={{ width: '1.5em', height: '1.5em', ...style }}>{income.priests}</PriestIcon>}
            {income.power && <PowerIcon amount={income.power} style={style} />}
            {income.powerTokens && <div style={{ fontSize: '0.75em' }}>+{income.powerTokens} PW</div>}
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
}> = ({ slot, type, faction, isBuilt }) => {
    const renderIcon = () => {
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

// ... inside PlayerBoard component ...



const PlayerBoard: React.FC<{ playerId: string; turnOrder: number }> = ({ playerId, turnOrder }) => {
    const gameState = useGameStore(s => s.gameState);
    const player = gameState?.players[playerId];

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
        if (typeof player.faction === 'string') {
            const factionName = player.faction;
            const found = FACTIONS.find(f => f.type === factionName || f.name === factionName);
            if (found) factionType = found.id;
        } else if (typeof player.faction === 'object' && 'type' in player.faction) {
            // @ts-ignore - we know it has type
            factionType = player.faction.type;
        }
    }
    const boardLayout = FACTION_BOARDS[factionType] || FACTION_BOARDS[FactionType.Nomads]; // Fallback
    const factionColor = FACTION_COLORS[factionType];

    // Count built buildings
    const buildings = Object.values(gameState?.map?.hexes || {})
        .map(h => h.building)
        .filter(b => b && b.ownerPlayerId === playerId);

    const dwellingCount = buildings.filter(b => b?.type === BuildingType.Dwelling).length;
    const tradingHouseCount = buildings.filter(b => b?.type === BuildingType.TradingHouse).length;
    const templeCount = buildings.filter(b => b?.type === BuildingType.Temple).length;
    const sanctuaryCount = buildings.filter(b => b?.type === BuildingType.Sanctuary).length;
    const strongholdCount = buildings.filter(b => b?.type === BuildingType.Stronghold).length;

    return (
        <div className="pb-resize-container">
            <div className="player-board-section" style={{ borderLeft: `5px solid ${factionColor}` }}>
                {/* Row 1: Header */}
                <div className="pb-header">
                    <div className="turn-order-badge">{turnOrder}</div>
                    <div className="pb-player-name">{player.name} ({FactionType[factionType]})</div>

                    <div className="resource-display">
                        <div className="resource-item"><CoinIcon /> {player.resources.coins}</div>
                        <div className="resource-item"><WorkerIcon /> {player.resources.workers}</div>
                        <div className="resource-item"><PriestIcon style={{ width: '1.5em', height: '1.5em' }} /> {player.resources.priests}</div>
                        <div className="resource-item">
                            <div className="pb-power-bowl">
                                <span>{player.resources.powerI}/{player.resources.powerII}/{player.resources.powerIII}</span>
                            </div>
                        </div>
                        <div className="resource-item ml-auto">
                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.25em', fontWeight: 'bold' }}>
                                <span>{player.VictoryPoints || 0} VP</span>
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
                                    <div className="pb-slot-sh-sa">
                                        <BuildingTrackSlot
                                            slot={boardLayout.stronghold}
                                            type={BuildingType.Stronghold}
                                            faction={factionType}
                                            isBuilt={strongholdCount > 0}
                                        />
                                    </div>
                                </div>

                                {/* Row 2: Trading Houses (4) */}
                                <div className="pb-building-row">
                                    <CostIndicator
                                        workers={boardLayout.tradingHouses[0].cost.workers}
                                        coins={boardLayout.tradingHouses[0].cost.coins}
                                    />
                                    {boardLayout.tradingHouses.map((slot, i) => (
                                        <div key={`tp-${i}`} className="pb-slot-tp-temple">
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
                                        <div key={`temple-${i}`} className="pb-slot-tp-temple">
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
                                    <div key={`dw-${i}`} className="pb-slot-dwelling">
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

                    {/* Column 2: Conversions */}
                    <div className="pb-conversions-col">
                        <div className="pb-section-title">Conversions</div>
                        <div className="conversion-area">
                            <button className="conversion-btn">1 Priest → 1 Worker</button>
                            <button className="conversion-btn">1 Worker → 1 Coin</button>
                            <button className="conversion-btn">5 PW → 1 Priest</button>
                            <button className="conversion-btn">3 PW → 1 Worker</button>
                            <button className="conversion-btn">1 PW → 1 Coin</button>
                            {factionType === FactionType.Alchemists && (
                                <button className="conversion-btn special">1 VP → 1 Worker</button>
                            )}
                        </div>
                    </div>

                    {/* Column 3: Favor Tiles */}
                    <div className="pb-favors-col">
                        <div className="pb-section-title">Favor Tiles</div>
                        <div className="favor-tiles-area">
                            {/* TODO: Render actual favor tiles from player state */}
                            <div className="pb-empty-text">None</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export const PlayerBoards: React.FC = () => {
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
                // 1.5% of width, matching 1.5cqw
                const newSize = Math.max(width * 0.015, 8); // Minimum 8px
                setScaleFontSize(newSize);
            }
        });

        observer.observe(containerRef.current);
        return () => observer.disconnect();
    }, []);

    if (!gameState || !gameState.players) return null;

    // Only show after faction selection
    if (gameState.phase === GamePhase.FactionSelection) {
        return (
            <div className="pb-waiting">
                Waiting for all players to select factions...
            </div>
        );
    }

    // Sort players by turn order
    const sortedPlayerIds = [...(gameState.order || [])];

    // If order is not set yet (should be), fallback to keys
    if (sortedPlayerIds.length === 0) {
        Object.keys(gameState.players).forEach(id => sortedPlayerIds.push(id));
    }

    return (
        <div className="pb-resize-container" ref={containerRef}>
            <div
                className="player-boards-container"
                style={{ fontSize: `${scaleFontSize}px` }}
            >
                {sortedPlayerIds.map((pid, index) => (
                    <PlayerBoard key={pid} playerId={pid} turnOrder={index + 1} />
                ))}
            </div>
        </div>
    );
};
