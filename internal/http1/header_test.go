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

func TestContentLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields []HeaderField
		want   int64
		found  bool
	}{
		{
			name: "missing",
			fields: []HeaderField{
				{Name: "Host", Value: "localhost"},
			},
			found: false,
		},
		{
			name: "present",
			fields: []HeaderField{
				{Name: "Content-Length", Value: "123"},
			},
			want:  123,
			found: true,
		},
		{
			name: "case insensitive name",
			fields: []HeaderField{
				{Name: "content-length", Value: "5"},
			},
			want:  5,
			found: true,
		},
		{
			name: "same duplicate value",
			fields: []HeaderField{
				{Name: "Content-Length", Value: "10"},
				{Name: "content-length", Value: "10"},
			},
			want:  10,
			found: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, found, err := ContentLength(tt.fields)
			if err != nil {
				t.Fatalf("ContentLength() error = %v", err)
			}
			if found != tt.found {
				t.Fatalf("ContentLength() found = %v, want %v", found, tt.found)
			}
			if got != tt.want {
				t.Fatalf("ContentLength() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestContentLengthRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "negative", value: "-1"},
		{name: "explicit plus", value: "+1"},
		{name: "not decimal", value: "abc"},
		{name: "fraction", value: "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := ContentLength([]HeaderField{
				{Name: "Content-Length", Value: tt.value},
			})
			if !errors.Is(err, ErrInvalidContentLength) {
				t.Fatalf("ContentLength() error = %v, want ErrInvalidContentLength", err)
			}
		})
	}
}

func TestContentLengthRejectsConflictingDuplicates(t *testing.T) {
	t.Parallel()

	_, _, err := ContentLength([]HeaderField{
		{Name: "Content-Length", Value: "10"},
		{Name: "Content-Length", Value: "11"},
	})
	if !errors.Is(err, ErrConflictingHeader) {
		t.Fatalf("ContentLength() error = %v, want ErrConflictingHeader", err)
	}
}
