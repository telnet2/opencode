import { createOpencodeClient, type Event, type SessionPromptResponse } from "@opencode-ai/sdk/v2"
import { Renderer } from "./renderer"
import type { ResolvedConfig, SessionStateEntry } from "./types"
import { loadSessionState, persistSessionState } from "./state"

export interface SimpleClient {
  sessionID: string
  sendPrompt: (text: string, opts?: { model?: string; agent?: string; provider?: string }) => Promise<SessionPromptResponse>
  close: () => void
}

export async function createSimpleClient(config: ResolvedConfig, renderer: Renderer): Promise<SimpleClient> {
  const headers: Record<string, string> = {}
  if (config.apiKey) {
    headers["authorization"] = `Bearer ${config.apiKey}`
  }

  const client = createOpencodeClient({
    baseUrl: config.url,
    headers,
  })

  // health check via list
  await client.session.list({ directory: config.directory }, { responseStyle: "data", throwOnError: true })

  let sessionID: string | undefined = config.session
  let cached: SessionStateEntry | undefined
  if (!sessionID) {
    cached = loadSessionState(config)
    sessionID = cached?.sessionID
  }

  if (!sessionID) {
    const created = await client.session.create(
      { directory: config.directory },
      { responseStyle: "data", throwOnError: true },
    )
    const createdSession = unwrapResponse(created)
    sessionID = createdSession.id
    cached = { sessionID, model: config.model, provider: config.provider, agent: config.agent, updatedAt: Date.now() }
    persistSessionState(config, cached)
  }

  renderer.trace("session", { sessionID })

  const abort = new AbortController()
  const streamPromise = client.event.subscribe({ directory: config.directory }, { signal: abort.signal })

  streamPromise.then(async ({ stream }) => {
    for await (const evt of stream) {
      const event = evt as Event
      if (!relevantEvent(event, sessionID!)) continue
      renderer.event(event)
    }
  }).catch((err) => {
    renderer.trace("event stream closed", { error: String(err) })
  })

  return {
    sessionID: sessionID!,
    async sendPrompt(text, opts): Promise<SessionPromptResponse> {
      const { model = config.model, agent = config.agent, provider = config.provider } = opts ?? {}
      const body: any = {
        parts: [{ type: "text", text }],
      }
      if (model && provider) {
        body.model = { modelID: model, providerID: provider }
      }
      if (agent) body.agent = agent
      const response = await client.session.prompt(
        {
          sessionID: sessionID!,
          directory: config.directory,
          ...body,
        },
        { responseStyle: "data", throwOnError: true },
      )
      const data = unwrapResponse(response)
      const entry: SessionStateEntry = {
        sessionID: sessionID!,
        model: model ?? cached?.model,
        provider: provider ?? cached?.provider,
        agent: agent ?? cached?.agent,
        updatedAt: Date.now(),
      }
      persistSessionState(config, entry)
      return data
    },
    close() {
      abort.abort()
    },
  }
}

function unwrapResponse<T>(response: T | { data: T }): T {
  if (response && typeof response === "object" && "data" in response) {
    return (response as { data: T }).data
  }
  return response as T
}

function relevantEvent(event: Event, sessionID: string): boolean {
  switch (event.type) {
    case "message.updated":
      return event.properties.info.sessionID === sessionID
    case "message.removed":
      return event.properties.sessionID === sessionID
    case "message.part.updated":
      return event.properties.part.sessionID === sessionID
    case "message.part.removed":
      return event.properties.sessionID === sessionID
    case "session.status":
    case "session.idle":
      return event.properties.sessionID === sessionID
    case "session.error":
      return event.properties.sessionID === sessionID
    case "session.updated":
    case "session.created":
    case "session.deleted":
      return event.properties.info.id === sessionID
    default:
      return false
  }
}
