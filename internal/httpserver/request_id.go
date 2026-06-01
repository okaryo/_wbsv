package httpserver

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/okaryo/_wbsv/internal/http1"
)

const RequestIDHeader = "X-Request-ID"

var requestIDCounter uint64

type requestIDContextKey struct{}

// RequestIDMiddleware attaches a request ID to the request context and response.
func RequestIDMiddleware() Middleware {
	return func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			requestID := requestIDFromHeaders(request.HTTP.Headers)
			if requestID == "" {
				requestID = nextRequestID()
			}

			ctx := request.Context
			if ctx == nil {
				ctx = context.Background()
			}
			request.Context = context.WithValue(ctx, requestIDContextKey{}, requestID)
			next.ServeHTTP(w, request)
			w.SetHeader(RequestIDHeader, requestID)
		})
	}
}

// RequestIDFromContext returns the request ID attached by RequestIDMiddleware.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok && requestID != ""
}

func requestIDFromHeaders(headers []http1.HeaderField) string {
	for _, header := range headers {
		if strings.EqualFold(header.Name, RequestIDHeader) {
			return strings.TrimSpace(header.Value)
		}
	}
	return ""
}

func nextRequestID() string {
	id := atomic.AddUint64(&requestIDCounter, 1)
	return fmt.Sprintf("wbsv-%d", id)
}
