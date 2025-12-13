import { z } from "zod"
import { SessionInfoSchema, FileDiffSchema } from "./session.js"
import { MessageInfoSchema } from "./message.js"
import { PartSchema } from "./part.js"

// Subagent context - added to events from subagent sessions
export const SubagentContextSchema = z.object({
  parentSessionID: z.string(),
  depth: z.number(), // 1 = direct subagent, 2 = subagent of subagent, etc.
  agentType: z.string().optional(), // e.g., "Explore", "Plan", etc.
})

export type SubagentContext = z.infer<typeof SubagentContextSchema>

// Session events
export const SessionCreatedEventSchema = z.object({
  type: z.literal("session.created"),
  properties: z.object({
    session: SessionInfoSchema,
  }),
})

export const SessionUpdatedEventSchema = z.object({
  type: z.literal("session.updated"),
  properties: z.object({
    session: SessionInfoSchema,
  }),
})

export const SessionDeletedEventSchema = z.object({
  type: z.literal("session.deleted"),
  properties: z.object({
    session: SessionInfoSchema,
  }),
})

export const SessionIdleEventSchema = z.object({
  type: z.literal("session.idle"),
  properties: z.object({
    sessionID: z.string(),
  }),
})

export const SessionCompactedEventSchema = z.object({
  type: z.literal("session.compacted"),
  properties: z.object({
    sessionID: z.string(),
  }),
})

export const SessionDiffEventSchema = z.object({
  type: z.literal("session.diff"),
  properties: z.object({
    sessionID: z.string(),
    diff: z.array(FileDiffSchema),
  }),
})

export const SessionStatusEventSchema = z.object({
  type: z.literal("session.status"),
  properties: z.object({
    sessionID: z.string(),
    status: z.object({
      state: z.enum(["idle", "running", "error"]),
      message: z.string().optional(),
    }),
  }),
})

// Message events
export const MessageCreatedEventSchema = z.object({
  type: z.literal("message.created"),
  properties: z.object({
    message: MessageInfoSchema,
  }),
})

export const MessageUpdatedEventSchema = z.object({
  type: z.literal("message.updated"),
  properties: z.object({
    message: MessageInfoSchema,
  }),
})

export const MessageRemovedEventSchema = z.object({
  type: z.literal("message.removed"),
  properties: z.object({
    sessionID: z.string(),
    messageID: z.string(),
  }),
})

export const MessagePartUpdatedEventSchema = z.object({
  type: z.literal("message.part.updated"),
  properties: z.object({
    part: PartSchema,
    delta: z.string().optional(),
  }),
})

export const MessagePartRemovedEventSchema = z.object({
  type: z.literal("message.part.removed"),
  properties: z.object({
    sessionID: z.string(),
    messageID: z.string(),
    partID: z.string(),
  }),
})

// Todo events
export const TodoUpdatedEventSchema = z.object({
  type: z.literal("todo.updated"),
  properties: z.object({
    sessionID: z.string(),
    todos: z.array(
      z.object({
        content: z.string(),
        status: z.enum(["pending", "in_progress", "completed"]),
        activeForm: z.string(),
      })
    ),
  }),
})

// Base event schema (without context)
const BaseEventSchema = z.discriminatedUnion("type", [
  SessionCreatedEventSchema,
  SessionUpdatedEventSchema,
  SessionDeletedEventSchema,
  SessionIdleEventSchema,
  SessionCompactedEventSchema,
  SessionDiffEventSchema,
  SessionStatusEventSchema,
  MessageCreatedEventSchema,
  MessageUpdatedEventSchema,
  MessageRemovedEventSchema,
  MessagePartUpdatedEventSchema,
  MessagePartRemovedEventSchema,
  TodoUpdatedEventSchema,
])

// Union of all events with optional subagent context
export const EventSchema = BaseEventSchema.and(
  z.object({
    context: SubagentContextSchema.optional(),
  })
)

export type Event = z.infer<typeof EventSchema>

// Event type enum for convenience
export type EventType = Event["type"]

// Helper to create typed events
export function createEvent<T extends EventType>(
  type: T,
  properties: Extract<Event, { type: T }>["properties"]
): Extract<Event, { type: T }> {
  return { type, properties } as Extract<Event, { type: T }>
}

// Category types for export
export type SessionEvent = z.infer<
  | typeof SessionCreatedEventSchema
  | typeof SessionUpdatedEventSchema
  | typeof SessionDeletedEventSchema
  | typeof SessionIdleEventSchema
  | typeof SessionCompactedEventSchema
  | typeof SessionDiffEventSchema
  | typeof SessionStatusEventSchema
>

export type MessageEvent = z.infer<
  | typeof MessageCreatedEventSchema
  | typeof MessageUpdatedEventSchema
  | typeof MessageRemovedEventSchema
>

export type PartEvent = z.infer<
  | typeof MessagePartUpdatedEventSchema
  | typeof MessagePartRemovedEventSchema
>

export type TodoEvent = z.infer<typeof TodoUpdatedEventSchema>
