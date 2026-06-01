package httpserver

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestLoggingMiddlewareLogsRequestAndResponse(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte("created\n"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("POST", "/items"),
	}

	LoggingMiddleware(logger)(app).ServeHTTP(writer, request)

	logLine := logs.String()
	if !strings.Contains(logLine, "POST /items -> 201 8B ") {
		t.Fatalf("log line = %q, want method target status and bytes", logLine)
	}
}

func TestLoggingMiddlewareDefaultsStatusToOK(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/empty"),
	}

	LoggingMiddleware(logger)(app).ServeHTTP(writer, request)

	logLine := logs.String()
	if !strings.Contains(logLine, "GET /empty -> 200 0B ") {
		t.Fatalf("log line = %q, want default status and zero bytes", logLine)
	}
}

func testRequest(method, target string) http1.Request {
	return http1.Request{
		RequestLine: http1.RequestLine{
			Method:        method,
			RequestTarget: target,
			Version:       "HTTP/1.1",
		},
	}
}
