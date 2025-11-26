# Memsh TUI Browser - Architecture & Implementation Plan

## Overview

The Memsh Browser is a TUI dialog that allows users to browse directories and files from a remote memsh session within OpenCode. It opens via a shortcut key and provides a file explorer interface similar to existing dialogs.

## Architecture

### Component Hierarchy

```
DialogProvider (existing)
└── Dialog (modal container)
    └── DialogMemshBrowser (new component)
        ├── Header (title, path breadcrumb, esc hint)
        ├── Search Input (filter current directory)
        ├── File List (scrollable)
        │   ├── Parent Directory Entry (..)
        │   ├── Directory Entries (📁)
        │   └── File Entries (📄)
        └── Footer (keybind hints)
```

### State Management

```typescript
interface BrowserState {
  // Connection state
  session: Session | null
  connectionStatus: "disconnected" | "connecting" | "connected" | "error"
  error: string | null

  // Navigation state
  currentPath: string
  history: string[]           // For back navigation
  historyIndex: number

  // Directory contents
  entries: DirectoryEntry[]
  loading: boolean

  // Selection state
  selectedIndex: number
  filter: string
}

interface DirectoryEntry {
  name: string
  type: "file" | "directory"
  size?: number
  modified?: string
  permissions?: string
}
```

### Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    DialogMemshBrowser                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐ │
│  │  Keybind     │────▶│   Action     │────▶│   State      │ │
│  │  Handler     │     │   Handler    │     │   Update     │ │
│  └──────────────┘     └──────────────┘     └──────────────┘ │
│         │                    │                    │          │
│         │                    │                    ▼          │
│         │                    │            ┌──────────────┐   │
│         │                    │            │    Render    │   │
│         │                    │            └──────────────┘   │
│         │                    │                    │          │
│         │                    ▼                    │          │
│         │            ┌──────────────┐             │          │
│         │            │   Session    │             │          │
│         │            │   (memsh)    │             │          │
│         │            └──────────────┘             │          │
│         │                    │                    │          │
│         │                    ▼                    │          │
│         │            ┌──────────────┐             │          │
│         └───────────▶│   ls()      │◀────────────┘          │
│                      │   cd()      │                         │
│                      │   pwd()     │                         │
│                      └──────────────┘                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Integration Points

### 1. Keybind Configuration

Add new keybind to `config.ts`:

```typescript
// In Config.Keybinds schema
memsh_browser: z.string().optional().default("<leader>f").describe("Open memsh file browser")
```

### 2. Command Registration

Register in `app.tsx`:

```typescript
command.register(() => [
  // ... existing commands
  {
    title: "Browse files (memsh)",
    value: "memsh.browser",
    keybind: "memsh_browser",
    category: "System",
    onSelect: () => {
      dialog.replace(() => <DialogMemshBrowser />)
    },
  },
])
```

### 3. Session Provider (New Context)

Create a new context for memsh session management:

```typescript
// context/memsh.tsx
export const { use: useMemsh, provider: MemshProvider } = createSimpleContext({
  name: "Memsh",
  init: () => {
    const [state, setState] = createStore({
      session: null as Session | null,
      status: "disconnected" as ConnectionStatus,
    })

    return {
      get session() { return state.session },
      get status() { return state.status },
      async connect(options: SessionOptions) { ... },
      async disconnect() { ... },
    }
  }
})
```

## Component Implementation

### DialogMemshBrowser.tsx

```typescript
import { useDialog } from "@tui/ui/dialog"
import { DialogSelect } from "@tui/ui/dialog-select"
import { createSession, Session } from "@opencode/memsh-cli"
import { createSignal, createEffect, onMount, onCleanup, For, Show } from "solid-js"
import { createStore } from "solid-js/store"
import { useTheme } from "@tui/context/theme"
import { Keybind } from "@/util/keybind"

interface DirectoryEntry {
  name: string
  type: "file" | "directory"
  size?: string
  isHidden: boolean
}

export function DialogMemshBrowser() {
  const dialog = useDialog()
  const { theme } = useTheme()

  const [state, setState] = createStore({
    session: null as Session | null,
    currentPath: "/",
    entries: [] as DirectoryEntry[],
    loading: true,
    showHidden: false,
    error: null as string | null,
  })

  // Initialize session on mount
  onMount(async () => {
    try {
      const session = await createSession({
        baseUrl: "http://localhost:8080", // configurable
      })
      setState("session", session)
      await loadDirectory("/")
    } catch (err) {
      setState("error", String(err))
    }
  })

  // Cleanup on unmount
  onCleanup(async () => {
    if (state.session) {
      await state.session.close()
    }
  })

  async function loadDirectory(path: string) {
    if (!state.session) return

    setState("loading", true)
    try {
      const rawEntries = await state.session.ls(path, {
        all: state.showHidden,
        long: true
      })

      const entries = await parseEntries(rawEntries, path)
      setState({
        currentPath: path,
        entries,
        loading: false,
        error: null,
      })
    } catch (err) {
      setState({ loading: false, error: String(err) })
    }
  }

  async function parseEntries(raw: string[], basePath: string): Promise<DirectoryEntry[]> {
    // Parse ls -l output or simple ls output
    // Returns structured entries
  }

  function navigateUp() {
    const parent = state.currentPath.split("/").slice(0, -1).join("/") || "/"
    loadDirectory(parent)
  }

  async function handleSelect(entry: DirectoryEntry) {
    if (entry.type === "directory") {
      const newPath = state.currentPath === "/"
        ? `/${entry.name}`
        : `${state.currentPath}/${entry.name}`
      await loadDirectory(newPath)
    }
    // For files, just show selection (future: preview, copy path, etc.)
  }

  const options = () => {
    const result = []

    // Add parent directory entry
    if (state.currentPath !== "/") {
      result.push({
        title: "..",
        value: { name: "..", type: "directory" as const, isHidden: false },
        description: "Parent directory",
        footer: "📁",
      })
    }

    // Filter and sort entries
    const filtered = state.entries
      .filter(e => state.showHidden || !e.isHidden)
      .sort((a, b) => {
        // Directories first, then alphabetical
        if (a.type !== b.type) return a.type === "directory" ? -1 : 1
        return a.name.localeCompare(b.name)
      })

    for (const entry of filtered) {
      result.push({
        title: entry.name,
        value: entry,
        footer: entry.type === "directory" ? "📁" : "📄",
        description: entry.size,
      })
    }

    return result
  }

  return (
    <Show when={!state.error} fallback={<ErrorView error={state.error} />}>
      <box flexDirection="column" gap={1}>
        <box paddingLeft={4} paddingRight={4}>
          <text fg={theme.textMuted}>{state.currentPath}</text>
        </box>
        <DialogSelect
          title="Memsh Browser"
          options={options()}
          onSelect={(opt) => handleSelect(opt.value)}
          keybind={[
            {
              keybind: Keybind.parse("backspace")[0],
              title: "back",
              onTrigger: () => navigateUp(),
            },
            {
              keybind: Keybind.parse("ctrl+h")[0],
              title: "hidden",
              onTrigger: () => setState("showHidden", !state.showHidden),
            },
            {
              keybind: Keybind.parse("ctrl+r")[0],
              title: "refresh",
              onTrigger: () => loadDirectory(state.currentPath),
            },
          ]}
        />
      </box>
    </Show>
  )
}
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `<leader>f` | Open memsh browser (global) |
| `Enter` | Open directory / Select file |
| `Backspace` | Navigate to parent directory |
| `Ctrl+H` | Toggle hidden files |
| `Ctrl+R` | Refresh current directory |
| `Up/Down` | Navigate list |
| `PageUp/PageDown` | Fast scroll |
| `Esc` | Close browser |
| `/` or start typing | Filter entries |

## File Structure

```
packages/opencode/src/cli/cmd/tui/
├── component/
│   ├── dialog-memsh-browser.tsx    # Main browser component
│   └── dialog-memsh-browser.md     # This document
├── context/
│   └── memsh.tsx                   # Memsh session context (optional)
└── app.tsx                         # Register keybind & command
```

## Implementation Steps

### Phase 1: Basic Browser (MVP)

1. **Add keybind configuration** (`config.ts`)
   - Add `memsh_browser` keybind with default `<leader>f`

2. **Create DialogMemshBrowser component** (`dialog-memsh-browser.tsx`)
   - Basic component structure
   - Session initialization
   - Directory listing with `session.ls()`
   - Navigation (enter directory, go back)

3. **Register command** (`app.tsx`)
   - Add command to open browser
   - Wire up keybind

4. **Add memsh-cli dependency**
   - Import and use Session class from `@opencode/memsh-cli`

### Phase 2: Enhanced Features

5. **Add file type icons**
   - Different icons for directories, files, symlinks

6. **Implement path breadcrumb**
   - Clickable path segments for quick navigation

7. **Add file preview** (optional)
   - Preview file contents on selection
   - Show file metadata (size, modified date, permissions)

8. **Add context menu actions**
   - Copy path to clipboard
   - Open in editor
   - Read file content

### Phase 3: Advanced Features

9. **Multiple session support**
   - Connect to different memsh servers
   - Session switcher

10. **Favorites/Bookmarks**
    - Save frequently accessed paths
    - Quick jump to bookmarked locations

11. **Search across directories**
    - Recursive file search using glob tool
    - Search file contents using grep tool

## Configuration Options

```jsonc
// opencode.json
{
  "memsh": {
    "defaultUrl": "http://localhost:8080",
    "showHiddenByDefault": false,
    "defaultPath": "/"
  },
  "keybinds": {
    "memsh_browser": "<leader>f"
  }
}
```

## Testing Considerations

1. **Unit tests for parsing**
   - Test `ls` output parsing
   - Test path manipulation

2. **Integration tests**
   - Mock memsh session
   - Test navigation flows

3. **Manual testing**
   - Test with different terminal sizes
   - Test with large directories
   - Test error handling (disconnection, invalid paths)

## Dependencies

- `@opencode/memsh-cli` - Memsh client library
- Existing TUI infrastructure:
  - `DialogSelect` for list rendering
  - `useDialog` for modal management
  - `useKeybind` for keyboard handling
  - `useTheme` for styling
