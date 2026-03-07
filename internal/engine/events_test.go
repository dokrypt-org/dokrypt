package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()
	require.NotNil(t, bus)
	assert.NotNil(t, bus.subscribers)
	assert.Empty(t, bus.subscribers)
}

func TestSubscribe_ReturnsChannel(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe(EventChainStarted)
	require.NotNil(t, ch)
}

func TestSubscribe_MultipleSubscribersForSameEvent(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe(EventChainStarted)
	ch2 := bus.Subscribe(EventChainStarted)
	require.NotNil(t, ch1)
	require.NotNil(t, ch2)

	assert.Len(t, bus.subscribers[EventChainStarted], 2)
}

func TestSubscribe_DifferentEventTypes(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe(EventChainStarted)
	ch2 := bus.Subscribe(EventChainStopped)
	require.NotNil(t, ch1)
	require.NotNil(t, ch2)

	assert.Len(t, bus.subscribers[EventChainStarted], 1)
	assert.Len(t, bus.subscribers[EventChainStopped], 1)
}

func TestPublish_SubscriberReceivesEvent(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe(EventChainStarted)

	bus.Publish(Event{
		Type: EventChainStarted,
		Data: map[string]any{"chain": "anvil"},
	})

	select {
	case evt := <-ch:
		assert.Equal(t, EventChainStarted, evt.Type)
		assert.Equal(t, "anvil", evt.Data["chain"])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPublish_SetsTimestampWhenZero(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe(EventChainStarted)

	before := time.Now()
	bus.Publish(Event{Type: EventChainStarted})
	after := time.Now()

	evt := <-ch
	assert.False(t, evt.Timestamp.IsZero())
	assert.True(t, !evt.Timestamp.Before(before) && !evt.Timestamp.After(after),
		"timestamp should be between before (%v) and after (%v), got %v", before, after, evt.Timestamp)
}

func TestPublish_PreservesExplicitTimestamp(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe(EventChainStarted)

	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	bus.Publish(Event{Type: EventChainStarted, Timestamp: ts})

	evt := <-ch
	assert.Equal(t, ts, evt.Timestamp)
}

func TestPublish_InitializesNilDataMap(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe(EventChainStarted)

	bus.Publish(Event{Type: EventChainStarted, Data: nil})

	evt := <-ch
	assert.NotNil(t, evt.Data)
}

func TestPublish_DoesNotAffectOtherSubscribers(t *testing.T) {
	bus := NewEventBus()
	chStarted := bus.Subscribe(EventChainStarted)
	chStopped := bus.Subscribe(EventChainStopped)

	bus.Publish(Event{Type: EventChainStarted, Data: map[string]any{"chain": "test"}})

	select {
	case evt := <-chStarted:
		assert.Equal(t, EventChainStarted, evt.Type)
	case <-time.After(time.Second):
		t.Fatal("started subscriber timed out")
	}

	select {
	case <-chStopped:
		t.Fatal("stopped subscriber should not have received an event")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPublish_MultipleSubscribersSameType(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe(EventEnvironmentUp)
	ch2 := bus.Subscribe(EventEnvironmentUp)

	bus.Publish(Event{Type: EventEnvironmentUp, Data: map[string]any{"project": "test"}})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case evt := <-ch:
			assert.Equal(t, EventEnvironmentUp, evt.Type)
			assert.Equal(t, "test", evt.Data["project"])
		case <-time.After(time.Second):
			t.Fatal("subscriber timed out")
		}
	}
}

func TestPublish_DropsWhenChannelFull(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe(EventChainStarted)

	for i := 0; i < 16; i++ {
		bus.Publish(Event{Type: EventChainStarted, Data: map[string]any{"i": i}})
	}

	done := make(chan struct{})
	go func() {
		bus.Publish(Event{Type: EventChainStarted, Data: map[string]any{"i": 16}})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked when channel is full; expected drop")
	}

	for i := 0; i < 16; i++ {
		<-ch
	}
}

func TestPublish_NoSubscribers(t *testing.T) {
	bus := NewEventBus()
	assert.NotPanics(t, func() {
		bus.Publish(Event{Type: EventChainStarted, Data: map[string]any{"test": true}})
	})
}

func TestPublish_ConcurrentSafe(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe(EventChainBlockMined)

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			bus.Publish(Event{Type: EventChainBlockMined, Data: map[string]any{"block": i}})
		}(i)
	}
	wg.Wait()

	received := 0
	for {
		select {
		case <-ch:
			received++
		default:
			goto done
		}
	}
done:
	assert.Greater(t, received, 0, "should have received at least some events")
}

func TestClose_ClosesAllSubscriberChannels(t *testing.T) {
	bus := NewEventBus()
	ch1 := bus.Subscribe(EventChainStarted)
	ch2 := bus.Subscribe(EventChainStopped)
	ch3 := bus.Subscribe(EventChainStarted)

	bus.Close()

	_, ok1 := <-ch1
	assert.False(t, ok1, "channel ch1 should be closed")
	_, ok2 := <-ch2
	assert.False(t, ok2, "channel ch2 should be closed")
	_, ok3 := <-ch3
	assert.False(t, ok3, "channel ch3 should be closed")

	assert.Empty(t, bus.subscribers)
}

func TestClose_EmptyBus(t *testing.T) {
	bus := NewEventBus()
	assert.NotPanics(t, func() {
		bus.Close()
	})
}

func TestEventType_Constants(t *testing.T) {
	types := []EventType{
		EventChainStarted,
		EventChainStopped,
		EventChainForked,
		EventChainBlockMined,
		EventServiceStarted,
		EventServiceStopped,
		EventServiceUnhealthy,
		EventSnapshotSaved,
		EventSnapshotRestored,
		EventPluginLoaded,
		EventTestPassed,
		EventTestFailed,
		EventEnvironmentUp,
		EventEnvironmentDown,
	}

	seen := make(map[EventType]bool)
	for _, et := range types {
		assert.NotEmpty(t, string(et), "EventType should not be empty")
		assert.False(t, seen[et], "duplicate EventType: %s", et)
		seen[et] = true
	}
}

func TestEvent_Fields(t *testing.T) {
	ts := time.Now()
	evt := Event{
		Type:      EventChainStarted,
		Timestamp: ts,
		Data:      map[string]any{"foo": "bar"},
	}

	assert.Equal(t, EventChainStarted, evt.Type)
	assert.Equal(t, ts, evt.Timestamp)
	assert.Equal(t, "bar", evt.Data["foo"])
}
