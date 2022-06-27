// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (tagScanner.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package tags

import (
	"errors"
	"fmt"
)

var ErrUnterminatedString = errors.New("unterminated string")

func TagScanner(tagString string) (tags map[string]string, err error) {

	saveError := func(e error) {
		if err == nil {
			err = e
		}
	}

	var i int
	var m int
	var c byte
	var ok bool
	var esc bool
	var key []byte
	var val []byte

	data := []byte(tagString)
	tags = make(map[string]string)

garbage:
	if i == len(data) {
		return
	}

	c = data[i]
	switch {
	case c > ' ' && c != '`' && c != '=':
		key, val = nil, nil // nolint
		m = i
		i++
		goto key
	default:
		i++
		goto garbage
	}

key:
	if i >= len(data) {
		if m >= 0 {
			key = data[m:i]
			tags[string(key)] = ""
		}
		return
	}

	c = data[i]
	switch {
	case c > ' ' && c != '`' && c != '=':
		i++
		goto key
	case c == '=':
		key = data[m:i]
		i++
		goto equal
	default:
		key = data[m:i]
		i++
		tags[string(key)] = ""
		goto garbage
	}

equal:
	if i >= len(data) {
		if m >= 0 {
			i--
			key = data[m:i]
			tags[string(key)] = ""
		}
		return
	}

	c = data[i]
	switch {
	case c > ' ' && c != '`' && c != '=':
		m = i
		i++
		goto ivalue
	case c == '`':
		m = i
		i++
		esc = false
		goto qvalue
	default:
		if key != nil {
			tags[string(key)] = string(val)
		}
		i++
		goto garbage
	}

ivalue:
	if i >= len(data) {
		if m >= 0 {
			val = data[m:i]
			tags[string(key)] = string(val)
		}
		return
	}

	c = data[i]
	switch {
	case c > ' ' && c != '`' && c != '=':
		i++
		goto ivalue
	default:
		val = data[m:i]
		tags[string(key)] = string(val)
		i++
		goto garbage
	}

qvalue:
	if i >= len(data) {
		if m >= 0 {
			saveError(ErrUnterminatedString)
		}
		return
	}

	c = data[i]
	switch c {
	case '\\':
		i += 2
		esc = true
		goto qvalue
	case '`':
		i++
		val = data[m:i]
		if esc {
			val, ok = unquoteBytes(val)
			if !ok {
				saveError(fmt.Errorf("error unquoting bytes %q", string(val)))
				goto garbage
			}
		} else {
			val = val[1 : len(val)-1]
		}
		tags[string(key)] = string(val)
		goto garbage
	default:
		i++
		goto qvalue
	}
}
