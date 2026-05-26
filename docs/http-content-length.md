# HTTP Content-Length

`Content-Length` tells an HTTP/1.x recipient how many bytes are in the message
body.

Example:

```text
POST /messages HTTP/1.1\r\n
Host: localhost:8080\r\n
Content-Length: 11\r\n
\r\n
hello world
```

After the empty line that terminates the header section, the recipient reads
exactly 11 bytes as the body.

## Why It Matters

TCP is a byte stream. It does not mark where one HTTP message body ends.

For fixed-length request bodies, `Content-Length` provides that boundary:

```text
headers end
  -> read Content-Length bytes
  -> body complete
```

Without such a boundary, the parser cannot know whether it has received the
whole body or should keep waiting for more bytes.

## Current Scope

The current helper only interprets the `Content-Length` header value from parsed
header fields.

It currently:

- Treats the field name case-insensitively.
- Accepts a non-negative decimal integer.
- Reports that the header is absent when not present.
- Allows duplicate `Content-Length` fields only when they have the same value.
- Rejects empty, signed, non-decimal, and conflicting values.

It does not read the body yet. Reading the body is the next step.
