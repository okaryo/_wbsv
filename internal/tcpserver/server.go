package tcpserver

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

const bufferSize = 4096

// ConnHandler handles one accepted TCP connection.
type ConnHandler func(context.Context, net.Conn)

// Server accepts raw TCP connections and handles each connection in a goroutine.
type Server struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	GracefulTimeout time.Duration
	HandlerWorkers  int
	Logger          *log.Logger
	ConnHandler     ConnHandler

	mu           sync.Mutex
	activeConns  map[net.Conn]struct{}
	shuttingDown bool
	wg           sync.WaitGroup
}

// Stats reports server-side concurrency counters for observation.
type Stats struct {
	ActiveConnections int
	Goroutines        int
}

// ListenAndServe starts listening on s.Addr and serves accepted connections.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listenerConfig := net.ListenConfig{}
	listener, err := listenerConfig.Listen(ctx, "tcp", s.Addr)
	if err != nil {
		return err
	}

	return s.Serve(ctx, listener)
}

// Serve accepts connections from listener until the context is canceled or an
// unrecoverable listener error occurs.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			s.beginShutdown()
			_ = listener.Close()

			if s.waitForConns(s.GracefulTimeout) {
				return
			}

			s.closeActiveConns()
			s.wg.Wait()
		})
	}

	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			shutdown()
		case <-done:
		}
	}()

	s.logf("listening on %s", listener.Addr())

	var connJobs chan net.Conn
	var workerWG sync.WaitGroup
	if s.HandlerWorkers > 0 {
		connJobs = make(chan net.Conn)
		workerWG.Add(s.HandlerWorkers)
		for range s.HandlerWorkers {
			go func() {
				defer workerWG.Done()
				for conn := range connJobs {
					s.serveTrackedConn(ctx, conn)
				}
			}()
		}
		defer workerWG.Wait()
		defer close(connJobs)
		s.logf("started %d connection handler workers", s.HandlerWorkers)
	}

	defer close(done)
	defer shutdown()

	for {
		s.logf("waiting for a connection")
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		if !s.trackConn(conn) {
			_ = conn.Close()
			return nil
		}

		if connJobs != nil {
			select {
			case connJobs <- conn:
			case <-ctx.Done():
				_ = conn.Close()
				s.untrackConn(conn)
				return nil
			}
			continue
		}

		go s.serveTrackedConn(ctx, conn)
	}
}

func (s *Server) serveTrackedConn(ctx context.Context, conn net.Conn) {
	connCtx, cancelConn := context.WithCancel(ctx)
	defer cancelConn()
	defer s.untrackConn(conn)
	s.handleConn(connCtx, conn)
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	s.logf("accepted connection from %s", conn.RemoteAddr())
	if s.ConnHandler != nil {
		s.ConnHandler(ctx, conn)
		return
	}

	buf := make([]byte, bufferSize)
	for {
		if s.ReadTimeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(s.ReadTimeout)); err != nil {
				s.logf("set read deadline error for %s: %v", conn.RemoteAddr(), err)
				return
			}
		}

		s.logf("waiting for bytes from %s", conn.RemoteAddr())
		n, err := conn.Read(buf)
		if n > 0 {
			s.logf("read %d bytes from %s", n, conn.RemoteAddr())

			if s.WriteTimeout > 0 {
				if err := conn.SetWriteDeadline(time.Now().Add(s.WriteTimeout)); err != nil {
					s.logf("set write deadline error for %s: %v", conn.RemoteAddr(), err)
					return
				}
			}

			written, writeErr := conn.Write(buf[:n])
			if writeErr != nil {
				var netErr net.Error
				if errors.As(writeErr, &netErr) && netErr.Timeout() {
					s.logf("write timeout for %s", conn.RemoteAddr())
				} else {
					s.logf("write error for %s: %v", conn.RemoteAddr(), writeErr)
				}
				return
			}
			s.logf("wrote %d bytes to %s", written, conn.RemoteAddr())
		}

		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				s.logf("read timeout for %s", conn.RemoteAddr())
			} else if !errors.Is(err, io.EOF) {
				s.logf("read error for %s: %v", conn.RemoteAddr(), err)
			}
			s.logf("closed connection from %s", conn.RemoteAddr())
			return
		}
	}
}

func (s *Server) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}

func (s *Server) trackConn(conn net.Conn) bool {
	s.mu.Lock()
	if s.shuttingDown {
		s.mu.Unlock()
		return false
	}

	s.wg.Add(1)
	if s.activeConns == nil {
		s.activeConns = make(map[net.Conn]struct{})
	}
	s.activeConns[conn] = struct{}{}
	s.mu.Unlock()

	s.logStats("tracked connection")
	return true
}

func (s *Server) untrackConn(conn net.Conn) {
	s.mu.Lock()
	delete(s.activeConns, conn)
	s.mu.Unlock()

	s.wg.Done()
	s.logStats("untracked connection")
}

func (s *Server) Stats() Stats {
	s.mu.Lock()
	activeConns := len(s.activeConns)
	s.mu.Unlock()

	return Stats{
		ActiveConnections: activeConns,
		Goroutines:        runtime.NumGoroutine(),
	}
}

// WaitForIdle waits until all tracked connections have been untracked.
func (s *Server) WaitForIdle(timeout time.Duration) bool {
	if timeout <= 0 {
		return s.Stats().ActiveConnections == 0
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if s.Stats().ActiveConnections == 0 {
			return true
		}

		select {
		case <-timer.C:
			return false
		case <-ticker.C:
		}
	}
}

func (s *Server) logStats(event string) {
	stats := s.Stats()
	s.logf("%s: active_connections=%d goroutines=%d", event, stats.ActiveConnections, stats.Goroutines)
}

func (s *Server) beginShutdown() {
	s.mu.Lock()
	s.shuttingDown = true
	s.mu.Unlock()
}

func (s *Server) waitForConns(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	if timeout <= 0 {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

func (s *Server) closeActiveConns() {
	s.mu.Lock()
	s.shuttingDown = true
	conns := make([]net.Conn, 0, len(s.activeConns))

	for conn := range s.activeConns {
		conns = append(conns, conn)
	}
	s.mu.Unlock()

	for _, conn := range conns {
		_ = conn.Close()
	}
}
