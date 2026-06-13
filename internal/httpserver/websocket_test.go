package httpserver

import (
	"errors"
	"testing"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestWebSocketAcceptValue(t *testing.T) {
	t.Parallel()

	request := webSocketUpgradeRequest()

	accept, err := webSocketAcceptValue(request.HTTP)
	if err != nil {
		t.Fatalf("webSocketAcceptValue() error = %v", err)
	}

	if accept != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("accept = %q, want RFC sample accept value", accept)
	}
}

func TestAcceptWebSocketWritesSwitchingProtocolsResponse(t *testing.T) {
	t.Parallel()

	writer := newBufferedResponseWriter()
	request := webSocketUpgradeRequest()

	if err := AcceptWebSocket(writer, request); err != nil {
		t.Fatalf("AcceptWebSocket() error = %v", err)
	}

	response := writer.Response()
	if response.StatusCode != 101 {
		t.Fatalf("status code = %d, want 101", response.StatusCode)
	}
	if got := headerValue(response.Headers, "Upgrade"); got != "websocket" {
		t.Fatalf("Upgrade = %q, want websocket", got)
	}
	if got := headerValue(response.Headers, "Connection"); got != "Upgrade" {
		t.Fatalf("Connection = %q, want Upgrade", got)
	}
	if got := headerValue(response.Headers, "Sec-WebSocket-Accept"); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("Sec-WebSocket-Accept = %q, want RFC sample accept value", got)
	}
}

func TestWebSocketAcceptValueRejectsInvalidUpgrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{
			name: "non GET",
			mutate: func(request *Request) {
				request.HTTP.RequestLine.Method = "POST"
			},
		},
		{
			name: "non HTTP/1.1",
			mutate: func(request *Request) {
				request.HTTP.RequestLine.Version = "HTTP/1.0"
			},
		},
		{
			name: "missing connection upgrade",
			mutate: func(request *Request) {
				request.HTTP.Headers = withoutTestHeader(request.HTTP.Headers, "Connection")
			},
		},
		{
			name: "missing upgrade websocket",
			mutate: func(request *Request) {
				request.HTTP.Headers = withoutTestHeader(request.HTTP.Headers, "Upgrade")
			},
		},
		{
			name: "unsupported version",
			mutate: func(request *Request) {
				replaceTestHeader(request.HTTP.Headers, "Sec-WebSocket-Version", "12")
			},
		},
		{
			name: "bad key",
			mutate: func(request *Request) {
				replaceTestHeader(request.HTTP.Headers, "Sec-WebSocket-Key", "bad")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := webSocketUpgradeRequest()
			tt.mutate(&request)

			_, err := webSocketAcceptValue(request.HTTP)
			if !errors.Is(err, ErrInvalidWebSocketUpgrade) {
				t.Fatalf("webSocketAcceptValue() error = %v, want ErrInvalidWebSocketUpgrade", err)
			}
		})
	}
}

func TestStatusTextIncludesSwitchingProtocols(t *testing.T) {
	t.Parallel()

	if got := http1.StatusText(101); got != "Switching Protocols" {
		t.Fatalf("StatusText(101) = %q, want Switching Protocols", got)
	}
}

func webSocketUpgradeRequest() Request {
	request := Request{HTTP: testRequest("GET", "/ws")}
	request.HTTP.Headers = []http1.HeaderField{
		{Name: "Host", Value: "server.example.com"},
		{Name: "Upgrade", Value: "websocket"},
		{Name: "Connection", Value: "keep-alive, Upgrade"},
		{Name: "Sec-WebSocket-Key", Value: "dGhlIHNhbXBsZSBub25jZQ=="},
		{Name: "Sec-WebSocket-Version", Value: "13"},
	}
	return request
}

func withoutTestHeader(headers []http1.HeaderField, name string) []http1.HeaderField {
	filtered := headers[:0]
	for _, header := range headers {
		if header.Name == name {
			continue
		}
		filtered = append(filtered, header)
	}
	return filtered
}

func replaceTestHeader(headers []http1.HeaderField, name string, value string) {
	for i, header := range headers {
		if header.Name == name {
			headers[i].Value = value
			return
		}
	}
}
