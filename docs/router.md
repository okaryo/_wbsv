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
- More advanced routers need more structure when path parameters, wildcards,
  and priorities are introduced.
