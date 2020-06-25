package errors

import (
	"github.com/valyala/fasthttp"
)

type ErrorType struct {
	Err    string
	Status int
}

func ErrUnauthorized() *ErrorType {
	return &ErrorType{
		Status: fasthttp.StatusUnauthorized,
	}
}

func ErrBadRequest() *ErrorType {
	return &ErrorType{
		Status: fasthttp.StatusBadRequest,
	}
}

func (e *ErrorType) WithMessage(msg string) *ErrorType {
	e.Err = msg
	return e
}

func (e *ErrorType) Code() int {
	return e.Status
}

func (e *ErrorType) Error() string {
	return e.Err
}
