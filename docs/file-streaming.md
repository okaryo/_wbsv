# File Streaming

File streaming sends a response body from an `io.Reader` instead of first
building the whole body in memory.

For a small response, buffering is simple:

```go
_, _ = w.Write([]byte("hello"))
```

For a large file, buffering the whole file would waste memory. A server can
instead open the file and copy bytes from the file descriptor to the connection:

```text
file -> buffer -> TCP connection
```

The response still needs a body boundary. For known-size files, the server can
send `Content-Length`:

```text
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Content-Length: 1024

...1024 bytes...
```

For unknown-size streams, the server can use chunked transfer encoding:

```text
HTTP/1.1 200 OK
Transfer-Encoding: chunked

...
```

## Current Implementation

`http1.Response` supports streaming body data from an `io.Reader`:

```go
response := http1.Response{
	StatusCode:  200,
	BodyReader: reader,
	BodyLength: size,
}
```

The low-level writer sends headers first, then copies from `BodyReader` to the
network writer. If `BodyLength` is known, it writes `Content-Length`. If the
length is unknown, the response must opt into chunked transfer encoding.

Application handlers can stream a body explicitly:

```go
w.StreamBody(reader, size)
```

or send a file:

```go
if err := w.SendFile("public/hello.txt"); err != nil {
	w.WriteHeader(404)
	return
}
```

`SendFile` opens the file, sets a content type from the extension when known,
sets `Last-Modified` from the file metadata, and streams the file body.

## Current Scope

The implementation is intentionally small:

- It avoids loading a whole file into memory.
- It streams fixed-length bodies with `Content-Length`.
- It can stream unknown-length readers when chunked transfer is enabled.
- It closes files opened by `SendFile` after the response is written.
- It does not yet support range requests.
- It does not yet support sendfile system calls.
- It does not flush bytes while the application handler is still running.

The last point is important. This implementation streams from the response
writer to the connection after the handler returns. It avoids large memory
buffers, but it is not yet the same as server-sent events or a handler-driven
stream that flushes chunks over time.

## Key Takeaways

- Streaming separates the response body source from the in-memory `[]byte`
  representation.
- Known-size streams can still use `Content-Length`.
- Unknown-size streams need another boundary mechanism such as chunked transfer.
- File descriptors need lifecycle management; opened files must be closed.
- File streaming is a step toward long-lived streaming responses, but it is not
  the whole design.
