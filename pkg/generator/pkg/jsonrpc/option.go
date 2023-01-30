package jsonrpc

type options struct {
	insecure           bool
	logOnError         bool
	logRequests        bool
	allowUnknownFields bool
	headersFromCtx     []interface{}
	customHeaders      map[string]string
}

type Option func(ops *options)

func prepareOpts(opts []Option) (options options) {

	options.customHeaders = make(map[string]string)
	for _, op := range opts {
		op(&options)
	}
	return
}

func HeaderFromCtx(headers ...interface{}) Option {
	return func(ops *options) {
		ops.headersFromCtx = headers
	}
}

func AllowUnknownFields(allowUnknownFields bool) Option {
	return func(ops *options) {
		ops.allowUnknownFields = allowUnknownFields
	}
}

func Insecure() Option {
	return func(ops *options) {
		ops.insecure = true
	}
}

func LogRequest() Option {
	return func(ops *options) {
		ops.logRequests = true
	}
}

func LogOnError() Option {
	return func(ops *options) {
		ops.logOnError = true
	}
}
