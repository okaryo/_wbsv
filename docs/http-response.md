# HTTP Response Writing

An HTTP/1.x response starts with a status line:

```text
HTTP/1.1 200 OK\r\n
```

Then response headers follow:

```text
Content-Type: text/plain\r\n
Content-Length: 5\r\n
\r\n
hello
```

The empty line separates headers from the body.

## Current Scope

`WriteResponse` writes a minimal fixed-length response:

```text
status line
headers
Content-Length
empty line
body
```

It always writes `Content-Length` based on the actual body length. If the caller
passes a `Content-Length` header, that value is ignored and replaced.

For `1xx`, `204`, and `304` responses, it does not write `Content-Length` or
body bytes.

The writer currently validates:

- HTTP version.
- Status code range.
- Reason phrase does not contain CR or LF.
- Header names and values do not contain invalid line-breaking characters.

## Limitations

This step does not yet implement:

- MIME type detection.
- Error response helpers.
- Chunked transfer encoding.
- Keep-alive or connection-close behavior.
