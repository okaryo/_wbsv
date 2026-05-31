# _wbsv

`_wbsv` is a learning-oriented Web server implementation in Go.

The goal of this project is not to build a production-ready framework or a
typical Web API application. The goal is to understand what sits underneath the
Web frameworks we usually use: TCP connections, HTTP parsing, request and
response semantics, connection lifecycles, concurrency, routing, middleware, and
streaming.

## Purpose

This project is for studying Web server internals step by step.

The intended learning style is:

- Build small pieces incrementally.
- Confirm the learning objective before each major step.
- Prefer understanding the mechanism over quickly adding features.
- Compare behavior with Go's standard library and common frameworks when useful.
- Keep the roadmap flexible as new questions and interests appear.

The project assumes that the learner is already comfortable with Go and has
backend development experience. Therefore, the focus is not on basic Go syntax
or ordinary CRUD API design, but on deeper implementation details.

## Learning Topics

This project may cover topics such as:

- TCP server basics: `listen`, `accept`, `read`, `write`, blocking I/O, timeouts,
  and connection lifecycle.
- HTTP request and response structure: request line, status line, headers,
  status codes, body handling, `Content-Length`, chunked transfer, MIME types,
  cookies, cache control, CORS, and range requests.
- Incremental parsing: tokenizer design, state machines, streaming reads, and
  malformed input handling.
- Concurrency: goroutines per connection, worker pools, channels, cancellation,
  context propagation, race conditions, and resource cleanup.
- Robustness: slow clients, timeouts, connection leaks, graceful shutdown, and
  backpressure.
- Framework-like layers: handlers, middleware, logging, recovery, request IDs,
  authentication hooks, compression, and rate limiting.
- Routing: static routes, path parameters, wildcards, trie or radix tree
  structures, route priority, and method matching.
- Streaming and practical Web features: server-sent events, WebSocket basics,
  chunked responses, file streaming, video streaming, and range requests.

## Non-goals

The following are not the main focus of this project:

- Building a full production-ready Web framework.
- Building a business-domain Web API application.
- Learning ordinary Web API architecture such as handler, service, repository,
  and database layers.
- Prioritizing framework convenience over implementation understanding.

Some production-oriented topics may still be explored when they help explain how
real Web servers behave.

## Approach

The preferred starting point is the lower layer:

1. Start with `net.Listen` and raw TCP connections.
2. Implement a minimal HTTP/1.1 request parser.
3. Write valid HTTP responses manually.
4. Add connection handling such as keep-alive and timeouts.
5. Introduce handler and middleware abstractions.
6. Build routing mechanisms.
7. Explore streaming, range requests, and other practical HTTP features.

At each stage, the implementation should remain small enough to inspect and
explain. When the design becomes unclear, the roadmap should be updated rather
than treated as fixed.

## Running the Current Server

The current implementation is a minimal HTTP/1.x server.

Start the server:

```sh
go run ./cmd/wbsv
```

The server closes a connection if the client sends no bytes before the read
timeout. The default timeout is 30 seconds. To make it easier to observe:

```sh
go run ./cmd/wbsv --read-timeout 5s
```

The server also sets a write timeout before writing the HTTP response:

```sh
go run ./cmd/wbsv --write-timeout 5s
```

Send a request from another terminal:

```sh
curl -i http://127.0.0.1:8080/hello
```

The server parses HTTP/1.1 requests, writes fixed responses, and supports basic
keep-alive. Send `Connection: close` when you want the response to close the
connection.

## Observing Blocking Behavior

The server logs before calling `Accept` and `Read`.

When the server prints this log:

```text
waiting for a connection
```

it is blocked in `listener.Accept()` until a client connects.

After a client connects, the server prints:

```text
waiting for HTTP request from 127.0.0.1:xxxxx
```

At that point, the connection goroutine is blocked while the HTTP parser reads
request bytes from the connection.

Try this sequence:

1. Start the server with `go run ./cmd/wbsv`.
2. Connect with `nc 127.0.0.1 8080`, but do not type anything yet.
3. Observe that the server accepted the connection and is now waiting for an HTTP request.
4. Type a minimal HTTP request in `nc`.
5. Observe the response and request handling logs.
6. Stop `nc` and observe the connection close log.

Example request for `nc`:

```sh
printf 'GET /hello HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n' | nc 127.0.0.1 8080
```

To observe a read timeout, connect with `nc` and do not type anything until the
timeout expires. The server should return `408 Request Timeout` and close that
connection.

## Project Documents

- `README.md`: project purpose, scope, and high-level learning direction.
- `AGENTS.md`: working instructions for AI agents and future contributors.
- `TODO.md`: living learning roadmap and progress tracker.
- `docs/tcp-connection-lifecycle.md`: notes on the current TCP server lifecycle.
- `docs/http-request-line.md`: notes on the first HTTP request parsing step.
- `docs/http-line-reading.md`: notes on reading CRLF-terminated HTTP lines.
- `docs/http-headers.md`: notes on parsing HTTP header fields.
- `docs/http-content-length.md`: notes on interpreting fixed body length.
- `docs/http-body.md`: notes on reading fixed-length HTTP bodies.
- `docs/http-request.md`: notes on composing the minimal HTTP request parser.
- `docs/http-response.md`: notes on writing fixed-length HTTP responses.
- `docs/http-content-type.md`: notes on response MIME type helpers.
- `docs/http-error-response.md`: notes on basic HTTP error responses.
- `docs/http-net-http-comparison.md`: notes comparing selected behavior with Go's `net/http`.
- `docs/http-server-integration.md`: notes on connecting the HTTP parser and writer to TCP.
