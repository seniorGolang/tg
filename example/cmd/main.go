package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/seniorGolang/tg/v2/example/config"
	"github.com/seniorGolang/tg/v2/example/implement"
	"github.com/seniorGolang/tg/v2/example/transport"
	"github.com/seniorGolang/tg/v2/example/utils/header"
)

func main() {

	log.Logger = config.Service().Logger()

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT)

	log.Info().Msg("start server")
	defer log.Info().Msg("shutdown server")

	svcUser := implement.NewUser()
	svcJsonRPC := implement.NewJsonRPC(log.With().Str("module", "jsonRPC").Logger())

	services := []transport.Option{
		transport.WithHeader(header.AppHeader, header.AppName),
		transport.WithRequestID("X-Request-Id"),
		transport.User(transport.NewUser(log.Logger, svcUser)),
		transport.ExampleRPC(transport.NewExampleRPC(log.Logger, svcJsonRPC)),
	}

	srv := transport.New(log.Logger, services...).WithLog().WithTrace().TraceJaeger("example")

	go func() {
		log.Info().Str("bind", config.Service().Bind).Msg("listen on")
		if err := srv.Fiber().Listen(config.Service().Bind); err != nil {
			log.Panic().Err(err).Stack().Msg("server error")
		}
	}()

	<-shutdown
}
