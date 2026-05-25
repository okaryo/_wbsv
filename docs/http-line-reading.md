# HTTP Line Reading

HTTP/1.x uses CRLF-terminated lines.

The request line and each header line end with:

```text
\r\n
```

The end of the header section is an empty line:

```text
\r\n
```

Together, the request line and headers look like this on the TCP stream:

```text
GET /hello HTTP/1.1\r\n
Host: localhost:8080\r\n
\r\n
```

## Why a Line Reader Is Needed

TCP is a byte stream. It does not preserve HTTP line boundaries.

A single `Read` call might return:

```text
GET /hello
```

and the next `Read` might return:

```text
 HTTP/1.1\r\nHost: localhost\r\n\r\n
```

The HTTP parser therefore needs a layer that buffers bytes until it has enough
data to find a CRLF boundary.

## Current Scope

`LineReader` reads one CRLF-terminated line and returns it without the trailing
CRLF.

It currently rejects:

- LF-only line endings.
- EOF before a CRLF is found.
- Lines longer than the configured maximum.

This is still not a full HTTP parser. It is only the boundary-finding step that
prepares input for parsers such as `ParseRequestLine`.
