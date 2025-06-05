package errors

import (
	"github.com/gofiber/fiber/v2"
)

var (
	ErrInternal       = newError("internal error", "internal", fiber.ErrInternalServerError.Code)
	ErrConflict       = newError("conflict on input data", "conflict", fiber.ErrConflict.Code)
	ErrForbidden      = newError("forbidden", "forbidden", fiber.ErrForbidden.Code)
	ErrBadRequest     = newError("bad request", "badRequest", fiber.ErrBadRequest.Code)
	ErrAlreadyExist   = newError("already exist", "alreadyExist")
	ErrAccessDenied   = newError("access denied", "accessDenied", fiber.ErrUnauthorized.Code)
	ErrUnauthorized   = newError("you are not authorized", "unauthorized", fiber.ErrUnauthorized.Code)
	ErrDeprecated     = newError("deprecated", "deprecated", fiber.ErrNotImplemented.Code)
	ErrNotImplemented = newError("not implemented", "notImplemented", fiber.ErrNotImplemented.Code)
)
