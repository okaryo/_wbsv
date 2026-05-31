package httpserver

import (
	"context"

	"github.com/okaryo/_wbsv/internal/http1"
)

// Request is the application-facing request model.
type Request struct {
	Context context.Context
	HTTP    http1.Request
}

// AppHandler maps one parsed HTTP request to one HTTP response.
type AppHandler interface {
	ServeHTTP(Request) http1.Response
}

// AppHandlerFunc adapts a function to AppHandler.
type AppHandlerFunc func(Request) http1.Response

// ServeHTTP calls f(request).
func (f AppHandlerFunc) ServeHTTP(request Request) http1.Response {
	return f(request)
}

var defaultAppHandler AppHandler = AppHandlerFunc(func(request Request) http1.Response {
	body := []byte("hello from _wbsv\n")
	response := http1.Response{
		StatusCode: 200,
		Headers: []http1.HeaderField{
			{Name: "X-WBSV-Method", Value: request.HTTP.RequestLine.Method},
			{Name: "X-WBSV-Target", Value: request.HTTP.RequestLine.RequestTarget},
		},
		Body: body,
	}
	return http1.WithContentType(response, "text/plain; charset=utf-8")
})
