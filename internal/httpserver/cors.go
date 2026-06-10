package httpserver

import (
	"strconv"
	"strings"

	"github.com/okaryo/_wbsv/internal/http1"
)

// CORSOptions configures cross-origin request handling.
type CORSOptions struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// CORSMiddleware adds CORS response headers and handles preflight requests.
func CORSMiddleware(options CORSOptions) Middleware {
	return func(next AppHandler) AppHandler {
		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			origin := headerValueFold(request.HTTP.Headers, "Origin")
			if origin == "" {
				next.ServeHTTP(w, request)
				return
			}

			allowedOrigin := corsAllowedOrigin(origin, options.AllowOrigins, options.AllowCredentials)
			if allowedOrigin == "" {
				next.ServeHTTP(w, request)
				return
			}

			writeCORSActualHeaders(w, allowedOrigin, options)

			if isCORSPreflight(request) {
				writeCORSPreflightHeaders(w, request, options)
				w.WriteHeader(204)
				return
			}

			next.ServeHTTP(w, request)
		})
	}
}

func isCORSPreflight(request Request) bool {
	return strings.EqualFold(request.HTTP.RequestLine.Method, "OPTIONS") &&
		headerValueFold(request.HTTP.Headers, "Access-Control-Request-Method") != ""
}

func writeCORSActualHeaders(w ResponseWriter, allowedOrigin string, options CORSOptions) {
	w.SetHeader("Access-Control-Allow-Origin", allowedOrigin)
	if allowedOrigin != "*" {
		w.AddHeader("Vary", "Origin")
	}
	if options.AllowCredentials {
		w.SetHeader("Access-Control-Allow-Credentials", "true")
	}
	if len(options.ExposeHeaders) > 0 {
		w.SetHeader("Access-Control-Expose-Headers", strings.Join(options.ExposeHeaders, ", "))
	}
}

func writeCORSPreflightHeaders(w ResponseWriter, request Request, options CORSOptions) {
	methods := options.AllowMethods
	if len(methods) == 0 {
		methods = []string{headerValueFold(request.HTTP.Headers, "Access-Control-Request-Method")}
	}
	w.SetHeader("Access-Control-Allow-Methods", strings.Join(methods, ", "))

	headers := options.AllowHeaders
	if len(headers) == 0 {
		headers = splitHeaderList(headerValueFold(request.HTTP.Headers, "Access-Control-Request-Headers"))
	}
	if len(headers) > 0 {
		w.SetHeader("Access-Control-Allow-Headers", strings.Join(headers, ", "))
	}
	if options.MaxAge > 0 {
		w.SetHeader("Access-Control-Max-Age", strconv.Itoa(options.MaxAge))
	}
}

func corsAllowedOrigin(origin string, allowed []string, allowCredentials bool) string {
	for _, candidate := range allowed {
		candidate = strings.TrimSpace(candidate)
		switch {
		case candidate == "*":
			if allowCredentials {
				return origin
			}
			return "*"
		case strings.EqualFold(candidate, origin):
			return origin
		}
	}
	return ""
}

func headerValueFold(headers []http1.HeaderField, name string) string {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}
	return ""
}

func splitHeaderList(value string) []string {
	var headers []string
	for part := range strings.SplitSeq(value, ",") {
		part = strings.Trim(part, " \t")
		if part != "" {
			headers = append(headers, part)
		}
	}
	return headers
}
