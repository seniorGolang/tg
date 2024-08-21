package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"go.opentelemetry.io/otel/attribute"

	"github.com/seniorGolang/tg/v2/example/config"
	"github.com/seniorGolang/tg/v2/example/implement"
	"github.com/seniorGolang/tg/v2/example/transport"
	"github.com/seniorGolang/tg/v2/example/utils"
	"github.com/seniorGolang/tg/v2/example/utils/header"
)

const (
	appVersion  = "local"
	serviceName = "example"
)

func main() {

	log.Logger = config.Service().Logger()

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	ctx := log.Logger.WithContext(context.Background())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT)

	log.Info().Msg("start server")
	defer log.Info().Msg("shutdown server")

	svcUser := implement.NewUser()
	svcJsonRPC := implement.NewJsonRPC(log.With().Str("module", "jsonRPC").Logger())

	services := []transport.Option{
		transport.Use(cors.New()),
		transport.Use(compress.New()),
		transport.Use(utils.LogRequest),
		transport.WithRequestID("X-Request-Id"),
		transport.WithHeader(header.AppHeader, header.AppName),
		transport.User(transport.NewUser(svcUser)),
		transport.ExampleRPC(transport.NewExampleRPC(svcJsonRPC)),
	}

	srv := transport.New(log.Logger, services...).WithLog().WithMetrics().WithTrace(ctx, serviceName, "localhost:4317", attribute.String("appVersion", appVersion))

	go func() {
		log.Info().Str("bind", config.Service().Bind).Msg("listen on")
		if err := srv.Fiber().Listen(config.Service().Bind); err != nil {
			log.Panic().Err(err).Stack().Msg("server error")
		}
	}()

	<-shutdown
}
