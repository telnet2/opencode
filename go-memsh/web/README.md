# MemSh Web Shell

A modern web-based interface for the MemSh in-memory shell with file system.

## Features

- 🐚 **Interactive Terminal**: Execute shell commands in real-time via WebSocket
- 💾 **Session Management**: Create, list, and remove isolated shell sessions
- 📁 **File Explorer**: Browse files and directories with MS Explorer-style tree view
- ↕️ **Import/Export**: Upload and download files and directories
- ⚡ **Real-Time Updates**: Live command execution and output streaming
- 🎨 **Modern UI**: Dark theme with professional styling

## Prerequisites

- Node.js 18+ and npm/yarn
- MemSh API server running (see `../cmd/apiserver`)

## Getting Started

### 1. Install Dependencies

```bash
npm install
# or
yarn install
```

### 2. Configure API Server

Copy the example environment file and configure the API server URL:

```bash
cp .env.local.example .env.local
```

Edit `.env.local`:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### 3. Start the API Server

In a separate terminal, start the MemSh API server:

```bash
cd ../cmd/apiserver
go run main.go -port 8080
```

### 4. Start the Development Server

```bash
npm run dev
# or
yarn dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

## Usage

### Creating a Session

1. Click the **"+ New Session"** button in the sidebar
2. A new session will be created and automatically selected
3. The terminal and file explorer will become active

### Using the Terminal

- Type commands in the input field at the bottom
- Press **Enter** to execute
- Use **↑** and **↓** arrow keys to navigate command history
- The prompt shows the current working directory
- Output is displayed in real-time

Example commands:
```bash
pwd                    # Show current directory
ls -la                 # List files
mkdir /home/user       # Create directory
cd /home/user          # Change directory
echo "Hello" > test.txt  # Create file
cat test.txt           # Read file
jq '.name' data.json   # Process JSON
curl https://api.github.com/users/octocat  # Fetch data
```

### File Explorer

- **Tree View**: Click folders to expand/collapse
- **Selection**: Click files to select (Ctrl/Cmd+Click for multiple)
- **Refresh**: Click 🔄 to reload the file tree

### Import Files

1. Click **"📄↑ Import File"** or **"📁↑ Import Dir"**
2. Enter the target path in the session filesystem
3. Select file(s) or directory from your computer
4. Click **"Import"**

### Export Files

1. Select a file or directory in the file explorer
2. Click **"📄↓ Export File"** or **"📁↓ Export Dir"**
3. The file will be downloaded to your computer
4. Directories are exported as `.tar.gz` archives

### Managing Sessions

- **Switch Session**: Click on a session in the sidebar
- **Remove Session**: Click the **×** button on a session
- Sessions maintain their state (working directory, files, etc.)

## Architecture

### Components

- **SessionManager**: Manages session lifecycle (create, list, remove)
- **Terminal**: Interactive command-line interface with WebSocket connection
- **FileExplorer**: Tree-based file browser with import/export
- **ImportExportDialog**: File upload/download interface

### API Integration

- **REST API**: Session management via HTTP POST endpoints
- **WebSocket**: Real-time command execution via JSON-RPC 2.0
- **API Client**: Abstraction layer for API communication

### State Management

- React hooks for local state management
- WebSocket client for persistent connections
- Session isolation with independent filesystems

## Development

### Project Structure

```
web/
├── app/
│   ├── page.tsx          # Main application page
│   ├── layout.tsx        # Root layout
│   └── globals.css       # Global styles
├── components/
│   ├── SessionManager.tsx
│   ├── Terminal.tsx
│   ├── FileExplorer.tsx
│   └── ImportExportDialog.tsx
├── lib/
│   ├── api-client.ts     # REST API client
│   └── websocket-client.ts  # WebSocket client
├── types/
│   └── api.ts            # TypeScript type definitions
├── package.json
├── tsconfig.json
└── next.config.js
```

### Building for Production

```bash
npm run build
npm start
```

### Linting

```bash
npm run lint
```

## Configuration

### Environment Variables

- `NEXT_PUBLIC_API_URL`: API server URL (default: `http://localhost:8080`)

### API Server

The web application requires the MemSh API server to be running. See [API.md](../API.md) for server documentation.

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+

Requires WebSocket support.

## Troubleshooting

### Connection Issues

- Ensure the API server is running
- Check that `NEXT_PUBLIC_API_URL` matches the server address
- Verify firewall settings allow WebSocket connections

### WebSocket Disconnects

- The client will automatically attempt to reconnect
- Check browser console for error messages
- Restart the API server if persistent issues occur

### Import/Export Errors

- Verify file paths are absolute (start with `/`)
- Check file permissions in the session filesystem
- Ensure files are not too large (browser memory limits)

## Contributing

Contributions are welcome! Please ensure:

- TypeScript types are properly defined
- Components follow the existing patterns
- CSS follows the design system
- Code is properly formatted

## License

Same license as the parent go-memsh project.
