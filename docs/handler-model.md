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

## Current Limitations

- Response writes are buffered in memory rather than streamed directly to the
  connection.
- Request contexts are connection-scoped rather than independently scoped per
  request.
- Middleware is not implemented yet.
- Panics in application handlers are not recovered yet.

Those limitations are intentional next steps. They show why production Web
servers usually expose a request context and a response writer abstraction.
