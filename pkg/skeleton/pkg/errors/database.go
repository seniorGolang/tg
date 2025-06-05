package errors

import (
	"github.com/gofiber/fiber/v2"
)

var (
	ErrNotFound     = newError("not found", "notFound", fiber.ErrNotFound.Code)
	ErrDatabase     = newError("database error", "errorDB", fiber.ErrInternalServerError.Code)
	ErrDatabaseConn = newError("database connection error", "databaseConn", fiber.ErrInternalServerError.Code)
)
