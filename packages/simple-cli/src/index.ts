#!/usr/bin/env bun
import { resolveConfig } from "./config"
import { Renderer } from "./renderer"
import { createSimpleClient } from "./client"
import { runRepl } from "./repl"

async function main() {
  try {
    const config = resolveConfig()
    const renderer = new Renderer({
      noColor: config.noColor,
      quiet: config.quiet,
      json: config.json,
      verbose: config.verbose,
    })

    const client = await createSimpleClient(config, renderer)
    await runRepl(config, client, renderer)
  } catch (err) {
    console.error("simple-cli error", err)
    process.exitCode = 1
  }
}

main()
