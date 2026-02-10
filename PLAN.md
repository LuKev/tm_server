# Replay Import + Concise Input Implementation Plan

## Goals

1. Add full Snellman bookmarklet import to replay.
2. Add user-facing manual log input that supports concise notation (and auto/snellman text detection).
3. Make concise notation a first-class replay ingestion format.
4. Remove dead `gameID == "local"` replay code path and associated branching.

## Constraints

- Keep existing BGA replay ingestion behavior unchanged.
- Preserve compatibility with existing concise generation and replay UI controls.
- Use Bazel for all test execution.
- Avoid destructive history operations.

## Design Summary

- Introduce a content-driven parsing pipeline in replay manager:
  - concise text -> `notation.ParseConciseLogStrict`
  - snellman text -> `notation.ConvertSnellmanToConcise` -> `notation.ParseConciseLogStrict`
  - fallback bga text -> `notation.NewBGAParser(...).Parse()`
- Keep import HTML parsing where it currently lives, but ensure Snellman HTML goes through Snellman text -> concise -> replay (via shared parse pipeline).
- Add new API endpoint for text imports.
- Add Import UI for pasted logs with source format selection.
- Remove all `local` replay test-mode branching from runtime path.

## Execution Phases

## Phase 1: Backend replay parsing pipeline refactor

- [x] Add shared helper in replay manager to parse raw log text by detected/declared format.
- [x] Remove `gameID == "local"` branches from `StartReplay` and `fetchLog`.
- [x] Update `ImportLog` to feed parsed text into the shared pipeline behavior.
- [x] Preserve existing session creation behavior and `GenerateConciseLog` outputs.

### Deliverables

- `server/internal/replay/manager.go` refactored for content-driven parsing.

## Phase 2: Concise parser hardening for Snellman-generated tokens

- [x] Add strict concise parser mode returning line/column-aware errors.
- [x] Ensure parser supports key concise tokens used by Snellman conversion:
  - `ACT-BON-*`
  - `ACT-TOWN-*`
  - `ACT-BR-*`
  - cult shorthand `+F`, `+W`, `+E`, `+A`
- [x] Ensure parsing failures do not silently drop actions in strict mode.

### Deliverables

- `server/internal/notation/parser.go` strict parser + added token coverage.

## Phase 3: Text import API

- [x] Add manager method for raw text imports with source option (`auto|concise|snellman|bga`).
- [x] Add API route: `POST /api/replay/import_text`.
- [x] Return actionable parse errors to client.

### Deliverables

- `server/internal/replay/manager.go` text import entry point.
- `server/internal/api/replay.go` new route + handler.

## Phase 4: Import page UI for pasted logs

- [x] Add log textarea and source selector to import screen.
- [x] Add import button that calls `/api/replay/import_text`.
- [x] Reuse existing status messaging UX and navigate to replay on success.
- [x] Keep bookmarklet import UX intact.

### Deliverables

- `client/src/components/ImportGame.tsx` updated UI + request handling.
- `client/src/components/ImportGame.css` styling updates if needed.

## Phase 5: Dead code cleanup for local replay path

- [x] Remove dead local-only comments/branches and references tied to `gameID == "local"` replay mode.
- [x] Keep any unrelated test fixtures intact.

### Deliverables

- Local path removed from runtime replay manager flow.

## Phase 6: Tests + verification + docs

- [x] Add/adjust notation tests for strict parse + new tokens.
- [x] Add/adjust replay tests for import flow and format routing.
- [x] Run Bazel tests for touched packages.
- [x] Update docs where needed for manual concise input.
- [x] Update `NOTES.md` with final implementation details and gotchas.

### Validation Commands

- `bazel test //server/internal/notation:notation_test`
- `bazel test //server/internal/replay:replay_test`
- `bazel test //server/internal/api:api_test` (if tests exist)

## Rollout / Risk Notes

- Highest risk: concise parser strictness causing previously tolerated malformed logs to fail fast.
- Mitigation: strict mode used for import entry points; retain non-strict helper behavior where appropriate.
- Secondary risk: token semantics for shorthand cult actions (`+E`) in compound actions.
- Mitigation: explicit unit tests around representative Snellman-converted rows.

## Done Criteria

- Snellman bookmarklet import successfully produces replay sessions.
- Manual concise log paste path works from UI through replay state load.
- `gameID == "local"` runtime path fully removed.
- Bazel tests pass for touched packages.
- `NOTES.md` updated with final architectural notes.
