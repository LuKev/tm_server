# Neural Model Production Runbook

## Required Outcome

The website is running a neural opponent only when all of the following are true:

1. A retained PyTorch checkpoint has a recorded parent, training buffer, git SHA,
   training command, and arena result.
2. The checkpoint is stored outside `/tmp` in durable local storage and in the
   user's Google One-backed Drive folder `Terra Mystica AI/models/<model-id>/`.
   A hash-identical copy may be mirrored to a Modal volume for serving.
3. `az_infer_torch` serves that exact checkpoint and `/healthz` reports its
   architecture, observation schema, input size, and action count.
4. The website backend has `TM_AZ_MODEL_URL` set to that service's `/evaluate`
   URL and `TM_AZ_REQUIRE_NEURAL=true`.
5. `GET /api/ai/status` returns HTTP 200 with `mode=neural` and checkpoint
   metadata. A real game on `/tm/ai` completes at least one model turn.

The current state does not meet this bar. The former promoted h512 checkpoint
was stored under `/tmp` and is gone. The surviving July checkpoints came from a
fresh-baseline recovery experiment and are not valid continuations of the old
promoted lineage.

The local serving path has been verified with the surviving July h512 candidate:
the Torch service loaded a `hex` checkpoint with `5,867` actions and input size
`3,458`; a backend running with required-neural mode reported `mode=neural` and
completed a neural `/api/ai/suggest` request. This proves the integration, not
the strength or promotion status of that checkpoint.

## Phase 1: Make A Durable Seed Model

Because the prior incumbent is gone, rebuild from self-play only. Do not label
the surviving fresh checkpoint as the old model.

1. Create a durable run directory such as
   `artifacts/az/runs/<run-id>/` (gitignored), a Google Drive model package, and
   a serving mirror in the Modal volume `tm-az-models`.
2. Run a small neural bootstrap using the surviving fresh checkpoint only if its
   manifest, action vocabulary, and observation schema validate. Otherwise train
   a new h512 bootstrap from fresh self-play.
3. Generate `336-840` ordered-matrix self-play games at `sims=8`. This is the
   iteration smoke/data pass, not a promotion arena.
4. Export with `az_export`, then train h512 with `az_train_torch`. Record the
   exact dataset and parent checkpoint hashes.
5. Reject obvious regressions with an `84`-game smoke arena. Any candidate that
   survives gets a separate `168`-game ordered promotion arena against its neural
   parent. Both candidate and baseline URLs are mandatory.
6. Copy the accepted checkpoint, action vocabulary, metrics, arena report, R1
   build-rate report, and manifest into `artifacts/az/models/<model-id>/`, then
   upload the whole package to Google Drive and verify its file sizes and hashes
   before changing `current`. Mirror the pinned checkpoint to Modal for serving.

The first recovered model will be weaker than the lost June incumbent. Its job
is to re-establish a trustworthy, reproducible neural lineage. Subsequent
iterations improve it through self-play.

## Phase 2: Strength Iterations

Each iteration uses only the current neural incumbent for self-play targets:

```text
incumbent checkpoint
  -> neural MCTS self-play shards
  -> mixed replay buffer
  -> az_export
  -> az_train_torch --init_checkpoint=<incumbent>
  -> 84-game rejection smoke
  -> 168-game neural-vs-neural promotion arena
  -> durable promotion
```

Use `336-840` games locally for quick iterations. On Modal, start with `10k`
games across CPU shards at `sims=8`, plus a smaller `sims=16` quality batch.
Training h512 is fine on CPU for the small lane; use one modest GPU for the
10k-game training grid because it shortens training cheaply. Self-play and arena
workers are primarily CPU workloads, with a local inference process per shard.

Promotion requires at least `168` games, raw win rate above `50%`, and a
predeclared 95% confidence lower-bound rule. The `84`-game arena can reject a
candidate but can never promote one. Record R1 Temple/Sanctuary/Stronghold rates
by faction in `terra-mystica-ai-r1-build-rates.md` for every retained candidate
and incumbent.

## Phase 3: Serve The Model

Deploy `az_infer_torch` as a separate always-on service. The service needs only
CPU for h512 website inference; GPU is unnecessary at current model size. Keep
the checkpoint on a mounted durable volume and pin the model ID rather than a
mutable filename.

Inference command shape:

```bash
cd server
bazel run //cmd/az_infer_torch:az_infer_torch -- \
  --checkpoint=/models/<model-id>/model.pt \
  --host=0.0.0.0 \
  --port="$PORT" \
  --torch-threads=2 \
  --torch-interop-threads=1
```

Configure the website backend:

```text
TM_AZ_MODEL_URL=https://<inference-host>/evaluate
TM_AZ_REQUIRE_NEURAL=true
```

Deployment verification:

1. `GET https://<inference-host>/healthz` returns checkpoint metadata.
2. `GET https://<backend-host>/api/ai/status` returns `mode=neural`.
3. Start `/tm/ai`, play through the first human action, and confirm the model
   responds without `neural_unavailable` or heuristic fallback logs.
4. Restart both services and repeat the status check to prove the checkpoint is
   durable.

## Operational Guardrails

- `/tmp` is scratch space only. While checkpoints remain small, Google Drive
  under the user's Google One plan is the durable source of record. A promotion
  is incomplete until Drive upload and read-back validation succeed.
- Never run a promotion arena without both `-candidate_url` and `-baseline_url`.
- Never infer lineage from filenames. Store SHA-256 hashes in the run manifest.
- Keep the prior promoted model so rollback is one configuration change.
- Alert on inference health failures and latency. Production should use
  `TM_AZ_REQUIRE_NEURAL=true`; a heuristic fallback is a development mode, not a
  production availability strategy.
