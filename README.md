# event-go
[![CI Status](https://github.com/itchyny/event-go/workflows/CI/badge.svg)](https://github.com/itchyny/event-go/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/itchyny/event-go)](https://goreportcard.com/report/github.com/itchyny/event-go)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/itchyny/event-go/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/itchyny/event-go/all.svg)](https://github.com/itchyny/event-go/releases)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/itchyny/event-go)](https://pkg.go.dev/github.com/itchyny/event-go)

### Simple synchronous event pub-sub package for Golang
This is a Go language package for publishing/subscribing domain events.
This is useful to decouple subdomains within applications.
Here is a sketch for using this package in real world applications.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/itchyny/event-go"
)

// Domain entity.
type User struct {
	ID      int64
	Created time.Time
	UserValue
}

// Domain value object.
type UserValue struct {
	Name  string
	Email string
}

// Domain event types.
const (
	EventTypeUserCreated event.Type = iota + 1
	EventTypeUserRetired
)

// Domain events.
type UserCreated struct{ User *User }
type UserRetired struct{ User *User }

// Define the type of the events.
func (ev *UserCreated) Type() event.Type { return EventTypeUserCreated }
func (ev *UserRetired) Type() event.Type { return EventTypeUserRetired }

type App struct{ event.Mapping }

func NewApp() *App {
	app := &App{event.NewMapping()}

	// Register mapping of event types and subscribers.
	app.
		On(
			EventTypeUserCreated,
			event.Func(func(ctx context.Context, ev event.Event) error {
				return app.SendNotification(ctx, NotifyUserCreated, ev.(*UserCreated).User)
			}),
		).
		On(
			EventTypeUserRetired,
			event.Func(func(ctx context.Context, ev event.Event) error {
				return app.SendNotification(ctx, NotifyUserRetired, ev.(*UserRetired).User)
			}),
		)

	return app
}

func (app *App) CreateUser(ctx context.Context, user *UserValue) (*User, error) {
	fmt.Printf("CreateUser: %#v\n", user)
	created := &User{1, time.Now(), *user}
	// Publish a domain event, instead of calling SendNotification directly.
	_ = app.Publish(ctx, &UserCreated{User: created})
	return created, nil
}

func (app *App) RetireUser(ctx context.Context, user *User) error {
	fmt.Printf("RetireUser: %#v\n", user)
	// Publish a domain event, instead of calling SendNotification directly.
	_ = app.Publish(ctx, &UserRetired{User: user})
	return nil
}

type NotificationType int

const (
	NotifyUserCreated NotificationType = iota + 1
	NotifyUserRetired
)

func (typ NotificationType) String() string {
	switch typ {
	case NotifyUserCreated:
		return "created"
	case NotifyUserRetired:
		return "retired"
	default:
		panic(typ)
	}
}

func (app *App) SendNotification(_ context.Context, typ NotificationType, user *User) error {
	fmt.Printf("SendNotification(%s): %#v\n", typ, user)
	// Send email or something.
	return nil
}

func main() {
	ctx := context.Background()
	app := NewApp()
	user, _ := app.CreateUser(ctx, &UserValue{"Test User", "test@example.com"})
	_ = app.RetireUser(ctx, user)
}
```

## Bug Tracker
Report bug at [Issues・itchyny/event-go - GitHub](https://github.com/itchyny/event-go/issues).

## Author
itchyny (https://github.com/itchyny)

## License
This software is released under the MIT License, see LICENSE.
