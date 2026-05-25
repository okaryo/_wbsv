# TCP Connection Lifecycle

This note documents the lifecycle of the current raw TCP echo server.

The server does not speak HTTP yet. It only accepts TCP connections, reads bytes,
and writes the same bytes back to the client.

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
  -> listener.Close()
  -> blocked Accept returns an error
  -> Serve returns nil
```

Closing the listener stops accepting new connections. The server also tracks
accepted connections and closes them during shutdown.

## Connection Lifecycle

Each accepted connection is handled in its own goroutine.

```text
listener.Accept()
  -> net.Conn
  -> track active connection
  -> go handleConn(conn)
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

The server now separates two shutdown actions:

- Close the listener so no new connections can be accepted.
- Close active connections so blocked `Read` or `Write` calls can return.

After closing active connections, `Serve` waits for connection goroutines to
finish. This is the first step toward graceful shutdown. A more complete server
would usually distinguish between graceful draining and forceful connection
closure.

## Key Takeaways

- A TCP server waits for connections with `Accept`.
- A TCP connection waits for bytes with `Read`.
- Both `Accept` and `Read` are blocking operations, not busy loops.
- A `net.Conn` is a bidirectional byte stream.
- TCP has no request, response, header, status code, or message boundary.
- Deadlines are used to prevent a connection goroutine from blocking forever.
- Closing a listener and closing a connection are different operations.
- Shutdown needs to consider both the listener and already accepted connections.
