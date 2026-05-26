package http1

import (
	"errors"
	"strings"
	"testing"
)

func TestLineReaderReadsCRLFTerminatedLines(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("GET / HTTP/1.1\r\nHost: localhost\r\n"), 1024)

	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine() error = %v", err)
	}
	if line != "GET / HTTP/1.1" {
		t.Fatalf("ReadLine() = %q, want %q", line, "GET / HTTP/1.1")
	}

	line, err = reader.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine() second error = %v", err)
	}
	if line != "Host: localhost" {
		t.Fatalf("ReadLine() second = %q, want %q", line, "Host: localhost")
	}
}

func TestLineReaderRejectsLFOnlyLineEnding(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("GET / HTTP/1.1\n"), 1024)

	_, err := reader.ReadLine()
	if !errors.Is(err, ErrLineEnding) {
		t.Fatalf("ReadLine() error = %v, want ErrLineEnding", err)
	}
}

func TestLineReaderRejectsEOFBeforeCRLF(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("GET / HTTP/1.1"), 1024)

	_, err := reader.ReadLine()
	if !errors.Is(err, ErrUnexpectedEOF) {
		t.Fatalf("ReadLine() error = %v, want ErrUnexpectedEOF", err)
	}
}

func TestLineReaderRejectsLongLine(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("GET /too-long HTTP/1.1\r\n"), len("GET / HTTP/1.1\r\n"))

	_, err := reader.ReadLine()
	if !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("ReadLine() error = %v, want ErrLineTooLong", err)
	}
}

func TestLineReaderStopsAfterMaxLineExceeded(t *testing.T) {
	t.Parallel()

	source := &chunkedStringReader{
		chunks: []string{"GET /too", "-long", " HTTP/1.1\r\n"},
	}
	reader := NewLineReader(source, len("GET /too"))

	_, err := reader.ReadLine()
	if !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("ReadLine() error = %v, want ErrLineTooLong", err)
	}
	if source.reads != 2 {
		t.Fatalf("source reads = %d, want 2", source.reads)
	}
}

func TestLineReaderSupportsIncrementalReads(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(&chunkedStringReader{
		chunks: []string{"GET /", " HTTP", "/1.1\r", "\n"},
	}, 1024)

	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine() error = %v", err)
	}
	if line != "GET / HTTP/1.1" {
		t.Fatalf("ReadLine() = %q, want %q", line, "GET / HTTP/1.1")
	}
}

type chunkedStringReader struct {
	chunks []string
	reads  int
}

func (r *chunkedStringReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, errors.New("unexpected read")
	}

	r.reads++
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	return copy(p, chunk), nil
}
