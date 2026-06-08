package httpserver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/okaryo/_wbsv/internal/http1"
)

var ErrInvalidCookie = errors.New("invalid cookie")

// Cookie is the application-facing representation of an HTTP cookie.
type Cookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HTTPOnly bool
	SameSite string
}

func requestCookies(headers []http1.HeaderField) []Cookie {
	var cookies []Cookie

	for _, header := range headers {
		if !strings.EqualFold(header.Name, "Cookie") {
			continue
		}

		for part := range strings.SplitSeq(header.Value, ";") {
			name, value, ok := strings.Cut(strings.Trim(part, " \t"), "=")
			if !ok || invalidCookieName(name) || invalidRequestCookieValue(value) {
				continue
			}
			cookies = append(cookies, Cookie{Name: name, Value: value})
		}
	}

	return cookies
}

func (c Cookie) setCookieValue() (string, error) {
	if invalidCookieName(c.Name) {
		return "", fmt.Errorf("%w: name", ErrInvalidCookie)
	}
	if invalidSetCookieValue(c.Value) {
		return "", fmt.Errorf("%w: value", ErrInvalidCookie)
	}

	var builder strings.Builder
	builder.WriteString(c.Name)
	builder.WriteByte('=')
	builder.WriteString(c.Value)

	if c.Path != "" {
		if invalidCookieAttributeValue(c.Path) {
			return "", fmt.Errorf("%w: path", ErrInvalidCookie)
		}
		builder.WriteString("; Path=")
		builder.WriteString(c.Path)
	}
	if c.Domain != "" {
		if invalidCookieAttributeValue(c.Domain) {
			return "", fmt.Errorf("%w: domain", ErrInvalidCookie)
		}
		builder.WriteString("; Domain=")
		builder.WriteString(c.Domain)
	}
	if !c.Expires.IsZero() {
		builder.WriteString("; Expires=")
		builder.WriteString(c.Expires.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	}
	if c.MaxAge > 0 {
		builder.WriteString("; Max-Age=")
		builder.WriteString(strconv.Itoa(c.MaxAge))
	} else if c.MaxAge < 0 {
		builder.WriteString("; Max-Age=0")
	}
	if c.Secure {
		builder.WriteString("; Secure")
	}
	if c.HTTPOnly {
		builder.WriteString("; HttpOnly")
	}
	if c.SameSite != "" {
		if invalidCookieAttributeValue(c.SameSite) {
			return "", fmt.Errorf("%w: same-site", ErrInvalidCookie)
		}
		builder.WriteString("; SameSite=")
		builder.WriteString(c.SameSite)
	}

	return builder.String(), nil
}

func invalidCookieName(name string) bool {
	return name == "" || strings.ContainsAny(name, " \t\r\n=;")
}

func invalidRequestCookieValue(value string) bool {
	return strings.ContainsAny(value, "\r\n;")
}

func invalidSetCookieValue(value string) bool {
	return strings.ContainsAny(value, "\r\n;")
}

func invalidCookieAttributeValue(value string) bool {
	return strings.ContainsAny(value, "\r\n;")
}
