package core

import (
	"context"
	"net/http"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/oasislabs/developer-gateway/rpc"
)

type HttpMiddlewareAuth struct {
	auth   Auth
	logger log.Logger
	next   rpc.HttpMiddleware
}

func NewHttpMiddlewareAuth(auth Auth, logger log.Logger, next rpc.HttpMiddleware) *HttpMiddlewareAuth {
	if auth == nil {
		panic("auth must be set")
	}

	if logger == nil {
		panic("log must be set")
	}

	if next == nil {
		panic("next must be set")
	}

	return &HttpMiddlewareAuth{
		auth:   auth,
		logger: logger.ForClass("auth", "HttpMiddlewareAuth"),
		next:   next,
	}
}

func (m *HttpMiddlewareAuth) ServeHTTP(req *http.Request) (interface{}, error) {
	value := req.Header.Get(m.auth.Key())
	id, err := m.auth.Verify(m.auth.Key(), value)

	if err != nil {
		return nil, rpc.HttpForbidden(req.Context(), "")
	}

	req = req.WithContext(context.WithValue(req.Context(), ContextKeyAuthID, id))
	return m.next.ServeHTTP(req)
}