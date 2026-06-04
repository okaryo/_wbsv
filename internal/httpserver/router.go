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
	routes    map[string]map[string]AppHandler
	paramRoot *routeNode
}

// NewRouter creates an empty static-path router.
func NewRouter() *Router {
	return &Router{
		routes:    make(map[string]map[string]AppHandler),
		paramRoot: &routeNode{},
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
	for _, match := range r.paramRoot.match(targetPath) {
		if dispatchMethod(w, request, match.methodRoutes, match.params, false) {
			return
		}
		for method, handler := range match.methodRoutes {
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

	r.paramRoot.insert(paramRoute{
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

type routeSegment struct {
	literal      string
	paramName    string
	wildcardName string
}

type routeNode struct {
	routes           []paramRoute
	literalChildren  map[string]*routeNode
	paramChildren    []routeEdge
	wildcardChildren []routeEdge
}

type routeEdge struct {
	segment routeSegment
	child   *routeNode
}

type routeMatch struct {
	methodRoutes map[string]AppHandler
	params       map[string]string
}

func (n *routeNode) insert(route paramRoute) {
	node := n
	for _, segment := range route.segments {
		switch {
		case segment.literal != "":
			node = node.literalChild(segment.literal)
		case segment.paramName != "":
			node = node.paramChild(segment.paramName)
		case segment.wildcardName != "":
			node = node.wildcardChild(segment.wildcardName)
		}
	}

	for i := range node.routes {
		if node.routes[i].pattern == route.pattern {
			for method, handler := range route.methodRoutes {
				node.routes[i].methodRoutes[method] = handler
			}
			return
		}
	}
	node.routes = append(node.routes, route)
}

func (n *routeNode) literalChild(literal string) *routeNode {
	if n.literalChildren == nil {
		n.literalChildren = make(map[string]*routeNode)
	}
	child := n.literalChildren[literal]
	if child == nil {
		child = &routeNode{}
		n.literalChildren[literal] = child
	}
	return child
}

func (n *routeNode) paramChild(name string) *routeNode {
	for _, edge := range n.paramChildren {
		if edge.segment.paramName == name {
			return edge.child
		}
	}

	child := &routeNode{}
	n.paramChildren = append(n.paramChildren, routeEdge{
		segment: routeSegment{paramName: name},
		child:   child,
	})
	return child
}

func (n *routeNode) wildcardChild(name string) *routeNode {
	for _, edge := range n.wildcardChildren {
		if edge.segment.wildcardName == name {
			return edge.child
		}
	}

	child := &routeNode{}
	n.wildcardChildren = append(n.wildcardChildren, routeEdge{
		segment: routeSegment{wildcardName: name},
		child:   child,
	})
	return child
}

func (n *routeNode) match(path string) []routeMatch {
	if n == nil {
		return nil
	}
	return n.matchSegments(splitPath(path), 0, nil)
}

func (n *routeNode) matchSegments(pathSegments []string, index int, params map[string]string) []routeMatch {
	var matches []routeMatch

	if index == len(pathSegments) {
		matches = append(matches, n.terminalMatches(params)...)
		for _, edge := range n.wildcardChildren {
			wildcardParams := withRouteParam(params, edge.segment.wildcardName, "")
			matches = append(matches, edge.child.matchSegments(pathSegments, index, wildcardParams)...)
		}
		return matches
	}

	segment := pathSegments[index]
	if child := n.literalChildren[segment]; child != nil {
		matches = append(matches, child.matchSegments(pathSegments, index+1, params)...)
	}
	for _, edge := range n.paramChildren {
		paramParams := withRouteParam(params, edge.segment.paramName, segment)
		matches = append(matches, edge.child.matchSegments(pathSegments, index+1, paramParams)...)
	}
	for _, edge := range n.wildcardChildren {
		wildcardParams := withRouteParam(params, edge.segment.wildcardName, strings.Join(pathSegments[index:], "/"))
		matches = append(matches, edge.child.matchSegments(pathSegments, len(pathSegments), wildcardParams)...)
	}

	return matches
}

func (n *routeNode) terminalMatches(params map[string]string) []routeMatch {
	if len(n.routes) == 0 {
		return nil
	}

	matches := make([]routeMatch, 0, len(n.routes))
	for _, route := range n.routes {
		matches = append(matches, routeMatch{
			methodRoutes: route.methodRoutes,
			params:       cloneRouteParams(params),
		})
	}
	return matches
}

func withRouteParam(params map[string]string, name string, value string) map[string]string {
	next := cloneRouteParams(params)
	next[name] = value
	return next
}

func cloneRouteParams(params map[string]string) map[string]string {
	cloned := make(map[string]string, len(params)+1)
	for name, value := range params {
		cloned[name] = value
	}
	return cloned
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
