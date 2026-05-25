package http1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrLineTooLong   = errors.New("line too long")
	ErrLineEnding    = errors.New("invalid line ending")
	ErrUnexpectedEOF = errors.New("unexpected EOF while reading line")
)

// LineReader reads CRLF-terminated protocol lines from a byte stream.
type LineReader struct {
	reader  *bufio.Reader
	maxLine int
}

// NewLineReader returns a reader that extracts HTTP/1.x CRLF-terminated lines.
func NewLineReader(r io.Reader, maxLine int) *LineReader {
	return &LineReader{
		reader:  bufio.NewReader(r),
		maxLine: maxLine,
	}
}

// ReadLine reads one line and returns it without the trailing CRLF.
func (r *LineReader) ReadLine() (string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", ErrUnexpectedEOF
		}
		return "", err
	}

	if r.maxLine > 0 && len(line) > r.maxLine {
		return "", fmt.Errorf("%w: %d bytes", ErrLineTooLong, len(line))
	}

	if !strings.HasSuffix(line, "\r\n") {
		return "", ErrLineEnding
	}

	return strings.TrimSuffix(line, "\r\n"), nil
}
