# net/http Comparison

This note records a small comparison point with Go's `net/http` behavior.

## Status Codes Without Bodies

Some HTTP response status codes do not carry a response body:

- `1xx` informational responses.
- `204 No Content`.
- `304 Not Modified`.

For these responses, `_wbsv` writes the status line and headers, then terminates
the header section with an empty line. It does not write `Content-Length` or
body bytes for those status codes.

Example:

```text
HTTP/1.1 204 No Content\r\n
\r\n
```

This mirrors the important behavior expected from an HTTP server: even if server
code accidentally attaches a body to a `204` response, the response writer should
not put those bytes on the wire.

## Why This Belongs in the Writer

The caller chooses the status code and body, but the response writer is
responsible for turning that model into valid HTTP bytes.

That is why `WriteResponse` already computes `Content-Length` from the actual
body, and why it now suppresses body output for status codes that cannot have
bodies.
