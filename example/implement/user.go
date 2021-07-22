package implement

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/tg/example/interfaces"
	"github.com/seniorGolang/tg/example/interfaces/types"
)

type UserService struct {
	log logrus.FieldLogger
}

func NewUser(log logrus.FieldLogger) (svc *UserService) {
	return &UserService{log: log}
}

func (svc *UserService) GetUser(ctx context.Context, cookie, userAgent string) (user *types.User, err error) {
	panic("implement me")
}

func (svc *UserService) UploadFile(ctx context.Context, fileBytes []byte) (err error) {
	panic("implement me")
}

func (svc *UserService) CustomResponse(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {
	panic("implement me")
}

func (svc *UserService) CustomHandler(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {
	panic("implement me")
}

func CustomResponseHandler(ctx *fiber.Ctx, svc interfaces.User, err error, arg0 int, arg1 string, opts ...interface{}) {
	panic("implement me")
}

func CustomHandler(ctx *fiber.Ctx, svc interfaces.User) error {
	panic("implement me")
}
