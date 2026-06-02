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

## Recovery Middleware

The recovery middleware uses `defer` and `recover` to catch panics from the
application handler.

```text
request
  -> recovery middleware installs deferred recover
  -> application handler panics
  -> deferred recover catches the panic
  -> response buffer is reset
  -> 500 Internal Server Error response is written
```

Because the current response writer is buffered, recovery can replace a partial
response before it is serialized to the TCP connection. A streaming response
writer would have a harder constraint: once headers or body bytes are sent, a
clean 500 response may no longer be possible.

Middleware order matters. `LoggingMiddleware` should wrap
`RecoveryMiddleware` if logs should record the recovered `500` response.

## Request ID Middleware

The request ID middleware attaches an ID to both the request context and the
response headers.

```text
incoming X-Request-ID exists
  -> reuse it

incoming X-Request-ID missing
  -> generate a new ID
```

Application code can read the value from the context:

```go
requestID, ok := httpserver.RequestIDFromContext(request.Context)
```

The same value is also returned in the `X-Request-ID` response header. This is a
common pattern for correlating server logs, client errors, and downstream
operations.

## Auth Middleware

The bearer auth middleware checks the `Authorization` request header before the
application handler runs.

```text
Authorization: Bearer <token>
```

If the token matches, the middleware calls the next handler. If the token is
missing or wrong, the middleware writes a `401 Unauthorized` response and does
not call the next handler.

This demonstrates a different middleware shape from logging. Logging usually
wraps the whole request and continues after `next.ServeHTTP`. Auth often decides
whether `next.ServeHTTP` should run at all.

## Compression Middleware

The gzip compression middleware buffers the inner handler response, checks
whether the request accepts gzip, and then replaces the response body before it
is written to the connection.

```text
request has Accept-Encoding: gzip
  -> inner handler writes buffered response
  -> middleware compresses body with gzip
  -> middleware adds Content-Encoding: gzip
  -> compressed response is written outward
```

This is easy in the current server because responses are already buffered. A
streaming implementation would need to decide when headers are committed and
then wrap body writes with a gzip writer.

## Current Limitations

- Response writes are buffered in memory rather than streamed directly to the
  connection.
- Request contexts are connection-scoped rather than independently scoped per
  request.
- Bearer auth uses one static token and does not model users, sessions, scopes,
  or token expiry.
- Gzip compression is response-buffer based and does not stream compressed
  output.
- Recovery can reset buffered responses, but this behavior will need revisiting
  when response writing becomes streaming.

Those limitations are intentional next steps. They show why production Web
servers usually expose a request context and a response writer abstraction.
