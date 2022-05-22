package utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func LogRequest(ctx *fiber.Ctx) (err error) {

	log.Ctx(ctx.UserContext()).Debug().
		Str("path", ctx.Path()).
		Str("method", ctx.Method()).
		Str("body", string(ctx.Body())).
		Send()
	return ctx.Next()
}
