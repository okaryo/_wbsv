package httpserver

import (
	"reflect"
	"testing"
)

func TestChainAppliesMiddlewaresInOrder(t *testing.T) {
	t.Parallel()

	var calls []string
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		calls = append(calls, "handler")
		_, _ = w.Write([]byte("ok"))
	})

	first := func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			calls = append(calls, "first before")
			next.ServeHTTP(w, request)
			calls = append(calls, "first after")
		})
	}
	second := func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			calls = append(calls, "second before")
			next.ServeHTTP(w, request)
			calls = append(calls, "second after")
		})
	}

	writer := newBufferedResponseWriter()
	Chain(app, first, second).ServeHTTP(writer, Request{})

	want := []string{
		"first before",
		"second before",
		"handler",
		"second after",
		"first after",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
	if got := string(writer.Response().Body); got != "ok" {
		t.Fatalf("body = %q, want %q", got, "ok")
	}
}

func TestChainUsesDefaultHandlerWhenHandlerIsNil(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	Chain(nil).ServeHTTP(writer, Request{})

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if string(response.Body) != "hello from _wbsv\n" {
		t.Fatalf("body = %q, want default response body", string(response.Body))
	}
}
