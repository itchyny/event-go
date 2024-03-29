package event_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itchyny/event-go"
)

const (
	eventTypeCreated event.Type = iota
	eventTypeUpdated
	eventTypeDeleted
	eventTypeOther
)

type eventCreated int

func (eventCreated) Type() event.Type {
	return eventTypeCreated
}

type eventUpdated int

func (eventUpdated) Type() event.Type {
	return eventTypeUpdated
}

type eventDeleted int

func (eventDeleted) Type() event.Type {
	return eventTypeDeleted
}

type eventOther int

func (eventOther) Type() event.Type {
	return eventTypeOther
}

type logged []event.Event

func (sub *logged) Handle(_ context.Context, ev event.Event) error {
	*sub = append(*sub, ev)
	return nil
}

func (sub logged) Events() []event.Event {
	return []event.Event(sub)
}

type suberr struct{}

func (sub suberr) Handle(context.Context, event.Event) error {
	return errors.New("handle error")
}

func TestMapping(t *testing.T) {
	ctx := context.Background()
	sub1, sub2, sub3 := &logged{}, &logged{}, &logged{}
	pub := event.NewMapping().
		On(eventTypeCreated, sub1).
		On(eventTypeCreated, sub2).On(eventTypeUpdated, sub2).On(eventTypeDeleted, sub2).
		On(eventTypeCreated, sub3).On(eventTypeUpdated, sub3)
	evs := []event.Event{
		eventCreated(1), eventUpdated(2), eventDeleted(3), eventOther(4),
	}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
	if expected := evs[:1]; !reflect.DeepEqual(sub1.Events(), expected) {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, sub1.Events())
	}
	if expected := evs[:3]; !reflect.DeepEqual(sub2.Events(), expected) {
		t.Errorf("sub2 handled events: expected %v, got %v", expected, sub2.Events())
	}
	if expected := evs[:2]; !reflect.DeepEqual(sub3.Events(), expected) {
		t.Errorf("sub3 handled events: expected %v, got %v", expected, sub3.Events())
	}
}

func TestMappingNested(t *testing.T) {
	ctx := context.Background()
	sub1, sub2, sub3 := &logged{}, &logged{}, &logged{}
	pub := event.NewMapping().
		On(eventTypeCreated,
			event.NewMapping().
				On(eventTypeCreated, sub1).
				On(eventTypeCreated, sub2).
				On(eventTypeUpdated, sub2).
				On(eventTypeDeleted, sub3)).
		On(eventTypeDeleted, sub3)
	evs := []event.Event{
		eventCreated(1), eventUpdated(2), eventDeleted(3), eventOther(4),
	}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
	if expected := evs[:1]; !reflect.DeepEqual(sub1.Events(), expected) {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, sub1.Events())
	}
	if expected := evs[:1]; !reflect.DeepEqual(sub2.Events(), expected) {
		t.Errorf("sub2 handled events: expected %v, got %v", expected, sub2.Events())
	}
	if expected := evs[2:3]; !reflect.DeepEqual(sub3.Events(), expected) {
		t.Errorf("sub3 handled events: expected %v, got %v", expected, sub3.Events())
	}
}

func TestMappingEmpty(t *testing.T) {
	ctx := context.Background()
	pub := event.NewMapping()
	evs := []event.Event{eventOther(0)}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
}

func TestMappingError(t *testing.T) {
	ctx := context.Background()
	sub1, sub2, sub3 := &logged{}, &logged{}, &suberr{}
	pub := event.NewMapping().
		On(eventTypeCreated, sub1).On(eventTypeCreated, sub3).
		On(eventTypeUpdated, sub2).On(eventTypeDeleted, sub2).On(eventTypeCreated, sub2).
		On(eventTypeUpdated, sub3)
	evs := []event.Event{
		eventCreated(1), eventUpdated(2), eventDeleted(3), eventOther(4),
	}
	for _, ev := range evs {
		err := pub.Publish(ctx, ev)
		if ev.Type() == eventTypeCreated || ev.Type() == eventTypeUpdated {
			if expected := "handle error"; err == nil || err.Error() != expected {
				t.Fatalf("expected %v, got %v", expected, err)
			}
		} else {
			if err != nil {
				t.Fatalf("got error: %v", err)
			}
		}
	}
	if expected := evs[:1]; !reflect.DeepEqual(sub1.Events(), expected) {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, sub1.Events())
	}
	if expected := evs[:3]; !reflect.DeepEqual(sub2.Events(), expected) {
		t.Errorf("sub2 handled events: expected %v, got %v", expected, sub2.Events())
	}
}

func TestDiscard(t *testing.T) {
	ctx := context.Background()
	pub := event.NewMapping().
		On(eventTypeCreated, event.Discard)
	evs := []event.Event{eventCreated(1)}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
}

func TestFunc(t *testing.T) {
	ctx := context.Background()
	var handled []event.Event
	pub := event.NewMapping().
		On(eventTypeCreated, event.Func(func(_ context.Context, ev event.Event) error {
			handled = append(handled, ev)
			return nil
		})).
		On(eventTypeUpdated, event.Func(func(context.Context, event.Event) error {
			return errors.New("handle error")
		}))
	evs := []event.Event{eventCreated(1), eventUpdated(2)}
	for _, ev := range evs {
		err := pub.Publish(ctx, ev)
		if ev.Type() == eventTypeCreated {
			if err != nil {
				t.Fatalf("got error: %v", err)
			}
		} else {
			if expected := "handle error"; err == nil || err.Error() != expected {
				t.Fatalf("expected %v, got %v", expected, err)
			}
		}
	}
	if expected := evs[:1]; !reflect.DeepEqual(handled, expected) {
		t.Errorf("handled events: expected %v, got %v", expected, handled)
	}
}

func TestFuncEmpty(t *testing.T) {
	ctx := context.Background()
	pub := event.NewMapping().
		On(eventTypeCreated, event.Func(nil))
	evs := []event.Event{eventCreated(1)}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
}

func TestOrderedEmpty(t *testing.T) {
	ctx := context.Background()
	pub := event.NewMapping().
		On(eventTypeCreated, event.Ordered{})
	evs := []event.Event{eventCreated(1)}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
}

func TestAsync(t *testing.T) {
	ctx := context.Background()
	sub1, sub2, sub3 := &logged{}, &logged{}, &logged{}
	pub := event.NewMapping().
		On(eventTypeCreated, event.Async{sub1, sub2, sub3})
	evs := []event.Event{eventCreated(1)}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
	if expected := evs[:]; !reflect.DeepEqual(sub1.Events(), expected) {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, sub1.Events())
	}
	if expected := evs[:]; !reflect.DeepEqual(sub2.Events(), expected) {
		t.Errorf("sub2 handled events: expected %v, got %v", expected, sub2.Events())
	}
	if expected := evs[:]; !reflect.DeepEqual(sub3.Events(), expected) {
		t.Errorf("sub3 handled events: expected %v, got %v", expected, sub3.Events())
	}
}

func TestAsyncEmpty(t *testing.T) {
	ctx := context.Background()
	pub := event.NewMapping().
		On(eventTypeCreated, event.Async{})
	evs := []event.Event{eventCreated(1)}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
}

func TestAsyncError(t *testing.T) {
	ctx := context.Background()
	sub1, sub2, sub3 := &logged{}, &logged{}, &suberr{}
	pub := event.NewMapping().
		On(eventTypeCreated, event.Async{sub1, event.Ordered{sub2, sub3, sub2}, sub3})
	evs := []event.Event{eventCreated(1), eventCreated(2)}
	for _, ev := range evs {
		if err, expected := pub.Publish(ctx, ev), "handle error"; err == nil || err.Error() != expected {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	}
	if expected := evs[:]; !reflect.DeepEqual(sub1.Events(), expected) {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, sub1.Events())
	}
	if expected := []event.Event{evs[0], evs[0], evs[1], evs[1]}; !reflect.DeepEqual(sub2.Events(), expected) {
		t.Errorf("sub2 handled events: expected %v, got %v", expected, sub2.Events())
	}
}

func TestLimited(t *testing.T) {
	ctx := context.Background()
	const max = 3
	var running, handled int32
	sub1 := event.NewLimited(
		event.Func(func(context.Context, event.Event) error {
			if n := atomic.AddInt32(&running, 1); n > max {
				t.Errorf("expected running max %d concurrency, got %d", max, n)
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&running, -1)
			atomic.AddInt32(&handled, 1)
			return nil
		}),
		max,
	)
	pub := event.NewMapping().
		On(eventTypeCreated, event.Async{sub1, sub1, sub1, sub1, sub1})
	if err := pub.Publish(ctx, eventCreated(1)); err != nil {
		t.Fatalf("got error: %v", err)
	}
	if expected := int32(5); handled != expected {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, handled)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()
	err, expected := pub.Publish(ctx, eventCreated(2)), context.DeadlineExceeded
	if err == nil || err != expected {
		t.Fatalf("expected %v, got %v", expected, err)
	}
	if expected := int32(5 + max); handled != expected {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, handled)
	}
}

func TestBuffer(t *testing.T) {
	ctx := context.Background()
	sub1, sub2 := &logged{}, &logged{}
	var pub *event.Buffer
	pub = event.NewBuffer(
		event.NewMapping().
			On(eventTypeCreated, sub1).
			On(eventTypeCreated, sub2).
			On(eventTypeUpdated, sub2).
			On(eventTypeOther, sub2).
			On(eventTypeUpdated, event.Func(func(ctx context.Context, ev event.Event) error {
				if int(ev.(eventUpdated)) == 3 {
					return errors.New("handle error")
				}
				return pub.Publish(ctx, eventOther(3))
			})),
	)
	evs := []event.Event{eventCreated(1), eventUpdated(2)}
	for _, ev := range evs {
		if err := pub.Publish(ctx, ev); err != nil {
			t.Fatalf("got error: %v", err)
		}
	}
	if expected := evs[:0]; !reflect.DeepEqual(sub1.Events(), expected) {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, sub1.Events())
	}
	if expected := evs[:0]; !reflect.DeepEqual(sub2.Events(), expected) {
		t.Errorf("sub2 handled events: expected %v, got %v", expected, sub2.Events())
	}
	if err := pub.Dispatch(ctx); err != nil {
		t.Fatalf("got error: %v", err)
	}
	if expected := evs[:1]; !reflect.DeepEqual(sub1.Events(), expected) {
		t.Errorf("sub1 handled events: expected %v, got %v", expected, sub1.Events())
	}
	if expected := append(evs, eventOther(3)); !reflect.DeepEqual(sub2.Events(), expected) {
		t.Errorf("sub2 handled events: expected %v, got %v", expected, sub2.Events())
	}
	if err := pub.Handle(ctx, eventUpdated(3)); err != nil {
		t.Fatalf("got error: %v", err)
	}
	if err, expected := pub.Dispatch(ctx), "handle error"; err == nil || err.Error() != expected {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
