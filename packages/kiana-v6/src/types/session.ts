import { z } from "zod"

export const FileDiffSchema = z.object({
  file: z.string(),
  additions: z.number(),
  deletions: z.number(),
  diff: z.string().optional(),
})

export type FileDiff = z.infer<typeof FileDiffSchema>

export const SessionSummarySchema = z.object({
  additions: z.number(),
  deletions: z.number(),
  files: z.number(),
  diffs: z.array(FileDiffSchema).optional(),
})

export type SessionSummary = z.infer<typeof SessionSummarySchema>

export const SessionShareSchema = z.object({
  url: z.string(),
})

export type SessionShare = z.infer<typeof SessionShareSchema>

export const SessionTimeSchema = z.object({
  created: z.number(),
  updated: z.number(),
  compacting: z.number().optional(),
})

export type SessionTime = z.infer<typeof SessionTimeSchema>

export const SessionRevertSchema = z.object({
  messageID: z.string(),
  partID: z.string().optional(),
  snapshot: z.string().optional(),
  diff: z.string().optional(),
})

export type SessionRevert = z.infer<typeof SessionRevertSchema>

export const SessionInfoSchema = z.object({
  id: z.string(),
  projectID: z.string(),
  directory: z.string(),
  parentID: z.string().optional(),
  title: z.string(),
  version: z.string(),
  time: SessionTimeSchema,
  summary: SessionSummarySchema.optional(),
  share: SessionShareSchema.optional(),
  revert: SessionRevertSchema.optional(),
})

export type SessionInfo = z.infer<typeof SessionInfoSchema>
