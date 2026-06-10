package httpserver

import (
	"bytes"
	"compress/gzip"
	"strings"

	"github.com/okaryo/_wbsv/internal/http1"
)

// GzipCompressionMiddleware compresses buffered response bodies when accepted.
func GzipCompressionMiddleware(minBodyBytes int) Middleware {
	return func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			buffer := newBufferedResponseWriter()
			next.ServeHTTP(buffer, request)

			response := buffer.Response()
			if shouldGzipResponse(request, response, minBodyBytes) {
				compressed, err := gzipBytes(response.Body)
				if err == nil {
					response.Body = compressed
					response.Headers = withoutHeader(response.Headers, "Content-Length")
					response.Headers = append(response.Headers,
						http1.HeaderField{Name: "Content-Encoding", Value: "gzip"},
						http1.HeaderField{Name: "Vary", Value: "Accept-Encoding"},
					)
				}
			}

			writeBufferedResponse(w, response)
		})
	}
}

func shouldGzipResponse(request Request, response http1.Response, minBodyBytes int) bool {
	if minBodyBytes < 0 {
		minBodyBytes = 0
	}
	if len(response.Body) < minBodyBytes {
		return false
	}
	if len(response.Body) == 0 || !responseStatusAllowsBody(response.StatusCode) {
		return false
	}
	if hasHeader(response.Headers, "Content-Encoding") {
		return false
	}
	return http1.HeaderHasToken(request.HTTP.Headers, "Accept-Encoding", "gzip")
}

func gzipBytes(body []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeBufferedResponse(w ResponseWriter, response http1.Response) {
	for _, header := range response.Headers {
		if strings.EqualFold(header.Name, "Content-Length") {
			continue
		}
		w.AddHeader(header.Name, header.Value)
	}
	if response.Chunked {
		w.UseChunkedEncoding()
	}
	if response.BodyReader != nil {
		w.WriteHeader(response.StatusCode)
		w.StreamBody(response.BodyReader, response.BodyLength)
		return
	}
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(response.Body)
}

func withoutHeader(headers []http1.HeaderField, name string) []http1.HeaderField {
	filtered := headers[:0]
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			continue
		}
		filtered = append(filtered, header)
	}
	return filtered
}

func hasHeader(headers []http1.HeaderField, name string) bool {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			return true
		}
	}
	return false
}

func responseStatusAllowsBody(statusCode int) bool {
	if statusCode >= 100 && statusCode < 200 {
		return false
	}
	return statusCode != 204 && statusCode != 304
}
