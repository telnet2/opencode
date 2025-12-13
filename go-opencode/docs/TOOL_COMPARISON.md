# Tooling Parity: TypeScript vs Go OpenCode

This document compares the built-in tools provided by the TypeScript OpenCode server (`packages/opencode`) with the Go OpenCode server (`go-opencode`). It highlights where Go faithfully clones TypeScript behavior and where gaps remain.

## Registry Coverage

| Aspect | TypeScript | Go | Notes |
| --- | --- | --- | --- |
| Built-in tools | `bash`, `read`, `glob`, `grep`, `list`, `edit`, `write`, `task`, `webfetch`, `todoread`, `todowrite`, `websearch`, `codesearch`, `workflow`, `invalid`, optional `batch`; plugin discovery of custom tools. | `bash`, `read`, `glob`, `grep`, `list`, `edit`, `write`, `webfetch`, `todoread`, `todowrite` (task registered separately). | Go omits `websearch`, `codesearch`, `workflow`, `invalid`, and the optional `batch` tool, and it lacks runtime plugin discovery. TypeScript also guards tool availability by provider/flags (e.g., enabling search for `opencode` users).【F:packages/opencode/src/tool/registry.ts†L84-L130】【F:go-opencode/internal/tool/registry.go†L104-L135】
| Registration hooks | Loads custom tools from configured directories and plugins, merging with built-ins. | Only registers compiled-in tools. | Plugin/tool discovery and per-provider filtering are missing in Go, reducing extensibility.【F:packages/opencode/src/tool/registry.ts†L31-L107】

## Read Tool

| Aspect | TypeScript | Go | Notes |
| --- | --- | --- | --- |
| Path handling & permissions | Resolves relative paths against the workspace and prompts/denies when accessing outside it. Blocks `.env` except whitelisted suffixes. | Expects absolute paths, only blocks exact `.env` filenames; no workspace guard or permission prompts. | External-directory protections and `.env` whitelist parity are missing in Go.【F:packages/opencode/src/tool/read.ts†L26-L74】【F:go-opencode/internal/tool/read.go†L16-L102】
| Missing file UX | Suggests close matches when a file is not found. | Returns a plain “file not found” error. | TypeScript provides suggestions to recover from typos; Go does not.【F:packages/opencode/src/tool/read.ts†L75-L94】
| Binary/media handling | Rejects binaries; streams images and PDFs as attachments with previews. | Rejects binaries; streams common images only. | PDF support and richer preview metadata are absent in Go.【F:packages/opencode/src/tool/read.ts†L96-L158】【F:go-opencode/internal/tool/read.go†L94-L169】
| Output formatting | Returns line-numbered block (`00001| text`), notes remaining lines, and warms LSP/file time tracking. | Returns tab-delimited lines, appends a generic “read more” note when truncated; no LSP or file-time integration. | Line numbering and metadata differ, and Go omits the “read past line N” guidance and editor integrations.【F:packages/opencode/src/tool/read.ts†L123-L151】【F:go-opencode/internal/tool/read.go†L117-L146】

## Edit Tool

| Aspect | TypeScript | Go | Notes |
| --- | --- | --- | --- |
| Workspace & permissions | Resolves relative paths, enforces workspace bounds with ask/deny, and asks for edit permission (including empty `oldString` writes). | No workspace containment or permission prompts. | Safety prompts and path normalization are missing in Go.【F:packages/opencode/src/tool/edit.ts†L44-L134】【F:go-opencode/internal/tool/edit.go†L72-L136】
| Failure handling | Validates file existence/type and requires `oldString != newString`; aborts when file missing. | Similar difference check but errors only on read/write failures; uses fuzzy replace if no exact match. | Go’s fuzzy fallback can mutate unintended regions without explicit permission or diffs.【F:go-opencode/internal/tool/edit.go†L82-L199】
| Result metadata | Returns unified diff text plus structured addition/deletion counts and LSP diagnostics. | Returns replacement count, before/after snapshots, and unified diff text with addition/deletion totals; still no diagnostics. | Go now surfaces change details but continues to omit diagnostics feedback.【F:packages/opencode/src/tool/edit.ts†L109-L172】【F:go-opencode/internal/tool/edit.go†L129-L229】
| Eventing | Publishes file-edited events via bus and updates file-time tracking. | Publishes a basic file-edited event when a session is present. | Partial parity; Go lacks file-time tracking and LSP warm-up.【F:packages/opencode/src/tool/edit.ts†L94-L151】【F:go-opencode/internal/tool/edit.go†L112-L136】

## Write Tool

| Aspect | TypeScript | Go | Notes |
| --- | --- | --- | --- |
| Workspace/permissions | Resolves relative paths, enforces workspace bounds, and asks before overwriting/creating based on agent permissions. | Writes directly after ensuring parent dirs; no permission gates or workspace enforcement. | Go bypasses safety prompts and external-directory checks.【F:packages/opencode/src/tool/write.ts†L20-L70】【F:go-opencode/internal/tool/write.go†L59-L115】
| Concurrency safety | Uses file-time assertions before overwriting existing files. | No stale-write protection. | Potential race/stale edits in Go compared to TS safeguards.【F:packages/opencode/src/tool/write.ts†L53-L56】
| Post-write metadata | Returns LSP diagnostics and file metadata; triggers bus event and file-time tracking. | Returns byte count plus before/after snapshots, diff text, and addition/deletion totals; publishes file-edited event. | Go adds diff-oriented metadata but still lacks diagnostics feedback and file-time updates.【F:packages/opencode/src/tool/write.ts†L71-L97】【F:go-opencode/internal/tool/write.go†L71-L105】

## List/LS Tool

| Aspect | TypeScript (`list`) | Go (`list`) | Notes |
| --- | --- | --- | --- |
| Scope & filtering | Uses ripgrep to build a tree with ignore globs (e.g., `node_modules`, `.git`), limits to 100 entries, and reports truncation. | Simple `os.ReadDir` with no default ignores or limits. | Go can surface noisy vendor/build dirs and lacks truncation metadata.【F:packages/opencode/src/tool/ls.ts†L8-L74】【F:go-opencode/internal/tool/list.go†L15-L88】
| Output | Hierarchical tree rooted at the workspace path. | Flat listing with type/size per entry. | Output shapes differ; TS tree is better for structure, Go for quick directory contents.【F:packages/opencode/src/tool/ls.ts†L75-L104】【F:go-opencode/internal/tool/list.go†L89-L116】

## Bash Tool

| Aspect | TypeScript | Go | Notes |
| --- | --- | --- | --- |
| Command parsing & safety | Parses commands with tree-sitter, enforces external-directory permission checks, and honors per-command allow/deny/ask patterns. Requires descriptions and supports explicit working directory. | Optional permission checker; defaults to “ask” for external dirs but lacks AST-based command validation. Requires description but offers fewer guardrails. | Go lacks structured parsing and detailed permission prompts used by TS to gate risky commands.【F:packages/opencode/src/tool/bash.ts†L34-L150】【F:go-opencode/internal/tool/bash.go†L15-L123】
| Output handling | Truncates long output, tracks execution metadata, and runs in chosen shell with timeout defaults. | Similar timeout/output truncation but without metadata updates or LSP/file integrations. | Core execution exists in both, but observability differs.【F:packages/opencode/src/tool/bash.ts†L131-L150】【F:go-opencode/internal/tool/bash.go†L90-L164】

## Task/Subagent Tool

| Aspect | TypeScript | Go | Notes |
| --- | --- | --- | --- |
| Support level | Fully integrated as a built-in tool with workflow orchestration and agent permissions. | Exists but registered separately and relies on external executor; lacks workflow orchestration features. | Go provides the entry point but not the richer orchestration present in TS.【F:packages/opencode/src/tool/registry.ts†L95-L105】【F:go-opencode/internal/tool/registry.go†L129-L135】

## Summary of Gaps

1. **Missing tools**: Go lacks search (`websearch`, `codesearch`), workflow, invalid-tool handler, and experimental batch functionality, as well as plugin-based tool discovery.【F:packages/opencode/src/tool/registry.ts†L84-L107】【F:go-opencode/internal/tool/registry.go†L104-L123】
2. **Safety and permissions**: TypeScript wraps read/edit/write/bash with workspace-boundary enforcement, permission prompts, and LSP/file-time integration; Go generally omits these checks, making writes/reads/edits less constrained.【F:packages/opencode/src/tool/read.ts†L26-L151】【F:packages/opencode/src/tool/edit.ts†L44-L172】【F:packages/opencode/src/tool/write.ts†L20-L97】【F:packages/opencode/src/tool/bash.ts†L34-L150】【F:go-opencode/internal/tool/read.go†L16-L146】【F:go-opencode/internal/tool/edit.go†L72-L229】【F:go-opencode/internal/tool/write.go†L59-L115】【F:go-opencode/internal/tool/bash.go†L15-L164】
3. **Result richness**: TypeScript tools emit diffs, diagnostics, and structured metadata (e.g., additions/deletions, preview snippets); Go now includes diff payloads and line totals for edits and writes but still omits diagnostics and other LSP-driven feedback.【F:packages/opencode/src/tool/read.ts†L123-L151】【F:packages/opencode/src/tool/edit.ts†L109-L172】【F:packages/opencode/src/tool/write.ts†L71-L97】【F:go-opencode/internal/tool/read.go†L117-L146】【F:go-opencode/internal/tool/edit.go†L129-L229】【F:go-opencode/internal/tool/write.go†L71-L105】
4. **UX polish**: TypeScript adds helpful behaviors—file-not-found suggestions, PDF/image previews with metadata, truncation notices with exact offsets, and hierarchical listings with ignores—that Go implementations currently lack.【F:packages/opencode/src/tool/read.ts†L75-L151】【F:packages/opencode/src/tool/ls.ts†L8-L104】【F:go-opencode/internal/tool/read.go†L83-L146】【F:go-opencode/internal/tool/list.go†L15-L116】

