package header

import (
	"github.com/seniorGolang/tg/v2/example/transport"
)

const (
	AppHeader = "X-Application-Id"
)

// pass header to log, span & response
func AppName(appName string) (header transport.Header) {

	var value interface{}
	if appName != "" {
		value = appName
	}
	return transport.Header{
		SpanKey:       "applicationName",
		SpanValue:     value,
		LogKey:        "appName",
		LogValue:      value,
		ResponseKey:   AppHeader,
		ResponseValue: value,
	}
}
