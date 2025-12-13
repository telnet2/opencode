# Simple CLI for OpenCode Server

## Purpose & Goals
- Provide a lightweight, Python REPL-style CLI to interact with an OpenCode server without a sidebar, command palette, or other UI chrome.
- Focus on a scrolling text conversation that shows assistant/user messages and tool calls inline.
- Preserve the existing `/` slash-command entry point semantics from the main CLI.
- Minimize dependencies and startup time; keep the design embeddable for other CLIs or scripts.

## Non-Goals
- No GUI/TUI chrome (sidebars, panes, menus, mouse interactions).
- No project tree navigation or file editor; stick to conversational interaction with tool call visibility.
- No speculative multi-panel layouts or command palette features.

## User Experience
- Single scrollable transcript that prints user inputs, assistant responses, and tool calls in chronological order.
- Tool calls rendered as compact blocks showing tool name, arguments, status, and outputs; stream output as it arrives when supported.
- Input loop mirrors a Python REPL: prompt shows current working directory or concise context (e.g., `simple-cli>`), reads a line, and sends it.
- `/` prefix triggers commands (e.g., `/help`, `/exit`, `/config`, `/provider`) consistent with existing behavior in `packages/opencode`.
- Support multiline entry via `\` line continuation to allow longer prompts without breaking the REPL feel.

## Functional Requirements
1. **Connection to OpenCode server**
   - Accept server URL and auth token from flags/env/config.
   - Verify server health on startup and provide actionable error messages.
2. **Send & stream messages**
   - Send user messages to the server and stream assistant responses line-by-line.
   - Display tool calls as they are issued; stream tool outputs where possible.
3. **Slash commands**
   - Maintain `/` command parser compatible with current CLI shortcuts; unknown commands fall back to help.
   - Provide `/exit` and `/help` built-ins even without a running server connection.
4. **Session handling**
   - Optional session resume by passing a session ID; otherwise start a new session per process.
   - Store minimal session metadata (session ID, model, timestamp) in a local state file for reuse.
5. **Logging & verbosity**
   - `--quiet`, `--verbose`, and `--json` output modes to support scripting and debugging.
   - Structured logs to stderr, conversation to stdout.
6. **Error handling**
   - Graceful handling of network failures with retry hints.
   - Clear formatting for tool errors separate from assistant messages.

## Architecture Outline
- **Entry point:** `packages/simple-cli/src/index.ts` invoked via `package.json` bin (e.g., `simple-opencode`).
- **Modules**
  - `cli.ts`: argument parsing, startup banner, environment validation.
  - `repl.ts`: main loop, prompt rendering, multiline input handling.
  - `commands.ts`: `/` command registry and dispatch; reuse parsers from `packages/opencode` where feasible.
  - `client.ts`: thin wrapper over existing OpenCode server API client (shared SDK if available).
  - `renderer.ts`: formatting of messages, tool calls, and streaming updates.
  - `state.ts`: manage session metadata caching and loading.
- **Shared code:** Prefer importing shared types/SDK from `packages/opencode` or `packages/sdk`; avoid duplicating request/response shapes.

## Configuration & Flags
- Default server URL from env `OPENCODE_SERVER_URL`; token from `OPENCODE_API_KEY`.
- CLI flags: `--url`, `--api-key`, `--model`, `--session`, `--quiet`, `--verbose`, `--json`, `--no-color`.
- Config file lookup: `~/.opencode/simple-cli.json` with optional project-level `.opencode/simple-cli.json` overriding globals.

## Compatibility with `/` Commands
- Keep `/` syntax identical to main CLI for:
  - `/model <name>`
  - `/agent <name>`
  - `/provider <name>`
  - `/help`
  - `/exit`
- Provide a compatibility layer so future additions in `packages/opencode` can register without code duplication (shared command map).

## Tool Call Presentation
- Show tool invocations with:
  - Header: `→ tool <name>` with timestamp and status (running/success/fail).
  - Arguments pretty-printed (respect redaction for secrets).
  - Streaming stdout/stderr lines with prefix.
  - Final result summary once completed.

## Telemetry & Metrics
- Minimal: count of requests, latencies, and tool call durations (in-memory only unless opted-in).
- Optional `--trace` flag to emit timing diagnostics for debugging.

## Testing Strategy
- Unit tests for slash command parsing, multiline input handling, and renderer formatting.
- Integration test against a mocked server to validate streaming and tool call rendering.
- Snapshot tests for `/help` output and typical conversation transcripts.

## Delivery Plan
1. **Scaffold** `packages/simple-cli` with build/test scripts and bin entry.
2. **REPL loop & slash commands** with compatibility layer tied to shared command definitions.
3. **Server client & streaming** integration using shared SDK; render tool calls incrementally.
4. **Config/flags & logging** including quiet/json modes and color toggles.
5. **Testing & docs** covering usage, configuration, and examples.
