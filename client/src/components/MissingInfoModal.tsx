import React, { useState } from 'react';
import { Modal } from './shared/Modal';

export interface MissingInfo {
    GlobalBonusCards: boolean;
    GlobalScoringTiles: boolean;
    BonusCardSelections: Record<number, Record<string, boolean> | undefined>;
    PlayerFactions: Record<string, boolean>;
}

export interface MissingInfoData {
    scoringTiles: string[];
    bonusCards: string[];
    bonusCardSelections: Record<string, Record<string, string>>;
}

interface MissingInfoModalProps {
    isOpen: boolean;
    missingInfo: MissingInfo | null;
    players?: string[];
    availableBonusCards?: string[];
    onSubmit: (data: MissingInfoData) => void;
    onClose: () => void;
}

export const MissingInfoModal: React.FC<MissingInfoModalProps> = ({ isOpen, missingInfo, players, availableBonusCards, onSubmit, onClose }) => {
    const [scoringTiles, setScoringTiles] = useState<string[]>(Array(6).fill(''));
    const [bonusCards, setBonusCards] = useState<string[]>([]);
    const [playerBonusCards, setPlayerBonusCards] = useState<Record<string, Record<string, string> | undefined>>({});

    // Scoring tiles with cleaner display names - grouped by building type
    const SCORING_TILES = [
        // Dwelling scoring
        { value: "SCORE3", label: "Dwelling scoring, Cult Water bonus" },
        { value: "SCORE5", label: "Dwelling scoring, Cult Fire bonus" },
        // Trading House scoring
        { value: "SCORE6", label: "Trading House scoring, Cult Water bonus" },
        { value: "SCORE8", label: "Trading House scoring, Cult Air bonus" },
        // Stronghold/Sanctuary scoring
        { value: "SCORE4", label: "Stronghold/Sanctuary scoring, Cult Fire bonus" },
        { value: "SCORE7", label: "Stronghold/Sanctuary scoring, Cult Air bonus" },
        // Temple scoring
        { value: "SCORE9", label: "Temple scoring, Priest bonus" },
        // Other scoring
        { value: "SCORE1", label: "Spade scoring, Cult Earth bonus" },
        { value: "SCORE2", label: "Town scoring, Cult Earth bonus" },
    ];

    // Check for duplicate scoring tiles
    const hasDuplicateScoringTiles = (): boolean => {
        const selectedTiles = scoringTiles.filter(t => t !== '');
        const uniqueTiles = new Set(selectedTiles);
        return selectedTiles.length !== uniqueTiles.size;
    };

    // Get list of already-selected tiles (for disabling in dropdowns)
    const getSelectedTilesExcept = (currentIndex: number): string[] => {
        return scoringTiles.filter((t, i) => t !== '' && i !== currentIndex);
    };

    // Bonus cards with cleaner display names
    const BONUS_CARDS = [
        { value: "BON-SPD", label: "Spade" },
        { value: "BON-4C", label: "Cult Advance + 4 Coins" },
        { value: "BON-6C", label: "6 Coins" },
        { value: "BON-SHIP", label: "Temporary Ship" },
        { value: "BON-WP", label: "Worker + Power" },
        { value: "BON-TP", label: "Trading House scoring" },
        { value: "BON-BB", label: "Stronghold/Sanctuary scoring" },
        { value: "BON-P", label: "Priest" },
        { value: "BON-DW", label: "Dwelling scoring" },
        { value: "BON-SHIP-VP", label: "Ship scoring" }
    ];

    // Calculate expected bonus card count: players + 3 (default to 7 for 4 players if players not known)
    const numPlayers = players?.length ?? 4;
    const expectedBonusCardCount = numPlayers + 3;

    const bonusCardOptions = missingInfo?.GlobalBonusCards
        ? bonusCards
        : (availableBonusCards && availableBonusCards.length > 0 ? availableBonusCards : BONUS_CARDS.map(c => c.value));

    const handleSubmit = (): void => {
        // Prevent submission if there are duplicate tiles
        if (hasDuplicateScoringTiles()) {
            alert('Please select unique scoring tiles for each round. Duplicates are not allowed.');
            return;
        }
        const data = {
            scoringTiles: scoringTiles.filter(t => t),
            bonusCards: bonusCards,
            bonusCardSelections: playerBonusCards as Record<string, Record<string, string>>,
        };
        onSubmit(data);
    };

    if (!missingInfo) return null;

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Missing Game Information">
            <div className="space-y-6">
                <p>The game log is missing some information. Please provide it below to continue.</p>

                {missingInfo.GlobalScoringTiles && (
                    <div>
                        <h3 className="font-semibold mb-2">Scoring Tiles (Round 1-6)</h3>
                        {hasDuplicateScoringTiles() && (
                            <p className="text-red-500 text-sm mb-2">⚠️ Duplicate tiles selected! Each round must have a unique tile.</p>
                        )}
                        {scoringTiles.map((tile, i) => {
                            const selectedElsewhere = getSelectedTilesExcept(i);
                            return (
                                <div key={i} className="flex gap-2 mb-2 items-center">
                                    <label className="w-20 text-sm">Round {i + 1}:</label>
                                    <select
                                        value={tile}
                                        onChange={(e) => {
                                            const newTiles = [...scoringTiles];
                                            newTiles[i] = e.target.value;
                                            setScoringTiles(newTiles);
                                        }}
                                        className="border p-1 rounded flex-1 text-sm"
                                    >
                                        <option value="">Select Tile...</option>
                                        {SCORING_TILES.map(t => (
                                            <option 
                                                key={t.value} 
                                                value={t.value}
                                                disabled={selectedElsewhere.includes(t.value)}
                                            >
                                                {t.label}{selectedElsewhere.includes(t.value) ? ' (used)' : ''}
                                            </option>
                                        ))}
                                    </select>
                                </div>
                            );
                        })}
                    </div>
                )}

                {missingInfo.GlobalBonusCards && (
                    <div>
                        <h3 className="font-semibold mb-2">Bonus Cards in Play</h3>
                        <p className="text-sm text-gray-500 mb-3">
                            Selected: {bonusCards.length}/{expectedBonusCardCount}
                        </p>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                            {BONUS_CARDS.map(card => (
                                <label key={card.value} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                    <input
                                        type="checkbox"
                                        checked={bonusCards.includes(card.value)}
                                        onChange={(e) => {
                                            if (e.target.checked) {
                                                if (bonusCards.length < expectedBonusCardCount) {
                                                    setBonusCards([...bonusCards, card.value]);
                                                }
                                            } else {
                                                setBonusCards(bonusCards.filter(c => c !== card.value));
                                            }
                                        }}
                                        disabled={!bonusCards.includes(card.value) && bonusCards.length >= expectedBonusCardCount}
                                    />
                                    <span>{card.label}</span>
                                </label>
                            ))}
                        </div>
                    </div>
                )}

                {/* Bonus Card Selections (Initial or Pass) */}
                {Object.keys(missingInfo.BonusCardSelections).map(roundStr => {
                    const round = parseInt(roundStr);
                    const selections = missingInfo.BonusCardSelections[round];
                    if (!selections || Object.keys(selections).length === 0 || !players || players.length === 0) return null;

                    const title = round === 0 ? "Initial Bonus Card Selection" : `Round ${String(round)} Bonus Card Selection`;
                    // Filter players to only those who need to select
                    const playersToSelect = players.filter(p => selections[p]);

                    return (
                        <div key={round}>
                            <h3>{title}</h3>
                            <p className="text-sm text-gray-500 mb-2">Select the bonus card for each player.</p>
                            {playersToSelect.map(player => (
                                <div key={player} className="flex gap-2 mb-2 items-center">
                                    <label className="w-32">{player}:</label>
                                    <select
                                        value={playerBonusCards[roundStr]?.[player] ?? ''}
                                        onChange={(e) => {
                                            setPlayerBonusCards({
                                                ...playerBonusCards,
                                                [roundStr]: {
                                                    ...(playerBonusCards[roundStr] ?? {}),
                                                    [player]: e.target.value
                                                }
                                            });
                                        }}
                                        className="border p-1 rounded flex-1"
                                    >
                                        <option value="">Select Bonus Card...</option>
                                        {bonusCardOptions.map(c => (
                                            <option key={c} value={c}>{c}</option>
                                        ))}
                                    </select>
                                </div>
                            ))}
                        </div>
                    );
                })}

                <div className="flex justify-end pt-4">
                    <button
                        className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700"
                        onClick={handleSubmit}
                    >
                        Submit Information
                    </button>
                </div>
            </div>
        </Modal>
    );
};
