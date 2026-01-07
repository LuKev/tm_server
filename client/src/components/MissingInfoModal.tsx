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
    const [bonusCards, setBonusCards] = useState<string[]>(Array(10).fill(''));
    const [playerBonusCards, setPlayerBonusCards] = useState<Record<string, Record<string, string> | undefined>>({});

    // Simple hardcoded options for now
    // Corrected mappings based on server/internal/replay/game_setup.go
    const SCORING_TILES = [
        "SCORE1 (Spade -> 2VP, 1 Earth -> 1C)",
        "SCORE2 (Town -> 5VP, 4 Earth -> 1 Spade)",
        "SCORE3 (Dwelling -> 2VP, 4 Water -> 1 Priest)",
        "SCORE4 (SH/SA -> 5VP, 2 Fire -> 1 Worker)",
        "SCORE5 (Dwelling -> 2VP, 4 Fire -> 4 Power)",
        "SCORE6 (TP -> 3VP, 4 Water -> 1 Spade)",
        "SCORE7 (SH/SA -> 5VP, 2 Air -> 1 Worker)",
        "SCORE8 (TP -> 3VP, 4 Air -> 1 Spade)",
        "SCORE9 (Temple -> 4VP, 1 Priest -> 2C)"
    ];

    const BONUS_CARDS = [
        "BON-SPD (Spade)",
        "BON-4C (Cult Advance)",
        "BON-6C (6 Coins)",
        "BON-SHIP (Shipping)",
        "BON-WP (Worker Power)",
        "BON-TP (Trading House VP)",
        "BON-BB (Stronghold/Sanctuary VP)",
        "BON-P (Priest)",
        "BON-DW (Dwelling VP)",
        "BON-SHIP-VP (Shipping VP)"
    ];

    const bonusCardOptions = missingInfo?.GlobalBonusCards
        ? bonusCards.filter(c => c)
        : (availableBonusCards && availableBonusCards.length > 0 ? availableBonusCards : BONUS_CARDS);

    const handleSubmit = (): void => {
        const data = {
            scoringTiles: scoringTiles.filter(t => t),
            bonusCards: bonusCards.filter(c => c),
            bonusCardSelections: playerBonusCards as Record<string, Record<string, string>>,
        };
        onSubmit(data);
    };

    if (!missingInfo) return null;

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Missing Game Information">
            <div className="space-y-4">
                <p>The game log is missing some information. Please provide it below to continue.</p>

                {missingInfo.GlobalScoringTiles && (
                    <div>
                        <h3>Scoring Tiles (Round 1-6)</h3>
                        {scoringTiles.map((tile, i) => (
                            <div key={i} className="flex gap-2 mb-2">
                                <label>Round {i + 1}:</label>
                                <select
                                    value={tile}
                                    onChange={(e) => {
                                        const newTiles = [...scoringTiles];
                                        newTiles[i] = e.target.value;
                                        setScoringTiles(newTiles);
                                    }}
                                    className="border p-1 rounded"
                                >
                                    <option value="">Select Tile...</option>
                                    {SCORING_TILES.map(t => (
                                        <option key={t} value={t}>{t}</option>
                                    ))}
                                </select>
                            </div>
                        ))}
                    </div>
                )}

                {missingInfo.GlobalBonusCards && (
                    <div>
                        <h3>Bonus Cards in Play</h3>
                        <p className="text-sm text-gray-500 mb-2">
                            Select {players ? players.length + 3 : 3} bonus cards used in this game.
                        </p>
                        <div className="grid grid-cols-2 gap-2">
                            {BONUS_CARDS.map(card => (
                                <label key={card} className="flex items-center gap-2">
                                    <input
                                        type="checkbox"
                                        checked={bonusCards.includes(card)}
                                        onChange={(e) => {
                                            if (e.target.checked) {
                                                if (bonusCards.filter(c => c).length < (players ? players.length + 3 : 10)) {
                                                    setBonusCards([...bonusCards.filter(c => c), card]);
                                                }
                                            } else {
                                                setBonusCards(bonusCards.filter(c => c !== card));
                                            }
                                        }}
                                    />
                                    <span className="text-sm">{card}</span>
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
