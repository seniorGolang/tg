package hasher

import (
	"encoding/binary"
	"fmt"
	"hash"
	"reflect"
	"time"
)

type visitFlag uint

const (
	visitFlagSet = iota << 1
)

type walker struct {
	h               hash.Hash64
	tag             string
	sets            bool
	zeroNil         bool
	stringer        bool
	ignoreZeroValue bool
}

type visitOpts struct {
	Flags       visitFlag
	Struct      interface{}
	StructField string
}

func (w *walker) visit(v reflect.Value, opts *visitOpts) (hash uint64, err error) {

	t := reflect.TypeOf(0)
	for {
		if v.Kind() == reflect.Interface {
			v = v.Elem()
			continue
		}
		if v.Kind() == reflect.Ptr {
			if w.zeroNil {
				t = v.Type().Elem()
			}
			v = reflect.Indirect(v)
			continue
		}
		break
	}
	if !v.IsValid() {
		v = reflect.Zero(t)
	}
	switch v.Kind() {
	case reflect.Int:
		v = reflect.ValueOf(v.Int())
	case reflect.Uint:
		v = reflect.ValueOf(v.Uint())
	case reflect.Bool:
		var tmp int8
		if v.Bool() {
			tmp = 1
		}
		v = reflect.ValueOf(tmp)
	}
	k := v.Kind()
	if k >= reflect.Int && k <= reflect.Complex64 {
		w.h.Reset()
		_ = binary.Write(w.h, binary.LittleEndian, v.Interface())
		return w.h.Sum64(), err
	}
	switch v.Type() {
	case timeType:
		w.h.Reset()
		b, err := v.Interface().(time.Time).MarshalBinary()
		if err != nil {
			return 0, err
		}
		err = binary.Write(w.h, binary.LittleEndian, b)
		return w.h.Sum64(), err
	}
	switch k {
	case reflect.Array:
		var h uint64
		l := v.Len()
		for i := 0; i < l; i++ {
			current, err := w.visit(v.Index(i), nil)
			if err != nil {
				return 0, err
			}
			h = hashUpdateOrdered(w.h, h, current)
		}
		return h, nil
	case reflect.Map:
		var includeMap IncMap
		if opts != nil && opts.Struct != nil {
			if v, ok := opts.Struct.(IncMap); ok {
				includeMap = v
			}
		}
		var h uint64
		for _, k := range v.MapKeys() {
			v := v.MapIndex(k)
			if includeMap != nil {
				incl, err := includeMap.HashIncludeMap(
					opts.StructField, k.Interface(), v.Interface())
				if err != nil {
					return 0, err
				}
				if !incl {
					continue
				}
			}
			kh, err := w.visit(k, nil)
			if err != nil {
				return 0, err
			}
			vh, err := w.visit(v, nil)
			if err != nil {
				return 0, err
			}
			fieldHash := hashUpdateOrdered(w.h, kh, vh)
			h = hashUpdateUnordered(h, fieldHash)
		}
		h = hashFinishUnordered(w.h, h)
		return h, nil
	case reflect.Struct:
		parent := v.Interface()
		var include Inc
		if impl, ok := parent.(Inc); ok {
			include = impl
		}
		if impl, ok := parent.(Hasher); ok {
			return impl.Hash()
		}
		if v.CanAddr() {
			vptr := v.Addr()
			parentptr := vptr.Interface()
			if impl, ok := parentptr.(Inc); ok {
				include = impl
			}
			if impl, ok := parentptr.(Hasher); ok {
				return impl.Hash()
			}
		}
		t := v.Type()
		h, err := w.visit(reflect.ValueOf(t.Name()), nil)
		if err != nil {
			return 0, err
		}
		l := v.NumField()
		for i := 0; i < l; i++ {
			if innerV := v.Field(i); v.CanSet() || t.Field(i).Name != "_" {
				var f visitFlag
				fieldType := t.Field(i)
				if fieldType.PkgPath != "" {
					continue
				}
				tag := fieldType.Tag.Get(w.tag)
				if tag == "ignore" || tag == "-" {
					continue
				}
				if w.ignoreZeroValue {
					if innerV.IsZero() {
						continue
					}
				}
				if tag == "string" || w.stringer {
					if impl, ok := innerV.Interface().(fmt.Stringer); ok {
						innerV = reflect.ValueOf(impl.String())
					} else if tag == "string" {
						return 0, &ErrNotStringer{
							Field: v.Type().Field(i).Name,
						}
					}
				}
				if include != nil {
					incl, err := include.HashInclude(fieldType.Name, innerV)
					if err != nil {
						return 0, err
					}
					if !incl {
						continue
					}
				}
				switch tag {
				case "set":
					f |= visitFlagSet
				}
				kh, err := w.visit(reflect.ValueOf(fieldType.Name), nil)
				if err != nil {
					return 0, err
				}
				vh, err := w.visit(innerV, &visitOpts{
					Flags:       f,
					Struct:      parent,
					StructField: fieldType.Name,
				})
				if err != nil {
					return 0, err
				}
				fieldHash := hashUpdateOrdered(w.h, kh, vh)
				h = hashUpdateUnordered(h, fieldHash)
			}
			h = hashFinishUnordered(w.h, h)
		}
		return h, nil
	case reflect.Slice:
		var h uint64
		var set bool
		if opts != nil {
			set = (opts.Flags & visitFlagSet) != 0
		}
		l := v.Len()
		for i := 0; i < l; i++ {
			current, err := w.visit(v.Index(i), nil)
			if err != nil {
				return 0, err
			}
			if set || w.sets {
				h = hashUpdateUnordered(h, current)
			} else {
				h = hashUpdateOrdered(w.h, h, current)
			}
		}
		h = hashFinishUnordered(w.h, h)
		return h, nil
	case reflect.String:
		w.h.Reset()
		_, err := w.h.Write([]byte(v.String()))
		return w.h.Sum64(), err
	default:
		return 0, fmt.Errorf("unknown kind to hash: %s", k)
	}
}

func hashUpdateOrdered(h hash.Hash64, a, b uint64) uint64 {

	h.Reset()
	e1 := binary.Write(h, binary.LittleEndian, a)
	e2 := binary.Write(h, binary.LittleEndian, b)
	if e1 != nil {
		panic(e1)
	}
	if e2 != nil {
		panic(e2)
	}
	return h.Sum64()
}

func hashUpdateUnordered(a, b uint64) uint64 {
	return a ^ b
}

func hashFinishUnordered(h hash.Hash64, a uint64) uint64 {

	h.Reset()
	e1 := binary.Write(h, binary.LittleEndian, a)
	if e1 != nil {
		panic(e1)
	}
	return h.Sum64()
}
