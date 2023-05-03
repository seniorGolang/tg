package cb

import (
	"errors"
)

var (
	ErrTooManyRequests = errors.New("too many requests")
	ErrOpenState       = errors.New("circuit breaker is open")
)
