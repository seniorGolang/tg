package jsonrpc

import (
	"github.com/google/uuid"
)

type RequestRPC struct {
	ID      ID          `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	JSONRPC string      `json:"jsonrpc"`
}

type RequestsRPC []*RequestRPC

func NewRequest(method string, params ...interface{}) *RequestRPC {

	request := &RequestRPC{
		ID:      uuid.New(),
		Method:  method,
		Params:  Params(params...),
		JSONRPC: Version,
	}
	return request
}

func NewRequestWithID(id ID, method string, params ...interface{}) *RequestRPC {

	request := &RequestRPC{
		ID:      id,
		Method:  method,
		Params:  Params(params...),
		JSONRPC: Version,
	}
	return request
}
