package httpserver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidSSEEvent = errors.New("invalid server-sent event")

// ServerSentEvent is one event in a text/event-stream response.
type ServerSentEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

func (w *bufferedResponseWriter) WriteEvent(event ServerSentEvent) error {
	encoded, err := encodeServerSentEvent(event)
	if err != nil {
		return err
	}

	w.SetHeader("Content-Type", "text/event-stream")
	_ = w.SetCacheControl("no-cache")
	w.UseChunkedEncoding()
	_, err = w.Write([]byte(encoded))
	return err
}

func encodeServerSentEvent(event ServerSentEvent) (string, error) {
	if invalidSSEFieldValue(event.ID) {
		return "", fmt.Errorf("%w: id", ErrInvalidSSEEvent)
	}
	if invalidSSEFieldValue(event.Event) {
		return "", fmt.Errorf("%w: event", ErrInvalidSSEEvent)
	}
	if event.Retry < 0 {
		return "", fmt.Errorf("%w: retry", ErrInvalidSSEEvent)
	}

	var builder strings.Builder
	if event.ID != "" {
		builder.WriteString("id: ")
		builder.WriteString(event.ID)
		builder.WriteByte('\n')
	}
	if event.Event != "" {
		builder.WriteString("event: ")
		builder.WriteString(event.Event)
		builder.WriteByte('\n')
	}
	if event.Retry > 0 {
		builder.WriteString("retry: ")
		builder.WriteString(strconv.Itoa(event.Retry))
		builder.WriteByte('\n')
	}

	data := normalizeSSEData(event.Data)
	for line := range strings.SplitSeq(data, "\n") {
		builder.WriteString("data: ")
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	builder.WriteByte('\n')

	return builder.String(), nil
}

func normalizeSSEData(data string) string {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	return strings.ReplaceAll(data, "\r", "\n")
}

func invalidSSEFieldValue(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}
