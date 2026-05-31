# Handler Model

The HTTP server now has a small application handler abstraction.

```go
type Request struct {
	Context context.Context
	HTTP    http1.Request
}

type AppHandler interface {
	ServeHTTP(Request) http1.Response
}
```

This separates two responsibilities:

- The server layer owns TCP connections, timeouts, HTTP parsing, response
  writing, and keep-alive decisions.
- The application layer receives an application request with a context and a
  parsed HTTP request, then returns a response.

The first version is intentionally simpler than `net/http`. It returns a whole
`http1.Response` value instead of writing through a streaming `ResponseWriter`.
That keeps the current learning focus on the boundary between protocol handling
and application behavior.

`AppHandlerFunc` adapts a function to the interface:

```go
handler := httpserver.AppHandlerFunc(func(request httpserver.Request) http1.Response {
	select {
	case <-request.Context.Done():
		return http1.ErrorResponse(408, "request canceled")
	default:
	}

	return http1.Response{
		StatusCode: 200,
		Body:       []byte("ok\n"),
	}
})
```

This mirrors the same idea as `http.HandlerFunc`: ordinary functions can become
handlers when they have the right shape.

## Request Context

The request context currently comes from the TCP connection context. When server
shutdown cancels the root context, each accepted connection receives a canceled
context too.

This does not forcibly stop application code. It is a notification channel.
Handlers must check `request.Context.Done()` or pass the context into operations
that understand cancellation.

## Current Limitations

- Handlers cannot stream a response yet.
- Request contexts are connection-scoped rather than independently scoped per
  request.
- Middleware is not implemented yet.
- Panics in application handlers are not recovered yet.

Those limitations are intentional next steps. They show why production Web
servers usually expose a request context and a response writer abstraction.
