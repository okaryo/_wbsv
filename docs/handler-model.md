# Handler Model

The HTTP server now has a small application handler abstraction.

```go
type Request struct {
	Context context.Context
	HTTP    http1.Request
}

type AppHandler interface {
	ServeHTTP(ResponseWriter, Request)
}

type ResponseWriter interface {
	AddHeader(name, value string)
	SetHeader(name, value string)
	WriteHeader(statusCode int)
	Write([]byte) (int, error)
}

type Middleware func(AppHandler) AppHandler
```

This separates two responsibilities:

- The server layer owns TCP connections, timeouts, HTTP parsing, response
  writing, and keep-alive decisions.
- The application layer receives a response writer and an application request,
  then writes the response it wants to send.

This is closer to `net/http` than returning a whole response value. The handler
does not need to know how the response is serialized onto the TCP connection. It
only records status, headers, and body through the writer.

`AppHandlerFunc` adapts a function to the interface:

```go
handler := httpserver.AppHandlerFunc(func(w httpserver.ResponseWriter, request httpserver.Request) {
	select {
	case <-request.Context.Done():
		w.WriteHeader(408)
		_, _ = w.Write([]byte("request canceled\n"))
		return
	default:
	}

	w.SetHeader("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
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

## Response Writer

The current response writer is buffered. It records headers, status, and body in
memory. After the handler returns, the server converts that buffered state into
an `http1.Response` and writes it to the connection.

`Write` implicitly selects status `200` when `WriteHeader` has not been called.
Only the first `WriteHeader` call changes the status code.

This model explains the shape of `net/http.ResponseWriter`, but it is not fully
streaming yet.

## Middleware Chain

Middleware wraps an `AppHandler` and returns another `AppHandler`.

```go
middleware := func(next httpserver.AppHandler) httpserver.AppHandler {
	return httpserver.AppHandlerFunc(func(w httpserver.ResponseWriter, request httpserver.Request) {
		w.AddHeader("X-Before", "middleware")
		next.ServeHTTP(w, request)
		w.AddHeader("X-After", "middleware")
	})
}
```

`Chain(app, first, second)` executes request-side behavior in registration
order:

```text
first before
  -> second before
    -> app
  -> second after
first after
```

This shape is useful for logging, authentication, recovery, request IDs,
compression, and rate limiting because those features can be added around the
application handler without changing the handler itself.

## Logging Middleware

The logging middleware wraps the response writer so it can observe the status
code and number of body bytes written by the application handler.

```text
request
  -> logging middleware starts timer
  -> application handler writes response
  -> logging middleware records method, target, status, bytes, duration
```

This is a common middleware pattern. The middleware does not need to know how the
server writes bytes to the TCP connection. It only watches the handler-facing
`ResponseWriter`.

## Current Limitations

- Response writes are buffered in memory rather than streamed directly to the
  connection.
- Request contexts are connection-scoped rather than independently scoped per
  request.
- Panics in application handlers are not recovered yet.

Those limitations are intentional next steps. They show why production Web
servers usually expose a request context and a response writer abstraction.
