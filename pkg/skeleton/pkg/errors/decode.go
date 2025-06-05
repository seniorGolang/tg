package errors

import (
	"encoding/json"
)

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Error  `json:"data,omitempty"`
}

func Decoder(errData json.RawMessage) (err error) {

	var srcErr rpcError
	if err = json.Unmarshal(errData, &srcErr); err != nil {
		return ErrInternal.SetCauseRaw(string(errData))
	}
	return makeError(srcErr.Message, srcErr.Data.TrKey, srcErr.Code).SetCauseRaw(srcErr.Data.Cause)
}
