package realtime

import (
	"sync"
	"time"
)

// Event represents a realtime event sent to subscribers.
// Type examples: "usage", "audit", "heartbeat".
// Data is a JSON-serializable structure specific to the event type.
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"ts"`
	Data      interface{} `json:"data"`
}

// Hub manages per-customer event subscriptions.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Event]struct{}
}

var defaultHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[chan Event]struct{}),
	}
}

// Subscribe registers a new subscriber channel for a given customer ID.
// The returned unsubscribe function must be called to clean up.
func (h *Hub) Subscribe(customerID string) (ch chan Event, unsubscribe func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch = make(chan Event, 100)
	set := h.subscribers[customerID]
	if set == nil {
		set = make(map[chan Event]struct{})
		h.subscribers[customerID] = set
	}
	set[ch] = struct{}{}
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if subs, ok := h.subscribers[customerID]; ok {
			delete(subs, ch)
			close(ch)
			if len(subs) == 0 {
				delete(h.subscribers, customerID)
			}
		}
	}
}

// Publish sends an event to all subscribers of a given customer ID.
func (h *Hub) Publish(customerID string, ev Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if subs, ok := h.subscribers[customerID]; ok {
		for ch := range subs {
			select {
			case ch <- ev:
			default:
				// drop if subscriber is too slow
			}
		}
	}
}

// SubscribeDefault subscribes to the global hub for a customer.
func SubscribeDefault(customerID string) (chan Event, func()) { return defaultHub.Subscribe(customerID) }

// PublishDefault publishes to the global hub for a customer.
func PublishDefault(customerID string, ev Event) { defaultHub.Publish(customerID, ev) }
