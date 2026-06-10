# Chunked Transfer

HTTP/1.1 chunked transfer encoding lets a server send a response body without
knowing the full body length before it starts writing.

A fixed-length response uses `Content-Length`:

```text
HTTP/1.1 200 OK
Content-Length: 11

hello world
```

A chunked response uses `Transfer-Encoding: chunked`:

```text
HTTP/1.1 200 OK
Transfer-Encoding: chunked

b
hello world
0

```

Each chunk starts with its size in hexadecimal, followed by CRLF, the bytes, and
another CRLF. A zero-sized chunk ends the body.

```text
<hex-size>\r\n
<bytes>\r\n
...
0\r\n
\r\n
```

## Why It Exists

Without chunked transfer encoding, a persistent HTTP/1.1 connection needs a
clear body boundary. `Content-Length` is one way to provide that boundary.

Chunked transfer is another boundary mechanism:

- The server does not need to buffer the entire response body first.
- The client can know where each chunk ends.
- The final `0` chunk tells the client where the whole response body ends.
- The connection can remain open for another HTTP request after the response.

This matters for streaming, generated responses, large files, and long-running
responses where the final size is not known upfront.

## Current Implementation

`http1.Response` can opt into chunked transfer encoding:

```go
response := http1.Response{
	StatusCode: 200,
	Body:       []byte("hello"),
	Chunked:    true,
}
```

Application handlers can request chunked encoding through the response writer:

```go
w.UseChunkedEncoding()
_, _ = w.Write([]byte("hello"))
```

The current implementation still buffers the application body before writing the
wire response. This means it teaches the HTTP/1.1 framing, but it is not yet a
true streaming response writer.

## Current Scope

The implementation is intentionally small:

- It writes `Transfer-Encoding: chunked`.
- It omits `Content-Length` for chunked responses.
- It writes one data chunk for the buffered body.
- It writes the final zero-sized chunk.
- It rejects chunked responses for `HTTP/1.0`.
- It does not yet expose a direct streaming writer that flushes each chunk as
  the application writes it.
- It does not support chunk trailers yet.

## Key Takeaways

- `Content-Length` and chunked transfer are both body boundary mechanisms.
- Chunk sizes are written in hexadecimal.
- The final `0` chunk terminates the response body.
- Chunked transfer is an HTTP/1.1 feature.
- True streaming requires changing the response writer so bytes can be sent to
  the connection before the whole handler finishes.
