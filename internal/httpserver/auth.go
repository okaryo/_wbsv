package httpserver

import (
	"crypto/subtle"
	"strings"

	"github.com/okaryo/_wbsv/internal/http1"
)

// BearerAuthMiddleware allows requests with a matching Authorization bearer token.
func BearerAuthMiddleware(token string) Middleware {
	return func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			if token == "" || bearerTokenMatches(request.HTTP.Headers, token) {
				next.ServeHTTP(w, request)
				return
			}

			w.SetHeader("WWW-Authenticate", `Bearer realm="wbsv"`)
			w.SetHeader("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(401)
			_, _ = w.Write([]byte("unauthorized\n"))
		})
	}
}

func bearerTokenMatches(headers []http1.HeaderField, want string) bool {
	for _, header := range headers {
		if !strings.EqualFold(header.Name, "Authorization") {
			continue
		}

		scheme, token, ok := strings.Cut(strings.TrimSpace(header.Value), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") {
			continue
		}
		token = strings.TrimSpace(token)
		if subtle.ConstantTimeCompare([]byte(token), []byte(want)) == 1 {
			return true
		}
	}

	return false
}
