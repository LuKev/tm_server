package game

import (
	"fmt"

	"github.com/lukev/tm_server/internal/game/factions"
	"github.com/lukev/tm_server/internal/models"
)

type replayAutoCostPlan struct {
	vpToCoins       int
	burn            int
	powerToPriests  int
	priestsToWorker int
	powerToWorkers  int
	workersToCoins  int
	powerToCoins    int
	steps           []replayAutoCostStep
}

type replayAutoCostStep struct {
	burn       int
	conversion ConversionType
	amount     int
}

func (gs *GameState) canAffordWithReplayAutoConversions(player *Player, cost factions.Cost) bool {
	if player == nil || player.Resources == nil {
		return false
	}
	if !gs.allowsReplayAutoConversions(cost) {
		return player.Resources.CanAfford(cost)
	}
	_, ok := planReplayAutoCost(gs, player, cost)
	return ok
}

func (gs *GameState) spendWithReplayAutoConversions(player *Player, cost factions.Cost) error {
	if player == nil || player.Resources == nil {
		return fmt.Errorf("player has no resources")
	}
	if !gs.allowsReplayAutoConversions(cost) {
		return player.Resources.Spend(cost)
	}

	plan, ok := planReplayAutoCost(gs, player, cost)
	if !ok {
		return player.Resources.Spend(cost)
	}
	if err := plan.Apply(gs, player); err != nil {
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
	plan, ok := planReplayAutoCost(gs, player, cost)
	if !ok {
		return nil
	}
	return plan.Apply(gs, player)
}

func (gs *GameState) allowsReplayAutoConversions(cost factions.Cost) bool {
	if gs == nil {
		return false
	}
	replayAutoConversions := gs.ReplayMode != nil && gs.ReplayMode["__replay__"] && gs.ReplayMode["__bga__"]
	if !gs.allowAZAutoConversions && !replayAutoConversions {
		return false
	}
	// Keep replay auto-funding scoped to non-power costs. Logged power spending
	// already has dedicated replay handling via explicit burn/power actions.
	return cost.Power == 0
}

// EnableAZAutoConversionsForClone enables funding only on a disposable AZ state clone.
func EnableAZAutoConversionsForClone(gs *GameState) {
	if gs != nil {
		gs.allowAZAutoConversions = true
	}
}

func (p replayAutoCostPlan) Apply(gs *GameState, player *Player) error {
	if gs == nil || player == nil || player.Resources == nil {
		return fmt.Errorf("nil game state or resources")
	}
	for _, step := range p.steps {
		if step.burn > 0 {
			if err := (&BurnPowerAction{BaseAction: BaseAction{Type: ActionBurnPower, PlayerID: player.ID}, Amount: step.burn}).Execute(gs); err != nil {
				return err
			}
			continue
		}
		if step.amount <= 0 {
			continue
		}
		action := &ConversionAction{
			BaseAction:     BaseAction{Type: ActionConversion, PlayerID: player.ID},
			ConversionType: step.conversion,
			Amount:         step.amount,
		}
		if err := action.Execute(gs); err != nil {
			return err
		}
	}
	return nil
}

func planReplayAutoCost(gs *GameState, player *Player, cost factions.Cost) (replayAutoCostPlan, bool) {
	if gs == nil || player == nil || player.Resources == nil {
		return replayAutoCostPlan{}, false
	}
	if player.Resources.CanAfford(cost) {
		return replayAutoCostPlan{}, true
	}
	cloneState := gs.CloneForUndo()
	clonePlayer := cloneState.GetPlayer(player.ID)
	if clonePlayer == nil || clonePlayer.Resources == nil {
		return replayAutoCostPlan{}, false
	}
	clone := clonePlayer.Resources
	plan := replayAutoCostPlan{}

	applyBurn := func(amount int) bool {
		if amount <= 0 {
			return true
		}
		action := &BurnPowerAction{BaseAction: BaseAction{Type: ActionBurnPower, PlayerID: player.ID}, Amount: amount}
		if err := action.Execute(cloneState); err != nil {
			return false
		}
		plan.burn += amount
		plan.steps = append(plan.steps, replayAutoCostStep{burn: amount})
		return true
	}
	applyConversion := func(conversion ConversionType, amount int) bool {
		if amount <= 0 {
			return true
		}
		action := &ConversionAction{
			BaseAction:     BaseAction{Type: ActionConversion, PlayerID: player.ID},
			ConversionType: conversion,
			Amount:         amount,
		}
		if err := action.Execute(cloneState); err != nil {
			return false
		}
		switch conversion {
		case ConversionPowerToPriest:
			plan.powerToPriests += amount
		case ConversionPowerToWorker:
			plan.powerToWorkers += amount
		case ConversionPowerToCoin:
			plan.powerToCoins += amount
		case ConversionPriestToWorker:
			plan.priestsToWorker += amount
		case ConversionWorkerToCoin:
			plan.workersToCoins += amount
		case ConversionAlchVPToCoin:
			plan.vpToCoins += amount
		}
		plan.steps = append(plan.steps, replayAutoCostStep{conversion: conversion, amount: amount})
		return true
	}

	ensureSpendablePower := func(required int) bool {
		needed := required - clone.Power.Bowl3
		if needed <= 0 {
			return true
		}
		burnAmount := needed
		if clonePlayer.Faction != nil && clonePlayer.Faction.GetType() == models.FactionChildrenOfTheWyrm {
			burnAmount = (needed + 1) / 2
		}
		return applyBurn(burnAmount) && clone.Power.Bowl3 >= required
	}
	powerConversionYield := func() int {
		if clonePlayer.Faction != nil && clonePlayer.Faction.GetType() == models.FactionTheEnlightened && clonePlayer.HasStrongholdAbility {
			return 2
		}
		return 1
	}
	convertPowerToPriests := func(shortfall int) bool {
		if shortfall <= 0 || isRiverwalkers(clonePlayer) {
			return shortfall <= 0
		}
		amount := (shortfall + powerConversionYield() - 1) / powerConversionYield()
		if !ensureSpendablePower(amount * 5) {
			return false
		}
		return applyConversion(ConversionPowerToPriest, amount)
	}
	convertPowerToWorkers := func(shortfall int) bool {
		if shortfall <= 0 {
			return true
		}
		amount := (shortfall + powerConversionYield() - 1) / powerConversionYield()
		if !ensureSpendablePower(amount * 3) {
			return false
		}
		return applyConversion(ConversionPowerToWorker, amount)
	}
	convertPowerToCoins := func(shortfall int) bool {
		if shortfall <= 0 {
			return true
		}
		amount := (shortfall + powerConversionYield() - 1) / powerConversionYield()
		if !ensureSpendablePower(amount) {
			return false
		}
		return applyConversion(ConversionPowerToCoin, amount)
	}

	if priestShortfall := cost.Priests - clone.Priests; priestShortfall > 0 {
		if !convertPowerToPriests(priestShortfall) {
			return replayAutoCostPlan{}, false
		}
	}

	if workerShortfall := cost.Workers - clone.Workers; workerShortfall > 0 {
		availablePriests := maxInt(0, clone.Priests-cost.Priests)
		workerYield := 1
		if clonePlayer.Faction != nil && clonePlayer.Faction.GetType() == models.FactionDynionGeifr {
			workerYield = 2
		}
		usePriests := minInt(availablePriests, (workerShortfall+workerYield-1)/workerYield)
		if usePriests > 0 && !applyConversion(ConversionPriestToWorker, usePriests) {
			return replayAutoCostPlan{}, false
		}
		workerShortfall = cost.Workers - clone.Workers
		if workerShortfall > 0 && !convertPowerToWorkers(workerShortfall) {
			return replayAutoCostPlan{}, false
		}
		if clone.Workers < cost.Workers {
			return replayAutoCostPlan{}, false
		}
	}

	if coinShortfall := cost.Coins - clone.Coins; coinShortfall > 0 {
		availableWorkers := maxInt(0, clone.Workers-cost.Workers)
		useWorkers := minInt(coinShortfall, availableWorkers)
		if useWorkers > 0 && !applyConversion(ConversionWorkerToCoin, useWorkers) {
			return replayAutoCostPlan{}, false
		}

		coinShortfall = cost.Coins - clone.Coins
		availablePriests := maxInt(0, clone.Priests-cost.Priests)
		coinsPerPriest := 1
		if clonePlayer.Faction != nil && clonePlayer.Faction.GetType() == models.FactionDynionGeifr {
			coinsPerPriest = 2
		}
		usePriests := minInt(availablePriests, (coinShortfall+coinsPerPriest-1)/coinsPerPriest)
		if usePriests > 0 {
			workersBefore := clone.Workers
			if !applyConversion(ConversionPriestToWorker, usePriests) {
				return replayAutoCostPlan{}, false
			}
			coinShortfall = cost.Coins - clone.Coins
			availableNewWorkers := maxInt(0, clone.Workers-maxInt(cost.Workers, workersBefore))
			useNewWorkers := minInt(coinShortfall, availableNewWorkers)
			if useNewWorkers > 0 && !applyConversion(ConversionWorkerToCoin, useNewWorkers) {
				return replayAutoCostPlan{}, false
			}
		}

		coinShortfall = cost.Coins - clone.Coins
		if coinShortfall > 0 && !convertPowerToCoins(coinShortfall) && (clonePlayer.Faction == nil || clonePlayer.Faction.GetType() != models.FactionAlchemists) {
			return replayAutoCostPlan{}, false
		}

		coinShortfall = cost.Coins - clone.Coins
		if coinShortfall > 0 && clonePlayer.Faction != nil && clonePlayer.Faction.GetType() == models.FactionAlchemists {
			useVP := minInt(coinShortfall, clonePlayer.VictoryPoints)
			if useVP > 0 && !applyConversion(ConversionAlchVPToCoin, useVP) {
				return replayAutoCostPlan{}, false
			}
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
