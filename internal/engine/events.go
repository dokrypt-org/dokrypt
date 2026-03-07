package engine

import (
	"sync"
	"time"
)

type EventType string

const (
	EventChainStarted     EventType = "chain.started"
	EventChainStopped     EventType = "chain.stopped"
	EventChainForked      EventType = "chain.forked"
	EventChainBlockMined  EventType = "chain.block_mined"
	EventServiceStarted   EventType = "service.started"
	EventServiceStopped   EventType = "service.stopped"
	EventServiceUnhealthy EventType = "service.unhealthy"
	EventSnapshotSaved    EventType = "snapshot.saved"
	EventSnapshotRestored EventType = "snapshot.restored"
	EventPluginLoaded     EventType = "plugin.loaded"
	EventTestPassed       EventType = "test.passed"
	EventTestFailed       EventType = "test.failed"
	EventEnvironmentUp    EventType = "environment.up"
	EventEnvironmentDown  EventType = "environment.down"
)

type Event struct {
	Type      EventType      `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

type EventBus struct {
	subscribers map[EventType][]chan Event
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan Event),
	}
}

func (b *EventBus) Subscribe(eventType EventType) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 16)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Data == nil {
		event.Data = make(map[string]any)
	}

	for _, ch := range b.subscribers[event.Type] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *EventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for eventType, channels := range b.subscribers {
		for _, ch := range channels {
			close(ch)
		}
		delete(b.subscribers, eventType)
	}
}
