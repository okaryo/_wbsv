package httpserver

import (
	"testing"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestBearerAuthMiddlewareAllowsMatchingToken(t *testing.T) {
	t.Parallel()

	called := false
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		called = true
		_, _ = w.Write([]byte("ok"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/private"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Authorization", Value: "Bearer secret"},
	}

	BearerAuthMiddleware("secret")(app).ServeHTTP(writer, request)

	if !called {
		t.Fatal("application handler was not called")
	}
	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
}

func TestBearerAuthMiddlewareRejectsMissingToken(t *testing.T) {
	t.Parallel()

	called := false
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		called = true
	})
	writer := newBufferedResponseWriter()

	BearerAuthMiddleware("secret")(app).ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/private"),
	})

	if called {
		t.Fatal("application handler was called")
	}
	response := writer.Response()
	if response.StatusCode != 401 {
		t.Fatalf("status code = %d, want 401", response.StatusCode)
	}
	if string(response.Body) != "unauthorized\n" {
		t.Fatalf("body = %q, want unauthorized body", string(response.Body))
	}
	if got := headerValue(response.Headers, "WWW-Authenticate"); got != `Bearer realm="wbsv"` {
		t.Fatalf("WWW-Authenticate = %q, want bearer challenge", got)
	}
}

func TestBearerAuthMiddlewareRejectsWrongToken(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		t.Fatal("application handler was called")
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/private"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Authorization", Value: "Bearer wrong"},
	}

	BearerAuthMiddleware("secret")(app).ServeHTTP(writer, request)

	response := writer.Response()
	if response.StatusCode != 401 {
		t.Fatalf("status code = %d, want 401", response.StatusCode)
	}
}

func TestBearerAuthMiddlewareIsBypassedWhenTokenIsEmpty(t *testing.T) {
	t.Parallel()

	called := false
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		called = true
	})
	writer := newBufferedResponseWriter()

	BearerAuthMiddleware("")(app).ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/private"),
	})

	if !called {
		t.Fatal("application handler was not called")
	}
}
