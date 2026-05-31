# Handler Model

The HTTP server now has a small application handler abstraction.

```go
type AppHandler interface {
	ServeHTTP(http1.Request) http1.Response
}
```

This separates two responsibilities:

- The server layer owns TCP connections, timeouts, HTTP parsing, response
  writing, and keep-alive decisions.
- The application layer receives a parsed request and returns a response.

The first version is intentionally simpler than `net/http`. It returns a whole
`http1.Response` value instead of writing through a streaming `ResponseWriter`.
That keeps the current learning focus on the boundary between protocol handling
and application behavior.

`AppHandlerFunc` adapts a function to the interface:

```go
handler := httpserver.AppHandlerFunc(func(request http1.Request) http1.Response {
	return http1.Response{
		StatusCode: 200,
		Body:       []byte("ok\n"),
	}
})
```

This mirrors the same idea as `http.HandlerFunc`: ordinary functions can become
handlers when they have the right shape.

## Current Limitations

- Handlers cannot stream a response yet.
- Handlers cannot observe connection cancellation through a request context yet.
- Middleware is not implemented yet.
- Panics in application handlers are not recovered yet.

Those limitations are intentional next steps. They show why production Web
servers usually expose a request context and a response writer abstraction.
