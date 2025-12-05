package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opencode-ai/opencode/pkg/types"
)

// TextPartInput represents a text part in the SDK format.
type TextPartInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SendMessageRequest represents the request to send a message.
// Supports both legacy "content" field and SDK "parts" array format.
type SendMessageRequest struct {
	Content string           `json:"content"`
	Parts   []TextPartInput  `json:"parts,omitempty"` // SDK format
	Agent   string           `json:"agent,omitempty"`
	Model   *types.ModelRef  `json:"model,omitempty"`
	Tools   map[string]bool  `json:"tools,omitempty"`
	Files   []types.FilePart `json:"files,omitempty"`
}

// GetContent returns the message content from either Content or Parts.
func (r *SendMessageRequest) GetContent() string {
	if r.Content != "" {
		return r.Content
	}
	// Extract text from parts (SDK format)
	for _, part := range r.Parts {
		if part.Type == "text" && part.Text != "" {
			return part.Text
		}
	}
	return ""
}

// MessageResponse represents a message with its parts.
type MessageResponse struct {
	Info  *types.Message `json:"info"`
	Parts []types.Part   `json:"parts"`
}

// sendMessage handles POST /session/{sessionID}/message
// This is a streaming endpoint that returns chunked JSON.
func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
		return
	}

	content := req.GetContent()
	if content == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "content is required")
		return
	}

	// Set streaming headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, "Streaming not supported")
		return
	}

	// Get session
	session, err := s.sessionService.Get(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "Session not found")
		return
	}

	// Create user message
	userMsg := &types.Message{
		ID:        generateID(),
		SessionID: sessionID,
		Role:      "user",
		Agent:     req.Agent,
		Model:     req.Model,
		Tools:     req.Tools,
		Time: types.MessageTime{
			Created: nowMillis(),
		},
	}

	// Store user message
	if err := s.sessionService.AddMessage(r.Context(), sessionID, userMsg); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Create user message parts
	userParts := []types.Part{
		&types.TextPart{
			ID:   generateID(),
			Type: "text",
			Text: content,
		},
	}

	// Add file parts if provided
	for _, file := range req.Files {
		file.ID = generateID()
		file.Type = "file"
		userParts = append(userParts, &file)
	}

	// Stream user message
	encoder := json.NewEncoder(w)
	encoder.Encode(MessageResponse{
		Info:  userMsg,
		Parts: userParts,
	})
	flusher.Flush()

	// Process message and generate response
	// This is where the LLM provider is called
	assistantMsg, parts, err := s.sessionService.ProcessMessage(r.Context(), session, content, req.Model, func(msg *types.Message, parts []types.Part) {
		// Stream each update
		encoder.Encode(MessageResponse{
			Info:  msg,
			Parts: parts,
		})
		flusher.Flush()
	})

	if err != nil {
		// Send final message with error - still include collected parts
		errResp := MessageResponse{
			Info:  assistantMsg,
			Parts: parts,
		}
		encoder.Encode(errResp)
		flusher.Flush()
		return
	}

	// Final message
	encoder.Encode(MessageResponse{
		Info:  assistantMsg,
		Parts: parts,
	})
	flusher.Flush()
}

// getMessages handles GET /session/{sessionID}/message
func (s *Server) getMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	messages, err := s.sessionService.GetMessages(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Include parts for each message
	var result []MessageResponse
	for _, msg := range messages {
		parts, _ := s.sessionService.GetParts(r.Context(), msg.ID)
		result = append(result, MessageResponse{
			Info:  msg,
			Parts: parts,
		})
	}

	writeJSON(w, http.StatusOK, result)
}
