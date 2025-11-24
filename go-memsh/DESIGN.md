# memsh Environment & Pipeline Reliability Plan

## Goals
Improve `go-memsh` so shell sessions preserve and isolate environment as expected and so pipelines within scripts stream data correctly. The plan targets three known gaps: (1) environment persistence across `Run` calls, (2) configurable environment isolation for nested `sh` invocations, and (3) pipeline-aware builtins inside scripts.

## Current Pain Points
- `Shell.Run` recreates runner state on each call and `cmdSh` uses cloned environments, so variable mutations are not retained between runs and isolation semantics are unclear.
- Nested scripts cannot choose between inheriting and isolating environment changes; temporary copies are either shared unsafely or discarded.
- Builtins always read/write via `Shell.stdin/stdout`, ignoring pipeline-provided streams, so `echo | grep` within scripts breaks and process substitution competes for the same descriptors.

## Design Overview
- Introduce a reusable session state layer that owns the runner, environment map, and working directory, enabling controlled reuse across `Run` calls without losing mutations.
- Provide explicit isolation modes for script execution, letting callers select whether a child script can mutate the parent environment.
- Make builtin commands pipeline-aware by honoring the contextual stdio provided by `interp.HandlerCtx` while keeping process substitution plumbing intact.

## Action Plan

### 1) Persist shell session state across `Run`
- Add `SessionState` (runner, env map, cwd, prevDir, pipe manager) and refactor `Shell` to delegate to it instead of recreating runner configuration on each `Run`.
- Ensure `SetIO` only rebinds stdio on the existing runner, preserving environment and directory between runs.
- Update tests to cover variable persistence across multiple `Run` calls and ensure regression coverage for `export`/`unset`.

### 2) Configurable script environment isolation
- Extend `cmdSh` (and any other script entrypoints) with a `ShellConfig.ScriptIsolation` flag to select clone-vs-merge behavior for environments.
- For isolation mode: run the script against a cloned `EnvironMap` and discard mutations after completion; for inheritance mode: track mutations (exports/unsets/dir changes) in the child runner and merge them back into the parent `SessionState`.
- Provide helper APIs to snapshot `SessionState` for callers that need clean shells (e.g., HTTP sessions or tests spawning new shells).
- Add tests demonstrating isolation (parent env unchanged) and inheritance (mutations visible) with nested `sh -c` scripts.

### 3) Pipeline-aware builtins inside scripts
- Thread the active pipeline stdio from `interp.HandlerCtx` into builtin implementations via a helper (e.g., `ctxReader/Writer`) instead of defaulting to `Shell.stdin/stdout`.
- Audit builtins (`cat`, `grep`, `head`, `tail`, `wc`, etc.) to consume from the contextual reader and write to the contextual writer while maintaining process substitution support in `openFile`.
- Add integration tests for pipelines inside scripts (e.g., `echo a b | grep b`, `cat file | head -n1`) and ensure ordering and buffering are correct.

### 4) Migration & compatibility considerations
- Keep the public `Shell` API stable; new configuration should be optional with sensible defaults (inheritance enabled to preserve current behavior where tests expect it).
- Document the new isolation flag and session snapshot helper in `README.md` or `API.md` once implemented.
- Verify process substitution and virtual pipe cleanup still work when pipeline-aware stdio is used.

### 5) Rollout sequence
1. Introduce `SessionState` and refactor `Shell.Run/SetIO` to use it; add persistence tests.
2. Implement script isolation flag and merge logic; add isolation/inheritance tests.
3. Make builtins pipeline-aware and add pipeline regression tests.
4. Run full test suite and update docs.
