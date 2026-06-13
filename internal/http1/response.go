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
	BodyReader   io.Reader
	BodyCloser   io.Closer
	BodyLength   int64
	Chunked      bool
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
	if response.Chunked && version != "HTTP/1.1" {
		return fmt.Errorf("%w: chunked response requires HTTP/1.1", ErrMalformedResponse)
	}
	if response.BodyReader != nil && response.BodyLength < 0 && !response.Chunked {
		return fmt.Errorf("%w: streaming response requires length or chunked encoding", ErrMalformedResponse)
	}
	if response.BodyCloser != nil {
		defer response.BodyCloser.Close()
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
		if strings.EqualFold(header.Name, "Content-Length") ||
			(response.Chunked && strings.EqualFold(header.Name, "Transfer-Encoding")) {
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

	if response.Chunked {
		if err := writeHeaderField(w, HeaderField{
			Name:  "Transfer-Encoding",
			Value: "chunked",
		}); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\r\n"); err != nil {
			return err
		}
		return writeChunkedBody(w, response)
	}

	if err := writeHeaderField(w, HeaderField{
		Name:  "Content-Length",
		Value: strconv.FormatInt(responseBodyLength(response), 10),
	}); err != nil {
		return err
	}

	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	return writeFixedBody(w, response)
}

func responseBodyLength(response Response) int64 {
	if response.BodyReader != nil {
		return response.BodyLength
	}
	return int64(len(response.Body))
}

func writeFixedBody(w io.Writer, response Response) error {
	if response.BodyReader != nil {
		_, err := io.CopyN(w, response.BodyReader, response.BodyLength)
		return err
	}
	if len(response.Body) == 0 {
		return nil
	}

	_, err := w.Write(response.Body)
	return err
}

func writeChunkedBody(w io.Writer, response Response) error {
	if response.BodyReader != nil {
		if err := writeChunkedReader(w, response.BodyReader); err != nil {
			return err
		}
	} else if len(response.Body) > 0 {
		if err := writeChunk(w, response.Body); err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, "0\r\n\r\n")
	return err
}

func writeChunkedReader(w io.Writer, reader io.Reader) error {
	buffer := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buffer)
		if n > 0 {
			if err := writeChunk(w, buffer[:n]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func writeChunk(w io.Writer, body []byte) error {
	if _, err := fmt.Fprintf(w, "%x\r\n", len(body)); err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\r\n")
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
	case 101:
		return "Switching Protocols"
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
	case 401:
		return "Unauthorized"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 408:
		return "Request Timeout"
	case 411:
		return "Length Required"
	case 413:
		return "Content Too Large"
	case 416:
		return "Range Not Satisfiable"
	case 429:
		return "Too Many Requests"
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
