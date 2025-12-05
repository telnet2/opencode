// Package session provides session management functionality.
package session

import (
	"context"

	"github.com/opencode-ai/opencode/internal/event"
	"github.com/opencode-ai/opencode/internal/storage"
	"github.com/opencode-ai/opencode/pkg/types"
)

// GetTodos retrieves todos for a session.
func GetTodos(ctx context.Context, store *storage.Storage, sessionID string) ([]types.TodoInfo, error) {
	var todos []types.TodoInfo
	err := store.Get(ctx, []string{"todo", sessionID}, &todos)
	if err == storage.ErrNotFound {
		return []types.TodoInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	return todos, nil
}

// UpdateTodos updates todos for a session and publishes an event.
func UpdateTodos(ctx context.Context, store *storage.Storage, sessionID string, todos []types.TodoInfo) error {
	if err := store.Put(ctx, []string{"todo", sessionID}, todos); err != nil {
		return err
	}
	event.Publish(event.Event{
		Type: event.TodoUpdated,
		Data: map[string]any{
			"sessionID": sessionID,
			"todos":     todos,
		},
	})
	return nil
}
