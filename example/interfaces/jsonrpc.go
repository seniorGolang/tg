package interfaces

import (
	"context"
)

// @tg jsonRPC-server log trace metrics
type JsonRPC interface {

	// @tg summary=`json RPC метод`
	// @tg arg1.type=string
	// @tg arg1.format=uuid
	Test(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (ret1 int, ret2 string, err error)
}
