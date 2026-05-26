package http1

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrBodyTooLarge      = errors.New("body too large")
	ErrUnexpectedBodyEOF = errors.New("unexpected EOF while reading body")
)

const bodyReadChunkSize = 4096

// ReadFixedBody reads exactly length bytes from the stream.
func (r *LineReader) ReadFixedBody(length int64, maxBody int64) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf("%w: negative length", ErrInvalidContentLength)
	}
	if maxBody > 0 && length > maxBody {
		return nil, fmt.Errorf("%w: %d bytes", ErrBodyTooLarge, length)
	}
	if length == 0 {
		return nil, nil
	}
	maxInt := int64(int(^uint(0) >> 1))
	if length > maxInt {
		return nil, fmt.Errorf("%w: %d bytes", ErrBodyTooLarge, length)
	}

	body := make([]byte, 0, int(length))
	remaining := length
	buf := make([]byte, bodyReadChunkSize)

	for remaining > 0 {
		readSize := len(buf)
		if remaining < int64(readSize) {
			readSize = int(remaining)
		}

		n, err := r.reader.Read(buf[:readSize])
		if n > 0 {
			body = append(body, buf[:n]...)
			remaining -= int64(n)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, ErrUnexpectedBodyEOF
			}
			return nil, err
		}
	}

	return body, nil
}
