package httpserver

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBufferedResponseWriterDefaultsStatusOnWrite(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	n, err := writer.Write([]byte("ok"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != 2 {
		t.Fatalf("written bytes = %d, want 2", n)
	}

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if string(response.Body) != "ok" {
		t.Fatalf("body = %q, want %q", string(response.Body), "ok")
	}
}

func TestBufferedResponseWriterKeepsFirstStatusCode(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	writer.WriteHeader(201)
	writer.WriteHeader(500)

	response := writer.Response()
	if response.StatusCode != 201 {
		t.Fatalf("status code = %d, want 201", response.StatusCode)
	}
}

func TestBufferedResponseWriterCanUseChunkedEncoding(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	writer.UseChunkedEncoding()
	_, _ = writer.Write([]byte("hello"))

	response := writer.Response()
	if !response.Chunked {
		t.Fatal("response.Chunked = false, want true")
	}
	if string(response.Body) != "hello" {
		t.Fatalf("body = %q, want hello", string(response.Body))
	}
}

func TestBufferedResponseWriterCanStreamBody(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()
	reader := strings.NewReader("streamed")

	writer.StreamBody(reader, 8)

	response := writer.Response()
	if response.StatusCode != 200 {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	if response.BodyReader == nil {
		t.Fatal("BodyReader is nil, want stream reader")
	}
	if response.BodyLength != 8 {
		t.Fatalf("BodyLength = %d, want 8", response.BodyLength)
	}
	body, err := io.ReadAll(response.BodyReader)
	if err != nil {
		t.Fatalf("read body reader: %v", err)
	}
	if string(body) != "streamed" {
		t.Fatalf("streamed body = %q, want streamed", string(body))
	}
}

func TestBufferedResponseWriterSendFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("file body"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	writer := newBufferedResponseWriter()
	if err := writer.SendFile(path); err != nil {
		t.Fatalf("SendFile: %v", err)
	}

	response := writer.Response()
	if response.BodyReader == nil {
		t.Fatal("BodyReader is nil, want file reader")
	}
	if response.BodyCloser == nil {
		t.Fatal("BodyCloser is nil, want file closer")
	}
	if response.BodyLength != int64(len("file body")) {
		t.Fatalf("BodyLength = %d, want file size", response.BodyLength)
	}
	if got := headerValue(response.Headers, "Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
	if got := headerValue(response.Headers, "Last-Modified"); got == "" {
		t.Fatal("Last-Modified is empty, want file modification time")
	}
	body, err := io.ReadAll(response.BodyReader)
	if err != nil {
		t.Fatalf("read body reader: %v", err)
	}
	if string(body) != "file body" {
		t.Fatalf("streamed body = %q, want file body", string(body))
	}
	_ = response.BodyCloser.Close()
}

func TestBufferedResponseWriterSetsAndAddsHeaders(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	writer.AddHeader("Set-Cookie", "a=1")
	writer.AddHeader("Set-Cookie", "b=2")
	writer.SetHeader("Content-Type", "text/plain")
	writer.SetHeader("content-type", "application/json")

	response := writer.Response()
	if len(response.Headers) != 3 {
		t.Fatalf("headers count = %d, want 3", len(response.Headers))
	}
	if response.Headers[0].Name != "Set-Cookie" || response.Headers[0].Value != "a=1" {
		t.Fatalf("first header = %+v, want Set-Cookie a=1", response.Headers[0])
	}
	if response.Headers[1].Name != "Set-Cookie" || response.Headers[1].Value != "b=2" {
		t.Fatalf("second header = %+v, want Set-Cookie b=2", response.Headers[1])
	}
	if response.Headers[2].Name != "Content-Type" || response.Headers[2].Value != "application/json" {
		t.Fatalf("third header = %+v, want Content-Type application/json", response.Headers[2])
	}
}
