package http1

import (
	"errors"
	"strings"
	"testing"
)

func TestReadRequest(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"POST /messages HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Content-Length: 11\r\n"+
			"\r\n"+
			"hello world",
	), 1024)

	got, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}

	if got.RequestLine.Method != "POST" {
		t.Fatalf("method = %q, want %q", got.RequestLine.Method, "POST")
	}
	if got.RequestLine.RequestTarget != "/messages" {
		t.Fatalf("target = %q, want %q", got.RequestLine.RequestTarget, "/messages")
	}
	if string(got.Body) != "hello world" {
		t.Fatalf("body = %q, want %q", string(got.Body), "hello world")
	}
	if len(got.Headers) != 2 {
		t.Fatalf("headers len = %d, want 2", len(got.Headers))
	}
}

func TestReadRequestWithoutBody(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"GET / HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"\r\n",
	), 1024)

	got, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
	if got.Body != nil {
		t.Fatalf("body = %q, want nil", string(got.Body))
	}
}

func TestReadRequestAllowsHTTP10WithoutHost(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"GET / HTTP/1.0\r\n"+
			"\r\n",
	), 1024)

	_, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if err != nil {
		t.Fatalf("ReadRequest() error = %v", err)
	}
}

func TestReadRequestRejectsHTTP11WithoutHost(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"GET / HTTP/1.1\r\n"+
			"\r\n",
	), 1024)

	_, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if !errors.Is(err, ErrMissingHost) {
		t.Fatalf("ReadRequest() error = %v, want ErrMissingHost", err)
	}
}

func TestReadRequestRejectsEmptyHTTP11Host(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"GET / HTTP/1.1\r\n"+
			"Host:\r\n"+
			"\r\n",
	), 1024)

	_, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if !errors.Is(err, ErrMissingHost) {
		t.Fatalf("ReadRequest() error = %v, want ErrMissingHost", err)
	}
}

func TestReadRequestRejectsTransferEncoding(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"POST / HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Transfer-Encoding: chunked\r\n"+
			"\r\n",
	), 1024)

	_, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if !errors.Is(err, ErrUnsupportedTransferEncoding) {
		t.Fatalf("ReadRequest() error = %v, want ErrUnsupportedTransferEncoding", err)
	}
}

func TestReadRequestPropagatesMalformedRequestLine(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("GET /\r\n\r\n"), 1024)

	_, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if !errors.Is(err, ErrMalformedRequestLine) {
		t.Fatalf("ReadRequest() error = %v, want ErrMalformedRequestLine", err)
	}
}

func TestReadRequestPropagatesMalformedHeader(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"GET / HTTP/1.1\r\n"+
			"Host localhost\r\n"+
			"\r\n",
	), 1024)

	_, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if !errors.Is(err, ErrMalformedHeader) {
		t.Fatalf("ReadRequest() error = %v, want ErrMalformedHeader", err)
	}
}

func TestReadRequestPropagatesBodyErrors(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"POST / HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Content-Length: 10\r\n"+
			"\r\n"+
			"short",
	), 1024)

	_, err := ReadRequest(reader, RequestLimits{
		MaxHeaders: 10,
		MaxBody:    1024,
	})
	if !errors.Is(err, ErrUnexpectedBodyEOF) {
		t.Fatalf("ReadRequest() error = %v, want ErrUnexpectedBodyEOF", err)
	}
}
