package httpserver

import "log"

// RecoveryMiddleware converts application panics into 500 responses.
func RecoveryMiddleware(logger *log.Logger) Middleware {
	return func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			defer func() {
				panicValue := recover()
				if panicValue == nil {
					return
				}

				if logger != nil {
					logger.Printf(
						"panic while handling %s %s: %v",
						request.HTTP.RequestLine.Method,
						request.HTTP.RequestLine.RequestTarget,
						panicValue,
					)
				}

				resetResponse(w)
				w.SetHeader("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(500)
				_, _ = w.Write([]byte("internal server error\n"))
			}()

			next.ServeHTTP(w, request)
		})
	}
}

func resetResponse(w ResponseWriter) {
	if resetter, ok := w.(interface{ reset() }); ok {
		resetter.reset()
	}
}
