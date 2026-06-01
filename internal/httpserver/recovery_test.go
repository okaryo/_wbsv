package httpserver

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestRecoveryMiddlewareConvertsPanicToInternalServerError(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		panic("boom")
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/panic"),
	}

	RecoveryMiddleware(logger)(app).ServeHTTP(writer, request)

	response := writer.Response()
	if response.StatusCode != 500 {
		t.Fatalf("status code = %d, want 500", response.StatusCode)
	}
	if string(response.Body) != "internal server error\n" {
		t.Fatalf("body = %q, want internal server error body", string(response.Body))
	}
	if !strings.Contains(logs.String(), "panic while handling GET /panic: boom") {
		t.Fatalf("logs = %q, want panic log", logs.String())
	}
}

func TestRecoveryMiddlewareResetsPartialBufferedResponse(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		w.SetHeader("X-Partial", "yes")
		w.WriteHeader(201)
		_, _ = w.Write([]byte("partial"))
		panic("after partial response")
	})
	writer := newBufferedResponseWriter()

	RecoveryMiddleware(nil)(app).ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/partial-panic"),
	})

	response := writer.Response()
	if response.StatusCode != 500 {
		t.Fatalf("status code = %d, want 500", response.StatusCode)
	}
	if strings.Contains(string(response.Body), "partial") {
		t.Fatalf("body = %q, want partial body removed", string(response.Body))
	}
	for _, header := range response.Headers {
		if header.Name == "X-Partial" {
			t.Fatalf("headers = %+v, want partial header removed", response.Headers)
		}
	}
}

func TestRecoveryMiddlewareAllowsLoggingMiddlewareToRecordRecoveredStatus(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		panic("boom")
	})
	handler := Chain(app, LoggingMiddleware(logger), RecoveryMiddleware(nil))
	writer := newBufferedResponseWriter()

	handler.ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/panic"),
	})

	if !strings.Contains(logs.String(), "GET /panic -> 500 22B ") {
		t.Fatalf("logs = %q, want recovered 500 log", logs.String())
	}
}
