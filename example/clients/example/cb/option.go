package cb

type Option func(ops *options)

type options struct {
	fallback     func(err error) error
	isSuccessful func(err error) (success bool)
}

func prepareOpts(opts []Option) (options options) {

	for _, op := range opts {
		op(&options)
	}
	return
}

func IsSuccessful(isSuccessful func(err error) (success bool)) Option {
	return func(ops *options) {
		ops.isSuccessful = isSuccessful
	}
}

func Fallback(fallback func(err error) error) Option {
	return func(ops *options) {
		ops.fallback = fallback
	}
}
