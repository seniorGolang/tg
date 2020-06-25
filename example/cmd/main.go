package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/tg/example/implement"
	"github.com/seniorGolang/tg/example/transport"
)

var log = logrus.New()

func main() {

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT)

	defer log.Info("goodbye")

	svcUser := implement.NewUser(log.WithField("module", "user"))
	svcJsonRPC := implement.NewJsonRPC(log.WithField("module", "jsonRPC"))

	srv := transport.New(log, svcJsonRPC, svcUser).WithLog(log).WithTrace().TraceJaeger("example")

	srv.ServeHTTP(":9000")

	<-shutdown

	log.Info("start shutdown server")
}
