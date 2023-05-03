package jsonrpc

import (
	"encoding/json"
	"strconv"
)

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Raw() (data json.RawMessage) {
	data, _ = json.Marshal(e)
	return
}

func (e *RPCError) Error() string {
	return strconv.Itoa(e.Code) + ": " + e.Message
}

type HTTPError struct {
	Code int
	err  error
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}
