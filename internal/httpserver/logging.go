package httpserver

import (
	"log"
	"time"
)

// LoggingMiddleware logs one line after each application request is handled.
func LoggingMiddleware(logger *log.Logger) Middleware {
	return func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			recorder := &loggingResponseWriter{ResponseWriter: w}
			startedAt := time.Now()

			next.ServeHTTP(recorder, request)

			if logger != nil {
				logger.Printf(
					"%s %s -> %d %dB %s",
					request.HTTP.RequestLine.Method,
					request.HTTP.RequestLine.RequestTarget,
					recorder.StatusCode(),
					recorder.BytesWritten(),
					time.Since(startedAt),
				)
			}
		})
	}
}

type loggingResponseWriter struct {
	ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = 200
	}

	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += n
	return n, err
}

func (w *loggingResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return 200
	}
	return w.statusCode
}

func (w *loggingResponseWriter) BytesWritten() int {
	return w.bytesWritten
}

func (w *loggingResponseWriter) reset() {
	if resetter, ok := w.ResponseWriter.(interface{ reset() }); ok {
		resetter.reset()
	}
	w.statusCode = 0
	w.bytesWritten = 0
}
