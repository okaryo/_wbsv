package http1

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMalformedHeader = errors.New("malformed header")
	ErrTooManyHeaders  = errors.New("too many headers")
)

// HeaderField is one HTTP header field.
type HeaderField struct {
	Name  string
	Value string
}

// ParseHeaderField parses one HTTP header line without the trailing CRLF.
func ParseHeaderField(line string) (HeaderField, error) {
	name, value, ok := strings.Cut(line, ":")
	if !ok {
		return HeaderField{}, fmt.Errorf("%w: missing colon", ErrMalformedHeader)
	}

	if name == "" {
		return HeaderField{}, fmt.Errorf("%w: empty field name", ErrMalformedHeader)
	}

	if strings.ContainsAny(name, " \t\r\n") {
		return HeaderField{}, fmt.Errorf("%w: invalid field name whitespace", ErrMalformedHeader)
	}

	if strings.ContainsAny(value, "\r\n") {
		return HeaderField{}, fmt.Errorf("%w: invalid field value newline", ErrMalformedHeader)
	}

	return HeaderField{
		Name:  name,
		Value: strings.Trim(value, " \t"),
	}, nil
}

// ReadHeaderFields reads header fields until the empty line that terminates the
// HTTP header section.
func ReadHeaderFields(reader *LineReader, maxHeaders int) ([]HeaderField, error) {
	var fields []HeaderField

	for {
		line, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		if line == "" {
			return fields, nil
		}

		if maxHeaders > 0 && len(fields) >= maxHeaders {
			return nil, fmt.Errorf("%w: more than %d headers", ErrTooManyHeaders, maxHeaders)
		}

		field, err := ParseHeaderField(line)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
}
