'use client';

import { useState, useEffect, useRef } from 'react';
import { SessionInfo } from '@/types/api';
import { WebSocketClient } from '@/lib/websocket-client';
import { apiClient } from '@/lib/api-client';

interface TerminalProps {
  session: SessionInfo;
  onCwdChange: (cwd: string) => void;
}

interface HistoryEntry {
  command: string;
  output: string[];
  error?: string;
  cwd: string;
}

export default function Terminal({ session, onCwdChange }: TerminalProps) {
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [currentCommand, setCurrentCommand] = useState('');
  const [loading, setLoading] = useState(false);
  const [connected, setConnected] = useState(false);
  const [wsClient, setWsClient] = useState<WebSocketClient | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const historyEndRef = useRef<HTMLDivElement>(null);
  const commandHistoryRef = useRef<string[]>([]);
  const historyIndexRef = useRef<number>(-1);

  useEffect(() => {
    // Initialize WebSocket client
    const client = new WebSocketClient(apiClient.getWebSocketURL());

    client.onConnect(() => {
      setConnected(true);
      console.log('WebSocket connected');
    });

    client.onDisconnect(() => {
      setConnected(false);
      console.log('WebSocket disconnected');
    });

    client.connect().catch((error) => {
      console.error('Failed to connect WebSocket:', error);
    });

    setWsClient(client);

    return () => {
      client.disconnect();
    };
  }, []);

  useEffect(() => {
    // Scroll to bottom when history updates
    historyEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [history]);

  const executeCommand = async () => {
    if (!currentCommand.trim() || !wsClient || !connected) {
      return;
    }

    const commandLine = currentCommand.trim();
    setCurrentCommand('');
    setLoading(true);

    // Add to command history
    commandHistoryRef.current.push(commandLine);
    historyIndexRef.current = commandHistoryRef.current.length;

    try {
      // Parse command and args
      const parts = parseCommandLine(commandLine);
      const [command, ...args] = parts;

      // Execute command via WebSocket
      const result = await wsClient.executeCommand(session.id, command, args);

      // Add to history
      setHistory((prev) => [
        ...prev,
        {
          command: commandLine,
          output: result.output,
          error: result.error,
          cwd: result.cwd,
        },
      ]);

      // Update current working directory
      onCwdChange(result.cwd);
    } catch (error) {
      setHistory((prev) => [
        ...prev,
        {
          command: commandLine,
          output: [],
          error: error instanceof Error ? error.message : 'Command failed',
          cwd: session.cwd,
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const parseCommandLine = (line: string): string[] => {
    // Simple command line parsing (handles quoted strings)
    const parts: string[] = [];
    let current = '';
    let inQuote = false;
    let quoteChar = '';

    for (let i = 0; i < line.length; i++) {
      const char = line[i];

      if ((char === '"' || char === "'") && !inQuote) {
        inQuote = true;
        quoteChar = char;
      } else if (char === quoteChar && inQuote) {
        inQuote = false;
        quoteChar = '';
      } else if (char === ' ' && !inQuote) {
        if (current) {
          parts.push(current);
          current = '';
        }
      } else {
        current += char;
      }
    }

    if (current) {
      parts.push(current);
    }

    return parts;
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      executeCommand();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (historyIndexRef.current > 0) {
        historyIndexRef.current--;
        setCurrentCommand(commandHistoryRef.current[historyIndexRef.current]);
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIndexRef.current < commandHistoryRef.current.length - 1) {
        historyIndexRef.current++;
        setCurrentCommand(commandHistoryRef.current[historyIndexRef.current]);
      } else {
        historyIndexRef.current = commandHistoryRef.current.length;
        setCurrentCommand('');
      }
    }
  };

  const handleClear = () => {
    setHistory([]);
  };

  return (
    <div className="terminal">
      <div className="terminal-header">
        <div className="terminal-title">
          <span className="terminal-icon">▶</span>
          Shell Terminal
        </div>
        <div className="terminal-status">
          <span className={`status-indicator ${connected ? 'connected' : 'disconnected'}`}>
            {connected ? '● Connected' : '○ Disconnected'}
          </span>
          <button onClick={handleClear} className="btn btn-sm">
            Clear
          </button>
        </div>
      </div>

      <div className="terminal-body">
        <div className="terminal-history">
          {history.map((entry, index) => (
            <div key={index} className="history-entry">
              <div className="command-line">
                <span className="prompt">{entry.cwd} $</span>
                <span className="command">{entry.command}</span>
              </div>
              {entry.error ? (
                <div className="error-output">{entry.error}</div>
              ) : (
                <div className="command-output">
                  {entry.output.map((line, lineIndex) => (
                    <div key={lineIndex}>{line}</div>
                  ))}
                </div>
              )}
            </div>
          ))}
          <div ref={historyEndRef} />
        </div>

        <div className="terminal-input-line">
          <span className="prompt">{session.cwd} $</span>
          <input
            ref={inputRef}
            type="text"
            value={currentCommand}
            onChange={(e) => setCurrentCommand(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={loading || !connected}
            placeholder={connected ? 'Enter command...' : 'Connecting...'}
            className="terminal-input"
            autoFocus
          />
          {loading && <span className="loading-spinner">⏳</span>}
        </div>
      </div>
    </div>
  );
}
