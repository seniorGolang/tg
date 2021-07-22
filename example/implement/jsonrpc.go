package implement

import (
	"context"

	"github.com/rs/zerolog"
)

type JsonRPCService struct {
	log zerolog.Logger
}

func NewJsonRPC(log zerolog.Logger) (svc *JsonRPCService) {
	return &JsonRPCService{log: log}
}

func (svc *JsonRPCService) Test(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (ret1 int, ret2 string, err error) {
	panic("implement me")
}
