# HTTP Content-Type

`Content-Type` tells the client how to interpret the response body bytes.

Example:

```text
HTTP/1.1 200 OK\r\n
Content-Type: text/plain; charset=utf-8\r\n
Content-Length: 5\r\n
\r\n
hello
```

`Content-Length` describes how many bytes are in the body. `Content-Type`
describes what those bytes mean.

## Current Scope

The current response helpers provide:

- `ContentTypeByPath`, which maps common file extensions to MIME types.
- `WithContentType`, which sets or replaces the `Content-Type` header.
- `WithContentTypeForPath`, which selects `Content-Type` from a path.

Unknown extensions default to:

```text
application/octet-stream
```

The mapping is intentionally small and explicit so the behavior is easy to
inspect while learning.

## Examples

```text
.html -> text/html; charset=utf-8
.txt  -> text/plain; charset=utf-8
.json -> application/json
.png  -> image/png
```

This step does not implement content sniffing. The response writer still only
writes headers and body bytes provided by the server code.
