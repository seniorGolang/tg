// GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT.
package transport

import (
	"context"
	"fmt"
	"github.com/seniorGolang/tg/v2/example/interfaces"
	"github.com/seniorGolang/tg/v2/example/interfaces/types"
	otel "go.opentelemetry.io/otel"
	trace "go.opentelemetry.io/otel/trace"
)

type traceUser struct {
	next interfaces.User
}

func traceMiddlewareUser(next interfaces.User) interfaces.User {
	return &traceUser{next: next}
}

func (svc traceUser) GetUser(ctx context.Context, cookie string, userAgent string) (user *types.User, err error) {

	var span trace.Span
	ctx, span = otel.Tracer(fmt.Sprintf("tg:%s", VersionTg)).Start(ctx, "user.getUser")
	defer func() {
		span.RecordError(err)
		span.End()
	}()
	return svc.next.GetUser(ctx, cookie, userAgent)
}

func (svc traceUser) CustomResponse(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {

	var span trace.Span
	ctx, span = otel.Tracer(fmt.Sprintf("tg:%s", VersionTg)).Start(ctx, "user.customResponse")
	defer func() {
		span.RecordError(err)
		span.End()
	}()
	return svc.next.CustomResponse(ctx, arg0, arg1, opts...)
}

func (svc traceUser) CustomHandler(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {

	var span trace.Span
	ctx, span = otel.Tracer(fmt.Sprintf("tg:%s", VersionTg)).Start(ctx, "user.customHandler")
	defer func() {
		span.RecordError(err)
		span.End()
	}()
	return svc.next.CustomHandler(ctx, arg0, arg1, opts...)
}
