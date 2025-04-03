package libws

type eventEmitter[K comparable, V any] interface {
	// On registers a new listener for the given event.
	On(event K, listener chan V)

	// Emit triggers all listeners registered for the given event synchronously,
	// sending the event's data to their channels.
	Emit(event K, data V)

	// RemoveListener removes the specified listener from the given event.
	RemoveListener(event K, listenerToRemove chan V)

	// RemoveAllListeners removes all listeners for all events.
	RemoveAllListeners()

	// Close closes all listener channels and removes all listeners to prevent memory leaks.
	Close()
}
