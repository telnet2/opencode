import { z } from "zod"
import { TokenUsageSchema } from "./message.js"

// Common time structures
export const PartTimeSchema = z.object({
  start: z.number().optional(),
  end: z.number().optional(),
})

export type PartTime = z.infer<typeof PartTimeSchema>

export const ToolTimeSchema = z.object({
  start: z.number(),
  end: z.number().optional(),
  compacted: z.number().optional(),
})

export type ToolTime = z.infer<typeof ToolTimeSchema>

// Base part fields
const BasePartSchema = z.object({
  id: z.string(),
  sessionID: z.string(),
  messageID: z.string(),
})

// Text part
export const TextPartSchema = BasePartSchema.extend({
  type: z.literal("text"),
  text: z.string(),
  synthetic: z.boolean().optional(),
  ignored: z.boolean().optional(),
  time: PartTimeSchema.optional(),
  metadata: z.record(z.string(), z.unknown()).optional(),
})

export type TextPart = z.infer<typeof TextPartSchema>

// Reasoning part
export const ReasoningPartSchema = BasePartSchema.extend({
  type: z.literal("reasoning"),
  text: z.string(),
  time: PartTimeSchema.optional(),
  metadata: z.record(z.string(), z.unknown()).optional(),
})

export type ReasoningPart = z.infer<typeof ReasoningPartSchema>

// Tool state variants
export const ToolStatePendingSchema = z.object({
  status: z.literal("pending"),
  input: z.record(z.string(), z.unknown()),
  raw: z.string(),
})

export const ToolStateRunningSchema = z.object({
  status: z.literal("running"),
  input: z.record(z.string(), z.unknown()),
  title: z.string().optional(),
  metadata: z.record(z.string(), z.unknown()).optional(),
  time: z.object({ start: z.number() }),
})

export const ToolStateCompletedSchema = z.object({
  status: z.literal("completed"),
  input: z.record(z.string(), z.unknown()),
  output: z.string(),
  title: z.string(),
  metadata: z.record(z.string(), z.unknown()),
  time: z.object({
    start: z.number(),
    end: z.number(),
    compacted: z.number().optional(),
  }),
  attachments: z.array(z.lazy(() => FilePartSchema)).optional(),
})

export const ToolStateErrorSchema = z.object({
  status: z.literal("error"),
  input: z.record(z.string(), z.unknown()),
  error: z.string(),
  metadata: z.record(z.string(), z.unknown()).optional(),
  time: z.object({
    start: z.number(),
    end: z.number(),
  }),
})

export const ToolStateSchema = z.discriminatedUnion("status", [
  ToolStatePendingSchema,
  ToolStateRunningSchema,
  ToolStateCompletedSchema,
  ToolStateErrorSchema,
])

export type ToolState = z.infer<typeof ToolStateSchema>

// Tool part
export const ToolPartSchema = BasePartSchema.extend({
  type: z.literal("tool"),
  callID: z.string(),
  tool: z.string(),
  state: ToolStateSchema,
  metadata: z.record(z.string(), z.unknown()).optional(),
})

export type ToolPart = z.infer<typeof ToolPartSchema>

// File part
export const FilePartSchema = BasePartSchema.extend({
  type: z.literal("file"),
  mime: z.string(),
  filename: z.string().optional(),
  url: z.string(),
  source: z
    .object({
      type: z.string(),
      url: z.string().optional(),
    })
    .optional(),
})

export type FilePart = z.infer<typeof FilePartSchema>

// Step start part
export const StepStartPartSchema = BasePartSchema.extend({
  type: z.literal("step-start"),
  snapshot: z.string().optional(),
})

export type StepStartPart = z.infer<typeof StepStartPartSchema>

// Step finish part
export const StepFinishPartSchema = BasePartSchema.extend({
  type: z.literal("step-finish"),
  reason: z.string(),
  snapshot: z.string().optional(),
  cost: z.number(),
  tokens: TokenUsageSchema.optional(),
})

export type StepFinishPart = z.infer<typeof StepFinishPartSchema>

// Snapshot part
export const SnapshotPartSchema = BasePartSchema.extend({
  type: z.literal("snapshot"),
  snapshot: z.string(),
})

export type SnapshotPart = z.infer<typeof SnapshotPartSchema>

// Patch part
export const PatchPartSchema = BasePartSchema.extend({
  type: z.literal("patch"),
  hash: z.string(),
  files: z.array(z.string()),
})

export type PatchPart = z.infer<typeof PatchPartSchema>

// Agent part
export const AgentPartSchema = BasePartSchema.extend({
  type: z.literal("agent"),
  name: z.string(),
  source: z
    .object({
      value: z.string(),
      start: z.number(),
      end: z.number(),
    })
    .optional(),
})

export type AgentPart = z.infer<typeof AgentPartSchema>

// Retry part
export const RetryPartSchema = BasePartSchema.extend({
  type: z.literal("retry"),
  attempt: z.number(),
  error: z.object({
    name: z.literal("APIError"),
    data: z.object({
      status: z.number().optional(),
      message: z.string(),
      retryable: z.boolean().optional(),
    }),
  }),
  time: z.object({
    created: z.number(),
  }),
})

export type RetryPart = z.infer<typeof RetryPartSchema>

// Compaction part
export const CompactionPartSchema = BasePartSchema.extend({
  type: z.literal("compaction"),
  auto: z.boolean(),
})

export type CompactionPart = z.infer<typeof CompactionPartSchema>

// Subtask part (Go-only, but included for compatibility)
export const SubtaskPartSchema = BasePartSchema.extend({
  type: z.literal("subtask"),
  prompt: z.string(),
  description: z.string(),
  agent: z.string(),
})

export type SubtaskPart = z.infer<typeof SubtaskPartSchema>

// Union of all parts
export const PartSchema = z.discriminatedUnion("type", [
  TextPartSchema,
  ReasoningPartSchema,
  ToolPartSchema,
  FilePartSchema,
  StepStartPartSchema,
  StepFinishPartSchema,
  SnapshotPartSchema,
  PatchPartSchema,
  AgentPartSchema,
  RetryPartSchema,
  CompactionPartSchema,
  SubtaskPartSchema,
])

export type Part = z.infer<typeof PartSchema>
