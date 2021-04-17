package event

import (
	"context"
	"sync"
)

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

// Ordered is an event subscriber to publish in specified order of subscribers.
type Ordered []Subscriber

// Handle implements Subscriber for Ordered.
func (sub Ordered) Handle(ctx context.Context, ev Event) error {
	for _, sub := range sub {
		if err := sub.Handle(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}

// Async is an event subscriber to publish events asynchronously.
type Async []Subscriber

// Handle implements Subscriber for Async.
func (sub Async) Handle(ctx context.Context, ev Event) error {
	var (
		wg   sync.WaitGroup
		once sync.Once
		err  error
	)
	wg.Add(len(sub))
	for _, sub := range sub {
		go func(sub Subscriber) {
			defer wg.Done()
			if e := sub.Handle(ctx, ev); e != nil {
				once.Do(func() { err = e })
			}
		}(sub)
	}
	wg.Wait()
	return err
}

// Mapping is an event publisher for mapping event types and subscribers.
type Mapping map[Type]Subscriber

// NewMapping creates a new event mapping publisher.
func NewMapping() Mapping {
	return make(Mapping)
}

// On registers the subscriber to listen on the event. This method returns the
// publisher to allow method chaining. Note that this method is not goroutine
// safe so register all the subscribers before start event publishing.
func (pub Mapping) On(typ Type, sub Subscriber) Mapping {
	if s, ok := pub[typ]; ok {
		if o, ok := s.(Ordered); ok {
			pub[typ] = append(o, sub)
		} else {
			pub[typ] = Ordered{s, sub}
		}
	} else {
		pub[typ] = sub
	}
	return pub
}

// Handle implements Subscriber for Mapping.
func (pub Mapping) Handle(ctx context.Context, ev Event) error {
	return pub.Publish(ctx, ev)
}

// Publish implements Publisher for Mapping.
func (pub Mapping) Publish(ctx context.Context, ev Event) error {
	if sub, ok := pub[ev.Type()]; ok {
		if err := sub.Handle(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}
