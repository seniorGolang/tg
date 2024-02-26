// GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT.
package transport

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/metrics"
	"github.com/DivPro/tg/v2/example/interfaces"
	"time"
)

type metricsExampleRPC struct {
	next            interfaces.ExampleRPC
	requestCount    metrics.Counter
	requestCountAll metrics.Counter
	requestLatency  metrics.Histogram
}

func metricsMiddlewareExampleRPC(next interfaces.ExampleRPC) interfaces.ExampleRPC {
	return &metricsExampleRPC{
		next:            next,
		requestCount:    RequestCount.With("service", "ExampleRPC"),
		requestCountAll: RequestCountAll.With("service", "ExampleRPC"),
		requestLatency:  RequestLatency.With("service", "ExampleRPC"),
	}
}

func (m metricsExampleRPC) Test(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (ret1 int, ret2 string, err error) {

	defer func(begin time.Time) {
		m.requestLatency.With("method", "test", "success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
	}(time.Now())

	defer m.requestCount.With("method", "test", "success", fmt.Sprint(err == nil)).Add(1)

	m.requestCountAll.With("method", "test").Add(1)

	return m.next.Test(ctx, arg0, arg1, opts...)
}

func (m metricsExampleRPC) Test2(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (ret1 int, ret2 string, err error) {

	defer func(begin time.Time) {
		m.requestLatency.With("method", "test2", "success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
	}(time.Now())

	defer m.requestCount.With("method", "test2", "success", fmt.Sprint(err == nil)).Add(1)

	m.requestCountAll.With("method", "test2").Add(1)

	return m.next.Test2(ctx, arg0, arg1, opts...)
}
