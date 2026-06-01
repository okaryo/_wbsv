package httpserver

// Middleware wraps an application handler with cross-cutting behavior.
type Middleware func(AppHandler) AppHandler

// Chain applies middlewares around handler in the order they are provided.
func Chain(handler AppHandler, middlewares ...Middleware) AppHandler {
	if handler == nil {
		handler = defaultAppHandler
	}

	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] == nil {
			continue
		}
		handler = middlewares[i](handler)
	}

	return handler
}
