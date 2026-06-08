# Cache Headers

HTTP caching is controlled by response headers and conditional request headers.

The response tells the client or intermediary cache how the representation may
be stored:

```text
Cache-Control: public, max-age=60
ETag: "resource-v1"
Last-Modified: Mon, 08 Jun 2026 09:15:00 GMT
```

The next request can ask whether the cached representation is still valid:

```text
If-None-Match: "resource-v1"
If-Modified-Since: Mon, 08 Jun 2026 09:15:00 GMT
```

If the server decides the representation has not changed, it can return:

```text
HTTP/1.1 304 Not Modified
ETag: "resource-v1"
Last-Modified: Mon, 08 Jun 2026 09:15:00 GMT
```

A `304` response has no response body. The client reuses the cached body it
already has.

## Cache-Control

`Cache-Control` describes storage and freshness policy.

Common examples:

```text
Cache-Control: no-store
Cache-Control: no-cache
Cache-Control: private, max-age=60
Cache-Control: public, max-age=31536000
```

Important distinctions:

- `no-store` means the response should not be stored.
- `no-cache` does not mean "never cache"; it means the cached response must be
  revalidated before reuse.
- `max-age` describes how many seconds the response may be considered fresh.
- `private` means the response is meant for a single user agent.
- `public` means shared caches may store the response.

## ETag

`ETag` is a server-generated representation identifier.

```text
ETag: "resource-v1"
```

If the client later sends:

```text
If-None-Match: "resource-v1"
```

and the current representation still has that ETag, the server can return
`304 Not Modified`.

ETags are useful when modification time is not precise enough or when the
representation changes without a meaningful timestamp.

## Last-Modified

`Last-Modified` is a timestamp for the representation:

```text
Last-Modified: Mon, 08 Jun 2026 09:15:00 GMT
```

The client can revalidate with:

```text
If-Modified-Since: Mon, 08 Jun 2026 09:15:00 GMT
```

If the resource has not changed since that time, the server can return
`304 Not Modified`.

## Current Implementation

Application handlers can set cache headers through the response writer:

```go
_ = w.SetCacheControl("public, max-age=60")
_ = w.SetETag("resource-v1")
w.SetLastModified(lastModified)
```

Handlers can check whether a request already has the current representation:

```go
if request.NotModified("resource-v1", lastModified) {
	_ = w.WriteNotModified("resource-v1", lastModified)
	return
}
```

`If-None-Match` takes priority over `If-Modified-Since`. This matters because an
ETag is usually a stronger representation check than a timestamp.

## Current Scope

The implementation is intentionally small:

- It supports basic `Cache-Control` header writing.
- It normalizes response ETags by adding quotes when needed.
- It supports weak ETag comparison for `If-None-Match`.
- It supports HTTP-date parsing for `If-Modified-Since`.
- It does not implement shared-cache behavior.
- It does not implement cache key variation beyond the existing `Vary` header
  used by gzip compression.
- It does not generate ETags automatically from response bodies.

## Key Takeaways

- `Cache-Control` controls freshness and storage policy.
- `ETag` and `Last-Modified` are validators.
- Conditional requests let the server avoid sending a body when the cached copy
  is still valid.
- `304 Not Modified` reuses the client's cached body and does not include a new
  body.
- Cache behavior is a protocol contract between the server, user agent, and
  possible intermediary caches.
