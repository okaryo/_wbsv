package http1

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMalformedRequestLine = errors.New("malformed request line")
	ErrUnsupportedVersion   = errors.New("unsupported HTTP version")
)

// RequestLine is the first line of an HTTP request.
//
// Example:
//
//	GET /hello HTTP/1.1
type RequestLine struct {
	Method        string
	RequestTarget string
	Version       string
}

// ParseRequestLine parses an HTTP request line without the trailing CRLF.
func ParseRequestLine(line string) (RequestLine, error) {
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return RequestLine{}, fmt.Errorf("%w: expected 3 parts", ErrMalformedRequestLine)
	}

	method, target, version := parts[0], parts[1], parts[2]
	if method == "" || target == "" || version == "" {
		return RequestLine{}, fmt.Errorf("%w: empty part", ErrMalformedRequestLine)
	}

	if strings.ContainsAny(method, "\r\n\t") ||
		strings.ContainsAny(target, "\r\n\t") ||
		strings.ContainsAny(version, "\r\n\t") {
		return RequestLine{}, fmt.Errorf("%w: contains control whitespace", ErrMalformedRequestLine)
	}

	if version != "HTTP/1.1" && version != "HTTP/1.0" {
		return RequestLine{}, fmt.Errorf("%w: %s", ErrUnsupportedVersion, version)
	}

	return RequestLine{
		Method:        method,
		RequestTarget: target,
		Version:       version,
	}, nil
}
