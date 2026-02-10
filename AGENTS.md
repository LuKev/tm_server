# Agent Instructions

- Always use Bazel commands in this repository.
- Do not use `go test` or `go build`; use the Bazel equivalent instead.
- Maintain a workspace notes file at `NOTES.md`.
- Treat `NOTES.md` as shared agent memory for this workspace: whenever a conversation reveals a non-trivial detail that could help in the future (constraints, decisions, gotchas, environment quirks, follow-ups), add to or update `NOTES.md`.
