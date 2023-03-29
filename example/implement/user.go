package implement

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/seniorGolang/tg/v2/example/interfaces"
	"github.com/seniorGolang/tg/v2/example/interfaces/types"
)

type UserService struct {
}

func NewUser() (svc *UserService) {
	return &UserService{}
}

func (svc *UserService) GetUser(ctx context.Context, cookie, userAgent string) (user *types.User, err error) {

	user = &types.User{
		UserID: 1000,
		Name:   "John Dow",
	}
	return
}

func (svc *UserService) CustomResponse(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {
	panic("implement me")
}

func (svc *UserService) CustomHandler(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error) {
	panic("implement me")
}

func CustomResponseHandler(ctx *fiber.Ctx, svc interfaces.User, arg0 int, arg1 string, opts ...interface{}) (err error) {
	panic("implement me")
}

func CustomHandler(ctx *fiber.Ctx, svc interfaces.User) error {
	panic("implement me")
}
