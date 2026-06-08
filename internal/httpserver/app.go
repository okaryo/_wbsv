package httpserver

import (
	"context"
	"time"

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

// Cookie returns the first request cookie with name.
func (r Request) Cookie(name string) (Cookie, bool) {
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			return cookie, true
		}
	}
	return Cookie{}, false
}

// Cookies returns cookies parsed from the request Cookie headers.
func (r Request) Cookies() []Cookie {
	return requestCookies(r.HTTP.Headers)
}

// IfNoneMatch reports whether the request If-None-Match header matches etag.
func (r Request) IfNoneMatch(etag string) bool {
	return requestETagMatches(r.HTTP.Headers, etag)
}

// IfModifiedSince reports whether lastModified is not newer than the request
// If-Modified-Since value.
func (r Request) IfModifiedSince(lastModified time.Time) bool {
	return requestNotModifiedSince(r.HTTP.Headers, lastModified)
}

// NotModified applies the usual cache revalidation order for safe responses:
// If-None-Match takes priority over If-Modified-Since when both are present.
func (r Request) NotModified(etag string, lastModified time.Time) bool {
	if hasHeader(r.HTTP.Headers, "If-None-Match") {
		return r.IfNoneMatch(etag)
	}
	return r.IfModifiedSince(lastModified)
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
