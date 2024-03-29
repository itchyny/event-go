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

// Discard is an event subscriber which ignores the event.
var Discard Func

// Func is an event subscriber built from a function.
type Func func(context.Context, Event) error

// Handle implements Subscriber for Func.
func (sub Func) Handle(ctx context.Context, ev Event) error {
	if sub == nil {
		return nil
	}
	return sub(ctx, ev)
}

// Ordered is an event subscriber to handle in specified order of subscribers.
type Ordered []Subscriber

// Handle implements Subscriber for Ordered.
func (sub Ordered) Handle(ctx context.Context, ev Event) error {
	var err error
	for _, sub := range sub {
		if e := sub.Handle(ctx, ev); e != nil {
			err = e
		}
	}
	return err
}

// Async is an event subscriber to handle asynchronously between subscribers.
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

// Limited is an event subscriber to limit the max concurrency of subscriber.
type Limited struct {
	subscriber Subscriber
	sem        chan struct{}
}

// NewLimited creates a new limited subscriber.
func NewLimited(sub Subscriber, max int) *Limited {
	return &Limited{sub, make(chan struct{}, max)}
}

// Handle implements Subscriber for Limited.
func (sub *Limited) Handle(ctx context.Context, ev Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case sub.sem <- struct{}{}:
		defer func() { <-sub.sem }()
		return sub.subscriber.Handle(ctx, ev)
	}
}

// Mapping is an event publisher for mapping event types and subscribers.
type Mapping map[Type]Subscriber

// NewMapping creates a new event mapping publisher.
func NewMapping() Mapping {
	return make(Mapping)
}

// On registers the subscriber to listen on the event. This method returns the
// publisher to allow method chaining. Note that this method is not goroutine
// safe so register all the subscribers before starting event publishing.
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
		return sub.Handle(ctx, ev)
	}
	return nil
}

// Buffer is an event publisher for delaying event dispatching. This is useful
// for buffering all the events during a transaction and dispatching them only
// after the transaction succeeded. This publisher is not goroutine safe, so
// create a new buffered publisher each request.
type Buffer struct {
	publisher Publisher
	events    []Event
}

// NewBuffer creates a new event buffered publisher.
func NewBuffer(pub Publisher) *Buffer {
	return &Buffer{publisher: pub}
}

// Handle implements Subscriber for Buffer.
func (pub *Buffer) Handle(ctx context.Context, ev Event) error {
	return pub.Publish(ctx, ev)
}

// Publish implements Publisher for Buffer.
func (pub *Buffer) Publish(_ context.Context, ev Event) error {
	pub.events = append(pub.events, ev)
	return nil
}

// Dispatch all the buffered events.
func (pub *Buffer) Dispatch(ctx context.Context) error {
	var (
		ev  Event
		err error
	)
	for len(pub.events) != 0 {
		ev, pub.events = pub.events[0], pub.events[1:]
		if e := pub.publisher.Publish(ctx, ev); e != nil {
			err = e
		}
	}
	return err
}
