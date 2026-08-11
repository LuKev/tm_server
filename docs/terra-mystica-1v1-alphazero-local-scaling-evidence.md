# Terra Mystica AlphaZero local scaling evidence

Date: 2026-08-11

This report is the durable record for the first provenance-clean Milestone 5
pilot and local throughput measurements. It establishes a working large-model
cycle and local batch/concurrency settings. It does **not** establish playing
strength; every arena in the pilot used only two paired holdout cases.

## Provenance and fixed configuration

| Field | Value |
| --- | --- |
| Engine commit | `9b93ec31cfe8501287ad7059ff02791d06e70981` |
| Hardware | Apple M5, 10 CPU cores, 16 GB unified memory |
| OS | macOS 26.1 (25B78) |
| Model | `large`: 20 residual blocks, 256 channels, 29,114,114 parameters, FP32 |
| Device | MPS, one PyTorch CPU thread |
| Optimizer | momentum SGD, LR `1e-4`, momentum `0.9`, L2 `1e-4` |
| Final checkpoint SHA-256 | `f61ce12a5638d377a54a6002f680c29ce0a42c97c9936c36ba343cf2dbdf9bff` |
| Final model ID | `model-30e2d7e15b701b1a5d08dc38e473ca01758bdba2f427427f39c37bcb2b51a714` |

The run manifest, all 16 trajectory shards, both checkpoints, and all three
arena reports identify the same real Git revision. All seven runner stages
exited zero and had empty stderr. The raw local run was created at
`/tmp/tm-az-exp-9b93ec3/01-provenance-clean-sgd-pilot`; that scratch path is
not a retention guarantee, so the decision metrics and content identities are
recorded here in Git.

## Two-cycle pilot

The runner used eight games/cycle, eight MCTS simulations, eight concurrent
games, 0.25 requested examples per available replay ply, a 10,000-shard replay
horizon, an eight-game decompression cache, and two paired arena cases. Cycle 2
loaded cycle 1's checkpoint and optimizer state.

| Metric | Cycle 1 | Cycle 2 |
| --- | ---: | ---: |
| Complete games | 8 | 8 |
| Decisions/game | 39.750 | 44.625 |
| Games/hour | 427.159 | 386.980 |
| Mean inference batch | 6.191 | 6.264 |
| Inference p95 | 60.707 ms | 62.068 ms |
| Replay plies available | 318 | 675 |
| SGD steps | 3 | 6 |
| Actual examples/replay ply | 0.302 | 0.284 |
| Learner examples/second | 54.884 | 63.981 |
| Diagnostic total loss, before to after | 3.534 to 3.333 | 3.373 to 3.578 |
| Diagnostic value loss, before to after | 1.146 to 0.941 | 0.996 to 1.198 |

The second update remained finite but worsened the replay diagnostic. This is
why learner intensity will be chosen using reserved arena calibration rather
than sampled training loss.

The three immutable arena reports were:

| Comparison | Report SHA-256 | Games | Score | VP margin |
| --- | --- | ---: | ---: | ---: |
| Cycle 1 vs uniform | `ca20b0fd18aaa0d94b409311d06897e9510f098d9134cd5bbc341eb5a43f41e3` | 4 | 0.500 | +0.75 |
| Cycle 2 vs uniform | `59ddc1d59b8ec71d656e5d58a26d2fccb4e1db4da9630fa5f59358241a773638` | 4 | 0.500 | +8.00 |
| Cycle 2 vs cycle 1 | `c32df073e247c2598e38c876c4f2cb4e6765c81e7bbf7e308f8fb702d696f952` | 4 | 0.500 | +8.00 |

All 12 games completed naturally with zero invalid actions, duplicate states,
tripwire hits, or infrastructure failures. The confidence intervals span the
full useful range; these reports prove mechanics and lineage only.

## Checkpoint-bound batch curve

Every point loaded the final checkpoint above with identical model, MPS,
FP32, seed, legal-action width, and thread count.

| Path | Batch | Throughput | Median latency |
| --- | ---: | ---: | ---: |
| Inference | 16 | 272.568 positions/s | 58.701 ms |
| Inference | 32 | 286.942 positions/s | 111.521 ms |
| Inference | 64 | 289.827 positions/s | 220.821 ms |
| SGD training | 16 | 95.128 examples/s | 168.194 ms |
| SGD training | 32 | 101.460 examples/s | 315.395 ms |
| SGD training | 64 | 102.397 examples/s | 625.018 ms |
| SGD training | 128 | 102.650 examples/s | 1,246.959 ms |

Batch 32 is the smallest measured near-ceiling point: it reaches 99.0% of the
batch-64 inference rate and 98.8% of the batch-128 training rate. Larger
batches add at most 1.2% throughput while multiplying latency.

## Counterbalanced actor concurrency

Each point replayed the same eight setup seeds at eight simulations. The
forward order was 1/2/4/8 concurrent games and the reverse order was 8/4/2/1.
All eight runs produced 68.625 decisions/game and the same semantic trajectory
digest:
`92a57e99de4dd96dc4d006d38a27b9e5ea46ba958883b25bf935b25733366cf9`.

| Concurrent games | Forward games/hour | Reverse games/hour | Mean | Order spread | Mean neural batch | Typical p95 |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 208.788 | 208.043 | 208.416 | 0.36% | 1.000 | 13.9 ms |
| 2 | 229.364 | 229.767 | 229.566 | 0.18% | 1.731 | 20.2 ms |
| 4 | 229.695 | 232.115 | 230.905 | 1.05% | 2.903 | 38.6 ms |
| 8 | 248.032 | 246.798 | 247.415 | 0.50% | 4.660 | 59.3 ms |

Eight concurrent games is the local throughput default: it is 7.2% faster
than four and 18.7% faster than serial in the counterbalanced mean. Two is the
low-latency setting. The mean neural batch of only 4.66 remains far below the
model's batch-32 kernel knee, but this alone does not justify distribution;
search and learner-quality experiments come first.

## Same-checkpoint search-compute curve

This curve gives the candidate and anchor the exact same checkpoint and changes
only MCTS simulations per decision. “32 vs 1,” for example, means 32 tree
simulations for the candidate's decisions and one for the anchor's decisions;
it does not mean 32 networks. Every point used all 28 fixed cases, swapped the
search roles across both seats, and therefore contains 56 complete games.

| Candidate simulations | W-D-L | Score (paired 95% interval) | Candidate VP | Anchor VP | VP margin | Wall time |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 28-0-28 | 0.500 (0.243–0.757) | 64.732 | 64.732 | 0.000 | 347 s |
| 8 | 34-2-20 | 0.625 (0.368–0.882) | 67.804 | 63.375 | +4.429 | 753 s |
| 32 | 40-2-14 | 0.732 (0.475–0.989) | 70.875 | 62.375 | +8.500 | 2,108 s |
| 128 | 39-0-17 | 0.696 (0.440–0.953) | 70.464 | 63.089 | +7.375 | 6,068 s |

The equal-compute control is exactly symmetric in score and VP. Eight and 32
simulations improve both observed score and absolute VP, and the gains are
balanced across candidate seats. The intervals remain wide, so this is not a
statistical claim that any higher budget is certainly stronger than one.
Nevertheless, 128 is dominated in the observed curve: it took 3,960 additional
seconds beyond 32 while producing a lower score and 1.125 less VP margin.

Budget 8 cost 2.17 times the control's wall time for +4.429 average VP margin;
budget 32 cost 2.80 times budget 8 for another +4.071 margin. Budget 128 then
cost 2.88 times budget 32 and lost 1.125 margin. Use 32 as the measured local
search-strength peak and reject 128 for this checkpoint. Eight is the efficient
self-play candidate, but the final self-play setting still requires the planned
downstream experiment that trains new checkpoints from equal data volume at
different generation budgets.

All four report identities bind the checkpoint, engine revision, official
`tm-v0-paired-v2` suite, baseline budget, and eight-game batching. All 224 games
completed naturally with zero invalid actions, duplicate states, tripwire hits,
or infrastructure failures. The immutable report SHA-256 values for budgets
1/8/32/128 are respectively
`4bf80b29d3ca817db94c50179e8a501b97ad581c3a343a053ff6e7b4ff42ae74`,
`c33d938cc0dd903ff13111a706012dbac8b814a0bc8cf7ef0cb61b21277e21f6`,
`7cff102846dff3592eb45339d341caabb761ca11bcc5187282ec733a38a5ead1`,
and `3ead27c88bf28970cca5dd9d3ed68491e7395dbe1acf98a5b36b195287e09d39`.

## Learner-intensity and replay-horizon screen

The learner screen restarted every arm from the same cycle-1 checkpoint and
used the same 16-game, 675-ply replay, SGD configuration, sampler seed, and
eight-case paired evaluation at eight simulations for both agents. The first
four arms changed only requested examples per replay ply. The parent checkpoint
SHA-256 was
`64c2439be1bf0d78151371b54230672638a8bc9cf9e4b1e3b8cc29679a5e1a12`
and its model ID was
`model-5907b2f920cbcb9f315a45a33fe2cf417adc2cab92d2be54295a178f64d17ccc`.
Each Brier value below is the position-weighted mean across setup and rounds
1–6, rather than an unweighted mean of phase summaries.

| Requested ratio | Actual ratio | Steps | Diagnostic value loss, before to after | W-D-L | VP margin | Candidate Brier | Parent Brier |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0.10 | 0.142 | 3 | 0.996 to 1.351 | 8-1-7 | +3.063 | 0.3470 | 0.2387 |
| 0.25 | 0.284 | 6 | 0.996 to 1.198 | 7-1-8 | +2.313 | 0.3049 | 0.2364 |
| 0.50 | 0.521 | 11 | 0.996 to 1.105 | 7-0-9 | -3.750 | 0.2990 | 0.2588 |
| 1.00 | 1.043 | 22 | 0.996 to 1.361 | 7-0-9 | -6.563 | 0.3263 | 0.2582 |

Every intensity worsened held-out value calibration, so none passed the
predeclared guardrail. The W-D-L and VP estimates use only 16 games per arm and
are too uncertain to overrule that consistent calibration result.

The immutable evaluation-report SHA-256 values for requested ratios
0.10/0.25/0.50/1.00 are respectively
`e41a0e10644eb14c833b548a54099fe2b8e5748589634d162e7680fc48714529`,
`a83e48b8266d3ad2dc83d9c66af105b3572933d9d38455749f01f824555bcfee`,
`fa1c9e7cc06cf3d503e7a0c7e8c6262f468822c992770f62603fedf4120bfb7d`,
and `43fdf093b7ae517d98c153386e436c501cf7dafa57c6ff79da7266d622f18ffc`.
The scratch run was created at
`/tmp/tm-az-exp-9b93ec3/05-learner-intensity-sweep`.

One bounded follow-up tested whether replaying the cycle-1 data already seen by
the parent caused the failure. At requested ratio 0.10, the newest eight games
produced two steps and the newest four games produced one. They also failed:

| Replay window | Actual ratio | Steps | W-D-L | VP margin | Candidate Brier | Parent Brier |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 8 games | 0.179 | 2 | 8-1-7 | +3.125 | 0.3118 | 0.2388 |
| 4 games | 0.179 | 1 | 7-1-8 | +3.688 | 0.2691 | 0.2377 |

The immutable report SHA-256 values for windows eight and four are
`7a04d6da9710ad0aacc355d3317787b3d7eead60edef5700eb91fa07ff84a340`
and `3a51a97a2cd6ebd831bd9ad39e2efa31c37402d8baead74e4706033b1d59995c`.
The scratch run was created at
`/tmp/tm-az-exp-9b93ec3/06-replay-window-sweep-ratio-0.10`.

All 96 games across both screens completed naturally with zero invalid
actions, duplicate states, tripwire hits, or infrastructure failures. The
screen therefore selects no learner setting. The next learner experiment must
first increase replay diversity to the complete balanced 168-game ordered
matchup set; choosing the least-bad tiny-replay arm would violate the gate.

## Reproduction shape

The pilot was launched through `//:run_az_cycles` with the exact engine commit
derived by `git rev-parse HEAD`. Batch points used
`//internal/az:az_model_benchmark` with `--checkpoint`, `--device mps`, and
explicit `--mode`/`--optimizer sgd`. Actor points used
`//cmd/az_selfplay:az_selfplay` with fixed `--seed 2000000`, `--games 8`,
`--simulations 8`, and only `--games-per-batch` changed. All build and test
operations used Bazel.

Next: generate the complete balanced 168-game replay set at the efficient
eight-simulation self-play budget, then repeat the learner screen on that fixed
larger dataset before confirming a survivor on the full holdout.
