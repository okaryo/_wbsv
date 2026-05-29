package http1

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ErrInvalidStatusCode = errors.New("invalid status code")
	ErrMalformedResponse = errors.New("malformed response")
)

// Response is a minimal HTTP/1.x response.
type Response struct {
	Version      string
	StatusCode   int
	ReasonPhrase string
	Headers      []HeaderField
	Body         []byte
}

// WriteResponse writes a fixed-length HTTP/1.x response.
func WriteResponse(w io.Writer, response Response) error {
	version := response.Version
	if version == "" {
		version = "HTTP/1.1"
	}
	if version != "HTTP/1.1" && version != "HTTP/1.0" {
		return fmt.Errorf("%w: %s", ErrUnsupportedVersion, version)
	}
	if response.StatusCode < 100 || response.StatusCode > 999 {
		return fmt.Errorf("%w: %d", ErrInvalidStatusCode, response.StatusCode)
	}

	reason := response.ReasonPhrase
	if reason == "" {
		reason = StatusText(response.StatusCode)
	}
	if strings.ContainsAny(reason, "\r\n") {
		return fmt.Errorf("%w: invalid reason phrase", ErrMalformedResponse)
	}

	if _, err := fmt.Fprintf(w, "%s %d %s\r\n", version, response.StatusCode, reason); err != nil {
		return err
	}

	for _, header := range response.Headers {
		if strings.EqualFold(header.Name, "Content-Length") {
			continue
		}
		if err := writeHeaderField(w, header); err != nil {
			return err
		}
	}

	if !statusAllowsBody(response.StatusCode) {
		_, err := io.WriteString(w, "\r\n")
		return err
	}

	if err := writeHeaderField(w, HeaderField{
		Name:  "Content-Length",
		Value: strconv.Itoa(len(response.Body)),
	}); err != nil {
		return err
	}

	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	if len(response.Body) == 0 {
		return nil
	}

	_, err := w.Write(response.Body)
	return err
}

// ErrorResponse returns a small plain-text error response.
func ErrorResponse(statusCode int, message string) Response {
	if message == "" {
		message = StatusText(statusCode)
	}
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}

	return WithContentType(Response{
		StatusCode: statusCode,
		Body:       []byte(message),
	}, "text/plain; charset=utf-8")
}

// StatusText returns a reason phrase for common HTTP status codes.
func StatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 411:
		return "Length Required"
	case 413:
		return "Content Too Large"
	case 500:
		return "Internal Server Error"
	case 501:
		return "Not Implemented"
	default:
		return "Status"
	}
}

func statusAllowsBody(statusCode int) bool {
	if statusCode >= 100 && statusCode < 200 {
		return false
	}
	return statusCode != 204 && statusCode != 304
}

func writeHeaderField(w io.Writer, header HeaderField) error {
	if header.Name == "" || strings.ContainsAny(header.Name, " \t\r\n") {
		return fmt.Errorf("%w: invalid header name", ErrMalformedResponse)
	}
	if strings.ContainsAny(header.Value, "\r\n") {
		return fmt.Errorf("%w: invalid header value", ErrMalformedResponse)
	}

	_, err := fmt.Fprintf(w, "%s: %s\r\n", header.Name, header.Value)
	return err
}
