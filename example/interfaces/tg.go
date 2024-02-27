// @tg version=0.0.0
// @tg title=`Example API`
// @tg description=`A service which provide Example API`
// @tg servers=`http://example.test`
// @tg packageJSON=`github.com/seniorGolang/json`
// @tg http-prefix=api/v1
// @tg circuit-breaker
//
//go:generate tg client --go --services . --outPath ../clients/example
//go:generate tg client --js --services . --outPath ../clients/example
//go:generate tg transport --services . --out ../transport --outSwagger ../swagger.yaml
package interfaces
