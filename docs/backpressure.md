# Backpressure

Backpressure is how a server reacts when work arrives faster than it can be
processed.

Without backpressure, a server may keep accepting work until memory,
goroutines, file descriptors, or downstream services are exhausted. With
backpressure, the server makes overload visible and bounded.

## In This Project

The TCP server has an optional active connection limit:

```sh
go run ./cmd/wbsv --max-active-conns 100
```

When `MaxActiveConns > 0`, the server checks the number of tracked active
connections before it starts handling a newly accepted connection. If the limit
has already been reached, the server closes the new connection and increments
`rejected_connections`.

This is a low-level TCP form of admission control:

```text
Accept connection
  -> active connection count below limit?
      yes: track and handle it
      no:  record rejection and close it
```

## Why Close Instead Of Queue Forever?

Queueing can smooth short bursts, but an unbounded queue hides overload. If
clients arrive faster than the server can process them, a growing queue means:

- more memory is retained;
- clients wait longer before useful work starts;
- shutdown takes longer;
- failure moves later and becomes harder to explain.

Rejecting work early is often easier to reason about. It gives the server a
clear upper bound and makes overload observable.

## TCP Backpressure vs HTTP Backpressure

This project currently rejects excess work at the TCP connection level by
closing the connection.

A more application-aware HTTP server could instead accept the connection, parse
the request, and return a response such as:

```text
HTTP/1.1 503 Service Unavailable
Connection: close
Retry-After: 1
```

That gives clients a clearer protocol-level signal, but it requires spending
more server work on the rejected request.

## Interaction With Worker Pools

A worker pool caps how many connection handlers run concurrently.

An active connection limit caps how many accepted connections the server is
willing to track at once.

Together they answer different questions:

- Worker pool: how many handlers may run at the same time?
- Active connection limit: how many accepted connections may exist at once?

If the worker pool is small and the active connection limit is large, many
connections may wait for workers. If the active connection limit is small, the
server rejects excess connections earlier.

## Key Takeaways

- Backpressure is a policy for overload, not a performance trick.
- Queueing is not free; waiting work still consumes resources.
- Rejecting early can be better than accepting work the server cannot process.
- TCP-level rejection is cheap but less informative than an HTTP `503`.
