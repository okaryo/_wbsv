package httpserver

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/okaryo/_wbsv/internal/http1"
)

const webSocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

var ErrInvalidWebSocketUpgrade = errors.New("invalid websocket upgrade")

// AcceptWebSocket writes the HTTP/1.1 WebSocket upgrade handshake response.
func AcceptWebSocket(w ResponseWriter, request Request) error {
	accept, err := webSocketAcceptValue(request.HTTP)
	if err != nil {
		return err
	}

	w.WriteHeader(101)
	w.SetHeader("Upgrade", "websocket")
	w.SetHeader("Connection", "Upgrade")
	w.SetHeader("Sec-WebSocket-Accept", accept)
	return nil
}

func webSocketAcceptValue(request http1.Request) (string, error) {
	if !strings.EqualFold(request.RequestLine.Method, "GET") {
		return "", fmt.Errorf("%w: method", ErrInvalidWebSocketUpgrade)
	}
	if request.RequestLine.Version != "HTTP/1.1" {
		return "", fmt.Errorf("%w: version", ErrInvalidWebSocketUpgrade)
	}
	if !http1.HeaderHasToken(request.Headers, "Connection", "Upgrade") {
		return "", fmt.Errorf("%w: connection", ErrInvalidWebSocketUpgrade)
	}
	if !headerEquals(request.Headers, "Upgrade", "websocket") {
		return "", fmt.Errorf("%w: upgrade", ErrInvalidWebSocketUpgrade)
	}
	if headerValueFold(request.Headers, "Sec-WebSocket-Version") != "13" {
		return "", fmt.Errorf("%w: version header", ErrInvalidWebSocketUpgrade)
	}

	key := strings.Trim(headerValueFold(request.Headers, "Sec-WebSocket-Key"), " \t")
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != 16 {
		return "", fmt.Errorf("%w: key", ErrInvalidWebSocketUpgrade)
	}

	sum := sha1.Sum([]byte(key + webSocketGUID))
	return base64.StdEncoding.EncodeToString(sum[:]), nil
}

func headerEquals(headers []http1.HeaderField, name string, value string) bool {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) && strings.EqualFold(strings.Trim(header.Value, " \t"), value) {
			return true
		}
	}
	return false
}
