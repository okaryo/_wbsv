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

## Project Documents

- `README.md`: project purpose, scope, and high-level learning direction.
- `AGENTS.md`: working instructions for AI agents and future contributors.
- `TODO.md`: living learning roadmap and progress tracker.
