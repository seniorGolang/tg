package errors

import (
	"fmt"
)

var errorsMap = make(map[string]Error)

type Error struct {
	TrKey string `json:"trKey"`

	Data  any `json:"data,omitempty"`
	Cause any `json:"cause,omitempty"`

	msg     string
	errCode int
}

func newError(msg, trKey string, code ...int) Error {

	errCode := -32603
	if len(code) != 0 {
		errCode = code[0]
	}
	errorsMap[trKey] = makeError(msg, trKey, errCode)
	return errorsMap[trKey]
}

func makeError(msg, trKey string, code int) Error {

	errCode := -32603
	if code != 0 {
		errCode = code
	}
	return Error{msg: msg, TrKey: trKey, errCode: errCode}
}

func (e Error) Unwrap() error {

	msg := e.msg
	if e.Cause != nil || e.Data != nil {
		return newError(msg, e.TrKey, e.errCode)
	}
	return nil
}

func (e Error) WithData(data interface{}) Error {

	e.Data = data
	return e
}

func (e Error) WithCustomMsg(msg string) Error {

	e.msg = msg
	return e
}

func (e Error) Error() (errStr string) {

	if e.Data != nil {
		errStr = fmt.Sprintf(": %v", e.Data)
	}
	if e.Cause != nil {
		errStr = fmt.Sprintf("%s cause: %v", errStr, e.Cause)
	}
	return e.msg + errStr
}

func (e Error) Code() int {
	return e.errCode
}

func (e Error) SetCause(format string, a ...interface{}) Error {

	e.Cause = fmt.Sprintf(format, a...)
	return e
}

func (e Error) SetCauseRaw(v interface{}) Error {

	e.Cause = v
	return e
}

func (e Error) SetCode(code int) Error {

	e.errCode = code
	return e
}

func Map() (errors map[string]Error) {

	errors = make(map[string]Error)
	for k, v := range errorsMap {
		errors[k] = v
	}
	return
}
