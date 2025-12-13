# Desktop client builds

OpenCode ships a SolidJS client (`packages/desktop`) that can be bundled as a native Tauri app (`packages/tauri`). This guide covers how to develop and build the Tauri desktop app and how to reuse the same UI inside an Electron shell.

## Prerequisites

- [Bun](https://bun.sh/) for workspace scripts
- Rust toolchain with the target for your OS (e.g., `aarch64-apple-darwin`, `x86_64-unknown-linux-gnu`)
- [Tauri CLI](https://tauri.app/) (installed via the workspace dev dependencies)
- Workspace dependencies installed from the repo root: `bun install`

## Tauri desktop app

The Tauri app lives in `packages/tauri` and depends on the Solid client from `packages/desktop`. Key scripts are exposed in `package.json`.

### Develop (hot reload)

```bash
cd packages/tauri
bun run tauri dev
```

- Runs the Vite dev server defined by the `dev` script.
- The Tauri `predev` hook builds/copies the `opencode` sidecar binary when missing so the native shell can spawn it at runtime.

### Release build

```bash
cd packages/tauri
bun run tauri build
```

- Produces the native bundles for your platform using the Tauri CLI.
- Set `RUST_TARGET`/`TAURI_ENV_TARGET_TRIPLE` to one of the supported targets in `scripts/utils.ts` if you need to cross-compile.

## Electron usage

The SolidJS desktop UI can also serve as an Electron renderer:

1. Start the UI in dev mode for live reload:
   ```bash
   cd packages/desktop
   bun run dev -- --host
   ```
2. Point an Electron `BrowserWindow` at the dev server URL during development, or run `bun run build` to emit static assets in `packages/desktop/dist` and load them via `loadFile` in your Electron main process.

This approach reuses the same SolidJS UI in both the Tauri native build and any Electron shell you provide.
