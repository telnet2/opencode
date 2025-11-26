import { describe, test, expect } from "bun:test"
import { z } from "zod"
import { Tool, ToolRegistry } from "./tool"

describe("Tool", () => {
  describe("define", () => {
    test("should create a tool with id", () => {
      const tool = Tool.define("test-tool", {
        description: "A test tool",
        parameters: z.object({
          name: z.string(),
        }),
        async execute(params, _ctx) {
          return {
            title: "Test",
            metadata: { name: params.name },
            output: `Hello, ${params.name}!`,
          }
        },
      })

      expect(tool.id).toBe("test-tool")
    })

    test("should initialize tool with init function", async () => {
      const tool = Tool.define("test-tool", async () => ({
        description: "A test tool",
        parameters: z.object({
          value: z.number(),
        }),
        async execute(params, _ctx) {
          return {
            title: "Test",
            metadata: { doubled: params.value * 2 },
            output: String(params.value * 2),
          }
        },
      }))

      const initialized = await tool.init()
      expect(initialized.description).toBe("A test tool")
    })
  })
})

describe("ToolRegistry", () => {
  test("should register and retrieve tools", () => {
    const registry = new ToolRegistry()

    const tool = Tool.define("my-tool", {
      description: "My tool",
      parameters: z.object({}),
      async execute(_params, _ctx) {
        return {
          title: "My Tool",
          metadata: {},
          output: "done",
        }
      },
    })

    registry.register(tool)

    expect(registry.has("my-tool")).toBe(true)
    expect(registry.get("my-tool")).toBe(tool)
    expect(registry.list()).toContain("my-tool")
  })

  test("should register multiple tools", () => {
    const registry = new ToolRegistry()

    const tool1 = Tool.define("tool-1", {
      description: "Tool 1",
      parameters: z.object({}),
      async execute(_params, _ctx) {
        return { title: "1", metadata: {}, output: "1" }
      },
    })

    const tool2 = Tool.define("tool-2", {
      description: "Tool 2",
      parameters: z.object({}),
      async execute(_params, _ctx) {
        return { title: "2", metadata: {}, output: "2" }
      },
    })

    registry.registerAll(tool1, tool2)

    expect(registry.has("tool-1")).toBe(true)
    expect(registry.has("tool-2")).toBe(true)
    expect(registry.list().length).toBe(2)
  })

  test("should return undefined for unknown tools", () => {
    const registry = new ToolRegistry()

    expect(registry.get("unknown")).toBeUndefined()
    expect(registry.has("unknown")).toBe(false)
  })

  test("should get initialized tool", async () => {
    const registry = new ToolRegistry()

    const tool = Tool.define("async-tool", async () => ({
      description: "Async tool",
      parameters: z.object({ input: z.string() }),
      async execute(params, _ctx) {
        return {
          title: "Async",
          metadata: { input: params.input },
          output: params.input.toUpperCase(),
        }
      },
    }))

    registry.register(tool)

    const initialized = await registry.getInitialized("async-tool")
    expect(initialized).toBeDefined()
    expect(initialized?.description).toBe("Async tool")

    // Second call should return cached version
    const cached = await registry.getInitialized("async-tool")
    expect(cached).toBe(initialized)
  })
})
