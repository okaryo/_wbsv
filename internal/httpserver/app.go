package httpserver

import "github.com/okaryo/_wbsv/internal/http1"

// AppHandler maps one parsed HTTP request to one HTTP response.
type AppHandler interface {
	ServeHTTP(http1.Request) http1.Response
}

// AppHandlerFunc adapts a function to AppHandler.
type AppHandlerFunc func(http1.Request) http1.Response

// ServeHTTP calls f(request).
func (f AppHandlerFunc) ServeHTTP(request http1.Request) http1.Response {
	return f(request)
}

var defaultAppHandler AppHandler = AppHandlerFunc(func(request http1.Request) http1.Response {
	body := []byte("hello from _wbsv\n")
	response := http1.Response{
		StatusCode: 200,
		Headers: []http1.HeaderField{
			{Name: "X-WBSV-Method", Value: request.RequestLine.Method},
			{Name: "X-WBSV-Target", Value: request.RequestLine.RequestTarget},
		},
		Body: body,
	}
	return http1.WithContentType(response, "text/plain; charset=utf-8")
})
