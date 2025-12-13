import { z } from "zod"
import { Client } from "@modelcontextprotocol/sdk/client/index.js"
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js"
import { Tool as MCPTool } from "@modelcontextprotocol/sdk/types.js"
import type { Tool, ToolContext, ToolResult } from "./tool.js"
import { defineTool } from "./tool.js"

/**
 * MCP Server Configuration
 */
export interface MCPServerConfig {
  /** Server name/identifier */
  name: string
  /** Command to start the server (e.g., "node", "npx") */
  command: string
  /** Arguments to pass to the command */
  args: string[]
  /** Environment variables for the server */
  env?: Record<string, string>
}

/**
 * MCP Client Manager
 */
export class MCPClientManager {
  private clients = new Map<string, Client>()
  private transports = new Map<string, StdioClientTransport>()
  private toolsCache = new Map<string, MCPTool[]>()

  /**
   * Connect to an MCP server
   */
  async connect(config: MCPServerConfig): Promise<void> {
    if (this.clients.has(config.name)) {
      return // Already connected
    }

    // Build env with proper typing
    const env: Record<string, string> = {}
    
    // Copy process.env, filtering out undefined values
    for (const [key, value] of Object.entries(process.env)) {
      if (value !== undefined) {
        env[key] = value
      }
    }
    
    // Override with config env
    if (config.env) {
      Object.assign(env, config.env)
    }
    
    const transport = new StdioClientTransport({
      command: config.command,
      args: config.args,
      env,
    })

    const client = new Client(
      {
        name: `kiana-v6-${config.name}`,
        version: "1.0.0",
      },
      {
        capabilities: {},
      }
    )

    await client.connect(transport)
    
    this.clients.set(config.name, client)
    this.transports.set(config.name, transport)

    // Fetch and cache tools
    const { tools } = await client.listTools()
    this.toolsCache.set(config.name, tools)
  }

  /**
   * Disconnect from an MCP server
   */
  async disconnect(serverName: string): Promise<void> {
    const client = this.clients.get(serverName)
    const transport = this.transports.get(serverName)

    if (client && transport) {
      await client.close()
      await transport.close()
      this.clients.delete(serverName)
      this.transports.delete(serverName)
      this.toolsCache.delete(serverName)
    }
  }

  /**
   * Disconnect from all servers
   */
  async disconnectAll(): Promise<void> {
    const servers = Array.from(this.clients.keys())
    await Promise.all(servers.map((name) => this.disconnect(name)))
  }

  /**
   * Get all tools from all connected MCP servers
   */
  getAllTools(): Array<{ serverName: string; tool: MCPTool }> {
    const allTools: Array<{ serverName: string; tool: MCPTool }> = []
    
    for (const [serverName, tools] of this.toolsCache.entries()) {
      for (const tool of tools) {
        allTools.push({ serverName, tool })
      }
    }
    
    return allTools
  }

  /**
   * Call a tool on an MCP server
   */
  async callTool(
    serverName: string,
    toolName: string,
    args: Record<string, unknown>
  ): Promise<any> {
    const client = this.clients.get(serverName)
    if (!client) {
      throw new Error(`MCP server "${serverName}" not connected`)
    }

    const result = await client.callTool({
      name: toolName,
      arguments: args,
    })

    return result
  }
}

// Global MCP client manager
let mcpManager: MCPClientManager | null = null

/**
 * Get or create the global MCP client manager
 */
export function getMCPManager(): MCPClientManager {
  if (!mcpManager) {
    mcpManager = new MCPClientManager()
  }
  return mcpManager
}

/**
 * Convert MCP JSON Schema to Zod schema
 * This is a simplified converter - may need to be extended for complex schemas
 */
function mcpSchemaToZod(schema: any): z.ZodType {
  if (!schema || typeof schema !== "object") {
    return z.any()
  }

  const type = schema.type

  if (type === "object") {
    const shape: Record<string, z.ZodType> = {}
    
    if (schema.properties) {
      for (const [key, propSchema] of Object.entries(schema.properties)) {
        let fieldSchema = mcpSchemaToZod(propSchema)
        
        // Check if field is required
        const required = schema.required?.includes(key) ?? false
        if (!required) {
          fieldSchema = fieldSchema.optional()
        }
        
        shape[key] = fieldSchema
      }
    }
    
    return z.object(shape)
  }

  if (type === "string") {
    let stringSchema = z.string()
    if (schema.enum) {
      return z.enum(schema.enum as [string, ...string[]])
    }
    return stringSchema
  }

  if (type === "number" || type === "integer") {
    return z.number()
  }

  if (type === "boolean") {
    return z.boolean()
  }

  if (type === "array") {
    const itemSchema = schema.items ? mcpSchemaToZod(schema.items) : z.any()
    return z.array(itemSchema)
  }

  // Default to any for unsupported types
  return z.any()
}

/**
 * Convert an MCP tool to a Kiana tool
 */
export function createMCPTool(serverName: string, mcpTool: MCPTool): Tool {
  const manager = getMCPManager()
  
  // Convert MCP input schema to Zod
  const zodSchema = mcpTool.inputSchema
    ? mcpSchemaToZod(mcpTool.inputSchema)
    : z.object({})

  // Create a unique tool ID: mcp_<server>_<tool>
  const toolId = `mcp_${serverName}_${mcpTool.name}`
  
  return defineTool(toolId, {
    description: mcpTool.description || `MCP tool: ${mcpTool.name} from ${serverName}`,
    parameters: zodSchema,
    execute: async (args: any, ctx: ToolContext): Promise<ToolResult> => {
      try {
        const result = await manager.callTool(serverName, mcpTool.name, args)
        
        // Extract text content from MCP result
        let output = ""
        if (result.content) {
          for (const item of result.content) {
            if (item.type === "text") {
              output += item.text + "\n"
            } else if (item.type === "image") {
              output += `[Image: ${item.data?.substring(0, 50)}...]\n`
            } else if (item.type === "resource") {
              output += `[Resource: ${item.resource?.uri}]\n`
            }
          }
        }

        return {
          title: `MCP: ${mcpTool.name}`,
          output: output.trim() || JSON.stringify(result),
          metadata: {
            serverName,
            toolName: mcpTool.name,
            isError: result.isError ?? false,
          },
        }
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error)
        throw new Error(`MCP tool "${mcpTool.name}" failed: ${errorMsg}`)
      }
    },
  })
}

/**
 * Initialize MCP servers and return tools
 */
export async function initializeMCPServers(
  configs: MCPServerConfig[]
): Promise<Record<string, Tool>> {
  const manager = getMCPManager()
  const tools: Record<string, Tool> = {}

  // Connect to all servers
  await Promise.all(configs.map((config) => manager.connect(config)))

  // Convert all MCP tools to Kiana tools
  const allTools = manager.getAllTools()
  for (const { serverName, tool } of allTools) {
    const kianaTool = createMCPTool(serverName, tool)
    tools[kianaTool.id] = kianaTool
  }

  return tools
}
