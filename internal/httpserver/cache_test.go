package httpserver

import (
	"errors"
	"testing"
	"time"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestBufferedResponseWriterSetsCacheHeaders(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()
	lastModified := time.Date(2026, 6, 8, 9, 15, 0, 0, time.UTC)

	if err := writer.SetCacheControl("public, max-age=60"); err != nil {
		t.Fatalf("SetCacheControl: %v", err)
	}
	if err := writer.SetETag("resource-v1"); err != nil {
		t.Fatalf("SetETag: %v", err)
	}
	writer.SetLastModified(lastModified)

	response := writer.Response()
	if got := headerValue(response.Headers, "Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("Cache-Control = %q, want public max-age", got)
	}
	if got := headerValue(response.Headers, "ETag"); got != `"resource-v1"` {
		t.Fatalf("ETag = %q, want quoted ETag", got)
	}
	if got := headerValue(response.Headers, "Last-Modified"); got != "Mon, 08 Jun 2026 09:15:00 GMT" {
		t.Fatalf("Last-Modified = %q, want HTTP time", got)
	}
}

func TestBufferedResponseWriterWriteNotModified(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()
	lastModified := time.Date(2026, 6, 8, 9, 15, 0, 0, time.UTC)

	if err := writer.WriteNotModified(`"resource-v1"`, lastModified); err != nil {
		t.Fatalf("WriteNotModified: %v", err)
	}

	response := writer.Response()
	if response.StatusCode != 304 {
		t.Fatalf("status code = %d, want 304", response.StatusCode)
	}
	if got := headerValue(response.Headers, "ETag"); got != `"resource-v1"` {
		t.Fatalf("ETag = %q, want quoted ETag", got)
	}
	if got := headerValue(response.Headers, "Last-Modified"); got != "Mon, 08 Jun 2026 09:15:00 GMT" {
		t.Fatalf("Last-Modified = %q, want HTTP time", got)
	}
}

func TestBufferedResponseWriterRejectsMalformedCacheHeaders(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	if err := writer.SetCacheControl("public\r\nX-Bad: yes"); !errors.Is(err, ErrInvalidCacheHeader) {
		t.Fatalf("SetCacheControl error = %v, want ErrInvalidCacheHeader", err)
	}
	if err := writer.SetETag(`bad"etag`); !errors.Is(err, ErrInvalidCacheHeader) {
		t.Fatalf("SetETag error = %v, want ErrInvalidCacheHeader", err)
	}
}

func TestRequestIfNoneMatch(t *testing.T) {
	t.Parallel()

	request := Request{
		HTTP: testRequest("GET", "/cached"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "If-None-Match", Value: `"old", W/"resource-v1"`},
	}

	if !request.IfNoneMatch("resource-v1") {
		t.Fatal("IfNoneMatch() = false, want true")
	}
	if request.IfNoneMatch("resource-v2") {
		t.Fatal("IfNoneMatch() = true, want false")
	}
}

func TestRequestIfModifiedSince(t *testing.T) {
	t.Parallel()

	request := Request{
		HTTP: testRequest("GET", "/cached"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "If-Modified-Since", Value: "Mon, 08 Jun 2026 09:15:00 GMT"},
	}

	lastModified := time.Date(2026, 6, 8, 9, 15, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 8, 9, 16, 0, 0, time.UTC)

	if !request.IfModifiedSince(lastModified) {
		t.Fatal("IfModifiedSince() = false, want true")
	}
	if request.IfModifiedSince(newer) {
		t.Fatal("IfModifiedSince() = true, want false")
	}
}

func TestRequestNotModifiedPrefersIfNoneMatch(t *testing.T) {
	t.Parallel()

	request := Request{
		HTTP: testRequest("GET", "/cached"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "If-None-Match", Value: `"different"`},
		{Name: "If-Modified-Since", Value: "Mon, 08 Jun 2026 09:15:00 GMT"},
	}

	lastModified := time.Date(2026, 6, 8, 9, 15, 0, 0, time.UTC)

	if request.NotModified("resource-v1", lastModified) {
		t.Fatal("NotModified() = true, want false because If-None-Match takes priority")
	}
}
