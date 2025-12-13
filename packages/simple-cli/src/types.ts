export interface CliOptions {
  url?: string
  apiKey?: string
  model?: string
  provider?: string
  agent?: string
  session?: string
  quiet?: boolean
  verbose?: boolean
  json?: boolean
  noColor?: boolean
  directory?: string
  trace?: boolean
}

export interface ResolvedConfig extends CliOptions {
  url: string
  apiKey?: string
  sessionFile: string
}

export interface SessionStateEntry {
  sessionID: string
  model?: string
  provider?: string
  agent?: string
  updatedAt: number
}

export interface StateFile {
  sessions: Record<string, SessionStateEntry>
  lastSessionID?: string
}
