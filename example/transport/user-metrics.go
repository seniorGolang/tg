// GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT.
package transport

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/metrics"
	"github.com/DivPro/tg/v2/example/interfaces"
	"github.com/DivPro/tg/v2/example/interfaces/types"
	"time"
)

type metricsUser struct {
	next            interfaces.User
	requestCount    metrics.Counter
	requestCountAll metrics.Counter
	requestLatency  metrics.Histogram
}

func metricsMiddlewareUser(next interfaces.User) interfaces.User {
	return &metricsUser{
		next:            next,
		requestCount:    RequestCount.With("service", "User"),
		requestCountAll: RequestCountAll.With("service", "User"),
		requestLatency:  RequestLatency.With("service", "User"),
	}
}

func (m metricsUser) GetUser(ctx context.Context, cookie string, userAgent string) (user *types.User, err error) {

	defer func(begin time.Time) {
		m.requestLatency.With("method", "getUser", "success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
	}(time.Now())

	defer m.requestCount.With("method", "getUser", "success", fmt.Sprint(err == nil)).Add(1)

	m.requestCountAll.With("method", "getUser").Add(1)

	return m.next.GetUser(ctx, cookie, userAgent)
}

func (m metricsUser) CustomResponse(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {

	defer func(begin time.Time) {
		m.requestLatency.With("method", "customResponse", "success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
	}(time.Now())

	defer m.requestCount.With("method", "customResponse", "success", fmt.Sprint(err == nil)).Add(1)

	m.requestCountAll.With("method", "customResponse").Add(1)

	return m.next.CustomResponse(ctx, arg0, arg1, opts...)
}

func (m metricsUser) CustomHandler(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {

	defer func(begin time.Time) {
		m.requestLatency.With("method", "customHandler", "success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
	}(time.Now())

	defer m.requestCount.With("method", "customHandler", "success", fmt.Sprint(err == nil)).Add(1)

	m.requestCountAll.With("method", "customHandler").Add(1)

	return m.next.CustomHandler(ctx, arg0, arg1, opts...)
}
