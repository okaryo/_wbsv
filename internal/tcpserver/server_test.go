package tcpserver

import (
	"context"
	"io"
	"log"
	"net"
	"testing"
	"time"
)

func TestServerEchoesBytes(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := &Server{
		Logger: log.New(io.Discard, "", 0),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx, listener)
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	const message = "hello tcp\n"
	if _, err := conn.Write([]byte(message)); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, len(message))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read echo: %v", err)
	}

	if got := string(buf); got != message {
		t.Fatalf("echoed message = %q, want %q", got, message)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after context cancellation")
	}
}
