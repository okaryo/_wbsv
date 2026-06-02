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
	if err := router.Handle("/ok", nil); !errors.Is(err, ErrMissingHandler) {
		t.Fatalf("nil handler error = %v, want ErrMissingHandler", err)
	}
	if err := router.HandleFunc("/ok", nil); !errors.Is(err, ErrMissingHandler) {
		t.Fatalf("nil handler func error = %v, want ErrMissingHandler", err)
	}
}
