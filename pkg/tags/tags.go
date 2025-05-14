// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (tags.go at 14.05.2020, 3:45) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package tags

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/seniorGolang/tg/v2/pkg/utils"
)

const (
	mark = "@tg"
)

type DocTags map[string]string

func (tags DocTags) MarshalJSON() (bytes []byte, err error) {

	if len(tags) == 0 {
		return json.Marshal(nil)
	}
	return json.Marshal(map[string]string(tags))
}

func (tags DocTags) Merge(t DocTags) DocTags {

	if tags == nil {
		tags = make(DocTags)
	}

	for k, v := range t {
		tags[k] = v
	}
	return tags
}

func ParseTags(docs []string) (tags DocTags) {

	tags = make(DocTags)

	textLines := make(map[string][]string)

	for _, doc := range docs {

		doc = strings.TrimSpace(strings.TrimPrefix(doc, "//"))

		if strings.HasPrefix(doc, mark) {

			values, _ := TagScanner(doc[len(mark):])

			for k, v := range values {

				if _, found := tags[k]; found {
					tags[k] += "," + v
				} else {
					tags[k] = v
				}
			}
		}
	}

	for key, value := range tags {

		if !strings.HasPrefix(value, "#") {
			continue
		}

		for textKey, text := range textLines {
			if value == textKey {
				tags[key] = strings.Join(text, "\n")
			}
		}
	}
	return
}

func (tags DocTags) IsSet(tagName string) (found bool) {
	_, found = tags[tagName]
	return
}

func (tags DocTags) Contains(word string) (found bool) {

	for key := range tags {
		if strings.Contains(key, word) {
			return true
		}
	}
	return
}

func (tags DocTags) ToDocs() (docs []string) {

	for key, value := range tags {
		docs = append(docs, fmt.Sprintf("// %s %s=`%v`", mark, key, value))
	}
	return
}

func (tags DocTags) Sub(prefix string) (subTags DocTags) {

	prefix += "."
	subTags = make(DocTags)
	for key, value := range tags {
		if strings.HasPrefix(key, prefix) {
			subTags[strings.TrimPrefix(key, prefix)] = value
		}
	}
	return
}

func (tags DocTags) Set(tagName string, values ...string) {
	tags[tagName] = strings.Join(values, ",")
}

func (tags DocTags) Value(tagName string, defValue ...string) (value string) {

	var found bool
	if value, found = tags[tagName]; !found {
		value = strings.Join(defValue, " ")
	}
	return
}

func (tags DocTags) ValueInt(tagName string, defValue ...int) (value int) {

	if len(defValue) != 0 {
		value = defValue[0]
	}
	if textValue, found := tags[tagName]; found {
		if newValue, err := strconv.Atoi(textValue); err == nil {
			return newValue
		}
	}
	return
}

func (tags DocTags) ValueBool(tagName string, defValue ...bool) (value bool) {

	if len(defValue) != 0 {
		value = defValue[0]
	}
	if textValue, found := tags[tagName]; found {
		if newValue, err := strconv.ParseBool(textValue); err == nil {
			return newValue
		}
	}
	return
}

func (tags DocTags) ToKeys(tagName, separator string, defValue ...string) map[string]int {
	return utils.SliceStringToMap(strings.Split(tags.Value(tagName, defValue...), separator))
}

func (tags DocTags) ToMap(tagName, separator, splitter string, defValue ...string) (m map[string]string) {

	m = make(map[string]string)

	pairs := strings.Split(tags.Value(tagName, defValue...), separator)

	for _, pair := range pairs {
		if kv := strings.Split(pair, splitter); len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return
}

func (tags DocTags) contains(tagName string) (found bool) { // nolint
	_, found = tags[tagName]
	return
}
