# Load Testing

Load testing sends many requests to the server so connection behavior,
latencies, status codes, and failure modes can be observed.

This project includes a small learning-oriented command:

```sh
go run ./cmd/wbsvload --url http://127.0.0.1:8080/hello --requests 100 --concurrency 10
```

It prints a summary:

```text
requests:    100
completed:   100
failed:      0
bytes read:  1700
duration:    25ms
latency min: 1ms
latency avg: 2ms
latency max: 8ms
status:
  200: 100
```

## Useful Runs

Start the server:

```sh
go run ./cmd/wbsv
```

Send a small load:

```sh
go run ./cmd/wbsvload --requests 100 --concurrency 10
```

Increase concurrency:

```sh
go run ./cmd/wbsvload --requests 1000 --concurrency 100
```

Disable client-side keep-alive reuse:

```sh
go run ./cmd/wbsvload --requests 1000 --concurrency 100 --disable-keep-alives
```

This is useful for comparing:

- Many requests over reused HTTP client connections.
- Many requests where each request tends to open a separate connection.

Send custom headers:

```sh
go run ./cmd/wbsvload \
  --url http://127.0.0.1:8080/private \
  --header "Authorization: Bearer secret"
```

## What To Observe

During a run, watch the server logs and the load test summary:

- Status code distribution.
- Request failures.
- Total bytes read by the client.
- Latency min, average, and max.
- Server logs around accepted connections and handled requests.

For connection lifecycle learning, compare keep-alive enabled versus disabled.
When keep-alive is disabled, the server should see more connection churn.

## Current Scope

The load tool is intentionally small:

- It uses Go's standard `net/http` client.
- It supports total request count and worker concurrency.
- It supports repeated request headers.
- It can disable client keep-alives.
- It summarizes status codes and simple latency values.
- It is not a statistically rigorous benchmark tool.
- It does not report percentiles yet.
- It does not track server-side goroutine counts yet.
- It does not generate slow clients or backpressure scenarios yet.

The goal is observation and repeatability, not benchmark-grade measurement.

## Key Takeaways

- Concurrency increases simultaneous in-flight work.
- Keep-alive changes connection churn and can hide per-request TCP setup cost.
- Load testing should separate transport errors from HTTP status codes.
- Latency values are only meaningful when interpreted with request count,
  concurrency, server logs, and machine conditions.
- A small repeatable load command is useful before exploring connection leaks,
  goroutine counts, worker pools, and backpressure.
