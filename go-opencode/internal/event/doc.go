/*
Package event provides a type-safe, pub/sub event system for the OpenCode server.

The event system enables decoupled communication between different components of the
server by allowing publishers to emit events and subscribers to react to them without
direct dependencies.

# Architecture

The package is built on top of watermill's gochannel for infrastructure while maintaining
direct-call semantics to preserve type information. It provides both synchronous and
asynchronous event publishing patterns.

# Event Types

The system supports various event categories:

Session Events:
  - session.created: New session created
  - session.updated: Session modified
  - session.deleted: Session removed
  - session.idle: Session became idle
  - session.status: Session status changed
  - session.diff: File differences detected
  - session.error: Session error occurred
  - session.compacted: Session history compacted

Message Events:
  - message.created: New message added
  - message.updated: Message modified
  - message.removed: Message deleted
  - message.part.updated: Message part updated (streaming)
  - message.part.removed: Message part removed

File Events:
  - file.edited: File was modified

Permission Events:
  - permission.updated: Permission request created
  - permission.replied: Permission request responded to

Client Tool Events:
  - client-tool.request: Tool execution requested
  - client-tool.registered: Tools registered by client
  - client-tool.unregistered: Tools unregistered by client
  - client-tool.executing: Tool execution started
  - client-tool.completed: Tool execution completed
  - client-tool.failed: Tool execution failed

# Basic Usage

Publishing events:

	// Asynchronous publishing (non-blocking)
	event.Publish(event.Event{
		Type: event.SessionCreated,
		Data: event.SessionCreatedData{
			Info: session,
		},
	})

	// Synchronous publishing (blocking until all subscribers complete)
	event.PublishSync(event.Event{
		Type: event.MessageUpdated,
		Data: event.MessageUpdatedData{
			Info: message,
		},
	})

Subscribing to specific events:

	unsubscribe := event.Subscribe(event.SessionCreated, func(e event.Event) {
		data := e.Data.(event.SessionCreatedData)
		log.Info("Session created", "id", data.Info.ID)
	})
	defer unsubscribe()

Subscribing to all events:

	unsubscribe := event.SubscribeAll(func(e event.Event) {
		log.Debug("Event received", "type", e.Type)
	})
	defer unsubscribe()

# Subscriber Safety Guidelines

When using PublishSync, subscribers are called synchronously in the publisher's
goroutine. To avoid blocking or deadlocks, subscribers MUST:

  - Complete quickly (avoid long-running operations)
  - Use non-blocking channel sends (select with default case)
  - Never call Publish/PublishSync from within a subscriber (no re-entrant publishing)
  - Never acquire locks that the publisher might hold

Example of a safe subscriber:

	event.SubscribeAll(func(e event.Event) {
	    select {
	    case eventChan <- e:
	        // Event sent successfully
	    default:
	        // Channel full, drop event to avoid blocking
	        log.Warn("Event dropped due to full channel", "type", e.Type)
	    }
	})

# Custom Event Bus

For testing or isolation, you can create custom bus instances:

	bus := event.NewBus()
	defer bus.Close()

	unsubscribe := bus.Subscribe(event.SessionCreated, handler)
	bus.PublishSync(event.Event{Type: event.SessionCreated, Data: data})

# SDK Compatibility

Many event types and data structures are designed to be compatible with the OpenCode
SDK. Event names and data field names follow SDK conventions where possible, with
compatibility notes in the type definitions.

# Testing

The package provides utilities for testing:

	// Reset global bus state (use in test cleanup)
	event.Reset()

# Thread Safety

The event bus is thread-safe and can be used concurrently from multiple goroutines.
Both publishing and subscribing operations are protected by internal synchronization.

# Performance Considerations

- Asynchronous publishing (Publish) creates a goroutine per subscriber per event
- Synchronous publishing (PublishSync) calls all subscribers in the current goroutine
- Use PublishSync for critical events where ordering matters
- Use Publish for fire-and-forget notifications
- Consider subscriber performance impact on PublishSync calls

# Integration with Watermill

The package uses watermill's gochannel internally, providing access to the underlying
pubsub infrastructure for advanced use cases:

	pubsub := event.PubSub()
	// Use watermill features like middleware, routing, etc.

This allows future migration to distributed message brokers if needed while maintaining
the current API.
*/
package event