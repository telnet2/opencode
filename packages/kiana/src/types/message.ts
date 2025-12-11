import { z } from "zod"
import { FileDiffSchema } from "./session.js"

export const ModelRefSchema = z.object({
  providerID: z.string(),
  modelID: z.string(),
})

export type ModelRef = z.infer<typeof ModelRefSchema>

export const MessageTimeSchema = z.object({
  created: z.number(),
  completed: z.number().optional(),
})

export type MessageTime = z.infer<typeof MessageTimeSchema>

export const MessagePathSchema = z.object({
  cwd: z.string(),
  root: z.string(),
})

export type MessagePath = z.infer<typeof MessagePathSchema>

export const CacheUsageSchema = z.object({
  read: z.number(),
  write: z.number(),
})

export type CacheUsage = z.infer<typeof CacheUsageSchema>

export const TokenUsageSchema = z.object({
  input: z.number(),
  output: z.number(),
  reasoning: z.number(),
  cache: CacheUsageSchema,
})

export type TokenUsage = z.infer<typeof TokenUsageSchema>

export const MessageErrorDataSchema = z.object({
  message: z.string(),
  providerID: z.string().optional(),
  status: z.number().optional(),
  retryable: z.boolean().optional(),
})

export type MessageErrorData = z.infer<typeof MessageErrorDataSchema>

export const MessageErrorSchema = z.object({
  name: z.enum([
    "UnknownError",
    "ProviderAuthError",
    "MessageOutputLengthError",
    "MessageAbortedError",
    "APIError",
  ]),
  data: MessageErrorDataSchema,
})

export type MessageError = z.infer<typeof MessageErrorSchema>

export const UserMessageSummarySchema = z.object({
  title: z.string().optional(),
  body: z.string().optional(),
  diffs: z.array(FileDiffSchema).optional(),
})

export type UserMessageSummary = z.infer<typeof UserMessageSummarySchema>

export const MessageInfoSchema = z.object({
  id: z.string(),
  sessionID: z.string(),
  role: z.enum(["user", "assistant"]),
  time: MessageTimeSchema,

  // User fields
  agent: z.string().optional(),
  model: ModelRefSchema.optional(),
  system: z.string().optional(),
  tools: z.record(z.string(), z.boolean()).optional(),
  summary: UserMessageSummarySchema.optional(),

  // Assistant fields
  parentID: z.string().optional(),
  modelID: z.string().optional(),
  providerID: z.string().optional(),
  mode: z.string().optional(),
  path: MessagePathSchema.optional(),
  cost: z.number().optional(),
  tokens: TokenUsageSchema.optional(),
  finish: z.string().optional(),
  error: MessageErrorSchema.optional(),
})

export type MessageInfo = z.infer<typeof MessageInfoSchema>
