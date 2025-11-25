import { describe, expect, test, beforeEach, afterEach, mock } from "bun:test"
import path from "path"
import os from "os"
import fs from "fs"
import { SystemPrompt } from "../../src/session/system"
import { Session } from "../../src/session"
import { Log } from "../../src/util/log"
import { Instance } from "../../src/project/instance"

const projectRoot = path.join(__dirname, "../..")
Log.init({ print: false })

describe("Custom Prompt Feature", () => {
  describe("parseCustomPromptInput (via Session.create)", () => {
    test("should detect absolute file path", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: "/path/to/prompt.txt",
          })

          expect(session.customPrompt).toBeDefined()
          expect(session.customPrompt?.type).toBe("file")
          expect(session.customPrompt?.value).toBe("/path/to/prompt.txt")
          expect(session.customPrompt?.loadedAt).toBeDefined()

          await Session.remove(session.id)
        },
      })
    })

    test("should detect home directory path (~)", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: "~/prompts/my-prompt.md",
          })

          expect(session.customPrompt?.type).toBe("file")
          expect(session.customPrompt?.value).toBe("~/prompts/my-prompt.md")

          await Session.remove(session.id)
        },
      })
    })

    test("should detect relative path (./ prefix)", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: "./prompts/local.txt",
          })

          expect(session.customPrompt?.type).toBe("file")
          expect(session.customPrompt?.value).toBe("./prompts/local.txt")

          await Session.remove(session.id)
        },
      })
    })

    test("should detect parent directory path (../ prefix)", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: "../shared/prompt.txt",
          })

          expect(session.customPrompt?.type).toBe("file")
          expect(session.customPrompt?.value).toBe("../shared/prompt.txt")

          await Session.remove(session.id)
        },
      })
    })

    test("should detect file by .txt extension", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: "my-prompt.txt",
          })

          expect(session.customPrompt?.type).toBe("file")

          await Session.remove(session.id)
        },
      })
    })

    test("should detect file by .md extension", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: "instructions.md",
          })

          expect(session.customPrompt?.type).toBe("file")

          await Session.remove(session.id)
        },
      })
    })

    test("should detect inline prompt with newlines", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const inlinePrompt = "You are a helpful assistant.\nFollow these rules:\n1. Be concise"
          const session = await Session.create({
            customPrompt: inlinePrompt,
          })

          expect(session.customPrompt?.type).toBe("inline")
          expect(session.customPrompt?.value).toBe(inlinePrompt)

          await Session.remove(session.id)
        },
      })
    })

    test("should use explicit type when object is provided", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: {
              type: "inline",
              value: "This looks like a path but is inline",
            },
          })

          expect(session.customPrompt?.type).toBe("inline")
          expect(session.customPrompt?.value).toBe("This looks like a path but is inline")

          await Session.remove(session.id)
        },
      })
    })

    test("should preserve custom variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: {
              type: "inline",
              value: "Project: ${CUSTOM_NAME}",
              variables: { CUSTOM_NAME: "MyProject" },
            },
          })

          expect(session.customPrompt?.variables).toEqual({ CUSTOM_NAME: "MyProject" })

          await Session.remove(session.id)
        },
      })
    })
  })

  describe("interpolateVariables", () => {
    test("should interpolate built-in variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Project: ${PROJECT_NAME}, Platform: ${PLATFORM}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toContain("Project: opencode")
          expect(result).toContain(`Platform: ${process.platform}`)

          await Session.remove(session.id)
        },
      })
    })

    test("should interpolate date/time variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Date: ${DATE}, Time: ${TIME}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          // Date should be in YYYY-MM-DD format
          expect(result).toMatch(/Date: \d{4}-\d{2}-\d{2}/)
          // Time should be in HH:MM:SS format
          expect(result).toMatch(/Time: \d{2}:\d{2}:\d{2}/)

          await Session.remove(session.id)
        },
      })
    })

    test("should interpolate session variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({ title: "Test Session" })
          const template = "Session: ${SESSION_ID}, Title: ${SESSION_TITLE}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toContain(`Session: ${session.id}`)
          expect(result).toContain("Title: Test Session")

          await Session.remove(session.id)
        },
      })
    })

    test("should interpolate model variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Model: ${MODEL_ID}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-sonnet" },
          })

          expect(result).toBe("Model: claude-3-sonnet")

          await Session.remove(session.id)
        },
      })
    })

    test("should interpolate agent name", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Agent: ${AGENT_NAME}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            agent: { name: "CodeReview" } as any,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe("Agent: CodeReview")

          await Session.remove(session.id)
        },
      })
    })

    test("should use default value when variable not found", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Value: ${UNKNOWN_VAR:default_value}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe("Value: default_value")

          await Session.remove(session.id)
        },
      })
    })

    test("should keep original placeholder when variable not found and no default", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Value: ${COMPLETELY_UNKNOWN}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe("Value: ${COMPLETELY_UNKNOWN}")

          await Session.remove(session.id)
        },
      })
    })

    test("should apply uppercase filter", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Platform: ${PLATFORM|uppercase}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe(`Platform: ${process.platform.toUpperCase()}`)

          await Session.remove(session.id)
        },
      })
    })

    test("should apply lowercase filter", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Name: ${PROJECT_NAME|lowercase}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe("Name: opencode")

          await Session.remove(session.id)
        },
      })
    })

    test("should apply capitalize filter", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Platform: ${PLATFORM|capitalize}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          const expected = process.platform.charAt(0).toUpperCase() + process.platform.slice(1).toLowerCase()
          expect(result).toBe(`Platform: ${expected}`)

          await Session.remove(session.id)
        },
      })
    })

    test("should use custom variables passed inline", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "Custom: ${MY_VAR}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
            customVars: { MY_VAR: "inline_value" },
          })

          expect(result).toBe("Custom: inline_value")

          await Session.remove(session.id)
        },
      })
    })

    test("should prioritize inline vars over session vars", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: {
              type: "inline",
              value: "test",
              variables: { MY_VAR: "session_value" },
            },
          })
          const template = "Value: ${MY_VAR}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
            customVars: { MY_VAR: "inline_value" },
          })

          expect(result).toBe("Value: inline_value")

          await Session.remove(session.id)
        },
      })
    })

    test("should handle complex template with multiple variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({ title: "Complex Test" })
          // Use escaped $ to create literal ${...} in the template
          const template = [
            "You are working on ${PROJECT_NAME} (${PRIMARY_LANGUAGE|capitalize}).",
            "Session: ${SESSION_TITLE}",
            "Model: ${MODEL_ID}",
            "Date: ${DATE}",
            "Custom: ${TEAM:Engineering}",
          ].join("\n")

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-5-sonnet" },
          })

          expect(result).toContain("opencode")
          expect(result).toContain("Complex Test")
          expect(result).toContain("claude-3-5-sonnet")
          expect(result).toContain("Engineering") // default value
          expect(result).toMatch(/\d{4}-\d{2}-\d{2}/) // date

          await Session.remove(session.id)
        },
      })
    })
  })

  describe("fromSession", () => {
    test("should return null when no custom prompt", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})

          const result = await SystemPrompt.fromSession(session.id, {
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBeNull()

          await Session.remove(session.id)
        },
      })
    })

    test("should return interpolated inline prompt", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: {
              type: "inline",
              value: "You are working on ${PROJECT_NAME}. Model: ${MODEL_ID}",
            },
          })

          const result = await SystemPrompt.fromSession(session.id, {
            model: { providerID: "anthropic", modelID: "gpt-4" },
          })

          expect(result).toBe("You are working on opencode. Model: gpt-4")

          await Session.remove(session.id)
        },
      })
    })

    test("should load and interpolate file prompt", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          // Create a temporary prompt file
          const tempDir = os.tmpdir()
          const promptFile = path.join(tempDir, "test-prompt.txt")
          fs.writeFileSync(promptFile, "Project ${PROJECT_NAME} on ${PLATFORM}")

          try {
            const session = await Session.create({
              customPrompt: {
                type: "file",
                value: promptFile,
              },
            })

            const result = await SystemPrompt.fromSession(session.id, {
              model: { providerID: "anthropic", modelID: "claude-3-opus" },
            })

            expect(result).toBe(`Project opencode on ${process.platform}`)

            await Session.remove(session.id)
          } finally {
            fs.unlinkSync(promptFile)
          }
        },
      })
    })

    test("should throw error for non-existent file", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: {
              type: "file",
              value: "/non/existent/path/prompt.txt",
            },
          })

          await expect(
            SystemPrompt.fromSession(session.id, {
              model: { providerID: "anthropic", modelID: "claude-3-opus" },
            }),
          ).rejects.toThrow("Failed to load prompt template")

          await Session.remove(session.id)
        },
      })
    })

    test("should respect file size limit", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          // Create a large file (> 100KB)
          const tempDir = os.tmpdir()
          const largeFile = path.join(tempDir, "large-prompt.txt")
          const largeContent = "x".repeat(101 * 1024) // 101 KB
          fs.writeFileSync(largeFile, largeContent)

          try {
            const session = await Session.create({
              customPrompt: {
                type: "file",
                value: largeFile,
              },
            })

            await expect(
              SystemPrompt.fromSession(session.id, {
                model: { providerID: "anthropic", modelID: "claude-3-opus" },
              }),
            ).rejects.toThrow("too large")

            await Session.remove(session.id)
          } finally {
            fs.unlinkSync(largeFile)
          }
        },
      })
    })

    test("should use session custom variables in interpolation", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({
            customPrompt: {
              type: "inline",
              value: "Team: ${TEAM}, Focus: ${FOCUS}",
              variables: { TEAM: "Platform", FOCUS: "Performance" },
            },
          })

          const result = await SystemPrompt.fromSession(session.id, {
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe("Team: Platform, Focus: Performance")

          await Session.remove(session.id)
        },
      })
    })
  })

  describe("Environment variable extraction (OPENCODE_VAR_*)", () => {
    const originalEnv = process.env

    beforeEach(() => {
      // Clear any existing OPENCODE_VAR_ variables
      for (const key of Object.keys(process.env)) {
        if (key.startsWith("OPENCODE_VAR_")) {
          delete process.env[key]
        }
      }
    })

    afterEach(() => {
      // Restore original env
      for (const key of Object.keys(process.env)) {
        if (key.startsWith("OPENCODE_VAR_")) {
          delete process.env[key]
        }
      }
    })

    test("should extract OPENCODE_VAR_ environment variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          process.env.OPENCODE_VAR_TEAM = "Backend"
          process.env.OPENCODE_VAR_ENV = "Production"

          const session = await Session.create({})
          const template = "Team: ${TEAM}, Env: ${ENV}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe("Team: Backend, Env: Production")

          await Session.remove(session.id)
        },
      })
    })
  })

  describe("Edge cases", () => {
    test("should handle empty template", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})

          const result = await SystemPrompt.interpolateVariables("", {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe("")

          await Session.remove(session.id)
        },
      })
    })

    test("should handle template with no variables", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          const template = "This is a plain text template with no variables."

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          expect(result).toBe(template)

          await Session.remove(session.id)
        },
      })
    })

    test("should handle malformed variable syntax gracefully", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          const session = await Session.create({})
          // These should not be matched by the regex
          const template = "Invalid: ${} ${ } ${123} ${lowercase}"

          const result = await SystemPrompt.interpolateVariables(template, {
            sessionID: session.id,
            model: { providerID: "anthropic", modelID: "claude-3-opus" },
          })

          // Should remain unchanged since they don't match the pattern
          expect(result).toBe(template)

          await Session.remove(session.id)
        },
      })
    })

    test("should handle single-line prompt as file (auto-detect edge case)", async () => {
      await Instance.provide({
        directory: projectRoot,
        fn: async () => {
          // Single line without path indicators or extensions - treated as file
          const session = await Session.create({
            customPrompt: "simple-prompt-name",
          })

          // Single line = treated as file due to auto-detection heuristic
          expect(session.customPrompt?.type).toBe("file")

          await Session.remove(session.id)
        },
      })
    })
  })
})
