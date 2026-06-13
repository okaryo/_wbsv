# WebSocket Upgrade

WebSocket starts as an HTTP/1.1 request and then upgrades the connection to a
different protocol.

The client sends an upgrade request:

```text
GET /ws HTTP/1.1
Host: server.example.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13
```

If the server accepts, it replies:

```text
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

After that point, the connection is no longer carrying ordinary HTTP
request/response messages. It carries WebSocket frames.

## Sec-WebSocket-Accept

The server proves that it understood the WebSocket handshake by computing:

```text
base64(sha1(Sec-WebSocket-Key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
```

The GUID is fixed by the WebSocket protocol.

For this key:

```text
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
```

the accept value is:

```text
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

## Upgrade Versus Normal HTTP

SSE keeps using an HTTP response body:

```text
HTTP response
  text/event-stream body
```

WebSocket changes the protocol after the handshake:

```text
HTTP request
HTTP 101 response
WebSocket frames
```

That is why a real WebSocket implementation needs direct control over the
underlying connection after the handshake.

## Current Implementation

Application handlers can accept a valid upgrade request:

```go
if err := httpserver.AcceptWebSocket(w, request); err != nil {
	w.WriteHeader(400)
	return
}
```

The helper validates:

- `GET` method.
- `HTTP/1.1`.
- `Connection: Upgrade`.
- `Upgrade: websocket`.
- `Sec-WebSocket-Version: 13`.
- A valid 16-byte base64 `Sec-WebSocket-Key`.

It then writes the `101 Switching Protocols` handshake response.

## Current Scope

The implementation is intentionally limited:

- It implements the HTTP upgrade handshake.
- It computes `Sec-WebSocket-Accept`.
- It does not parse or write WebSocket frames.
- It does not implement masking, opcodes, fragmentation, ping/pong, or close
  frames.
- It does not yet hijack the connection and hand it to a WebSocket frame loop.

In the current server architecture, after a `101` response the HTTP handler
returns instead of trying to parse WebSocket frames as HTTP requests. The TCP
server then closes the connection. This is not a complete WebSocket server; it
is a focused learning step for the upgrade handshake.

## Key Takeaways

- WebSocket starts with HTTP but does not remain ordinary HTTP.
- `101 Switching Protocols` marks the protocol transition.
- `Sec-WebSocket-Accept` is derived from the client key and a fixed GUID.
- After upgrade, the server must stop using the HTTP parser for that connection.
- A complete implementation needs connection hijacking and WebSocket frame
  handling.
