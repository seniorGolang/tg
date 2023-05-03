package jsonrpc

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type CurlCommand struct {
	slice []string
}

type nopCloser struct {
	io.Reader
}

func toCurl(req *http.Request) (command *CurlCommand, err error) {

	command = &CurlCommand{}
	command.append("curl")
	command.append("-X", bashEscape(req.Method))
	if req.Body != nil {
		var body []byte
		if body, err = io.ReadAll(req.Body); err != nil {
			return
		}
		req.Body = nopCloser{bytes.NewBuffer(body)}
		bodyEscaped := bashEscape(string(body))
		command.append("-d", bodyEscaped)
	}
	var keys []string
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		command.append("-H", bashEscape(fmt.Sprintf("%s: %s", k, strings.Join(req.Header[k], " "))))
	}
	command.append(bashEscape(req.URL.String()))
	return
}

func (c *CurlCommand) append(newSlice ...string) {
	c.slice = append(c.slice, newSlice...)
}

func (c *CurlCommand) String() string {
	return strings.Join(c.slice, " ")
}

func bashEscape(str string) string {
	return `'` + strings.Replace(str, `'`, `'\''`, -1) + `'`
}

func (nopCloser) Close() error { return nil }
