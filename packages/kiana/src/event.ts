import type { Event, EventType, SessionEvent, MessageEvent, PartEvent, TodoEvent } from "./types/event.js"

// Re-export types for convenience
export type { Event as EventTypes, EventType, SessionEvent, MessageEvent, PartEvent, TodoEvent }

type EventCallback = (event: Event) => void
type TypedCallback<T extends EventType> = (
  event: Extract<Event, { type: T }>
) => void

/**
 * Simple typed event bus for Kiana events.
 * Supports both typed subscriptions (by event type) and global subscriptions.
 */
export class EventBus {
  private listeners: Map<EventType | "*", Set<EventCallback>> = new Map()

  /**
   * Subscribe to a specific event type.
   * Returns an unsubscribe function.
   */
  subscribe(callback: EventCallback): () => void
  subscribe<T extends EventType>(type: T, callback: TypedCallback<T>): () => void
  subscribe<T extends EventType>(
    typeOrCallback: T | EventCallback,
    callback?: TypedCallback<T>
  ): () => void {
    if (typeof typeOrCallback === "function") {
      // Global subscription
      return this.subscribeAll(typeOrCallback)
    }

    // Type-specific subscription
    const type = typeOrCallback
    const cb = callback!
    const listeners = this.listeners.get(type) ?? new Set()
    listeners.add(cb as EventCallback)
    this.listeners.set(type, listeners)

    return () => {
      listeners.delete(cb as EventCallback)
      if (listeners.size === 0) {
        this.listeners.delete(type)
      }
    }
  }

  /**
   * Subscribe to all events.
   * Returns an unsubscribe function.
   */
  subscribeAll(callback: EventCallback): () => void {
    const listeners = this.listeners.get("*") ?? new Set()
    listeners.add(callback)
    this.listeners.set("*", listeners)

    return () => {
      listeners.delete(callback)
      if (listeners.size === 0) {
        this.listeners.delete("*")
      }
    }
  }

  /**
   * Publish an event to all matching subscribers.
   * Alias: emit
   */
  publish(event: Event): void {
    // Notify type-specific listeners
    const typeListeners = this.listeners.get(event.type)
    if (typeListeners) {
      for (const callback of typeListeners) {
        try {
          callback(event)
        } catch (err) {
          console.error(`Error in event listener for ${event.type}:`, err)
        }
      }
    }

    // Notify global listeners
    const globalListeners = this.listeners.get("*")
    if (globalListeners) {
      for (const callback of globalListeners) {
        try {
          callback(event)
        } catch (err) {
          console.error(`Error in global event listener:`, err)
        }
      }
    }
  }

  /**
   * Emit an event (alias for publish).
   */
  emit(event: Event): void {
    this.publish(event)
  }

  /**
   * Remove all listeners.
   */
  clear(): void {
    this.listeners.clear()
  }
}

// Default singleton instance
export const eventBus = new EventBus()
