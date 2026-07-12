# Model Opponent Railway Notes

The live model-opponent flow uses the normal lobby and game board. Creating a
model game reserves a `TM-AZ-<gameID>` seat, applies the selected human/model
factions during snellman setup, and then lets `websocket.BotManager` execute
model turns through the same game manager path as human actions.

## Railway Deploy

The checked-in Railway config is unchanged:

- `server/nixpacks.toml` builds the backend from `server/` and starts `./out`.
- `client/nixpacks.toml` runs `npm run build`, copies `Caddyfile` into `dist/`,
  and starts Caddy.
- The production client keeps `VITE_BASE_PATH=/tm` for React Router, while Vite
  assets stay at `/assets` unless `VITE_ASSET_BASE_PATH` is explicitly set.
  Using `VITE_BASE_PATH` as the Vite asset base makes the production page blank
  because the static server rewrites `/tm/assets/*` to the SPA document.

For the deployed server to play with a neural checkpoint, set:

```text
TM_AZ_MODEL_URL=https://<inference-service>/evaluate
TM_AZ_REQUIRE_NEURAL=true
```

With `TM_AZ_REQUIRE_NEURAL=true`, backend startup fails unless the inference
service's `/healthz` endpoint returns valid checkpoint metadata. Verify
`GET /api/ai/status` returns `mode=neural` after every deploy. Without the
require flag, missing or failed inference can still use the heuristic evaluator;
that mode is for development only.

The end-to-end training, durable promotion, inference deployment, and website
verification process is in `terra-mystica-neural-production-runbook.md`.

## Local Verification

Use the repository Bazel checks from `server/`:

```bash
bazel build //cmd/server:server
bazel test //internal/websocket:websocket_test --test_output=errors
bazel test //:client_build_test --test_output=errors --nocache_test_results
```

The broad Playwright wrapper is useful as a larger smoke, but it currently
contains unrelated replay/UI failures outside the model-opponent path.
