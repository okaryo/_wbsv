# HTTP Headers

HTTP/1.x headers are line-based fields after the request line.

Each header field has this shape:

```text
field-name ":" optional-whitespace field-value optional-whitespace
```

Example:

```text
Host: localhost:8080\r\n
Accept: */*\r\n
\r\n
```

The empty line terminates the header section.

## Current Scope

The current parser separates two responsibilities:

- `LineReader` extracts CRLF-terminated lines from the TCP byte stream.
- `ParseHeaderField` parses one `Name: value` line.
- `ReadHeaderFields` reads header lines until the empty line.

The parser currently preserves header order and original field-name casing.
Header names are case-insensitive in HTTP, but normalization is left for a later
step.

The parser currently rejects:

- Header lines without a colon.
- Empty field names.
- Field names containing spaces, tabs, CR, or LF.
- Field values containing CR or LF.
- More than the configured maximum number of headers.

It trims optional spaces and tabs around the field value.

This step does not interpret specific headers such as `Content-Length`,
`Connection`, `Host`, or `Transfer-Encoding` yet.
