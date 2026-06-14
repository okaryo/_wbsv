# Worker Pools

A worker pool uses a fixed number of goroutines to process work items. In this
project, the work item is an accepted TCP connection.

The default TCP server behavior is goroutine-per-connection:

```text
Accept conn A -> start goroutine A
Accept conn B -> start goroutine B
Accept conn C -> start goroutine C
```

With `HandlerWorkers > 0`, accepted connections are sent to a worker channel:

```text
Accept conn A -> worker 1 handles A
Accept conn B -> worker 2 handles B
Accept conn C -> waits until worker 1 or 2 is free
```

## What It Controls

The worker pool limits how many connection handlers run at the same time. This
can protect the process from starting an unbounded number of handler goroutines
when many clients connect at once.

In this project:

- `HandlerWorkers == 0` keeps the original goroutine-per-connection behavior.
- `HandlerWorkers > 0` starts that many connection handler workers.
- Accepted connections are still tracked as active connections.
- A connection waiting for a worker still consumes resources.

## What It Does Not Control

A worker pool does not make overload disappear. It moves pressure to a different
place.

If all workers are busy:

- new accepted connections may wait for an available worker;
- the accept loop may block while handing a connection to the worker channel;
- the operating system may still have pending connections in its TCP listen
  backlog;
- clients may experience slower responses or connection timeouts.

This is why worker pools are closely related to backpressure. They cap server
work, but the server still needs a policy for what to do when work arrives
faster than it can be handled.

This project also exposes `MaxActiveConns` as a simple admission-control limit.
When the limit is reached, newly accepted connections are closed instead of
being queued behind existing work.

## How To Try It

Start the server with two connection workers:

```sh
go run ./cmd/wbsv --handler-workers 2
```

Start the server with a connection limit:

```sh
go run ./cmd/wbsv --handler-workers 2 --max-active-conns 10
```

Then run a concurrent load test:

```sh
go run ./cmd/wbsvload --requests 100 --concurrency 20 --disable-keep-alives
```

Compare this with the default mode:

```sh
go run ./cmd/wbsv
```

Watch the server logs for `active_connections` and `goroutines`. With a worker
pool, handler concurrency is capped, but active connections can still grow when
clients arrive faster than workers finish.

## Key Takeaways

- Goroutine-per-connection is simple and often works well in Go.
- A worker pool caps handler goroutines, but it can make accepted connections
  wait.
- Waiting connections are still real resources.
- Worker pools and backpressure should be studied together.
- An active connection limit makes overload explicit by rejecting excess
  connections.
