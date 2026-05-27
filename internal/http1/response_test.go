package http1

import (
	"bytes"
	"errors"
	"testing"
)

func TestWriteResponse(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := WriteResponse(&buf, Response{
		StatusCode: 200,
		Headers: []HeaderField{
			{Name: "Content-Type", Value: "text/plain"},
		},
		Body: []byte("hello"),
	})
	if err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	want := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 5\r\n" +
		"\r\n" +
		"hello"
	if got := buf.String(); got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

func TestWriteResponseOverridesContentLength(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := WriteResponse(&buf, Response{
		StatusCode: 200,
		Headers: []HeaderField{
			{Name: "Content-Length", Value: "999"},
		},
		Body: []byte("hello"),
	})
	if err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	want := "HTTP/1.1 200 OK\r\n" +
		"Content-Length: 5\r\n" +
		"\r\n" +
		"hello"
	if got := buf.String(); got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

func TestWriteResponseSupportsCustomReasonPhrase(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := WriteResponse(&buf, Response{
		StatusCode:   299,
		ReasonPhrase: "Custom",
	})
	if err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	want := "HTTP/1.1 299 Custom\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n"
	if got := buf.String(); got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

func TestWriteResponseRejectsInvalidStatusCode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := WriteResponse(&buf, Response{StatusCode: 99})
	if !errors.Is(err, ErrInvalidStatusCode) {
		t.Fatalf("WriteResponse() error = %v, want ErrInvalidStatusCode", err)
	}
}

func TestWriteResponseRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := WriteResponse(&buf, Response{
		StatusCode: 200,
		Headers: []HeaderField{
			{Name: "Bad Header", Value: "value"},
		},
	})
	if !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("WriteResponse() error = %v, want ErrMalformedResponse", err)
	}
}

func TestWriteResponseRejectsMalformedReasonPhrase(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := WriteResponse(&buf, Response{
		StatusCode:   200,
		ReasonPhrase: "OK\r\nInjected",
	})
	if !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("WriteResponse() error = %v, want ErrMalformedResponse", err)
	}
}
