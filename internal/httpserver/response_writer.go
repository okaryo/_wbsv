package httpserver

import (
	"strings"

	"github.com/okaryo/_wbsv/internal/http1"
)

// ResponseWriter lets application handlers build a response.
type ResponseWriter interface {
	AddHeader(name, value string)
	SetHeader(name, value string)
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
