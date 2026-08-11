package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lukev/tm_server/internal/az"
)

func main() {
	var (
		inference           = flag.String("inference", "", "path to az_inference")
		candidateCheckpoint = flag.String("candidate-checkpoint", "", "candidate snapshot")
		candidateLatest     = flag.String("candidate-latest", "", "candidate latest.json")
		baselineCheckpoint  = flag.String("baseline-checkpoint", "", "baseline snapshot; empty uses random initialization")
		baselineLatest      = flag.String("baseline-latest", "", "baseline latest.json")
		baselineUniform     = flag.Bool("baseline-uniform", false, "use exact uniform-policy/zero-value ancestor")
		modelConfig         = flag.String("model-config", "auto", "auto, debug, main, or large")
		inferenceDevice     = flag.String("inference-device", "cpu", "inference device: cpu, cuda, mps, or auto")
		inferenceThreads    = flag.Int("inference-torch-threads", 1, "PyTorch CPU threads; zero keeps the default")
		candidateSeed       = flag.Int64("candidate-model-seed", 1, "candidate random seed when no checkpoint is supplied")
		baselineSeed        = flag.Int64("baseline-model-seed", 0, "baseline random seed when no checkpoint is supplied")
		output              = flag.String("output", "", "immutable evaluation report directory")
		engineCommit        = flag.String("engine-commit", "", "required engine source revision")
		simulations         = flag.Int("simulations", 1, "MCTS simulations per decision")
		candidateSims       = flag.Int("candidate-simulations", 0, "candidate MCTS simulations; zero uses --simulations")
		baselineSims        = flag.Int("baseline-simulations", 0, "baseline MCTS simulations; zero uses --simulations")
		caseCount           = flag.Int("cases", 0, "number of fixed holdout cases; zero uses all")
		maxPlies            = flag.Int("max-plies", 2000, "natural-completion tripwire")
	)
	flag.Parse()
	if *inference == "" || *output == "" || *engineCommit == "" {
		fatal(fmt.Errorf("--inference, --output, and --engine-commit are required"))
	}
	candidatePath, err := az.ResolveCheckpointReference(*candidateCheckpoint, *candidateLatest)
	if err != nil {
		fatal(err)
	}
	baselinePath, err := az.ResolveCheckpointReference(*baselineCheckpoint, *baselineLatest)
	if err != nil {
		fatal(err)
	}
	if *baselineUniform && baselinePath != "" {
		fatal(fmt.Errorf("--baseline-uniform cannot be combined with a baseline checkpoint"))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	candidate, err := az.StartPythonEvaluatorWithOptions(ctx, *inference, az.PythonEvaluatorOptions{
		Checkpoint: candidatePath, ModelConfig: *modelConfig, Device: *inferenceDevice,
		Seed: *candidateSeed, TorchThreads: *inferenceThreads,
	})
	if err != nil {
		fatal(err)
	}
	defer candidate.Close()
	var baseline az.Evaluator = az.UniformEvaluator{}
	if !*baselineUniform {
		if baselinePath == candidatePath && *baselineSeed == *candidateSeed {
			baseline = candidate
		} else {
			baselineProcess, err := az.StartPythonEvaluatorWithOptions(ctx, *inference, az.PythonEvaluatorOptions{
				Checkpoint: baselinePath, ModelConfig: *modelConfig, Device: *inferenceDevice,
				Seed: *baselineSeed, TorchThreads: *inferenceThreads,
			})
			if err != nil {
				fatal(err)
			}
			defer baselineProcess.Close()
			baseline = baselineProcess
		}
	}

	cases := az.HoldoutCases()
	if *caseCount < 0 || *caseCount > len(cases) {
		fatal(fmt.Errorf("--cases must be between 0 and %d", len(cases)))
	}
	if *caseCount > 0 {
		cases = cases[:*caseCount]
	}
	holdoutSuiteID, err := az.HoldoutSuiteIdentity(cases)
	if err != nil {
		fatal(err)
	}
	candidateBudget, baselineBudget := *candidateSims, *baselineSims
	if candidateBudget == 0 {
		candidateBudget = *simulations
	}
	if baselineBudget == 0 {
		baselineBudget = *simulations
	}
	report, err := az.RunPairedArena(ctx, candidate, baseline, cases, az.ArenaConfig{
		HoldoutSuiteID:       holdoutSuiteID,
		CandidateSimulations: candidateBudget, BaselineSimulations: baselineBudget,
		CPUCT: 1.5, MaxPlies: *maxPlies,
	}, *engineCommit)
	if err != nil {
		fatal(err)
	}
	ref, err := az.WriteArenaReport(*output, report)
	if err != nil {
		fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Report  az.ReportRef    `json:"report"`
		Summary az.ArenaSummary `json:"summary"`
	}{Report: ref, Summary: report.Summary}); err != nil {
		fatal(err)
	}
	if !report.Summary.StrengthValid {
		fatal(fmt.Errorf("arena attempt recorded failures; strength result is invalid"))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
