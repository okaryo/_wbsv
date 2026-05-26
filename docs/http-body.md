# HTTP Body Reading

After the empty line that terminates the headers, the message body begins.

For fixed-length bodies, `Content-Length` tells the parser exactly how many
bytes to read:

```text
Content-Length: 11\r\n
\r\n
hello world
```

The body in this example is exactly 11 bytes.

## Why the Same Reader Matters

The line reader may have already read bytes beyond the header terminator into
its internal buffer. Those bytes can include the beginning of the body.

For that reason, body reading must continue through the same buffered reader
instead of reading directly from the original TCP connection.

```text
TCP connection
  -> buffered LineReader
    -> request line
    -> headers
    -> body
```

Mixing direct `conn.Read` calls with reads through the buffered reader can lose
access to bytes that were already buffered.

## Current Scope

`ReadFixedBody` reads exactly the number of bytes determined by
`Content-Length`.

It currently:

- Returns `nil` for a zero-length body.
- Rejects bodies larger than the configured maximum.
- Reads incrementally in chunks.
- Reports an error if EOF arrives before the expected number of bytes.

It does not implement `Transfer-Encoding: chunked` yet.
