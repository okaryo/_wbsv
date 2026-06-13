# Server-Sent Events

Server-Sent Events let a server send a stream of text events to a client over
HTTP.

The response uses this content type:

```text
Content-Type: text/event-stream
```

Each event is made of field lines and ends with a blank line:

```text
id: 42
event: message
data: hello

```

Multiple `data` lines are joined by the browser with newline characters:

```text
data: first line
data: second line

```

## Browser Model

Browsers expose SSE through `EventSource`:

```js
const events = new EventSource("/events")

events.addEventListener("message", (event) => {
  console.log(event.data)
})
```

SSE is one-way:

```text
server -> client
```

The client still uses ordinary HTTP requests for messages going back to the
server.

## Current Implementation

Application handlers can write an event:

```go
_ = w.WriteEvent(ServerSentEvent{
	ID:    "42",
	Event: "message",
	Data:  "hello",
	Retry: 3000,
})
```

The writer sets:

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Transfer-Encoding: chunked
```

and appends the encoded event to the response body.

## Current Scope

The implementation is intentionally small:

- It encodes `id`, `event`, `retry`, and `data` fields.
- It splits multiline data into multiple `data:` lines.
- It sets the response up as an event stream.
- It uses chunked transfer encoding for the HTTP/1.1 response.
- It does not yet flush events while the handler is still running.
- It does not yet keep the connection open for long-lived event production.
- It does not yet implement heartbeats or reconnect state handling with
  `Last-Event-ID`.

This means the current implementation teaches the event-stream format, but it
is not yet a full live SSE server.

## Key Takeaways

- SSE is an HTTP response format for server-to-client events.
- It is simpler than WebSocket when communication is mostly one-way.
- The event body is line-oriented text.
- Blank lines terminate events.
- Long-lived SSE requires connection lifecycle, flushing, cancellation, and
  backpressure handling.
