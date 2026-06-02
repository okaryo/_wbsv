package httpserver

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestGzipCompressionMiddlewareCompressesAcceptedResponse(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		w.SetHeader("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello hello hello\n"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/gzip"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Accept-Encoding", Value: "br, gzip"},
	}

	GzipCompressionMiddleware(1)(app).ServeHTTP(writer, request)

	response := writer.Response()
	if got := headerValue(response.Headers, "Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := headerValue(response.Headers, "Vary"); got != "Accept-Encoding" {
		t.Fatalf("Vary = %q, want Accept-Encoding", got)
	}

	body := gunzip(t, response.Body)
	if body != "hello hello hello\n" {
		t.Fatalf("decompressed body = %q, want original body", body)
	}
}

func TestGzipCompressionMiddlewareSkipsWhenNotAccepted(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("plain\n"))
	})
	writer := newBufferedResponseWriter()

	GzipCompressionMiddleware(1)(app).ServeHTTP(writer, Request{
		HTTP: testRequest("GET", "/plain"),
	})

	response := writer.Response()
	if got := headerValue(response.Headers, "Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if string(response.Body) != "plain\n" {
		t.Fatalf("body = %q, want plain body", string(response.Body))
	}
}

func TestGzipCompressionMiddlewareSkipsExistingContentEncoding(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		w.SetHeader("Content-Encoding", "identity")
		_, _ = w.Write([]byte("already encoded"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/encoded"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Accept-Encoding", Value: "gzip"},
	}

	GzipCompressionMiddleware(1)(app).ServeHTTP(writer, request)

	response := writer.Response()
	if got := headerValue(response.Headers, "Content-Encoding"); got != "identity" {
		t.Fatalf("Content-Encoding = %q, want identity", got)
	}
	if string(response.Body) != "already encoded" {
		t.Fatalf("body = %q, want unmodified body", string(response.Body))
	}
}

func TestGzipCompressionMiddlewareSkipsSmallResponse(t *testing.T) {
	t.Parallel()

	app := AppHandlerFunc(func(w ResponseWriter, request Request) {
		_, _ = w.Write([]byte("small"))
	})
	writer := newBufferedResponseWriter()
	request := Request{
		HTTP: testRequest("GET", "/small"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Accept-Encoding", Value: "gzip"},
	}

	GzipCompressionMiddleware(10)(app).ServeHTTP(writer, request)

	response := writer.Response()
	if got := headerValue(response.Headers, "Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if string(response.Body) != "small" {
		t.Fatalf("body = %q, want small body", string(response.Body))
	}
}

func gunzip(t *testing.T, body []byte) string {
	t.Helper()

	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new gzip reader: %v", err)
	}
	defer reader.Close()

	plain, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	return string(plain)
}
