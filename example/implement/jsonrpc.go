package implement

import (
	"context"

	"github.com/sirupsen/logrus"
)

type JsonRPCService struct {
	log logrus.FieldLogger
}

func NewJsonRPC(log logrus.FieldLogger) (svc *JsonRPCService) {
	return &JsonRPCService{log: log}
}

func (svc *JsonRPCService) Test(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (ret1 int, ret2 string, err error) {
	panic("implement me")
}
