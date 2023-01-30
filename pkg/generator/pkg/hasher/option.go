package hasher

type options struct {
	zeroNil         bool
	useStringer     bool
	slicesAsSets    bool
	ignoreZeroValue bool
}

type Option func(ops *options)

func prepareOpts(opts []Option) (options options) {

	for _, op := range opts {
		op(&options)
	}
	return
}

func ZeroNil() Option {
	return func(ops *options) {
		ops.zeroNil = true
	}
}

func UseStringer() Option {
	return func(ops *options) {
		ops.useStringer = true
	}
}

func SlicesAsSets() Option {
	return func(ops *options) {
		ops.slicesAsSets = true
	}
}

func IgnoreZeroValue() Option {
	return func(ops *options) {
		ops.ignoreZeroValue = true
	}
}
