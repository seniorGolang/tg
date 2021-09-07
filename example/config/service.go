package config

import (
	"io"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const formatJSON = "json"

type ServiceConfig struct {
	LogLevel  string `envconfig:"LOGGER_LEVEL" default:"debug"`
	LogFormat string `envconfig:"LOGGER_FORMAT" default:"console"`

	Bind        string `envconfig:"BIND_SERVICE" default:":9000"`
	MetricsBind string `envconfig:"BIND_METRICS" default:":9090"`
	HealthBind  string `envconfig:"BIND_HEALTH" default:":9091"`
}

var service *ServiceConfig // nolint

func Service() ServiceConfig {

	if service != nil {
		return *service
	}
	service = &ServiceConfig{}
	if err := envconfig.Process("", service); err != nil {
		panic(err)
	}
	return *service
}

func (cfg ServiceConfig) Logger() (logger zerolog.Logger) {

	level := zerolog.InfoLevel
	if newLevel, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		level = newLevel
	}
	var out io.Writer = os.Stdout
	if cfg.LogFormat != formatJSON {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.StampMicro}
	}
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	return zerolog.New(out).Level(level).With().Timestamp().Logger()
}
