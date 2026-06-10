# CORS

Cross-Origin Resource Sharing is a browser-enforced HTTP mechanism. It controls
whether JavaScript running on one origin may read responses from another origin.

An origin is the tuple:

```text
scheme + host + port
```

For example, these are different origins:

```text
https://app.example.com
https://api.example.com
http://app.example.com
https://app.example.com:8443
```

CORS is not primarily a server-to-server security mechanism. Non-browser clients
can usually send the same HTTP requests without CORS enforcement. The server
still needs authentication and authorization.

## Actual Requests

For a cross-origin browser request, the browser sends an `Origin` header:

```text
Origin: https://app.example.com
```

If the server allows that origin, it includes:

```text
Access-Control-Allow-Origin: https://app.example.com
```

If credentials such as cookies or authorization state are allowed, the server
also includes:

```text
Access-Control-Allow-Credentials: true
```

When credentials are involved, `Access-Control-Allow-Origin: *` is not usable by
the browser. A server that wants to allow credentials must return a concrete
origin.

## Preflight Requests

Some cross-origin requests require a preflight request before the browser sends
the actual request. A preflight is an `OPTIONS` request:

```text
OPTIONS /items HTTP/1.1
Origin: https://app.example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Authorization, X-Client
```

The server responds with the methods and headers it allows:

```text
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST
Access-Control-Allow-Headers: Authorization, X-Client
Access-Control-Max-Age: 600
```

The application handler does not need to process the preflight as a normal
business request.

## Current Implementation

`CORSMiddleware` adds CORS headers for allowed origins and handles preflight
requests:

```go
httpserver.CORSMiddleware(httpserver.CORSOptions{
	AllowOrigins:     []string{"https://app.example.com"},
	AllowMethods:     []string{"GET", "POST", "OPTIONS"},
	AllowHeaders:     []string{"Authorization", "X-Client"},
	AllowCredentials: true,
	MaxAge:           600,
})
```

The command line can enable CORS:

```text
--cors-allow-origin=https://app.example.com
--cors-allow-methods="GET, POST, OPTIONS"
--cors-allow-headers="Authorization, X-Client"
--cors-allow-credentials
--cors-max-age=600
```

If no allowed origin is configured, CORS is disabled.

## Current Scope

The implementation is intentionally small:

- It supports explicit allowed origins and `*`.
- It echoes the concrete origin when `*` is configured with credentials.
- It handles preflight requests before application handlers run.
- It can echo requested preflight headers when no configured header list is
  provided.
- It does not support origin patterns or regular expressions.
- It does not validate every possible CORS configuration mistake.
- It does not replace authentication, authorization, CSRF protection, or
  SameSite cookie design.

## Key Takeaways

- CORS is enforced by browsers, not by raw HTTP clients.
- `Origin` tells the server which browser origin initiated the request.
- Preflight lets the browser ask whether a method and header set is allowed.
- Credentialed CORS cannot use wildcard `Access-Control-Allow-Origin: *`.
- CORS decides whether browser JavaScript may read the response; it is not a
  substitute for server-side access control.
