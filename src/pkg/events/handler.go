package events

import "context"

type Handler interface {
	HandleEvent(ctx context.Context, event Event) error
}

type HandlerFunc func(ctx context.Context, event Event) error

func (f HandlerFunc) HandleEvent(ctx context.Context, event Event) error {
	return f(ctx, event)
}
