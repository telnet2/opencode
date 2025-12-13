/**
 * Interactive mode input handling for kiana-v6 CLI.
 *
 * Features:
 * - Multi-line input with Enter to send, Ctrl+J for newline
 * - ESC double-tap to abort (with 2s timeout)
 * - Simple prompt: `>` first line, `»` continuation
 */

import * as readline from "node:readline"
import { EventEmitter } from "node:events"

// ANSI codes
const ANSI = {
  clearLine: "\x1b[2K",
  cursorLeft: "\x1b[G",
  cursorUp: "\x1b[A",
  cursorDown: "\x1b[B",
  reset: "\x1b[0m",
  dim: "\x1b[2m",
  cyan: "\x1b[36m",
  yellow: "\x1b[33m",
}

export type InteractiveMode = "idle" | "streaming" | "esc_pending" | "aborting"

export interface InteractiveEvents {
  submit: (message: string) => void
  abort: () => void
  exit: () => void
}

export class InteractiveInput extends EventEmitter {
  private mode: InteractiveMode = "idle"
  private inputLines: string[] = [""]
  private cursorLine: number = 0
  private cursorCol: number = 0
  private escTimeout?: NodeJS.Timeout
  private rl?: readline.Interface

  private readonly ESC_TIMEOUT_MS = 2000
  private readonly PROMPT_FIRST = "> "
  private readonly PROMPT_CONT = "» "

  constructor() {
    super()
  }

  /**
   * Start interactive input in raw mode.
   */
  start(): void {
    if (!process.stdin.isTTY) {
      console.error("Interactive mode requires a TTY")
      process.exit(1)
    }

    process.stdin.setRawMode(true)
    process.stdin.resume()
    process.stdin.setEncoding("utf8")

    process.stdin.on("data", this.handleKeypress.bind(this))

    this.showPrompt()
  }

  /**
   * Stop interactive input and cleanup.
   */
  stop(): void {
    if (this.escTimeout) {
      clearTimeout(this.escTimeout)
      this.escTimeout = undefined
    }

    if (process.stdin.isTTY) {
      process.stdin.setRawMode(false)
    }
    process.stdin.pause()
  }

  /**
   * Set the current mode.
   */
  setMode(mode: InteractiveMode): void {
    this.mode = mode

    if (mode === "idle") {
      // Clear any pending ESC timeout
      if (this.escTimeout) {
        clearTimeout(this.escTimeout)
        this.escTimeout = undefined
      }
      // Reset input state
      this.inputLines = [""]
      this.cursorLine = 0
      this.cursorCol = 0
      // Show prompt
      this.showPrompt()
    }
  }

  /**
   * Get current mode.
   */
  getMode(): InteractiveMode {
    return this.mode
  }

  /**
   * Show the input prompt.
   */
  showPrompt(): void {
    process.stdout.write(this.PROMPT_FIRST)
  }

  /**
   * Show ESC warning message.
   */
  showEscWarning(): void {
    process.stdout.write(`\n${ANSI.yellow}Press ESC again to cancel...${ANSI.reset}`)
  }

  /**
   * Clear ESC warning message.
   */
  clearEscWarning(): void {
    // Move up one line and clear it
    process.stdout.write(`${ANSI.cursorUp}${ANSI.clearLine}${ANSI.cursorLeft}`)
  }

  private handleKeypress(key: string): void {
    // Handle based on current mode
    switch (this.mode) {
      case "idle":
        this.handleIdleKeypress(key)
        break
      case "streaming":
        this.handleStreamingKeypress(key)
        break
      case "esc_pending":
        this.handleEscPendingKeypress(key)
        break
      case "aborting":
        // Ignore input while aborting
        break
    }
  }

  private handleIdleKeypress(key: string): void {
    const code = key.charCodeAt(0)

    // Ctrl+C - exit
    if (key === "\x03") {
      process.stdout.write("\n")
      this.emit("exit")
      return
    }

    // Ctrl+D - exit if empty, otherwise ignore
    if (key === "\x04") {
      if (this.inputLines.length === 1 && this.inputLines[0] === "") {
        process.stdout.write("\n")
        this.emit("exit")
      }
      return
    }

    // Enter (CR) - submit message
    if (key === "\r" || key === "\n" && code === 13) {
      const message = this.inputLines.join("\n").trim()
      if (message) {
        process.stdout.write("\n")
        this.mode = "streaming"
        this.emit("submit", message)
      }
      return
    }

    // Ctrl+J (LF) - insert newline
    if (key === "\n" || code === 10) {
      this.inputLines.push("")
      this.cursorLine++
      this.cursorCol = 0
      process.stdout.write(`\n${this.PROMPT_CONT}`)
      return
    }

    // Backspace
    if (key === "\x7f" || key === "\b") {
      if (this.cursorCol > 0) {
        // Delete character in current line
        const line = this.inputLines[this.cursorLine]
        this.inputLines[this.cursorLine] =
          line.slice(0, this.cursorCol - 1) + line.slice(this.cursorCol)
        this.cursorCol--
        // Redraw current line
        this.redrawCurrentLine()
      } else if (this.cursorLine > 0) {
        // Merge with previous line
        const currentLine = this.inputLines[this.cursorLine]
        this.inputLines.splice(this.cursorLine, 1)
        this.cursorLine--
        this.cursorCol = this.inputLines[this.cursorLine].length
        this.inputLines[this.cursorLine] += currentLine
        // Move cursor up and redraw
        process.stdout.write(ANSI.cursorUp)
        this.redrawCurrentLine()
        // Clear the line below
        process.stdout.write(`\n${ANSI.clearLine}${ANSI.cursorUp}`)
        this.redrawCurrentLine()
      }
      return
    }

    // ESC - ignore in idle mode
    if (key === "\x1b") {
      return
    }

    // Arrow keys (escape sequences)
    if (key.startsWith("\x1b[")) {
      // For now, ignore arrow keys (could implement cursor movement later)
      return
    }

    // Regular character - insert at cursor
    if (key.length === 1 && code >= 32) {
      const line = this.inputLines[this.cursorLine]
      this.inputLines[this.cursorLine] =
        line.slice(0, this.cursorCol) + key + line.slice(this.cursorCol)
      this.cursorCol++
      process.stdout.write(key)
    }
  }

  private handleStreamingKeypress(key: string): void {
    // ESC - start abort sequence
    if (key === "\x1b") {
      this.mode = "esc_pending"
      this.showEscWarning()

      // Set timeout to cancel pending abort
      this.escTimeout = setTimeout(() => {
        if (this.mode === "esc_pending") {
          this.clearEscWarning()
          this.mode = "streaming"
        }
      }, this.ESC_TIMEOUT_MS)
      return
    }

    // Ctrl+C - immediate abort
    if (key === "\x03") {
      this.mode = "aborting"
      this.emit("abort")
      return
    }
  }

  private handleEscPendingKeypress(key: string): void {
    // Second ESC - confirm abort
    if (key === "\x1b") {
      if (this.escTimeout) {
        clearTimeout(this.escTimeout)
        this.escTimeout = undefined
      }
      this.clearEscWarning()
      this.mode = "aborting"
      process.stdout.write(`\n${ANSI.dim}Aborted${ANSI.reset}\n`)
      this.emit("abort")
      return
    }

    // Any other key - cancel pending abort
    if (this.escTimeout) {
      clearTimeout(this.escTimeout)
      this.escTimeout = undefined
    }
    this.clearEscWarning()
    this.mode = "streaming"
  }

  private redrawCurrentLine(): void {
    const prompt = this.cursorLine === 0 ? this.PROMPT_FIRST : this.PROMPT_CONT
    const line = this.inputLines[this.cursorLine]
    process.stdout.write(`${ANSI.clearLine}${ANSI.cursorLeft}${prompt}${line}`)
  }
}
