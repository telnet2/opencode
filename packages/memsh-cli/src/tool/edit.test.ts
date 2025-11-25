import { describe, test, expect } from "bun:test"
import { EditTool } from "./edit"

describe("EditTool", () => {
  test("should have correct id", () => {
    expect(EditTool.id).toBe("edit")
  })

  test("should initialize with description and parameters", async () => {
    const initialized = await EditTool.init()

    expect(initialized.description).toBeDefined()
    expect(initialized.parameters).toBeDefined()
  })

  test("should validate that oldString and newString are different", async () => {
    const initialized = await EditTool.init()

    // Mock session
    const mockSession = {
      exists: async () => true,
      isDirectory: async () => false,
      readFile: async () => "original content",
      writeFile: async () => {},
    }

    const ctx = {
      session: mockSession as any,
      abort: new AbortController().signal,
      metadata: () => {},
    }

    await expect(
      initialized.execute(
        {
          filePath: "/test.txt",
          oldString: "same",
          newString: "same",
        },
        ctx,
      ),
    ).rejects.toThrow("oldString and newString must be different")
  })

  test("should throw error when file not found", async () => {
    const initialized = await EditTool.init()

    const mockSession = {
      exists: async () => false,
    }

    const ctx = {
      session: mockSession as any,
      abort: new AbortController().signal,
      metadata: () => {},
    }

    await expect(
      initialized.execute(
        {
          filePath: "/nonexistent.txt",
          oldString: "old",
          newString: "new",
        },
        ctx,
      ),
    ).rejects.toThrow("File not found: /nonexistent.txt")
  })

  test("should throw error when oldString not found in content", async () => {
    const initialized = await EditTool.init()

    const mockSession = {
      exists: async () => true,
      isDirectory: async () => false,
      readFile: async () => "file content without the search string",
    }

    const ctx = {
      session: mockSession as any,
      abort: new AbortController().signal,
      metadata: () => {},
    }

    await expect(
      initialized.execute(
        {
          filePath: "/test.txt",
          oldString: "not found",
          newString: "replacement",
        },
        ctx,
      ),
    ).rejects.toThrow("oldString not found in content")
  })

  test("should detect multiple matches", async () => {
    const initialized = await EditTool.init()

    const mockSession = {
      exists: async () => true,
      isDirectory: async () => false,
      readFile: async () => "hello world hello world",
    }

    const ctx = {
      session: mockSession as any,
      abort: new AbortController().signal,
      metadata: () => {},
    }

    await expect(
      initialized.execute(
        {
          filePath: "/test.txt",
          oldString: "hello",
          newString: "hi",
        },
        ctx,
      ),
    ).rejects.toThrow("Found multiple matches")
  })

  test("should allow replaceAll for multiple matches", async () => {
    const initialized = await EditTool.init()

    let writtenContent = ""
    const mockSession = {
      exists: async () => true,
      isDirectory: async () => false,
      readFile: async () => "hello world hello world",
      writeFile: async (_path: string, content: string) => {
        writtenContent = content
      },
    }

    const ctx = {
      session: mockSession as any,
      abort: new AbortController().signal,
      metadata: () => {},
    }

    const result = await initialized.execute(
      {
        filePath: "/test.txt",
        oldString: "hello",
        newString: "hi",
        replaceAll: true,
      },
      ctx,
    )

    expect(writtenContent).toBe("hi world hi world")
    expect(result.metadata.additions).toBeGreaterThanOrEqual(0)
  })
})
