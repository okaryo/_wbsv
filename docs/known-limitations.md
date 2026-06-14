# Known Limitations

`_wbsv` is a learning project, not a production Web server. The implementation
intentionally keeps many behaviors small and visible so each mechanism can be
studied.

This document records the main limitations so future steps can distinguish
between deliberate learning scope and accidental missing behavior.

## Protocol Scope

- The server focuses on HTTP/1.x behavior.
- HTTP/2 and HTTP/3 are not implemented.
- TLS termination is not implemented.
- HTTP/1.0 keep-alive is not implemented.
- Request and response parsing cover selected behaviors, not the full RFC
  surface.
- Transfer-Encoding request bodies are rejected rather than decoded.
- Chunked request bodies are not implemented.
- Request trailers and response trailers are not implemented.

## Request Parsing

- The parser is intentionally explicit and small.
- Header normalization is minimal.
- Header size and count limits exist, but the server does not implement every
  production hardening rule.
- Multiple `Content-Length` values are checked for consistency, but many other
  header conflict cases are out of scope.
- The parser does not implement every method-specific body rule.

## Response Writing

- Most application responses are buffered before being written to the
  connection.
- Chunked responses teach HTTP/1.1 framing but still use a buffered application
  body.
- True incremental chunk flushing is not implemented.
- Error responses are intentionally simple.
- HTTP compression is implemented as buffered gzip rewriting, not as a streaming
  compressor.

## Connection Lifecycle

- Basic keep-alive is implemented for HTTP/1.1.
- Read and write deadlines are coarse connection-level safeguards.
- Graceful shutdown has a bounded wait, then force-closes active connections.
- The server does not implement advanced connection draining policies.
- The active connection limit rejects excess accepted TCP connections by
  closing them, not by returning an HTTP `503`.
- OS-level listen backlog behavior is not directly controlled by this project.

## Concurrency And Backpressure

- The default model is goroutine-per-connection.
- The optional worker pool caps connection handler concurrency, but accepted
  connections can still wait for workers.
- The active connection limit is a simple admission-control mechanism.
- There is no priority queue, adaptive load shedding, or per-client fairness.
- Metrics are log snapshots and test helpers, not a production observability
  system.

## Routing

- The router supports static routes, method matching, path parameters,
  final-segment wildcards, priority rules, and a segment trie.
- Wildcards are intentionally limited to the final path segment.
- Route registration conflict handling is intentionally simpler than production
  routers.
- URL normalization, escaped path handling, and redirect behavior are limited.

## Middleware And Application Model

- The handler and middleware model is intentionally small.
- Middleware runs around buffered application responses.
- Authentication is a bearer-token learning hook, not a complete auth system.
- Rate limiting is a global fixed-window limiter.
- There is no per-route middleware stack, dependency injection system, or
  application lifecycle framework.

## Practical HTTP Features

- Cookies, cache headers, CORS, range requests, SSE framing, and the WebSocket
  upgrade handshake are implemented as focused learning steps.
- The SSE implementation teaches event framing but does not yet flush long-lived
  events while a handler is running.
- The WebSocket implementation handles the upgrade handshake only. It does not
  hijack the connection or parse WebSocket frames.
- Range requests support a single byte range.
- File streaming uses `io.Reader` copying but does not implement sendfile or
  platform-specific zero-copy behavior.

## Load Testing

- `wbsvload` is a small repeatable load generator.
- It is useful for observing concurrency, connection churn, status counts,
  failures, bytes read, and simple latency values.
- It is not a statistically rigorous benchmark tool.
- It does not report percentiles yet.
- It does not generate slow clients or complex backpressure scenarios yet.

## Production Gaps

The project does not currently provide:

- TLS certificates or ALPN.
- HTTP/2 multiplexing.
- HTTP/3 over QUIC.
- Robust access logging formats.
- Structured metrics and tracing.
- Config reloads.
- Static file security hardening.
- Reverse proxy behavior.
- Full RFC compliance.
- Security review suitable for Internet exposure.

These gaps are acceptable for the current goal. They are useful future learning
paths, not failures of the current implementation.
