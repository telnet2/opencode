import { z } from "zod"
import { spawn } from "node:child_process"
import { defineTool } from "./tool.js"

const DEFAULT_MAX_OUTPUT_LENGTH = 30_000
const MAX_OUTPUT_LENGTH = (() => {
  const parsed = Number(process.env.KIANA_BASH_MAX_OUTPUT_LENGTH)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : DEFAULT_MAX_OUTPUT_LENGTH
})()
const DEFAULT_TIMEOUT = 2 * 60 * 1000 // 2 minutes
const MAX_TIMEOUT = 10 * 60 * 1000 // 10 minutes
const SIGKILL_TIMEOUT_MS = 200

const DESCRIPTION = `Executes a given bash command in a persistent shell session with optional timeout, ensuring proper handling and security measures.

Before executing the command, please follow these steps:

1. Directory Verification:
   - If the command will create new directories or files, first use the List tool to verify the parent directory exists and is the correct location
   - For example, before running "mkdir foo/bar", first use List to check that "foo" exists and is the intended parent directory

2. Command Execution:
   - Always quote file paths that contain spaces with double quotes (e.g., cd "path with spaces/file.txt")
   - Examples of proper quoting:
     - cd "/Users/name/My Documents" (correct)
     - cd /Users/name/My Documents (incorrect - will fail)
     - python "/path/with spaces/script.py" (correct)
     - python /path/with spaces/script.py (incorrect - will fail)
   - After ensuring proper quoting, execute the command.
   - Capture the output of the command.

Usage notes:
  - The command argument is required.
  - You can specify an optional timeout in milliseconds (up to 600000ms / 10 minutes). If not specified, commands will timeout after 120000ms (2 minutes).
  - It is very helpful if you write a clear, concise description of what this command does in 5-10 words.
  - If the output exceeds 30000 characters, output will be truncated before being returned to you.
  - VERY IMPORTANT: You MUST avoid using search commands like \`find\` and \`grep\`. Instead use Grep, Glob, or Task to search. You MUST avoid read tools like \`cat\`, \`head\`, \`tail\`, and \`ls\`, and use Read and List to read files.
  - If you _still_ need to run \`grep\`, STOP. ALWAYS USE ripgrep at \`rg\` (or /usr/bin/rg) first, which all users have pre-installed.
  - When issuing multiple commands, use the ';' or '&&' operator to separate them. DO NOT use newlines (newlines are ok in quoted strings).
  - Try to maintain your current working directory throughout the session by using absolute paths and avoiding usage of \`cd\`. You may use \`cd\` if the User explicitly requests it.
    <good-example>
    pytest /foo/bar/tests
    </good-example>
    <bad-example>
    cd /foo/bar && pytest tests
    </bad-example>


# Committing changes with git

IMPORTANT: ONLY COMMIT IF THE USER ASKS YOU TO.

If and only if the user asks you to create a new git commit, follow these steps carefully:

1. You have the capability to call multiple tools in a single response. When multiple independent pieces of information are requested, batch your tool calls together for optimal performance. ALWAYS run the following bash commands in parallel, each using the Bash tool:
   - Run a git status command to see all untracked files.
   - Run a git diff command to see both staged and unstaged changes that will be committed.
   - Run a git log command to see recent commit messages, so that you can follow this repository's commit message style.

2. Analyze all staged changes (both previously staged and newly added) and draft a commit message.

3. Create the commit with a meaningful message.

Important notes:
- Use the git context at the start of this conversation to determine which files are relevant to your commit.
- NEVER update the git config
- DO NOT push to the remote repository
- IMPORTANT: Never use git commands with the -i flag (like git rebase -i or git add -i) since they require interactive input which is not supported.
- If there are no changes to commit (i.e., no untracked files and no modifications), do not create an empty commit

# Creating pull requests
Use the gh command via the Bash tool for ALL GitHub-related tasks including working with issues, pull requests, checks, and releases.`

export const bashTool = defineTool("bash", {
  description: DESCRIPTION,
  parameters: z.object({
    command: z.string().describe("The command to execute"),
    timeout: z.number().describe("Optional timeout in milliseconds").optional(),
    description: z
      .string()
      .describe(
        "Clear, concise description of what this command does in 5-10 words. Examples:\nInput: ls\nOutput: Lists files in current directory\n\nInput: git status\nOutput: Shows working tree status\n\nInput: npm install\nOutput: Installs package dependencies\n\nInput: mkdir foo\nOutput: Creates directory 'foo'"
      ),
  }),
  async execute(params, ctx) {
    if (params.timeout !== undefined && params.timeout < 0) {
      throw new Error(`Invalid timeout value: ${params.timeout}. Timeout must be a positive number.`)
    }
    const timeout = Math.min(params.timeout ?? DEFAULT_TIMEOUT, MAX_TIMEOUT)

    // Determine shell to use
    const shell = (() => {
      const s = process.env.SHELL
      if (s) {
        // Avoid fish and nu shells
        if (!new Set(["/bin/fish", "/bin/nu", "/usr/bin/fish", "/usr/bin/nu"]).has(s)) {
          return s
        }
      }

      if (process.platform === "darwin") {
        return "/bin/zsh"
      }

      if (process.platform === "win32") {
        return process.env.COMSPEC || "cmd.exe"
      }

      // Try to find bash
      return "/bin/bash"
    })()

    const proc = spawn(params.command, {
      shell,
      cwd: ctx.workingDirectory,
      env: {
        ...process.env,
      },
      stdio: ["ignore", "pipe", "pipe"],
      detached: process.platform !== "win32",
    })

    let output = ""

    // Initialize metadata with empty output
    ctx.metadata({
      metadata: {
        output: "",
        description: params.description,
      },
    })

    const append = (chunk: Buffer) => {
      if (output.length <= MAX_OUTPUT_LENGTH) {
        output += chunk.toString()
        ctx.metadata({
          metadata: {
            output,
            description: params.description,
          },
        })
      }
    }

    proc.stdout?.on("data", append)
    proc.stderr?.on("data", append)

    let timedOut = false
    let aborted = false
    let exited = false

    const killTree = async () => {
      const pid = proc.pid
      if (!pid || exited) {
        return
      }

      if (process.platform === "win32") {
        await new Promise<void>((resolve) => {
          const killer = spawn("taskkill", ["/pid", String(pid), "/f", "/t"], { stdio: "ignore" })
          killer.once("exit", resolve)
          killer.once("error", () => resolve())
        })
        return
      }

      try {
        process.kill(-pid, "SIGTERM")
        await sleep(SIGKILL_TIMEOUT_MS)
        if (!exited) {
          process.kill(-pid, "SIGKILL")
        }
      } catch {
        proc.kill("SIGTERM")
        await sleep(SIGKILL_TIMEOUT_MS)
        if (!exited) {
          proc.kill("SIGKILL")
        }
      }
    }

    if (ctx.abort.aborted) {
      aborted = true
      await killTree()
    }

    const abortHandler = () => {
      aborted = true
      void killTree()
    }

    ctx.abort.addEventListener("abort", abortHandler, { once: true })

    const timeoutTimer = setTimeout(() => {
      timedOut = true
      void killTree()
    }, timeout + 100)

    await new Promise<void>((resolve, reject) => {
      const cleanup = () => {
        clearTimeout(timeoutTimer)
        ctx.abort.removeEventListener("abort", abortHandler)
      }

      proc.once("exit", () => {
        exited = true
        cleanup()
        resolve()
      })

      proc.once("error", (error) => {
        exited = true
        cleanup()
        reject(error)
      })
    })

    const resultMetadata: string[] = ["<bash_metadata>"]

    if (output.length > MAX_OUTPUT_LENGTH) {
      output = output.slice(0, MAX_OUTPUT_LENGTH)
      resultMetadata.push(`bash tool truncated output as it exceeded ${MAX_OUTPUT_LENGTH} char limit`)
    }

    if (timedOut) {
      resultMetadata.push(`bash tool terminated command after exceeding timeout ${timeout} ms`)
    }

    if (aborted) {
      resultMetadata.push("User aborted the command")
    }

    if (resultMetadata.length > 1) {
      resultMetadata.push("</bash_metadata>")
      output += "\n\n" + resultMetadata.join("\n")
    }

    return {
      title: params.description,
      metadata: {
        output,
        exit: proc.exitCode,
        description: params.description,
      },
      output,
    }
  },
})

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
