package httpserver

import (
	"testing"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestRequestIDMiddlewareAddsGeneratedRequestID(t *testing.T) {
	t.Parallel()

	var contextRequestID string
	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		var ok bool
		contextRequestID, ok = RequestIDFromContext(request.Context)
		if !ok {
			t.Fatal("request ID was not added to context")
		}
		_, _ = w.Write([]byte("ok"))
	})
	writer := newBufferedResponseWriter()

	RequestIDMiddleware()(app).ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/generated-id"),
	})

	response := writer.Response()
	headerRequestID := headerValue(response.Headers, RequestIDHeader)
	if headerRequestID == "" {
		t.Fatalf("headers = %+v, want generated request ID header", response.Headers)
	}
	if headerRequestID != contextRequestID {
		t.Fatalf("header request ID = %q, context request ID = %q", headerRequestID, contextRequestID)
	}
}

func TestRequestIDMiddlewareReusesIncomingRequestID(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		requestID, ok := RequestIDFromContext(request.Context)
		if !ok {
			t.Fatal("request ID was not added to context")
		}
		w.SetHeader("X-App-Request-ID", requestID)
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/incoming-id"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: RequestIDHeader, Value: "client-request-123"},
	}

	RequestIDMiddleware()(app).ServeHTTP(writer, request)

	response := writer.Response()
	if got := headerValue(response.Headers, RequestIDHeader); got != "client-request-123" {
		t.Fatalf("request ID header = %q, want incoming request ID", got)
	}
	if got := headerValue(response.Headers, "X-App-Request-ID"); got != "client-request-123" {
		t.Fatalf("app request ID header = %q, want incoming request ID", got)
	}
}

func headerValue(headers []http1.HeaderField, name string) string {
	for _, header := range headers {
		if header.Name == name {
			return header.Value
		}
	}
	return ""
}
