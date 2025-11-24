'use client';

import { useState, useEffect } from 'react';
import { SessionInfo } from '@/types/api';
import { apiClient } from '@/lib/api-client';

interface SessionManagerProps {
  currentSession: SessionInfo | null;
  onSessionSelect: (session: SessionInfo) => void;
  onSessionCreate: (session: SessionInfo) => void;
  onSessionRemove: (sessionId: string) => void;
}

export default function SessionManager({
  currentSession,
  onSessionSelect,
  onSessionCreate,
  onSessionRemove,
}: SessionManagerProps) {
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadSessions();
  }, []);

  const loadSessions = async () => {
    try {
      setLoading(true);
      setError(null);
      const sessionList = await apiClient.listSessions();
      setSessions(sessionList);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sessions');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateSession = async () => {
    try {
      setLoading(true);
      setError(null);
      const newSession = await apiClient.createSession();
      setSessions([...sessions, newSession]);
      onSessionCreate(newSession);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session');
    } finally {
      setLoading(false);
    }
  };

  const handleRemoveSession = async (sessionId: string) => {
    try {
      setLoading(true);
      setError(null);
      await apiClient.removeSession(sessionId);
      setSessions(sessions.filter((s) => s.id !== sessionId));
      onSessionRemove(sessionId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove session');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  return (
    <div className="session-manager">
      <div className="session-header">
        <h2>Sessions</h2>
        <button
          onClick={handleCreateSession}
          disabled={loading}
          className="btn btn-primary"
        >
          + New Session
        </button>
      </div>

      {error && (
        <div className="error-message">
          {error}
        </div>
      )}

      {loading && sessions.length === 0 ? (
        <div className="loading">Loading sessions...</div>
      ) : (
        <div className="session-list">
          {sessions.length === 0 ? (
            <div className="empty-state">
              No sessions. Create one to get started.
            </div>
          ) : (
            sessions.map((session) => (
              <div
                key={session.id}
                className={`session-item ${
                  currentSession?.id === session.id ? 'active' : ''
                }`}
                onClick={() => onSessionSelect(session)}
              >
                <div className="session-info">
                  <div className="session-id">
                    {session.id.substring(0, 8)}...
                  </div>
                  <div className="session-cwd">
                    <span className="label">CWD:</span> {session.cwd}
                  </div>
                  <div className="session-time">
                    <span className="label">Created:</span>{' '}
                    {formatDate(session.created_at)}
                  </div>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleRemoveSession(session.id);
                  }}
                  disabled={loading}
                  className="btn btn-danger btn-sm"
                  title="Remove session"
                >
                  ×
                </button>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
