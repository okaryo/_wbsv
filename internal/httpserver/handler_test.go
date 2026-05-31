package httpserver

import (
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"time"
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
		handler.ServeConn(serverConn)
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

func serveWithPipe(t *testing.T, request string) string {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handler := &Handler{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		Logger:       log.New(io.Discard, "", 0),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()
		handler.ServeConn(serverConn)
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
