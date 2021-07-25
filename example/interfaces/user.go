// @tg version=0.0.0
// @tg title=`Example API`
// @tg description=`A service which provide Example API`
// @tg servers=`http://example.test`
//go:generate tg client --services . --outPath ../clients/example
//go:generate tg transport --services . --out ../transport --outSwagger ../swagger.yaml
package interfaces

import (
	"context"

	"github.com/seniorGolang/tg/example/interfaces/types"
)

// @tg http-prefix=api/v2
// @tg http-server log metrics trace
// общий код 400 для всех методов, кроме UploadFile
// @tg 400=github.com/seniorGolang/tg/example/error:ErrorType
type User interface {

	// @tg summary=`Данные пользователя`
	// @tg desc=`Возвращает данные пользователя код успеха 204`
	// @tg http-method=GET
	// @tg http-success=204
	// @tg http-path=/user/info
	// @tg http-cookies=cookie|sessionCookie
	// @tg http-headers=userAgent|User-Agent
	// @tg 401=github.com/seniorGolang/tg/example/error:ErrorType
	GetUser(ctx context.Context, cookie, userAgent string) (user *types.User, err error)

	// @tg summary=`Загрузка аватара пользователя`
	// @tg desc=`Загрузка файла`
	// @tg http-method=POST
	// @tg uploadTo=fileBytes
	// @tg http-path=/user/file
	// @tg http-upload=fileBytes|fileBytes
	// @tg 400=-
	UploadFile(ctx context.Context, fileBytes []byte) (err error)

	// @tg summary=`Метод со сторонним обработчиком ответа`
	// @tg http-method=PATCH
	// @tg http-path=/user/custom/response
	// @tg http-response=github.com/seniorGolang/tg/example/implement:CustomResponseHandler
	CustomResponse(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error)

	// @tg summary=`Метод полностью обрабатываемый кастомным хендлером`
	// @tg http-method=DELETE
	// @tg http-path=/user/custom
	// @tg handler=github.com/seniorGolang/tg/example/implement:CustomHandler
	CustomHandler(ctx context.Context, arg0 int, arg1 string, opts ...interface{}) (err error)
}
