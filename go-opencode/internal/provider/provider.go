// Package provider provides LLM provider abstraction using Eino framework.
package provider

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/opencode-ai/opencode/pkg/types"
)

// Provider represents an LLM provider with Eino ChatModel.
type Provider interface {
	// ID returns the provider identifier.
	ID() string

	// Name returns the human-readable provider name.
	Name() string

	// Models returns the list of available models.
	Models() []types.Model

	// ChatModel returns the Eino ChatModel for this provider.
	ChatModel() model.ToolCallingChatModel

	// CreateCompletion creates a streaming completion.
	CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionStream, error)
}

// CompletionRequest represents a request to generate a completion.
type CompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []*schema.Message `json:"messages"`
	Tools       []*schema.ToolInfo `json:"tools,omitempty"`
	MaxTokens   int               `json:"maxTokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"topP,omitempty"`
	StopWords   []string          `json:"stopWords,omitempty"`
}

// CompletionStream wraps an Eino stream reader.
type CompletionStream struct {
	reader *schema.StreamReader[*schema.Message]
}

// NewCompletionStream creates a new completion stream.
func NewCompletionStream(reader *schema.StreamReader[*schema.Message]) *CompletionStream {
	return &CompletionStream{reader: reader}
}

// Recv receives the next message chunk from the stream.
func (s *CompletionStream) Recv() (*schema.Message, error) {
	return s.reader.Recv()
}

// Close closes the stream.
func (s *CompletionStream) Close() {
	s.reader.Close()
}

// ToolInfo represents a tool definition for the LLM.
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// ConvertToEinoTools converts internal tool definitions to Eino format.
func ConvertToEinoTools(tools []ToolInfo) []*schema.ToolInfo {
	result := make([]*schema.ToolInfo, len(tools))
	for i, t := range tools {
		// Parse parameters from JSON schema
		var params map[string]*schema.ParameterInfo
		if len(t.Parameters) > 0 {
			params = parseJSONSchemaToParams(t.Parameters)
		}

		result[i] = &schema.ToolInfo{
			Name: t.Name,
			Desc: t.Description,
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		}
	}
	return result
}

// parseJSONSchemaToParams converts JSON Schema to Eino ParameterInfo.
func parseJSONSchemaToParams(schemaJSON json.RawMessage) map[string]*schema.ParameterInfo {
	var jsonSchema struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}

	if err := json.Unmarshal(schemaJSON, &jsonSchema); err != nil {
		return nil
	}

	requiredSet := make(map[string]bool)
	for _, r := range jsonSchema.Required {
		requiredSet[r] = true
	}

	params := make(map[string]*schema.ParameterInfo)
	for name, prop := range jsonSchema.Properties {
		paramType := schema.String
		switch prop.Type {
		case "integer":
			paramType = schema.Integer
		case "number":
			paramType = schema.Number
		case "boolean":
			paramType = schema.Boolean
		case "array":
			paramType = schema.Array
		case "object":
			paramType = schema.Object
		}

		params[name] = &schema.ParameterInfo{
			Type:     paramType,
			Desc:     prop.Description,
			Required: requiredSet[name],
		}
	}

	return params
}

// ConvertFromEinoMessage converts Eino message to internal types.
func ConvertFromEinoMessage(msg *schema.Message, sessionID string) *types.Message {
	role := "assistant"
	if msg.Role == schema.User {
		role = "user"
	} else if msg.Role == schema.System {
		role = "system"
	} else if msg.Role == schema.Tool {
		role = "tool"
	}

	return &types.Message{
		SessionID: sessionID,
		Role:      role,
	}
}

// ConvertToEinoMessages converts internal messages to Eino format.
func ConvertToEinoMessages(messages []*types.Message, parts map[string][]types.Part) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages))

	for _, msg := range messages {
		role := schema.Assistant
		switch msg.Role {
		case "user":
			role = schema.User
		case "system":
			role = schema.System
		case "tool":
			role = schema.Tool
		}

		// Build content from parts
		content := ""
		var toolCalls []schema.ToolCall

		if msgParts, ok := parts[msg.ID]; ok {
			for _, part := range msgParts {
				switch p := part.(type) {
				case *types.TextPart:
					content += p.Text
				case *types.ToolPart:
					inputJSON, _ := json.Marshal(p.Input)
					toolCalls = append(toolCalls, schema.ToolCall{
						ID: p.ToolCallID,
						Function: schema.FunctionCall{
							Name:      p.ToolName,
							Arguments: string(inputJSON),
						},
					})
				}
			}
		}

		einoMsg := &schema.Message{
			Role:      role,
			Content:   content,
			ToolCalls: toolCalls,
		}

		result = append(result, einoMsg)
	}

	return result
}
