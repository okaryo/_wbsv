package httpserver

import (
	"errors"
	"strings"
)

var (
	ErrInvalidRoutePath = errors.New("invalid route path")
	ErrMissingHandler   = errors.New("missing route handler")
)

// Router matches static request paths to application handlers.
type Router struct {
	routes map[string]AppHandler
}

// NewRouter creates an empty static-path router.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]AppHandler),
	}
}

// Handle registers handler for exact path matches.
func (r *Router) Handle(path string, handler AppHandler) error {
	if path == "" || !strings.HasPrefix(path, "/") {
		return ErrInvalidRoutePath
	}
	if handler == nil {
		return ErrMissingHandler
	}
	r.routes[path] = handler
	return nil
}

// HandleFunc registers handler for exact path matches.
func (r *Router) HandleFunc(path string, handler func(ResponseWriter, Request)) error {
	if handler == nil {
		return ErrMissingHandler
	}
	return r.Handle(path, AppHandlerFunc(handler))
}

// ServeHTTP dispatches request to the handler registered for its path.
func (r *Router) ServeHTTP(w ResponseWriter, request Request) {
	targetPath := routePath(request.HTTP.RequestLine.RequestTarget)
	handler := r.routes[targetPath]
	if handler == nil {
		w.SetHeader("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		_, _ = w.Write([]byte("not found\n"))
		return
	}

	handler.ServeHTTP(w, request)
}

func routePath(target string) string {
	path := target
	if queryStart := strings.IndexByte(path, '?'); queryStart >= 0 {
		path = path[:queryStart]
	}
	if path == "" {
		return "/"
	}
	return path
}
