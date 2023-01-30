package hasher

import (
	"hash/fnv"
	"reflect"
	"time"
)

const (
	hashTag = "hash"
)

var timeType = reflect.TypeOf(time.Time{})

func Hash(v interface{}, opts ...Option) (hash uint64, err error) {

	values := prepareOpts(opts)
	w := &walker{
		tag:             hashTag,
		h:               fnv.New64(),
		zeroNil:         values.zeroNil,
		stringer:        values.useStringer,
		sets:            values.slicesAsSets,
		ignoreZeroValue: values.ignoreZeroValue,
	}
	w.h.Reset()
	return w.visit(reflect.ValueOf(v), nil)
}
