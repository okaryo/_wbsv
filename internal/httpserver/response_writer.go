package httpserver

import (
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
	WriteNotModified(etag string, lastModified time.Time) error
	WriteHeader(statusCode int)
	Write([]byte) (int, error)
}

type bufferedResponseWriter struct {
	statusCode int
	headers    []http1.HeaderField
	body       []byte
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
	}
}

func (w *bufferedResponseWriter) reset() {
	w.statusCode = 0
	w.headers = nil
	w.body = nil
}
