package viewer

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

const (
	dumpMethod     = "Dump"
	supportedFlags = "0-+# "
)

type formatState struct {
	value          interface{}
	fs             fmt.State
	depth          int
	pointers       map[uintptr]int
	ignoreNextType bool
	cs             *ConfigState
}

func (f *formatState) buildDefaultFormat() (format string) {

	buf := bytes.NewBuffer(percentBytes)
	for _, flag := range supportedFlags {
		if f.fs.Flag(int(flag)) {
			buf.WriteRune(flag)
		}
	}
	buf.WriteRune('v')
	format = buf.String()
	return format
}

func (f *formatState) constructOrigFormat(verb rune) (format string) {

	buf := bytes.NewBuffer(percentBytes)
	for _, flag := range supportedFlags {
		if f.fs.Flag(int(flag)) {
			buf.WriteRune(flag)
		}
	}
	if width, ok := f.fs.Width(); ok {
		buf.WriteString(strconv.Itoa(width))
	}
	if precision, ok := f.fs.Precision(); ok {
		buf.Write(precisionBytes)
		buf.WriteString(strconv.Itoa(precision))
	}
	buf.WriteRune(verb)
	format = buf.String()
	return format
}

func (f *formatState) unpackValue(v reflect.Value) reflect.Value {

	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		f.ignoreNextType = false
		if !v.IsNil() {
			v = v.Elem()
		}
	}
	return v
}

func (f *formatState) formatPtr(v reflect.Value) {

	showTypes := f.fs.Flag('#')
	if v.IsNil() && (!showTypes || f.ignoreNextType) {
		_, _ = f.fs.Write(nilAngleBytes)
		return
	}
	for k, depth := range f.pointers {
		if depth >= f.depth {
			delete(f.pointers, k)
		}
	}
	ve := v
	indirect := 0
	nilFound := false
	cycleFound := false

	for ve.Kind() == reflect.Ptr {
		if ve.IsNil() {
			nilFound = true
			break
		}
		indirect++
		addr := ve.Pointer()
		if pd, ok := f.pointers[addr]; ok && pd < f.depth {
			cycleFound = true
			indirect--
			break
		}
		ve = ve.Elem()
		f.pointers[addr] = f.depth
		if ve.Kind() == reflect.Interface {
			if ve.IsNil() {
				nilFound = true
				break
			}
			ve = ve.Elem()
		}
	}
	if showTypes && !f.ignoreNextType {
		_, _ = f.fs.Write(openParenBytes)
		_, _ = f.fs.Write(bytes.Repeat(asteriskBytes, indirect))
		_, _ = f.fs.Write([]byte(ve.Type().String()))
		_, _ = f.fs.Write(closeParenBytes)
	}
	switch {
	case nilFound:
		_, _ = f.fs.Write(nilAngleBytes)

	case cycleFound:
		_, _ = f.fs.Write(circularShortBytes)

	default:
		f.ignoreNextType = true
		f.format(ve)
	}
}

func (f *formatState) format(v reflect.Value, opts ...option) {

	if toString := v.MethodByName("String"); toString.IsValid() {
		if v.Kind() == reflect.Ptr && v.IsNil() {
			_, _ = f.fs.Write(applyOptions([]byte("nil")))
			return
		}
		if values := toString.Call(nil); len(values) == 1 {
			_, _ = f.fs.Write(applyOptions([]byte(values[0].String()), opts...))
			return
		}
	}

	kind := v.Kind()
	if kind == reflect.Invalid {
		_, _ = f.fs.Write(invalidAngleBytes)
		return
	}

	if kind == reflect.Ptr {
		f.formatPtr(v)
		return
	}

	if !f.ignoreNextType && f.fs.Flag('#') {
		_, _ = f.fs.Write(openParenBytes)
		_, _ = f.fs.Write([]byte(v.Type().String()))
		_, _ = f.fs.Write(closeParenBytes)
	}
	f.ignoreNextType = false

	if method := v.MethodByName(dumpMethod); method.IsValid() {
		if results := method.Call([]reflect.Value{}); len(results) == 1 {
			_, _ = f.fs.Write([]byte(results[0].String()))
			return
		}
	}
	switch kind {
	case reflect.Invalid:

	case reflect.Bool:
		printBool(f.fs, v.Bool())

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		_, _ = f.fs.Write(applyOptions(intBytes(v.Int(), 10), opts...))

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		_, _ = f.fs.Write(applyOptions(uintBytes(v.Uint(), 10), opts...))

	case reflect.Float32:
		_, _ = f.fs.Write(applyOptions(floatBytes(v.Float(), 32), opts...))

	case reflect.Float64:
		_, _ = f.fs.Write(applyOptions(floatBytes(v.Float(), 64), opts...))

	case reflect.Complex64:
		printComplex(f.fs, v.Complex(), 32)

	case reflect.Complex128:
		printComplex(f.fs, v.Complex(), 64)

	case reflect.Slice:
		if v.IsNil() {
			_, _ = f.fs.Write(nilAngleBytes)
			break
		}
		fallthrough

	case reflect.Array:
		_, _ = f.fs.Write(openBracketBytes)
		f.depth++
		if (f.cs.MaxDepth != 0) && (f.depth > f.cs.MaxDepth) {
			_, _ = f.fs.Write(maxShortBytes)
		} else {
			numEntries := v.Len()

			if numEntries > 16 {
				for i := 0; i < 4; i++ {
					if i > 0 {
						_, _ = f.fs.Write(spaceBytes)
					}
					f.ignoreNextType = true
					f.format(f.unpackValue(v.Index(i)))
				}
				f.format(reflect.ValueOf(fmt.Sprintf(" <-[%d]->", numEntries)))
				for i := numEntries - 4; i < numEntries; i++ {
					if i > 0 {
						_, _ = f.fs.Write(spaceBytes)
					}
					f.ignoreNextType = true
					f.format(f.unpackValue(v.Index(i)))
				}
				break
			}
			for i := 0; i < numEntries; i++ {
				if i > 0 {
					_, _ = f.fs.Write(spaceBytes)
				}
				f.ignoreNextType = true
				f.format(f.unpackValue(v.Index(i)))
			}
		}
		f.depth--
		_, _ = f.fs.Write(closeBracketBytes)

	case reflect.String:
		_, _ = f.fs.Write(applyOptions([]byte(v.String()), opts...))

	case reflect.Interface:

		if v.IsNil() {
			_, _ = f.fs.Write(nilAngleBytes)
		}

	case reflect.Ptr:
		f.format(v.Elem(), opts...)
	case reflect.Map:

		if v.IsNil() {
			_, _ = f.fs.Write(nilAngleBytes)
			break
		}
		_, _ = f.fs.Write(openMapBytes)
		f.depth++
		if (f.cs.MaxDepth != 0) && (f.depth > f.cs.MaxDepth) {
			_, _ = f.fs.Write(maxShortBytes)
		} else {
			keys := v.MapKeys()
			for i, key := range keys {
				if i > 0 {
					_, _ = f.fs.Write(spaceBytes)
				}
				f.ignoreNextType = true
				f.format(f.unpackValue(key))
				_, _ = f.fs.Write(colonBytes)
				f.ignoreNextType = true
				f.format(f.unpackValue(v.MapIndex(key)))
			}
		}
		f.depth--
		_, _ = f.fs.Write(closeMapBytes)

	case reflect.Struct:

		if v.Type() == reflect.TypeOf(time.Time{}) {
			_, _ = f.fs.Write([]byte(v.Interface().(time.Time).Format(time.RFC3339)))
			break
		}
		numFields := v.NumField()
		_, _ = f.fs.Write(openBraceBytes)
		f.depth++
		if (f.cs.MaxDepth != 0) && (f.depth > f.cs.MaxDepth) {
			_, _ = f.fs.Write(maxShortBytes)
		} else {
			vt := v.Type()
			for i := 0; i < numFields; i++ {
				if i > 0 {
					_, _ = f.fs.Write(spaceBytes)
				}
				vtf := vt.Field(i)
				if f.fs.Flag('+') || f.fs.Flag('#') {
					_, _ = f.fs.Write([]byte(vtf.Name))
					_, _ = f.fs.Write(colonBytes)
				}
				f.format(f.unpackValue(v.Field(i)), tagToOption(vtf.Tag.Get(tagName)))
			}
		}
		f.depth--
		_, _ = f.fs.Write(closeBraceBytes)

	case reflect.Uintptr:
		printHexPtr(f.fs, uintptr(v.Uint()))

	case reflect.UnsafePointer, reflect.Chan, reflect.Func:
		printHexPtr(f.fs, v.Pointer())

	default:
		format := f.buildDefaultFormat()
		if v.CanInterface() {
			_, _ = fmt.Fprintf(f.fs, format, v.Interface())
		} else {
			_, _ = fmt.Fprintf(f.fs, format, v.String())
		}
	}
}

func (f *formatState) Format(fs fmt.State, verb rune) {

	f.fs = fs
	if verb != 'v' {
		format := f.constructOrigFormat(verb)
		_, _ = fmt.Fprintf(fs, format, f.value)
		return
	}
	if f.value == nil {
		if fs.Flag('#') {
			_, _ = fs.Write(interfaceBytes)
		}
		_, _ = fs.Write(nilAngleBytes)
		return
	}
	f.format(reflect.ValueOf(f.value))
}
