package event

import "context"

// Type is the event type. The underlying type is int to define nonduplicate
// event types with iota and to quickly selecting the subscribers on a new event.
type Type int

// Event is the interface for an event. The Type() method is used to
// select the registered subscribers.
type Event interface {
	Type() Type
}

// Subscriber is the interface for an event subscriber.
type Subscriber interface {
	// Handle an event.
	Handle(context.Context, Event) error
}

// Publisher is the interface for an event publisher.
type Publisher interface {
	// A publisher is a subscriber.
	Subscriber
	// Publish an event.
	Publish(context.Context, Event) error
}
