package http1

import (
	"errors"
	"strings"
	"testing"
)

func TestParseHeaderField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want HeaderField
	}{
		{
			name: "host",
			line: "Host: localhost:8080",
			want: HeaderField{Name: "Host", Value: "localhost:8080"},
		},
		{
			name: "trims optional whitespace around value",
			line: "Content-Type: \t text/plain \t",
			want: HeaderField{Name: "Content-Type", Value: "text/plain"},
		},
		{
			name: "empty value",
			line: "X-Empty:",
			want: HeaderField{Name: "X-Empty", Value: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseHeaderField(tt.line)
			if err != nil {
				t.Fatalf("ParseHeaderField() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseHeaderField() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseHeaderFieldRejectsMalformedLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
	}{
		{name: "missing colon", line: "Host localhost"},
		{name: "empty name", line: ": value"},
		{name: "space in name", line: "Bad Name: value"},
		{name: "tab in name", line: "Bad\tName: value"},
		{name: "newline in value", line: "X-Test: hello\nworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseHeaderField(tt.line)
			if !errors.Is(err, ErrMalformedHeader) {
				t.Fatalf("ParseHeaderField() error = %v, want ErrMalformedHeader", err)
			}
		})
	}
}

func TestReadHeaderFields(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"Host: localhost\r\n"+
			"Accept: */*\r\n"+
			"\r\n"+
			"body starts here",
	), 1024)

	got, err := ReadHeaderFields(reader, 10)
	if err != nil {
		t.Fatalf("ReadHeaderFields() error = %v", err)
	}

	want := []HeaderField{
		{Name: "Host", Value: "localhost"},
		{Name: "Accept", Value: "*/*"},
	}
	if len(got) != len(want) {
		t.Fatalf("ReadHeaderFields() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ReadHeaderFields()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestReadHeaderFieldsRejectsTooManyHeaders(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader(
		"A: 1\r\n"+
			"B: 2\r\n"+
			"\r\n",
	), 1024)

	_, err := ReadHeaderFields(reader, 1)
	if !errors.Is(err, ErrTooManyHeaders) {
		t.Fatalf("ReadHeaderFields() error = %v, want ErrTooManyHeaders", err)
	}
}

func TestReadHeaderFieldsRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	reader := NewLineReader(strings.NewReader("Host localhost\r\n\r\n"), 1024)

	_, err := ReadHeaderFields(reader, 10)
	if !errors.Is(err, ErrMalformedHeader) {
		t.Fatalf("ReadHeaderFields() error = %v, want ErrMalformedHeader", err)
	}
}
