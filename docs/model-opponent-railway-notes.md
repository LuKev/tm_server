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

For the deployed server to play with a neural checkpoint, set:

```text
TM_AZ_MODEL_URL=https://<inference-service>/evaluate
```

If `TM_AZ_MODEL_URL` is not set or is unavailable at server start, the bot
manager falls back to the heuristic evaluator. That keeps Railway deploys
healthy, but the opponent is not the promoted neural model.

## Local Verification

Use the repository Bazel checks from `server/`:

```bash
bazel build //cmd/server:server
bazel test //internal/websocket:websocket_test --test_output=errors
bazel test //:client_build_test --test_output=errors --nocache_test_results
```

The broad Playwright wrapper is useful as a larger smoke, but it currently
contains unrelated replay/UI failures outside the model-opponent path.
