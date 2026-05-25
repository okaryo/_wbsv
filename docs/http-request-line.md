# HTTP Request Line

This note documents the first HTTP parsing step.

An HTTP/1.x request starts with a request line:

```text
GET /hello HTTP/1.1
```

The request line has three parts separated by a single space:

```text
method SP request-target SP HTTP-version
```

In the example above:

- `GET` is the method.
- `/hello` is the request target.
- `HTTP/1.1` is the HTTP version.

The actual bytes on the TCP connection include a trailing CRLF:

```text
GET /hello HTTP/1.1\r\n
```

The current `ParseRequestLine` function parses the line after the trailing CRLF
has already been removed. Splitting TCP bytes into lines is a separate problem
and will be handled when incremental parsing is introduced.

## Why This Is Separate From TCP

TCP only provides a byte stream. It does not know where the request line starts
or ends.

The HTTP parser must eventually find the first CRLF, extract the bytes before
it, and then parse those bytes as a request line.

This is why HTTP parsing has two layers:

```text
TCP bytes
  -> find protocol boundaries
  -> parse request line, headers, and body
```

The current step only implements the second part for the request line.

## Current Scope

The parser currently accepts:

- `HTTP/1.1`
- `HTTP/1.0`
- Any non-empty method token without control whitespace.
- Any non-empty request target without CRLF.

The parser currently rejects:

- Empty request lines.
- Lines without exactly three parts.
- Empty method, target, or version.
- Request lines containing CRLF inside the parsed fields.
- Unsupported HTTP versions.

This is intentionally small. More protocol details will be added as the parser
grows.
