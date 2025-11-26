// Package event provides a pub/sub event system for the server using watermill.
package event

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// EventType represents the type of event.
type EventType string

const (
	SessionCreated     EventType = "session.created"
	SessionUpdated     EventType = "session.updated"
	SessionDeleted     EventType = "session.deleted"
	MessageCreated     EventType = "message.created"
	MessageUpdated     EventType = "message.updated"
	MessageRemoved     EventType = "message.removed"
	PartUpdated        EventType = "part.updated"
	FileEdited         EventType = "file.edited"
	PermissionRequired EventType = "permission.required"
	PermissionResolved EventType = "permission.resolved"
)

// Event represents an event to be published.
type Event struct {
	Type EventType `json:"type"`
	Data any       `json:"data"`
}

// Subscriber is a function that receives events.
type Subscriber func(event Event)

// subscriberEntry wraps a subscriber with an ID.
type subscriberEntry struct {
	id uint64
	fn Subscriber
}

// Bus is the event bus that manages pub/sub using watermill.
// It uses watermill's gochannel for infrastructure while maintaining
// the original direct-call semantics to preserve type information.
type Bus struct {
	mu sync.RWMutex

	// Watermill pub/sub infrastructure for potential future middleware/routing
	pubsub *gochannel.GoChannel

	// Direct subscriber tracking - preserves type information
	subscribers map[EventType][]subscriberEntry
	global      []subscriberEntry

	nextID       uint64
	closed       bool
	closedCancel context.CancelFunc
	closedCtx    context.Context
}

// globalBus is the default event bus instance.
var globalBus = newBus()

// newBus creates a new event bus with watermill infrastructure.
func newBus() *Bus {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bus{
		pubsub: gochannel.NewGoChannel(
			gochannel.Config{
				OutputChannelBuffer: 100,
				Persistent:          false,
			},
			watermill.NopLogger{},
		),
		subscribers:  make(map[EventType][]subscriberEntry),
		closedCtx:    ctx,
		closedCancel: cancel,
	}
}

// newID generates a unique subscriber ID.
func (b *Bus) newID() uint64 {
	return atomic.AddUint64(&b.nextID, 1)
}

// Subscribe registers a subscriber for a specific event type.
// Returns an unsubscribe function.
func Subscribe(eventType EventType, fn Subscriber) func() {
	return globalBus.Subscribe(eventType, fn)
}

func (b *Bus) Subscribe(eventType EventType, fn Subscriber) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return func() {}
	}

	id := b.newID()
	entry := subscriberEntry{id: id, fn: fn}
	b.subscribers[eventType] = append(b.subscribers[eventType], entry)

	// Return unsubscribe function
	return func() {
		b.unsubscribe(eventType, id)
	}
}

// SubscribeAll registers a subscriber for all events.
// Returns an unsubscribe function.
func SubscribeAll(fn Subscriber) func() {
	return globalBus.SubscribeAll(fn)
}

func (b *Bus) SubscribeAll(fn Subscriber) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return func() {}
	}

	id := b.newID()
	entry := subscriberEntry{id: id, fn: fn}
	b.global = append(b.global, entry)

	return func() {
		b.unsubscribeGlobal(id)
	}
}

// unsubscribe removes a subscriber for a specific event type.
func (b *Bus) unsubscribe(eventType EventType, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[eventType]
	for i, entry := range subs {
		if entry.id == id {
			b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

// unsubscribeGlobal removes a global subscriber.
func (b *Bus) unsubscribeGlobal(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, entry := range b.global {
		if entry.id == id {
			b.global = append(b.global[:i], b.global[i+1:]...)
			break
		}
	}
}

// Publish sends an event to all subscribers asynchronously.
// Each subscriber is called in its own goroutine to prevent blocking.
func Publish(event Event) {
	globalBus.Publish(event)
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}

	// Collect all subscribers that should receive this event
	subs := make([]Subscriber, 0, len(b.subscribers[event.Type])+len(b.global))
	for _, entry := range b.subscribers[event.Type] {
		subs = append(subs, entry.fn)
	}
	for _, entry := range b.global {
		subs = append(subs, entry.fn)
	}
	b.mu.RUnlock()

	// Publish to all subscribers concurrently
	for _, sub := range subs {
		go sub(event)
	}
}

// PublishSync sends an event to all subscribers synchronously.
// All subscribers are called in the current goroutine before returning.
func PublishSync(event Event) {
	globalBus.PublishSync(event)
}

func (b *Bus) PublishSync(event Event) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}

	// Collect subscribers under read lock
	subs := make([]Subscriber, 0, len(b.subscribers[event.Type])+len(b.global))
	for _, entry := range b.subscribers[event.Type] {
		subs = append(subs, entry.fn)
	}
	for _, entry := range b.global {
		subs = append(subs, entry.fn)
	}
	b.mu.RUnlock()

	// Call all subscribers synchronously
	for _, sub := range subs {
		sub(event)
	}
}

// NewBus creates a new event bus instance.
func NewBus() *Bus {
	return newBus()
}

// Reset clears all subscribers from the global bus (for testing).
func Reset() {
	globalBus.mu.Lock()
	globalBus.closed = true
	globalBus.closedCancel()
	globalBus.mu.Unlock()

	// Close the old pubsub
	_ = globalBus.pubsub.Close()

	// Small delay to allow goroutines to clean up
	time.Sleep(10 * time.Millisecond)

	// Create a new global bus
	globalBus = newBus()
}

// Close closes the bus and all its subscribers.
func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.closedCancel()

	b.subscribers = make(map[EventType][]subscriberEntry)
	b.global = nil
	b.mu.Unlock()

	return b.pubsub.Close()
}

// PubSub returns the underlying watermill GoChannel for advanced use cases.
// This can be used for middleware, routing, or when switching to distributed backends.
func (b *Bus) PubSub() *gochannel.GoChannel {
	return b.pubsub
}

// PubSub returns the global bus's underlying watermill GoChannel.
func PubSub() *gochannel.GoChannel {
	return globalBus.PubSub()
}
