package httpserver

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/okaryo/_wbsv/internal/http1"
)

var ErrInvalidRange = errors.New("invalid range")

// ByteRange is one satisfiable byte range for a representation.
type ByteRange struct {
	Start  int64
	End    int64
	Length int64
}

func (r Request) ByteRange(size int64) (ByteRange, bool, error) {
	return requestByteRange(r.HTTP.Headers, size)
}

func requestByteRange(headers []http1.HeaderField, size int64) (ByteRange, bool, error) {
	value := headerValueFold(headers, "Range")
	if value == "" {
		return ByteRange{}, false, nil
	}
	if size < 0 {
		return ByteRange{}, true, fmt.Errorf("%w: negative size", ErrInvalidRange)
	}

	spec, ok := strings.CutPrefix(strings.Trim(value, " \t"), "bytes=")
	if !ok {
		return ByteRange{}, true, fmt.Errorf("%w: unsupported unit", ErrInvalidRange)
	}
	if strings.Contains(spec, ",") {
		return ByteRange{}, true, fmt.Errorf("%w: multiple ranges unsupported", ErrInvalidRange)
	}

	startText, endText, ok := strings.Cut(strings.Trim(spec, " \t"), "-")
	if !ok {
		return ByteRange{}, true, fmt.Errorf("%w: missing dash", ErrInvalidRange)
	}

	switch {
	case startText == "":
		return suffixByteRange(endText, size)
	case endText == "":
		return openEndedByteRange(startText, size)
	default:
		return closedByteRange(startText, endText, size)
	}
}

func closedByteRange(startText string, endText string, size int64) (ByteRange, bool, error) {
	start, err := parseRangeNumber(startText)
	if err != nil {
		return ByteRange{}, true, err
	}
	end, err := parseRangeNumber(endText)
	if err != nil {
		return ByteRange{}, true, err
	}
	if start > end || start >= size {
		return ByteRange{}, true, fmt.Errorf("%w: unsatisfiable", ErrInvalidRange)
	}
	if end >= size {
		end = size - 1
	}
	return byteRange(start, end), true, nil
}

func openEndedByteRange(startText string, size int64) (ByteRange, bool, error) {
	start, err := parseRangeNumber(startText)
	if err != nil {
		return ByteRange{}, true, err
	}
	if start >= size {
		return ByteRange{}, true, fmt.Errorf("%w: unsatisfiable", ErrInvalidRange)
	}
	return byteRange(start, size-1), true, nil
}

func suffixByteRange(lengthText string, size int64) (ByteRange, bool, error) {
	length, err := parseRangeNumber(lengthText)
	if err != nil {
		return ByteRange{}, true, err
	}
	if length == 0 {
		return ByteRange{}, true, fmt.Errorf("%w: zero suffix length", ErrInvalidRange)
	}
	if length > size {
		length = size
	}
	return byteRange(size-length, size-1), true, nil
}

func parseRangeNumber(value string) (int64, error) {
	if value == "" || strings.ContainsAny(value, " \t+") || strings.HasPrefix(value, "-") {
		return 0, fmt.Errorf("%w: invalid number", ErrInvalidRange)
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("%w: invalid number", ErrInvalidRange)
	}
	return number, nil
}

func byteRange(start int64, end int64) ByteRange {
	return ByteRange{
		Start:  start,
		End:    end,
		Length: end - start + 1,
	}
}

func (w *bufferedResponseWriter) StreamRange(reader io.ReaderAt, size int64, byteRange ByteRange) error {
	if byteRange.Start < 0 || byteRange.End < byteRange.Start || byteRange.End >= size {
		return fmt.Errorf("%w: unsatisfiable", ErrInvalidRange)
	}

	w.SetHeader("Accept-Ranges", "bytes")
	w.SetHeader("Content-Range", contentRange(byteRange, size))
	w.WriteHeader(206)
	w.StreamBody(io.NewSectionReader(reader, byteRange.Start, byteRange.Length), byteRange.Length)
	return nil
}

func (w *bufferedResponseWriter) WriteRangeNotSatisfiable(size int64) {
	w.SetHeader("Accept-Ranges", "bytes")
	w.SetHeader("Content-Range", "bytes */"+strconv.FormatInt(size, 10))
	w.WriteHeader(416)
}

func contentRange(byteRange ByteRange, size int64) string {
	return "bytes " +
		strconv.FormatInt(byteRange.Start, 10) + "-" +
		strconv.FormatInt(byteRange.End, 10) + "/" +
		strconv.FormatInt(size, 10)
}
