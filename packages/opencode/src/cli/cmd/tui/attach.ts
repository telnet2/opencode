import { cmd } from "../cmd"
import { tui } from "./app"
import { Log } from "@/util/log"

export const AttachCommand = cmd({
  command: "attach <url>",
  describe: "attach to a running opencode server",
  builder: (yargs) =>
    yargs
      .positional("url", {
        type: "string",
        describe: "http://localhost:4096",
        demandOption: true,
      })
      .option("dir", {
        type: "string",
        description: "directory to run in",
      }),
  handler: async (args) => {
    // Capture console.log/error/warn/debug to log file in TUI mode
    Log.captureConsole()

    if (args.dir) process.chdir(args.dir)
    await tui({
      url: args.url,
      args: {},
    })
  },
})
