# HTTP Request Parsing

`ReadRequest` combines the smaller parsing steps into one minimal HTTP/1.x
request parser.

The current sequence is:

```text
LineReader.ReadLine
  -> ParseRequestLine
  -> ReadHeaderFields
  -> request-level validation
  -> ContentLength
  -> ReadFixedBody
```

This still runs on a buffered byte stream. It is not connected to the TCP server
yet.

## Current Request Model

The parsed request contains:

- The request line.
- Parsed header fields.
- The fixed-length body, if `Content-Length` is present.

The parser preserves header order and original field-name casing.

## Current Validation

The parser currently rejects:

- Malformed request lines.
- Malformed header fields.
- HTTP/1.1 requests without a non-empty `Host` header.
- Unsupported `Transfer-Encoding`.
- Invalid or conflicting `Content-Length` values.
- Bodies larger than the configured maximum.
- Bodies that end before the declared `Content-Length`.

The parser currently treats requests without `Content-Length` as having no body.

`Transfer-Encoding: chunked` is intentionally rejected for now because chunked
body parsing has not been implemented yet.
