import React from 'react';

interface ReplayControlsProps {
    onStart: (restart?: boolean) => void;
    onNext: () => void;
    onToggleAutoPlay: () => void;
    isAutoPlaying: boolean;
    currentIndex: number;
    totalActions: number;
    gameId: string;
}

export const ReplayControls: React.FC<ReplayControlsProps> = ({
    onStart,
    onNext,
    onToggleAutoPlay,
    isAutoPlaying,
    currentIndex,
    totalActions,
    gameId,
}) => {
    return (
        <div className="flex items-center gap-4 bg-white p-4 rounded-lg shadow-md mb-4">
            <div className="text-lg font-bold text-gray-700">
                Replay: {gameId}
            </div>

            <div className="flex-1" />

            <div className="text-sm text-gray-600 font-mono">
                Action: {currentIndex} / {totalActions}
            </div>

            <div className="flex gap-2">
                <button
                    onClick={() => onStart(true)}
                    className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded font-medium transition-colors"
                >
                    Restart
                </button>

                <button
                    onClick={onNext}
                    disabled={isAutoPlaying || currentIndex >= totalActions}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    Next
                </button>

                <button
                    onClick={onToggleAutoPlay}
                    disabled={currentIndex >= totalActions}
                    className={`px-4 py-2 rounded font-medium transition-colors text-white ${isAutoPlaying
                        ? 'bg-red-500 hover:bg-red-600'
                        : 'bg-purple-600 hover:bg-purple-700'
                        } disabled:opacity-50 disabled:cursor-not-allowed`}
                >
                    {isAutoPlaying ? 'Pause' : 'Auto-Play'}
                </button>
            </div>
        </div>
    );
};
