// GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT.
package transport

import (
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

type ServiceRoute interface {
	SetRoutes(route *router.Router)
}

type Option func(srv *Server)
type Handler = fasthttp.RequestHandler
type ErrorHandler func(err error) error

func Service(svc ServiceRoute) Option {
	return func(srv *Server) {
		svc.SetRoutes(srv.Router())
	}
}

func JsonRPC(svc *httpJsonRPC) Option {
	return func(srv *Server) {
		srv.httpJsonRPC = svc
		svc.SetRoutes(srv.Router())
	}
}

func User(svc *httpUser) Option {
	return func(srv *Server) {
		srv.httpUser = svc
		svc.SetRoutes(srv.Router())
	}
}

func AfterHTTP(handler Handler) Option {
	return func(srv *Server) {
		srv.httpAfter = append(srv.httpAfter, handler)
	}
}

func BeforeHTTP(handler Handler) Option {
	return func(srv *Server) {
		srv.httpBefore = append(srv.httpBefore, handler)
	}
}

func MaxBodySize(max int) Option {
	return func(srv *Server) {
		srv.maxRequestBodySize = max
	}
}
