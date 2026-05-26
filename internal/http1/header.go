package http1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrMalformedHeader      = errors.New("malformed header")
	ErrTooManyHeaders       = errors.New("too many headers")
	ErrInvalidContentLength = errors.New("invalid Content-Length")
	ErrConflictingHeader    = errors.New("conflicting header")
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

// ContentLength returns the Content-Length value from the header fields.
//
// The returned bool is false when no Content-Length header is present.
func ContentLength(fields []HeaderField) (int64, bool, error) {
	var (
		length int64
		found  bool
	)

	for _, field := range fields {
		if !strings.EqualFold(field.Name, "Content-Length") {
			continue
		}

		current, err := parseContentLengthValue(field.Value)
		if err != nil {
			return 0, false, err
		}

		if found && current != length {
			return 0, false, fmt.Errorf("%w: Content-Length", ErrConflictingHeader)
		}

		length = current
		found = true
	}

	return length, found, nil
}

func parseContentLengthValue(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("%w: empty", ErrInvalidContentLength)
	}
	if strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return 0, fmt.Errorf("%w: signed value", ErrInvalidContentLength)
	}

	length, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidContentLength, value)
	}
	if length < 0 {
		return 0, fmt.Errorf("%w: negative", ErrInvalidContentLength)
	}

	return length, nil
}
