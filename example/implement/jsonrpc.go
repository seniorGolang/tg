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

func (svc *JsonRPCService) Test(_ context.Context, arg0 int, arg1 string, opts ...interface{}) (ret1 int, ret2 string, err error) {
	return arg0, arg1, nil
}

func (svc *JsonRPCService) Test2(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (ret1 int, ret2 string, err error) {
	// TODO implement me
	panic("implement me")
}
