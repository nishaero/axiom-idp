//go:build ignore

package catalog

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EventBus implements an event bus for service discovery events
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]*Subscriber
	allSubs     []*Subscriber
	logger      *logrus.Logger
}

// Subscriber represents a subscriber to the event bus
type Subscriber struct {
	ID         string
	HandleFunc func(event *Event)
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]*Subscriber),
		logger:      logrus.New(),
	}
}

// SetLogger sets the logger for the event bus
func (e *EventBus) SetLogger(logger *logrus.Logger) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logger = logger
}

// Publish publishes an event to all matching subscribers
func (e *EventBus) Publish(event *Event) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.logger != nil {
		e.logger.WithFields(logrus.Fields{
			"type":    event.Type,
			"timestamp": event.Timestamp,
		}).Debug("Event published")
	}

	// Publish to type-specific subscribers
	if subscribers, exists := e.subscribers[event.Type]; exists {
		for _, sub := range subscribers {
			go func(s *Subscriber) {
				defer func() {
					if r := recover(); r != nil {
						if e.logger != nil {
							e.logger.WithField("event", event.Type).Error("Subscriber panic recovered")
						}
					}
				}()
				s.HandleFunc(event)
			}(sub)
		}
	}

	// Publish to all subscribers
	for _, sub := range e.allSubs {
		go func(s *Subscriber) {
			defer func() {
				if r := recover(); r != nil {
					if e.logger != nil {
						e.logger.Error("Subscriber panic recovered")
					}
				}
			}()
			s.HandleFunc(event)
		}(sub)
	}
}

// Subscribe subscribes a subscriber to a specific event type
func (e *EventBus) Subscribe(eventType EventType, subscriber *Subscriber) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subscribers[eventType] = append(e.subscribers[eventType], subscriber)

	if e.logger != nil {
		e.logger.WithFields(logrus.Fields{
			"subscriber": subscriber.ID,
			"type":       eventType,
		}).Debug("Subscribed to event type")
	}
}

// Unsubscribe removes a subscriber from a specific event type
func (e *EventBus) Unsubscribe(eventType EventType, subscriber *Subscriber) {
	e.mu.Lock()
	defer e.mu.Unlock()

	subscribers := e.subscribers[eventType]
	for i, sub := range subscribers {
		if sub == subscriber {
			e.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			if e.logger != nil {
				e.logger.WithFields(logrus.Fields{
					"subscriber": subscriber.ID,
					"type":       eventType,
				}).Debug("Unsubscribed from event type")
			}
			return
		}
	}
}

// SubscribeAll subscribes a subscriber to all event types
func (e *EventBus) SubscribeAll(subscriber *Subscriber) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.allSubs = append(e.allSubs, subscriber)

	if e.logger != nil {
		e.logger.WithField("subscriber", subscriber.ID).Debug("Subscribed to all event types")
	}
}

// UnsubscribeAll removes a subscriber from all event types
func (e *EventBus) UnsubscribeAll(subscriber *Subscriber) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Remove from all type-specific subscriptions
	for eventType, subscribers := range e.subscribers {
		for i, sub := range subscribers {
			if sub == subscriber {
				e.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			}
		}
	}

	// Remove from all subscribers
	for i, sub := range e.allSubs {
		if sub == subscriber {
			e.allSubs = append(e.allSubs[:i], e.allSubs[i+1:]...)
			break
		}
	}

	if e.logger != nil {
		e.logger.WithField("subscriber", subscriber.ID).Debug("Unsubscribed from all event types")
	}
}

// GetSubscriberCount returns the number of subscribers for an event type
func (e *EventBus) GetSubscriberCount(eventType EventType) int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.subscribers[eventType])
}

// GetAllSubscriberCount returns the total number of subscribers
func (e *EventBus) GetAllSubscriberCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	count := len(e.allSubs)
	for _, subs := range e.subscribers {
		count += len(subs)
	}
	return count
}

// ClearAll removes all subscribers
func (e *EventBus) ClearAll() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subscribers = make(map[EventType][]*Subscriber)
	e.allSubs = make([]*Subscriber, 0)
}

// Subscribe creates a new subscriber with custom handler
func (e *EventBus) Subscribe(eventType EventType, handler func(event *Event)) *Subscriber {
	subscriber := &Subscriber{
		ID:         generateSubscriberID(),
		HandleFunc: handler,
	}
	e.Subscribe(eventType, subscriber)
	return subscriber
}

// SubscribeAll creates a new subscriber for all event types
func (e *EventBus) SubscribeAll(handler func(event *Event)) *Subscriber {
	subscriber := &Subscriber{
		ID:         generateSubscriberID(),
		HandleFunc: handler,
	}
	e.SubscribeAll(subscriber)
	return subscriber
}

// Unsubscribe removes a subscriber
func (e *EventBus) Unsubscribe(subscriber *Subscriber) {
	// Determine which event type the subscriber is subscribed to
	e.mu.RLock()
	for eventType := range e.subscribers {
		for _, sub := range e.subscribers[eventType] {
			if sub == subscriber {
				e.mu.RUnlock()
				e.Unsubscribe(eventType, subscriber)
				return
			}
		}
	}
	// Check all subscribers
	for _, sub := range e.allSubs {
		if sub == subscriber {
			e.mu.RUnlock()
			e.UnsubscribeAll(subscriber)
			return
		}
	}
	e.mu.RUnlock()
}

// generateSubscriberID generates a unique subscriber ID
func generateSubscriberID() string {
	return "sub-" + time.Now().Format("20060102150405.000000000")
}

// EventBusStats holds statistics about the event bus
type EventBusStats struct {
	TotalEvents      int64          `json:"total_events"`
	TotalSubscribers int64          `json:"total_subscribers"`
	EventsByType     map[EventType]int64 `json:"events_by_type"`
	SubscribersByType map[EventType]int64 `json:"subscribers_by_type"`
	StartTime        time.Time      `json:"start_time"`
	LastEventTime    time.Time      `json:"last_event_time"`
}

// GetStats returns statistics about the event bus
func (e *EventBus) GetStats() *EventBusStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &EventBusStats{
		StartTime:       time.Now(),
		EventsByType:    make(map[EventType]int64),
		SubscribersByType: make(map[EventType]int64),
	}

	for eventType, subs := range e.subscribers {
		stats.SubscribersByType[eventType] = int64(len(subs))
	}
	stats.TotalSubscribers = int64(len(e.allSubs))

	return stats
}
