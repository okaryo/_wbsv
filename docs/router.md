# Router

The first router implementation performs static path matching.

```text
request target: /hello?name=wbsv
route path:     /hello
```

The router strips the query string before matching. It then looks up the path in
a map:

```go
routes map[string]AppHandler
```

This is the simplest useful router shape:

```text
request path
  -> map lookup
  -> handler
```

If no route matches, the router writes a `404 Not Found` response.

## Current Scope

The router does not inspect the HTTP method yet. This means `GET /hello` and
`POST /hello` currently match the same route.

That limitation is intentional. It keeps this step focused on static path
matching before introducing method matching and route priority.

## Key Takeaways

- A router is an application handler that dispatches to other handlers.
- Static path matching can be implemented with a map lookup.
- The request target may contain a query string, but route matching usually
  uses only the path.
- More advanced routers need more structure when path parameters, wildcards,
  and priorities are introduced.
