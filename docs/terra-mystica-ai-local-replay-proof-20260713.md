# Local Replay Training Proof - 2026-07-13

## Goal

Test whether a larger self-play-only replay buffer, a stronger-search tranche, and a smaller training update can produce a clear local improvement before scaling compute on Modal.

## Training Data

| Source | Games | Simulations | Positions | Average final VP |
| --- | ---: | ---: | ---: | ---: |
| Historical neural self-play | 1,512 | 8 | 98,081 | mixed |
| Fresh broad tranche | 750 | 8 | 48,800 | 82.53 |
| Fresh quality tranche | 250 | 32 | 15,202 | 84.00 |
| **Combined replay** | **2,512** | mixed | **162,083** | - |

Fresh positions were 39.5% of the replay buffer. All positions and targets came from self-play.

The first candidate used two epochs at learning rate `2e-5` and failed the parent gate, 159-173-4 with `-1.25 VP`. A more conservative candidate trained for one epoch at `1e-5`.

## Strength Gates

| Gate | Opponent | Games | Result | Score | 95% CI | Candidate VP | Baseline VP | VP change |
| --- | --- | ---: | --- | ---: | --- | ---: | ---: | ---: |
| Parent | Iteration 5 | 336 paired | 216-116-4 | 64.88% | 59.78%-69.99% | 90.08 | 81.88 | **+8.20** |
| Anchor | Iteration 1 | 672 paired | 419-242-11 | 63.17% | 59.52%-66.82% | 89.45 | 81.81 | **+7.65** |

The candidate passed both gates. The anchor CI lower bound exceeded 50% by 9.52 percentage points.

## Round 1 Building Frequencies

Values are average R1 construction transitions per player. A building followed by another upgrade in the same round is counted at each stage.

| Population | Samples | Dwelling | Trading House | Temple | Sanctuary | Stronghold | Temple presence | SH presence |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Fresh self-play, 8 sims | 1,500 | 1.819 | 1.329 | 0.125 | 0.000 | 0.002 | 11.93% | 0.20% |
| Fresh self-play, 32 sims | 500 | 1.400 | 0.866 | 0.054 | 0.004 | 0.020 | 5.20% | 2.00% |
| Parent gate candidate | 336 | 1.899 | 1.342 | 0.110 | 0.003 | 0.003 | 10.12% | 0.30% |
| Parent gate baseline | 336 | 1.842 | 1.393 | 0.140 | 0.003 | 0.000 | 13.10% | 0.00% |
| Anchor gate candidate | 672 | 1.835 | 1.424 | 0.147 | 0.003 | 0.004 | 13.99% | 0.45% |
| Anchor gate baseline | 672 | 1.911 | 1.342 | 0.147 | 0.001 | 0.001 | 14.29% | 0.15% |

## Conclusion

The local experiment proves that replay-buffer training can produce a clear strength gain when the update is sufficiently conservative. The critical change was not merely more data: two epochs at `2e-5` regressed, while one epoch at `1e-5` passed both large paired gates decisively.

R1 Temple and Stronghold behavior remains weak. Increased playing strength did not solve the opening-policy proxy, so opening improvement still needs targeted self-play sampling or a better policy representation rather than additional blind fine-tuning.
