# HTTP Server Integration

The project now connects the HTTP parser and response writer to the TCP server.

The current runtime flow is:

```text
TCP listener
  -> Accept
  -> connection goroutine
  -> HTTP request parser
  -> application handler
  -> HTTP response writer
  -> next request or close connection
```

The server now supports basic HTTP/1.1 keep-alive. It can read multiple requests
from the same TCP connection until the connection should close.

## Current Default Response

If no application handler is configured, the server uses a small default handler
that returns:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Content-Length: 17

hello from _wbsv
```

It also includes debug headers showing the parsed method and request target.

## Current Error Behavior

If request parsing fails, the server returns a plain-text error response.

Examples:

- malformed request line -> `400 Bad Request`
- oversized body -> `413 Content Too Large`
- `Transfer-Encoding` -> `501 Not Implemented`

## Connection Behavior

For HTTP/1.1 requests, the server keeps the connection open by default.

The server closes the connection after the response when:

- The request contains `Connection: close`.
- The request uses HTTP/1.0.
- Request parsing fails.
- The read timeout expires while waiting for a request.
- Writing the response times out.

HTTP/1.0 keep-alive is not implemented yet.

For keep-alive connections, the read deadline is refreshed before each request.
This means an idle connection cannot wait forever between requests.

## Shutdown Behavior

The TCP server stops accepting new connections when the root context is
canceled. Existing HTTP connections are allowed to finish during the configured
graceful shutdown timeout.

If a connection is still active after that timeout, the server closes it. This
unblocks handlers that are waiting in `Read` or `Write`, then `Serve` waits for
the connection goroutines to exit.
