package libws

import (
	"sync"
	"testing"
)

func TestSingleListener(t *testing.T) {
	emitter := NewEventEmitter[string, int]()
	var mu sync.Mutex
	var results []int

	// Registers a single listener for the "event" event.
	emitter.On("event", func(data int) {
		mu.Lock()
		results = append(results, data)
		mu.Unlock()
	})

	emitter.Emit("event", 42)

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 1 || results[0] != 42 {
		t.Errorf("Expected to receive [42], but got %v", results)
	}
}

func TestMultipleListeners(t *testing.T) {
	emitter := NewEventEmitter[string, int]()
	var mu sync.Mutex
	var results []int

	// Registers two listeners for the same event.
	emitter.On("event", func(data int) {
		mu.Lock()
		results = append(results, data)
		mu.Unlock()
	})

	emitter.On("event", func(data int) {
		mu.Lock()
		results = append(results, data*2)
		mu.Unlock()
	})

	emitter.Emit("event", 10)

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 2 {
		t.Errorf("Expected 2 callbacks, but got %d", len(results))
	}

	// Verifies that one receives the original data and the other receives double the value.
	found10, found20 := false, false
	for _, v := range results {
		if v == 10 {
			found10 = true
		}
		if v == 20 {
			found20 = true
		}
	}
	if !found10 || !found20 {
		t.Errorf("Expected results 10 and 20, but got %v", results)
	}
}

func TestNoListeners(t *testing.T) {
	emitter := NewEventEmitter[string, int]()
	// When emitting an event with no listeners, no error or call should occur.
	emitter.Emit("nonexistentEvent", 100)
}

func TestMultipleEvents(t *testing.T) {
	emitter := NewEventEmitter[string, int]()
	var event1Result, event2Result int

	// Registers listeners for different events.
	emitter.On("event1", func(data int) {
		event1Result = data
	})
	emitter.On("event2", func(data int) {
		event2Result = data
	})

	emitter.Emit("event1", 5)
	emitter.Emit("event2", 15)

	if event1Result != 5 {
		t.Errorf("For 'event1', expected 5, got %d", event1Result)
	}
	if event2Result != 15 {
		t.Errorf("For 'event2', expected 15, got %d", event2Result)
	}
}

func TestConcurrent(t *testing.T) {
	emitter := NewEventEmitter[string, int]()
	var mu sync.Mutex
	var results []int
	var wg sync.WaitGroup

	// Concurrently registers 10 listeners.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			emitter.On("event", func(data int) {
				mu.Lock()
				results = append(results, data+i)
				mu.Unlock()
			})
		}(i)
	}
	wg.Wait()

	// Concurrent emission: 10 events are emitted.
	for j := 0; j < 10; j++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			emitter.Emit("event", j)
		}(j)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	// Expect 10 (listeners) * 10 (emissions) = 100 callbacks.
	if len(results) != 100 {
		t.Errorf("Expected 100 callbacks, but got %d", len(results))
	}
}
