package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// SessionInfo represents session information for API responses
type SessionInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	Cwd       string    `json:"cwd"`
}

// CreateSessionResponse represents the response for session creation
type CreateSessionResponse struct {
	Session SessionInfo `json:"session"`
}

// ListSessionsResponse represents the response for listing sessions
type ListSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

// RemoveSessionRequest represents the request for removing a session
type RemoveSessionRequest struct {
	SessionID string `json:"session_id"`
}

// RemoveSessionResponse represents the response for session removal
type RemoveSessionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// APIServer wraps the session manager and provides HTTP handlers
type APIServer struct {
	SessionManager *SessionManager
}

// NewAPIServer creates a new API server
func NewAPIServer() *APIServer {
	return &APIServer{
		SessionManager: NewSessionManager(),
	}
}

// HandleCreateSession handles POST /api/v1/session/create
func (s *APIServer) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := s.SessionManager.CreateSession()
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := CreateSessionResponse{
		Session: SessionInfo{
			ID:        session.ID,
			CreatedAt: session.CreatedAt,
			LastUsed:  session.LastUsed,
			Cwd:       session.Shell.GetCwd(),
		},
	}

	respondJSON(w, response, http.StatusCreated)
}

// HandleListSessions handles POST /api/v1/session/list
func (s *APIServer) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions := s.SessionManager.ListSessions()

	sessionInfos := make([]SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		sessionInfos = append(sessionInfos, SessionInfo{
			ID:        session.ID,
			CreatedAt: session.CreatedAt,
			LastUsed:  session.LastUsed,
			Cwd:       session.Shell.GetCwd(),
		})
	}

	response := ListSessionsResponse{
		Sessions: sessionInfos,
	}

	respondJSON(w, response, http.StatusOK)
}

// HandleRemoveSession handles POST /api/v1/session/remove
func (s *APIServer) HandleRemoveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RemoveSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := s.SessionManager.RemoveSession(req.SessionID)
	if err != nil {
		respondError(w, err.Error(), http.StatusNotFound)
		return
	}

	response := RemoveSessionResponse{
		Success: true,
		Message: "Session removed successfully",
	}

	respondJSON(w, response, http.StatusOK)
}

// HandleREPL handles WebSocket connection for /api/v1/session/repl
func (s *APIServer) HandleREPL(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Read and process JSON-RPC messages
	for {
		var request JSONRPCRequest
		err := conn.ReadJSON(&request)
		if err != nil {
			// Connection closed or error reading
			break
		}

		// Process JSON-RPC request
		response := HandleJSONRPC(r.Context(), s.SessionManager, &request)

		// Send response
		err = conn.WriteJSON(response)
		if err != nil {
			// Error writing, close connection
			break
		}
	}
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
