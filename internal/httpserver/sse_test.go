package httpserver

import (
	"errors"
	"testing"
)

func TestEncodeServerSentEvent(t *testing.T) {
	t.Parallel()

	got, err := encodeServerSentEvent(ServerSentEvent{
		ID:    "42",
		Event: "message",
		Data:  "hello",
		Retry: 3000,
	})
	if err != nil {
		t.Fatalf("encodeServerSentEvent() error = %v", err)
	}

	want := "id: 42\n" +
		"event: message\n" +
		"retry: 3000\n" +
		"data: hello\n" +
		"\n"
	if got != want {
		t.Fatalf("event = %q, want %q", got, want)
	}
}

func TestEncodeServerSentEventSplitsMultilineData(t *testing.T) {
	t.Parallel()

	got, err := encodeServerSentEvent(ServerSentEvent{
		Data: "first\r\nsecond\rthird",
	})
	if err != nil {
		t.Fatalf("encodeServerSentEvent() error = %v", err)
	}

	want := "data: first\n" +
		"data: second\n" +
		"data: third\n" +
		"\n"
	if got != want {
		t.Fatalf("event = %q, want %q", got, want)
	}
}

func TestEncodeServerSentEventRejectsMalformedFields(t *testing.T) {
	t.Parallel()

	tests := []ServerSentEvent{
		{ID: "bad\nid", Data: "hello"},
		{Event: "bad\revent", Data: "hello"},
		{Data: "hello", Retry: -1},
	}

	for _, event := range tests {
		_, err := encodeServerSentEvent(event)
		if !errors.Is(err, ErrInvalidSSEEvent) {
			t.Fatalf("encodeServerSentEvent(%+v) error = %v, want ErrInvalidSSEEvent", event, err)
		}
	}
}

func TestBufferedResponseWriterWriteEvent(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	if err := writer.WriteEvent(ServerSentEvent{Event: "message", Data: "hello"}); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if !response.Chunked {
		t.Fatal("response.Chunked = false, want true")
	}
	if got := headerValue(response.Headers, "Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if got := headerValue(response.Headers, "Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	wantBody := "event: message\n" +
		"data: hello\n" +
		"\n"
	if string(response.Body) != wantBody {
		t.Fatalf("body = %q, want %q", string(response.Body), wantBody)
	}
}
