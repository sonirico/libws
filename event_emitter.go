package libws

import (
	"sync"
)

type callback[T any] func(T)

// EventEmitterCallback is a simple event emitter. It maps events (of type K) to listeners,
// which are receive-only channels (of type V).
// Warning: It is strongly recommended to use receive-only channels as listeners
// to prevent data races or unexpected behavior.
type EventEmitterCallback[K comparable, V any] struct {
	listeners map[K][]callback[V]
	lock      sync.RWMutex
}

// NewEventEmitter creates a new EventEmitterCallback and returns a pointer to it.
func NewEventEmitter[K comparable, V any]() *EventEmitterCallback[K, V] {
	return &EventEmitterCallback[K, V]{
		listeners: make(map[K][]callback[V]),
	}
}

// On registers a new listener for the given event.
func (e *EventEmitterCallback[K, V]) On(event K, listener callback[V]) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.listeners[event] = append(e.listeners[event], listener)
}

// Emit triggers all listeners registered for the given event synchronously,
// sending the event's data to their channels. The method waits until all data has been sent before returning.
// Emit will not send to any channels if the EventEmitterCallback has been closed.
func (e *EventEmitterCallback[K, V]) Emit(event K, data V) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	listeners, found := e.listeners[event]
	if !found {
		return
	}

	for _, listener := range listeners {
		listener(data)
	}
}

// Close closes all listener channels and removes all listeners to prevent memory leaks.
func (e *EventEmitterCallback[K, V]) Close() {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.listeners = make(map[K][]callback[V])
}
