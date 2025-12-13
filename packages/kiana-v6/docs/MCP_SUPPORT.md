# Model Context Protocol (MCP) Support

Kiana-v6 now supports the Model Context Protocol (MCP), allowing you to connect to external MCP servers and use their tools seamlessly alongside Kiana's built-in tools.

## What is MCP?

The Model Context Protocol is a standard protocol for connecting AI applications to external data sources and tools. It allows you to:

- Connect to multiple MCP servers simultaneously
- Use tools from MCP servers as if they were native Kiana tools
- Integrate with the growing ecosystem of MCP servers

## Configuration

To use MCP servers, add an `mcpServers` array to your `kiana.jsonc` config file:

```jsonc
{
  "provider": {
    "type": "anthropic",
    "apiKey": "YOUR_API_KEY",
    "model": "claude-sonnet-4-20250514"
  },
  
  // MCP Servers configuration
  "mcpServers": [
    {
      "name": "filesystem",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/directory"],
      "env": {
        "SOME_VAR": "value"
      }
    },
    {
      "name": "github",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_your_token_here"
      }
    }
  ]
}
```

### Configuration Fields

Each MCP server configuration has the following fields:

- **name** (string, required): Unique identifier for the server
- **command** (string, required): The command to start the MCP server (e.g., `"node"`, `"npx"`)
- **args** (string[], required): Arguments to pass to the command
- **env** (object, optional): Environment variables to pass to the server process

## Tool Naming

Tools from MCP servers are automatically prefixed to avoid naming conflicts:

```
mcp_<server-name>_<tool-name>
```

For example:
- A tool named `read_file` from the `filesystem` server becomes `mcp_filesystem_read_file`
- A tool named `create_issue` from the `github` server becomes `mcp_github_create_issue`

The AI model can use these tools just like any other Kiana tool.

## Example MCP Servers

### Filesystem Server

Provides file system access within a specified directory:

```jsonc
{
  "name": "filesystem",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users/you/projects"],
  "env": {}
}
```

### GitHub Server

Provides GitHub repository access:

```jsonc
{
  "name": "github",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-github"],
  "env": {
    "GITHUB_TOKEN": "your-github-token"
  }
}
```

### PostgreSQL Server

Provides database access:

```jsonc
{
  "name": "postgres",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-postgres", "postgresql://user:pass@localhost/db"],
  "env": {}
}
```

### Custom MCP Server

You can also run your own custom MCP servers:

```jsonc
{
  "name": "my-custom-server",
  "command": "node",
  "args": ["/path/to/my-mcp-server/index.js"],
  "env": {
    "API_KEY": "secret"
  }
}
```

## Usage Example

Once configured, the AI can use MCP tools automatically:

```bash
kiana-v6 -i -H
```

Then you can ask:

```
> List the files in the current directory using the filesystem server
```

The AI will automatically discover and use the appropriate MCP tool.

## Programmatic Usage

You can also use MCP servers programmatically:

```typescript
import { CodingAgent, createLanguageModel } from "kiana-v6"

const agent = new CodingAgent({
  model: createLanguageModel({
    type: "anthropic",
    apiKey: process.env.ANTHROPIC_API_KEY!,
    model: "claude-sonnet-4-20250514",
  }),
  mcpServers: [
    {
      name: "filesystem",
      command: "npx",
      args: ["-y", "@modelcontextprotocol/server-filesystem", process.cwd()],
    },
  ],
})

// The agent will automatically initialize MCP servers
const result = await agent.generate({
  prompt: "List all TypeScript files in the current directory",
})

console.log(result.text)

// Cleanup MCP connections when done
await agent.cleanup()
```

## Advanced Features

### Manual Tool Management

You can also manually manage MCP tools:

```typescript
import { 
  initializeMCPServers, 
  getMCPManager, 
  createMCPTool 
} from "kiana-v6"

// Initialize MCP servers
const mcpTools = await initializeMCPServers([
  {
    name: "filesystem",
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem", "."],
  },
])

// Get the manager for more control
const manager = getMCPManager()

// Call a tool directly
const result = await manager.callTool("filesystem", "read_file", {
  path: "./README.md",
})

// Cleanup
await manager.disconnectAll()
```

### Schema Conversion

MCP tools use JSON Schema for their input parameters. Kiana automatically converts these to Zod schemas for validation. The conversion supports:

- Basic types: string, number, boolean, integer
- Objects with nested properties
- Arrays
- Enums
- Optional/required fields

## Troubleshooting

### Server Connection Issues

If an MCP server fails to connect:

1. Check that the command and args are correct
2. Verify any required environment variables are set
3. Ensure the MCP server package is installed or accessible
4. Check the console for error messages during initialization

### Tool Discovery

If MCP tools are not being used:

1. Verify the server is properly configured in `mcpServers`
2. Check that the server successfully connected (look for initialization errors)
3. Try listing available tools programmatically using `agent.getTools()`

### Performance

MCP servers run as separate processes and communicate via stdio. For best performance:

- Reuse the same agent instance across multiple requests
- Only connect to MCP servers you actually need
- Call `agent.cleanup()` when done to properly close connections

## Finding MCP Servers

You can find more MCP servers at:

- [Official MCP Servers](https://github.com/modelcontextprotocol)
- [Awesome MCP Servers](https://github.com/punkpeye/awesome-mcp-servers)
- Build your own using the [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)

## Limitations

- Currently only supports stdio transport (not SSE or HTTP)
- Schema conversion supports common JSON Schema features but may not handle very complex schemas
- MCP resources and prompts are not yet supported (coming soon)
