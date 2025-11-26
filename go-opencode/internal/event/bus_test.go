package event

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBus_Subscribe(t *testing.T) {
	bus := NewBus()

	var received Event
	var wg sync.WaitGroup
	wg.Add(1)

	unsub := bus.Subscribe(SessionCreated, func(e Event) {
		received = e
		wg.Done()
	})
	defer unsub()

	event := Event{Type: SessionCreated, Data: "test-session"}
	bus.Publish(event)

	// Wait for async delivery
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if received.Type != SessionCreated {
			t.Errorf("Expected SessionCreated, got %v", received.Type)
		}
		if received.Data != "test-session" {
			t.Errorf("Expected 'test-session', got %v", received.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for event")
	}
}

func TestBus_SubscribeAll(t *testing.T) {
	bus := NewBus()

	var count int32
	var wg sync.WaitGroup
	wg.Add(3)

	unsub := bus.SubscribeAll(func(e Event) {
		atomic.AddInt32(&count, 1)
		wg.Done()
	})
	defer unsub()

	// Publish different event types
	bus.Publish(Event{Type: SessionCreated, Data: nil})
	bus.Publish(Event{Type: MessageCreated, Data: nil})
	bus.Publish(Event{Type: FileEdited, Data: nil})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if atomic.LoadInt32(&count) != 3 {
			t.Errorf("Expected 3 events, got %d", count)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for events")
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	bus := NewBus()

	var count int32
	unsub := bus.Subscribe(SessionCreated, func(e Event) {
		atomic.AddInt32(&count, 1)
	})

	// Publish once
	bus.PublishSync(Event{Type: SessionCreated, Data: nil})
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1 event before unsub, got %d", count)
	}

	// Unsubscribe
	unsub()

	// Publish again - should not be received
	bus.PublishSync(Event{Type: SessionCreated, Data: nil})
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected still 1 event after unsub, got %d", count)
	}
}

func TestBus_UnsubscribeGlobal(t *testing.T) {
	bus := NewBus()

	var count int32
	unsub := bus.SubscribeAll(func(e Event) {
		atomic.AddInt32(&count, 1)
	})

	// Publish once
	bus.PublishSync(Event{Type: SessionCreated, Data: nil})
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1 event before unsub, got %d", count)
	}

	// Unsubscribe
	unsub()

	// Publish again
	bus.PublishSync(Event{Type: MessageCreated, Data: nil})
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected still 1 event after unsub, got %d", count)
	}
}

func TestBus_PublishSync(t *testing.T) {
	bus := NewBus()

	var received []EventType
	var mu sync.Mutex

	bus.Subscribe(SessionCreated, func(e Event) {
		mu.Lock()
		received = append(received, e.Type)
		mu.Unlock()
	})
	bus.Subscribe(SessionUpdated, func(e Event) {
		mu.Lock()
		received = append(received, e.Type)
		mu.Unlock()
	})

	// PublishSync should complete before returning
	bus.PublishSync(Event{Type: SessionCreated, Data: nil})
	bus.PublishSync(Event{Type: SessionUpdated, Data: nil})

	mu.Lock()
	if len(received) != 2 {
		t.Errorf("Expected 2 events, got %d", len(received))
	}
	mu.Unlock()
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := NewBus()

	var count int32
	var wg sync.WaitGroup
	wg.Add(3)

	for i := 0; i < 3; i++ {
		bus.Subscribe(SessionCreated, func(e Event) {
			atomic.AddInt32(&count, 1)
			wg.Done()
		})
	}

	bus.Publish(Event{Type: SessionCreated, Data: nil})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if atomic.LoadInt32(&count) != 3 {
			t.Errorf("Expected 3 subscribers to receive event, got %d", count)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for events")
	}
}

func TestBus_NoSubscribers(t *testing.T) {
	bus := NewBus()

	// Should not panic with no subscribers
	bus.Publish(Event{Type: SessionCreated, Data: nil})
	bus.PublishSync(Event{Type: SessionCreated, Data: nil})
}

func TestBus_EventTypeFiltering(t *testing.T) {
	bus := NewBus()

	var sessionCount, messageCount int32

	bus.Subscribe(SessionCreated, func(e Event) {
		atomic.AddInt32(&sessionCount, 1)
	})
	bus.Subscribe(MessageCreated, func(e Event) {
		atomic.AddInt32(&messageCount, 1)
	})

	bus.PublishSync(Event{Type: SessionCreated, Data: nil})
	bus.PublishSync(Event{Type: SessionCreated, Data: nil})
	bus.PublishSync(Event{Type: MessageCreated, Data: nil})

	if atomic.LoadInt32(&sessionCount) != 2 {
		t.Errorf("Expected 2 session events, got %d", sessionCount)
	}
	if atomic.LoadInt32(&messageCount) != 1 {
		t.Errorf("Expected 1 message event, got %d", messageCount)
	}
}

func TestGlobalBus_Reset(t *testing.T) {
	// Subscribe to global bus
	var count int32
	Subscribe(SessionCreated, func(e Event) {
		atomic.AddInt32(&count, 1)
	})

	PublishSync(Event{Type: SessionCreated, Data: nil})
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected 1 event before reset, got %d", count)
	}

	// Reset
	Reset()

	// Publish again - no subscribers
	PublishSync(Event{Type: SessionCreated, Data: nil})
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected still 1 event after reset, got %d", count)
	}
}

func TestBus_ConcurrentSubscribePublish(t *testing.T) {
	bus := NewBus()

	var count int32
	var wg sync.WaitGroup

	// Start publishers and subscribers concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unsub := bus.Subscribe(SessionCreated, func(e Event) {
				atomic.AddInt32(&count, 1)
			})
			defer unsub()

			for j := 0; j < 10; j++ {
				bus.Publish(Event{Type: SessionCreated, Data: nil})
			}
		}()
	}

	wg.Wait()
	// Give time for async events to be delivered
	time.Sleep(100 * time.Millisecond)

	// Just verify no panic/deadlock occurred
	if atomic.LoadInt32(&count) == 0 {
		t.Log("Warning: no events received, but no panic occurred")
	}
}
