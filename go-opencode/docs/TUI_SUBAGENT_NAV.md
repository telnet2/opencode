# Subagent Session Navigation in the TUI

## What the `<leader> + ←/→` shortcut does
- The TypeScript TUI binds `session_child_cycle` and `session_child_cycle_reverse` to `<leader>right`/`<leader>left` by default, with the leader set to `ctrl+x`. This makes `ctrl+x` + arrow keys the out-of-the-box way to jump across subagent sessions.【F:packages/opencode/src/config/config.ts†L421-L474】
- When the Task tool renders, it explicitly reminds users of these bindings (“…to navigate between subagent sessions”).【F:packages/opencode/src/cli/cmd/tui/routes/session/index.tsx†L1514-L1517】
- Pressing the shortcut calls `moveChild`, which cycles through the current session’s parent/children list and `navigate`s to the next or previous session. This powers the seamless jump between a parent session and any child subagent sessions.【F:packages/opencode/src/cli/cmd/tui/routes/session/index.tsx†L223-L238】【F:packages/opencode/src/cli/cmd/tui/routes/session/index.tsx†L740-L757】

## Parity in Go OpenCode
- Go exposes the same keybind names and defaults (`leader` = `ctrl+x`, `session_child_cycle` = `<leader>right`, `session_child_cycle_reverse` = `<leader>left`), so the TUI receives identical config when pointed at the Go server—no keymap drift to break `ctrl+x` navigation.【F:go-opencode/pkg/types/config.go†L235-L280】
- Subagent runs in Go create real child sessions via the SubagentExecutor, so there is session lineage for the TUI to traverse when the shortcut fires.【F:go-opencode/internal/executor/subagent.go†L65-L139】
- The Go server also exposes the `/session/{id}/children` endpoint, ensuring the TUI can fetch and display the parent/child graph that the shortcut cycles through.【F:go-opencode/internal/server/routes.go†L33-L43】

**Conclusion:** Both the TypeScript and Go OpenCode stacks support `ctrl+x` + left/right to hop between parent and subagent sessions. Matching default keybinds plus the Go server’s child-session APIs keep the experience consistent across runtimes.
