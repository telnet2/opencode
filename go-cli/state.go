package main

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// SessionStateEntry persists the last session choices.
type SessionStateEntry struct {
    SessionID string `json:"sessionID"`
    Model     string `json:"model,omitempty"`
    Provider  string `json:"provider,omitempty"`
    Agent     string `json:"agent,omitempty"`
    UpdatedAt int64  `json:"updatedAt"`
}

type stateFile struct {
    Sessions      map[string]SessionStateEntry `json:"sessions"`
    LastSessionID string                      `json:"lastSessionID,omitempty"`
}

func ensureDir(path string) error {
    dir := filepath.Dir(path)
    return os.MkdirAll(dir, 0o755)
}

func readState(path string) stateFile {
    data, err := os.ReadFile(path)
    if err != nil {
        return stateFile{Sessions: map[string]SessionStateEntry{}}
    }
    var s stateFile
    if err := json.Unmarshal(data, &s); err != nil {
        return stateFile{Sessions: map[string]SessionStateEntry{}}
    }
    if s.Sessions == nil {
        s.Sessions = map[string]SessionStateEntry{}
    }
    return s
}

func loadSessionState(cfg ResolvedConfig) *SessionStateEntry {
    st := readState(cfg.SessionFile)
    if cfg.Session != "" {
        if entry, ok := st.Sessions[cfg.Session]; ok {
            return &entry
        }
        return nil
    }
    if st.LastSessionID != "" {
        if entry, ok := st.Sessions[st.LastSessionID]; ok {
            return &entry
        }
    }
    return nil
}

func persistSessionState(cfg ResolvedConfig, entry SessionStateEntry) {
    st := readState(cfg.SessionFile)
    st.Sessions[entry.SessionID] = entry
    st.LastSessionID = entry.SessionID
    if err := ensureDir(cfg.SessionFile); err != nil {
        return
    }
    data, err := json.MarshalIndent(st, "", "  ")
    if err != nil {
        return
    }
    _ = os.WriteFile(cfg.SessionFile, data, 0o644)
}
