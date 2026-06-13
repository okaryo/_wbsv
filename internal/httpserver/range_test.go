package httpserver

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestRequestByteRangeParsesClosedRange(t *testing.T) {
	t.Parallel()

	request := requestWithRange("bytes=2-5")

	byteRange, ok, err := request.ByteRange(10)
	if err != nil {
		t.Fatalf("ByteRange() error = %v", err)
	}
	if !ok {
		t.Fatal("ByteRange() ok = false, want true")
	}
	if byteRange != (ByteRange{Start: 2, End: 5, Length: 4}) {
		t.Fatalf("range = %+v, want 2-5 length 4", byteRange)
	}
}

func TestRequestByteRangeClampsEndToRepresentationSize(t *testing.T) {
	t.Parallel()

	request := requestWithRange("bytes=7-99")

	byteRange, ok, err := request.ByteRange(10)
	if err != nil {
		t.Fatalf("ByteRange() error = %v", err)
	}
	if !ok {
		t.Fatal("ByteRange() ok = false, want true")
	}
	if byteRange != (ByteRange{Start: 7, End: 9, Length: 3}) {
		t.Fatalf("range = %+v, want 7-9 length 3", byteRange)
	}
}

func TestRequestByteRangeParsesOpenEndedRange(t *testing.T) {
	t.Parallel()

	request := requestWithRange("bytes=5-")

	byteRange, ok, err := request.ByteRange(10)
	if err != nil {
		t.Fatalf("ByteRange() error = %v", err)
	}
	if !ok {
		t.Fatal("ByteRange() ok = false, want true")
	}
	if byteRange != (ByteRange{Start: 5, End: 9, Length: 5}) {
		t.Fatalf("range = %+v, want 5-9 length 5", byteRange)
	}
}

func TestRequestByteRangeParsesSuffixRange(t *testing.T) {
	t.Parallel()

	request := requestWithRange("bytes=-4")

	byteRange, ok, err := request.ByteRange(10)
	if err != nil {
		t.Fatalf("ByteRange() error = %v", err)
	}
	if !ok {
		t.Fatal("ByteRange() ok = false, want true")
	}
	if byteRange != (ByteRange{Start: 6, End: 9, Length: 4}) {
		t.Fatalf("range = %+v, want 6-9 length 4", byteRange)
	}
}

func TestRequestByteRangeReturnsFalseWhenHeaderMissing(t *testing.T) {
	t.Parallel()

	byteRange, ok, err := Request{HTTP: testRequest("GET", "/file")}.ByteRange(10)
	if err != nil {
		t.Fatalf("ByteRange() error = %v", err)
	}
	if ok {
		t.Fatalf("ByteRange() ok = true with range %+v, want false", byteRange)
	}
}

func TestRequestByteRangeRejectsUnsupportedRanges(t *testing.T) {
	t.Parallel()

	tests := []string{
		"items=0-1",
		"bytes=8-2",
		"bytes=20-30",
		"bytes=-0",
		"bytes=0-1,4-5",
		"bytes=bad",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			_, ok, err := requestWithRange(value).ByteRange(10)
			if !ok {
				t.Fatal("ByteRange() ok = false, want true for present Range header")
			}
			if !errors.Is(err, ErrInvalidRange) {
				t.Fatalf("ByteRange() error = %v, want ErrInvalidRange", err)
			}
		})
	}
}

func TestBufferedResponseWriterStreamRange(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()
	reader := strings.NewReader("0123456789")

	if err := writer.StreamRange(reader, 10, ByteRange{Start: 2, End: 5, Length: 4}); err != nil {
		t.Fatalf("StreamRange: %v", err)
	}

	response := writer.Response()
	if response.StatusCode != 206 {
		t.Fatalf("status code = %d, want 206", response.StatusCode)
	}
	if got := headerValue(response.Headers, "Accept-Ranges"); got != "bytes" {
		t.Fatalf("Accept-Ranges = %q, want bytes", got)
	}
	if got := headerValue(response.Headers, "Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("Content-Range = %q, want bytes 2-5/10", got)
	}
	if response.BodyLength != 4 {
		t.Fatalf("BodyLength = %d, want 4", response.BodyLength)
	}
	body, err := io.ReadAll(response.BodyReader)
	if err != nil {
		t.Fatalf("read body reader: %v", err)
	}
	if string(body) != "2345" {
		t.Fatalf("body = %q, want 2345", string(body))
	}
}

func TestBufferedResponseWriterWriteRangeNotSatisfiable(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()
	writer.WriteRangeNotSatisfiable(10)

	response := writer.Response()
	if response.StatusCode != 416 {
		t.Fatalf("status code = %d, want 416", response.StatusCode)
	}
	if got := headerValue(response.Headers, "Accept-Ranges"); got != "bytes" {
		t.Fatalf("Accept-Ranges = %q, want bytes", got)
	}
	if got := headerValue(response.Headers, "Content-Range"); got != "bytes */10" {
		t.Fatalf("Content-Range = %q, want bytes */10", got)
	}
}

func requestWithRange(value string) Request {
	request := Request{HTTP: testRequest("GET", "/file")}
	request.HTTP.Headers = []http1.HeaderField{{Name: "Range", Value: value}}
	return request
}
