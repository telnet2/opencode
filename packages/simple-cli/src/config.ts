import fs from "fs"
import os from "os"
import path from "path"
import type { CliOptions, ResolvedConfig } from "./types"

function parseArgs(argv: string[]): CliOptions {
  const options: CliOptions = {}
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]
    const next = () => argv[i + 1]
    switch (arg) {
      case "--url":
        options.url = next()
        i++
        break
      case "--api-key":
        options.apiKey = next()
        i++
        break
      case "--model":
        options.model = next()
        i++
        break
      case "--provider":
        options.provider = next()
        i++
        break
      case "--agent":
        options.agent = next()
        i++
        break
      case "--session":
        options.session = next()
        i++
        break
      case "--directory":
        options.directory = next()
        i++
        break
      case "--quiet":
        options.quiet = true
        break
      case "--verbose":
        options.verbose = true
        break
      case "--json":
        options.json = true
        break
      case "--no-color":
        options.noColor = true
        break
      case "--trace":
        options.trace = true
        break
      default:
        break
    }
  }
  return options
}

function loadConfigFile(cwd: string): Partial<CliOptions> {
  const homeConfig = path.join(os.homedir(), ".opencode", "simple-cli.json")
  const projectConfig = path.join(cwd, ".opencode", "simple-cli.json")

  const configs: Array<Partial<CliOptions>> = []
  if (fs.existsSync(homeConfig)) {
    configs.push(JSON.parse(fs.readFileSync(homeConfig, "utf8")))
  }
  if (fs.existsSync(projectConfig)) {
    configs.push(JSON.parse(fs.readFileSync(projectConfig, "utf8")))
  }

  return Object.assign({}, ...configs)
}

export function resolveConfig(argv = process.argv.slice(2)): ResolvedConfig {
  const cwd = process.cwd()
  const cli = parseArgs(argv)
  const fileConfig = loadConfigFile(cwd)
  const url = cli.url ?? fileConfig.url ?? process.env.OPENCODE_SERVER_URL
  if (!url) {
    throw new Error("Missing server URL. Provide --url or set OPENCODE_SERVER_URL.")
  }

  const apiKey = cli.apiKey ?? fileConfig.apiKey ?? process.env.OPENCODE_API_KEY

  return {
    ...fileConfig,
    ...cli,
    url,
    apiKey,
    sessionFile: path.join(os.homedir(), ".opencode", "simple-cli-state.json"),
  }
}
