package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lukev/tm_server/internal/az/actions"
	"github.com/lukev/tm_server/internal/az/env"
	"github.com/lukev/tm_server/internal/az/mcts"
	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

type BotGameConfig struct {
	PlayerID    string
	Faction     models.FactionType
	Simulations int
	BatchSize   int
	CPUCT       float64
	Temperature float64
	MaxDepth    int
	MoveDelayMs int
}

type BotManager struct {
	games         *game.Manager
	evaluator     model.Evaluator
	requireNeural bool

	mu      sync.Mutex
	configs map[string]BotGameConfig
	running map[string]bool
}

func NewBotManager(games *game.Manager) *BotManager {
	evaluator := model.LoadEvaluator(model.EvaluatorConfig{HTTPURL: os.Getenv("TM_AZ_MODEL_URL")})
	return &BotManager{
		games:         games,
		evaluator:     evaluator,
		requireNeural: strings.EqualFold(strings.TrimSpace(os.Getenv("TM_AZ_REQUIRE_NEURAL")), "true"),
		configs:       make(map[string]BotGameConfig),
		running:       make(map[string]bool),
	}
}

func (b *BotManager) RegisterGame(gameID string, config BotGameConfig) {
	if b == nil || gameID == "" || config.PlayerID == "" {
		return
	}
	config = normalizeBotConfig(config)
	b.mu.Lock()
	b.configs[gameID] = config
	b.mu.Unlock()
}

func (b *BotManager) Trigger(gameID string, hub *Hub) {
	if b == nil || b.games == nil || hub == nil || gameID == "" {
		return
	}
	b.mu.Lock()
	if _, ok := b.configs[gameID]; !ok {
		b.mu.Unlock()
		return
	}
	if b.running[gameID] {
		b.mu.Unlock()
		return
	}
	b.running[gameID] = true
	b.mu.Unlock()

	go func() {
		defer func() {
			b.mu.Lock()
			delete(b.running, gameID)
			b.mu.Unlock()
		}()
		b.run(gameID, hub)
	}()
}

func (b *BotManager) run(gameID string, hub *Hub) {
	for step := 0; step < 80; step++ {
		config, ok := b.configFor(gameID)
		if !ok {
			return
		}
		gs, ok := b.games.GetGame(gameID)
		if !ok || gs == nil || gs.Phase == game.PhaseEnd {
			b.broadcastStatus(hub, gameID, config.PlayerID, false, "")
			return
		}
		if !botCanAct(gs, config.PlayerID) {
			b.broadcastStatus(hub, gameID, config.PlayerID, false, "")
			return
		}
		revision, ok := b.games.GetRevision(gameID)
		if !ok {
			return
		}
		position := env.NewPosition(gs, config.PlayerID)
		legal := position.LegalActions()
		if len(legal) == 0 || legal[0].PlayerID != config.PlayerID {
			b.broadcastStatus(hub, gameID, config.PlayerID, false, "")
			return
		}

		b.broadcastStatus(hub, gameID, config.PlayerID, true, "")
		if config.MoveDelayMs > 0 {
			time.Sleep(time.Duration(config.MoveDelayMs) * time.Millisecond)
		}

		option, label, err := b.chooseAction(position, legal, config)
		if err != nil {
			b.broadcastStatus(hub, gameID, config.PlayerID, false, err.Error())
			log.Printf("bot action selection failed for game %s: %v", gameID, err)
			return
		}
		result, err := b.games.ExecuteActionWithMeta(gameID, option.Action, game.ActionMeta{
			ActionID:         fmt.Sprintf("bot:%s:%d:%s", config.PlayerID, revision, option.ID),
			ExpectedRevision: revision,
			SeatID:           config.PlayerID,
		})
		if err != nil {
			b.broadcastStatus(hub, gameID, config.PlayerID, false, err.Error())
			log.Printf("bot action execution failed for game %s: %v", gameID, err)
			return
		}
		if result != nil && result.Duplicate {
			b.broadcastStatus(hub, gameID, config.PlayerID, false, "")
			return
		}
		b.broadcastGameState(hub, gameID)
		b.broadcastStatus(hub, gameID, config.PlayerID, false, label)
		time.Sleep(25 * time.Millisecond)
	}
	log.Printf("bot action loop reached safety limit for game %s", gameID)
}

func botCanAct(gs *game.GameState, playerID string) bool {
	if gs == nil || strings.TrimSpace(playerID) == "" {
		return false
	}
	pendingPlayerID := strings.TrimSpace(gs.PendingTurnConfirmationPlayerID)
	return pendingPlayerID == "" || pendingPlayerID == strings.TrimSpace(playerID)
}

func (b *BotManager) configFor(gameID string) (BotGameConfig, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	config, ok := b.configs[gameID]
	return config, ok
}

func (b *BotManager) chooseAction(position *env.Position, legal []actions.Option, config BotGameConfig) (actions.Option, string, error) {
	if position == nil || position.State == nil {
		return actions.Option{}, "", fmt.Errorf("missing position")
	}
	if position.State.Phase == game.PhaseFactionSelection && config.Faction != models.FactionUnknown {
		for _, option := range legal {
			if selected, ok := option.Action.(*game.SelectFactionAction); ok && selected.FactionType == config.Faction {
				return option, option.Label, nil
			}
		}
	}

	failures := model.FailureCount(b.evaluator)
	result := mcts.Search(position, b.evaluator, mcts.Config{
		Simulations: config.Simulations,
		BatchSize:   config.BatchSize,
		CPUCT:       config.CPUCT,
		Temperature: config.Temperature,
		MaxDepth:    config.MaxDepth,
	})
	if b.requireNeural && model.FailureCount(b.evaluator) != failures {
		return actions.Option{}, "", fmt.Errorf("neural evaluator failed during search")
	}
	if result.Selected.ID == "" {
		return actions.Option{}, "", fmt.Errorf("search did not select an action")
	}
	for _, option := range legal {
		if option.ID == result.Selected.ID {
			return option, result.Selected.Label, nil
		}
	}
	return actions.Option{}, "", fmt.Errorf("selected action is no longer legal: %s", result.Selected.ID)
}

func (b *BotManager) broadcastGameState(hub *Hub, gameID string) {
	gameState := b.games.SerializeGameState(gameID)
	if gameState == nil {
		return
	}
	stateMsg, _ := json.Marshal(map[string]any{
		"type":    "game_state_update",
		"payload": gameState,
	})
	hub.BroadcastToGame(gameID, stateMsg)
	if pendingDecision, ok := gameState["pendingDecision"]; ok && pendingDecision != nil {
		decisionMsg, _ := json.Marshal(map[string]any{
			"type":    "decision_required",
			"payload": pendingDecision,
		})
		hub.BroadcastToGame(gameID, decisionMsg)
	}
}

func (b *BotManager) broadcastStatus(hub *Hub, gameID, playerID string, thinking bool, lastMove string) {
	msg, _ := json.Marshal(map[string]any{
		"type": "bot_status",
		"payload": map[string]any{
			"gameId":   gameID,
			"playerId": playerID,
			"thinking": thinking,
			"lastMove": lastMove,
		},
	})
	hub.BroadcastToGame(gameID, msg)
}

func normalizeBotConfig(config BotGameConfig) BotGameConfig {
	if config.Simulations <= 0 {
		config.Simulations = 64
	}
	if config.BatchSize < 0 {
		config.BatchSize = 0
	}
	if config.CPUCT <= 0 {
		config.CPUCT = 1.5
	}
	if config.Temperature < 0 {
		config.Temperature = 0
	}
	if config.MaxDepth <= 0 {
		config.MaxDepth = 500
	}
	if config.MoveDelayMs < 0 {
		config.MoveDelayMs = 0
	}
	return config
}
