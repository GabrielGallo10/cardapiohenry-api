package realtime

import (
	"sync"
	"time"
)

type Event struct {
	Type string `json:"type"`
	At   string `json:"at"`
}

type hub struct {
	mu   sync.RWMutex
	subs map[chan Event]struct{}
}

var broker = &hub{
	subs: make(map[chan Event]struct{}),
}

func Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 8)
	broker.mu.Lock()
	broker.subs[ch] = struct{}{}
	broker.mu.Unlock()

	cancel := func() {
		broker.mu.Lock()
		if _, ok := broker.subs[ch]; ok {
			delete(broker.subs, ch)
			close(ch)
		}
		broker.mu.Unlock()
	}
	return ch, cancel
}

func Publish(eventType string) {
	evt := Event{
		Type: eventType,
		At:   time.Now().UTC().Format(time.RFC3339),
	}

	broker.mu.RLock()
	for ch := range broker.subs {
		select {
		case ch <- evt:
		default:
		}
	}
	broker.mu.RUnlock()
}
