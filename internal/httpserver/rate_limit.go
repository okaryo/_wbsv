package httpserver

import (
	"strconv"
	"sync"
	"time"
)

// FixedWindowRateLimitMiddleware limits requests in a shared fixed time window.
func FixedWindowRateLimitMiddleware(limit int, window time.Duration) Middleware {
	return fixedWindowRateLimitMiddleware(limit, window, time.Now)
}

func fixedWindowRateLimitMiddleware(limit int, window time.Duration, now func() time.Time) Middleware {
	return func(next AppHandler) AppHandler {
		if limit <= 0 || window <= 0 {
			return next
		}
		if now == nil {
			now = time.Now
		}

		limiter := &fixedWindowLimiter{
			limit:  limit,
			window: window,
			now:    now,
		}

		return AppHandlerFunc(func(w ResponseWriter, request Request) {
			allowed, retryAfter := limiter.allow()
			if allowed {
				next.ServeHTTP(w, request)
				return
			}

			w.SetHeader("Retry-After", strconv.Itoa(retryAfter))
			w.SetHeader("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(429)
			_, _ = w.Write([]byte("too many requests\n"))
		})
	}
}

type fixedWindowLimiter struct {
	mu sync.Mutex

	limit       int
	window      time.Duration
	now         func() time.Time
	windowStart time.Time
	count       int
}

func (l *fixedWindowLimiter) allow() (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.now()
	if l.windowStart.IsZero() || !current.Before(l.windowStart.Add(l.window)) {
		l.windowStart = current
		l.count = 0
	}

	if l.count < l.limit {
		l.count++
		return true, 0
	}

	return false, retryAfterSeconds(l.windowStart.Add(l.window).Sub(current))
}

func retryAfterSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 0
	}
	return int((duration + time.Second - time.Nanosecond) / time.Second)
}
