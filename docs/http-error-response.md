# HTTP Error Responses

An error response is still a normal HTTP response.

Example:

```text
HTTP/1.1 400 Bad Request\r\n
Content-Type: text/plain; charset=utf-8\r\n
Content-Length: 12\r\n
\r\n
bad request\n
```

The status code communicates the response class to the client. The body provides
a short human-readable message.

## Current Scope

`ErrorResponse` builds a small plain-text response:

- It sets the status code.
- It uses the status text when no custom message is provided.
- It appends a trailing newline to the body.
- It sets `Content-Type: text/plain; charset=utf-8`.

The normal `WriteResponse` path still computes `Content-Length` from the body.

## Limitations

This is intentionally minimal. It does not yet implement:

- Structured JSON error bodies.
- Error-to-status mapping.
- Internal error hiding.
- Connection close behavior.
- Method-specific response rules.
