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

func TestWriteResponseOmitsBodyForNoBodyStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response Response
		want     string
	}{
		{
			name: "204",
			response: Response{
				StatusCode: 204,
				Body:       []byte("must not be written"),
			},
			want: "HTTP/1.1 204 No Content\r\n\r\n",
		},
		{
			name: "304",
			response: Response{
				StatusCode: 304,
				Body:       []byte("must not be written"),
			},
			want: "HTTP/1.1 304 Not Modified\r\n\r\n",
		},
		{
			name: "informational",
			response: Response{
				StatusCode:   100,
				ReasonPhrase: "Continue",
				Body:         []byte("must not be written"),
			},
			want: "HTTP/1.1 100 Continue\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := WriteResponse(&buf, tt.response); err != nil {
				t.Fatalf("WriteResponse() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Fatalf("response = %q, want %q", got, tt.want)
			}
		})
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

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	response := ErrorResponse(400, "bad request")

	if err := WriteResponse(&buf, response); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	want := "HTTP/1.1 400 Bad Request\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Content-Length: 12\r\n" +
		"\r\n" +
		"bad request\n"
	if got := buf.String(); got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

func TestErrorResponseUsesStatusTextWhenMessageIsEmpty(t *testing.T) {
	t.Parallel()

	response := ErrorResponse(501, "")

	if string(response.Body) != "Not Implemented\n" {
		t.Fatalf("body = %q, want %q", string(response.Body), "Not Implemented\n")
	}
	if len(response.Headers) != 1 {
		t.Fatalf("headers len = %d, want 1", len(response.Headers))
	}
	if response.Headers[0] != (HeaderField{Name: "Content-Type", Value: "text/plain; charset=utf-8"}) {
		t.Fatalf("header = %#v, want text/plain Content-Type", response.Headers[0])
	}
}
