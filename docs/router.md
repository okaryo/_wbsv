# Router

The router now performs static path, method, and path-parameter matching.

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
- More advanced routers need more structure when path parameters, wildcards,
  and priorities are introduced.
