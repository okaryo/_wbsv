package httpserver

import (
	"context"

	"github.com/okaryo/_wbsv/internal/http1"
)

// Request is the application-facing request model.
type Request struct {
	Context context.Context
	HTTP    http1.Request
	Params  map[string]string
}

// Param returns a path parameter value.
func (r Request) Param(name string) string {
	if r.Params == nil {
		return ""
	}
	return r.Params[name]
}

// AppHandler maps one parsed HTTP request to one HTTP response.
type AppHandler interface {
	ServeHTTP(ResponseWriter, Request)
}

// AppHandlerFunc adapts a function to AppHandler.
type AppHandlerFunc func(ResponseWriter, Request)

// ServeHTTP calls f(request).
func (f AppHandlerFunc) ServeHTTP(w ResponseWriter, request Request) {
	f(w, request)
}

var defaultAppHandler AppHandler = AppHandlerFunc(func(w ResponseWriter, request Request) {
	w.AddHeader("X-WBSV-Method", request.HTTP.RequestLine.Method)
	w.AddHeader("X-WBSV-Target", request.HTTP.RequestLine.RequestTarget)
	w.SetHeader("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("hello from _wbsv\n"))
})
