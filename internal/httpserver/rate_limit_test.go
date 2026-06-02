package httpserver

import (
	"testing"
	"time"
)

func TestFixedWindowRateLimitMiddlewareRejectsOverLimitRequests(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	calls := 0
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		calls++
		_, _ = w.Write([]byte("ok"))
	})
	handler := fixedWindowRateLimitMiddleware(2, time.Minute, func() time.Time {
		return now
	})(app)

	first := newBufferedResponseWriter()
	handler.ServeHTTP(first, Request{
		HTTP: testRequest("GET", "/limited"),
	})
	second := newBufferedResponseWriter()
	handler.ServeHTTP(second, Request{
		HTTP: testRequest("GET", "/limited"),
	})
	third := newBufferedResponseWriter()
	handler.ServeHTTP(third, Request{
		HTTP: testRequest("GET", "/limited"),
	})

	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2", calls)
	}
	response := third.Response()
	if response.StatusCode != 429 {
		t.Fatalf("status code = %d, want 429", response.StatusCode)
	}
	if string(response.Body) != "too many requests\n" {
		t.Fatalf("body = %q, want too many requests body", string(response.Body))
	}
	if got := headerValue(response.Headers, "Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want 60", got)
	}
}

func TestFixedWindowRateLimitMiddlewareResetsAfterWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	calls := 0
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		calls++
		_, _ = w.Write([]byte("ok"))
	})
	handler := fixedWindowRateLimitMiddleware(1, time.Minute, func() time.Time {
		return now
	})(app)

	first := newBufferedResponseWriter()
	handler.ServeHTTP(first, Request{
		HTTP: testRequest("GET", "/limited"),
	})
	rejected := newBufferedResponseWriter()
	handler.ServeHTTP(rejected, Request{
		HTTP: testRequest("GET", "/limited"),
	})

	now = now.Add(time.Minute)

	allowedAgain := newBufferedResponseWriter()
	handler.ServeHTTP(allowedAgain, Request{
		HTTP: testRequest("GET", "/limited"),
	})

	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2", calls)
	}
	if rejected.Response().StatusCode != 429 {
		t.Fatalf("rejected status code = %d, want 429", rejected.Response().StatusCode)
	}
	if allowedAgain.Response().StatusCode != 200 {
		t.Fatalf("allowed-again status code = %d, want 200", allowedAgain.Response().StatusCode)
	}
}

func TestFixedWindowRateLimitMiddlewareDisabledWhenLimitIsZero(t *testing.T) {
	t.Parallel()

	calls := 0
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		calls++
	})
	handler := FixedWindowRateLimitMiddleware(0, time.Minute)(app)

	for i := 0; i < 3; i++ {
		writer := newBufferedResponseWriter()
		handler.ServeHTTP(writer, Request{
			HTTP: testRequest("GET", "/unlimited"),
		})
	}

	if calls != 3 {
		t.Fatalf("handler calls = %d, want 3", calls)
	}
}
