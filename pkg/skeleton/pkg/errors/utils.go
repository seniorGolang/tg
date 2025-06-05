package errors

import (
	"context"
	stdError "errors"
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var Is = errors.Is
var New = errors.New
var Wrap = errors.Wrap
var Join = stdError.Join
var Wrapf = errors.Wrapf
var Errorf = errors.Errorf
var WithStack = errors.WithStack
var WithMessage = errors.WithMessage
var WithMessagef = errors.WithMessagef

func ExitOnError(ctx context.Context, err error, msg string) {
	if err != nil {
		log.Ctx(ctx).Panic().Err(err).Msg(msg)
	}
}

func HTTPToError(code int, errMessage string) error {
	switch code {
	case http.StatusBadRequest:
		return ErrBadRequest.SetCauseRaw(errMessage)
	case http.StatusUnauthorized:
		return ErrUnauthorized.SetCauseRaw(errMessage)
	case http.StatusForbidden:
		return ErrForbidden.SetCauseRaw(errMessage)
	case http.StatusNotFound:
		return ErrNotFound.SetCauseRaw(errMessage)
	case http.StatusConflict:
		return ErrConflict.SetCauseRaw(errMessage)
	case http.StatusInternalServerError:
		return ErrInternal.SetCauseRaw(errMessage)
	case http.StatusNotImplemented:
		return ErrNotImplemented.SetCauseRaw(errMessage)
	default:
		return Errorf("HTTP Code: %d, Message: %s", code, errMessage)
	}
}
