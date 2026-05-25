package http1

import (
	"errors"
	"testing"
)

func TestParseRequestLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want RequestLine
	}{
		{
			name: "origin form target",
			line: "GET /hello HTTP/1.1",
			want: RequestLine{
				Method:        "GET",
				RequestTarget: "/hello",
				Version:       "HTTP/1.1",
			},
		},
		{
			name: "absolute form target",
			line: "GET http://example.com/hello HTTP/1.1",
			want: RequestLine{
				Method:        "GET",
				RequestTarget: "http://example.com/hello",
				Version:       "HTTP/1.1",
			},
		},
		{
			name: "http 1.0",
			line: "HEAD / HTTP/1.0",
			want: RequestLine{
				Method:        "HEAD",
				RequestTarget: "/",
				Version:       "HTTP/1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseRequestLine(tt.line)
			if err != nil {
				t.Fatalf("ParseRequestLine() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("ParseRequestLine() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseRequestLineRejectsMalformedLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
	}{
		{name: "empty", line: ""},
		{name: "missing version", line: "GET /"},
		{name: "missing target", line: "GET  HTTP/1.1"},
		{name: "too many parts", line: "GET / HTTP/1.1 extra"},
		{name: "tab in method", line: "GE\tT / HTTP/1.1"},
		{name: "tab in target", line: "GET /foo\tbar HTTP/1.1"},
		{name: "newline in target", line: "GET /\nhello HTTP/1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseRequestLine(tt.line)
			if !errors.Is(err, ErrMalformedRequestLine) {
				t.Fatalf("ParseRequestLine() error = %v, want ErrMalformedRequestLine", err)
			}
		})
	}
}

func TestParseRequestLineRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	_, err := ParseRequestLine("GET / HTTP/2")
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("ParseRequestLine() error = %v, want ErrUnsupportedVersion", err)
	}
}
