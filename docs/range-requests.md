# Range Requests

Range requests let a client ask for only part of a representation.

The most common form is a byte range:

```text
Range: bytes=1000-1999
```

If the range is satisfiable, the server responds with `206 Partial Content`:

```text
HTTP/1.1 206 Partial Content
Accept-Ranges: bytes
Content-Range: bytes 1000-1999/10000
Content-Length: 1000

...1000 bytes...
```

If the range cannot be satisfied, the server responds with
`416 Range Not Satisfiable`:

```text
HTTP/1.1 416 Range Not Satisfiable
Accept-Ranges: bytes
Content-Range: bytes */10000
Content-Length: 0
```

## Range Forms

This project supports one byte range at a time.

Closed range:

```text
Range: bytes=2-5
```

Open-ended range:

```text
Range: bytes=5-
```

Suffix range:

```text
Range: bytes=-500
```

Multiple ranges are intentionally unsupported for now:

```text
Range: bytes=0-99,200-299
```

Multiple ranges require multipart response bodies and add a lot of framing
complexity, so they are a separate topic.

## Why Range Requests Matter

Range requests are useful when the client does not need the entire body:

- Video and audio seeking.
- Resuming interrupted downloads.
- PDF viewers loading selected pages or byte regions.
- Large file clients that fetch data in chunks.

They work especially well with files because files support random access. In Go,
that maps naturally to `io.ReaderAt`.

## Current Implementation

Handlers can parse a request range:

```go
byteRange, ok, err := request.ByteRange(fileSize)
```

If the range is valid, the handler can stream that section:

```go
_ = w.StreamRange(file, fileSize, byteRange)
```

If the range is present but unsatisfiable, the handler can write:

```go
w.WriteRangeNotSatisfiable(fileSize)
```

The range body uses `io.NewSectionReader`, so only the requested byte interval
is read from the source.

## Current Scope

The implementation is intentionally small:

- It supports only `bytes` ranges.
- It supports only one range per request.
- It supports closed, open-ended, and suffix byte ranges.
- It writes `206 Partial Content` with `Content-Range`.
- It writes `416 Range Not Satisfiable` with `Content-Range: bytes */size`.
- It does not implement multipart byte ranges.
- It does not yet integrate `Range` directly into `SendFile`.
- It does not implement `If-Range`.

## Key Takeaways

- Range requests are about selecting part of a representation, not merely
  streaming.
- A valid partial response uses `206`, not `200`.
- `Content-Range` describes the selected byte interval and total size.
- Unsatisfiable ranges use `416` and `Content-Range: bytes */size`.
- Efficient range serving needs a source that can read from an offset, such as
  a file or another `io.ReaderAt`.
