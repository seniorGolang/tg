package jsonrpc

import (
	"encoding/json"

	"github.com/google/uuid"
)

type ResponseRPC struct {
	ID      uuid.UUID       `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Error   *RPCError       `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type ResponsesRPC []*ResponseRPC

func (res ResponsesRPC) AsMap() map[ID]*ResponseRPC {

	resMap := make(map[ID]*ResponseRPC, 0)
	for _, r := range res {
		resMap[r.ID] = r
	}
	return resMap
}

func (res ResponsesRPC) GetByID(id ID) *ResponseRPC {

	for _, r := range res {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (res ResponsesRPC) HasError() bool {

	for _, resp := range res {
		if resp.Error != nil {
			return true
		}
	}
	return false
}

func (responseRPC *ResponseRPC) GetObject(object interface{}) error {

	js, err := json.Marshal(responseRPC.Result)
	if err != nil {
		return err
	}
	err = json.Unmarshal(js, object)
	if err != nil {
		return err
	}
	return nil
}
