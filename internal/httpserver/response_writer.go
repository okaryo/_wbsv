package httpserver

import (
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/okaryo/_wbsv/internal/http1"
)

// ResponseWriter lets application handlers build a response.
type ResponseWriter interface {
	AddHeader(name, value string)
	SetHeader(name, value string)
	SetCacheControl(value string) error
	SetCookie(cookie Cookie) error
	SetETag(etag string) error
	SetLastModified(lastModified time.Time)
	StreamBody(reader io.Reader, contentLength int64)
	SendFile(path string) error
	UseChunkedEncoding()
	WriteNotModified(etag string, lastModified time.Time) error
	WriteHeader(statusCode int)
	Write([]byte) (int, error)
}

type bufferedResponseWriter struct {
	statusCode int
	headers    []http1.HeaderField
	body       []byte
	bodyReader io.Reader
	bodyCloser io.Closer
	bodyLength int64
	chunked    bool
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{}
}

func (w *bufferedResponseWriter) AddHeader(name, value string) {
	w.headers = append(w.headers, http1.HeaderField{Name: name, Value: value})
}

func (w *bufferedResponseWriter) SetHeader(name, value string) {
	for i, header := range w.headers {
		if strings.EqualFold(header.Name, name) {
			w.headers[i].Value = value
			return
		}
	}

	w.AddHeader(name, value)
}

func (w *bufferedResponseWriter) SetCacheControl(value string) error {
	if invalidCacheHeaderValue(value) {
		return ErrInvalidCacheHeader
	}
	w.SetHeader("Cache-Control", value)
	return nil
}

func (w *bufferedResponseWriter) SetCookie(cookie Cookie) error {
	value, err := cookie.setCookieValue()
	if err != nil {
		return err
	}

	w.AddHeader("Set-Cookie", value)
	return nil
}

func (w *bufferedResponseWriter) SetETag(etag string) error {
	value, err := responseETagValue(etag)
	if err != nil {
		return err
	}
	w.SetHeader("ETag", value)
	return nil
}

func (w *bufferedResponseWriter) SetLastModified(lastModified time.Time) {
	if lastModified.IsZero() {
		return
	}
	w.SetHeader("Last-Modified", httpTime(lastModified))
}

func (w *bufferedResponseWriter) StreamBody(reader io.Reader, contentLength int64) {
	if w.statusCode == 0 {
		w.statusCode = 200
	}
	w.body = nil
	w.bodyReader = reader
	w.bodyCloser = nil
	if closer, ok := reader.(io.Closer); ok {
		w.bodyCloser = closer
	}
	w.bodyLength = contentLength
}

func (w *bufferedResponseWriter) SendFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	if info.IsDir() {
		_ = file.Close()
		return os.ErrInvalid
	}

	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		w.SetHeader("Content-Type", contentType)
	}
	w.SetLastModified(info.ModTime())
	w.StreamBody(file, info.Size())
	return nil
}

func (w *bufferedResponseWriter) UseChunkedEncoding() {
	w.chunked = true
}

func (w *bufferedResponseWriter) WriteNotModified(etag string, lastModified time.Time) error {
	if etag != "" {
		if err := w.SetETag(etag); err != nil {
			return err
		}
	}
	w.SetLastModified(lastModified)
	w.WriteHeader(304)
	return nil
}

func (w *bufferedResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = statusCode
}

func (w *bufferedResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = 200
	}
	w.bodyReader = nil
	w.bodyCloser = nil
	w.bodyLength = 0
	w.body = append(w.body, p...)
	return len(p), nil
}

func (w *bufferedResponseWriter) Response() http1.Response {
	statusCode := w.statusCode
	if statusCode == 0 {
		statusCode = 200
	}

	return http1.Response{
		StatusCode: statusCode,
		Headers:    append([]http1.HeaderField(nil), w.headers...),
		Body:       append([]byte(nil), w.body...),
		BodyReader: w.bodyReader,
		BodyCloser: w.bodyCloser,
		BodyLength: w.bodyLength,
		Chunked:    w.chunked,
	}
}

func (w *bufferedResponseWriter) reset() {
	w.statusCode = 0
	w.headers = nil
	w.body = nil
	w.bodyReader = nil
	w.bodyCloser = nil
	w.bodyLength = 0
	w.chunked = false
}
