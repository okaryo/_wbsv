# Cookies

HTTP cookies are application data carried through HTTP headers.

The client sends cookies with the `Cookie` request header:

```text
Cookie: session=abc123; theme=dark
```

The server sets or replaces cookies with one `Set-Cookie` response header per
cookie:

```text
Set-Cookie: session=abc123; Path=/; Max-Age=3600; HttpOnly
Set-Cookie: theme=dark; Path=/
```

This matters because `Cookie` and `Set-Cookie` are not symmetric:

- A request usually carries many cookies in one `Cookie` header.
- A response sends each cookie as a separate `Set-Cookie` header.
- `Set-Cookie` can include attributes such as `Path`, `Domain`, `Max-Age`,
  `Expires`, `Secure`, `HttpOnly`, and `SameSite`.

## Current Implementation

Application handlers can read request cookies from `Request`:

```go
cookie, ok := request.Cookie("session")
```

They can also inspect all parsed request cookies:

```go
for _, cookie := range request.Cookies() {
	// cookie.Name and cookie.Value
}
```

Response cookies are written through the response writer:

```go
_ = w.SetCookie(Cookie{
	Name:     "session",
	Value:    "abc123",
	Path:     "/",
	MaxAge:   3600,
	HTTPOnly: true,
	Secure:   true,
	SameSite: "Lax",
})
```

`SetCookie` appends a `Set-Cookie` header. It intentionally does not use
`SetHeader`, because multiple cookies must be allowed in the same response.

## Current Scope

The parser is intentionally small:

- It parses simple `name=value` cookie pairs.
- It skips malformed request cookie pairs instead of failing the whole request.
- It does not decode percent-encoded or quoted cookie values.
- It does not implement signed, encrypted, or session-store cookies.
- It validates enough to avoid newline and semicolon injection in generated
  `Set-Cookie` headers.

This keeps the focus on the HTTP header mechanics before adding application
security behavior.

## Key Takeaways

- Cookies are header-based state carried over stateless HTTP requests.
- `Cookie` is a request header; `Set-Cookie` is a response header.
- `Set-Cookie` must support multiple header fields with the same name.
- Cookie attributes control browser behavior, not server-side storage by
  themselves.
- `HttpOnly`, `Secure`, and `SameSite` are security-related browser directives,
  but they do not replace server-side validation.
