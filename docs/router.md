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

## Comparison With Common Go Routers

The current router is intentionally smaller than production routers. It is
useful because the important moving parts are visible: path splitting, route
priority, parameter capture, method dispatch, and `404` versus `405` behavior.

### `net/http.ServeMux`

Modern `net/http.ServeMux` supports method, host, path, single-segment
wildcards, and remainder wildcards in patterns such as:

```text
GET /users/{id}
/files/{path...}
example.com/
```

It uses a specificity rule: if multiple patterns match, the most specific
pattern wins. If two patterns overlap but neither is more specific, registration
panics because the conflict is ambiguous.

Compared with this project, `ServeMux` also handles concerns this router does
not yet handle:

- Host-aware patterns.
- `GET` also matching `HEAD`.
- Path sanitizing and redirect behavior.
- Segment-by-segment URL unescaping.
- Conflict detection at registration time.

### Echo

Echo's router is based on a radix tree and exposes application-facing route
syntax such as:

```text
/users/:id
/users/*
```

Its documented path matching order is:

```text
static
  -> param
  -> match-any
```

This is very close to the priority order used here. The main difference is that
Echo's router is optimized for framework use: it integrates method registration,
context objects, middleware, and route metadata around a more compact tree.

### `httprouter`

`httprouter` uses a compressed dynamic trie, also called a radix tree. It is
designed around explicit route matches: a request should match exactly one
route or no route. That design avoids many ambiguous priority cases.

Its named parameters match one path segment:

```text
/user/:user
```

and it has built-in support for method-aware routing, `405 Method Not Allowed`,
trailing slash correction, and custom not-found or method-not-allowed handlers.

Compared with this project, `httprouter` focuses much more on performance and
allocation behavior. This project currently favors inspectability over
optimization.

### chi

chi is built to stay compatible with `net/http` handlers while adding a router,
middleware stack, route groups, subrouters, and request context integration. Its
router is based on a Patricia radix trie.

The main design difference is scope. This project's router only dispatches to
an `AppHandler`. chi's router also helps organize larger API surfaces through
composition:

```text
middleware
  -> route group
  -> subrouter
  -> handler
```

## Current Limitations

The router in this project deliberately leaves several production concerns out
of scope for now:

- It does not detect all conflicting route patterns.
- It does not support host-aware routes.
- It does not normalize or clean paths.
- It does not percent-decode path segments before matching.
- It does not redirect for trailing slash differences.
- It does not compress the trie into a radix tree.
- It does not expose route introspection or generated route documentation.

These omissions are useful learning boundaries. The current implementation
shows the core matching mechanics before adding framework-level behavior.

## References

- `net/http.ServeMux`: https://pkg.go.dev/net/http#ServeMux
- Echo routing: https://echo.labstack.com/docs/routing
- `httprouter`: https://github.com/julienschmidt/httprouter
- chi: https://github.com/go-chi/chi

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
