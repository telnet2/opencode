# Interactive Mode Design

## Overview

Add interactive REPL mode to kiana-v6 CLI with `-i/--interactive` flag.

## Features

### Input Handling
- **Enter** sends message
- **Ctrl+J** inserts newline
- **Ctrl+C** exits interactive mode
- Multi-line display: `>` first line, `»` continuation lines

### ESC Interrupt
- First ESC: shows "Press ESC again to cancel"
- Second ESC (within 2s): calls `agent.abort()`, returns to prompt
- Timeout: if second ESC not pressed within 2s, cancel pending abort

### Visual Flow
```
> first line of prompt
» second line
» third line

[Agent streams response here]

--- Session idle ---

> next prompt
```

## Implementation

### New File: `src/interactive.ts`

```typescript
interface InteractiveState {
  mode: 'idle' | 'streaming' | 'esc_pending' | 'aborting'
  inputLines: string[]
  cursorLine: number
  cursorCol: number
  escTimeout?: NodeJS.Timeout
}

class InteractiveInput {
  start(): void      // Enter raw mode, start listening
  stop(): void       // Exit raw mode, cleanup
  showPrompt(): void // Display input prompt
  showEscWarning(): void
}
```

### Changes to `cli.ts`

1. Add `-i/--interactive` flag to Args interface and parseArgs
2. Add to printHelp()
3. Add `runInteractive()` function
4. Validation: `-i` + `-p` is invalid

### State Machine

```
IDLE → (Enter) → STREAMING → (ESC) → ESC_PENDING → (ESC) → ABORTING → IDLE
                     ↑                    │
                     └── (2s timeout) ────┘
```

## Decisions

- Custom readline wrapper (no external deps)
- Input hidden during response
- Implies `-H` (human-readable output)
- Session persists across prompts in same interactive session
