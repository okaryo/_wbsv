package http1

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingHost                 = errors.New("missing Host header")
	ErrUnsupportedTransferEncoding = errors.New("unsupported Transfer-Encoding")
)

// Request is a parsed HTTP/1.x request.
type Request struct {
	RequestLine RequestLine
	Headers     []HeaderField
	Body        []byte
}

// RequestLimits controls bounded request parsing.
type RequestLimits struct {
	MaxHeaders int
	MaxBody    int64
}

// ReadRequest reads one HTTP/1.x request from reader.
func ReadRequest(reader *LineReader, limits RequestLimits) (Request, error) {
	line, err := reader.ReadLine()
	if err != nil {
		return Request{}, err
	}

	requestLine, err := ParseRequestLine(line)
	if err != nil {
		return Request{}, err
	}

	headers, err := ReadHeaderFields(reader, limits.MaxHeaders)
	if err != nil {
		return Request{}, err
	}

	if err := validateRequestHeaders(requestLine, headers); err != nil {
		return Request{}, err
	}

	length, found, err := ContentLength(headers)
	if err != nil {
		return Request{}, err
	}

	var body []byte
	if found {
		body, err = reader.ReadFixedBody(length, limits.MaxBody)
		if err != nil {
			return Request{}, err
		}
	}

	return Request{
		RequestLine: requestLine,
		Headers:     headers,
		Body:        body,
	}, nil
}

func validateRequestHeaders(requestLine RequestLine, headers []HeaderField) error {
	if requestLine.Version == "HTTP/1.1" && !hasNonEmptyHeader(headers, "Host") {
		return fmt.Errorf("%w: HTTP/1.1 requires Host", ErrMissingHost)
	}

	if hasNonEmptyHeader(headers, "Transfer-Encoding") {
		return fmt.Errorf("%w: Transfer-Encoding", ErrUnsupportedTransferEncoding)
	}

	return nil
}

func hasNonEmptyHeader(headers []HeaderField, name string) bool {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) && header.Value != "" {
			return true
		}
	}
	return false
}
