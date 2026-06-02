# TODO

This file is a living learning roadmap for the Web server implementation.

The roadmap is intentionally flexible. Update it whenever the learning goal,
implementation direction, or level of detail changes.

## Current Learning Goal

Build a small Web server in Go from lower-level primitives and use it to
understand the mechanics usually hidden by Web frameworks.

Initial focus:

- TCP connection handling.
- HTTP/1.1 request and response structure.
- Incremental parsing.
- Concurrency and connection lifecycle.
- Handler, middleware, and routing internals.
- Streaming and practical HTTP behavior.

## Roadmap

### 0. Project Setup

- [x] Define the project purpose.
- [x] Create initial project documentation.
- [x] Decide the first implementation milestone.
- [ ] Decide how to organize learning notes.
- [x] Decide the initial package layout after the first milestone is clear.

First implementation milestone:

- Build a raw TCP echo server using `net.Listen`.
- Keep HTTP out of scope until the basic TCP connection lifecycle is visible.

### 1. Raw TCP Server

- [x] Create a minimal TCP server with `net.Listen`.
- [x] Accept connections in a loop.
- [x] Read bytes from a connection.
- [x] Write bytes back to a connection.
- [x] Run each connection in a separate goroutine.
- [ ] Observe blocking behavior manually.
- [x] Add read deadlines.
- [x] Add write deadlines.
- [x] Document the connection lifecycle.
- [x] Track active connections.
- [x] Close active connections on shutdown.
- [x] Make shutdown cleanup idempotent with `sync.Once`.

Questions to answer:

- What blocks during `Accept`, `Read`, and `Write`?
- What happens when a client connects but sends no data?
- What happens when the server does not close a connection?
- Where can connection leaks happen?

### 2. Minimal HTTP Request Parsing

- [x] Parse the request line.
- [x] Parse headers.
- [x] Handle `Content-Length`.
- [x] Read request bodies incrementally.
- [x] Return errors for malformed requests.
- [x] Separate line boundary reading from request line parsing.
- [x] Separate tokenizer, parser state, and parsed request model.
- [x] Add tests for partial reads and malformed input.

Questions to answer:

- Why is HTTP parsing naturally incremental?
- How should the parser handle incomplete data?
- Where does a tokenizer help?
- What should be treated as a protocol error?

### 3. HTTP Response Writing

- [x] Write a valid HTTP status line.
- [x] Write response headers.
- [x] Write fixed-length response bodies.
- [x] Set `Content-Length` correctly.
- [x] Set common MIME types.
- [x] Implement basic error responses.
- [x] Compare behavior with `net/http`.

Questions to answer:

- When is `Content-Length` required?
- What happens if the declared length and actual body length differ?
- How should status codes affect response bodies?

### 4. Connection Management

- [x] Handle one HTTP request per TCP connection.
- [x] Implement basic HTTP/1.1 keep-alive behavior.
- [x] Support `Connection: close`.
- [x] Add read timeout behavior.
- [x] Add write timeout behavior.
- [x] Explore slow-client behavior.
- [x] Add graceful shutdown.
- [x] Confirm goroutines exit as expected.

Questions to answer:

- When should a connection be reused?
- When should the server close the connection?
- How do deadlines interact with keep-alive?
- How can slow clients consume server resources?

### 5. Handler and Middleware Model

- [x] Define a minimal handler interface.
- [x] Add a request context model.
- [x] Add a response writer abstraction.
- [x] Implement middleware chaining.
- [x] Add logging middleware.
- [x] Add recovery middleware.
- [x] Add request ID middleware.
- [x] Add bearer auth middleware.
- [x] Add gzip compression middleware.
- [x] Add fixed-window rate-limit middleware.

Questions to answer:

- What does a handler abstraction hide?
- What makes middleware order important?
- Where should cancellation and deadlines be exposed?

### 6. Router Internals

- [ ] Implement static route matching.
- [ ] Add method matching.
- [ ] Add path parameters.
- [ ] Add wildcard routes.
- [ ] Implement route priority rules.
- [ ] Explore trie or radix tree routing.
- [ ] Compare with routing behavior in common Go frameworks.

Questions to answer:

- Why do routers often use tree structures?
- How should route conflicts be resolved?
- What is the tradeoff between simple matching and optimized matching?

### 7. Practical HTTP Features

- [ ] Implement cookies.
- [ ] Implement cache-related headers.
- [ ] Implement CORS handling.
- [ ] Implement chunked transfer responses.
- [ ] Implement file streaming.
- [ ] Implement range requests.
- [ ] Explore server-sent events.
- [ ] Explore WebSocket upgrade basics.

Questions to answer:

- How does streaming change response writing?
- What does chunked transfer solve?
- How do range requests support large files and video streaming?
- What changes when a connection is upgraded?

### 8. Robustness and Load Behavior

- [ ] Add simple load testing scripts or commands.
- [ ] Observe goroutine counts under concurrent clients.
- [ ] Detect connection leaks.
- [ ] Explore worker pools.
- [ ] Explore backpressure.
- [ ] Run race detection where applicable.
- [ ] Document known limitations.

Questions to answer:

- What resource grows with each connection?
- Where can races occur?
- When is goroutine-per-connection enough?
- When might a worker pool help?

## Learning Log

Use this section to record notable decisions, discoveries, and direction changes.

- Initial direction: focus on Web server internals rather than ordinary Web API
  application architecture.
- First implementation milestone: start with a raw TCP echo server before
  implementing HTTP parsing.
- Added logs around `Accept`, `Read`, and `Write` so blocking behavior can be
  observed manually with `nc`.
- Added a read timeout so idle connected clients do not keep a connection
  goroutine blocked forever.
- Added a write timeout so a connection can be closed if writing bytes back to
  the client blocks for too long.
- Documented the current TCP server and connection lifecycle in
  `docs/tcp-connection-lifecycle.md`.
- Added active connection tracking so shutdown closes accepted connections, not
  only the listener.
- Guarded shutdown cleanup with `sync.Once` so listener and active connection
  close operations only run once per `Serve` call.
- Added the first HTTP parsing unit: request line parsing for method,
  request-target, and HTTP version.
- Added CRLF line reading so request lines and headers can be extracted from a
  TCP byte stream before being parsed.
- Updated line reading to enforce the maximum line length while reading instead
  of building an oversized string first.
- Added minimal HTTP header parsing for `Name: value` lines and the empty line
  that terminates the header section.
- Added `Content-Length` interpretation for fixed-length request bodies.
- Added fixed-length body reading through the same buffered reader used for
  request lines and headers.
- Added `ReadRequest` to compose request line, headers, `Content-Length`, body
  reading, and request-level validation.
- Added fixed-length HTTP response writing with status line, headers,
  `Content-Length`, and body.
- Added common MIME type helpers for response `Content-Type`.
- Added basic plain-text HTTP error responses.
- Compared response body rules with `net/http` and suppressed body output for
  `1xx`, `204`, and `304` responses.
- Connected the HTTP parser and response writer to the TCP server. The runtime
  now handles one HTTP request per connection.
- Added basic HTTP/1.1 keep-alive and `Connection: close` handling.
- Added tests for keep-alive idle read timeout and blocked response write
  timeout.
- Added graceful shutdown with a bounded wait before force-closing active
  connections.
- Added a minimal application handler interface so protocol handling and
  application behavior can evolve separately.
- Added a connection-derived request context so application handlers can observe
  shutdown and cancellation.
- Added a buffered response writer abstraction so application handlers can build
  responses without returning `http1.Response` values directly.
- Added middleware chaining with `func(AppHandler) AppHandler` so cross-cutting
  behavior can wrap application handlers.
- Added logging middleware that records method, target, status, response bytes,
  and duration after application handling.
- Added recovery middleware that converts application panics into buffered
  `500 Internal Server Error` responses.
- Added request ID middleware that propagates `X-Request-ID` through response
  headers and request context.
- Added bearer auth middleware that can stop unauthorized requests before the
  application handler runs.
- Added gzip compression middleware that rewrites buffered responses when the
  request accepts gzip.
- Added fixed-window rate-limit middleware that returns `429 Too Many Requests`
  without calling the application handler when the global limit is exceeded.
