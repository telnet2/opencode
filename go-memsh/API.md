# MemSh API Documentation

The MemSh API provides a REST API and WebSocket-based JSON-RPC interface for creating and managing shell sessions.

## Overview

- **REST API**: Session management (create, list, remove)
- **WebSocket JSON-RPC**: Execute commands in sessions via REPL interface
- **Session Isolation**: Each session has its own filesystem and environment
- **Stateful**: Sessions maintain working directory and environment between commands

## Getting Started

### Start the API Server

```bash
cd cmd/apiserver
go run main.go -port 8080
```

### Run the Example Client

```bash
cd cmd/apiclient
go run main.go -server http://localhost:8080
```

## REST API Endpoints

### 1. Create Session

Create a new shell session with isolated filesystem.

**Endpoint:** `POST /api/v1/session/create`

**Request:** Empty body

**Response:**
```json
{
  "session": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-01-01T12:00:00Z",
    "last_used": "2024-01-01T12:00:00Z",
    "cwd": "/"
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/session/create
```

### 2. List Sessions

List all active sessions.

**Endpoint:** `POST /api/v1/session/list`

**Request:** Empty body

**Response:**
```json
{
  "sessions": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2024-01-01T12:00:00Z",
      "last_used": "2024-01-01T12:00:00Z",
      "cwd": "/home/user"
    }
  ]
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/session/list
```

### 3. Remove Session

Remove a session and clean up its resources.

**Endpoint:** `POST /api/v1/session/remove`

**Request:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Session removed successfully"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/session/remove \
  -H "Content-Type: application/json" \
  -d '{"session_id": "550e8400-e29b-41d4-a716-446655440000"}'
```

## WebSocket JSON-RPC REPL

Execute shell commands in a session using JSON-RPC 2.0 over WebSocket.

**Endpoint:** `WS /api/v1/session/repl`

### JSON-RPC Request Format

```json
{
  "jsonrpc": "2.0",
  "method": "shell.execute",
  "params": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "command": "ls",
    "args": ["-la", "/home"]
  },
  "id": 1
}
```

### JSON-RPC Response Format

**Success:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "output": [
      "total 0",
      "drwxr-xr-x  2 user user 0 Jan  1 12:00 .",
      "drwxr-xr-x  3 user user 0 Jan  1 12:00 .."
    ],
    "cwd": "/home",
    "error": ""
  },
  "id": 1
}
```

**Error:**
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid session",
    "data": "session not found: invalid-id"
  },
  "id": 1
}
```

### JSON-RPC Methods

#### shell.execute

Execute a shell command in a session.

**Parameters:**
- `session_id` (string, required): Session ID from session creation
- `command` (string, required): Command to execute
- `args` (array of strings, optional): Command arguments

**Result:**
- `output` (array of strings): Command output lines
- `cwd` (string): Current working directory after command execution
- `error` (string): Error message if command failed (empty if successful)

**Error Codes:**
- `-32700`: Parse error
- `-32600`: Invalid request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error

## Usage Examples

### JavaScript (Browser/Node.js)

```javascript
// Create session
const response = await fetch('http://localhost:8080/api/v1/session/create', {
  method: 'POST'
});
const { session } = await response.json();
console.log('Session ID:', session.id);

// Connect to WebSocket REPL
const ws = new WebSocket('ws://localhost:8080/api/v1/session/repl');

ws.onopen = () => {
  // Execute command
  ws.send(JSON.stringify({
    jsonrpc: '2.0',
    method: 'shell.execute',
    params: {
      session_id: session.id,
      command: 'ls',
      args: ['-la']
    },
    id: 1
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  if (response.result) {
    console.log('Output:', response.result.output);
    console.log('CWD:', response.result.cwd);
  } else if (response.error) {
    console.error('Error:', response.error.message);
  }
};
```

### Python

```python
import requests
import websocket
import json

# Create session
resp = requests.post('http://localhost:8080/api/v1/session/create')
session_id = resp.json()['session']['id']
print(f'Session ID: {session_id}')

# Connect to WebSocket
ws = websocket.create_connection('ws://localhost:8080/api/v1/session/repl')

# Execute command
request = {
    'jsonrpc': '2.0',
    'method': 'shell.execute',
    'params': {
        'session_id': session_id,
        'command': 'pwd',
        'args': []
    },
    'id': 1
}

ws.send(json.dumps(request))
response = json.loads(ws.recv())

if 'result' in response:
    print('Output:', response['result']['output'])
    print('CWD:', response['result']['cwd'])
else:
    print('Error:', response['error']['message'])

ws.close()
```

### Go

See `cmd/apiclient/main.go` for a complete Go example.

## Command Examples

### Navigate Filesystem
```json
{
  "jsonrpc": "2.0",
  "method": "shell.execute",
  "params": {
    "session_id": "...",
    "command": "cd",
    "args": ["/home/user"]
  },
  "id": 1
}
```

### Create Files and Directories
```json
{
  "jsonrpc": "2.0",
  "method": "shell.execute",
  "params": {
    "session_id": "...",
    "command": "mkdir",
    "args": ["-p", "/home/user/project"]
  },
  "id": 2
}
```

### Process JSON with jq
```json
{
  "jsonrpc": "2.0",
  "method": "shell.execute",
  "params": {
    "session_id": "...",
    "command": "echo",
    "args": ["{\"name\":\"test\"}", ">", "/data.json"]
  },
  "id": 3
}
```

### Fetch Data with curl
```json
{
  "jsonrpc": "2.0",
  "method": "shell.execute",
  "params": {
    "session_id": "...",
    "command": "curl",
    "args": ["-s", "https://api.github.com/users/octocat"]
  },
  "id": 4
}
```

## Architecture

### Components

1. **Session Manager**: Manages lifecycle of shell sessions
2. **JSON-RPC Handler**: Processes JSON-RPC 2.0 requests
3. **HTTP Handlers**: REST API endpoints for session management
4. **WebSocket Handler**: Bidirectional communication for REPL

### Session Lifecycle

1. **Create**: `POST /api/v1/session/create` → Returns session ID
2. **Use**: Connect to WebSocket and execute commands
3. **Maintain**: Session persists between commands
4. **Remove**: `POST /api/v1/session/remove` → Cleans up resources

### Session State

Each session maintains:
- **Filesystem**: Isolated in-memory filesystem (afero.MemMapFs)
- **Working Directory**: Persistent across commands
- **Environment Variables**: Session-specific environment
- **Command History**: Tracked via last_used timestamp

## Best Practices

1. **Session Management**:
   - Create sessions when needed
   - Remove sessions when done to free resources
   - Track session IDs for multi-session scenarios

2. **Error Handling**:
   - Check `response.error` in JSON-RPC responses
   - Check `result.error` for command execution errors
   - HTTP status codes indicate REST API errors

3. **Command Execution**:
   - Commands run in session context
   - Working directory persists between commands
   - Use `result.cwd` to track directory changes

4. **WebSocket**:
   - Keep connection alive for multiple commands
   - One request-response at a time per connection
   - Reconnect if connection is lost

## Health Check

**Endpoint:** `GET /health`

Returns `200 OK` if server is running.

```bash
curl http://localhost:8080/health
```
