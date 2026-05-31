package httpserver

import (
	"context"
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/okaryo/_wbsv/internal/http1"
)

func TestHandlerServesOneHTTPRequest(t *testing.T) {
	t.Parallel()

	response := serveWithPipe(t,
		"GET /hello HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Connection: close\r\n"+
			"\r\n",
	)

	want := "HTTP/1.1 200 OK\r\n" +
		"X-WBSV-Method: GET\r\n" +
		"X-WBSV-Target: /hello\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Connection: close\r\n" +
		"Content-Length: 17\r\n" +
		"\r\n" +
		"hello from _wbsv\n"
	if response != want {
		t.Fatalf("response = %q, want %q", response, want)
	}
}

func TestHandlerKeepsHTTP11ConnectionAlive(t *testing.T) {
	t.Parallel()

	response := serveWithPipe(t,
		"GET /first HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"\r\n"+
			"GET /second HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Connection: close\r\n"+
			"\r\n",
	)

	if got := strings.Count(response, "HTTP/1.1 200 OK\r\n"); got != 2 {
		t.Fatalf("200 response count = %d, want 2; response = %q", got, response)
	}
	if !strings.Contains(response, "X-WBSV-Target: /first\r\n") {
		t.Fatalf("response = %q, want first target header", response)
	}
	if !strings.Contains(response, "X-WBSV-Target: /second\r\n") {
		t.Fatalf("response = %q, want second target header", response)
	}
	if !strings.Contains(response, "Connection: close\r\n") {
		t.Fatalf("response = %q, want close header on final response", response)
	}
}

func TestHandlerUsesApplicationHandler(t *testing.T) {
	t.Parallel()

	response := serveWithHandler(t,
		&Handler{
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			Logger:       log.New(io.Discard, "", 0),
			App: AppHandlerFunc(func(request Request) http1.Response {
				return http1.Response{
					StatusCode: 201,
					Headers: []http1.HeaderField{
						{Name: "X-App-Target", Value: request.HTTP.RequestLine.RequestTarget},
					},
					Body: []byte("created\n"),
				}
			}),
		},
		"POST /items HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Connection: close\r\n"+
			"Content-Length: 0\r\n"+
			"\r\n",
	)

	want := "HTTP/1.1 201 Created\r\n" +
		"X-App-Target: /items\r\n" +
		"Connection: close\r\n" +
		"Content-Length: 8\r\n" +
		"\r\n" +
		"created\n"
	if response != want {
		t.Fatalf("response = %q, want %q", response, want)
	}
}

func TestHandlerPassesContextToApplicationHandler(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	seenDone := make(chan struct{})

	response := serveWithContextAndHandler(t,
		ctx,
		&Handler{
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			Logger:       log.New(io.Discard, "", 0),
			App: AppHandlerFunc(func(request Request) http1.Response {
				cancel()
				select {
				case <-request.Context.Done():
					close(seenDone)
				case <-time.After(time.Second):
				}

				return http1.Response{
					StatusCode: 200,
					Body:       []byte("context observed\n"),
				}
			}),
		},
		"GET /context HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Connection: close\r\n"+
			"\r\n",
	)

	select {
	case <-seenDone:
	default:
		t.Fatal("application handler did not observe request context cancellation")
	}
	if !strings.Contains(response, "context observed\n") {
		t.Fatalf("response = %q, want context response body", response)
	}
}

func TestHandlerReturnsBadRequestForMalformedRequest(t *testing.T) {
	t.Parallel()

	response := serveWithPipe(t, "GET /\r\n\r\n")

	if !strings.HasPrefix(response, "HTTP/1.1 400 Bad Request\r\n") {
		t.Fatalf("response = %q, want 400 response", response)
	}
	if !strings.Contains(response, "bad request\n") {
		t.Fatalf("response = %q, want bad request body", response)
	}
}

func TestHandlerReturnsRequestTimeout(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handler := &Handler{
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: time.Second,
		Logger:       log.New(io.Discard, "", 0),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()
		handler.ServeConn(context.Background(), serverConn)
	}()

	if err := clientConn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}

	response, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return")
	}

	if !strings.HasPrefix(string(response), "HTTP/1.1 408 Request Timeout\r\n") {
		t.Fatalf("response = %q, want 408 response", string(response))
	}
}

func TestHandlerReturnsRequestTimeoutWhileWaitingForNextKeepAliveRequest(t *testing.T) {
	t.Parallel()

	response := serveWithHandler(t,
		&Handler{
			ReadTimeout:  10 * time.Millisecond,
			WriteTimeout: time.Second,
			Logger:       log.New(io.Discard, "", 0),
		},
		"GET /first HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"\r\n",
	)

	if !strings.Contains(response, "X-WBSV-Target: /first\r\n") {
		t.Fatalf("response = %q, want first response", response)
	}
	if !strings.Contains(response, "HTTP/1.1 408 Request Timeout\r\n") {
		t.Fatalf("response = %q, want 408 response after keep-alive idle timeout", response)
	}
}

func TestHandlerReturnsWhenResponseWriteTimesOut(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handler := &Handler{
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Millisecond,
		Logger:       log.New(io.Discard, "", 0),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()
		handler.ServeConn(context.Background(), serverConn)
	}()

	if err := clientConn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	if _, err := io.WriteString(clientConn,
		"GET /slow-reader HTTP/1.1\r\n"+
			"Host: localhost\r\n"+
			"Connection: close\r\n"+
			"\r\n",
	); err != nil {
		t.Fatalf("write request: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after write timeout")
	}
}

func serveWithPipe(t *testing.T, request string) string {
	t.Helper()

	handler := &Handler{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		Logger:       log.New(io.Discard, "", 0),
	}

	return serveWithHandler(t, handler, request)
}

func serveWithHandler(t *testing.T, handler *Handler, request string) string {
	t.Helper()

	return serveWithContextAndHandler(t, context.Background(), handler, request)
}

func serveWithContextAndHandler(t *testing.T, ctx context.Context, handler *Handler, request string) string {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()
		handler.ServeConn(ctx, serverConn)
	}()

	if err := clientConn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	if _, err := io.WriteString(clientConn, request); err != nil {
		t.Fatalf("write request: %v", err)
	}

	response, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return")
	}

	return string(response)
}
