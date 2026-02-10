# Agent Instructions

- Always use Bazel commands in this repository.
- Do not use `go test` or `go build`; use the Bazel equivalent instead.
- Maintain a workspace notes file at `NOTES.md`.
- Treat `NOTES.md` as shared agent memory for this workspace: whenever a conversation reveals a non-trivial detail that could help in the future (constraints, decisions, gotchas, environment quirks, follow-ups), add to or update `NOTES.md`.

## Skills
A skill is a set of local instructions to follow that is stored in a `SKILL.md` file. Below is the list of skills that can be used. Each entry includes a name, description, and file path so you can open the source for full instructions when using a specific skill.

### Available skills
- reduce-tech-debt: Reduce maintainability debt by finding and consolidating duplicated code and by identifying and removing dead code. Use when requests mention tech debt cleanup, repeated logic, copy-paste code, unused functions/files/exports, or safe refactoring to improve maintainability without changing behavior. (file: /Users/kevin/projects/tm_server/skills/reduce-tech-debt/SKILL.md)
