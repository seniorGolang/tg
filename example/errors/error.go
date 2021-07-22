package errors

import (
	"github.com/gofiber/fiber/v2"
)

type ErrorType struct {
	Err    string
	Status int
}

func ErrUnauthorized() *ErrorType {
	return &ErrorType{
		Status: fiber.StatusUnauthorized,
	}
}

func ErrBadRequest() *ErrorType {
	return &ErrorType{
		Status: fiber.StatusBadRequest,
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
