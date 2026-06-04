# Router

The router now performs static path, method, path-parameter, and wildcard
matching.

```text
request target: /hello?name=wbsv
route path:     /hello
```

The router strips the query string before matching. It then looks up the path and
method in nested maps:

```go
routes map[string]map[string]AppHandler
```

Static routes use nested map lookups:

```text
request path
  -> map lookup
  -> request method
    -> map lookup
    -> handler
```

If no path matches, the router writes a `404 Not Found` response. If the path
exists but the method does not match, the router writes a
`405 Method Not Allowed` response with an `Allow` header.

## Path Parameters

Path parameters use `:name` segments:

```text
route: /users/:id
path:  /users/42
param: id = 42
```

The router matches parameter routes segment by segment. When a parameter segment
matches, the value is stored in the application request:

```go
id := request.Param("id")
```

Static routes are checked before parameter routes. This means `/users/me` can
take priority over `/users/:id`.

## Wildcards

Wildcard routes use a final `*name` segment:

```text
route: /assets/*path
path:  /assets/css/app.css
param: path = css/app.css
```

The wildcard must be the last route segment because it consumes the rest of the
path. It may also match an empty remainder, so `/assets/*path` can match
`/assets`.

The current priority is:

```text
static route
  -> parameter route
  -> wildcard route
```

Parameter and wildcard routes are also ordered by specificity. The router checks
segments from left to right and treats literal segments as more specific than
parameters, and parameters as more specific than wildcards:

```text
literal segment
  -> parameter segment
  -> wildcard segment
```

For example, `/users/:id/books` takes priority over `/users/:id/:section` for
`/users/42/books`, even if the generic route was registered first. Similarly,
`/assets/images/*path` takes priority over `/assets/*path` for
`/assets/images/logo.png`.

If two routes have the same specificity, registration order remains the final
tie-breaker. This keeps conflict resolution deterministic.

## Segment Trie

Parameter and wildcard routes are stored in a segment trie:

```text
/users/:id/books

root
  users
    :id
      books
```

During matching, the router walks one path segment at a time. At each node it
tries children in priority order:

```text
literal child
  -> parameter child
  -> wildcard child
```

This makes route priority part of the traversal instead of sorting and scanning
all registered parameter routes. For example, when matching
`/users/42/books`, the router naturally tries the literal `books` branch before
a parameter branch such as `:section`.

This is still not a radix tree. A radix tree compresses common prefixes so one
edge can represent more than one unit of input. This project currently uses the
simpler segment trie because it makes the matching rules easier to inspect.

## Current Scope

`Handle(path, handler)` still registers an any-method route. This keeps path
matching available while method-specific routes are introduced with
`HandleMethod(method, path, handler)`.

Exact method routes take priority over any-method routes for the same path.
Parameter routes with the same shape are grouped by method.

## Key Takeaways

- A router is an application handler that dispatches to other handlers.
- Static path and method matching can be implemented with nested map lookups.
- The request target may contain a query string, but route matching usually
  uses only the path.
- A known path with an unsupported method should return `405`, not `404`.
- Path parameters require preserving matched values and passing them to the
  selected handler.
- Wildcards are useful for file-like paths because they can capture multiple
  remaining segments.
- Route priority decides which matching pattern should win when several
  patterns could match the same path.
- A segment trie can encode route priority in the traversal order instead of
  checking every registered pattern.
- A radix tree is a more compact tree that can reduce the number of nodes and
  comparisons for larger route tables.
