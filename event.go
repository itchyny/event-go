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

// Mapping is an event publisher for mapping event types and subscribers.
type Mapping map[Type][]Subscriber

// NewMapping creates a new event mapping publisher.
func NewMapping() Mapping {
	return make(Mapping)
}

// On registers the subscriber to listen on the event. This method returns the
// publisher to allow method chaining. Note that this method is not goroutine
// safe so register all the subscribers before start event publishing.
func (pub Mapping) On(typ Type, sub Subscriber) Mapping {
	if _, ok := pub[typ]; !ok {
		pub[typ] = []Subscriber{}
	}
	pub[typ] = append(pub[typ], sub)
	return pub
}

// Handle implements Subscriber for Mapping.
func (pub Mapping) Handle(ctx context.Context, ev Event) error {
	return pub.Publish(ctx, ev)
}

// Publish implements Publisher for Mapping.
func (pub Mapping) Publish(ctx context.Context, ev Event) error {
	if subs, ok := pub[ev.Type()]; ok {
		for _, sub := range subs {
			if err := sub.Handle(ctx, ev); err != nil {
				return err
			}
		}
	}
	return nil
}
