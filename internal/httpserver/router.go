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
	routes      map[string]map[string]AppHandler
	paramRoutes []paramRoute
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
	if hasPathParams(path) {
		return r.handleParamRoute(method, path, handler)
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
	if methodRoutes != nil {
		if dispatchMethod(w, request, methodRoutes, nil, true) {
			return
		}
		return
	}

	allowed := make(map[string]AppHandler)
	for _, route := range r.paramRoutes {
		if route.hasWildcard() {
			continue
		}
		params, ok := route.match(targetPath)
		if !ok {
			continue
		}

		if dispatchMethod(w, request, route.methodRoutes, params, false) {
			return
		}
		for method, handler := range route.methodRoutes {
			allowed[method] = handler
		}
	}

	for _, route := range r.paramRoutes {
		if !route.hasWildcard() {
			continue
		}
		params, ok := route.match(targetPath)
		if !ok {
			continue
		}

		if dispatchMethod(w, request, route.methodRoutes, params, false) {
			return
		}
		for method, handler := range route.methodRoutes {
			allowed[method] = handler
		}
	}

	if len(allowed) > 0 {
		writeMethodNotAllowed(w, allowed)
		return
	}

	w.SetHeader("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(404)
	_, _ = w.Write([]byte("not found\n"))
}

func dispatchMethod(w ResponseWriter, request Request, methodRoutes map[string]AppHandler, params map[string]string, writeFailure bool) bool {
	method := strings.ToUpper(request.HTTP.RequestLine.Method)
	handler := methodRoutes[method]
	if handler == nil {
		handler = methodRoutes[anyMethod]
	}
	if handler == nil {
		if writeFailure {
			writeMethodNotAllowed(w, methodRoutes)
			return true
		}
		return false
	}

	if params != nil {
		request.Params = params
	}

	handler.ServeHTTP(w, request)
	return true
}

func writeMethodNotAllowed(w ResponseWriter, methodRoutes map[string]AppHandler) {
	w.SetHeader("Allow", allowedMethods(methodRoutes))
	w.SetHeader("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(405)
	_, _ = w.Write([]byte("method not allowed\n"))
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

func (r *Router) handleParamRoute(method string, path string, handler AppHandler) error {
	segments, err := parseRouteSegments(path)
	if err != nil {
		return err
	}

	for i := range r.paramRoutes {
		if r.paramRoutes[i].pattern == path {
			r.paramRoutes[i].methodRoutes[method] = handler
			return nil
		}
	}

	r.paramRoutes = append(r.paramRoutes, paramRoute{
		pattern:      path,
		segments:     segments,
		methodRoutes: map[string]AppHandler{method: handler},
	})
	return nil
}

type paramRoute struct {
	pattern      string
	segments     []routeSegment
	methodRoutes map[string]AppHandler
}

func (r paramRoute) match(path string) (map[string]string, bool) {
	pathSegments := splitPath(path)
	if !r.hasWildcard() && len(pathSegments) != len(r.segments) {
		return nil, false
	}
	if r.hasWildcard() && len(pathSegments) < len(r.segments)-1 {
		return nil, false
	}

	params := make(map[string]string)
	for i, routeSegment := range r.segments {
		if routeSegment.wildcardName != "" {
			params[routeSegment.wildcardName] = strings.Join(pathSegments[i:], "/")
			return params, true
		}

		pathSegment := pathSegments[i]
		if routeSegment.paramName != "" {
			params[routeSegment.paramName] = pathSegment
			continue
		}
		if routeSegment.literal != pathSegment {
			return nil, false
		}
	}

	return params, true
}

func (r paramRoute) hasWildcard() bool {
	return len(r.segments) > 0 && r.segments[len(r.segments)-1].wildcardName != ""
}

type routeSegment struct {
	literal      string
	paramName    string
	wildcardName string
}

func parseRouteSegments(path string) ([]routeSegment, error) {
	rawSegments := splitPath(path)
	segments := make([]routeSegment, 0, len(rawSegments))
	seenParams := make(map[string]struct{})

	for i, rawSegment := range rawSegments {
		if strings.HasPrefix(rawSegment, ":") {
			name := strings.TrimPrefix(rawSegment, ":")
			if invalidRouteParamName(name) {
				return nil, ErrInvalidRoutePath
			}
			if _, ok := seenParams[name]; ok {
				return nil, ErrInvalidRoutePath
			}
			seenParams[name] = struct{}{}
			segments = append(segments, routeSegment{paramName: name})
			continue
		}
		if strings.HasPrefix(rawSegment, "*") {
			name := strings.TrimPrefix(rawSegment, "*")
			if invalidRouteParamName(name) || i != len(rawSegments)-1 {
				return nil, ErrInvalidRoutePath
			}
			if _, ok := seenParams[name]; ok {
				return nil, ErrInvalidRoutePath
			}
			seenParams[name] = struct{}{}
			segments = append(segments, routeSegment{wildcardName: name})
			continue
		}
		segments = append(segments, routeSegment{literal: rawSegment})
	}

	return segments, nil
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func hasPathParams(path string) bool {
	for _, segment := range splitPath(path) {
		if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*") {
			return true
		}
	}
	return false
}

func invalidRouteParamName(name string) bool {
	return name == "" || strings.ContainsAny(name, " \t\r\n")
}
