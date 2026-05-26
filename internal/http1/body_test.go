package http1

import (
	"errors"
	"strings"
	"testing"
)

func TestLineReaderReadsFixedBody(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"POST / HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Content-Length: 11\r\n"+
			"\r\n"+
			"hello world"+
			"GET /next HTTP/1.1\r\n",
	), 1024)

	if _, err := reader.ReadLine(); err != nil {
		t.Fatalf("read request line: %v", err)
	}
	if _, err := ReadHeaderFields(reader, 10); err != nil {
		t.Fatalf("read headers: %v", err)
	}

	body, err := reader.ReadFixedBody(11, 1024)
	if err != nil {
		t.Fatalf("ReadFixedBody() error = %v", err)
	}
	if string(body) != "hello world" {
		t.Fatalf("ReadFixedBody() = %q, want %q", string(body), "hello world")
	}

	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("read next line: %v", err)
	}
	if line != "GET /next HTTP/1.1" {
		t.Fatalf("next line = %q, want %q", line, "GET /next HTTP/1.1")
	}
}

func TestLineReaderReadsFixedBodyFromIncrementalSource(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(&chunkedStringReader{
		chunks: []string{"hel", "lo ", "world"},
	}, 1024)

	body, err := reader.ReadFixedBody(11, 1024)
	if err != nil {
		t.Fatalf("ReadFixedBody() error = %v", err)
	}
	if string(body) != "hello world" {
		t.Fatalf("ReadFixedBody() = %q, want %q", string(body), "hello world")
	}
}

func TestLineReaderReadFixedBodyReturnsNilForZeroLength(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("unused"), 1024)

	body, err := reader.ReadFixedBody(0, 1024)
	if err != nil {
		t.Fatalf("ReadFixedBody() error = %v", err)
	}
	if body != nil {
		t.Fatalf("ReadFixedBody() = %q, want nil", string(body))
	}
}

func TestLineReaderReadFixedBodyRejectsTooLargeBody(t *testing.T) {
	t.Parallel()

	source := &chunkedStringReader{
		chunks: []string{"hello"},
	}
	reader := NewLineReader(source, 1024)

	_, err := reader.ReadFixedBody(6, 5)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("ReadFixedBody() error = %v, want ErrBodyTooLarge", err)
	}
	if source.reads != 0 {
		t.Fatalf("source reads = %d, want 0", source.reads)
	}
}

func TestLineReaderReadFixedBodyRejectsUnexpectedEOF(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("hello"), 1024)

	_, err := reader.ReadFixedBody(6, 1024)
	if !errors.Is(err, ErrUnexpectedBodyEOF) {
		t.Fatalf("ReadFixedBody() error = %v, want ErrUnexpectedBodyEOF", err)
	}
}
