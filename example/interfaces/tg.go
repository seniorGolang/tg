// @tg version=0.0.0
// @tg title=`Example API`
// @tg description=`A service which provide Example API`
// @tg servers=`http://example.test`
// @tg packageJSON=`github.com/seniorGolang/json`
// @tg http-prefix=api/v1
//
//go:generate tg client --services . --outPath ../clients/example
//go:generate tg transport --services . --out ../transport --outSwagger ../swagger.yaml
package interfaces
