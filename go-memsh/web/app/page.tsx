'use client';

import { useState, useEffect } from 'react';
import { SessionInfo } from '@/types/api';
import { WebSocketClient } from '@/lib/websocket-client';
import { apiClient } from '@/lib/api-client';
import SessionManager from '@/components/SessionManager';
import Terminal from '@/components/Terminal';
import FileExplorer from '@/components/FileExplorer';
import ImportExportDialog from '@/components/ImportExportDialog';

export default function Home() {
  const [currentSession, setCurrentSession] = useState<SessionInfo | null>(null);
  const [wsClient, setWsClient] = useState<WebSocketClient | null>(null);
  const [showImportExport, setShowImportExport] = useState<{
    type: 'import' | 'export';
    isDir: boolean;
    path?: string;
  } | null>(null);

  useEffect(() => {
    // Initialize WebSocket client when session is selected
    if (currentSession) {
      const client = new WebSocketClient(apiClient.getWebSocketURL());
      client.connect().catch((error) => {
        console.error('Failed to connect WebSocket:', error);
      });
      setWsClient(client);

      return () => {
        client.disconnect();
      };
    }
  }, [currentSession?.id]);

  const handleSessionSelect = (session: SessionInfo) => {
    setCurrentSession(session);
  };

  const handleSessionCreate = (session: SessionInfo) => {
    setCurrentSession(session);
  };

  const handleSessionRemove = (sessionId: string) => {
    if (currentSession?.id === sessionId) {
      setCurrentSession(null);
      if (wsClient) {
        wsClient.disconnect();
        setWsClient(null);
      }
    }
  };

  const handleCwdChange = (cwd: string) => {
    if (currentSession) {
      setCurrentSession({ ...currentSession, cwd });
    }
  };

  const handleImportExport = (
    type: 'import' | 'export',
    isDir: boolean,
    path?: string
  ) => {
    setShowImportExport({ type, isDir, path });
  };

  return (
    <div className="app">
      <header className="app-header">
        <h1>🐚 MemSh Web Shell</h1>
        <p className="subtitle">In-Memory Shell with File System</p>
      </header>

      <main className="app-main">
        <aside className="sidebar">
          <SessionManager
            currentSession={currentSession}
            onSessionSelect={handleSessionSelect}
            onSessionCreate={handleSessionCreate}
            onSessionRemove={handleSessionRemove}
          />
        </aside>

        <div className="content">
          {currentSession ? (
            <>
              <div className="content-top">
                <Terminal
                  session={currentSession}
                  onCwdChange={handleCwdChange}
                />
              </div>
              <div className="content-bottom">
                <FileExplorer
                  session={currentSession}
                  wsClient={wsClient}
                  onImportExport={handleImportExport}
                />
              </div>
            </>
          ) : (
            <div className="empty-state-main">
              <div className="empty-icon">🐚</div>
              <h2>Welcome to MemSh Web Shell</h2>
              <p>Create or select a session to get started</p>
              <div className="features">
                <div className="feature">
                  <div className="feature-icon">💾</div>
                  <h3>In-Memory File System</h3>
                  <p>Isolated filesystem for each session</p>
                </div>
                <div className="feature">
                  <div className="feature-icon">⚡</div>
                  <h3>Real-Time Execution</h3>
                  <p>Execute shell commands via WebSocket</p>
                </div>
                <div className="feature">
                  <div className="feature-icon">📁</div>
                  <h3>File Management</h3>
                  <p>Import and export files and directories</p>
                </div>
              </div>
            </div>
          )}
        </div>
      </main>

      {showImportExport && currentSession && (
        <ImportExportDialog
          session={currentSession}
          wsClient={wsClient}
          type={showImportExport.type}
          isDir={showImportExport.isDir}
          path={showImportExport.path}
          onClose={() => setShowImportExport(null)}
        />
      )}
    </div>
  );
}
