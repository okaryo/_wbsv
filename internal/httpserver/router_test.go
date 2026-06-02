package httpserver

import (
	"errors"
	"testing"
)

func TestRouterDispatchesStaticPath(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleFunc("/hello", func(w ResponseWriter, request Request) {
		w.SetHeader("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello route\n"))
	}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	writer := newBufferedResponseWriter()
	router.ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/hello"),
	})

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if string(response.Body) != "hello route\n" {
		t.Fatalf("body = %q, want hello route body", string(response.Body))
	}
}

func TestRouterIgnoresQueryStringForStaticPathMatch(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleFunc("/search", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("search\n"))
	}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	writer := newBufferedResponseWriter()
	router.ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/search?q=go"),
	})

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if string(response.Body) != "search\n" {
		t.Fatalf("body = %q, want search body", string(response.Body))
	}
}

func TestRouterDispatchesByMethodAndPath(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleMethodFunc("GET", "/items", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("list items\n"))
	}); err != nil {
		t.Fatalf("handle GET: %v", err)
	}
	if err := router.HandleMethodFunc("POST", "/items", func(w ResponseWriter, request Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte("create item\n"))
	}); err != nil {
		t.Fatalf("handle POST: %v", err)
	}

	getWriter := newBufferedResponseWriter()
	router.ServeHTTP(getWriter, Request{
		HTTP: testRequest("GET", "/items"),
	})
	postWriter := newBufferedResponseWriter()
	router.ServeHTTP(postWriter, Request{
		HTTP: testRequest("POST", "/items"),
	})

	if got := string(getWriter.Response().Body); got != "list items\n" {
		t.Fatalf("GET body = %q, want list items", got)
	}
	if postWriter.Response().StatusCode != 201 {
		t.Fatalf("POST status code = %d, want 201", postWriter.Response().StatusCode)
	}
	if got := string(postWriter.Response().Body); got != "create item\n" {
		t.Fatalf("POST body = %q, want create item", got)
	}
}

func TestRouterReturnsMethodNotAllowedForKnownPathAndUnknownMethod(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleMethodFunc("POST", "/items", func(w ResponseWriter, request Request) {}); err != nil {
		t.Fatalf("handle POST: %v", err)
	}
	if err := router.HandleMethodFunc("GET", "/items", func(w ResponseWriter, request Request) {}); err != nil {
		t.Fatalf("handle GET: %v", err)
	}
	writer := newBufferedResponseWriter()

	router.ServeHTTP(writer, Request{
		HTTP: testRequest("DELETE", "/items"),
	})

	response := writer.Response()
	if response.StatusCode != 405 {
		t.Fatalf("status code = %d, want 405", response.StatusCode)
	}
	if got := headerValue(response.Headers, "Allow"); got != "GET, POST" {
		t.Fatalf("Allow = %q, want GET, POST", got)
	}
	if string(response.Body) != "method not allowed\n" {
		t.Fatalf("body = %q, want method not allowed body", string(response.Body))
	}
}

func TestRouterPrefersExactMethodOverAnyMethod(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleFunc("/items", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("any method\n"))
	}); err != nil {
		t.Fatalf("handle any method: %v", err)
	}
	if err := router.HandleMethodFunc("GET", "/items", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("get only\n"))
	}); err != nil {
		t.Fatalf("handle GET: %v", err)
	}

	getWriter := newBufferedResponseWriter()
	router.ServeHTTP(getWriter, Request{
		HTTP: testRequest("GET", "/items"),
	})
	postWriter := newBufferedResponseWriter()
	router.ServeHTTP(postWriter, Request{
		HTTP: testRequest("POST", "/items"),
	})

	if got := string(getWriter.Response().Body); got != "get only\n" {
		t.Fatalf("GET body = %q, want exact method handler", got)
	}
	if got := string(postWriter.Response().Body); got != "any method\n" {
		t.Fatalf("POST body = %q, want any-method handler", got)
	}
}

func TestRouterDispatchesPathParameters(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleMethodFunc("GET", "/users/:id", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("user " + request.Param("id") + "\n"))
	}); err != nil {
		t.Fatalf("handle param route: %v", err)
	}

	writer := newBufferedResponseWriter()
	router.ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/users/42"),
	})

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if string(response.Body) != "user 42\n" {
		t.Fatalf("body = %q, want path parameter body", string(response.Body))
	}
}

func TestRouterIgnoresQueryStringForPathParameterMatch(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleFunc("/users/:id", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte(request.Param("id")))
	}); err != nil {
		t.Fatalf("handle param route: %v", err)
	}

	writer := newBufferedResponseWriter()
	router.ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/users/42?expand=true"),
	})

	if got := string(writer.Response().Body); got != "42" {
		t.Fatalf("body = %q, want path parameter without query", got)
	}
}

func TestRouterPrefersStaticRouteOverPathParameterRoute(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleFunc("/users/:id", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("param " + request.Param("id")))
	}); err != nil {
		t.Fatalf("handle param route: %v", err)
	}
	if err := router.HandleFunc("/users/me", func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("static me"))
	}); err != nil {
		t.Fatalf("handle static route: %v", err)
	}

	writer := newBufferedResponseWriter()
	router.ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/users/me"),
	})

	if got := string(writer.Response().Body); got != "static me" {
		t.Fatalf("body = %q, want static route body", got)
	}
}

func TestRouterReturnsMethodNotAllowedForPathParameterRoute(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleMethodFunc("GET", "/users/:id", func(w ResponseWriter, request Request) {}); err != nil {
		t.Fatalf("handle param route: %v", err)
	}
	writer := newBufferedResponseWriter()

	router.ServeHTTP(writer, Request{
		HTTP: testRequest("POST", "/users/42"),
	})

	response := writer.Response()
	if response.StatusCode != 405 {
		t.Fatalf("status code = %d, want 405", response.StatusCode)
	}
	if got := headerValue(response.Headers, "Allow"); got != "GET" {
		t.Fatalf("Allow = %q, want GET", got)
	}
}

func TestRouterReturnsNotFoundForUnregisteredPath(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	writer := newBufferedResponseWriter()

	router.ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/missing"),
	})

	response := writer.Response()
	if response.StatusCode != 404 {
		t.Fatalf("status code = %d, want 404", response.StatusCode)
	}
	if string(response.Body) != "not found\n" {
		t.Fatalf("body = %q, want not found body", string(response.Body))
	}
}

func TestRouterRejectsInvalidRegistrations(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	if err := router.HandleFunc("", func(ResponseWriter, Request) {}); !errors.Is(err, ErrInvalidRoutePath) {
		t.Fatalf("empty path error = %v, want ErrInvalidRoutePath", err)
	}
	if err := router.HandleFunc("relative", func(ResponseWriter, Request) {}); !errors.Is(err, ErrInvalidRoutePath) {
		t.Fatalf("relative path error = %v, want ErrInvalidRoutePath", err)
	}
	if err := router.HandleMethodFunc("BAD METHOD", "/ok", func(ResponseWriter, Request) {}); !errors.Is(err, ErrInvalidRouteMethod) {
		t.Fatalf("bad method error = %v, want ErrInvalidRouteMethod", err)
	}
	if err := router.HandleFunc("/users/:", func(ResponseWriter, Request) {}); !errors.Is(err, ErrInvalidRoutePath) {
		t.Fatalf("empty parameter name error = %v, want ErrInvalidRoutePath", err)
	}
	if err := router.HandleFunc("/users/:id/books/:id", func(ResponseWriter, Request) {}); !errors.Is(err, ErrInvalidRoutePath) {
		t.Fatalf("duplicate parameter name error = %v, want ErrInvalidRoutePath", err)
	}
	if err := router.Handle("/ok", nil); !errors.Is(err, ErrMissingHandler) {
		t.Fatalf("nil handler error = %v, want ErrMissingHandler", err)
	}
	if err := router.HandleFunc("/ok", nil); !errors.Is(err, ErrMissingHandler) {
		t.Fatalf("nil handler func error = %v, want ErrMissingHandler", err)
	}
}
