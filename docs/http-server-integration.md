# HTTP Server Integration

The project now connects the HTTP parser and response writer to the TCP server.

The current runtime flow is:

```text
TCP listener
  -> Accept
  -> connection goroutine
  -> HTTP request parser
  -> fixed response
  -> HTTP response writer
  -> close connection
```

This is still intentionally small. The server handles one HTTP request per TCP
connection and then closes the connection.

## Current Response

For a valid request, the server returns:

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

## Limitations

The server does not implement keep-alive yet.

Each connection is closed after one request and one response. This keeps the
first integration step focused on connecting the byte stream, request parser,
and response writer.
