import fs from "fs"
import path from "path"
import type { ResolvedConfig, SessionStateEntry, StateFile } from "./types"

function ensureDir(filePath: string) {
  const dir = path.dirname(filePath)
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true })
  }
}

function readState(filePath: string): StateFile {
  try {
    const raw = fs.readFileSync(filePath, "utf8")
    return JSON.parse(raw) as StateFile
  } catch {
    return { sessions: {} }
  }
}

export function loadSessionState(config: ResolvedConfig): SessionStateEntry | undefined {
  const state = readState(config.sessionFile)
  if (config.session && state.sessions[config.session]) {
    return state.sessions[config.session]
  }
  if (config.session) return undefined
  if (state.lastSessionID && state.sessions[state.lastSessionID]) return state.sessions[state.lastSessionID]
  return undefined
}

export function persistSessionState(config: ResolvedConfig, entry: SessionStateEntry) {
  const state = readState(config.sessionFile)
  ensureDir(config.sessionFile)
  state.sessions[entry.sessionID] = entry
  state.lastSessionID = entry.sessionID
  fs.writeFileSync(config.sessionFile, JSON.stringify(state, null, 2))
}
