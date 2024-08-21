package context

import (
	"context"
	"reflect"
)

type contextKey string
type Context = context.Context

var TODO = context.TODO
var Canceled = context.Canceled
var Background = context.Background

func WithCtx[T any](ctx context.Context, value T) context.Context {
	return context.WithValue(ctx, contextKey(reflect.TypeOf(value).String()), value)
}

func FromCtx[T any](ctx context.Context, defaults ...T) (value T) {

	var ok bool
	if value, ok = ctx.Value(contextKey(reflect.TypeOf(value).String())).(T); !ok {
		if len(defaults) != 0 {
			value = defaults[0]
		}
	}
	return
}
