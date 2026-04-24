package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
)

type replayAutoCostPlan struct {
	burn            int
	powerToPriests  int
	priestsToWorker int
	powerToWorkers  int
	workersToCoins  int
	powerToCoins    int
}

func (gs *GameState) canAffordWithReplayAutoConversions(player *Player, cost factions.Cost) bool {
	if player == nil || player.Resources == nil {
		return false
	}
	if !gs.allowsReplayAutoConversions(cost) {
		return player.Resources.CanAfford(cost)
	}
	_, ok := planReplayAutoCost(player.Resources, cost)
	return ok
}

func (gs *GameState) spendWithReplayAutoConversions(player *Player, cost factions.Cost) error {
	if player == nil || player.Resources == nil {
		return fmt.Errorf("player has no resources")
	}
	if !gs.allowsReplayAutoConversions(cost) {
		return player.Resources.Spend(cost)
	}

	plan, ok := planReplayAutoCost(player.Resources, cost)
	if !ok {
		return player.Resources.Spend(cost)
	}
	if err := plan.Apply(player.Resources); err != nil {
		return err
	}
	return player.Resources.Spend(cost)
}

func (gs *GameState) prepareReplayAutoConversions(player *Player, cost factions.Cost) error {
	if player == nil || player.Resources == nil {
		return fmt.Errorf("player has no resources")
	}
	if !gs.allowsReplayAutoConversions(cost) {
		return nil
	}
	plan, ok := planReplayAutoCost(player.Resources, cost)
	if !ok {
		return nil
	}
	return plan.Apply(player.Resources)
}

func (gs *GameState) allowsReplayAutoConversions(cost factions.Cost) bool {
	if gs == nil || gs.ReplayMode == nil || !gs.ReplayMode["__replay__"] {
		return false
	}
	// Keep replay auto-funding scoped to non-power costs. Logged power spending
	// already has dedicated replay handling via explicit burn/power actions.
	return cost.Power == 0
}

func (p replayAutoCostPlan) Apply(resources *ResourcePool) error {
	if resources == nil {
		return fmt.Errorf("nil resources")
	}
	if p.burn > 0 {
		if err := resources.BurnPower(p.burn); err != nil {
			return err
		}
	}
	if p.powerToPriests > 0 {
		if err := resources.ConvertPowerToPriests(p.powerToPriests); err != nil {
			return err
		}
	}
	if p.priestsToWorker > 0 {
		if err := resources.ConvertPriestToWorker(p.priestsToWorker); err != nil {
			return err
		}
	}
	if p.powerToWorkers > 0 {
		if err := resources.ConvertPowerToWorkers(p.powerToWorkers); err != nil {
			return err
		}
	}
	if p.workersToCoins > 0 {
		if err := resources.ConvertWorkerToCoin(p.workersToCoins); err != nil {
			return err
		}
	}
	if p.powerToCoins > 0 {
		if err := resources.ConvertPowerToCoins(p.powerToCoins); err != nil {
			return err
		}
	}
	return nil
}

func planReplayAutoCost(resources *ResourcePool, cost factions.Cost) (replayAutoCostPlan, bool) {
	if resources == nil {
		return replayAutoCostPlan{}, false
	}
	if resources.CanAfford(cost) {
		return replayAutoCostPlan{}, true
	}

	clone := resources.Clone()
	plan := replayAutoCostPlan{}
	reservedPower := cost.Power

	ensureSpendablePower := func(required int) bool {
		needed := required - clone.Power.Bowl3
		if needed <= 0 {
			return true
		}
		if !clone.Power.CanBurn(needed) {
			return false
		}
		if err := clone.Power.BurnPower(needed); err != nil {
			return false
		}
		plan.burn += needed
		return true
	}
	convertPowerToPriests := func(amount int) bool {
		if amount <= 0 {
			return true
		}
		required := reservedPower + amount*5
		if !ensureSpendablePower(required) {
			return false
		}
		if err := clone.ConvertPowerToPriests(amount); err != nil {
			return false
		}
		plan.powerToPriests += amount
		return true
	}
	convertPowerToWorkers := func(amount int) bool {
		if amount <= 0 {
			return true
		}
		required := reservedPower + amount*3
		if !ensureSpendablePower(required) {
			return false
		}
		if err := clone.ConvertPowerToWorkers(amount); err != nil {
			return false
		}
		plan.powerToWorkers += amount
		return true
	}
	convertPowerToCoins := func(amount int) bool {
		if amount <= 0 {
			return true
		}
		required := reservedPower + amount
		if !ensureSpendablePower(required) {
			return false
		}
		if err := clone.ConvertPowerToCoins(amount); err != nil {
			return false
		}
		plan.powerToCoins += amount
		return true
	}
	convertPriestsToWorkers := func(amount int) bool {
		if amount <= 0 {
			return true
		}
		if err := clone.ConvertPriestToWorker(amount); err != nil {
			return false
		}
		plan.priestsToWorker += amount
		return true
	}
	convertWorkersToCoins := func(amount int) bool {
		if amount <= 0 {
			return true
		}
		if err := clone.ConvertWorkerToCoin(amount); err != nil {
			return false
		}
		plan.workersToCoins += amount
		return true
	}

	if priestShortfall := cost.Priests - clone.Priests; priestShortfall > 0 {
		if !convertPowerToPriests(priestShortfall) {
			return replayAutoCostPlan{}, false
		}
	}

	if workerShortfall := cost.Workers - clone.Workers; workerShortfall > 0 {
		availablePriests := maxInt(0, clone.Priests-cost.Priests)
		usePriests := minInt(workerShortfall, availablePriests)
		if usePriests > 0 && !convertPriestsToWorkers(usePriests) {
			return replayAutoCostPlan{}, false
		}
		workerShortfall = cost.Workers - clone.Workers
		if workerShortfall > 0 && !convertPowerToWorkers(workerShortfall) {
			return replayAutoCostPlan{}, false
		}
	}

	if coinShortfall := cost.Coins - clone.Coins; coinShortfall > 0 {
		availableWorkers := maxInt(0, clone.Workers-cost.Workers)
		useWorkers := minInt(coinShortfall, availableWorkers)
		if useWorkers > 0 && !convertWorkersToCoins(useWorkers) {
			return replayAutoCostPlan{}, false
		}

		coinShortfall = cost.Coins - clone.Coins
		if coinShortfall > 0 {
			availablePriests := maxInt(0, clone.Priests-cost.Priests)
			usePriests := minInt(coinShortfall, availablePriests)
			if usePriests > 0 {
				if !convertPriestsToWorkers(usePriests) || !convertWorkersToCoins(usePriests) {
					return replayAutoCostPlan{}, false
				}
			}
		}

		coinShortfall = cost.Coins - clone.Coins
		if coinShortfall > 0 && !convertPowerToCoins(coinShortfall) {
			return replayAutoCostPlan{}, false
		}
	}

	return plan, clone.CanAfford(cost)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
