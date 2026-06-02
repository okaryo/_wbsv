package httpserver

import (
	"errors"
	"sort"
	"strings"
)

var (
	ErrInvalidRoutePath   = errors.New("invalid route path")
	ErrInvalidRouteMethod = errors.New("invalid route method")
	ErrMissingHandler     = errors.New("missing route handler")
)

const anyMethod = ""

// Router matches static request paths to application handlers.
type Router struct {
	routes map[string]map[string]AppHandler
}

// NewRouter creates an empty static-path router.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]AppHandler),
	}
}

// Handle registers handler for exact path matches with any method.
func (r *Router) Handle(path string, handler AppHandler) error {
	return r.HandleMethod(anyMethod, path, handler)
}

// HandleMethod registers handler for exact method and path matches.
func (r *Router) HandleMethod(method string, path string, handler AppHandler) error {
	method = strings.ToUpper(method)
	if method != anyMethod && strings.ContainsAny(method, " \t\r\n") {
		return ErrInvalidRouteMethod
	}
	if path == "" || !strings.HasPrefix(path, "/") {
		return ErrInvalidRoutePath
	}
	if handler == nil {
		return ErrMissingHandler
	}
	if r.routes[path] == nil {
		r.routes[path] = make(map[string]AppHandler)
	}
	r.routes[path][method] = handler
	return nil
}

// HandleFunc registers handler for exact path matches with any method.
func (r *Router) HandleFunc(path string, handler func(ResponseWriter, Request)) error {
	if handler == nil {
		return ErrMissingHandler
	}
	return r.Handle(path, AppHandlerFunc(handler))
}

// HandleMethodFunc registers handler for exact method and path matches.
func (r *Router) HandleMethodFunc(method string, path string, handler func(ResponseWriter, Request)) error {
	if handler == nil {
		return ErrMissingHandler
	}
	return r.HandleMethod(method, path, AppHandlerFunc(handler))
}

// ServeHTTP dispatches request to the handler registered for its path.
func (r *Router) ServeHTTP(w ResponseWriter, request Request) {
	targetPath := routePath(request.HTTP.RequestLine.RequestTarget)
	methodRoutes := r.routes[targetPath]
	if methodRoutes == nil {
		w.SetHeader("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		_, _ = w.Write([]byte("not found\n"))
		return
	}

	method := strings.ToUpper(request.HTTP.RequestLine.Method)
	handler := methodRoutes[method]
	if handler == nil {
		handler = methodRoutes[anyMethod]
	}
	if handler == nil {
		w.SetHeader("Allow", allowedMethods(methodRoutes))
		w.SetHeader("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(405)
		_, _ = w.Write([]byte("method not allowed\n"))
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

func allowedMethods(methodRoutes map[string]AppHandler) string {
	methods := make([]string, 0, len(methodRoutes))
	for method := range methodRoutes {
		if method == anyMethod {
			continue
		}
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return strings.Join(methods, ", ")
}
