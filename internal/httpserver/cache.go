package httpserver

import (
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/okaryo/_wbsv/internal/http1"
)

var ErrInvalidCacheHeader = errors.New("invalid cache header")

func requestETagMatches(headers []http1.HeaderField, etag string) bool {
	expected, err := responseETagValue(etag)
	if err != nil {
		return false
	}

	for _, header := range headers {
		if !strings.EqualFold(header.Name, "If-None-Match") {
			continue
		}

		for part := range strings.SplitSeq(header.Value, ",") {
			candidate := strings.Trim(part, " \t")
			if candidate == "*" || weakETagValue(candidate) == weakETagValue(expected) {
				return true
			}
		}
	}

	return false
}

func requestNotModifiedSince(headers []http1.HeaderField, lastModified time.Time) bool {
	if lastModified.IsZero() {
		return false
	}

	for _, header := range headers {
		if !strings.EqualFold(header.Name, "If-Modified-Since") {
			continue
		}

		since, err := stdhttp.ParseTime(header.Value)
		if err != nil {
			return false
		}
		return !lastModified.UTC().Truncate(time.Second).After(since.UTC())
	}

	return false
}

func responseETagValue(etag string) (string, error) {
	etag = strings.Trim(etag, " \t")
	if etag == "" || etag == "*" || strings.ContainsAny(etag, "\r\n") {
		return "", fmt.Errorf("%w: ETag", ErrInvalidCacheHeader)
	}
	if strings.HasPrefix(etag, `W/"`) && strings.HasSuffix(etag, `"`) {
		return etag, nil
	}
	if strings.HasPrefix(etag, `"`) && strings.HasSuffix(etag, `"`) {
		return etag, nil
	}
	if strings.ContainsAny(etag, `"`) {
		return "", fmt.Errorf("%w: ETag", ErrInvalidCacheHeader)
	}
	return `"` + etag + `"`, nil
}

func weakETagValue(etag string) string {
	if strings.HasPrefix(etag, "W/") {
		return strings.TrimPrefix(etag, "W/")
	}
	return etag
}

func httpTime(t time.Time) string {
	return t.UTC().Format(stdhttp.TimeFormat)
}

func invalidCacheHeaderValue(value string) bool {
	return value == "" || strings.ContainsAny(value, "\r\n")
}
