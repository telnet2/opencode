import { spawn } from "node:child_process"
import * as fs from "node:fs"
import * as path from "node:path"

let rgPath: string | null = null
let checkedRg = false

/**
 * Find ripgrep binary, returns null if not available
 */
async function findRg(): Promise<string | null> {
  if (checkedRg) return rgPath

  checkedRg = true

  // Check common locations
  const candidates = ["rg", "/usr/bin/rg", "/usr/local/bin/rg"]

  for (const candidate of candidates) {
    try {
      const result = await execCommand(candidate, ["--version"])
      if (result.exitCode === 0) {
        rgPath = candidate
        return rgPath
      }
    } catch {
      // Not found, try next
    }
  }

  return null
}

/**
 * Execute a command and return output
 */
function execCommand(
  cmd: string,
  args: string[],
  options?: { cwd?: string; maxBuffer?: number }
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve, reject) => {
    const proc = spawn(cmd, args, {
      cwd: options?.cwd,
      stdio: ["ignore", "pipe", "pipe"],
    })

    let stdout = ""
    let stderr = ""
    const maxBuffer = options?.maxBuffer ?? 1024 * 1024 * 10 // 10MB

    proc.stdout.on("data", (data) => {
      if (stdout.length < maxBuffer) {
        stdout += data.toString()
      }
    })

    proc.stderr.on("data", (data) => {
      if (stderr.length < maxBuffer) {
        stderr += data.toString()
      }
    })

    proc.on("error", reject)
    proc.on("close", (code) => {
      resolve({ stdout, stderr, exitCode: code ?? 0 })
    })
  })
}

/**
 * List files matching glob patterns using ripgrep or find
 */
export async function* listFiles(options: {
  cwd: string
  glob?: string[]
}): AsyncGenerator<string> {
  const rg = await findRg()

  if (rg) {
    // Use ripgrep
    const args = ["--files", "--follow", "--hidden", "--glob=!.git/*"]
    if (options.glob) {
      for (const g of options.glob) {
        args.push(`--glob=${g}`)
      }
    }

    const result = await execCommand(rg, args, { cwd: options.cwd })
    if (result.exitCode === 0 || result.exitCode === 1) {
      // exitCode 1 means no matches, which is ok
      const lines = result.stdout.trim().split("\n").filter(Boolean)
      for (const line of lines) {
        yield line
      }
    }
  } else {
    // Fallback to find command
    const args = [".", "-type", "f", "-not", "-path", "*/.git/*"]
    if (options.glob && options.glob.length > 0) {
      // Convert glob patterns to find -name patterns
      for (const g of options.glob) {
        if (g.startsWith("!")) {
          // Exclusion pattern
          const pattern = g.slice(1)
          args.push("-not", "-path", `*/${pattern}`)
        } else {
          args.push("-name", g.replace("**/*", "*"))
        }
      }
    }

    const result = await execCommand("find", args, { cwd: options.cwd })
    if (result.exitCode === 0) {
      const lines = result.stdout
        .trim()
        .split("\n")
        .filter(Boolean)
        .map((line) => line.replace(/^\.\//, ""))
      for (const line of lines) {
        yield line
      }
    }
  }
}

/**
 * Search file contents using ripgrep or grep
 */
export async function searchContent(options: {
  cwd: string
  pattern: string
  glob?: string
}): Promise<
  Array<{
    path: string
    lineNum: number
    lineText: string
  }>
> {
  const rg = await findRg()

  if (rg) {
    // Use ripgrep
    const args = ["-nH", "--field-match-separator=|", "--regexp", options.pattern]
    if (options.glob) {
      args.push("--glob", options.glob)
    }
    args.push(options.cwd)

    const result = await execCommand(rg, args)

    if (result.exitCode === 1) {
      // No matches
      return []
    }

    if (result.exitCode !== 0) {
      throw new Error(`ripgrep failed: ${result.stderr}`)
    }

    return parseSearchOutput(result.stdout, "|")
  } else {
    // Fallback to grep
    const args = ["-rnH", "--include=" + (options.glob || "*"), options.pattern, options.cwd]

    const result = await execCommand("grep", args)

    if (result.exitCode === 1) {
      // No matches
      return []
    }

    if (result.exitCode !== 0) {
      throw new Error(`grep failed: ${result.stderr}`)
    }

    return parseSearchOutput(result.stdout, ":")
  }
}

/**
 * Parse search output (works for both rg and grep)
 */
function parseSearchOutput(
  output: string,
  separator: string
): Array<{ path: string; lineNum: number; lineText: string }> {
  const lines = output.trim().split("\n")
  const matches: Array<{ path: string; lineNum: number; lineText: string }> = []

  for (const line of lines) {
    if (!line) continue

    const parts = line.split(separator)
    if (parts.length < 3) continue

    const filePath = parts[0]
    const lineNumStr = parts[1]
    const lineText = parts.slice(2).join(separator)

    const lineNum = parseInt(lineNumStr, 10)
    if (isNaN(lineNum)) continue

    matches.push({
      path: filePath,
      lineNum,
      lineText,
    })
  }

  return matches
}

/**
 * Check if ripgrep is available
 */
export async function hasRipgrep(): Promise<boolean> {
  return (await findRg()) !== null
}
