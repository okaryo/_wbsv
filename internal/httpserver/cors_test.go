package httpserver

import (
	"testing"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestCORSMiddlewareAddsHeadersForAllowedActualRequest(t *testing.T) {
	t.Parallel()

	var called bool
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		called = true
		w.SetHeader("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/cors"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Origin", Value: "https://app.example.com"},
	}

	CORSMiddleware(CORSOptions{
		AllowOrigins:     []string{"https://app.example.com"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	})(app).ServeHTTP(writer, request)

	response := writer.Response()
	if !called {
		t.Fatal("handler was not called")
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want app origin", got)
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
	if got := headerValue(response.Headers, "Access-Control-Expose-Headers"); got != "X-Request-ID" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want X-Request-ID", got)
	}
	if got := headerValue(response.Headers, "Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin", got)
	}
	if string(response.Body) != "ok" {
		t.Fatalf("body = %q, want ok", string(response.Body))
	}
}

func TestCORSMiddlewareHandlesPreflightRequest(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		t.Fatal("handler should not be called for preflight")
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("OPTIONS", "/cors"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Origin", Value: "https://app.example.com"},
		{Name: "Access-Control-Request-Method", Value: "POST"},
		{Name: "Access-Control-Request-Headers", Value: "Authorization, X-Client"},
	}

	CORSMiddleware(CORSOptions{
		AllowOrigins: []string{"https://app.example.com"},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Authorization", "X-Client"},
		MaxAge:       600,
	})(app).ServeHTTP(writer, request)

	response := writer.Response()
	if response.StatusCode != 204 {
		t.Fatalf("status code = %d, want 204", response.StatusCode)
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want app origin", got)
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want GET, POST", got)
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Headers"); got != "Authorization, X-Client" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want Authorization, X-Client", got)
	}
	if got := headerValue(response.Headers, "Access-Control-Max-Age"); got != "600" {
		t.Fatalf("Access-Control-Max-Age = %q, want 600", got)
	}
}

func TestCORSMiddlewareEchoesRequestedPreflightValuesWhenDefaultsAreEmpty(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		t.Fatal("handler should not be called for preflight")
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("OPTIONS", "/cors"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Origin", Value: "https://app.example.com"},
		{Name: "Access-Control-Request-Method", Value: "PATCH"},
		{Name: "Access-Control-Request-Headers", Value: "Authorization, X-Client"},
	}

	CORSMiddleware(CORSOptions{
		AllowOrigins: []string{"https://app.example.com"},
	})(app).ServeHTTP(writer, request)

	response := writer.Response()
	if got := headerValue(response.Headers, "Access-Control-Allow-Methods"); got != "PATCH" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want PATCH", got)
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Headers"); got != "Authorization, X-Client" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want requested headers", got)
	}
}

func TestCORSMiddlewareSkipsDisallowedOrigin(t *testing.T) {
	t.Parallel()

	var called bool
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		called = true
		_, _ = w.Write([]byte("ok"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/cors"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Origin", Value: "https://evil.example.com"},
	}

	CORSMiddleware(CORSOptions{
		AllowOrigins: []string{"https://app.example.com"},
	})(app).ServeHTTP(writer, request)

	response := writer.Response()
	if !called {
		t.Fatal("handler was not called")
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORSMiddlewareAllowsWildcardOrigin(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("ok"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/cors"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Origin", Value: "https://app.example.com"},
	}

	CORSMiddleware(CORSOptions{
		AllowOrigins: []string{"*"},
	})(app).ServeHTTP(writer, request)

	if got := headerValue(writer.Response().Headers, "Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want wildcard", got)
	}
}

func TestCORSMiddlewareEchoesOriginForWildcardWithCredentials(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("ok"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/cors"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Origin", Value: "https://app.example.com"},
	}

	CORSMiddleware(CORSOptions{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})(app).ServeHTTP(writer, request)

	response := writer.Response()
	if got := headerValue(response.Headers, "Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want echoed origin", got)
	}
	if got := headerValue(response.Headers, "Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
}
