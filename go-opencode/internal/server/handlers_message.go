package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opencode-ai/opencode/internal/event"
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
	textPart := &types.TextPart{
		ID:   generateID(),
		Type: "text",
		Text: content,
	}
	userParts := []types.Part{textPart}

	// Save text part to storage
	if err := s.sessionService.SavePart(r.Context(), userMsg.ID, textPart); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	// Add file parts if provided
	for _, file := range req.Files {
		file.ID = generateID()
		file.Type = "file"
		userParts = append(userParts, &file)
		// Save file part to storage
		if err := s.sessionService.SavePart(r.Context(), userMsg.ID, &file); err != nil {
			writeError(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
	}

	// Publish user message via SSE (not in HTTP response)
	event.Publish(event.Event{
		Type: "message.created",
		Data: event.MessageCreatedData{Info: userMsg},
	})

	// Process message and generate response
	// This is where the LLM provider is called
	// Updates are published via SSE, not streamed in HTTP response
	// IMPORTANT: Use background context for LLM processing to avoid cancellation
	// when the HTTP request completes. The LLM call can take seconds/minutes.
	if req.Model != nil {
		fmt.Printf("[message] Processing with provider=%s model=%s\n", req.Model.ProviderID, req.Model.ModelID)
	} else {
		fmt.Printf("[message] Processing with no model specified\n")
	}
	llmCtx := context.Background()
	assistantMsg, parts, err := s.sessionService.ProcessMessage(llmCtx, session, content, req.Model, func(msg *types.Message, parts []types.Part) {
		// Publish updates via SSE
		event.Publish(event.Event{
			Type: "message.updated",
			Data: event.MessageUpdatedData{Info: msg},
		})
	})

	// Create JSON encoder for response
	encoder := json.NewEncoder(w)

	if err != nil {
		// Create error object
		msgError := types.NewUnknownError(err.Error())

		// Send error response as an assistant message
		if assistantMsg != nil {
			// We have a partial assistant message - add error info
			assistantMsg.Error = msgError
			encoder.Encode(MessageResponse{
				Info:  assistantMsg,
				Parts: parts,
			})
		} else {
			// No assistant message yet - create error message
			errorMsg := &types.Message{
				ID:        generateID(),
				SessionID: sessionID,
				Role:      "assistant",
				Time: types.MessageTime{
					Created: nowMillis(),
				},
				Error:  msgError,
				Tokens: &types.TokenUsage{Input: 0, Output: 0}, // TUI expects tokens to be present
			}
			if req.Model != nil {
				errorMsg.ProviderID = req.Model.ProviderID
				errorMsg.ModelID = req.Model.ModelID
			}
			errorParts := []types.Part{
				&types.TextPart{
					ID:   generateID(),
					Type: "text",
					Text: fmt.Sprintf("Error: %s", err.Error()),
				},
			}
			encoder.Encode(MessageResponse{
				Info:  errorMsg,
				Parts: errorParts,
			})

			// Publish session.error event via SSE
			event.Publish(event.Event{
				Type: "session.error",
				Data: event.SessionErrorData{
					SessionID: sessionID,
					Error:     msgError,
				},
			})
		}
		flusher.Flush()
		return
	}

	// Final message - only send if we have a valid assistant message
	if assistantMsg != nil {
		encoder.Encode(MessageResponse{
			Info:  assistantMsg,
			Parts: parts,
		})
		flusher.Flush()
	}
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
	// Initialize as empty slice to ensure we return [] not null
	result := make([]MessageResponse, 0, len(messages))
	for _, msg := range messages {
		parts, _ := s.sessionService.GetParts(r.Context(), msg.ID)
		// Ensure parts is not null
		if parts == nil {
			parts = []types.Part{}
		}
		result = append(result, MessageResponse{
			Info:  msg,
			Parts: parts,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// getMessage handles GET /session/{sessionID}/message/{messageID}
func (s *Server) getMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	messageID := chi.URLParam(r, "messageID")

	msg, err := s.sessionService.GetMessage(r.Context(), sessionID, messageID)
	if err != nil {
		writeError(w, http.StatusNotFound, ErrCodeNotFound, "Message not found")
		return
	}

	parts, _ := s.sessionService.GetParts(r.Context(), messageID)
	// Ensure parts is not null
	if parts == nil {
		parts = []types.Part{}
	}

	writeJSON(w, http.StatusOK, MessageResponse{
		Info:  msg,
		Parts: parts,
	})
}
