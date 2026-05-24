package tcpserver

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
)

const bufferSize = 4096

// Server accepts raw TCP connections and echoes received bytes back to clients.
type Server struct {
	Addr   string
	Logger *log.Logger
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
	defer listener.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-done:
		}
	}()

	s.logf("listening on %s", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	s.logf("accepted connection from %s", conn.RemoteAddr())

	buf := make([]byte, bufferSize)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			if _, writeErr := conn.Write(buf[:n]); writeErr != nil {
				s.logf("write error for %s: %v", conn.RemoteAddr(), writeErr)
				return
			}
		}

		if err != nil {
			if !errors.Is(err, io.EOF) {
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
