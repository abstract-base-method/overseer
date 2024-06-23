package engine

import (
	"context"
	v1 "overseer/build/go"
)

type EventBus interface {
	Submit(ctx context.Context, event *v1.Event) (<-chan *v1.EventReceipt, error)
}

type EventPredicate func(ctx context.Context, event *v1.EventRecord) (bool, error)

type EventHandler interface {
	Name() string
	Predicate() EventPredicate
	Handle(ctx context.Context, event *v1.EventRecord) (<-chan *v1.EventReceipt, error)
}
