# TCP Connection Lifecycle

This note documents the lifecycle of the TCP server layer. The default
connection handler is still a raw echo loop, but the command-line server now
plugs an HTTP/1.x handler into the same TCP lifecycle.

## Server Lifecycle

The server starts by creating a TCP listener.

```text
ListenAndServe
  -> net.ListenConfig.Listen
  -> Serve
  -> listener.Accept loop
```

`listener.Accept()` blocks until a client establishes a TCP connection. This is
not a busy loop. The goroutine running `Serve` sleeps inside `Accept` while there
is no new connection.

When the process receives a shutdown signal, the root context is canceled. The
server reacts by closing the listener.

```text
context canceled
  -> shutdown runs once
  -> listener.Close()
  -> wait for active connections
  -> close remaining active connections after the graceful timeout
  -> blocked Accept returns an error
  -> wait for connection goroutines
  -> Serve returns nil
```

Closing the listener stops accepting new connections. The server also tracks
accepted connections so shutdown can wait for them or force close them.

## Connection Lifecycle

Each accepted connection is handled in its own goroutine.

```text
listener.Accept()
  -> net.Conn
  -> track active connection
  -> create connection context
  -> go handleConn(ctx, conn)
```

Inside `handleConn`, the current lifecycle is:

```text
accepted
  -> set read deadline
  -> Read
  -> set write deadline
  -> Write
  -> repeat
  -> return
  -> deferred conn.Close()
  -> untrack active connection
```

The `Read` call blocks until one of these happens:

- The client sends bytes.
- The client closes the connection.
- The read deadline expires.
- Another network error occurs.

If bytes are read, the server writes the same bytes back to the client. The
`Write` call can also block if the client or network cannot receive bytes fast
enough. The write deadline prevents that connection goroutine from waiting
forever during `Write`.

## Close Conditions

The current connection handler closes a connection when `handleConn` returns.
The close itself is performed by:

```go
defer conn.Close()
```

`handleConn` returns when:

- `Read` returns `io.EOF` because the client closed the connection.
- `Read` returns a timeout error.
- `Read` returns another network error.
- `SetReadDeadline` fails.
- `SetWriteDeadline` fails.
- `Write` returns a timeout error.
- `Write` returns another network error.

## Shutdown Behavior

The server now separates three shutdown actions:

- Close the listener so no new connections can be accepted.
- Mark the server as shutting down and wait for active connection goroutines to
  finish naturally.
- Close remaining active connections after the graceful timeout so blocked
  `Read` or `Write` calls can return.

This means shutdown has a graceful phase and a forceful phase. During the
graceful phase, existing connections may finish their current work. During the
forceful phase, the server closes any connection that is still active.

Each accepted connection also receives a context derived from the server context.
Canceling the server context notifies the connection handler through
`ctx.Done()`, but it does not automatically stop the handler. The handler must
observe the context or return because the connection was closed.

The shutdown cleanup is guarded by `sync.Once` because shutdown can be triggered
from more than one path. For example, the context watcher may close the listener
to unblock `Accept`, and the deferred cleanup still runs when `Serve` returns.
Only one of those paths should perform the actual close operations.

## Key Takeaways

- A TCP server waits for connections with `Accept`.
- A TCP connection waits for bytes with `Read`.
- Both `Accept` and `Read` are blocking operations, not busy loops.
- A `net.Conn` is a bidirectional byte stream.
- TCP has no request, response, header, status code, or message boundary.
- Deadlines are used to prevent a connection goroutine from blocking forever.
- Closing a listener and closing a connection are different operations.
- Shutdown needs to consider both the listener and already accepted connections.
- Graceful shutdown is a bounded wait; forceful close is still needed for stuck
  or slow connections.
