package tcpserver

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"sync"
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

func TestServerClosesIdleConnectionAfterReadTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := &Server{
		ReadTimeout: 20 * time.Millisecond,
		Logger:      log.New(io.Discard, "", 0),
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

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client read deadline: %v", err)
	}

	buf := make([]byte, 1)
	n, err := conn.Read(buf)
	if err == nil {
		t.Fatalf("read succeeded with %d bytes; want closed idle connection", n)
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

func TestServerClosesActiveConnectionsOnShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := &Server{
		ReadTimeout: time.Hour,
		Logger:      log.New(io.Discard, "", 0),
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

	cancel()

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client read deadline: %v", err)
	}

	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err == nil {
		t.Fatal("read succeeded; want closed connection after server shutdown")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after context cancellation")
	}
}

func TestServerWaitsForActiveConnectionDuringGracefulShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	server := &Server{
		GracefulTimeout: time.Second,
		Logger:          log.New(io.Discard, "", 0),
		ConnHandler: func(context.Context, net.Conn) {
			close(started)
			<-release
		},
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

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	cancel()

	select {
	case err := <-errCh:
		t.Fatalf("server returned before active connection finished: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	close(release)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after active connection finished")
	}
}

func TestServerForceClosesActiveConnectionAfterGracefulTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := make(chan struct{})
	readDone := make(chan struct{})
	server := &Server{
		GracefulTimeout: 20 * time.Millisecond,
		Logger:          log.New(io.Discard, "", 0),
		ConnHandler: func(_ context.Context, conn net.Conn) {
			close(started)
			_, _ = conn.Read(make([]byte, 1))
			close(readDone)
		},
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

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	cancel()

	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("blocked connection was not force closed")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after force closing active connection")
	}
}

func TestServerCancelsConnectionContextOnShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := make(chan struct{})
	contextDone := make(chan struct{})
	server := &Server{
		GracefulTimeout: time.Second,
		Logger:          log.New(io.Discard, "", 0),
		ConnHandler: func(ctx context.Context, _ net.Conn) {
			close(started)
			<-ctx.Done()
			close(contextDone)
		},
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

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	cancel()

	select {
	case <-contextDone:
	case <-time.After(time.Second):
		t.Fatal("connection context was not canceled")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after connection context cancellation")
	}
}

func TestServerStatsReportsActiveConnectionsAndGoroutines(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	server := &Server{
		Logger: log.New(io.Discard, "", 0),
		ConnHandler: func(context.Context, net.Conn) {
			close(started)
			<-release
		},
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

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	stats := server.Stats()
	if stats.ActiveConnections != 1 {
		t.Fatalf("ActiveConnections = %d, want 1", stats.ActiveConnections)
	}
	if stats.Goroutines <= 0 {
		t.Fatalf("Goroutines = %d, want positive value", stats.Goroutines)
	}

	close(release)

	deadline := time.After(time.Second)
	for {
		if server.Stats().ActiveConnections == 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("active connection was not untracked")
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func TestServerWaitForIdleReturnsTrueAfterConnectionsClose(t *testing.T) {
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

	for range 5 {
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatalf("dial: %v", err)
		}

		if err := conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatalf("set deadline: %v", err)
		}

		const message = "hello"
		if _, err := conn.Write([]byte(message)); err != nil {
			t.Fatalf("write: %v", err)
		}
		buf := make([]byte, len(message))
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Fatalf("read echo: %v", err)
		}
		if err := conn.Close(); err != nil {
			t.Fatalf("close client conn: %v", err)
		}
	}

	if !server.WaitForIdle(time.Second) {
		t.Fatalf("server did not become idle; stats = %+v", server.Stats())
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func TestServerWaitForIdleReturnsFalseWhileConnectionIsActive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	server := &Server{
		Logger: log.New(io.Discard, "", 0),
		ConnHandler: func(context.Context, net.Conn) {
			close(started)
			<-release
		},
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

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	if server.WaitForIdle(20 * time.Millisecond) {
		t.Fatal("WaitForIdle() = true, want false while connection is active")
	}

	close(release)
	if !server.WaitForIdle(time.Second) {
		t.Fatalf("server did not become idle after release; stats = %+v", server.Stats())
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func TestServerLimitsConnectionHandlersWithWorkerPool(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := make(chan struct{}, 3)
	release := make(chan struct{})
	server := &Server{
		HandlerWorkers: 2,
		Logger:         log.New(io.Discard, "", 0),
		ConnHandler: func(context.Context, net.Conn) {
			started <- struct{}{}
			<-release
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx, listener)
	}()

	conns := make([]net.Conn, 0, 3)
	for range 3 {
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		conns = append(conns, conn)
	}
	defer func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("handler did not start")
		}
	}

	deadline := time.After(time.Second)
	for server.Stats().ActiveConnections < 3 {
		select {
		case <-deadline:
			t.Fatalf("server did not track all connections; stats = %+v", server.Stats())
		case <-time.After(10 * time.Millisecond):
		}
	}

	select {
	case <-started:
		t.Fatal("third handler started before a worker was released")
	case <-time.After(30 * time.Millisecond):
	}

	close(release)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("third handler did not start after a worker was released")
	}

	if !server.WaitForIdle(time.Second) {
		t.Fatalf("server did not become idle; stats = %+v", server.Stats())
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func TestServerClosesListenerOnceOnShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	listener := &blockingListener{
		done: make(chan struct{}),
	}

	server := &Server{
		Logger: log.New(io.Discard, "", 0),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx, listener)
	}()

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop after context cancellation")
	}

	if got := listener.CloseCalls(); got != 1 {
		t.Fatalf("listener Close calls = %d, want 1", got)
	}
}

func TestServerSetsReadAndWriteDeadlines(t *testing.T) {
	t.Parallel()

	conn := &scriptedConn{
		readData: []byte("hello"),
	}

	server := &Server{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		Logger:       log.New(io.Discard, "", 0),
	}

	server.handleConn(context.Background(), conn)

	if conn.readDeadline.IsZero() {
		t.Fatal("read deadline was not set")
	}
	if conn.writeDeadline.IsZero() {
		t.Fatal("write deadline was not set")
	}
	if got := conn.written.String(); got != "hello" {
		t.Fatalf("written data = %q, want %q", got, "hello")
	}
	if !conn.closed {
		t.Fatal("connection was not closed")
	}
}

type scriptedConn struct {
	readData []byte
	readDone bool

	written bytes.Buffer

	readDeadline  time.Time
	writeDeadline time.Time
	closed        bool
}

func (c *scriptedConn) Read(b []byte) (int, error) {
	if c.readDone {
		return 0, io.EOF
	}

	c.readDone = true
	return copy(b, c.readData), nil
}

func (c *scriptedConn) Write(b []byte) (int, error) {
	return c.written.Write(b)
}

func (c *scriptedConn) Close() error {
	c.closed = true
	return nil
}

func (c *scriptedConn) LocalAddr() net.Addr {
	return testAddr("local")
}

func (c *scriptedConn) RemoteAddr() net.Addr {
	return testAddr("remote")
}

func (c *scriptedConn) SetDeadline(t time.Time) error {
	c.readDeadline = t
	c.writeDeadline = t
	return nil
}

func (c *scriptedConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

func (c *scriptedConn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return nil
}

type testAddr string

func (a testAddr) Network() string {
	return "test"
}

func (a testAddr) String() string {
	return string(a)
}

type blockingListener struct {
	mu         sync.Mutex
	done       chan struct{}
	closeCalls int
}

func (l *blockingListener) Accept() (net.Conn, error) {
	<-l.done
	return nil, net.ErrClosed
}

func (l *blockingListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.closeCalls++
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return nil
}

func (l *blockingListener) CloseCalls() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.closeCalls
}

func (l *blockingListener) Addr() net.Addr {
	return testAddr("listener")
}
