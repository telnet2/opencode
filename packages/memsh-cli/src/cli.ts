#!/usr/bin/env bun
/**
 * memsh-cli - CLI for interacting with go-memsh service
 *
 * Usage:
 *   memsh-cli --server http://localhost:8080
 *   memsh-cli --server http://localhost:8080 --command "ls -la"
 *   memsh-cli --server http://localhost:8080 --session <session-id>
 */

import { createMemshEnvironment, createSession } from "./index"

interface CliArgs {
  server: string
  command?: string
  session?: string
  interactive?: boolean
  help?: boolean
}

function parseArgs(): CliArgs {
  const args: CliArgs = {
    server: process.env.MEMSH_SERVER ?? "http://localhost:8080",
  }

  for (let i = 2; i < process.argv.length; i++) {
    const arg = process.argv[i]
    const next = process.argv[i + 1]

    switch (arg) {
      case "--server":
      case "-s":
        if (next) {
          args.server = next
          i++
        }
        break
      case "--command":
      case "-c":
        args.command = next
        i++
        break
      case "--session":
        args.session = next
        i++
        break
      case "--interactive":
      case "-i":
        args.interactive = true
        break
      case "--help":
      case "-h":
        args.help = true
        break
    }
  }

  return args
}

function printHelp(): void {
  console.log(`
memsh-cli - TypeScript client for go-memsh service

Usage:
  memsh-cli [options]

Options:
  --server, -s <url>     Server URL (default: http://localhost:8080)
  --command, -c <cmd>    Execute a single command and exit
  --session <id>         Connect to existing session
  --interactive, -i      Start interactive REPL mode
  --help, -h             Show this help message

Environment Variables:
  MEMSH_SERVER          Default server URL

Examples:
  # Execute a single command
  memsh-cli -s http://localhost:8080 -c "ls -la"

  # Start interactive session
  memsh-cli -s http://localhost:8080 -i

  # Connect to existing session
  memsh-cli -s http://localhost:8080 --session abc123 -c "pwd"
`)
}

async function runInteractive(env: Awaited<ReturnType<typeof createMemshEnvironment>>): Promise<void> {
  const readline = await import("readline")
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  })

  console.log(`Connected to session: ${env.session.id}`)
  console.log(`Working directory: ${env.session.cwd}`)
  console.log('Type "exit" to quit.\n')

  const prompt = () => {
    rl.question(`${env.session.cwd}$ `, async (input: string) => {
      const cmd = input.trim()

      if (cmd === "exit" || cmd === "quit") {
        rl.close()
        await env.close()
        process.exit(0)
      }

      if (!cmd) {
        prompt()
        return
      }

      try {
        const result = await env.session.execute(cmd)
        if (result.output.length > 0) {
          console.log(result.output.join("\n"))
        }
        if (result.error) {
          console.error(`Error: ${result.error}`)
        }
      } catch (error) {
        console.error(`Error: ${error instanceof Error ? error.message : String(error)}`)
      }

      prompt()
    })
  }

  prompt()
}

async function main(): Promise<void> {
  const args = parseArgs()

  if (args.help) {
    printHelp()
    process.exit(0)
  }

  try {
    if (args.command && !args.interactive) {
      // Single command mode
      const session = await createSession({
        baseUrl: args.server,
        sessionId: args.session,
      })

      const result = await session.execute(args.command)

      if (result.output.length > 0) {
        console.log(result.output.join("\n"))
      }

      if (result.error) {
        console.error(`Error: ${result.error}`)
        process.exit(1)
      }

      await session.close(!args.session) // Remove session if we created it
    } else if (args.interactive || !args.command) {
      // Interactive mode
      const env = await createMemshEnvironment({
        baseUrl: args.server,
        sessionId: args.session,
      })

      await runInteractive(env)
    }
  } catch (error) {
    console.error(`Error: ${error instanceof Error ? error.message : String(error)}`)
    process.exit(1)
  }
}

main()
