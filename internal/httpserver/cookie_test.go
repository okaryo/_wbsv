package httpserver

import (
	"errors"
	"testing"
	"time"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestRequestCookiesParsesCookieHeaders(t *testing.T) {
	t.Parallel()

	request := Request{
		HTTP: testRequest("GET", "/cookies"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Cookie", Value: "session=abc123; theme=dark"},
		{Name: "cookie", Value: "empty=; bad; spaced =ignored"},
	}

	cookies := request.Cookies()
	if len(cookies) != 3 {
		t.Fatalf("cookies len = %d, want 3: %+v", len(cookies), cookies)
	}
	if cookies[0] != (Cookie{Name: "session", Value: "abc123"}) {
		t.Fatalf("cookies[0] = %+v, want session cookie", cookies[0])
	}
	if cookies[1] != (Cookie{Name: "theme", Value: "dark"}) {
		t.Fatalf("cookies[1] = %+v, want theme cookie", cookies[1])
	}
	if cookies[2] != (Cookie{Name: "empty", Value: ""}) {
		t.Fatalf("cookies[2] = %+v, want empty cookie", cookies[2])
	}
}

func TestRequestCookieReturnsFirstCookieByName(t *testing.T) {
	t.Parallel()

	request := Request{
		HTTP: testRequest("GET", "/cookies"),
	}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Cookie", Value: "session=first; session=second"},
	}

	cookie, ok := request.Cookie("session")
	if !ok {
		t.Fatal("Cookie() ok = false, want true")
	}
	if cookie.Value != "first" {
		t.Fatalf("cookie value = %q, want first", cookie.Value)
	}
}

func TestBufferedResponseWriterSetCookieAddsSetCookieHeader(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()
	expires := time.Date(2026, 6, 5, 12, 30, 0, 0, time.UTC)

	if err := writer.SetCookie(Cookie{
		Name:     "session",
		Value:    "abc123",
		Path:     "/",
		Expires:  expires,
		MaxAge:   3600,
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Lax",
	}); err != nil {
		t.Fatalf("SetCookie: %v", err)
	}

	got := headerValue(writer.Response().Headers, "Set-Cookie")
	want := "session=abc123; Path=/; Expires=Fri, 05 Jun 2026 12:30:00 GMT; Max-Age=3600; Secure; HttpOnly; SameSite=Lax"
	if got != want {
		t.Fatalf("Set-Cookie = %q, want %q", got, want)
	}
}

func TestBufferedResponseWriterSetCookieAddsMultipleHeaders(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	if err := writer.SetCookie(Cookie{Name: "a", Value: "1"}); err != nil {
		t.Fatalf("SetCookie a: %v", err)
	}
	if err := writer.SetCookie(Cookie{Name: "b", Value: "2"}); err != nil {
		t.Fatalf("SetCookie b: %v", err)
	}

	response := writer.Response()
	if len(response.Headers) != 2 {
		t.Fatalf("headers len = %d, want 2", len(response.Headers))
	}
	if response.Headers[0] != (http1.HeaderField{Name: "Set-Cookie", Value: "a=1"}) {
		t.Fatalf("headers[0] = %+v, want first cookie", response.Headers[0])
	}
	if response.Headers[1] != (http1.HeaderField{Name: "Set-Cookie", Value: "b=2"}) {
		t.Fatalf("headers[1] = %+v, want second cookie", response.Headers[1])
	}
}

func TestBufferedResponseWriterSetCookieRejectsMalformedCookie(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()

	err := writer.SetCookie(Cookie{Name: "bad name", Value: "ok"})
	if !errors.Is(err, ErrInvalidCookie) {
		t.Fatalf("SetCookie() error = %v, want ErrInvalidCookie", err)
	}
	if len(writer.Response().Headers) != 0 {
		t.Fatalf("headers = %+v, want no Set-Cookie header", writer.Response().Headers)
	}
}
